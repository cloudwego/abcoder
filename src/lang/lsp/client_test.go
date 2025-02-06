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
	"os"
	"sync"
	"testing"
	"time"

	"github.com/cloudwego/abcoder/src/lang/log"
)

var golangLSP *LSPClient
var rustLSP *LSPClient
var rootDir = "/Users/bytedance/GOPATH/work/abcoder/testdata"

func TestMain(m *testing.M) {
	log.SetLogLevel(log.DebugLevel)
	var err error
	golangLSP, err = NewLSPClient(rootDir+"/golang", "", 0, ClientOptions{
		Server:   "gopls",
		Language: "go",
		Verbose:  true,
	})
	if err != nil {
		fmt.Printf("Failed to initialize golang LSP client: %v", err)
		os.Exit(1)
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		rustLSP, err = NewLSPClient("/root/codes/abcoder/tmp/lust-example-item", "/root/codes/abcoder/tmp/lust-example-item/src/lib.rs", time.Second*30, ClientOptions{
			Server:   "rust-analyzer",
			Language: "rust",
			Verbose:  true,
		})
		if err != nil {
			fmt.Printf("Failed to initialize rust: %v", err)
			os.Exit(1)
		}
	}()
	go func() {
		defer wg.Done()
		rustLSP, err = NewLSPClient("/root/codes/abcoder/testdata/rust2", "/root/codes/abcoder/testdata/rust2/Cargo.toml", time.Second*15, ClientOptions{
			Server:   "rust-analyzer",
			Language: "rust",
			Verbose:  true,
		})
		if err != nil {
			fmt.Printf("Failed to initialize rust LSP client: %v", err)
		}
	}()
	wg.Wait()

	c := m.Run()

	golangLSP.Close()
	rustLSP.Close()
	os.Exit(c)
}

func TestGolang(t *testing.T) {
	uri := NewURI(rootDir + "/golang/pkg/entity/entity.go")

	// documentSymbol
	t.Run("documentSymbol", func(t *testing.T) {
		symbols, err := golangLSP.DocumentSymbols(context.Background(), uri)
		if err != nil {
			t.Fatalf("Document Symbol failed: %v", err)
		}
		fmt.Printf("Document Symbol: %#v\n", symbols)
		// js, err := json.Marshal(symbols)
		// if err != nil {
		// 	t.Fatalf("Marshal Document Symbol failed: %v", err)
		// }
		// os.WriteFile("./symbol_golang.json", js, 0644)
	})

	// references
	t.Run("references", func(t *testing.T) {
		// reference to Function
		id := Location{
			URI: uri,
			Range: Range{
				Start: Position{
					Line:      8,
					Character: 8,
				},
			},
		}
		references, err := golangLSP.References(context.Background(), id)
		if err != nil {
			t.Fatalf("Find Reference failed: %v", err)
		}
		fmt.Printf("Find Reference: %#v\n", references)
	})

	// semanticTokens
	t.Run("semanticTokens", func(t *testing.T) {
		id := Location{
			URI: uri,
			Range: Range{
				Start: Position{
					Line:      0,
					Character: 0,
				},
				End: Position{
					Line:      40,
					Character: 0,
				},
			},
		}
		tokens, err := golangLSP.SemanticTokens(context.Background(), id)
		if err != nil {
			t.Fatalf("Semantic Tokens failed: %v", err)
		}
		fmt.Printf("Semantic Tokens: %#v\n", tokens)
		// js, err := json.Marshal(tokens)
		// if err != nil {
		// 	t.Fatalf("Marshal Semantic Tokens failed: %v", err)
		// }
		// os.WriteFile("./sytax-golang.json", js, 0644)
	})
}

func TestRust(t *testing.T) {
	// url encode
	uri := NewURI(rootDir + "/rust2/src/entity/mod.rs")

	// documentSymbol
	t.Run("documentSymbol", func(t *testing.T) {
		symbols, err := rustLSP.DocumentSymbols(context.Background(), uri)
		if err != nil {
			t.Fatalf("Document Symbol failed: %v", err)
		}
		t.Logf("Document Symbol: %#v", symbols)
		js, err := json.Marshal(symbols)
		if err != nil {
			t.Fatalf("Marshal Document Symbol failed: %v", err)
		}
		println(string(js))
	})

	// references
	t.Run("references", func(t *testing.T) {
		// reference to Function
		id := Location{
			URI: uri,
			Range: Range{
				Start: Position{
					Line:      13,
					Character: 13,
				},
			},
		}
		references, err := rustLSP.References(context.Background(), id)
		if err != nil {
			t.Fatalf("Find Reference failed: %v", err)
		}
		t.Logf("Find Reference: %#v", references)
	})

	// semanticTokens
	t.Run("semanticTokens", func(t *testing.T) {
		id := Location{
			URI: uri,
			Range: Range{
				Start: Position{
					Line:      0,
					Character: 0,
				},
				End: Position{
					Line:      66,
					Character: 0,
				},
			},
		}
		tokens, err := rustLSP.SemanticTokens(context.Background(), id)
		if err != nil {
			t.Fatalf("Semantic Tokens failed: %v", err)
		}
		js, err := json.Marshal(tokens)
		if err != nil {
			t.Fatalf("Marshal Semantic Tokens failed: %v", err)
		}
		println(string(js))
	})

	// definition
	t.Run("definition", func(t *testing.T) {
		uri := NewURI("/root/codes/abcoder/testdata/rust2/src/main.rs")
		definition, err := rustLSP.Definition(context.Background(), uri, Position{9, 27})
		if err != nil {
			t.Fatalf("Find Definition failed: %v", err)
		}
		if len(definition) != 1 {
			t.Fatalf("Find Definition failed: %v", definition)
		}
		t.Logf("Find Definition: %#v", definition)
	})

	t.Run("workspaceSymbol", func(t *testing.T) {
		symbols, err := rustLSP.WorkspaceSymbols(context.Background(), "add")
		if err != nil {
			t.Fatalf("Workspace Symbol failed: %v", err)
		}
		t.Logf("Workspace Symbol: %#v", symbols)
	})
}

func TestSearchSymbol(t *testing.T) {
	syms, err := rustLSP.DocumentSymbols(context.Background(), NewURI("/root/codes/abcoder/tmp/lust-example-item/src/lib.rs"))
	if err != nil {
		t.Fatalf("Document Symbol failed: %v", err)
	}
	t.Logf("Document Symbol: %#v", syms)
	symbols, err := rustLSP.WorkspaceSymbols(context.Background(), "Request")
	if err != nil {
		t.Fatalf("Workspace Symbol failed: %v", err)
	}
	t.Logf("Workspace Symbol: %#v", symbols)
}

func TestFileStructure(t *testing.T) {
	symbols, err := rustLSP.FileStructure(context.Background(), NewURI("/root/codes/abcoder/tmp/lust-example-item/target/debug/build/lust-gen-e0683cdee43abe70/out/lust_gen.rs"))
	if err != nil {
		t.Fatalf("File Structure failed: %v", err)
	}
	t.Logf("File Structure: %#v", symbols)
}
