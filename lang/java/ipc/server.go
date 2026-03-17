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
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/cloudwego/abcoder/lang/java/pb"
	"github.com/google/uuid"
)

const (
	// DefaultSocketDir is the default directory for Unix Domain Socket files
	DefaultSocketDir = "/tmp"

	// DefaultConnectTimeout is the timeout for Java process to connect
	DefaultConnectTimeout = 30 * time.Second

	// DefaultReadTimeout is the timeout for reading individual messages
	DefaultReadTimeout = 5 * time.Minute
)

// ServerConfig holds configuration for the Java Parser server
type ServerConfig struct {
	// JarPath is the path to the Java Parser JAR file
	JarPath string

	// JavaHome is the path to Java installation (optional)
	JavaHome string

	// SocketDir is the directory for Unix Domain Socket files
	SocketDir string

	// ConnectTimeout is the timeout for Java process to connect
	ConnectTimeout time.Duration

	// ReadTimeout is the timeout for reading messages
	ReadTimeout time.Duration

	// Debug enables debug logging
	Debug bool
}

// DefaultConfig returns a default server configuration
func DefaultConfig() *ServerConfig {
	return &ServerConfig{
		SocketDir:      DefaultSocketDir,
		ConnectTimeout: DefaultConnectTimeout,
		ReadTimeout:    DefaultReadTimeout,
		Debug:          false,
	}
}

// JavaParserServer manages the Java Parser subprocess and IPC communication
type JavaParserServer struct {
	config     *ServerConfig
	socketPath string
	listener   *net.UnixListener
	javaCmd    *exec.Cmd
	conn       *net.UnixConn

	mu       sync.Mutex
	running  bool
	stopOnce sync.Once
}

// NewJavaParserServer creates a new Java Parser server with the given configuration
func NewJavaParserServer(config *ServerConfig) *JavaParserServer {
	if config == nil {
		config = DefaultConfig()
	}
	return &JavaParserServer{
		config: config,
	}
}

// Start initializes the UDS listener and starts the Java subprocess.
func (s *JavaParserServer) Start(ctx context.Context, repoPath string, analyzerConfig *pb.AnalyzerConfig) (<-chan *pb.AnalyzeResponse, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil, fmt.Errorf("server is already running")
	}
	s.running = true
	s.mu.Unlock()

	// Step 1: Create Unix Domain Socket listener
	if err := s.createSocketListener(); err != nil {
		s.cleanup()
		return nil, fmt.Errorf("failed to create socket listener: %w", err)
	}

	// Step 2: Start Java subprocess
	if err := s.startJavaProcess(ctx); err != nil {
		s.cleanup()
		return nil, fmt.Errorf("failed to start Java process: %w", err)
	}

	// Step 3: Accept connection from Java process
	if err := s.acceptConnection(ctx); err != nil {
		s.cleanup()
		return nil, fmt.Errorf("failed to accept connection: %w", err)
	}

	// Step 4: Send analyze request
	if err := s.sendAnalyzeRequest(repoPath, analyzerConfig); err != nil {
		s.cleanup()
		return nil, fmt.Errorf("failed to send analyze request: %w", err)
	}

	// Step 5: Start reading responses in a goroutine
	responseChan := make(chan *pb.AnalyzeResponse, 100)
	go s.readResponses(ctx, responseChan)

	return responseChan, nil
}

// createSocketListener creates a Unix Domain Socket listener
func (s *JavaParserServer) createSocketListener() error {
	// Generate unique socket path
	socketName := fmt.Sprintf("java-parser-%s.sock", uuid.New().String()[:8])
	s.socketPath = filepath.Join(s.config.SocketDir, socketName)

	// Remove existing socket file if present
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing socket file: %w", err)
	}

	// Create Unix Domain Socket listener
	addr := &net.UnixAddr{Name: s.socketPath, Net: "unix"}
	listener, err := net.ListenUnix("unix", addr)
	if err != nil {
		return fmt.Errorf("failed to create Unix socket listener: %w", err)
	}

	s.listener = listener

	// Always log socket path
	log.Printf("[JavaParserServer] Socket listener created at %s", s.socketPath)

	return nil
}

// startJavaProcess starts the Java Parser JAR as a subprocess
func (s *JavaParserServer) startJavaProcess(ctx context.Context) error {
	// Determine Java command
	javaCmd := "java"
	if s.config.JavaHome != "" {
		javaCmd = filepath.Join(s.config.JavaHome, "bin", "java")
	}

	// Build command arguments
	args := []string{
		"-jar", s.config.JarPath,
		"--uds", s.socketPath,
	}

	// Create command with context for cancellation
	s.javaCmd = exec.CommandContext(ctx, javaCmd, args...)

	// Redirect stdout and stderr to Go's console for debugging
	s.javaCmd.Stdout = os.Stdout
	s.javaCmd.Stderr = os.Stderr

	// Start the Java process
	if err := s.javaCmd.Start(); err != nil {
		return fmt.Errorf("failed to start Java process: %w", err)
	}

	if s.config.Debug {
		log.Printf("[JavaParserServer] Java process started with PID %d", s.javaCmd.Process.Pid)
	}

	// Monitor process in background
	go func() {
		if err := s.javaCmd.Wait(); err != nil {
			if s.config.Debug {
				log.Printf("[JavaParserServer] Java process exited: %v", err)
			}
		}
	}()

	return nil
}

// acceptConnection waits for the Java process to connect
func (s *JavaParserServer) acceptConnection(ctx context.Context) error {
	// Set accept deadline
	deadline := time.Now().Add(s.config.ConnectTimeout)
	if err := s.listener.SetDeadline(deadline); err != nil {
		return fmt.Errorf("failed to set listener deadline: %w", err)
	}

	// Accept connection
	conn, err := s.listener.AcceptUnix()
	if err != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return fmt.Errorf("failed to accept connection: %w", err)
		}
	}

	s.conn = conn

	if s.config.Debug {
		log.Printf("[JavaParserServer] Connection accepted from Java process")
	}

	return nil
}

// sendAnalyzeRequest sends the analyze request to the Java process
func (s *JavaParserServer) sendAnalyzeRequest(repoPath string, config *pb.AnalyzerConfig) error {
	if config == nil {
		config = &pb.AnalyzerConfig{}
	}

	request := &pb.AnalyzeRequest{
		RequestId: uuid.New().String(),
		RepoPath:  repoPath,
		Config:    config,
	}

	writer := NewProtocolWriter(s.conn)
	writer.SetDebug(true) // Enable debug logging for sent messages
	if err := writer.WriteRequest(request); err != nil {
		return fmt.Errorf("failed to write analyze request: %w", err)
	}

	log.Printf("[JavaParserServer] Analyze request sent for repo: %s", repoPath)

	return nil
}

// readResponses reads responses from the Java process and sends them to the channel
func (s *JavaParserServer) readResponses(ctx context.Context, responseChan chan<- *pb.AnalyzeResponse) {
	defer close(responseChan)
	defer s.cleanup()

	reader := NewProtocolReader(s.conn)
	reader.SetDebug(true) // Enable debug logging for received messages

	for {
		select {
		case <-ctx.Done():
			if s.config.Debug {
				log.Printf("[JavaParserServer] Context cancelled, stopping response reader")
			}
			return
		default:
		}

		// Set read deadline
		if s.config.ReadTimeout > 0 {
			if err := s.conn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout)); err != nil {
				log.Printf("[JavaParserServer] Failed to set read deadline: %v", err)
			}
		}

		// Read next message
		outer, err := reader.ReadMessage()
		if err != nil {
			if err == io.EOF {
				if s.config.Debug {
					log.Printf("[JavaParserServer] End of stream reached")
				}
				return
			}

			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Printf("[JavaParserServer] Read timeout, continuing...")
				continue
			}

			log.Printf("[JavaParserServer] Error reading message: %v", err)
			return
		}

		// 只处理 analyze_response，其它消息（heartbeat/stop 等）忽略
		if outer == nil || outer.Type != pb.TYPE_ANALYZE_RESPONSE {
			continue
		}
		resp := outer.GetAnalyzeResponse()
		if resp == nil {
			continue
		}

		// Send message to channel
		select {
		case responseChan <- resp:
			if s.config.Debug {
				if prog := resp.GetProgress(); prog != nil {
					log.Printf("[JavaParserServer] Progress: %d%% - %s", prog.Percentage, prog.Message)
				}
			}

			// Check for summary (indicates completion)
			if sum := resp.GetSummary(); sum != nil {
				if s.config.Debug {
					log.Printf("[JavaParserServer] Analysis complete: %d classes, %d files in %dms",
						sum.LocalClassCount, sum.FileCount, sum.TotalTimeMs)
				}
				return
			}

			// Check for errors
			if errInfo := resp.GetError(); errInfo != nil {
				code := string(errInfo.Code)
				if code == "" {
					code = "unknown"
				}
				log.Printf("[JavaParserServer] Error from Java parser: %s - %s", code, errInfo.Message)
			}

		case <-ctx.Done():
			return
		}
	}
}

// Stop gracefully stops the server and cleans up resources
func (s *JavaParserServer) Stop() {
	s.stopOnce.Do(func() {
		s.cleanup()
	})
}

// cleanup releases all resources
func (s *JavaParserServer) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}

	if s.listener != nil {
		s.listener.Close()
		s.listener = nil
	}

	if s.socketPath != "" {
		os.Remove(s.socketPath)
	}

	if s.javaCmd != nil && s.javaCmd.Process != nil {
		s.javaCmd.Process.Kill()
		s.javaCmd = nil
	}

	s.running = false

	if s.config.Debug {
		log.Printf("[JavaParserServer] Cleanup complete")
	}
}

// GetSocketPath returns the current socket path
func (s *JavaParserServer) GetSocketPath() string {
	return s.socketPath
}

// IsRunning returns whether the server is currently running
func (s *JavaParserServer) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
