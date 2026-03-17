// Copyright 2025 CloudWeGo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ipc

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/cloudwego/abcoder/lang/java/pb"
)

const (
	// MaxMessageSize is the maximum allowed message size (64MB)
	MaxMessageSize = 64 * 1024 * 1024

	// DefaultBufferSize is the default buffer size for reading
	DefaultBufferSize = 64 * 1024

	// MaxLogMessageLength is the max length of message content to log
	MaxLogMessageLength = 500
)

var (
	ErrMessageTooLarge = errors.New("message size exceeds maximum allowed")
	ErrInvalidLength   = errors.New("invalid length encoding")
	ErrUnexpectedEOF   = errors.New("unexpected end of stream")
)

// ProtocolReader handles reading length-prefixed JSON messages from a stream.
//
// Wire format:
//   - 4 bytes: message length (big-endian uint32)
//   - bytes: JSON message body
type ProtocolReader struct {
	reader *bufio.Reader
	buf    []byte
	debug  bool
}

// NewProtocolReader creates a new protocol reader with the given io.Reader.
func NewProtocolReader(r io.Reader) *ProtocolReader {
	return &ProtocolReader{
		reader: bufio.NewReaderSize(r, DefaultBufferSize),
		buf:    make([]byte, 0, 4096),
		debug:  false,
	}
}

// SetDebug enables or disables debug logging
func (pr *ProtocolReader) SetDebug(enabled bool) {
	pr.debug = enabled
}

// ReadMessage reads a single length-prefixed JSON message.
// Returns io.EOF when the stream ends normally.
func (pr *ProtocolReader) ReadMessage() (*pb.Message, error) {
	// Step A: Read 4-byte length prefix (big-endian)
	var lengthBuf [4]byte
	_, err := io.ReadFull(pr.reader, lengthBuf[:])
	if err != nil {
		if err == io.EOF {
			if pr.debug {
				log.Printf("[Protocol] <<< EOF received")
			}
			return nil, io.EOF
		}
		return nil, fmt.Errorf("failed to read message length: %w", err)
	}

	length := binary.BigEndian.Uint32(lengthBuf[:])

	if pr.debug {
		log.Printf("[Protocol] <<< Reading message, length=%d bytes", length)
	}

	// Validate message size
	if length > MaxMessageSize {
		return nil, fmt.Errorf("%w: %d bytes (max: %d)", ErrMessageTooLarge, length, MaxMessageSize)
	}

	if length == 0 {
		if pr.debug {
			log.Printf("[Protocol] <<< Empty message received")
		}
		return &pb.Message{}, nil
	}

	// Step B: Read exactly 'length' bytes of message body
	if cap(pr.buf) < int(length) {
		pr.buf = make([]byte, length)
	} else {
		pr.buf = pr.buf[:length]
	}

	_, err = io.ReadFull(pr.reader, pr.buf)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("%w: expected %d bytes", ErrUnexpectedEOF, length)
		}
		return nil, fmt.Errorf("failed to read message body: %w", err)
	}

	// Debug: Log raw JSON
	if pr.debug {
		jsonStr := string(pr.buf)
		if len(jsonStr) > MaxLogMessageLength {
			jsonStr = jsonStr[:MaxLogMessageLength] + "...(truncated)"
		}
		log.Printf("[Protocol] <<< Received JSON: %s", jsonStr)
	}

	// Step C: Unmarshal JSON message (outer wrapper)
	type rawMessage struct {
		Type      string          `json:"type"`
		RequestId string          `json:"requestId,omitempty"`
		Payload   json.RawMessage `json:"payload,omitempty"`
	}
	var raw rawMessage
	if err := json.Unmarshal(pr.buf, &raw); err != nil {
		if pr.debug {
			log.Printf("[Protocol] <<< Failed to unmarshal: %v", err)
		}
		return nil, fmt.Errorf("failed to unmarshal JSON message: %w", err)
	}

	msg := &pb.Message{Type: raw.Type, RequestId: raw.RequestId}
	if len(raw.Payload) == 0 {
		if pr.debug {
			log.Printf("[Protocol] <<< Parsed message: type=%s, requestId=%s (no payload)", msg.Type, msg.RequestId)
		}
		return msg, nil
	}

	switch raw.Type {
	case pb.TYPE_ANALYZE_RESPONSE:
		type rawAnalyzeResponse struct {
			RequestId   string          `json:"requestId"`
			PayloadType string          `json:"payloadType"`
			Payload     json.RawMessage `json:"payload"`
		}
		var arRaw rawAnalyzeResponse
		if err := json.Unmarshal(raw.Payload, &arRaw); err != nil {
			return nil, fmt.Errorf("failed to unmarshal analyze_response payload: %w", err)
		}

		ar := &pb.AnalyzeResponse{RequestId: arRaw.RequestId, PayloadType: arRaw.PayloadType}
		var payload any
		switch arRaw.PayloadType {
		case pb.PAYLOAD_PROGRESS:
			payload = &pb.ProgressUpdate{}
		case pb.PAYLOAD_FILE_INFO:
			payload = &pb.FileInfo{}
		case pb.PAYLOAD_CLASS_INFO:
			payload = &pb.ClassInfo{}
		case pb.PAYLOAD_METHOD_CALL:
			payload = &pb.MethodCallInfo{}
		case pb.PAYLOAD_SUMMARY:
			payload = &pb.Summary{}
		case pb.PAYLOAD_ERROR:
			payload = &pb.ErrorInfo{}
		default:
			// 未知 payloadType：保留原始 JSON
			payload = json.RawMessage(arRaw.Payload)
		}

		if rm, ok := payload.(json.RawMessage); ok {
			ar.Payload = rm
		} else {
			if err := json.Unmarshal(arRaw.Payload, payload); err != nil {
				return nil, fmt.Errorf("failed to unmarshal analyze_response inner payload (%s): %w", arRaw.PayloadType, err)
			}
			ar.Payload = payload
		}
		msg.Payload = ar

	case pb.TYPE_ANALYZE_REQUEST:
		var req pb.AnalyzeRequest
		if err := json.Unmarshal(raw.Payload, &req); err != nil {
			return nil, fmt.Errorf("failed to unmarshal analyze_request payload: %w", err)
		}
		msg.Payload = &req

	case pb.TYPE_STOP_REQUEST:
		var stop pb.StopRequest
		if err := json.Unmarshal(raw.Payload, &stop); err != nil {
			return nil, fmt.Errorf("failed to unmarshal stop_request payload: %w", err)
		}
		msg.Payload = &stop

	case pb.TYPE_HEARTBEAT:
		var hb pb.Heartbeat
		if err := json.Unmarshal(raw.Payload, &hb); err != nil {
			return nil, fmt.Errorf("failed to unmarshal heartbeat payload: %w", err)
		}
		msg.Payload = &hb

	default:
		msg.Payload = json.RawMessage(raw.Payload)
	}

	if pr.debug {
		if ar := msg.GetAnalyzeResponse(); ar != nil {
			log.Printf("[Protocol] <<< Parsed message: type=%s, requestId=%s, payloadType=%s", msg.Type, msg.RequestId, ar.PayloadType)
		} else {
			log.Printf("[Protocol] <<< Parsed message: type=%s, requestId=%s", msg.Type, msg.RequestId)
		}
	}

	return msg, nil
}

// ProtocolWriter handles writing length-prefixed JSON messages to a stream.
type ProtocolWriter struct {
	writer io.Writer
	debug  bool
}

// NewProtocolWriter creates a new protocol writer with the given io.Writer.
func NewProtocolWriter(w io.Writer) *ProtocolWriter {
	return &ProtocolWriter{
		writer: w,
		debug:  false,
	}
}

// SetDebug enables or disables debug logging
func (pw *ProtocolWriter) SetDebug(enabled bool) {
	pw.debug = enabled
}

// WriteMessage writes a length-prefixed JSON Message.
func (pw *ProtocolWriter) WriteMessage(msg *pb.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON message: %w", err)
	}

	if pw.debug {
		jsonStr := string(data)
		if len(jsonStr) > MaxLogMessageLength {
			jsonStr = jsonStr[:MaxLogMessageLength] + "...(truncated)"
		}
		log.Printf("[Protocol] >>> Sending message, length=%d bytes", len(data))
		log.Printf("[Protocol] >>> JSON: %s", jsonStr)
	}

	// Write 4-byte length prefix (big-endian)
	var lengthBuf [4]byte
	binary.BigEndian.PutUint32(lengthBuf[:], uint32(len(data)))
	if _, err := pw.writer.Write(lengthBuf[:]); err != nil {
		return fmt.Errorf("failed to write message length: %w", err)
	}

	// Write message body
	if _, err := pw.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write message body: %w", err)
	}

	if pw.debug {
		log.Printf("[Protocol] >>> Message sent successfully")
	}

	return nil
}

// WriteRequest writes a length-prefixed JSON AnalyzeRequest message (wrapped by Message).
func (pw *ProtocolWriter) WriteRequest(req *pb.AnalyzeRequest) error {
	msg := &pb.Message{
		Type:      pb.TYPE_ANALYZE_REQUEST,
		RequestId: req.RequestId,
		Payload:   req,
	}
	return pw.WriteMessage(msg)
}

// WriteResponse writes a length-prefixed JSON AnalyzeResponse message (wrapped by Message).
func (pw *ProtocolWriter) WriteResponse(resp *pb.AnalyzeResponse) error {
	msg := &pb.Message{
		Type:      pb.TYPE_ANALYZE_RESPONSE,
		RequestId: resp.RequestId,
		Payload:   resp,
	}
	return pw.WriteMessage(msg)
}

// MessageIterator provides an iterator interface for reading messages.
type MessageIterator struct {
	reader *ProtocolReader
	err    error
	msg    *pb.Message
}

// NewMessageIterator creates a new message iterator.
func NewMessageIterator(r io.Reader) *MessageIterator {
	return &MessageIterator{
		reader: NewProtocolReader(r),
	}
}

// Next advances the iterator to the next message.
func (it *MessageIterator) Next() bool {
	it.msg, it.err = it.reader.ReadMessage()
	return it.err == nil
}

// Message returns the current message.
func (it *MessageIterator) Message() *pb.AnalyzeResponse {
	return it.msg.GetAnalyzeResponse()
}

// RawMessage returns the current raw outer message.
func (it *MessageIterator) RawMessage() *pb.Message {
	return it.msg
}

// Err returns any error that occurred during iteration.
func (it *MessageIterator) Err() error {
	if it.err == io.EOF {
		return nil
	}
	return it.err
}

// ReadAllMessages reads all messages from the reader until EOF.
func ReadAllMessages(r io.Reader) ([]*pb.AnalyzeResponse, error) {
	var messages []*pb.AnalyzeResponse
	it := NewMessageIterator(r)

	for it.Next() {
		msg := it.Message()
		if msg == nil {
			continue
		}
		msgCopy := *msg
		messages = append(messages, &msgCopy)
	}

	if err := it.Err(); err != nil {
		return messages, err
	}

	return messages, nil
}
