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
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/cloudwego/abcoder/lang/log"
	lsp "github.com/cloudwego/abcoder/lang/lsp"
	"github.com/cloudwego/abcoder/lang/uniast"
)

type PythonSpec struct {
	repo          string
	topModuleName string
	topModulePath string
	sysPaths      []string
}

func (c *PythonSpec) ProtectedSymbolKinds() []lsp.SymbolKind {
	return []lsp.SymbolKind{}
}

func NewPythonSpec() *PythonSpec {
	cmd := exec.Command("python", "-c", "import sys ; print('\\n'.join(sys.path))")
	output, err := cmd.Output()
	if err != nil {
		log.Error("Failed to get sys.path: %v\n", err)
		return nil
	}
	sysPaths := strings.Split(string(output), "\n")
	// Match more specific paths first
	sort.Slice(sysPaths, func(i, j int) bool {
		return len(sysPaths[i]) > len(sysPaths[j])
	})
	log.Info("PythonSpec: using sysPaths %+v\n", sysPaths)
	return &PythonSpec{sysPaths: sysPaths}
}

func (c *PythonSpec) WorkSpace(root string) (map[string]string, error) {
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
		}
		return nil
	}
	if err := filepath.Walk(root, scanner); err != nil {
		return nil, err
	}

	c.topModulePath = absPath
	// TODO: find a way to infer the module (project) name.
	c.topModuleName = "current"
	rets[c.topModuleName] = c.topModulePath
	return rets, nil
}

// returns: modName, pkgPath, error
func (c *PythonSpec) NameSpace(path string, file *uniast.File) (string, string, error) {
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

	for _, sysPath := range c.sysPaths {
		if strings.HasPrefix(path, sysPath) {
			relPath, err := filepath.Rel(sysPath, path)
			if err != nil {
				return "", "", err
			}
			relPath = strings.TrimSuffix(relPath, ".py")
			pkgPath := strings.ReplaceAll(relPath, string(os.PathSeparator), ".")
			modPath := strings.Split(pkgPath, ".")
			if len(modPath) >= 1 {
				modName := modPath[0]
				return modName, pkgPath, nil
			}
			panic(fmt.Sprintf("Malformed Namespace %s, pkgPath %s", path, pkgPath))
		}
	}
	log.Error("Namespace not found for path: %s\n", path)
	return "", "", fmt.Errorf("namespace not found for path: %s", path)
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
	return typ == "function" || typ == "variable" || typ == "property" || typ == "class" || typ == "type"
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
	// XXX: maybe just use __main__?
	return sym.Kind == lsp.SKFunction && sym.Name == "main"
}

func (c *PythonSpec) IsEntitySymbol(sym lsp.DocumentSymbol) bool {
	typ := sym.Kind
	return typ == lsp.SKObject || typ == lsp.SKMethod || typ == lsp.SKFunction || typ == lsp.SKVariable ||
		typ == lsp.SKStruct || typ == lsp.SKEnum || typ == lsp.SKTypeParameter || typ == lsp.SKConstant || typ == lsp.SKClass
}

func (c *PythonSpec) IsPublicSymbol(sym lsp.DocumentSymbol) bool {
	// builtin methods are exported
	if strings.HasPrefix(sym.Name, "__") && strings.HasSuffix(sym.Name, "__") {
		return true
	}
	if strings.HasPrefix(sym.Name, "_") {
		return false
	}
	return true
}

func (c *PythonSpec) HasImplSymbol() bool {
	return true
}

func invalidPos() lsp.Position {
	return lsp.Position{
		Line:      -1,
		Character: -1,
	}
}

// returns interface, receiver, first method
func (c *PythonSpec) ImplSymbol(sym lsp.DocumentSymbol) (int, int, int) {
	// reference: https://docs.python.org/3/reference/grammar.html
	if sym.Kind != lsp.SKClass {
		return -1, -1, -1
	}

	implType := -1
	receiverType := -1
	firstMethod := -1

	// state 0: goto state -1 when we see a 'class'
	state := 0
	clsnamepos := invalidPos()
	curpos := sym.Location.Range.Start
	for i := range len(sym.Text) {
		if state == -1 {
			break
		}
		switch state {
		case 0:
			if i+6 >= len(sym.Text) {
				// class text does not contain a 'class'
				// should be an import
				return -1, -1, -1
			}
			next6chars := sym.Text[i : i+6]
			// heuristics should work with reasonable python code
			if next6chars == "class " {
				clsnamepos = curpos
				state = -1
			}
		}
		if sym.Text[i] == '\n' {
			curpos.Line++
			curpos.Character = 0
		} else {
			curpos.Character++
		}
	}

	for i, t := range sym.Tokens {
		if receiverType == -1 && clsnamepos.Less(t.Location.Range.Start) {
			receiverType = i
		}
	}

	return implType, receiverType, firstMethod
}

// returns: receiver, typeParams, inputParams, outputParams
func (c *PythonSpec) FunctionSymbol(sym lsp.DocumentSymbol) (int, []int, []int, []int) {
	// FunctionSymbol do not return receivers.
	// TODO type params in python (nobody uses them)
	// reference: https://docs.python.org/3/reference/grammar.html
	receiver := -1
	// python actually has these but TODO
	typeParams := []int{}

	// Hell, manually parse function text to get locations of key tokens since LSP does not support this...
	//
	// state 0: goto state 1 when we see a def
	// state 1: goto state 2 when we see a (
	// state 2: we're in the param list.
	//          collect input params by checking entity tokens.
	//          goto state 3 when we see a )
	// state 3: collect output params.
	// 			finish when we see a :
	state := 0
	paren_depth := 0
	// defpos := invalidPos()
	lparenpos := invalidPos()
	rparenpos := invalidPos()
	bodypos := invalidPos()
	curpos := sym.Location.Range.Start
	for i := range len(sym.Text) {
		if state == -1 {
			break
		}
		switch state {
		case 0:
			if i+4 >= len(sym.Text) {
				// function text does not contain a 'def'
				// should be an import
				return -1, []int{}, []int{}, []int{}
			}
			next4chars := sym.Text[i : i+4]
			// heuristics should work with reasonable python code
			if next4chars == "def " {
				// defpos = curpos
				state = 1
			}
		case 1:
			if sym.Text[i] == '(' {
				lparenpos = curpos
				paren_depth = 1
				state = 2
			}
		case 2:
			if sym.Text[i] == ')' {
				rparenpos = curpos
				paren_depth -= 1
				if paren_depth == 0 {
					state = 3
				}
			}
		case 3:
			if sym.Text[i] == ':' {
				bodypos = curpos
				state = -1
			}
		}
		if sym.Text[i] == '\n' {
			curpos.Line++
			curpos.Character = 0
		} else {
			curpos.Character++
		}
	}

	paramsrange := lsp.Range{
		Start: lparenpos,
		End:   rparenpos,
	}
	returnrange := lsp.Range{
		Start: rparenpos,
		End:   bodypos,
	}
	inputParams := []int{}
	outputParams := []int{}
	for i, t := range sym.Tokens {
		if paramsrange.Include(t.Location.Range) {
			if c.IsEntityToken(t) {
				inputParams = append(inputParams, i)
			}
		}
		if returnrange.Include(t.Location.Range) {
			if c.IsEntityToken(t) {
				outputParams = append(outputParams, i)
			}
		}
	}

	return receiver, typeParams, inputParams, outputParams
}

func (c *PythonSpec) GetUnloadedSymbol(from lsp.Token, define lsp.Location) (string, error) {
	panic("TODO")
}

func (c *PythonSpec) FileImports(content []byte) ([]uniast.Import, error) {
	// Reference:
	// https://docs.python.org/3/reference/grammar.html
	// There are two types of imports in Python:
	// import-as: on ONE line
	// 		import xxx as x, yyy as y
	// from-import: on ONE line
	// 		from ... import *
	// 		from ... import xxx as x, yyy as y
	//   or on POSSIBLY MULTIPLE lines, enclosed by parentheses
	// 		from ... import ( xxx, yyy as y ... )
	// And imports are simple stmts, so they MUST end with \n.
	patterns := []string{
		// Matches: import <anything> (on a single line)
		`(?m)^import\s+(.*)$`,
		// Matches: from <anything> import <anything> (on a single line, without parentheses)
		`(?m)^from\s+(.*?)\s+import\s+([^()\n]*)$`,
		// Matches: from <anything> import ( <anything> ) where <anything> can span multiple lines
		`(?m)^from\s+(.*?)\s+import\s+\(([\s\S]*?)\)$`,
	}

	res := []uniast.Import{}
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("error compiling regex pattern '%s': %w", p, err)
		}
		matches := re.FindAllStringSubmatch(string(content), -1) // -1 to find all non-overlapping matches
		for _, match := range matches {
			res = append(res, uniast.Import{
				Path: match[0],
			})
		}
	}
	return res, nil
}
