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
	"context"
	"encoding/json"

	"github.com/cloudwego/abcoder/internal/utils"
	"github.com/cloudwego/abcoder/llm/prompt"
	"github.com/cloudwego/abcoder/llm/tool"
	"github.com/mark3labs/mcp-go/mcp"
)

// NewTool creates a new MCP tool with the given name, description, schema, and handler function.
// The handler is automatically wrapped to ensure concurrency safety:
//   - Each request receives an isolated context.Context instance
//   - No context values are shared between concurrent requests
//   - The handler can be safely invoked concurrently
// For details on concurrency safety, see CONCURRENCY.md
func NewTool[R any, T any](name string, desc string, schema json.RawMessage, handler func(ctx context.Context, req R) (*T, error)) Tool {
	return Tool{ // get_repo_structure
		Tool: mcp.NewToolWithRawSchema(name, desc, schema),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req R
			if err := request.BindArguments(&req); err != nil {
				return nil, err
			}
			var final string
			var isError bool
			if resp, err := handler(ctx, req); err != nil {
				isError = true
				final = err.Error()
			} else if js, err := utils.MarshalJSONBytes(resp); err != nil {
				isError = true
				final = err.Error()
			} else {
				final = string(js)
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(final),
				},
				IsError: isError,
			}, nil
		},
	}
}

func getASTTools(opts tool.ASTReadToolsOptions) []Tool {
	ast := tool.NewASTReadTools(opts)
	return []Tool{
		NewTool(tool.ToolListRepos, tool.DescListRepos, tool.SchemaListRepos, ast.ListRepos),
		NewTool(tool.ToolGetRepoStructure, tool.DescGetRepoStructure, tool.SchemaGetRepoStructure, ast.GetRepoStructure),
		NewTool(tool.ToolGetPackageStructure, tool.DescGetPackageStructure, tool.SchemaGetPackageStructure, ast.GetPackageStructure),
		NewTool(tool.ToolGetFileStructure, tool.DescGetFileStructure, tool.SchemaGetFileStructure, ast.GetFileStructure),
		NewTool(tool.ToolGetASTNode, tool.DescGetASTNode, tool.SchemaGetASTNode, ast.GetASTNode),
	}
}

func handleAnalyzeRepoPrompt(
	ctx context.Context,
	request mcp.GetPromptRequest,
) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: "A prompt for analyze code repository",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: prompt.PromptAnalyzeRepo,
				},
			},
		},
	}, nil
}
