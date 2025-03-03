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

package parse

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/Knetic/govaluate"
	. "github.com/cloudwego/abcoder/src/uniast"
	"golang.org/x/mod/modfile"
)

func shouldIgnoreDir(path string) bool {
	return strings.Contains(path, ".git") || strings.Contains(path, "vendor/") || strings.Contains(path, "kitex_gen") || strings.Contains(path, "hertz_gen")
}

func shouldIgnoreFile(path string) bool {
	return !strings.Contains(path, ".go") || strings.Contains(path, "_test.go")
}

type cache map[interface{}]bool

func (c cache) Visited(val interface{}) bool {
	ok := c[val]
	if !ok {
		c[val] = true
	}
	return ok
}

func hasMain(file []byte) bool {
	if !bytes.Contains(file, []byte("package main")) || !bytes.Contains(file, []byte("func main()")) {
		return false
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "any.go", file, parser.SkipObjectResolution)
	if err != nil {
		return false
	}
	if f.Name.Name != "main" {
		return false
	}
	for _, decl := range f.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Name.Name == "main" {
				return true
			}
		}
	}
	return false
}

func isSysPkg(importPath string) bool {
	return !strings.Contains(strings.Split(importPath, "/")[0], ".")
}

var (
	verReg = regexp.MustCompile(`/v\d+$`)
	litReg = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

func getPackageAlias(importPath string) string {
	// Remove the version suffix if present (e.g., "/v2" or "/v10")

	basePath := verReg.ReplaceAllString(importPath, "")

	// Get the base name of the package
	alias := path.Base(basePath)

	// Replace any non-valid identifier characters with underscores
	if ps := strings.Split(alias, "-"); len(ps) > 1 {
		alias = ps[1]
	}

	return alias
}

func splitVersion(module string) (string, string) {
	if strings.Contains(module, "@") {
		parts := strings.Split(module, "@")
		return parts[0], parts[1]
	}
	return module, ""
}

func getModuleName(modFilePath string) (string, []byte, error) {
	file, err := os.Open(modFilePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read file: %v", err)
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "module") {
			// Assuming 'module' keyword is followed by module name
			parts := strings.Split(line, " ")
			if len(parts) > 1 {
				return parts[1], data, nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", data, fmt.Errorf("failed to scan file: %v", err)
	}

	return "", data, nil
}

// parse go.mod and get a map of module name to module_path@version
func parseModuleFile(data []byte) (map[string]string, error) {
	ast, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod file: %v", err)
	}
	modules := make(map[string]string)
	for _, req := range ast.Require {
		if req.Indirect {
			continue
		}
		modules[req.Mod.Path] = req.Mod.Path + "@" + req.Mod.Version
	}
	// replaces
	for _, replace := range ast.Replace {
		modules[replace.Old.Path] = replace.New.Path + "@" + replace.New.Version
	}
	return modules, nil
}

func isGoBuiltins(name string) bool {
	switch name {
	case "append", "cap", "close", "complex", "copy", "delete", "imag", "len", "make", "new", "panic", "print", "println", "real", "recover":
		return true
	case "string", "bool", "byte", "complex64", "complex128", "error", "float32", "float64", "int", "int8", "int16", "int32", "int64", "rune", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return true
	case "interface{}", "any":
		return true
	default:
		return false
	}
}

func isPkgScope(scope *types.Scope) bool {
	return scope != nil && scope.Parent() == types.Universe
}

func getTypeKind(n ast.Expr) TypeKind {
	switch n.(type) {
	case *ast.StructType:
		return TypeKindStruct
	case *ast.InterfaceType:
		return TypeKindInterface
	default:
		return TypeKindNamed
	}
}

func getNamedType(typ types.Type) (ty types.Object, isPointer bool) {
	if pt, ok := typ.(*types.Pointer); ok {
		typ = pt.Elem()
	} else if name, ok := typ.(*types.Named); ok {
		return name.Obj(), isPointer
	}
	return nil, isPointer
}

func extractName(typ string) string {
	if strings.Contains(typ, ".") {
		return strings.Split(typ, ".")[1]
	}
	return typ
}

func parseExpr(expr string) (interface{}, error) {
	// Create a map of parameters to pass to the expression evaluator.
	parameters := map[string]interface{}{
		"iota": 0,
	}

	// Create the expression evaluator.
	eval, err := govaluate.NewEvaluableExpression(expr)
	if err != nil {
		return nil, err
	}

	result, err := eval.Evaluate(parameters)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func newIdentity(mod, pkg, name string) Identity {
	return Identity{ModPath: mod, PkgPath: pkg, Name: name}
}

func isUpperCase(c byte) bool {
	return c >= 'A' && c <= 'Z'
}
