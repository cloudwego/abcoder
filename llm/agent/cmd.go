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
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/abcoder/llm"
	"github.com/cloudwego/abcoder/llm/log"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/schema"
)

type AgentOptions struct {
	ASTsDir      string
	MaxHistories int
	MaxSteps     int
	Model        llm.ModelConfig
}

type Agent struct {
	opts      AgentOptions
	analyzer  *llm.ReactAgent
	histories *Histories
}

// run agent as a repl cmd server
func NewAgent(opts AgentOptions) *Agent {
	ag := NewRepoAnalyzer(context.Background(), RepoAnnalyzerOptions{
		ASTsDir:     opts.ASTsDir,
		MaxSteps:    opts.MaxSteps,
		ModelConfig: opts.Model,
	})

	histories := NewHistories(opts.MaxHistories)

	return &Agent{
		opts:      opts,
		analyzer:  ag,
		histories: histories,
	}
}

func (a *Agent) Generate(ctx context.Context, msgs []*schema.Message) (*schema.Message, error) {
	return a.analyzer.Generate(ctx, msgs, agent.WithComposeOptions(compose.WithCallbacks(llm.CallbackHandler{})))
}

func (a *Agent) Run(ctx context.Context) {
	fmt.Fprintf(os.Stdout, "Hello! I'm ABCoder, your coding assistant. What can I do for you today?\n")

	sc := bufio.NewScanner(os.Stdin)

	for sc.Scan() {

		query := strings.TrimSpace(sc.Text())
		if query == "" {
			continue
		}
		if query == "exit" {
			break
		}

		// get histories
		a.histories.Add(&schema.Message{
			Role:    schema.User,
			Content: query,
		})

		resp, err := a.Generate(ctx, a.histories.Get())
		if err != nil {
			log.Error("Failed to run agent: %v\n", err)
			continue
		}

		a.histories.Add(resp)

		fmt.Fprintf(os.Stdout, "\n%s\n", resp.Content)
	}
}

type Histories struct {
	max int
	hs  []*schema.Message
}

func NewHistories(max int) *Histories {
	return &Histories{
		max: max,
		hs:  make([]*schema.Message, 0, max),
	}
}

func (h *Histories) Add(msg *schema.Message) {
	if len(h.hs) >= h.max {
		h.hs = h.hs[1:]
	}
	h.hs = append(h.hs, msg)
}

func (h *Histories) Get() []*schema.Message {
	return h.hs
}
