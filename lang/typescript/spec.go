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

package typescript

import (
	"fmt"
	"path/filepath"
	"strings"

	lsp "github.com/cloudwego/abcoder/lang/lsp"
	"github.com/cloudwego/abcoder/lang/uniast"
)

var _ lsp.LanguageSpec = (*TypeScriptSpec)(nil)

type TypeScriptSpec struct {
	repo string
}

func NewTypeScriptSpec() *TypeScriptSpec {
	return &TypeScriptSpec{}
}

func (c *TypeScriptSpec) FileImports(content []byte) ([]uniast.Import, error) {
	// TODO: Parse TypeScript import statements
	return []uniast.Import{}, nil
}

func (c *TypeScriptSpec) IsExternalEntityToken(tok lsp.Token) bool {
	if !c.IsEntityToken(tok) {
		return false
	}
	for _, m := range tok.Modifiers {
		if m == "defaultLibrary" {
			return true
		}
	}
	return false
}

func (c *TypeScriptSpec) TokenKind(tok lsp.Token) lsp.SymbolKind {
	switch tok.Type {
	case "class":
		return lsp.SKClass
	case "interface":
		return lsp.SKInterface
	case "function":
		return lsp.SKFunction
	case "method":
		return lsp.SKMethod
	case "property":
		return lsp.SKProperty
	case "variable":
		return lsp.SKVariable
	case "const":
		return lsp.SKConstant
	case "enum":
		return lsp.SKEnum
	case "enumMember":
		return lsp.SKEnumMember
	case "type":
		return lsp.SKTypeParameter
	case "namespace":
		return lsp.SKNamespace
	case "module":
		return lsp.SKModule
	default:
		return lsp.SKUnknown
	}
}

func (c *TypeScriptSpec) IsStdToken(tok lsp.Token) bool {
	for _, m := range tok.Modifiers {
		if m == "defaultLibrary" {
			return true
		}
	}
	return false
}

func (c *TypeScriptSpec) IsDocToken(tok lsp.Token) bool {
	for _, m := range tok.Modifiers {
		if m == "documentation" {
			return true
		}
	}
	return false
}

func (c *TypeScriptSpec) DeclareTokenOfSymbol(sym lsp.DocumentSymbol) int {
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

func (c *TypeScriptSpec) IsPublicSymbol(sym lsp.DocumentSymbol) bool {
	// In TypeScript, symbols are public by default unless marked private/protected
	id := c.DeclareTokenOfSymbol(sym)
	if id == -1 {
		return true
	}
	for _, m := range sym.Tokens[id].Modifiers {
		if m == "private" || m == "protected" {
			return false
		}
	}
	return true
}

func (c *TypeScriptSpec) IsMainFunction(sym lsp.DocumentSymbol) bool {
	// TypeScript doesn't have a main function concept
	return false
}

func (c *TypeScriptSpec) IsEntitySymbol(sym lsp.DocumentSymbol) bool {
	typ := sym.Kind
	return typ == lsp.SKClass || typ == lsp.SKMethod || typ == lsp.SKFunction || 
		typ == lsp.SKVariable || typ == lsp.SKInterface || typ == lsp.SKConstant ||
		typ == lsp.SKEnum || typ == lsp.SKTypeParameter || typ == lsp.SKNamespace ||
		typ == lsp.SKModule
}

func (c *TypeScriptSpec) IsEntityToken(tok lsp.Token) bool {
	typ := tok.Type
	return typ == "class" || typ == "interface" || typ == "function" || 
		typ == "method" || typ == "property" || typ == "variable" || 
		typ == "const" || typ == "enum" || typ == "enumMember" || 
		typ == "type" || typ == "namespace" || typ == "module"
}

func (c *TypeScriptSpec) HasImplSymbol() bool {
	// TypeScript uses class/interface implementation, not impl blocks like Rust
	return false
}

func (c *TypeScriptSpec) ImplSymbol(sym lsp.DocumentSymbol) (int, int, int) {
	// TypeScript doesn't have impl blocks
	return -1, -1, -1
}

func (c *TypeScriptSpec) FunctionSymbol(sym lsp.DocumentSymbol) (int, []int, []int, []int) {
	// TODO: Implement TypeScript function parsing
	return -1, nil, nil, nil
}

func (c *TypeScriptSpec) ShouldSkip(path string) bool {
	if strings.Contains(path, "/node_modules/") {
		return true
	}
	if !strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".tsx") {
		return true
	}
	return false
}

func (c *TypeScriptSpec) NameSpace(path string) (string, string, error) {
	if !strings.HasPrefix(path, c.repo) {
		// External module
		return "", "", fmt.Errorf("external module: %s", path)
	}

	// Calculate relative path from repo root
	rel, err := filepath.Rel(c.repo, path)
	if err != nil {
		return "", "", err
	}

	// Remove file extension
	rel = strings.TrimSuffix(rel, ".ts")
	rel = strings.TrimSuffix(rel, ".tsx")
	
	// Remove index suffix if present
	if strings.HasSuffix(rel, "/index") {
		rel = strings.TrimSuffix(rel, "/index")
	}

	// Convert path to module name
	module := strings.ReplaceAll(rel, string(filepath.Separator), ".")
	
	return module, module, nil
}

func (c *TypeScriptSpec) WorkSpace(root string) (map[string]string, error) {
	c.repo = root
	// For TypeScript, we don't need to collect modules like Rust
	// The module system is based on file paths
	return map[string]string{}, nil
}

func (c *TypeScriptSpec) GetUnloadedSymbol(from lsp.Token, loc lsp.Location) (string, error) {
	// TODO: Implement TypeScript unloaded symbol extraction
	return "", nil
}