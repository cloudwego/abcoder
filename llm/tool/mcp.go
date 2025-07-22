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

package tool

import (
	"context"
	"errors"
	"sync"

	emcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type MCPConfig struct {
	Type   MCPType
	Comand string
	Args   []string
	Envs   []string
	SSEURL string
}

type MCPType string

const (
	MCPTypeStdio MCPType = "stdio"
	MCPTypeSSE   MCPType = "sse"
)

type MCPClient struct {
	cli *client.Client
}

func NewMCPClient(opts MCPConfig) (*MCPClient, error) {
	var cli *client.Client
	var err error
	switch opts.Type {
	case MCPTypeStdio:
		if opts.Comand == "" {
			return nil, errors.New("comand is empty")
		}
		cli, err = client.NewStdioMCPClient(opts.Comand, opts.Envs, opts.Args...)
		if err != nil {
			return nil, err
		}

	case MCPTypeSSE:
		if opts.SSEURL == "" {
			return nil, errors.New("sse url is empty")
		}
		cli, err := client.NewSSEMCPClient(opts.SSEURL)
		if err != nil {
			return nil, err
		}
		return &MCPClient{
			cli: cli,
		}, nil
	default:
		return nil, errors.New("unsupported mcp type")
	}
	return &MCPClient{
		cli: cli,
	}, nil
}

func (c *MCPClient) Start(ctx context.Context) error {
	if err := c.cli.Start(ctx); err != nil {
		return err
	}
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "abcoder",
		Version: "1.0.0",
	}
	_, err := c.cli.Initialize(ctx, initRequest)
	if err != nil {
		return err
	}
	return nil
}

func (c *MCPClient) GetTools(ctx context.Context) ([]Tool, error) {
	mcpTools, err := emcp.GetTools(ctx, &emcp.Config{Cli: c.cli})
	if err != nil {
		return nil, err
	}
	var tools []Tool
	for _, t := range mcpTools {
		tools = append(tools, t)
	}
	return tools, nil
}

var seqThinkCli *MCPClient

func GetSequentialThinkingTools(ctx context.Context) ([]Tool, error) {
	sync.OnceFunc(func() {
		cli, err := NewMCPClient(MCPConfig{
			Type:   MCPTypeStdio,
			Comand: "npx",
			Args: []string{
				"-y",
				"@modelcontextprotocol/server-sequential-thinking",
			},
		})
		if err != nil {
			panic(err)
		}
		err = cli.Start(ctx)
		if err != nil {
			panic(err)
		}
		seqThinkCli = cli
	})()
	return seqThinkCli.GetTools(ctx)
}

var gitCli *MCPClient

func GetGitTools(ctx context.Context) ([]Tool, error) {
	sync.OnceFunc(func() {
		cli, err := NewMCPClient(MCPConfig{
			Type:   MCPTypeStdio,
			Comand: "uvx",
			Args: []string{
				"mcp-server-git",
			},
		})
		if err != nil {
			panic(err)
		}
		err = cli.Start(ctx)
		if err != nil {
			panic(err)
		}
		gitCli = cli
	})()
	return gitCli.GetTools(ctx)
}
