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

package lsp

import "github.com/cloudwego/abcoder/src/lang/uniast"

type Language string

const (
	Rust   Language = "rust"
	Golang Language = "golang"
)

func (l Language) String() string {
	switch l {
	case Rust:
		return "rust"
	case Golang:
		return "go"
	default:
		return "unknown"
	}
}

type LanguageSpec interface {
	// initialize a root workspace, and return all modules [modulename=>abs-path] inside
	WorkSpace(root string) (map[string]string, error)

	// give an absolute file path and returns its module name and package path
	// external path should alse be supported
	// FIXEM: some language (like rust) may have sub-mods inside a file, but we still consider it as a unity mod here
	NameSpace(path string) (string, string, error)

	// tells if a file belang to language AST
	ShouldSkip(path string) bool

	// return the first declaration token of a symbol, as Type-Name
	DeclareTokenOfSymbol(sym DocumentSymbol) int

	// tells if a token is an AST entity
	IsEntityToken(tok Token) bool

	// tells if a token is a std token
	IsStdToken(tok Token) bool

	// return the SymbolKind of a token
	TokenKind(tok Token) SymbolKind

	// tells if a symbol is a main function
	IsMainFunction(sym DocumentSymbol) bool

	// tells if a symbol is a language symbol (func, type, variable, etc) in workspace
	IsEntitySymbol(sym DocumentSymbol) bool

	// tells if a symbol is public in workspace
	IsPublicSymbol(sym DocumentSymbol) bool

	// declare if the language has impl symbol
	// if it return true, the ImplSymbol() will be called
	HasImplSymbol() bool
	// if a symbol is an impl symbol, return the token index of interface type, receiver type and first-method start (-1 means not found)
	// ortherwise the collector will use FunctionSymbol() as receiver type token index (-1 means not found)
	ImplSymbol(sym DocumentSymbol) (int, int, int)

	// if a symbol is a Function or Method symbol,  return the token index of Receiver (-1 means not found),TypeParameters, InputParameters and Outputs
	FunctionSymbol(sym DocumentSymbol) (int, []int, []int, []int)
}

// Patcher is used to patch the AST of a module
type ModulePatcher interface {
	Patch(ast *uniast.Module)
}
