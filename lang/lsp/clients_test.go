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

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/cloudwego/abcoder/lang/uniast"
)

func checkSymNames(t *testing.T, symbols map[Range]*DocumentSymbol, expectedNames []string) {
	t.Helper()
	var symNames []string
	for _, sym := range symbols {
		symNames = append(symNames, sym.Name)
	}
	slices.Sort(symNames)
	slices.Sort(expectedNames)
	failMsg := ""
	if len(symNames) != len(expectedNames) {
		failMsg = fmt.Sprintf("Symbol count mismatch: expected %d, got %d", len(expectedNames), len(symNames))
	}
	for i := range symNames {
		if symNames[i] != expectedNames[i] {
			failMsg = fmt.Sprintf("Symbol name mismatch at index %d: expected %s, got %s", i, expectedNames[i], symNames[i])
			break
		}
	}
	if failMsg != "" {
		t.Fatal(failMsg)
		for i := range symNames {
			t.Logf("Symbol[%d]: %s", i, symNames[i])
		}
		for i := range expectedNames {
			t.Logf("Expected[%d]: %s", i, expectedNames[i])
		}
	}
}

func TestGolangLSP(t *testing.T) {
	golangLSP, goTestCase, err := InitLSPForFirstTest(uniast.Golang, "gopls")
	if err != nil {
		t.Fatalf("Failed to initialize Golang LSP client: %v", err)
	}
	defer golangLSP.Close()

	uri := NewURI(goTestCase + "/pkg/entity/entity.go")
	// documentSymbol
	expectedSymNames := `(MyStruct).String
(MyStructC).String
(MyStructD).DFunction
(MyStructD).String
A
G1
Integer
InterfaceB
MyStruct
MyStructC
MyStructD
V1`
	// references
	refRange := Range{ // MyStructC
		Start: Position{
			Line:      16,
			Character: 5,
		},
	}

	// documentSymbol
	t.Run("documentSymbol", func(t *testing.T) {
		symbols, err := golangLSP.DocumentSymbols(context.Background(), uri)
		if err != nil {
			t.Fatalf("Document Symbol failed: %v", err)
		}
		checkSymNames(t, symbols, strings.Split(expectedSymNames, "\n"))
		if _, err := json.Marshal(symbols); err != nil {
			t.Fatalf("Marshal Document Symbols failed: %v", err)
		}
	})

	// references
	t.Run("references", func(t *testing.T) {
		id := Location{
			URI:   uri,
			Range: refRange,
		}
		references, err := golangLSP.References(context.Background(), id)
		if err != nil {
			t.Fatalf("Reference failed: %v", err)
		}
		if len(references) != 3 {
			t.Fatalf("Expected 3 references, got %d", len(references))
		}
		if _, err := json.Marshal(references); err != nil {
			t.Fatalf("Marshal References failed: %v", err)
		}
	})
}

func TestRustLSP(t *testing.T) {
	rustLSP, rustTestCase, err := InitLSPForFirstTest(uniast.Rust, "rust-analyzer")
	if err != nil {
		t.Fatalf("Failed to initialize rust LSP client: %v", err)
	}
	defer rustLSP.Close()

	// documentSymbol
	entity_mod_uri := NewURI(rustTestCase + "/src/entity/mod.rs")
	expectedSymNames := `a
A
add
add
add
b
B
func
impl MyStruct
impl MyTrait for MyStruct
impl std::ops::Add<MyInt> for MyInt2
inter
MyEnum
MyInt
MY_INT
MyInt2
my_macro
MY_STATIC
MyStruct
my_trait
my_trait
MyTrait
new
Output`
	t.Run("documentSymbol", func(t *testing.T) {
		symbols, err := rustLSP.DocumentSymbols(context.Background(), entity_mod_uri)
		if err != nil {
			t.Fatalf("Document Symbol failed: %v", err)
		}
		checkSymNames(t, symbols, strings.Split(expectedSymNames, "\n"))
		if _, err := json.Marshal(symbols); err != nil {
			t.Fatalf("Marshal Document Symbols failed: %v", err)
		}
	})

	// references
	refRange := Range{
		Start: Position{
			Line:      48,
			Character: 6,
		},
	} // trait $0MyTrait {
	t.Run("references", func(t *testing.T) {
		id := Location{
			URI:   entity_mod_uri,
			Range: refRange,
		}
		references, err := rustLSP.References(context.Background(), id)
		if err != nil {
			t.Fatalf("Find Reference failed: %v", err)
		}
		if len(references) != 4 {
			t.Fatalf("Expected 4 references, got %d\n%+v\n", len(references), references)
		}
		if _, err := json.Marshal(references); err != nil {
			t.Fatalf("Marshal Reference failed: %v", err)
		}
	})

	// semanticTokens
	semtoksRange := Range{
		Start: Position{
			Line:      0,
			Character: 0,
		},
		End: Position{
			Line:      66,
			Character: 0,
		},
	}
	t.Run("semanticTokens", func(t *testing.T) {
		id := Location{
			URI:   entity_mod_uri,
			Range: semtoksRange,
		}
		tokens, err := rustLSP.SemanticTokens(context.Background(), id)
		if err != nil {
			t.Fatalf("Semantic Tokens failed: %v", err)
		}
		if len(tokens) != 149 {
			t.Fatalf("Expected 149 semantic tokens, got %d\n%+v", len(tokens), tokens)
		}
		if len(tokens) == 0 {
			t.Fatalf("Semantic Tokens should not be empty")
		}
		if _, err := json.Marshal(tokens); err != nil {
			t.Fatalf("Marshal Semantic Tokens failed: %v", err)
		}
	})

	// definition
	main_uri := NewURI(rustTestCase + "/src/main.rs")
	t.Run("definition", func(t *testing.T) {
		for _, pos := range []Position{
			{Line: 37, Character: 23},
			{Line: 20, Character: 4},
			{Line: 21, Character: 4},
			{Line: 27, Character: 16},
			{Line: 23, Character: 24},
			{Line: 24, Character: 11},
			{Line: 18, Character: 4},
			{Line: 17, Character: 20},
			{Line: 33, Character: 23},
			{Line: 38, Character: 18},
			{Line: 18, Character: 31},
			{Line: 19, Character: 35},
			{Line: 37, Character: 12},
			{Line: 20, Character: 27},
			{Line: 16, Character: 3},
		} {
			definition, err := rustLSP.Definition(context.Background(), main_uri, pos)
			if err != nil {
				t.Fatalf("Find Definition failed: %v", err)
			}
			if len(definition) != 1 {
				t.Fatalf("Find Definition should have found entry, but got none at %#v", pos)
			}
		}
	})

	// fileStructure
	t.Run("FileStructure", func(t *testing.T) {
		symbols, err := rustLSP.FileStructure(context.Background(), main_uri)
		if err != nil {
			t.Fatalf("File Structure failed: %v", err)
		}
		if _, err := json.Marshal(symbols); err != nil {
			t.Fatalf("Marshal File Structure failed: %v", err)
		}
	})
}
