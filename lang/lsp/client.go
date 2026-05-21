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
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	retry "github.com/avast/retry-go/v4"
	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/lang/uniast"
	lsp "github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
	"golang.org/x/sync/singleflight"
)

type LSPClient struct {
	*jsonrpc2.Conn
	*lspHandler
	tokenTypes     []string
	tokenModifiers []string
	files          map[DocumentURI]*TextDocumentItem
	// filesMu guards files. Lock briefly when checking/inserting an entry;
	// the per-file Mu inside TextDocumentItem guards per-document caches.
	filesMu  sync.RWMutex
	provider LanguageServiceProvider

	// In-flight request dedup. When N workers simultaneously ask for
	// DocumentSymbols / SemanticTokens / Definition of the same key, only
	// one RPC is sent; the rest wait on the first one's result. After the
	// result lands it goes into the per-file cache so future calls are
	// instant.
	docSymFlight    singleflight.Group // key: URI
	semTokFlight    singleflight.Group // key: URI (full-doc semantic tokens)
	definitionFlight singleflight.Group // key: URI + ":" + line + ":" + col

	ClientOptions
	LspOptions map[string]string
}

type ClientOptions struct {
	Server string
	uniast.Language
	Verbose               bool
	InitializationOptions interface{}
}

func NewLSPClient(repo string, openfile string, wait time.Duration, opts ClientOptions) (*LSPClient, error) {
	// launch golang LSP server
	svr, err := startLSPSever(opts.Server, opts)
	if err != nil {
		return nil, err
	}

	cli, err := initLSPClient(context.Background(), svr, NewURI(repo), opts.Verbose, opts.Language, opts.InitializationOptions)
	if err != nil {
		return nil, err
	}

	cli.ClientOptions = opts
	cli.files = make(map[DocumentURI]*TextDocumentItem)

	cli.provider = GetProvider(opts.Language)
	cli.Verbose = opts.Verbose

	if openfile != "" {
		_, err := cli.DidOpen(context.Background(), NewURI(openfile))
		if err != nil {
			return nil, err
		}
	}

	time.Sleep(wait)

	return cli, nil
}

func (c *LSPClient) Close() error {
	c.lspHandler.Close()
	return c.Conn.Close()
}

// Extra wrapper around jsonrpc2 that retries transient RPC failures.
// clangd (and other LSPs) occasionally reject otherwise-valid requests
// with -32602 "Invalid params" / "trying to get AST for non-added
// document" while a file is still being ready'd, or recover after a
// brief pause. We retry up to 3 attempts with a fixed 50ms gap and skip
// retry for terminal errors: MethodNotFound (-32601, server doesn't
// implement the endpoint) and context cancellation (caller bailed).
func (cli *LSPClient) Call(ctx context.Context, method string, params, result any, opts ...jsonrpc2.CallOption) error {
	var raw json.RawMessage
	err := cli.Conn.Call(ctx, method, params, &raw)
	if err != nil && shouldRetryRPC(err) {
		raw = nil
		err = retry.Do(
			func() error {
				raw = nil
				return cli.Conn.Call(ctx, method, params, &raw)
			},
			retry.Context(ctx),
			retry.Attempts(2), // initial call already happened; 2 more = 3 total
			retry.Delay(50*time.Millisecond),
			retry.DelayType(retry.FixedDelay),
			retry.LastErrorOnly(true),
			retry.RetryIf(shouldRetryRPC),
		)
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, result)
}

// shouldRetryRPC reports whether err is worth retrying. Terminal cases:
// MethodNotFound (server doesn't implement) and ctx cancel/deadline.
func shouldRetryRPC(err error) bool {
	if err == nil {
		return false
	}
	if IsJSONRPCMethodNotFound(err) {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

// IsJSONRPCMethodNotFound reports whether err is a jsonrpc2 -32601
// (server doesn't implement the method) — a terminal classification
// signalling no point in retrying.
func IsJSONRPCMethodNotFound(err error) bool {
	if err == nil {
		return false
	}
	var jrpcErr *jsonrpc2.Error
	if errors.As(err, &jrpcErr) {
		return jrpcErr.Code == -32601
	}
	return false
}

type initializeParams struct {
	ProcessID int `json:"processId,omitempty"`

	// RootPath is DEPRECATED in favor of the RootURI field.
	RootPath string `json:"rootPath,omitempty"`

	RootURI               lsp.DocumentURI `json:"rootUri,omitempty"`
	ClientInfo            lsp.ClientInfo  `json:"clientInfo,omitempty"`
	Trace                 lsp.Trace       `json:"trace,omitempty"`
	InitializationOptions interface{}     `json:"initializationOptions,omitempty"`
	Capabilities          interface{}     `json:"capabilities"`

	WorkDoneToken string `json:"workDoneToken,omitempty"`
}

type initializeResult struct {
	Capabilities interface{} `json:"capabilities,omitempty"`
}

func (c *LSPClient) InitFiles() {
	if c.files == nil {
		c.files = make(map[DocumentURI]*TextDocumentItem)
	}
}

func initLSPClient(ctx context.Context, svr io.ReadWriteCloser, dir DocumentURI, verbose bool, language uniast.Language, InitializationOptions interface{}) (*LSPClient, error) {
	h := newLSPHandler()
	stream := jsonrpc2.NewBufferedStream(svr, jsonrpc2.VSCodeObjectCodec{})
	conn := jsonrpc2.NewConn(ctx, stream, h)
	cli := &LSPClient{Conn: conn, lspHandler: h}

	// Initialize the LSP server
	trace := "off"
	if verbose {
		trace = "verbose"
	}

	// NOTICE: some features need to be enabled explicitly
	cs := map[string]interface{}{
		"workspace": map[string]interface{}{
			"symbol": map[string]interface{}{
				"dynamicRegistration": true,
			},
		},
		"textDocument": map[string]interface{}{
			"documentSymbol": map[string]interface{}{
				// Java uses tree-sitter instead of hierarchical symbols
				// Golang stays the same as older versions. ABCoder do not use gopls, so don't play with it.
				"hierarchicalDocumentSymbolSupport": (language != uniast.Java && language != uniast.Golang),
			},
		},
	}

	initParams := initializeParams{
		ProcessID:             os.Getpid(),
		RootURI:               lsp.DocumentURI(dir),
		Capabilities:          cs,
		Trace:                 lsp.Trace(trace),
		ClientInfo:            lsp.ClientInfo{Name: "vscode"},
		InitializationOptions: InitializationOptions,
	}

	var initResult initializeResult
	if err := conn.Call(ctx, "initialize", initParams, &initResult); err != nil {
		return nil, err
	}

	vs, ok := initResult.Capabilities.(map[string]interface{})
	if !ok || vs == nil {
		return nil, fmt.Errorf("invalid server capabilities: %v", initResult.Capabilities)
	}
	// check server's capabilities
	definitionProvider, ok := vs["definitionProvider"].(bool)
	if !ok || !definitionProvider {
		return nil, fmt.Errorf("server did not provide Definition")
	}
	typeDefinitionProvider, ok := vs["typeDefinitionProvider"].(bool)
	if !ok || !typeDefinitionProvider {
		return nil, fmt.Errorf("server did not provide TypeDefinition")
	}

	documentSymbolProvider, ok := vs["documentSymbolProvider"].(bool)
	if !ok || !documentSymbolProvider {
		return nil, fmt.Errorf("server did not provide DocumentSymbol")
	}
	referencesProvider, ok := vs["referencesProvider"].(bool)
	if !ok || !referencesProvider {
		return nil, fmt.Errorf("server did not provide References")
	}

	// SemanticTokensLegend
	semanticTokensProvider, ok := vs["semanticTokensProvider"].(map[string]interface{})
	if !ok || semanticTokensProvider == nil {
		return nil, fmt.Errorf("server did not provide SemanticTokensProvider")
	}
	legend, ok := semanticTokensProvider["legend"].(map[string]interface{})
	if !ok || legend == nil {
		return nil, fmt.Errorf("server did not provide SemanticTokensProvider.legend")
	}
	tokenTypes, ok := legend["tokenTypes"].([]interface{})
	if !ok || tokenTypes == nil {
		return nil, fmt.Errorf("server did not provide SemanticTokensProvider.legend.tokenTypes")
	}
	tokenModifiers, ok := legend["tokenModifiers"].([]interface{})
	if !ok || tokenModifiers == nil {
		return nil, fmt.Errorf("server did not provide SemanticTokensProvider.legend.tokenModifiers")
	}
	// store to global
	for _, t := range tokenTypes {
		cli.tokenTypes = append(cli.tokenTypes, t.(string))
	}
	for _, m := range tokenModifiers {
		cli.tokenModifiers = append(cli.tokenModifiers, m.(string))
	}

	// notify the server that we have initialized
	if err := conn.Notify(ctx, "initialized", lsp.InitializeParams{}); err != nil {
		return nil, err
	}
	return cli, nil
}

type rwc struct {
	io.ReadCloser
	io.WriteCloser
	cmd *exec.Cmd
}

func (rwc rwc) Close() error {
	if err := rwc.WriteCloser.Close(); err != nil {
		return err
	}
	if rc, ok := rwc.ReadCloser.(io.Closer); ok {
		return rc.Close()
	}
	return rwc.cmd.Wait()
}

// start a LSP process and return its io
func startLSPSever(path string, opts ClientOptions) (io.ReadWriteCloser, error) {

	var cmd *exec.Cmd
	if uniast.Java == opts.Language || uniast.Cpp == opts.Language {
		parts := strings.Fields(path)
		cmd = exec.Command(parts[0], parts[1:]...)
	} else {
		// Launch rust-analyzer
		cmd = exec.Command(path)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("Failed to get stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("Failed to get stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("Failed to get stderr pipe: %v", err)
	}
	// Read stderr in a separate goroutine
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Error("LSP server stderr: %s\n", scanner.Text())
			// os.Exit(2)
		}
	}()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("Failed to start LSP server: %v", err)
	}

	return rwc{stdout, stdin, cmd}, nil
}
