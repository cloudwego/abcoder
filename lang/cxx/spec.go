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
	"fmt"
	"path/filepath"
	"strings"

	lsp "github.com/cloudwego/abcoder/lang/lsp"
	"github.com/cloudwego/abcoder/lang/utils"
)

type CxxSpec struct {
	repo string
}

func NewCxxSpec() *CxxSpec {
	return &CxxSpec{}
}

// XXX: maybe multi module support for C++?
func (c *CxxSpec) WorkSpace(root string) (map[string]string, error) {
	c.repo = root
	rets := map[string]string{}
	absPath, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}
	rets["current"] = absPath
	return rets, nil
}

// returns: mod, path, error
func (c *CxxSpec) NameSpace(path string) (string, string, error) {
	// external lib: only standard library (system headers), in /usr/
	if !strings.HasPrefix(path, c.repo) {
		if strings.HasPrefix(path, "/usr") {
			// assume it is c system library
			return "cstdlib", "cstdlib", nil
		}
		panic(fmt.Sprintf("external lib: %s\n", path))
	}

	return "current", "current", nil

}

func (c *CxxSpec) ShouldSkip(path string) bool {
	if strings.HasSuffix(path, ".c") || strings.HasSuffix(path, ".h") {
		return false
	}
	return true
}

func (c *CxxSpec) IsDocToken(tok lsp.Token) bool {
	return tok.Type == "comment"
}

func (c *CxxSpec) DeclareTokenOfSymbol(sym lsp.DocumentSymbol) int {
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

func (c *CxxSpec) IsEntityToken(tok lsp.Token) bool {
	return tok.Type == "class" || tok.Type == "function" || tok.Type == "variable"
}

func (c *CxxSpec) IsStdToken(tok lsp.Token) bool {
	panic("TODO")
}

func (c *CxxSpec) TokenKind(tok lsp.Token) lsp.SymbolKind {
	switch tok.Type {
	case "class":
		return lsp.SKStruct
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
	// type: TODO
	case "interface", "concept", "method", "modifier", "namespace", "type":
		panic(fmt.Sprintf("Unsupported token type: %s at %+v\n", tok.Type, tok.Location))
	case "bracket", "comment", "label", "operator", "property", "unknown":
		return lsp.SKUnknown
	}
	panic(fmt.Sprintf("Weird token type: %s at %+v\n", tok.Type, tok.Location))
}

func (c *CxxSpec) IsMainFunction(sym lsp.DocumentSymbol) bool {
	return sym.Kind == lsp.SKFunction && sym.Name == "main"
}

func (c *CxxSpec) IsEntitySymbol(sym lsp.DocumentSymbol) bool {
	typ := sym.Kind
	return typ == lsp.SKFunction || typ == lsp.SKVariable || typ == lsp.SKClass

}

func (c *CxxSpec) IsPublicSymbol(sym lsp.DocumentSymbol) bool {
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

// TODO(cpp): support C++ OOP
func (c *CxxSpec) HasImplSymbol() bool {
	return false
}

func (c *CxxSpec) ImplSymbol(sym lsp.DocumentSymbol) (int, int, int) {
	panic("TODO")
}

func (c *CxxSpec) FunctionSymbol(sym lsp.DocumentSymbol) (int, []int, []int, []int) {
	// No receiver and no type params for C
	if sym.Kind != lsp.SKFunction {
		return -1, nil, nil, nil
	}
	receiver := -1
	typeParams := []int{}
	inputParams := []int{}
	outputs := []int{}

	// general format: RETURNVALUE NAME "("  PARAMS  ")"  BODY
	//                             --------
	//                             fnNameText
	// state machine   phase 0           phase 1        phase 2: break
	// TODO: attributes may contain parens. also inline structs.

	endRelOffset := 0
	lines := utils.CountLinesCached(sym.Text)
	phase := 0
	for i, tok := range sym.Tokens {
		switch phase {
		case 0:
			if tok.Type == "function" {
				offset := lsp.RelativePostionWithLines(*lines, sym.Location.Range.Start, tok.Location.Range.Start)
				endRelOffset = offset + strings.Index(sym.Text[offset:], ")")
				phase = 1
				continue
			}
			if c.IsEntityToken(tok) {
				outputs = append(outputs, i)
			}
		case 1:
			offset := lsp.RelativePostionWithLines(*lines, sym.Location.Range.Start, tok.Location.Range.Start)
			if offset > endRelOffset {
				phase = 2
				continue
			}
			if c.IsEntityToken(tok) {
				inputParams = append(inputParams, i)
			}
		}
	}
	return receiver, typeParams, inputParams, outputs
}

func (c *CxxSpec) GetUnloadedSymbol(from lsp.Token, define lsp.Location) (string, error) {
	panic("TODO")
}
