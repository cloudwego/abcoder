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

package cpp

import (
	"fmt"
	"path/filepath"
	"strings"

	lsp "github.com/cloudwego/abcoder/lang/lsp"
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/cloudwego/abcoder/lang/utils"
)

type CppSpec struct {
	repo string
}

func (c *CppSpec) ProtectedSymbolKinds() []lsp.SymbolKind {
	return []lsp.SymbolKind{lsp.SKFunction, lsp.SKMethod, lsp.SKVariable, lsp.SKConstant, lsp.SKClass, lsp.SKStruct}
}

func NewCppSpec() *CppSpec {
	return &CppSpec{}
}

func (c *CppSpec) FileImports(content []byte) ([]uniast.Import, error) {
	return nil, nil
}

// XXX: maybe multi module support for C++?
func (c *CppSpec) WorkSpace(root string) (map[string]string, error) {
	c.repo = root
	rets := map[string]string{}
	absPath, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}
	rets["current"] = absPath
	return rets, nil
}

// returns: modname, pathpath, error
// Multiple symbols with the same name could occur (for example in the Linux kernel).
// The identify is mod::pkg::name. So we use the pkg (the file name) to distinguish them.
func (c *CppSpec) NameSpace(path string, file *uniast.File) (string, string, error) {
	// external lib: only standard library (system headers), in /usr/
	if !strings.HasPrefix(path, c.repo) {
		if strings.HasPrefix(path, "/usr") {
			// assume it is c system library
			return "cstdlib", "cstdlib", nil
		}
		return "external", "external", nil
	}

	relpath, _ := filepath.Rel(c.repo, path)
	return "current", relpath, nil
}

func (c *CppSpec) ShouldSkip(path string) bool {
	if (strings.HasSuffix(path, ".cpp") && !strings.HasSuffix(path, "_test.cpp")) || strings.HasSuffix(path, ".h") {
		return false
	}
	return true
}

func (c *CppSpec) IsDocToken(tok lsp.Token) bool {
	return tok.Type == "comment"
}

func (c *CppSpec) DeclareTokenOfSymbol(sym lsp.DocumentSymbol) int {
	for i, t := range sym.Tokens {
		if c.IsDocToken(t) {
			continue
		}
		for _, m := range t.Modifiers {
			if m == "declaration" {
				return i
			}
		}
	}
	return -1
}

func (c *CppSpec) IsEntityToken(tok lsp.Token) bool {
	for _, m := range tok.Modifiers {
		if m == "declaration" || m == "definition" {
			return false
		}
	}

	return tok.Type == "class" || tok.Type == "function" || tok.Type == "method" || tok.Type == "variable"
}

func (c *CppSpec) IsStdToken(tok lsp.Token) bool {
	panic("TODO")
}

func (c *CppSpec) TokenKind(tok lsp.Token) lsp.SymbolKind {
	switch tok.Type {
	case "class":
		return lsp.SKClass
	case "enum":
		return lsp.SKEnum
	case "enumMember":
		return lsp.SKEnumMember
	case "function", "macro":
		return lsp.SKFunction
	// rust spec does not treat parameter as a variable
	case "parameter":
		return lsp.SKVariable
	case "typeParameter":
		return lsp.SKTypeParameter
	case "method":
		return lsp.SKMethod
	case "namespace":
		return lsp.SKNamespace
	case "variable":
		return lsp.SKVariable
	case "interface", "concept", "modifier", "type", "bracket", "comment", "label", "operator", "property", "unknown":
		return lsp.SKUnknown
	}
	panic(fmt.Sprintf("Weird token type: %s at %+v\n", tok.Type, tok.Location))
}

func (c *CppSpec) IsMainFunction(sym lsp.DocumentSymbol) bool {
	return sym.Kind == lsp.SKFunction && sym.Name == "main"
}

func (c *CppSpec) IsEntitySymbol(sym lsp.DocumentSymbol) bool {
	typ := sym.Kind
	return typ == lsp.SKFunction || typ == lsp.SKMethod || typ == lsp.SKVariable || typ == lsp.SKConstant || typ == lsp.SKClass || typ == lsp.SKStruct
}

func (c *CppSpec) IsPublicSymbol(sym lsp.DocumentSymbol) bool {
	id := c.DeclareTokenOfSymbol(sym)
	if id == -1 {
		return false
	}
	for _, m := range sym.Tokens[id].Modifiers {
		if m == "globalScope" {
			return true
		}
	}
	return false
}

func (c *CppSpec) HasImplSymbol() bool {
	return true
}

func (c *CppSpec) ImplSymbol(sym lsp.DocumentSymbol) (int, int, int) {
	inter := -1
	fn := -1

	// Only treat class/struct as impl container in C++
	if sym.Kind != lsp.SKClass && sym.Kind != lsp.SKStruct {
		return inter, -1, fn
	}

	want := cppShortTypeName(sym.Name)
	if want == "" {
		return inter, -1, fn
	}

	// Prefer type-ish tokens that match the receiver name.
	for i, tok := range sym.Tokens {
		if tok.Text != want {
			continue
		}
		switch tok.Type {
		case "class", "struct":
			return inter, i, fn
		}
	}

	return inter, -1, fn
}

func cppShortTypeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	// Drop namespace qualifiers
	if i := strings.LastIndex(name, "::"); i >= 0 {
		name = name[i+2:]
	}

	// Drop template args
	if i := strings.IndexByte(name, '<'); i >= 0 {
		name = name[:i]
	}

	// Drop leading keywords if they leak into Name (rare)
	name = strings.TrimPrefix(name, "class ")
	name = strings.TrimPrefix(name, "struct ")
	name = strings.TrimSpace(name)

	return name
}

func (c *CppSpec) GetUnloadedSymbol(from lsp.Token, define lsp.Location) (string, error) {
	panic("TODO")
}

func (c *CppSpec) FunctionSymbol(sym lsp.DocumentSymbol) (int, []int, []int, []int) {
	// C++: function or method (and sometimes qualified names are still SKFunction)
	if sym.Kind != lsp.SKFunction && sym.Kind != lsp.SKMethod && !strings.Contains(sym.Name, "::") {
		return -1, nil, nil, nil
	}

	receiver := -1
	typeParams := make([]int, 0, 4)
	inputParams := make([]int, 0, 8)
	outputs := make([]int, 0, 4)

	lines := utils.CountLinesPooled(sym.Text)

	// 1) type params
	for i, tok := range sym.Tokens {
		if tok.Type == "typeParameter" {
			typeParams = append(typeParams, i)
		}
	}

	// 2) find name token (method/function)
	nameTokIdx := -1
	for i, tok := range sym.Tokens {
		if tok.Type == "method" || tok.Type == "function" {
			nameTokIdx = i
			break
		}
	}
	if nameTokIdx < 0 {
		return -1, typeParams, nil, nil
	}

	// 3) receiver: parse from qualified name "Person::SayHi" -> "Person"
	recvShort := receiverShortName(sym.Name)
	if recvShort != "" {
		for i := 0; i < nameTokIdx; i++ { // receiver must be before method name in signature
			tok := sym.Tokens[i]
			if tok.Text != recvShort {
				continue
			}
			// prefer type-ish token kinds for receiver
			if tok.Type == "class" || tok.Type == "struct" || tok.Type == "type" {
				receiver = i
				break
			}
		}
		if receiver < 0 {
			for i := 0; i < nameTokIdx; i++ {
				tok := sym.Tokens[i]
				if tok.Text == recvShort && c.IsEntityToken(tok) {
					receiver = i
					break
				}
			}
		}
	}

	nameOff := lsp.RelativePostionWithLines(*lines, sym.Location.Range.Start, sym.Tokens[nameTokIdx].Location.Range.Start)

	// 4) find params bounds
	paramL, paramR := -1, -1
	if nameOff >= 0 && nameOff < len(sym.Text) {
		open := strings.Index(sym.Text[nameOff:], "(")
		if open >= 0 {
			paramL = nameOff + open
			paramR = findMatchingParen(sym.Text, paramL)
		}
	}

	// 5) classify tokens
	for i, tok := range sym.Tokens {
		if !c.IsEntityToken(tok) {
			continue
		}
		if tok.Type == "typeParameter" || tok.Type == "namespace" {
			continue
		}

		off := lsp.RelativePostionWithLines(*lines, sym.Location.Range.Start, tok.Location.Range.Start)

		// inputs
		if paramL >= 0 && paramR >= 0 && off > paramL && off <= paramR {
			inputParams = append(inputParams, i)
			continue
		}

		// outputs: before name token, excluding receiver
		if off >= 0 && off < nameOff {
			if i == receiver {
				continue
			}
			outputs = append(outputs, i)
		}
	}

	return receiver, typeParams, inputParams, outputs
}

func receiverShortName(qualified string) string {
	parts := strings.Split(qualified, "::")
	if len(parts) < 2 {
		return ""
	}
	recv := parts[len(parts)-2]
	if i := strings.IndexByte(recv, '<'); i >= 0 { // Foo<T> -> Foo
		recv = recv[:i]
	}
	return strings.TrimSpace(recv)
}

func findMatchingParen(s string, openIdx int) int {
	if openIdx < 0 || openIdx >= len(s) || s[openIdx] != '(' {
		return -1
	}
	depth := 0
	for i := openIdx; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
