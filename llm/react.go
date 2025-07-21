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

	"github.com/cloudwego/abcoder/internal/utils"
	"github.com/cloudwego/abcoder/llm/log"
	"github.com/cloudwego/abcoder/llm/prompt"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

var _ Generator = (*ReactAgent)(nil)

type ReactAgent struct {
	opts ReactAgentOptions
	*react.Agent
}

type ReactAgentOptions struct {
	SysPrompt prompt.Prompt `json:"-"`
	*react.AgentConfig
}

func NewReactAgent(name string, opts ReactAgentOptions) *ReactAgent {
	if opts.AgentConfig.MessageModifier == nil {
		opts.AgentConfig.MessageModifier = newMessageModifier(opts.SysPrompt.String(), name, opts.AgentConfig.MaxStep)
	}
	agent, err := react.NewAgent(context.Background(), opts.AgentConfig)
	if err != nil {
		panic(err)
	}
	return &ReactAgent{
		opts:  opts,
		Agent: agent,
	}
}

func newMessageModifier(sysPrompt string, name string, limit int) func(ctx context.Context, input []*schema.Message) []*schema.Message {
	return func(ctx context.Context, input []*schema.Message) []*schema.Message {
		log.Debug("newMessageModifier, name: %v, limit: %d, input: %v", name, limit, len(input))
		if limit > 0 && len(input) >= limit-1 {
			input = append(input, schema.UserMessage("当前迭代次数已达最大值，请立即输出结论，不要继续调用工具!"))
		}
		res := appendSysPrompt(sysPrompt, input)
		// res = summaryMessagesOndemands(ctx, chat, res)
		return res
	}
}

func appendSysPrompt(sysPrompt string, input []*schema.Message) []*schema.Message {
	res := make([]*schema.Message, 0, len(input)+1)
	res = append(res, schema.SystemMessage(sysPrompt))
	res = append(res, input...)
	return res
}

func (p *ReactAgent) Call(ctx context.Context, input string) (string, error) {
	// 初始化输入
	// sysMsg := schema.SystemMessage(p.opts.SysPrompt)
	// log.Debug("[SysPrompt] %s", p.opts.SysPrompt)
	inputMsg := schema.UserMessage(input)
	log.Debug("[User] %s", input)
	inputMsgs := []*schema.Message{inputMsg}
	out, err := p.Generate(ctx, inputMsgs, agent.WithComposeOptions(compose.WithCallbacks(CallbackHandler{})))
	if err != nil {
		return "", utils.WrapError(err, "ReactAgent RoundTrip error")
	}
	return out.Content, nil
}

/*
	type Handler interface {
		OnStart(ctx context.Context, info *RunInfo, input CallbackInput) context.Context
		OnEnd(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context

		OnError(ctx context.Context, info *RunInfo, err error) context.Context

		OnStartWithStreamInput(ctx context.Context, info *RunInfo,
			input *schema.StreamReader[CallbackInput]) context.Context
		OnEndWithStreamOutput(ctx context.Context, info *RunInfo,
			output *schema.StreamReader[CallbackOutput]) context.Context
	}
*/

type CallbackHandler struct{}

var _ callbacks.Handler = (*CallbackHandler)(nil)

func (h CallbackHandler) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	log.Debug("<OnStart>\n\tINFO: %+v", info)
	// if cb, ok := input.(*model.CallbackInput); ok {
	// 	for _, t := range cb.Tools {
	// 		log.Debug("\tTOOL: %#v", t)
	// 		// if t.ParamsOneOf != nil {
	// 		// 	desc, _ := t.ParamsOneOf.ToOpenAPIV3()
	// 		// 	js, _ := desc.MarshalJSON()
	// 		// 	log.Debug("\tParams: %s", js)
	// 		// }
	// 	}
	// }
	log.Debug("</OnStart>")
	return ctx
}

func (h CallbackHandler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	log.Debug("<OnEnd>\n\tINFO %+v\n\tOUTPUT: %v\n</OnEnd>", info, output)
	// Implementation for handling end of callback
	return ctx
}

func (h CallbackHandler) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	log.Error("<OnError>\n\tINFO: %+v\n\tERROR: %v\n</OnError>", info, err)
	// Implementation for handling errors
	return ctx
}

func (h CallbackHandler) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo,
	input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	// Implementation for handling stream input start
	return ctx
}

func (h CallbackHandler) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo,
	output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	// Implementation for handling stream output end
	return ctx
}
