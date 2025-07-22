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

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
)

func NewChatModel(m ModelConfig) (model ChatModel) {
	if m.MaxTokens == 0 {
		m.MaxTokens = 16 * 1024
	}
	var err error
	switch m.APIType {
	case ModelTypeARK:
		model, err = ark.NewChatModel(context.Background(), &ark.ChatModelConfig{
			BaseURL:     m.BaseURL,
			APIKey:      m.APIKey,
			Model:       m.ModelName,
			Temperature: m.Temperature,
			MaxTokens:   &m.MaxTokens,
		})
		if err != nil {
			panic(err)
		}
	case ModelTypeOpenAI:
		model, err = openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
			BaseURL:     m.BaseURL,
			APIKey:      m.APIKey,
			Model:       m.ModelName,
			Temperature: m.Temperature,
			MaxTokens:   &m.MaxTokens,
		})
		if err != nil {
			panic(err)
		}
		return model
	case ModelTypeOllama:
		model, err = ollama.NewChatModel(context.Background(), &ollama.ChatModelConfig{
			BaseURL: m.BaseURL,
			Model:   m.ModelName,
		})
		if err != nil {
			panic(err)
		}
	case ModelTypeClaude:
		model, err = claude.NewChatModel(context.Background(), &claude.Config{
			BaseURL:     &m.BaseURL,
			APIKey:      m.APIKey,
			Model:       m.ModelName,
			Temperature: m.Temperature,
			MaxTokens:   m.MaxTokens,
		})
	default:
		panic("unsupported model type " + m.APIType)
	}
	return
}
