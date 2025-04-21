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
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cloudwego/abcoder/lang/utils"
	"github.com/sourcegraph/go-lsp"
)

// The SymbolKind values are defined at https://microsoft.github.io/language-server-protocol/specification.
const (
	SKUnknown       SymbolKind = 1
	SKFile          SymbolKind = 1
	SKModule        SymbolKind = 2
	SKNamespace     SymbolKind = 3
	SKPackage       SymbolKind = 4
	SKClass         SymbolKind = 5
	SKMethod        SymbolKind = 6
	SKProperty      SymbolKind = 7
	SKField         SymbolKind = 8
	SKConstructor   SymbolKind = 9
	SKEnum          SymbolKind = 10
	SKInterface     SymbolKind = 11
	SKFunction      SymbolKind = 12
	SKVariable      SymbolKind = 13
	SKConstant      SymbolKind = 14
	SKString        SymbolKind = 15
	SKNumber        SymbolKind = 16
	SKBoolean       SymbolKind = 17
	SKArray         SymbolKind = 18
	SKObject        SymbolKind = 19
	SKKey           SymbolKind = 20
	SKNull          SymbolKind = 21
	SKEnumMember    SymbolKind = 22
	SKStruct        SymbolKind = 23
	SKEvent         SymbolKind = 24
	SKOperator      SymbolKind = 25
	SKTypeParameter SymbolKind = 26
)

type SymbolKind = lsp.SymbolKind

type Position lsp.Position

func (r Position) Less(s Position) bool {
	if r.Line != s.Line {
		return r.Line < s.Line
	}
	return r.Character < s.Character
}

func (r Position) String() string {
	return fmt.Sprintf("%d:%d", r.Line, r.Character)
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type _Range Range

func (r Range) String() string {
	return fmt.Sprintf("%s-%s", r.Start, r.End)
}

func (r Range) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

func (r Range) MarshalJSON() ([]byte, error) {
	return json.Marshal(_Range(r))
}

type Location struct {
	URI   DocumentURI `json:"uri"`
	Range Range       `json:"range"`
}

func (l Location) String() string {
	return fmt.Sprintf("%s:%d:%d-%d:%d", l.URI, l.Range.Start.Line, l.Range.Start.Character, l.Range.End.Line, l.Range.End.Character)
}

var locationMarshalJSONInline = true

func SetLocationMarshalJSONInline(inline bool) {
	locationMarshalJSONInline = inline
}

type location Location

func (l Location) MarshalJSON() ([]byte, error) {
	if locationMarshalJSONInline {
		return []byte(fmt.Sprintf("%q", l.String())), nil
	}
	return json.Marshal(location(l))
}

func (l Location) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

type DocumentURI lsp.DocumentURI

func (l DocumentURI) File() string {
	return strings.TrimPrefix(string(l), "file://")
}

func NewURI(file string) DocumentURI {
	if !filepath.IsAbs(file) {
		file, _ = filepath.Abs(file)
	}
	return DocumentURI("file://" + file)
}

type DocumentRange struct {
	TextDocument lsp.TextDocumentIdentifier `json:"textDocument"`
	Range        Range                      `json:"range"`
}

type TextDocumentItem struct {
	URI         DocumentURI               `json:"uri"`
	LanguageID  string                    `json:"languageId"`
	Version     int                       `json:"version"`
	Text        string                    `json:"text"`
	LineCounts  []int                     `json:"-"`
	Symbols     map[Range]*DocumentSymbol `json:"-"`
	Definitions map[Position][]Location   `json:"-"`
}

type DocumentSymbol struct {
	Name     string            `json:"name"`
	Kind     SymbolKind        `json:"kind"`
	Tags     []json.RawMessage `json:"tags"`
	Location Location          `json:"location"`
	Children []*DocumentSymbol `json:"children"`
	Text     string            `json:"text"`
	Tokens   []Token           `json:"tokens"`
}

func (s *DocumentSymbol) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	r := *s
	if js, err := json.Marshal(r); err != nil {
		return nil, err
	} else {
		return js, nil
	}
}

func (s *DocumentSymbol) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *DocumentSymbol) String() string {
	if s == nil {
		return "null"
	}
	return fmt.Sprintf("%s %s %s", s.Name, s.Kind, s.Location)
}

type SemanticTokens struct {
	ResultID string   `json:"resultId"`
	Data     []uint32 `json:"data"`
}

type Token struct {
	Location  Location `json:"location"`
	Type      string   `json:"type"`
	Modifiers []string `json:"modifiers"`
	Text      string   `json:"text"`
}

func (t *Token) String() string {
	return fmt.Sprintf("%s %s %v %s", t.Text, t.Type, t.Modifiers, t.Location)
}

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

func (cli *LSPClient) DidOpen(ctx context.Context, file DocumentURI) (*TextDocumentItem, error) {
	if f, ok := cli.files[file]; ok {
		return f, nil
	}
	text, err := os.ReadFile(file.File())
	if err != nil {
		return nil, err
	}
	f := &TextDocumentItem{
		URI:        DocumentURI(file),
		LanguageID: cli.Language.String(),
		Version:    1,
		Text:       string(text),
		LineCounts: utils.CountLines(string(text)),
	}
	cli.files[file] = f
	req := DidOpenTextDocumentParams{
		TextDocument: *f,
	}
	if err := cli.Notify(ctx, "textDocument/didOpen", req); err != nil {
		return nil, err
	}
	return f, nil
}

func (cli *LSPClient) DocumentSymbols(ctx context.Context, file DocumentURI) (map[Range]*DocumentSymbol, error) {
	// f, ok := cli.files[file]
	// if ok {
	// 	return f.Symbols, nil
	// }
	// open file first
	f, err := cli.DidOpen(ctx, file)
	if err != nil {
		return nil, err
	}
	if f.Symbols != nil {
		return f.Symbols, nil
	}
	req := lsp.DocumentSymbolParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: lsp.DocumentURI(file),
		},
	}
	var resp []DocumentSymbol
	if err := cli.Call(ctx, "textDocument/documentSymbol", req, &resp); err != nil {
		return nil, err
	}
	// cache symbols
	f.Symbols = make(map[Range]*DocumentSymbol, len(resp))
	for i := range resp {
		s := &resp[i]
		f.Symbols[s.Location.Range] = s
	}
	return f.Symbols, nil
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

func (cli *LSPClient) SemanticTokens(ctx context.Context, id Location) ([]Token, error) {
	// open file first
	syms, err := cli.DocumentSymbols(ctx, id.URI)
	if err != nil {
		return nil, err
	}
	sym := syms[id.Range]
	if sym != nil && sym.Tokens != nil {
		return sym.Tokens, nil
	}

	uri := lsp.DocumentURI(id.URI)
	req := DocumentRange{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: uri,
		},
		Range: id.Range,
	}

	var resp SemanticTokens
	if err := cli.Call(ctx, "textDocument/semanticTokens/range", req, &resp); err != nil {
		return nil, err
	}

	toks := cli.getAllTokens(resp, id.URI)
	if sym != nil {
		sym.Tokens = toks
	}
	return toks, nil
}

func (cli *LSPClient) Definition(ctx context.Context, uri DocumentURI, pos Position) ([]Location, error) {
	// open file first
	f, err := cli.DidOpen(ctx, uri)
	if err != nil {
		return nil, err
	}

	// call
	req := lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: lsp.DocumentURI(uri),
		},
		Position: lsp.Position(pos),
	}
	var resp []Location
	if err := cli.Call(ctx, "textDocument/definition", req, &resp); err != nil {
		return nil, err
	}
	// cache definitions
	if f.Definitions == nil {
		f.Definitions = make(map[Position][]Location)
	}
	f.Definitions[pos] = resp
	return resp, nil
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

// read file and get the text of block of range
func (cli *LSPClient) Locate(id Location) (string, error) {
	f, ok := cli.files[id.URI]
	if !ok {
		// open file os
		fd, err := os.ReadFile(id.URI.File())
		if err != nil {
			return "", err
		}
		text := string(fd)
		f = &TextDocumentItem{
			URI:        DocumentURI(id.URI),
			LanguageID: cli.Language.String(),
			Version:    1,
			Text:       text,
			LineCounts: utils.CountLines(text),
		}
		cli.files[id.URI] = f
	}

	text := f.Text
	// get block text of range
	start := f.LineCounts[id.Range.Start.Line] + id.Range.Start.Character
	end := f.LineCounts[id.Range.End.Line] + id.Range.End.Character
	return text[start:end], nil
}

// get line text of pos
func (cli *LSPClient) Line(uri DocumentURI, pos int) string {
	f, ok := cli.files[uri]
	if !ok {
		// open file os
		fd, err := os.ReadFile(uri.File())
		if err != nil {
			return ""
		}
		text := string(fd)
		f = &TextDocumentItem{
			URI:        DocumentURI(uri),
			LanguageID: cli.Language.String(),
			Version:    1,
			Text:       text,
			LineCounts: utils.CountLines(text),
		}
		cli.files[uri] = f
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
	f, ok := cli.files[uri]
	if !ok {
		// open file os
		fd, err := os.ReadFile(uri.File())
		if err != nil {
			return nil
		}
		text := string(fd)
		f = &TextDocumentItem{
			URI:        DocumentURI(uri),
			LanguageID: cli.Language.String(),
			Version:    1,
			Text:       text,
			LineCounts: utils.CountLines(text),
		}
		cli.files[uri] = f
	}
	return f.LineCounts
}

func (cli *LSPClient) GetFile(uri DocumentURI) *TextDocumentItem {
	return cli.files[uri]
}

func (cli *LSPClient) GetParent(sym *DocumentSymbol) (ret *DocumentSymbol) {
	if sym == nil {
		return nil
	}
	if f, ok := cli.files[sym.Location.URI]; ok {
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

func (a Location) Include(b Location) bool {
	if a == b {
		return true
	}
	if a.URI != b.URI {
		return false
	}
	return isPositionInRange(b.Range.Start, a.Range, false) && isPositionInRange(b.Range.End, a.Range, true)
}

func (a Range) Include(b Range) bool {
	return isPositionInRange(b.Start, a, false) && isPositionInRange(b.End, a, true)
}

func isPositionInRange(pos Position, r Range, close bool) bool {
	if pos.Line < r.Start.Line || pos.Line > r.End.Line {
		return false
	}
	if pos.Line == r.Start.Line && pos.Character < r.Start.Character {
		return false
	}
	if pos.Line == r.End.Line {
		if close {
			return pos.Character <= r.End.Character
		} else {
			return pos.Character < r.End.Character
		}
	}
	return true
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

func (cli *LSPClient) WorkspaceSymbols(ctx context.Context, query string) ([]DocumentSymbol, error) {
	req := lsp.WorkspaceSymbolParams{
		Query: query,
	}
	var resp []DocumentSymbol
	if err := cli.Call(ctx, "workspace/symbol", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
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
