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
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/lang/uniast"
	lsp "github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
)

type LSPClient struct {
	*jsonrpc2.Conn
	*lspHandler
	tokenTypes     []string
	tokenModifiers []string
	files          map[DocumentURI]*TextDocumentItem
	// TODO: now only cache semantic tokens
	cachedResults map[string]SemanticTokens
	ClientOptions
}

type ClientOptions struct {
	Server string
	uniast.Language
	Verbose      bool
	CacheResults bool
}

func NewLSPClient(repo string, openfile string, wait time.Duration, opts ClientOptions) (*LSPClient, error) {
	// launch golang LSP server
	svr, err := startLSPSever(opts.Server)
	if err != nil {
		return nil, err
	}

	cli, err := initLSPClient(context.Background(), svr, NewURI(repo), opts.Verbose)
	if err != nil {
		return nil, err
	}

	cli.ClientOptions = opts
	cli.files = make(map[DocumentURI]*TextDocumentItem)
	if opts.CacheResults {
		cli.cachedResults = make(map[string]SemanticTokens)
	}

	if openfile != "" {
		_, err := cli.DidOpen(context.Background(), NewURI(openfile))
		if err != nil {
			return nil, err
		}

		// wait for "textDocument/publishDiagnostics" notification
		// 	resp := cli.WaitFirstNotify("textDocument/publishDiagnostics")
		// again:
		// 	var diagnostics lsp.PublishDiagnosticsParams
		// 	if err := json.Unmarshal(*resp.Params, &diagnostics); err != nil {
		// 		logger.Fatalf("Failed to unmarshal diagnostics: %v", err)
		// 	}
		// 	if len(diagnostics.Diagnostics) > 0 {
		// 		// wait again
		// 		resp = cli.WaitFirstNotify("textDocument/publishDiagnostics")
		// 		if retry > 0 {
		// 			retry--
		// 			goto again
		// 		}
		// 	}
	}

	time.Sleep(wait)

	return cli, nil
}

func (c *LSPClient) Close() error {
	c.lspHandler.Close()
	return c.Conn.Close()
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

func initLSPClient(ctx context.Context, svr io.ReadWriteCloser, dir DocumentURI, verbose bool) (*LSPClient, error) {
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
		"documentSymbol": map[string]interface{}{
			"hierarchicalDocumentSymbolSupport": true,
		},
	}

	initParams := initializeParams{
		ProcessID:    os.Getpid(),
		RootURI:      lsp.DocumentURI(dir),
		Capabilities: cs,
		Trace:        lsp.Trace(trace),
		ClientInfo:   lsp.ClientInfo{Name: "vscode"},
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

	// implementationProvider, ok := vs["implementationProvider"].(bool)
	// if !ok || !implementationProvider {
	// 	return nil, fmt.Errorf("server did not provide Implementation")
	// }
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
func startLSPSever(path string) (io.ReadWriteCloser, error) {
	// Launch rust-analyzer
	cmd := exec.Command(path)

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
