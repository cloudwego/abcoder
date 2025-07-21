/**
 * Copyright 2025 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"testing"
	"time"

	alog "github.com/cloudwego/abcoder/llm/log"
	"github.com/cloudwego/abcoder/llm/tool"

	"github.com/mark3labs/mcp-go/server"
)

func sendAndRecv(t *testing.T, initRequest any, stdinWriter *io.PipeWriter, stdoutReader *io.PipeReader) any {
	requestBytes, err := json.Marshal(initRequest)
	if err != nil {
		t.Fatal(err)
	}
	_, err = stdinWriter.Write(append(requestBytes, '\n'))
	if err != nil {
		t.Fatal(err)
	}

	// Read response
	scanner := bufio.NewScanner(stdoutReader)
	if !scanner.Scan() {
		t.Fatal("failed to read response")
	}
	responseBytes := scanner.Bytes()

	var response any
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	return response
}

func TestASTServer(t *testing.T) {
	alog.SetLogLevel(alog.DebugLevel)
	astOpts := tool.ASTReadToolsOptions{
		RepoASTsDir: "../../testdata/asts",
	}
	svr := NewServer(ServerOptions{
		ServerName:          "abcoder",
		ServerVersion:       "1.0.0",
		ASTReadToolsOptions: astOpts,
	})

	// Create pipes for stdin and stdout
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	// Create server

	stdioServer := server.NewStdioServer(svr.Server)
	stdioServer.SetErrorLogger(log.New(io.Discard, "", 0))

	// Create context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create error channel to catch server errors
	serverErrCh := make(chan error, 1)

	// Start server in goroutine
	go func() {
		err := stdioServer.Listen(ctx, stdinReader, stdoutWriter)
		if err != nil && err != io.EOF && err != context.Canceled {
			serverErrCh <- err
		}
		stdoutWriter.Close()
		close(serverErrCh)
	}()

	time.Sleep(100 * time.Millisecond)

	// Create test message
	initRequest := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"clientInfo": map[string]any{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	// Send request
	resp := sendAndRecv(t, initRequest, stdinWriter, stdoutReader)
	t.Logf("resp %#v", resp)

	// Clean up
	cancel()
	stdinWriter.Close()

	// Check for server errors
	if err := <-serverErrCh; err != nil {
		t.Errorf("unexpected server error: %v", err)
	}
}
