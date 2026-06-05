// Copyright 2025 CloudWeGo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package cpp_lsp implements the LanguageServiceProvider interface for
// clangd-backed C++ workspaces. It exists to wire LSP requests that the
// generic LSPClient doesn't issue directly — most importantly the
// textDocument/prepareTypeHierarchy + typeHierarchy/supertypes pair,
// which lets the C++ collector skip its own brittle text-level base
// class parsing.
package cpp_lsp

import (
	"context"

	"github.com/cloudwego/abcoder/lang/lsp"
)

type CppProvider struct{}

// Hover is not currently used by the C++ collector. clangd supports the
// standard textDocument/hover but we have no consumer.
func (p *CppProvider) Hover(ctx context.Context, cli *lsp.LSPClient, uri lsp.DocumentURI, line, character int) (*lsp.Hover, error) {
	params := lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: uri},
		Position:     lsp.Position{Line: line, Character: character},
	}
	var result lsp.Hover
	if err := cli.Call(ctx, "textDocument/hover", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Implementation - textDocument/implementation. clangd implements this
// for resolving derived overrides.
func (p *CppProvider) Implementation(ctx context.Context, cli *lsp.LSPClient, uri lsp.DocumentURI, pos lsp.Position) ([]lsp.Location, error) {
	params := lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: uri},
		Position:     pos,
	}
	var result []lsp.Location
	if err := cli.Call(ctx, "textDocument/implementation", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// WorkspaceSearchSymbols - workspace/symbol.
func (p *CppProvider) WorkspaceSearchSymbols(ctx context.Context, cli *lsp.LSPClient, query string) ([]lsp.SymbolInformation, error) {
	params := struct {
		Query string `json:"query"`
	}{Query: query}
	var result []lsp.SymbolInformation
	if err := cli.Call(ctx, "workspace/symbol", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// PrepareTypeHierarchy - textDocument/prepareTypeHierarchy.
// Given a position on a class identifier, returns the hierarchy item(s)
// rooted at that class.
func (p *CppProvider) PrepareTypeHierarchy(ctx context.Context, cli *lsp.LSPClient, uri lsp.DocumentURI, pos lsp.Position) ([]lsp.TypeHierarchyItem, error) {
	params := lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: uri},
		Position:     pos,
	}
	var result []lsp.TypeHierarchyItem
	if err := cli.Call(ctx, "textDocument/prepareTypeHierarchy", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// TypeHierarchySupertypes - typeHierarchy/supertypes.
// Returns the bases (direct supertypes) of the given hierarchy item.
// This is what we use to replace the text-level BaseClassRefs parsing.
func (p *CppProvider) TypeHierarchySupertypes(ctx context.Context, cli *lsp.LSPClient, item lsp.TypeHierarchyItem) ([]lsp.TypeHierarchyItem, error) {
	params := struct {
		Item lsp.TypeHierarchyItem `json:"item"`
	}{Item: item}
	var result []lsp.TypeHierarchyItem
	if err := cli.Call(ctx, "typeHierarchy/supertypes", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// TypeHierarchySubtypes - typeHierarchy/subtypes.
func (p *CppProvider) TypeHierarchySubtypes(ctx context.Context, cli *lsp.LSPClient, item lsp.TypeHierarchyItem) ([]lsp.TypeHierarchyItem, error) {
	params := struct {
		Item lsp.TypeHierarchyItem `json:"item"`
	}{Item: item}
	var result []lsp.TypeHierarchyItem
	if err := cli.Call(ctx, "typeHierarchy/subtypes", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
