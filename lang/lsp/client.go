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

	// --- restart resilience: clangd can segfault (e.g. in typeParents on
	// pathological template typeHierarchy), which closes the jsonrpc2 conn
	// and would otherwise make every later request fail forever. connMu
	// guards swapping the live Conn/lspHandler/gen during a respawn;
	// in-flight callers keep using their captured (now-old) conn pointer,
	// which stays valid (only the field write is synchronized). ---
	connMu      sync.RWMutex
	gen         uint64
	repoURI     DocumentURI
	autoRestart bool // respawn the server on connection loss (C++ only)
	restartMu   sync.Mutex
	restarts    int
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

	// restart resilience: remember how to respawn, enable for C++ (the only
	// language whose server is known to crash mid-parse). gen starts at 1.
	cli.repoURI = NewURI(repo)
	cli.autoRestart = opts.Language == uniast.Cpp
	cli.gen = 1

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
	c.connMu.RLock()
	conn := c.Conn
	h := c.lspHandler
	c.connMu.RUnlock()
	if h != nil {
		h.Close()
	}
	if conn != nil {
		return conn.Close()
	}
	return nil
}

// curConn returns the live connection and its generation under a read lock.
// Callers capture the pointer and use it without holding the lock; a
// concurrent restart that swaps c.Conn is safe because the old conn object
// stays valid (GC-reachable) for the in-flight call.
func (cli *LSPClient) curConn() (*jsonrpc2.Conn, uint64) {
	cli.connMu.RLock()
	defer cli.connMu.RUnlock()
	return cli.Conn, cli.gen
}

// Notify shadows the embedded jsonrpc2.Conn.Notify so notifications go
// through the restart-aware connection accessor (and trigger a respawn if
// the transport has died).
func (cli *LSPClient) Notify(ctx context.Context, method string, params any, opts ...jsonrpc2.CallOption) error {
	conn, gen := cli.curConn()
	err := conn.Notify(ctx, method, params, opts...)
	if err != nil && IsConnClosed(err) {
		cli.maybeRestart(gen)
	}
	return err
}

// IsConnClosed reports whether err means the LSP transport died (server
// crashed / pipe closed) — unrecoverable on the same connection.
func IsConnClosed(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, jsonrpc2.ErrClosed) {
		return true
	}
	s := err.Error()
	return strings.Contains(s, "connection is closed") ||
		strings.Contains(s, "broken pipe") ||
		strings.Contains(s, "use of closed network connection")
}

// maybeRestart respawns the LSP server iff it hasn't already been restarted
// since the caller captured observedGen. Safe to call from many goroutines
// that all observed the same dead connection — only the first wins; the rest
// see a bumped generation and return. After a restart, every cached document
// is marked not-opened so the next DidOpen re-notifies the fresh server.
func (cli *LSPClient) maybeRestart(observedGen uint64) {
	if !cli.autoRestart {
		return
	}
	cli.restartMu.Lock()
	defer cli.restartMu.Unlock()
	cli.connMu.RLock()
	cur := cli.gen
	cli.connMu.RUnlock()
	if cur != observedGen {
		return // already restarted by another goroutine
	}
	log.Error("LSP server connection lost; restarting (restart #%d)...", cli.restarts+1)
	svr, err := startLSPSever(cli.Server, cli.ClientOptions)
	if err != nil {
		log.Error("LSP restart: failed to start server: %v", err)
		return
	}
	newcli, err := initLSPClient(context.Background(), svr, cli.repoURI, cli.Verbose, cli.Language, cli.InitializationOptions)
	if err != nil {
		log.Error("LSP restart: failed to init server: %v", err)
		return
	}
	cli.connMu.Lock()
	oldConn := cli.Conn
	oldH := cli.lspHandler
	cli.Conn = newcli.Conn
	cli.lspHandler = newcli.lspHandler
	if len(newcli.tokenTypes) > 0 {
		cli.tokenTypes = newcli.tokenTypes
	}
	if len(newcli.tokenModifiers) > 0 {
		cli.tokenModifiers = newcli.tokenModifiers
	}
	cli.gen++
	newGen := cli.gen
	cli.connMu.Unlock()
	cli.restarts++
	cli.resetServerOpened()
	if oldH != nil {
		oldH.Close()
	}
	if oldConn != nil {
		_ = oldConn.Close()
	}
	log.Error("LSP server restarted (gen=%d).", newGen)
}

// resetServerOpened marks every cached document as not-yet-opened on the
// (new) server so DidOpen re-sends textDocument/didOpen after a restart.
func (cli *LSPClient) resetServerOpened() {
	cli.filesMu.RLock()
	fs := make([]*TextDocumentItem, 0, len(cli.files))
	for _, f := range cli.files {
		fs = append(fs, f)
	}
	cli.filesMu.RUnlock()
	for _, f := range fs {
		if f == nil || f.Mu == nil {
			continue
		}
		f.Mu.Lock()
		f.ServerOpened = false
		f.Mu.Unlock()
	}
}

// Extra wrapper around jsonrpc2 that retries transient RPC failures.
// clangd (and other LSPs) occasionally reject otherwise-valid requests
// with -32602 "Invalid params" / "trying to get AST for non-added
// document" while a file is still being ready'd, or recover after a
// brief pause. We retry up to 3 attempts with a fixed 50ms gap and skip
// retry for terminal errors: MethodNotFound (-32601, server doesn't
// implement the endpoint) and context cancellation (caller bailed).
func (cli *LSPClient) Call(ctx context.Context, method string, params, result any, opts ...jsonrpc2.CallOption) error {
	conn, gen := cli.curConn()
	var raw json.RawMessage
	err := conn.Call(ctx, method, params, &raw)
	if err != nil && IsConnClosed(err) {
		// The server crashed (e.g. clangd segfault in typeParents on a
		// pathological template typeHierarchy). Retrying on the dead conn
		// is pointless; respawn it so subsequent symbols keep working, and
		// surface the error so THIS call is skipped (C++ base collection
		// falls back to collectCppBasesViaAST).
		cli.maybeRestart(gen)
		return err
	}
	if err != nil && shouldRetryRPC(err) {
		raw = nil
		err = retry.Do(
			func() error {
				raw = nil
				return conn.Call(ctx, method, params, &raw)
			},
			retry.Context(ctx),
			retry.Attempts(2), // initial call already happened; 2 more = 3 total
			retry.Delay(50*time.Millisecond),
			retry.DelayType(retry.FixedDelay),
			retry.LastErrorOnly(true),
			retry.RetryIf(shouldRetryRPC),
		)
		if err != nil && IsConnClosed(err) {
			cli.maybeRestart(gen)
			return err
		}
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

	// SemanticTokensLegend (optional). Newer LSP servers (e.g. gopls
	// since the LSP 3.17 client-capability gating became strict) won't
	// advertise `semanticTokensProvider` unless the client declares
	// matching capability — and many abcoder language paths (Go uses
	// the native parser, Java uses tree-sitter) never call
	// SemanticTokens. Treat absence as "this server doesn't support
	// semantic tokens"; the LSP call itself will surface a method-not-
	// found error if anything tries.
	if semanticTokensProvider, ok := vs["semanticTokensProvider"].(map[string]interface{}); ok && semanticTokensProvider != nil {
		if legend, ok := semanticTokensProvider["legend"].(map[string]interface{}); ok && legend != nil {
			if tokenTypes, ok := legend["tokenTypes"].([]interface{}); ok {
				for _, t := range tokenTypes {
					if s, ok := t.(string); ok {
						cli.tokenTypes = append(cli.tokenTypes, s)
					}
				}
			}
			if tokenModifiers, ok := legend["tokenModifiers"].([]interface{}); ok {
				for _, m := range tokenModifiers {
					if s, ok := m.(string); ok {
						cli.tokenModifiers = append(cli.tokenModifiers, s)
					}
				}
			}
		}
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
