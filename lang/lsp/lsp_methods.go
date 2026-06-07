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
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"sync"

	"github.com/cloudwego/abcoder/lang/utils"
	lsp "github.com/sourcegraph/go-lsp"
)

type DocumentRange struct {
	TextDocument lsp.TextDocumentIdentifier `json:"textDocument"`
	Range        Range                      `json:"range"`
}

type SemanticTokensFullParams struct {
	TextDocument lsp.TextDocumentIdentifier `json:"textDocument"`
}

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

func (cli *LSPClient) DidOpen(ctx context.Context, file DocumentURI) (*TextDocumentItem, error) {
	cli.filesMu.RLock()
	f, ok := cli.files[file]
	cli.filesMu.RUnlock()
	if ok {
		// An entry exists locally — but ensureLocalFile() may have created
		// it without notifying clangd. Send didOpen now if needed, so the
		// server's side gets the file content and subsequent AST queries
		// don't fail with "trying to get AST for non-added document".
		f.Mu.Lock()
		if f.ServerOpened {
			f.Mu.Unlock()
			return f, nil
		}
		f.ServerOpened = true
		params := DidOpenTextDocumentParams{TextDocument: *f}
		f.Mu.Unlock()
		if err := cli.Notify(ctx, "textDocument/didOpen", params); err != nil {
			// roll back so a later DidOpen can retry
			f.Mu.Lock()
			f.ServerOpened = false
			f.Mu.Unlock()
			return nil, err
		}
		return f, nil
	}
	text, err := os.ReadFile(file.File())
	if err != nil {
		return nil, err
	}
	nf := &TextDocumentItem{
		URI:          DocumentURI(file),
		LanguageID:   cli.Language.String(),
		Version:      1,
		Text:         string(text),
		LineCounts:   utils.CountLines(string(text)),
		Mu:           &sync.Mutex{},
		ServerOpened: true, // we're about to send didOpen below
	}
	cli.filesMu.Lock()
	if _, ok := cli.files[file]; ok {
		// lost the race; reuse the existing entry (recurse so it notifies
		// the server if the winner created a local-only stub).
		cli.filesMu.Unlock()
		return cli.DidOpen(ctx, file)
	}
	cli.files[file] = nf
	cli.filesMu.Unlock()
	req := DidOpenTextDocumentParams{
		TextDocument: *nf,
	}
	if err := cli.Notify(ctx, "textDocument/didOpen", req); err != nil {
		// roll back: server doesn't know about the file, future callers
		// must re-attempt the notification.
		nf.Mu.Lock()
		nf.ServerOpened = false
		nf.Mu.Unlock()
		return nil, err
	}
	return nf, nil
}

func flattenDocumentSymbols(symbols []*DocumentSymbol, uri DocumentURI) []*DocumentSymbol {
	var result []*DocumentSymbol
	for _, sym := range symbols {
		var location Location
		if sym.Range != nil {
			location = Location{
				URI:   uri,
				Range: *sym.Range,
			}
		} else {
			location = sym.Location
		}
		// Preserve SelectionRange — it points at the symbol's NAME token
		// (vs Range which spans the whole body). Some downstream paths
		// (e.g. typeHierarchy/prepareTypeHierarchy) need a position on
		// the identifier specifically.
		flatSymbol := DocumentSymbol{
			// copy
			Name:           sym.Name,
			Detail:         sym.Detail,
			Kind:           sym.Kind,
			Tags:           sym.Tags,
			Text:           sym.Text,
			Tokens:         sym.Tokens,
			Node:           sym.Node,
			Children:       sym.Children,
			SelectionRange: sym.SelectionRange,
			// new
			Location: location,
			// empty
			Role:  0,
			Range: nil,
		}
		result = append(result, &flatSymbol)

		if len(sym.Children) > 0 {
			childSymbols := flattenDocumentSymbols(sym.Children, uri)
			result = append(result, childSymbols...)
		}
	}
	return result
}

func (cli *LSPClient) DocumentSymbols(ctx context.Context, file DocumentURI) (map[Range]*DocumentSymbol, error) {
	// open file first
	f, err := cli.DidOpen(ctx, file)
	if err != nil {
		return nil, err
	}
	f.Mu.Lock()
	if f.Symbols != nil {
		syms := f.Symbols
		f.Mu.Unlock()
		return syms, nil
	}
	f.Mu.Unlock()

	// Deduplicate concurrent requests for the same URI. Without this, 32
	// workers all hitting a freshly-encountered external header would each
	// fire their own documentSymbol RPC; clangd serializes per-document
	// parses so they'd queue up anyway.
	v, err, _ := cli.docSymFlight.Do(string(file), func() (interface{}, error) {
		// Re-check after acquiring the flight: another flight just finished.
		f.Mu.Lock()
		if f.Symbols != nil {
			cached := f.Symbols
			f.Mu.Unlock()
			return cached, nil
		}
		f.Mu.Unlock()

		uri := lsp.DocumentURI(file)
		req := lsp.DocumentSymbolParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: uri},
		}
		var resp []*DocumentSymbol
		if err := cli.Call(ctx, "textDocument/documentSymbol", req, &resp); err != nil {
			return nil, err
		}
		respFlatten := flattenDocumentSymbols(resp, file)
		built := make(map[Range]*DocumentSymbol, len(respFlatten))
		for i := range respFlatten {
			s := respFlatten[i]
			built[s.Location.Range] = s
		}
		f.Mu.Lock()
		if f.Symbols == nil {
			f.Symbols = built
		}
		out := f.Symbols
		f.Mu.Unlock()
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(map[Range]*DocumentSymbol), nil
}

func (cli *LSPClient) References(ctx context.Context, id Location) ([]Location, error) {
	if _, err := cli.DidOpen(ctx, id.URI); err != nil {
		return nil, err
	}
	uri := lsp.DocumentURI(id.URI)
	req := lsp.ReferenceParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: uri,
			},
			Position: lsp.Position{
				Line:      id.Range.Start.Line,
				Character: id.Range.Start.Character + 1,
			},
		},
		Context: lsp.ReferenceContext{
			IncludeDeclaration: true,
		},
	}
	var resp []Location
	if err := cli.Call(ctx, "textDocument/references", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (cli *LSPClient) getSemanticTokensRange(ctx context.Context, req DocumentRange, resp *SemanticTokens) error {
	uri := DocumentURI(req.TextDocument.URI)
	f, err := cli.DidOpen(ctx, uri)
	if err != nil {
		return err
	}

	f.Mu.Lock()
	have := f.SemanticTokens
	f.Mu.Unlock()
	if have == nil {
		v, err, _ := cli.semTokFlight.Do(string(uri), func() (interface{}, error) {
			f.Mu.Lock()
			if f.SemanticTokens != nil {
				cached := f.SemanticTokens
				f.Mu.Unlock()
				return cached, nil
			}
			f.Mu.Unlock()

			req1 := SemanticTokensFullParams{TextDocument: req.TextDocument}
			var fullResp SemanticTokens
			if err := cli.Call(ctx, "textDocument/semanticTokens/full", req1, &fullResp); err != nil {
				return nil, err
			}
			f.Mu.Lock()
			if f.SemanticTokens == nil {
				f.SemanticTokens = &fullResp
			}
			out := f.SemanticTokens
			f.Mu.Unlock()
			return out, nil
		})
		if err != nil {
			return err
		}
		have = v.(*SemanticTokens)
	}

	resp.ResultID = have.ResultID
	resp.Data = make([]uint32, len(have.Data))
	copy(resp.Data, have.Data)

	filterSemanticTokensInRange(resp, req.Range)
	return nil
}

func filterSemanticTokensInRange(resp *SemanticTokens, r Range) {
	curPos := Position{
		Line:      0,
		Character: 0,
	}
	newData := []uint32{}
	includedIs := []int{}
	for i := 0; i < len(resp.Data); i += 5 {
		deltaLine := int(resp.Data[i])
		deltaStart := int(resp.Data[i+1])
		if deltaLine != 0 {
			curPos.Line += deltaLine
			curPos.Character = deltaStart
		} else {
			curPos.Character += deltaStart
		}
		if isPositionInRange(curPos, r, true) {
			if len(newData) == 0 {
				// add range start to initial delta
				newData = append(newData, resp.Data[i:i+5]...)
				newData[0] = uint32(curPos.Line)
				newData[1] = uint32(curPos.Character)
			} else {
				newData = append(newData, resp.Data[i:i+5]...)
			}
			includedIs = append(includedIs, i)
		}
	}
	resp.Data = newData
}

func (cli *LSPClient) SemanticTokens(ctx context.Context, id Location) ([]Token, error) {
	// open file first
	syms, err := cli.DocumentSymbols(ctx, id.URI)
	if err != nil {
		return nil, err
	}
	sym := syms[id.Range]
	f := cli.lookupFile(id.URI)
	if sym != nil {
		if f != nil {
			f.Mu.Lock()
			toks := sym.Tokens
			f.Mu.Unlock()
			if toks != nil {
				return toks, nil
			}
		} else if sym.Tokens != nil {
			return sym.Tokens, nil
		}
	}

	uri := lsp.DocumentURI(id.URI)
	req := DocumentRange{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: uri,
		},
		Range: id.Range,
	}

	var resp SemanticTokens
	if err := cli.getSemanticTokensRange(ctx, req, &resp); err != nil {
		return nil, err
	}

	toks := cli.getAllTokens(resp, id.URI)
	if sym != nil {
		if f != nil {
			f.Mu.Lock()
			if sym.Tokens == nil {
				sym.Tokens = toks
			}
			toks = sym.Tokens
			f.Mu.Unlock()
		} else {
			sym.Tokens = toks
		}
	}
	return toks, nil
}

// lookupFile returns the cached TextDocumentItem if open, otherwise nil.
func (cli *LSPClient) lookupFile(uri DocumentURI) *TextDocumentItem {
	cli.filesMu.RLock()
	f := cli.files[uri]
	cli.filesMu.RUnlock()
	return f
}

func (cli *LSPClient) Definition(ctx context.Context, uri DocumentURI, pos Position) ([]Location, error) {
	// open file first
	f, err := cli.DidOpen(ctx, uri)
	if err != nil {
		return nil, err
	}
	f.Mu.Lock()
	if f.Definitions != nil {
		if locations, ok := f.Definitions[pos]; ok {
			f.Mu.Unlock()
			return locations, nil
		}
	}
	f.Mu.Unlock()

	key := string(uri) + ":" + strconv.Itoa(pos.Line) + ":" + strconv.Itoa(pos.Character)
	v, err, _ := cli.definitionFlight.Do(key, func() (interface{}, error) {
		f.Mu.Lock()
		if f.Definitions != nil {
			if locations, ok := f.Definitions[pos]; ok {
				f.Mu.Unlock()
				return locations, nil
			}
		}
		f.Mu.Unlock()

		req := lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: lsp.DocumentURI(uri)},
			Position:     lsp.Position(pos),
		}
		var resp []Location
		if err := cli.Call(ctx, "textDocument/definition", req, &resp); err != nil {
			return nil, err
		}
		f.Mu.Lock()
		if f.Definitions == nil {
			f.Definitions = make(map[Position][]Location)
		}
		if existing, ok := f.Definitions[pos]; ok {
			resp = existing
		} else {
			f.Definitions[pos] = resp
		}
		f.Mu.Unlock()
		return resp, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]Location), nil
}

func (cli *LSPClient) TypeDefinition(ctx context.Context, uri DocumentURI, pos Position) ([]Location, error) {
	req := lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: lsp.DocumentURI(uri),
		},
		Position: lsp.Position(pos),
	}
	var resp []Location
	if err := cli.Call(ctx, "textDocument/typeDefinition", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ensureLocalFile returns the cached TextDocumentItem for uri, reading and
// caching the file if necessary. Unlike DidOpen it does NOT send a didOpen
// notification — use this for read-only helpers (Locate, Line, ...) that
// just need the file body. The stub it creates has ServerOpened=false so
// that the next DidOpen() will still notify the LSP server. Safe for
// concurrent use.
func (cli *LSPClient) ensureLocalFile(uri DocumentURI) (*TextDocumentItem, error) {
	if f := cli.lookupFile(uri); f != nil {
		return f, nil
	}
	fd, err := os.ReadFile(uri.File())
	if err != nil {
		return nil, err
	}
	text := string(fd)
	nf := &TextDocumentItem{
		URI:          DocumentURI(uri),
		LanguageID:   cli.Language.String(),
		Version:      1,
		Text:         text,
		LineCounts:   utils.CountLines(text),
		Mu:           &sync.Mutex{},
		ServerOpened: false, // local-only stub; DidOpen() will notify if asked
	}
	cli.filesMu.Lock()
	if existing, ok := cli.files[uri]; ok {
		cli.filesMu.Unlock()
		return existing, nil
	}
	cli.files[uri] = nf
	cli.filesMu.Unlock()
	return nf, nil
}

// read file and get the text of block of range
func (cli *LSPClient) Locate(id Location) (string, error) {
	f, err := cli.ensureLocalFile(id.URI)
	if err != nil {
		return "", err
	}
	text := f.Text
	// get block text of range
	if id.Range.Start.Line < 0 || id.Range.Start.Line >= len(f.LineCounts) ||
		id.Range.End.Line < 0 || id.Range.End.Line >= len(f.LineCounts) {
		return "", fmt.Errorf("range %s out of bounds for %s", id.Range, id.URI)
	}
	start := f.LineCounts[id.Range.Start.Line] + id.Range.Start.Character
	end := f.LineCounts[id.Range.End.Line] + id.Range.End.Character
	if start < 0 || end > len(text) || start > end {
		return "", fmt.Errorf("range %s out of bounds for %s", id.Range, id.URI)
	}
	return text[start:end], nil
}

// get line text of pos
func (cli *LSPClient) Line(uri DocumentURI, pos int) string {
	f, err := cli.ensureLocalFile(uri)
	if err != nil {
		return ""
	}
	if pos < 0 || pos >= len(f.LineCounts) {
		return ""
	}
	start := f.LineCounts[pos]
	end := len(f.Text)
	if pos+1 < len(f.LineCounts) {
		end = f.LineCounts[pos+1]
	}
	return f.Text[start:end]
}

func (cli *LSPClient) LineCounts(uri DocumentURI) []int {
	f, err := cli.ensureLocalFile(uri)
	if err != nil {
		return nil
	}
	return f.LineCounts
}

func (cli *LSPClient) GetFile(uri DocumentURI) *TextDocumentItem {
	return cli.lookupFile(uri)
}

func (cli *LSPClient) GetParent(sym *DocumentSymbol) (ret *DocumentSymbol) {
	if sym == nil {
		return nil
	}
	if f := cli.lookupFile(sym.Location.URI); f != nil {
		f.Mu.Lock()
		defer f.Mu.Unlock()
		for _, s := range f.Symbols {
			if s != sym && s.Location.Range.Include(sym.Location.Range) {
				if ret == nil || ret.Location.Range.Include(s.Location.Range) {
					ret = s
				}
			}
		}
	}
	return
}

func (cli *LSPClient) getAllTokens(tokens SemanticTokens, file DocumentURI) []Token {
	start := Position{Line: 0, Character: 0}
	end := Position{Line: math.MaxInt32, Character: math.MaxInt32}
	return cli.getRangeTokens(tokens, file, Range{Start: start, End: end})
}

func (cli *LSPClient) getRangeTokens(tokens SemanticTokens, file DocumentURI, r Range) []Token {
	symbols := make([]Token, 0, len(tokens.Data)/5)
	line := 0
	character := 0

	for i := 0; i < len(tokens.Data); i += 5 {
		deltaLine := int(tokens.Data[i])
		deltaStart := int(tokens.Data[i+1])
		length := int(tokens.Data[i+2])
		tokenType := int(tokens.Data[i+3])
		tokenModifiersBitset := int(tokens.Data[i+4])

		line += deltaLine
		if deltaLine == 0 {
			character += deltaStart
		} else {
			character = deltaStart
		}

		currentPos := Position{Line: line, Character: character}
		if isPositionInRange(currentPos, r, false) {
			// fmt.Printf("Token at line %d, character %d, length %d, type %d, modifiers %b\n", line, character, length, tokenType, tokenModifiersBitset)
			tokenTypeName := getSemanticTokenType(tokenType, cli.tokenTypes)
			tokenModifierNames := getSemanticTokenModifier(tokenModifiersBitset, cli.tokenModifiers)
			loc := Location{URI: file, Range: Range{Start: currentPos, End: Position{Line: line, Character: character + length}}}
			text, _ := cli.Locate(loc)
			symbols = append(symbols, Token{
				Location:  loc,
				Type:      tokenTypeName,
				Modifiers: tokenModifierNames,
				Text:      text,
			})
		}
	}

	// sort it by start position
	sort.Slice(symbols, func(i, j int) bool {
		if symbols[i].Location.URI != symbols[j].Location.URI {
			return symbols[i].Location.URI < symbols[j].Location.URI
		}
		if symbols[i].Location.Range.Start.Line != symbols[j].Location.Range.Start.Line {
			return symbols[i].Location.Range.Start.Line < symbols[j].Location.Range.Start.Line
		}
		return symbols[i].Location.Range.Start.Character < symbols[j].Location.Range.Start.Character
	})

	return symbols
}

func (cli *LSPClient) FileStructure(ctx context.Context, file DocumentURI) ([]*DocumentSymbol, error) {
	syms, err := cli.DocumentSymbols(ctx, file)
	if err != nil {
		return nil, err
	}
	// construct symbol hierarchy through range relation, and represent it to DocumentSymobl.Children
	symbols := make([]*DocumentSymbol, 0, len(syms))
	for _, sym := range syms {
		symbols = append(symbols, sym)
	}
	return constructSymbolHierarchy(symbols), nil
}

func getSemanticTokenType(id int, semanticTokenTypes []string) string {
	if id < len(semanticTokenTypes) {
		return semanticTokenTypes[id]
	}
	return fmt.Sprintf("unknown(%d)", id)
}

func getSemanticTokenModifier(bitset int, semanticTokenModifiers []string) []string {
	var result []string
	for i, modifier := range semanticTokenModifiers {
		if bitset&(1<<uint(i)) != 0 {
			result = append(result, modifier)
		}
	}
	for i := len(semanticTokenModifiers); i < 32; i++ {
		if bitset&(1<<uint(i)) != 0 {
			result = append(result, fmt.Sprintf("unknown(%d)", i))
		}
	}
	return result
}

// constructSymbolHierarchy constructs a symbol hierarchy through range relation and represents it in DocumentSymbol.Children.
func constructSymbolHierarchy(symbols []*DocumentSymbol) []*DocumentSymbol {
	// Sort symbols by their start position
	sort.Slice(symbols, func(i, j int) bool {
		if symbols[i].Location.Range.Start.Line == symbols[j].Location.Range.Start.Line {
			return symbols[i].Location.Range.Start.Character < symbols[j].Location.Range.Start.Character
		}
		return symbols[i].Location.Range.Start.Line < symbols[j].Location.Range.Start.Line
	})

	var rootSymbols []*DocumentSymbol
	var stack []*DocumentSymbol

	for i := range symbols {
		symbol := symbols[i]

		// Pop symbols from the stack that are not parents of the current symbol
		for len(stack) > 0 && !stack[len(stack)-1].Location.Range.Include(symbol.Location.Range) {
			stack = stack[:len(stack)-1]
		}

		// If the stack is not empty, the top symbol is the parent of the current symbol
		if len(stack) > 0 {
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, symbol)
		} else {
			// If the stack is empty, the current symbol is a root symbol
			rootSymbols = append(rootSymbols, symbol)
		}

		// Push the current symbol onto the stack
		stack = append(stack, symbol)
	}

	return rootSymbols
}
