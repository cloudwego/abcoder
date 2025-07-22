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
	"encoding/json"

	"github.com/cloudwego/eino/components/tool"
)

// Tool is the interface for LLM-calling tools.
type Tool interface {
	tool.BaseTool
}

// ToolConfig is the config for LLM-calling tools.
type ToolConfig struct {
	Name string `json:"name"` // name of the tool
	FunctionConfig
}

// FunctionConfig is the config for function
// It can be either builtin function or stdio-plugin or MCP (TODO)
type FunctionConfig struct {
	Type    FuncType        `json:"type"`    // plugin type includes builtin, stdio
	URI     string          `json:"uri"`     // exec-path for stdio, func-URI for func
	Options json.RawMessage `json:"options"` // raw config for specific plugin
}

type FuncType string

const (
	FuncTypeBuiltin FuncType = "builtin"
	FuncTypeStdio   FuncType = "stdio"
)
