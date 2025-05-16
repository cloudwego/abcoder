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

package rust

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	lsp "github.com/cloudwego/abcoder/lang/lsp"
	"github.com/cloudwego/abcoder/lang/utils"
)

var _ lsp.LanguageSpec = (*RustSpec)(nil)

type RustSpec struct {
	repo   string
	crates []Module // path => name
}

type Module struct {
	Name string
	Path string
}

func NewRustSpec() *RustSpec {
	return &RustSpec{
		crates: []Module{},
	}
}

// func (c *RustSpec) HandleUnloadedSymbol(from lsp.Token, def lsp.Location) *lsp.DocumentSymbol {
// 	bs, err := os.ReadFile(def.URI.File())
// 	if err != nil {
// 		return nil
// 	}
// 	text := string(bs)
// 	lines := utils.CountLinesCached(text)
// 	defer utils.PutCount(lines)
// 	ds := lsp.GetDistance(text, lsp.Position{}, from.Location.Range.Start)
// 	if ds < 0 || ds >= len(text) {
// 		return nil
// 	}
// }

func (c *RustSpec) IsExternalEntityToken(tok lsp.Token) bool {
	if !c.IsEntityToken(tok) {
		return false
	}
	isStatic := tok.Type == "static"
	for _, m := range tok.Modifiers {
		if m == "library" {
			return true
		} else if isStatic && m == "macro" {
			// NOTICE: rust-analyzer didn't mark static macro as symbol here, thus we explicitly mark it as external to let collector collect its definition
			return true
		}
	}
	return false
}

func (c *RustSpec) TokenKind(tok lsp.Token) lsp.SymbolKind {
	switch tok.Type {
	case "macro":
		return lsp.SKFunction
	case "method":
		return lsp.SKMethod
	case "function":
		return lsp.SKFunction
	case "static":
		return lsp.SKVariable
	case "const":
		return lsp.SKConstant
	case "struct":
		return lsp.SKStruct
	case "enum":
		return lsp.SKEnum
	case "enumMember":
		return lsp.SKEnumMember
	case "typeAlias":
		return lsp.SKTypeParameter
	case "interface":
		return lsp.SKInterface
	case "variable":
		return lsp.SKVariable
	default:
		return lsp.SKUnknown
	}
}

func (c *RustSpec) IsStdToken(tok lsp.Token) bool {
	for _, m := range tok.Modifiers {
		if m == "defaultLibrary" {
			return true
		}
	}
	return false
}

func (c *RustSpec) IsDocToken(tok lsp.Token) bool {
	for _, m := range tok.Modifiers {
		if m == "documentation" || m == "injected" {
			return true
		}
	}
	return false
}

func (c *RustSpec) DeclareTokenOfSymbol(sym lsp.DocumentSymbol) int {
	for i, t := range sym.Tokens {
		if c.IsDocToken(t) {
			continue
		}
		for _, m := range t.Modifiers {
			if m == "declaration" {
				return i
			}
		}
	}
	return -1
}

func (c *RustSpec) IsPublicSymbol(sym lsp.DocumentSymbol) bool {
	id := c.DeclareTokenOfSymbol(sym)
	if id == -1 {
		return false
	}
	for _, m := range sym.Tokens[id].Modifiers {
		if m == "public" {
			return true
		}
	}
	return false
}

func (c *RustSpec) IsMainFunction(sym lsp.DocumentSymbol) bool {
	return sym.Kind == lsp.SKFunction && sym.Name == "main"
}

// Include: struct, enum , trait, typeAlias, const, static, variable, function, method of type, macro
// Exclude: field, method of trait, impl object, enum member
func (c *RustSpec) IsEntitySymbol(sym lsp.DocumentSymbol) bool {
	typ := sym.Kind
	return typ == lsp.SKMethod || typ == lsp.SKFunction || typ == lsp.SKVariable || typ == lsp.SKStruct || typ == lsp.SKEnum || typ == lsp.SKTypeParameter || typ == lsp.SKInterface || typ == lsp.SKConstant
}

func (c *RustSpec) IsEntityToken(tok lsp.Token) bool {
	typ := tok.Type
	return typ == "macro" || typ == "method" || typ == "function" || typ == "static" || typ == "const" || typ == "struct" || typ == "enum" || typ == "enumMember" || typ == "typeAlias" || typ == "interface" || typ == "variable"
}

func (c *RustSpec) HasImplSymbol() bool {
	return true
}

func hasKeyword(tokens []lsp.Token, keyword string) int {
	for i, tok := range tokens {
		if tok.Text == keyword && tok.Type == "keyword" {
			return i
		}
	}
	return -1
}

func findSpecificToken(tokens []lsp.Token, typ string, text string) int {
	for i := 0; i < len(tokens); i++ {
		if tokens[i].Type == typ && tokens[i].Text == text {
			return i
		}
	}
	return -1
}

func findSpecifiTokenUntil(tokens []lsp.Token, typ string, text string, start int, end int) int {
	for i := start; i < end; i++ {
		if tokens[i].Type == typ && tokens[i].Text == text {
			return i
		}
	}
	return -1
}

func (c *RustSpec) firstNotDocToken(tokens []lsp.Token) int {
	for i, tok := range tokens {
		if !c.IsDocToken(tok) {
			return i
		}
	}
	return -1
}

func (c *RustSpec) ImplSymbol(sym lsp.DocumentSymbol) (int, int, int) {
	tokens := sym.Tokens
	if sym.Kind != lsp.SKObject || hasKeyword(sym.Tokens, "impl") < 0 {
		return -1, -1, -1
	}

	start := c.firstNotDocToken(tokens)
	if start < 0 {
		return -1, -1, -1
	}

	// find the impl type token
	var implType, receiverType = -1, -1
	var fn = start + findSpecificToken(tokens[start:], "keyword", "fn")
	var forToken = findSpecifiTokenUntil(tokens, "keyword", "for", start, fn)

	for i := start; i < forToken; i++ {
		if tokens[i].Type == "interface" {
			implType = i
			break
		}
	}

	for i := forToken + 1; i < len(tokens); i++ {
		if c.IsEntityToken(tokens[i]) {
			receiverType = i
			break
		}
	}

	// check if `fn` has `pub` ahead
	if fn > 0 && tokens[fn-1].Type == "keyword" && tokens[fn-1].Text == "pub" {
		return implType, receiverType, fn - 1
	}
	return implType, receiverType, fn
}

func (c *RustSpec) FunctionSymbol(sym lsp.DocumentSymbol) (int, []int, []int, []int) {
	tokens := sym.Tokens
	if sym.Kind != lsp.SKMethod && sym.Kind != lsp.SKFunction {
		return -1, nil, nil, nil
	}

	start := c.firstNotDocToken(tokens)
	if start < 0 {
		return -1, nil, nil, nil
	}

	// exclude #[xxx]
	fn := start + findSpecificToken(tokens[start:], "keyword", "fn")
	if fn < 0 {
		return -1, nil, nil, nil
	}
	where := start + findSpecificToken(tokens[start:], "keyword", "where")
	if where == -1 {
		where = len(tokens) - 1
	}
	lines := utils.CountLinesCached(sym.Text)

	// find the typeParam's type token between "fn" and "("
	var typeParams []int
	s, e := findPair(sym.Text, *lines, sym.Location.Range.Start, sym.Tokens, '<', '>', fn+1, where, '(')
	for ; s >= 0 && s <= e; s++ {
		if c.IsEntityToken(tokens[s]) {
			typeParams = append(typeParams, s)
		}
	}

	// find the first '(' ')' pair after "fn"
	lc, rc := findPair(sym.Text, *lines, sym.Location.Range.Start, sym.Tokens, '(', ')', e, where, '<')
	// collect the inputParam's type token
	var inputParams []int
	for s := lc; s >= 0 && s <= rc; s++ {
		if c.IsEntityToken(tokens[s]) {
			inputParams = append(inputParams, s)
		}
	}

	// find the  outputs's type token
	var outputs []int
	if where == len(tokens)-1 {
		e = findSingle(sym.Text, *lines, sym.Location.Range.Start, sym.Tokens, "{", rc, where)
	} else {
		e = where
	}
	for s = rc + 1; s >= 0 && s <= e; s++ {
		// the first entity token
		if c.IsEntityToken(tokens[s]) {
			outputs = append(outputs, s)
		}
	}

	utils.PutCount(lines)

	return -1, typeParams, inputParams, outputs
}

func findSingle(text string, lines []int, textPos lsp.Position, tokens []lsp.Token, sep string, start int, end int) int {
	if start < 0 {
		start = 0
	}
	if end >= len(tokens) {
		end = len(tokens) - 1
	}
	if start >= len(tokens) {
		return -1
	}
	sPos := lsp.RelativePostionWithLines(lines, textPos, tokens[start].Location.Range.Start)
	ePos := lsp.RelativePostionWithLines(lines, textPos, tokens[end].Location.Range.End)
	pos := strings.Index(text[sPos:ePos], sep)
	if pos == -1 {
		return -1
	}
	pos += sPos
	for i := start; i <= end && i < len(tokens); i++ {
		rel := lsp.RelativePostionWithLines(lines, textPos, tokens[i].Location.Range.Start)
		if rel > pos {
			return i - 1
		}
	}
	return -1
}

func findPair(text string, lines []int, textPos lsp.Position, tokens []lsp.Token, lchar rune, rchar rune, start int, end int, notAllow rune) (int, int) {
	if start < 0 {
		start = 0
	}
	if end >= len(tokens) {
		end = len(tokens) - 1
	}
	if start >= len(tokens) {
		return -1, -1
	}

	startIndex := lsp.RelativePostionWithLines(lines, textPos, tokens[start].Location.Range.Start)

	lArrow := -1
	lCount := 0
	rArrow := -1
	notAllowCount := 0
	ctext := text[startIndex:]
	for i, c := range ctext {
		if c == notAllow && lCount == 0 {
			return -1, -1
		} else if c == lchar && notAllowCount == 0 {
			lCount++
			if lCount == 1 {
				lArrow = i
			}
		} else if c == rchar && notAllowCount == 0 {
			if rchar == '>' && ctext[i-1] == '-' {
				// notice: -> is not a pair in Rust
				continue
			}
			lCount--
			if lCount == 0 {
				rArrow = i
				break
			}
		}
	}
	if lArrow == -1 || rArrow == -1 {
		return -1, -1
	}
	lArrow += startIndex
	rArrow += startIndex

	s := -1
	e := -1
	for i := start; i <= end && i < len(tokens); i++ {
		rel := lsp.RelativePostionWithLines(lines, textPos, tokens[i].Location.Range.Start)
		if rel >= lArrow && s == -1 {
			s = i
		}
		if rel > rArrow {
			e = i - 1
			break
		}
	}

	return s, e
}

// find the [lsep, rspe] range after i token's end
// func findTokens(s int, e int, lsep, rsep string, symbol *DocumentSymbol) (int, int) {
// 	if s < 0 {
// 		s = 0
// 	}
// 	// find the range of the token from i token
// 	startIndex := lsp.RelativePostion(symbol.Text, symbol.Location.Range.Start, symbol.Tokens[s].Location.Range.End)
// 	lArrow := strings.Index(symbol.Text[startIndex:], lsep)
// 	rArrow := strings.Index(symbol.Text[startIndex:], rsep)

// 	start := -1
// 	end := -1
// 	if lArrow != -1 && rArrow != -1 {
// 		lArrow += startIndex
// 		rArrow += startIndex
// 		lLine := strings.Count(symbol.Text[:lArrow], "\n")
// 		if lLine > 0 {
// 			lArrow = lArrow - strings.LastIndex(symbol.Text[:lArrow], "\n") - 1
// 		} else {
// 			lArrow += symbol.Location.Range.Start.Character
// 		}
// 		rLine := strings.Count(symbol.Text[:rArrow], "\n")
// 		if rLine > 0 {
// 			rArrow = rArrow - strings.LastIndex(symbol.Text[:rArrow], "\n") - 1
// 		} else {
// 			rArrow += symbol.Location.Range.Start.Character
// 		}
// 		lPos := Position{lLine + symbol.Location.Range.Start.Line, lArrow}
// 		rPos := Position{rLine + symbol.Location.Range.Start.Line, rArrow}
// 		for ; s <= e; s++ {
// 			if symbol.Tokens[s].Location.Range.Start.Less(lPos) {
// 				continue
// 			}
// 			if rPos.Less(symbol.Tokens[s].Location.Range.Start) {
// 				end = s - 1
// 				break
// 			}
// 			if start == -1 {
// 				start = s
// 			}
// 		}
// 	}
// 	return start, end
// }

var crateReg = regexp.MustCompile(`^[a-z][a-z0-9\-_]*\-\d+\.\d+\.\d+$`)

func getCrateAndMod(path string) (string, string) {
	// find last /src
	idx := strings.LastIndex(path, "/src/")
	mod := getMod(path[idx+5:])
	path = path[:idx]
	idx = strings.LastIndex(path, "/")
	crate := path[idx+1:]
	return crate, mod
}

func (c *RustSpec) ShouldSkip(path string) bool {
	if strings.Contains(path, "/target/") {
		return true
	} else if !strings.HasSuffix(path, ".rs") {
		return true
	}
	return false
}

func (c *RustSpec) NameSpace(path string) (string, string, error) {
	// external lib
	if !strings.HasPrefix(path, c.repo) {
		crate, mod := getCrateAndMod(path)
		var cname = crate
		if crateReg.MatchString(crate) {
			// third-party lib
			// NOTICE: replace "-" befor version with "@"
			idx := strings.LastIndex(crate, "-")
			cname = crate[:idx]
			cversion := crate[idx+1:]
			crate = cname + "@" + cversion
		} else if !strings.Contains(path, "rust/library/std") {
			// none-std lib, can't give crate at present
			crate = ""
		}
		if mod == "" {
			return crate, cname, nil
		}
		return crate, cname + "::" + mod, nil
	}

	// check if path has prefix in a crate
	for _, n := range c.crates {
		if strings.HasPrefix(path, n.Path) {
			rel, err := filepath.Rel(n.Path, path)
			if err != nil {
				return "", "", err
			}
			pkg := getMod(rel)
			if pkg == "" {
				return n.Name, n.Name, nil
			}
			return n.Name, n.Name + "::" + pkg, nil
		}
	}
	return "", "", fmt.Errorf("not found crate for %s", path)
}

func getMod(relPath string) string {
	base := filepath.Base(relPath)
	// lib path, its namespace is the parent dir
	if base == "mod.rs" {
		relPath = filepath.Dir(relPath)
	} else if base == "lib.rs" || relPath == "main.rs" {
		relPath = ""
	} else {
		relPath = strings.TrimSuffix(relPath, ".rs")
	}
	if relPath == "" {
		return ""
	}
	return strings.ReplaceAll(relPath, string(filepath.Separator), "::")
}

var nameRegex = regexp.MustCompile(`name\s*=\s*"([^"]+)"`)

// implement LanguageSpec.CollectModules
func (c *RustSpec) WorkSpace(root string) (map[string]string, error) {
	c.repo = root
	var rets = map[string]string{}

	scanner := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		base := filepath.Base(path)
		// dir := filepath.Dir(path)

		// collect module
		if base == "Cargo.toml" {
			// read Cargo.toml
			cargo, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			// parse Cargo.toml
			lines := strings.Split(string(cargo), "\n")

			for i, line := range lines {
				// locate [package]
				if strings.HasPrefix(line, "[package]") {
					if err := c.collect(&i, lines, path, rets); err != nil {
						return err
					}
				}

				// // locate [bin]
				// if strings.HasPrefix(line, "[bin]") {
				// 	c.collect(i, lines, path, root, &rets)
				// 	// TODO:
				// }
			}
		}
		return nil
	}
	err := filepath.Walk(root, scanner)
	if err != nil {
		return nil, err
	}
	// sort c.crates by length of path
	sort.Slice(c.crates, func(i, j int) bool {
		return len(c.crates[i].Path) > len(c.crates[j].Path)
	})

	return rets, nil
}

func (c *RustSpec) collect(i *int, lines []string, path string, rets map[string]string) error {
	dir := filepath.Join(filepath.Dir(path), "src")
	if _, err := os.Stat(dir); err != nil {
		dir = filepath.Dir(path)
	}
	for j := *i + 1; j < len(lines); j++ {
		// name = ""
		if m := nameRegex.FindStringSubmatch(lines[j]); m != nil {
			// rel, err := filepath.Rel(root, dir)
			// if err != nil {
			// 	rel = dir
			// }
			c.crates = append(c.crates, Module{
				Name: m[1],
				Path: dir,
			})
			rets[m[1]] = dir
			*i = j
			return nil
		}
		// // path = ""
		// if m := pathRegex.FindStringSubmatch(lines[j]); m != nil {
		// 	mod.Path = m[1]
		// }
	}
	return nil
}

func (c *RustSpec) GetUnloadedSymbol(from lsp.Token, loc lsp.Location) (string, error) {
	// TODO: may need handle more cases
	return ExtractLazyStaticeSymbol(loc)
}
