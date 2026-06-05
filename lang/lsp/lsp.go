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
	"path/filepath"
	"strings"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"

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

type SymbolRole int

const (
	DEFINITION SymbolRole = 1
	REFERENCE  SymbolRole = 2
)

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

func (r Range) String() string {
	return fmt.Sprintf("%s-%s", r.Start, r.End)
}

func (r Range) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

type _Range Range

func (r Range) MarshalJSON() ([]byte, error) {
	return json.Marshal(_Range(r))
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

func (a Range) Include(b Range) bool {
	return isPositionInRange(b.Start, a, false) && isPositionInRange(b.End, a, true)
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

type _Location Location

func (l Location) MarshalJSON() ([]byte, error) {
	if locationMarshalJSONInline {
		return []byte(fmt.Sprintf("%q", l.String())), nil
	}
	return json.Marshal(_Location(l))
}

func (l Location) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
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

type DocumentURI lsp.DocumentURI

func (l DocumentURI) File() string {
	return strings.TrimPrefix(string(l), "file://")
}

func NewURI(file string) DocumentURI {
	if !filepath.IsAbs(file) {
		file, _ = filepath.Abs(file)
	}
	// Canonicalise via realpath so symlinks don't produce two URIs for the
	// same physical file. clangd resolves symlinks internally and returns
	// realpath-based URIs in its responses; if we registered the file
	// under the symlinked path, the response URI wouldn't match our
	// `cli.files` key (and breaks every downstream Location comparison).
	if real, err := filepath.EvalSymlinks(file); err == nil && real != "" {
		file = real
	}
	return DocumentURI("file://" + file)
}

type TextDocumentItem struct {
	URI            DocumentURI               `json:"uri"`
	LanguageID     string                    `json:"languageId"`
	Version        int                       `json:"version"`
	Text           string                    `json:"text"`
	LineCounts     []int                     `json:"-"`
	Symbols        map[Range]*DocumentSymbol `json:"-"`
	Definitions    map[Position][]Location   `json:"-"`
	SemanticTokens *SemanticTokens           `json:"-"`
	// Mu protects Symbols, Definitions, SemanticTokens, ServerOpened from
	// concurrent access. Pointer so that copying TextDocumentItem (e.g.
	// when building didOpen params) doesn't copy the lock. RPC calls are
	// issued without holding this lock; only cache check / write are
	// guarded.
	Mu *sync.Mutex `json:"-"`
	// ServerOpened tracks whether we've actually sent textDocument/didOpen
	// for this file to the LSP server. Read helpers like ensureLocalFile
	// populate the local cache without notifying the server; DidOpen must
	// still send the notification on first transition to true so clangd
	// can answer subsequent AST queries (e.g. documentSymbol/definition).
	ServerOpened bool `json:"-"`
}

type DocumentSymbol struct {
	Name string `json:"name"`
	// Detail is the optional "more detail for this symbol" string from
	// the LSP spec — clangd populates it with the function signature
	// (e.g. "void(int x)") for SKMethod/SKFunction kinds, which lets us
	// skip our own text-level signature extraction in extractCppCallSig.
	Detail   string            `json:"detail,omitempty"`
	Kind     SymbolKind        `json:"kind"`
	Tags     []json.RawMessage `json:"tags"`
	Children []*DocumentSymbol `json:"children"`
	Text     string            `json:"text"`
	Tokens   []Token           `json:"tokens"`
	Node     *sitter.Node      `json:"-"`
	Role     SymbolRole        `json:"-"`

	// Older LSPs might return SymbolInformation[] which have `Location`.
	// Newer LSPs return DocumentSymbol[] which have `Range` and `SelectionRange`.
	// ABCoder uses `Location`, and converts `Range` to `Location` when needed.
	Location       Location `json:"location"`
	Range          *Range   `json:"range"`
	SelectionRange *Range   `json:"selectionRange"`
}

type TextDocumentPositionParams struct {
	/**
	 * The text document.
	 */
	TextDocument TextDocumentIdentifier `json:"textDocument"`

	/**
	 * The position inside the text document.
	 */
	Position Position `json:"position"`
}

type TextDocumentIdentifier struct {
	/**
	 * The text document's URI.
	 */
	URI DocumentURI `json:"uri"`
}

type Hover struct {
	Contents []MarkedString `json:"contents"`
	Range    *Range         `json:"range,omitempty"`
}

type MarkedString markedString

type markedString struct {
	Language string `json:"language"`
	Value    string `json:"value"`

	isRawString bool
}

type WorkspaceSymbolParams struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type SymbolInformation struct {
	Name          string     `json:"name"`
	Kind          SymbolKind `json:"kind"`
	Location      Location   `json:"location"`
	ContainerName string     `json:"containerName,omitempty"`
}

// ASTNode mirrors clangd's `textDocument/ast` response shape. It's NOT
// part of the LSP standard — clangd-only — and we use it to skip text
// heuristics for function body / typedef / using-declaration kind
// detection. Other LSPs returning "method not found" is normal and
// callers must fall back to text inspection.
//
// Reference: https://clangd.llvm.org/extensions#ast
//
// clangd uses short kind names (no "Decl" suffix). The constants below
// cover every kind the C++ collector keys off — keep them in sync with
// clangd's `clang::Decl::Kind` short-name table when extending.
const (
	ASTRoleDeclaration = "declaration"

	ASTKindCompound          = "Compound" // CompoundStmt — a function body
	ASTKindTypedef           = "Typedef"
	ASTKindTypeAlias         = "TypeAlias"
	ASTKindTypeAliasTemplate = "TypeAliasTemplate"
	ASTKindUsing             = "Using"
	ASTKindUsingShadow       = "UsingShadow"
	ASTKindUsingPack         = "UsingPack"
)

// AstHasFunctionBody reports whether `n` is a function declaration node
// with a function body (a Compound child). Lambdas in default arguments
// and other nested local constructs sit DEEPER than direct children, so
// a single-level scan rules them out — the classic
// `void f(std::function<void()> cb = []{});` declaration must NOT be
// classified as having a body.
func (n *ASTNode) HasFunctionBody() bool {
	if n == nil {
		return false
	}
	for _, ch := range n.Children {
		if ch != nil && ch.Kind == ASTKindCompound {
			return true
		}
	}
	return false
}

type ASTNode struct {
	// Role is a coarse category ("declaration", "statement", "expression",
	// "specifier", "type", "templateArgument"). Not load-bearing for our
	// use — we key off Kind.
	Role string `json:"role"`
	// Kind is the clang AST node class. Examples we care about:
	//   FunctionDecl / CXXMethodDecl / CXXConstructorDecl / CXXDestructorDecl
	//   TypedefDecl / TypeAliasDecl / TypeAliasTemplateDecl
	//   UsingDecl / UsingDirectiveDecl / UsingShadowDecl / UsingPackDecl
	//   CompoundStmt  (presence in children = function has body)
	Kind string `json:"kind"`
	// Detail is the clang-pretty-printed name/type of this node.
	Detail string `json:"detail,omitempty"`
	// Arcana is the raw clang "ast-dump" line for this node.
	Arcana string `json:"arcana,omitempty"`
	// Range covers the entire node.
	Range Range `json:"range,omitempty"`
	// Children are nested AST nodes.
	Children []*ASTNode `json:"children,omitempty"`
}

// TypeHierarchyItem represents a node in the type hierarchy tree.
//
// @since 3.17.0
type TypeHierarchyItem struct {
	Name           string      `json:"name"`
	Kind           SymbolKind  `json:"kind"`
	Detail         string      `json:"detail,omitempty"`
	URI            DocumentURI `json:"uri"`
	Range          Range       `json:"range"`
	SelectionRange Range       `json:"selectionRange"`
	Data           interface{} `json:"data,omitempty"`
}

func (cli *LSPClient) WorkspaceSymbols(ctx context.Context, query string) ([]DocumentSymbol, error) {
	req := WorkspaceSymbolParams{
		Query: query,
	}
	var resp []DocumentSymbol
	if err := cli.Call(ctx, "workspace/symbol", req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
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

func (cli *LSPClient) Hover(ctx context.Context, uri DocumentURI, line, character int) (*Hover, error) {
	if cli.provider != nil {
		// The type assertion is safe because the provider is for the specific language.
		return cli.provider.Hover(ctx, cli, uri, line, character)
	}
	// Default hover implementation (or return an error if not supported)
	// Default implementation (or return an error if not supported)
	return nil, fmt.Errorf("Hover not supported for this language")
}

func (cli *LSPClient) Implementation(ctx context.Context, uri DocumentURI, pos Position) ([]Location, error) {
	if cli.provider != nil {
		return cli.provider.Implementation(ctx, cli, uri, pos)
	}
	// Default implementation (or return an error if not supported)
	return nil, fmt.Errorf("implementation not supported for this language")
}

func (cli *LSPClient) WorkspaceSearchSymbols(ctx context.Context, query string) ([]SymbolInformation, error) {
	if cli.provider != nil {
		return cli.provider.WorkspaceSearchSymbols(ctx, cli, query)
	}
	// Default implementation (or return an error if not supported)
	return nil, fmt.Errorf("WorkspaceSearchSymbols not supported for this language")
}

func (cli *LSPClient) PrepareTypeHierarchy(ctx context.Context, uri DocumentURI, pos Position) ([]TypeHierarchyItem, error) {
	if cli.provider != nil {
		return cli.provider.PrepareTypeHierarchy(ctx, cli, uri, pos)
	}
	// Default implementation (or return an error if not supported)
	return nil, fmt.Errorf("PrepareTypeHierarchy not supported for this language")
}

func (cli *LSPClient) TypeHierarchySupertypes(ctx context.Context, item TypeHierarchyItem) ([]TypeHierarchyItem, error) {
	if cli.provider != nil {
		return cli.provider.TypeHierarchySupertypes(ctx, cli, item)
	}
	// Default implementation (or return an error if not supported)
	return nil, fmt.Errorf("TypeHierarchySupertypes not supported for this language")
}

func (cli *LSPClient) TypeHierarchySubtypes(ctx context.Context, item TypeHierarchyItem) ([]TypeHierarchyItem, error) {
	if cli.provider != nil {
		return cli.provider.TypeHierarchySubtypes(ctx, cli, item)
	}
	// Default implementation (or return an error if not supported)
	return nil, fmt.Errorf("TypeHierarchySubtypes not supported for this language")
}

// AST issues clangd's `textDocument/ast` request directly. The endpoint
// is a clangd extension (not standard LSP); other LSPs return
// MethodNotFound and the caller must fall back to text heuristics.
func (cli *LSPClient) AST(ctx context.Context, uri DocumentURI, rng Range) (*ASTNode, error) {
	params := struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Range        Range                  `json:"range"`
	}{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Range:        rng,
	}
	var result *ASTNode
	if err := cli.Call(ctx, "textDocument/ast", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}
