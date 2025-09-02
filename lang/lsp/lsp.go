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
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

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

func MakeLocation(uri DocumentURI, startLine, startChar, endLine, endChar int) Location {
	return Location{
		URI: uri,
		Range: Range{
			Start: Position{Line: startLine, Character: startChar},
			End:   Position{Line: endLine, Character: endChar},
		},
	}
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
	return DocumentURI("file://" + file)
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
