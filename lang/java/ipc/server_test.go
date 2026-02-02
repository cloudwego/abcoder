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
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/abcoder/lang/java/pb"
)

func TestServerConfig(t *testing.T) {
	config := DefaultConfig()

	if config.SocketDir != DefaultSocketDir {
		t.Errorf("Expected SocketDir %s, got %s", DefaultSocketDir, config.SocketDir)
	}

	if config.ConnectTimeout != DefaultConnectTimeout {
		t.Errorf("Expected ConnectTimeout %v, got %v", DefaultConnectTimeout, config.ConnectTimeout)
	}

	if config.ReadTimeout != DefaultReadTimeout {
		t.Errorf("Expected ReadTimeout %v, got %v", DefaultReadTimeout, config.ReadTimeout)
	}
}

func TestNewJavaParserServer(t *testing.T) {
	config := &ServerConfig{
		JarPath:        "/path/to/parser.jar",
		JavaHome:       "/path/to/java",
		SocketDir:      "/tmp",
		ConnectTimeout: 10 * time.Second,
		Debug:          true,
	}

	server := NewJavaParserServer(config)

	if server == nil {
		t.Fatal("Server should not be nil")
	}

	if server.config.JarPath != config.JarPath {
		t.Errorf("JarPath mismatch")
	}

	if server.IsRunning() {
		t.Error("Server should not be running initially")
	}
}

func TestNewJavaParserServerWithNilConfig(t *testing.T) {
	server := NewJavaParserServer(nil)

	if server == nil {
		t.Fatal("Server should not be nil")
	}

	if server.config == nil {
		t.Fatal("Default config should be set")
	}

	if server.config.SocketDir != DefaultSocketDir {
		t.Errorf("Expected default SocketDir")
	}
}

func TestSocketCreation(t *testing.T) {
	config := &ServerConfig{
		SocketDir: os.TempDir(),
	}

	server := NewJavaParserServer(config)

	// Call the internal method to create socket
	err := server.createSocketListener()
	if err != nil {
		t.Fatalf("Failed to create socket listener: %v", err)
	}

	// Check that socket file was created
	socketPath := server.GetSocketPath()
	if socketPath == "" {
		t.Fatal("Socket path should be set")
	}

	// Verify it's a valid unix socket path
	if !strings.HasPrefix(socketPath, os.TempDir()) {
		t.Errorf("Socket should be in temp dir, got: %s", socketPath)
	}

	// Clean up
	server.cleanup()

	// Verify socket file is removed
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Error("Socket file should be removed after cleanup")
	}
}

func TestServerStartWithoutJar(t *testing.T) {
	config := &ServerConfig{
		JarPath:        "/nonexistent/path/to/parser.jar",
		SocketDir:      os.TempDir(),
		ConnectTimeout: 100 * time.Millisecond,
	}

	server := NewJavaParserServer(config)
	ctx := context.Background()

	// This should fail because the JAR doesn't exist
	_, err := server.Start(ctx, "/test/repo", nil)
	if err == nil {
		t.Error("Expected error when JAR doesn't exist")
		server.Stop()
	}
}

func TestServerDoubleStart(t *testing.T) {
	config := &ServerConfig{
		SocketDir: os.TempDir(),
	}

	server := NewJavaParserServer(config)

	// Manually set running to true to simulate a running server
	server.mu.Lock()
	server.running = true
	server.mu.Unlock()

	ctx := context.Background()
	_, err := server.Start(ctx, "/test/repo", nil)

	if err == nil {
		t.Error("Expected error when starting already running server")
	}

	// Clean up
	server.mu.Lock()
	server.running = false
	server.mu.Unlock()
}

func TestServerCleanup(t *testing.T) {
	config := &ServerConfig{
		SocketDir: os.TempDir(),
	}

	server := NewJavaParserServer(config)

	// Create socket
	err := server.createSocketListener()
	if err != nil {
		t.Fatalf("Failed to create socket: %v", err)
	}

	socketPath := server.GetSocketPath()

	// Verify socket exists
	if _, err := os.Stat(socketPath); err != nil {
		t.Fatalf("Socket should exist: %v", err)
	}

	// Call Stop
	server.Stop()

	// Verify cleanup
	if server.IsRunning() {
		t.Error("Server should not be running after Stop")
	}

	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Error("Socket file should be removed after Stop")
	}
}

func TestServerStopIdempotent(t *testing.T) {
	config := &ServerConfig{
		SocketDir: os.TempDir(),
	}

	server := NewJavaParserServer(config)

	// Create socket
	server.createSocketListener()

	// Stop multiple times - should not panic
	server.Stop()
	server.Stop()
	server.Stop()
}

// TestMockJavaConnection tests the server with a mock Java client
func TestMockJavaConnection(t *testing.T) {
	config := &ServerConfig{
		SocketDir:      os.TempDir(),
		ConnectTimeout: 5 * time.Second,
		ReadTimeout:    5 * time.Second,
		Debug:          true,
	}

	server := NewJavaParserServer(config)

	// Create socket listener
	err := server.createSocketListener()
	if err != nil {
		t.Fatalf("Failed to create socket: %v", err)
	}

	socketPath := server.GetSocketPath()
	defer server.Stop()

	// Simulate Java client connecting in a goroutine
	clientDone := make(chan error, 1)
	go func() {
		// Give server time to start listening
		time.Sleep(100 * time.Millisecond)

		// Connect to the socket
		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			clientDone <- err
			return
		}
		defer conn.Close()

		// Read the request from the server (outer message)
		reader := NewProtocolReader(conn)
		_, _ = reader.ReadMessage() // server 会先发 analyze_request，这里读掉即可

		// Write some responses back
		writer := NewProtocolWriter(conn)

		// Send progress update
		progress := &pb.AnalyzeResponse{
			RequestId:   "test-request",
			PayloadType: pb.PAYLOAD_PROGRESS,
			Payload: &pb.ProgressUpdate{
				Percentage: 50,
				Phase:      "parsing",
			},
		}
		if err := writer.WriteResponse(progress); err != nil {
			clientDone <- err
			return
		}

		// Send class info
		classInfo := &pb.AnalyzeResponse{
			RequestId:   "test-request",
			PayloadType: pb.PAYLOAD_CLASS_INFO,
			Payload: &pb.ClassInfo{
				ClassName: "com.example.Test",
				FilePath:  "src/main/java/com/example/Test.java",
				ClassType: pb.ClassType_CLASS_TYPE_CLASS,
			},
		}
		if err := writer.WriteResponse(classInfo); err != nil {
			clientDone <- err
			return
		}

		// Send summary to indicate completion
		summary := &pb.AnalyzeResponse{
			RequestId:   "test-request",
			PayloadType: pb.PAYLOAD_SUMMARY,
			Payload: &pb.Summary{
				Success:         true,
				LocalClassCount: 1,
				TotalTimeMs:     100,
			},
		}
		if err := writer.WriteResponse(summary); err != nil {
			clientDone <- err
			return
		}

		clientDone <- nil

	}()

	// Accept connection on server side
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.acceptConnection(ctx)
	if err != nil {
		t.Fatalf("Failed to accept connection: %v", err)
	}

	// Send request
	err = server.sendAnalyzeRequest("/test/repo", &pb.AnalyzerConfig{})
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read responses
	responseChan := make(chan *pb.AnalyzeResponse, 10)
	go server.readResponses(ctx, responseChan)

	// Collect responses
	var responses []*pb.AnalyzeResponse
	for resp := range responseChan {
		responses = append(responses, resp)
	}

	// Wait for client to finish
	select {
	case err := <-clientDone:
		if err != nil {
			t.Fatalf("Client error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Client timed out")
	}

	// Verify responses
	if len(responses) != 3 {
		t.Errorf("Expected 3 responses, got %d", len(responses))
	}

	// Check response types
	if len(responses) >= 1 {
		if responses[0].GetProgress() == nil {
			t.Error("First response should be progress")
		}
	}
	if len(responses) >= 2 {
		if responses[1].GetClassInfo() == nil {
			t.Error("Second response should be class info")
		}
	}
	if len(responses) >= 3 {
		if responses[2].GetSummary() == nil {
			t.Error("Third response should be summary")
		}
	}
}

func TestAnalyzerConfigConversion(t *testing.T) {
	config := &pb.AnalyzerConfig{
		ResolveMavenDependencies: true,
		M2RepositoryPath:         "/home/user/.m2/repository",
		ExtraJarPaths:            []string{"/lib/extra.jar"},
		IncludeExternalClasses:   true,
	}

	if !config.ResolveMavenDependencies {
		t.Error("ResolveMavenDependencies should be true")
	}

	if config.M2RepositoryPath != "/home/user/.m2/repository" {
		t.Errorf("Unexpected M2RepositoryPath: %s", config.M2RepositoryPath)
	}

	if len(config.ExtraJarPaths) != 1 {
		t.Errorf("Expected 1 extra jar path, got %d", len(config.ExtraJarPaths))
	}

	if !config.IncludeExternalClasses {
		t.Error("IncludeExternalClasses should be true")
	}
}
