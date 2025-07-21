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

package agent

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"testing"

	"github.com/cloudwego/abcoder/llm"
	"github.com/cloudwego/abcoder/llm/log"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/schema"
)

func TestAnalyzer(t *testing.T) {
	log.SetLogLevel(log.DebugLevel)

	repoAnnalyzer := NewRepoAnalyzer(context.Background(), RepoAnnalyzerOptions{
		ModelConfig: llm.ModelConfig{
			Name: "claude",
			// Type:      llm.ModelTypeClaude,
			// APIKey:    os.Getenv("CLAUDE_API_KEY"),
			// EndPoint:  "claude-3-7-sonnet-20250219",
			// BaseURL:   "https://api.anthropic.com/",
			Type:      llm.ModelTypeARK,
			APIKey:    os.Getenv("ARK_API_KEY"),
			ModelName: os.Getenv("ARK_DEEPSEEK_V3"),
			MaxTokens: 1024 * 128,
		},
		MaxSteps: 100,
		ASTsDir:  "../../testdata/asts",
	})
	msgs, err := repoAnnalyzer.Generate(context.Background(), []*schema.Message{{
		Role:    schema.User,
		Content: `'localsession'如何异步传递上下文session？`}},
		agent.WithComposeOptions(compose.WithCallbacks(llm.CallbackHandler{})),
	)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", msgs)
}
