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

package collect

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	. "github.com/cloudwego/abcoder/lang/lsp"
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/sourcegraph/jsonrpc2"
)

type testJSONRPCHandler func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request)

func (h testJSONRPCHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	h(ctx, conn, req)
}

func TestCppASTForSymbol_MissDoesNotFallbackToTranslationUnitRoot(t *testing.T) {
	cli := &LSPClient{ClientOptions: ClientOptions{Language: uniast.Cpp}}
	c := NewCollector(t.TempDir(), cli)
	uri := NewURI(filepath.Join(t.TempDir(), "miss.cpp"))

	// The cached AST only contains an unrelated definition with a body.
	// Looking up a symbol on another line should report "not found"
	// instead of falling back to the translation-unit root.
	c.cppFileASTCache[uri] = &ASTNode{
		Kind: "TranslationUnit",
		Children: []*ASTNode{{
			Role: "declaration",
			Kind: "Function",
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 16},
			},
			Children: []*ASTNode{{Kind: "Compound"}},
		}},
	}

	sym := &DocumentSymbol{Location: Location{
		URI: uri,
		Range: Range{
			Start: Position{Line: 1, Character: 0},
			End:   Position{Line: 1, Character: 11},
		},
	}}

	if got := c.cppASTForSymbol(context.Background(), sym); got != nil {
		t.Fatalf("cppASTForSymbol() = %#v, want nil when no declaration node matches the symbol range", got)
	}

	hasBody, ok := c.cppASTHasBody(context.Background(), sym)
	if ok || hasBody {
		t.Fatalf("cppASTHasBody() = (%v, %v), want (false, false) when symbol lookup misses", hasBody, ok)
	}
}

func TestCppFileAST_WholeFileRangeIncludesLastLineContent(t *testing.T) {
	const text = "void earlier() {}\nvoid last();"
	file := filepath.Join(t.TempDir(), "last_line_no_newline.cpp")
	if err := os.WriteFile(file, []byte(text), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	uri := NewURI(file)

	var (
		mu       sync.Mutex
		gotRange Range
	)
	serverHandler := testJSONRPCHandler(func(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
		if req.Notif {
			return
		}
		switch req.Method {
		case "textDocument/ast":
			var params struct {
				TextDocument TextDocumentIdentifier `json:"textDocument"`
				Range        Range                  `json:"range"`
			}
			if req.Params != nil {
				if err := json.Unmarshal(*req.Params, &params); err != nil {
					t.Errorf("unmarshal textDocument/ast params: %v", err)
				}
			}
			mu.Lock()
			gotRange = params.Range
			mu.Unlock()
			if err := conn.Reply(ctx, req.ID, &ASTNode{Kind: "TranslationUnit"}); err != nil {
				t.Errorf("reply textDocument/ast: %v", err)
			}
		default:
			if err := conn.Reply(ctx, req.ID, nil); err != nil {
				t.Errorf("reply %s: %v", req.Method, err)
			}
		}
	})

	clientRW, serverRW := net.Pipe()
	clientConn := jsonrpc2.NewConn(context.Background(), jsonrpc2.NewBufferedStream(clientRW, jsonrpc2.VSCodeObjectCodec{}), testJSONRPCHandler(func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request) {}))
	serverConn := jsonrpc2.NewConn(context.Background(), jsonrpc2.NewBufferedStream(serverRW, jsonrpc2.VSCodeObjectCodec{}), serverHandler)
	t.Cleanup(func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
	})

	cli := &LSPClient{
		Conn:          clientConn,
		ClientOptions: ClientOptions{Language: uniast.Cpp},
	}
	cli.InitFiles()
	if _, err := cli.DidOpen(context.Background(), uri); err != nil {
		t.Fatalf("DidOpen() failed: %v", err)
	}

	c := NewCollector(t.TempDir(), cli)

	if got := c.cppFileAST(context.Background(), uri); got == nil {
		t.Fatal("cppFileAST() returned nil")
	}

	mu.Lock()
	defer mu.Unlock()
	if gotRange.End.Line != 1 {
		t.Fatalf("textDocument/ast end line = %d, want 1", gotRange.End.Line)
	}
	wantLastLineEnd := len("void last();")
	if gotRange.End.Character != wantLastLineEnd {
		t.Fatalf("textDocument/ast end character = %d, want %d to include the last line without trailing newline", gotRange.End.Character, wantLastLineEnd)
	}
}
