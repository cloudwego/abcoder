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

package cxx

import (
	lsp "github.com/cloudwego/abcoder/lang/lsp"
)

type CxxSpec struct {
	repo string
}

func NewCxxSpec() *CxxSpec {
	return &CxxSpec{}
}

func (c *CxxSpec) WorkSpace(root string) (map[string]string, error) {
	panic("TODO")
}

func (c *CxxSpec) NameSpace(path string) (string, string, error) {
	panic("TODO")
}

func (c *CxxSpec) ShouldSkip(path string) bool {
	panic("TODO")
}

func (c *CxxSpec) DeclareTokenOfSymbol(sym lsp.DocumentSymbol) int {
	panic("TODO")
}

func (c *CxxSpec) IsEntityToken(tok lsp.Token) bool {
	panic("TODO")
}

func (c *CxxSpec) IsStdToken(tok lsp.Token) bool {
	panic("TODO")
}

func (c *CxxSpec) TokenKind(tok lsp.Token) lsp.SymbolKind {
	panic("TODO")
}

func (c *CxxSpec) IsMainFunction(sym lsp.DocumentSymbol) bool {
	panic("TODO")
}

func (c *CxxSpec) IsEntitySymbol(sym lsp.DocumentSymbol) bool {
	panic("TODO")
}

func (c *CxxSpec) IsPublicSymbol(sym lsp.DocumentSymbol) bool {
	panic("TODO")
}

func (c *CxxSpec) HasImplSymbol() bool {
	panic("TODO")
}

func (c *CxxSpec) ImplSymbol(sym lsp.DocumentSymbol) (int, int, int) {
	panic("TODO")
}

func (c *CxxSpec) FunctionSymbol(sym lsp.DocumentSymbol) (int, []int, []int, []int) {
	panic("TODO")
}

func (c *CxxSpec) GetUnloadedSymbol(from lsp.Token, define lsp.Location) (string, error) {
	panic("TODO")
}
