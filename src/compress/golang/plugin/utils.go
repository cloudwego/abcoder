/**
 * Copyright 2024 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

func shouldIgnoreDir(path string) bool {
	return strings.Contains(path, ".git")
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
