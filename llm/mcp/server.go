package mcp

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

import (
	"context"
	"log"

	alog "github.com/cloudwego/abcoder/llm/log"
	"github.com/cloudwego/abcoder/llm/tool"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	Server *server.MCPServer
}

type Tool struct {
	mcp.Tool
	Handler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

type ServerOptions struct {
	ServerName    string
	ServerVersion string
	Verbose       bool
	tool.ASTReadToolsOptions
}

func NewServer(options ServerOptions) *Server {
	opts := []server.ServerOption{
		server.WithPromptCapabilities(false),
		server.WithToolCapabilities(false),
	}
	if options.Verbose {
		opts = append(opts, server.WithLogging())
	}
	// Create a new MCP server
	mcpServer := server.NewMCPServer(options.ServerName, options.ServerVersion, opts...)

	// Enable sampling capability
	// mcpServer.EnableSampling()

	tools := getASTTools(options.ASTReadToolsOptions)
	for _, tool := range tools {
		mcpServer.AddTool(tool.Tool, tool.Handler)
	}

	mcpServer.AddPrompt(mcp.NewPrompt("prompt_analyze_repo", mcp.WithPromptDescription("A prompt for analyzing code repository")), handleAnalyzeRepoPrompt)

	mcpServer.AddNotificationHandler("notification", handleNotification)

	// // Start the stdio server
	// log.Println("Starting sampling example server...")
	// if err := server.ServeStdio(mcpServer); err != nil {
	// 	log.Fatalf("Server error: %v", err)
	// }
	return &Server{
		Server: mcpServer,
	}
}

func handleNotification(
	ctx context.Context,
	notification mcp.JSONRPCNotification,
) {
	log.Printf("Received notification: %s", notification.Method)
}

func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.Server, server.WithErrorLogger(log.Default()))
}

func (s *Server) ServeHTTP(addr string) error {
	httpServer := server.NewStreamableHTTPServer(s.Server, server.WithLogger(alog.NewStdLogger()))
	return httpServer.Start(addr)
}
