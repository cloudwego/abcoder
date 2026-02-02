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
	"bytes"
	"io"
	"testing"

	"github.com/cloudwego/abcoder/lang/java/pb"
)

func TestProtocolRoundTrip(t *testing.T) {
	testCases := []struct {
		name string
		resp *pb.AnalyzeResponse
	}{
		{
			name: "progress_update",
			resp: &pb.AnalyzeResponse{
				RequestId:   "test-request-1",
				PayloadType: pb.PAYLOAD_PROGRESS,
				Payload: &pb.ProgressUpdate{
					Percentage:     50,
					Phase:          "parsing",
					Message:        "Parsing files...",
					ProcessedFiles: 10,
					TotalFiles:     20,
				},
			},
		},
		{
			name: "class_info",
			resp: &pb.AnalyzeResponse{
				RequestId:   "test-request-2",
				PayloadType: pb.PAYLOAD_CLASS_INFO,
				Payload: &pb.ClassInfo{
					ClassName:       "com.example.TestClass",
					FilePath:        "src/main/java/com/example/TestClass.java",
					ClassType:       pb.ClassType_CLASS_TYPE_CLASS,
					Imports:         []string{"java.util.List"},
					ExtendsTypes:    []string{"BaseClass"},
					ImplementsTypes: []string{"Interface1"},
					StartLine:       10,
					EndLine:         100,
				},
			},
		},
		{
			name: "summary",
			resp: &pb.AnalyzeResponse{
				RequestId:   "test-request-3",
				PayloadType: pb.PAYLOAD_SUMMARY,
				Payload: &pb.Summary{
					TotalTimeMs:     5000,
					LocalClassCount: 100,
					FileCount:       30,
					Success:         true,
					Message:         "Analysis completed",
				},
			},
		},
		{
			name: "error",
			resp: &pb.AnalyzeResponse{
				RequestId:   "test-request-4",
				PayloadType: pb.PAYLOAD_ERROR,
				Payload: &pb.ErrorInfo{
					Code:    pb.ErrorCode_ERROR_PARSE_FAILED,
					Message: "Failed to parse file",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			writer := NewProtocolWriter(buf)
			if err := writer.WriteResponse(tc.resp); err != nil {
				t.Fatalf("Failed to write: %v", err)
			}

			reader := NewProtocolReader(buf)
			outer, err := reader.ReadMessage()
			if err != nil {
				t.Fatalf("Failed to read: %v", err)
			}
			if outer.Type != pb.TYPE_ANALYZE_RESPONSE {
				t.Fatalf("Unexpected outer type: %s", outer.Type)
			}
			result := outer.GetAnalyzeResponse()
			if result == nil {
				t.Fatalf("Expected analyze response payload")
			}

			if result.RequestId != tc.resp.RequestId {
				t.Errorf("RequestId mismatch: got %s, want %s", result.RequestId, tc.resp.RequestId)
			}
			if result.PayloadType != tc.resp.PayloadType {
				t.Errorf("PayloadType mismatch: got %s, want %s", result.PayloadType, tc.resp.PayloadType)
			}
		})
	}
}

func TestMultipleMessages(t *testing.T) {
	messages := []*pb.AnalyzeResponse{
		{RequestId: "1", PayloadType: pb.PAYLOAD_PROGRESS, Payload: &pb.ProgressUpdate{Percentage: 25}},
		{RequestId: "1", PayloadType: pb.PAYLOAD_PROGRESS, Payload: &pb.ProgressUpdate{Percentage: 50}},
		{RequestId: "1", PayloadType: pb.PAYLOAD_SUMMARY, Payload: &pb.Summary{Success: true}},
	}

	buf := &bytes.Buffer{}
	writer := NewProtocolWriter(buf)
	for _, msg := range messages {
		if err := writer.WriteResponse(msg); err != nil {
			t.Fatalf("Failed to write: %v", err)
		}
	}

	reader := NewProtocolReader(buf)
	for i := range messages {
		outer, err := reader.ReadMessage()
		if err != nil {
			t.Fatalf("Failed to read message %d: %v", i, err)
		}
		if outer.Type != pb.TYPE_ANALYZE_RESPONSE {
			t.Fatalf("Message %d: unexpected outer type: %s", i, outer.Type)
		}
		result := outer.GetAnalyzeResponse()
		if result == nil {
			t.Fatalf("Message %d: expected analyze response payload", i)
		}
		if result.RequestId != messages[i].RequestId {
			t.Errorf("Message %d: RequestId mismatch", i)
		}
	}

	_, err := reader.ReadMessage()
	if err != io.EOF {
		t.Errorf("Expected EOF, got: %v", err)
	}
}

func TestWriteRequest(t *testing.T) {
	req := &pb.AnalyzeRequest{
		RequestId: "test-id",
		RepoPath:  "/path/to/repo",
		Config: &pb.AnalyzerConfig{
			ResolveMavenDependencies: true,
			M2RepositoryPath:         "/home/user/.m2/repository",
		},
	}

	buf := &bytes.Buffer{}
	writer := NewProtocolWriter(buf)
	if err := writer.WriteRequest(req); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Buffer should not be empty")
	}

	// 读回去确认外层包装类型正确
	reader := NewProtocolReader(buf)
	outer, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read back: %v", err)
	}
	if outer.Type != pb.TYPE_ANALYZE_REQUEST {
		t.Fatalf("Unexpected outer type: %s", outer.Type)
	}
	gotReq := outer.GetAnalyzeRequest()
	if gotReq == nil {
		t.Fatalf("Expected analyze request payload")
	}
	if gotReq.RequestId != req.RequestId {
		t.Fatalf("RequestId mismatch: got %s, want %s", gotReq.RequestId, req.RequestId)
	}
}

func TestEmptyStream(t *testing.T) {
	buf := &bytes.Buffer{}
	reader := NewProtocolReader(buf)

	_, err := reader.ReadMessage()
	if err != io.EOF {
		t.Errorf("Expected EOF, got: %v", err)
	}
}
