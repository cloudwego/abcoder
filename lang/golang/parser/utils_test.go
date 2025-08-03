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

package parser

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func getTypeForTest(t *testing.T, src, name string) types.Type {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err, "Failed to parse source code for %s", name)

	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("test", fset, []*ast.File{f}, nil)
	require.NoError(t, err, "Failed to type-check source code for %s", name)

	obj := pkg.Scope().Lookup(name)
	require.NotNil(t, obj, "Object '%s' not found in source", name)

	return obj.Type()
}

func objectsToNames(objs []types.Object) []string {
	names := make([]string, len(objs))
	for i, obj := range objs {
		if obj.Pkg() != nil {
			names[i] = obj.Pkg().Path() + "." + obj.Name()
		} else {
			names[i] = obj.Name()
		}
	}
	slices.Sort(names)
	return names
}

func Test_getNamedTypes(t *testing.T) {
	testCases := []struct {
		name              string
		source            string
		targetVar         string
		expectedNames     []string
		expectedIsPointer bool
		expectedIsNamed   bool
	}{
		{
			name: "Simple Named Type",
			source: `package main
type MyInt int`,
			targetVar:         "MyInt",
			expectedNames:     []string{"test.MyInt"},
			expectedIsPointer: false,
			expectedIsNamed:   true,
		},
		{
			name: "Pointer to Named Type",
			source: `package main
type MyInt int
var p *MyInt`,
			targetVar:         "p",
			expectedNames:     []string{"test.MyInt"},
			expectedIsPointer: true,
			expectedIsNamed:   true,
		},
		{
			name: "Slice of Named Type",
			source: `package main
type MyStruct struct{}; var s []*MyStruct`,
			targetVar:         "s",
			expectedNames:     []string{"test.MyStruct"},
			expectedIsPointer: false,
			expectedIsNamed:   false,
		},
		{
			name: "Array of Named Type",
			source: `package main
type MyInt int; var a [5]MyInt`,
			targetVar:         "a",
			expectedNames:     []string{"test.MyInt"},
			expectedIsPointer: false,
			expectedIsNamed:   false,
		},
		{
			name: "Map with Named Types",
			source: `package main
type KeyType int; type ValueType string; var m map[*KeyType]ValueType`,
			targetVar:         "m",
			expectedNames:     []string{"test.KeyType", "test.ValueType"},
			expectedIsPointer: false,
			expectedIsNamed:   false,
		},
		{
			name: "Struct with Named Fields",
			source: `package main
					type MyInt int
					type MyString string
					var s struct {
						Field1 MyInt
						Field2 *MyString
					}`,
			targetVar:         "s",
			expectedNames:     []string{"test.MyInt", "test.MyString"},
			expectedIsPointer: false,
			expectedIsNamed:   false,
		},
		{
			name: "Interface with Embedded and Explicit Methods",
			source: `package main
					import "io"
					type MyInterface interface{
						io.Reader
						MyMethod(arg io.Writer)
					}`,
			targetVar:         "MyInterface",
			expectedNames:     []string{"test.MyInterface"},
			expectedIsPointer: false,
			expectedIsNamed:   true,
		},
		{
			name:              "Function Signature",
			source:            `package main; import "bytes"; type MyInt int; var fn func(a MyInt) *bytes.Buffer`,
			targetVar:         "fn",
			expectedNames:     []string{"bytes.Buffer", "test.MyInt"},
			expectedIsPointer: false,
			expectedIsNamed:   false,
		},
		{
			name:              "Type Alias",
			source:            `package main; type MyInt int; type IntAlias = MyInt`,
			targetVar:         "IntAlias",
			expectedNames:     []string{"test.MyInt"},
			expectedIsPointer: false,
			expectedIsNamed:   true,
		},
		{
			name:              "Recursive Struct (Cycle)",
			source:            `package main; type Node struct{ Next *Node }`,
			targetVar:         "Node",
			expectedNames:     []string{"test.Node"},
			expectedIsPointer: false,
			expectedIsNamed:   true,
		},
		{
			name:              "No Named Types",
			source:            `package main; var i int`,
			targetVar:         "i",
			expectedNames:     []string{},
			expectedIsPointer: false,
			expectedIsNamed:   false,
		},
		{
			name: "Tuple from function return",
			source: `package main
import "net/http"
var f func() (*http.Request, error)`,
			targetVar:         "f",
			expectedNames:     []string{"error", "net/http.Request"}, // error is a builtin interface, not considered a named type object here
			expectedIsPointer: false,
			expectedIsNamed:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			typ := getTypeForTest(t, tc.source, tc.targetVar)
			visited := make(map[types.Type]bool)

			tys, isPointer, isNamed := getNamedTypes(typ, visited)

			actualNames := objectsToNames(tys)

			require.Equal(t, tc.expectedNames, actualNames, "Named types mismatch")
			require.Equal(t, tc.expectedIsPointer, isPointer, "isPointer mismatch")
			require.Equal(t, tc.expectedIsNamed, isNamed, "isNamed mismatch")
		})
	}
}
