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

package llm

import (
	"context"
	"strings"

	"github.com/cloudwego/abcoder/llm/prompt"
	"github.com/cloudwego/abcoder/llm/tool"
	"github.com/cloudwego/eino/components/model"
	etool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
)

type ModelConfig struct {
	Name        string    `json:"name"` // alias of the config, not endpoint!
	APIType     ModelType `json:"type"`
	BaseURL     string    `json:"base_url"`
	APIKey      string    `json:"api_key"`
	ModelName   string    `json:"model_name"` // the endpoint of the model, like `claude-opus-4-20250514`
	Temperature *float32  `json:"temperature"`
	// TopP        *float32  `json:"top_p"`
	MaxTokens int `json:"max_tokens"`
}

type ModelType string

func NewModelType(t string) ModelType {
	switch strings.ToLower(t) {
	case "ollama":
		return ModelTypeOllama
	case "ark":
		return ModelTypeARK
	case "openai":
		return ModelTypeOpenAI
	case "claude":
		return ModelTypeClaude
	}
	return ModelTypeUnknown
}

const (
	ModelTypeUnknown ModelType = ""
	ModelTypeOllama  ModelType = "ollama"
	ModelTypeARK     ModelType = "ark"
	ModelTypeOpenAI  ModelType = "openai" // Fixed typo in constant name
	ModelTypeClaude  ModelType = "claude"
)

type AgentConfig struct {
	WithModel string        `json:"with_model"`
	WithTools []string      `json:"with_tools"`
	MaxSteps  int           `json:"max_steps"`
	Prompt    prompt.Prompt `json:"prompt"`
}

// Generator is the interface for calling
type Generator interface {
	// Call calls the LLM with the input.
	Call(ctx context.Context, input string) (string, error)
}

// ChatModel is the interface for making LLM backend.
type ChatModel interface {
	model.ToolCallingChatModel
}

func MakeAgent(source any, sysPrompt prompt.Prompt, models map[string]ChatModel, tools map[string]tool.Tool, executor AgentConfig) Generator {
	if len(executor.WithModel) == 0 {
		panic("executor model must be set")
	}

	ts := make([]tool.Tool, 0, len(executor.WithTools))
	for _, tn := range executor.WithTools {
		t, ok := tools[tn]
		if !ok {
			panic("tool " + tn + " not found")
		}
		ts = append(ts, t)
	}
	exeName := executor.WithModel
	exeModel, ok := models[exeName]
	if !ok {
		panic("model " + exeName + " not found")
	}
	tcfg := compose.ToolsNodeConfig{}
	for _, t := range ts {
		tcfg.Tools = append(tcfg.Tools, t.(etool.BaseTool))
	}
	agent := NewReactAgent("", ReactAgentOptions{
		SysPrompt: sysPrompt,
		AgentConfig: &react.AgentConfig{
			ToolCallingModel: exeModel,
			ToolsConfig:      tcfg,
			MaxStep:          executor.MaxSteps,
			MessageModifier:  newMessageModifier(sysPrompt.String(), exeName, executor.MaxSteps),
		},
	})
	return agent
}
