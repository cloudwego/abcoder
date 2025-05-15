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

package python

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	lsp "github.com/cloudwego/abcoder/lang/lsp"
)

type PythonSpec struct {
	repo          string
	topModuleName string
	topModulePath string
}

func NewPythonSpec() *PythonSpec {
	return &PythonSpec{}
}

func (c *PythonSpec) WorkSpace(root string) (map[string]string, error) {
	// In python, pyspeak:modules are included by pyspeak:packages.
	// This is the opposite of ours.
	c.repo = root
	rets := map[string]string{}
	absPath, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	num_projfiles := 0
	scanner := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		base := filepath.Base(path)
		if base == "pyproject.toml" {
			num_projfiles++
			if num_projfiles > 1 {
				panic("multiple pyproject.toml files found")
			}
			// it's hard to infer the name or package from pyproject.toml
		}
		return nil
	}
	if err := filepath.Walk(root, scanner); err != nil {
		return nil, err
	}

	// XXX ad-hoc way
	if strings.Contains(c.repo, "astropy") {
		panic("TODO")
	} else {
		c.topModulePath = absPath
		c.topModuleName = "current"
		rets[c.topModuleName] = c.topModulePath
	}
	return rets, nil
}

// returns: modName, pkgPath, error
func (c *PythonSpec) NameSpace(path string) (string, string, error) {
	if strings.HasPrefix(path, c.topModulePath) {
		// internal module
		modName := c.topModuleName
		relPath, err := filepath.Rel(c.topModulePath, path)
		if err != nil {
			return "", "", err
		}
		// todo: handle __init__.py
		relPath = strings.TrimSuffix(relPath, ".py")
		pkgPath := strings.ReplaceAll(relPath, string(os.PathSeparator), ".")
		return modName, pkgPath, nil
	}

	if strings.HasSuffix(path, "stdlib/3/builtins.pyi") {
		// builtin module
		return "builtins", "builtins", nil
	}

	// XXX: hardcoded python version
	condaPrefix := "/home/zhenyang/anaconda3/envs/abcoder/lib/python3.11"
	if strings.HasPrefix(path, condaPrefix) {
		modName := "builtins"
		relPath, err := filepath.Rel(condaPrefix, path)
		if err != nil {
			return "", "", err
		}
		relPath = strings.TrimSuffix(relPath, ".py")
		pkgPath := strings.ReplaceAll(relPath, string(os.PathSeparator), ".")
		return modName, pkgPath, nil
	}

	panic(fmt.Sprintf("Namespace %s", path))
}

func (c *PythonSpec) ShouldSkip(path string) bool {
	if !strings.HasSuffix(path, ".py") {
		return true
	}
	return false
}

func (c *PythonSpec) IsDocToken(tok lsp.Token) bool {
	return tok.Type == "comment"
}

func (c *PythonSpec) DeclareTokenOfSymbol(sym lsp.DocumentSymbol) int {
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

func (c *PythonSpec) IsEntityToken(tok lsp.Token) bool {
	typ := tok.Type
	return typ == "function" || typ == "parameter" || typ == "variable" || typ == "property" || typ == "class" || typ == "type"
}

func (c *PythonSpec) IsStdToken(tok lsp.Token) bool {
	panic("TODO")
}

func (c *PythonSpec) TokenKind(tok lsp.Token) lsp.SymbolKind {
	switch tok.Type {
	case "namespace":
		return lsp.SKNamespace
	case "type":
		return lsp.SKObject // no direct match; mapped to Object conservatively
	case "class":
		return lsp.SKClass
	case "enum":
		return lsp.SKEnum
	case "interface":
		return lsp.SKInterface
	case "struct":
		return lsp.SKStruct
	case "typeParameter":
		return lsp.SKTypeParameter
	case "parameter":
		return lsp.SKVariable
	case "variable":
		return lsp.SKVariable
	case "property":
		return lsp.SKProperty
	case "enumMember":
		return lsp.SKEnumMember
	case "event":
		return lsp.SKEvent
	case "function":
		return lsp.SKFunction
	case "method":
		return lsp.SKMethod
	case "macro":
		return lsp.SKFunction
	case "string":
		return lsp.SKString
	case "number":
		return lsp.SKNumber
	case "operator":
		return lsp.SKOperator
	default:
		return lsp.SKUnknown
	}
}

func (c *PythonSpec) IsMainFunction(sym lsp.DocumentSymbol) bool {
	return sym.Kind == lsp.SKFunction && sym.Name == "main"
}

func (c *PythonSpec) IsEntitySymbol(sym lsp.DocumentSymbol) bool {
	typ := sym.Kind
	return typ == lsp.SKObject || typ == lsp.SKMethod || typ == lsp.SKFunction || typ == lsp.SKVariable ||
		typ == lsp.SKStruct || typ == lsp.SKEnum || typ == lsp.SKTypeParameter || typ == lsp.SKConstant || typ == lsp.SKClass
}

func (c *PythonSpec) IsPublicSymbol(sym lsp.DocumentSymbol) bool {
	if strings.HasPrefix(sym.Name, "_") {
		return false
	}
	return true
}

func (c *PythonSpec) HasImplSymbol() bool {
	// Python does not have direct impl symbols
	return false
}

func (c *PythonSpec) ImplSymbol(sym lsp.DocumentSymbol) (int, int, int) {
	panic("TODO")
}

// returns: receiver, typeParams, inputParams, outputParams
func (c *PythonSpec) FunctionSymbol(sym lsp.DocumentSymbol) (int, []int, []int, []int) {
	// no receiver. no type params in python
	// reference: https://docs.python.org/3/reference/grammar.html
	receiver := -1
	typeParams := []int{}

	// state 0: goto state 1 when we see a def
	// state 1: goto state 2 when we see a (
	// state 2: we're in the param list.
	//          collect input params by checking entity tokens.
	//          goto state 3 when we see a )
	// state 3: collect output params.
	// 			finish when we see a :
	state := 0
	paren_depth := 0
	inputParams := []int{}
	outputParams := []int{}
	for i, t := range sym.Tokens {
		if state == -1 {
			break
		}
		switch state {
		case 0:
			if t.Text == "def" {
				state = 1
			}
		case 1:
			if t.Text == "(" {
				state = 2
				paren_depth = 1
			}
		case 2:
			if t.Text == ")" {
				paren_depth -= 1
				if paren_depth == 0 {
					state = 3
				}
			} else if c.IsEntityToken(t) {
				inputParams = append(inputParams, i)
			}
		case 3:
			// no-op
			if t.Text == ":" {
				state = -1
			} else if c.IsEntityToken(t) {
				outputParams = append(outputParams, i)
			}
		}
	}

	return receiver, typeParams, inputParams, outputParams
}

func (c *PythonSpec) GetUnloadedSymbol(from lsp.Token, define lsp.Location) (string, error) {
	panic("TODO")
}
