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
	"fmt"

	abutil "github.com/cloudwego/abcoder/internal/utils"
	"github.com/cloudwego/abcoder/lang/patch"
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/cloudwego/abcoder/llm/log"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

const (
	ToolWriteASTNode = "write_ast_node"
)

type ASTWriteToolsOptions struct {
	PatchOptions patch.Options
}

type ASTWriteTools struct {
	opts    ASTWriteToolsOptions
	repo    *uniast.Repository
	patcher *patch.Patcher
	tools   map[string]tool.InvokableTool
}

func NewASTWriteTools(repo *uniast.Repository, opts ASTWriteToolsOptions) *ASTWriteTools {
	ret := &ASTWriteTools{
		repo:    repo,
		opts:    opts,
		patcher: patch.NewPatcher(repo, opts.PatchOptions),
		tools:   map[string]tool.InvokableTool{},
	}

	tt, err := utils.InferTool(string(ToolWriteASTNode),
		"add or modify an ast node inside the repo. If the node is newly-added, the 'file' and 'type' fields are required",
		ret.WriteASTNode)
	if err != nil {
		panic(err)
	}
	ret.tools[string(ToolWriteASTNode)] = tt
	return ret
}

func (t ASTWriteTools) GetTools() []Tool {
	ret := make([]Tool, 0, len(t.tools))
	for _, tt := range t.tools {
		ret = append(ret, tt)
	}
	return ret
}

func (t ASTWriteTools) GetTool(name string) Tool {
	return t.tools[name]
}

type WriteASTNodeReq struct {
	ID        uniast.Identity   `json:"id" jsonschema:"description=the id of the ast node"`
	Codes     string            `json:"codes" jsonschema:"description=the codes of the ast node"`
	Type      string            `json:"type" jsonschema:"description=the type of the ast node, must be enum of 'FUNC'|'TYPE'|'VAR'"`
	File      string            `json:"file,omitempty" jsonschema:"description=the file path for newly-added ast node"`
	AddedDeps []uniast.Identity `json:"added_deps" jsonschema:"description=the added dependencies of the ast node"`
}

type WriteASTNodeResp struct {
	Success    bool              `json:"success" jsonschema:"description=whether the ast node is written successfully"`
	Message    string            `json:"message" jsonschema:"description=the feedback message"`
	References []uniast.Identity `json:"references,omitempty" jsonschema:"description=the references of the ast node"`
}

func (t ASTWriteTools) WriteASTNode(_ context.Context, req WriteASTNodeReq) (*WriteASTNodeResp, error) {
	log.Debug("write ast node, req: %v", abutil.MarshalJSONIndentNoError(req))
	node := t.repo.GetNode(req.ID)
	if node == nil && req.File == "" {
		return nil, fmt.Errorf("file path is required for newly-added ast node")
	}
	var file string
	var typ uniast.NodeType
	if node == nil {
		file = req.File
		typ = uniast.NewNodeType(req.Type)
	} else {
		file = node.FileLine().File
		typ = node.Type
	}
	if err := t.patcher.Patch(patch.Patch{
		Id:    req.ID,
		Codes: req.Codes,
		File:  file,
		Type:  typ,
	}); err != nil {
		return nil, fmt.Errorf("patch node '%s' failed: %v", node.Identity, err)
	}
	if err := t.patcher.Flush(); err != nil {
		return nil, fmt.Errorf("flush patcher failed: %v", err)
	}
	// get git diff of current
	msg := "Write the ast node successfully. Please check if need change References too."
	// diff, err := GitDiff(context.Background(), t.opts.PatchOptions.RepoDir)
	// if err == nil {
	// 	msg += "Current git diff:\n" + diff
	// }
	refs := make([]uniast.Identity, len(node.References))
	for i, ref := range node.References {
		refs[i] = ref.Identity
	}
	resp := &WriteASTNodeResp{
		Success:    true,
		Message:    msg,
		References: refs,
	}
	log.Debug("write ast node, resp: %v", abutil.MarshalJSONIndentNoError(resp))
	return resp, nil
}
