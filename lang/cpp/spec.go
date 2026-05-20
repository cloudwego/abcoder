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

package cpp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	lsp "github.com/cloudwego/abcoder/lang/lsp"
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/cloudwego/abcoder/lang/utils"
)

// clangd semantic-token type names.
const (
	tokClass         = "class"
	tokStruct        = "struct"
	tokType          = "type"
	tokInterface     = "interface"
	tokConcept       = "concept"
	tokEnum          = "enum"
	tokEnumMember    = "enumMember"
	tokFunction      = "function"
	tokMethod        = "method"
	tokMacro         = "macro"
	tokVariable      = "variable"
	tokParameter     = "parameter"
	tokTypeParameter = "typeParameter"
	tokNamespace     = "namespace"
	tokComment       = "comment"
	tokModifier      = "modifier"
	tokBracket       = "bracket"
	tokLabel         = "label"
	tokOperator      = "operator"
	tokProperty      = "property"
	tokUnknown       = "unknown"
)

// clangd semantic-token modifier names.
const (
	modDeclaration   = "declaration"
	modDefinition    = "definition"
	modGlobalScope   = "globalScope"
	modDefaultLibrary = "defaultLibrary"
)

// isReferenceTypeToken reports whether tok references (not declares) a type —
// eligible as a base class or alias target.
func isReferenceTypeToken(tok lsp.Token) bool {
	switch tok.Type {
	case tokClass, tokStruct, tokType:
	default:
		return false
	}
	for _, m := range tok.Modifiers {
		if m == modDeclaration || m == modDefinition {
			return false
		}
	}
	return true
}

type CppSpec struct {
	repo     string // repository root absolute realpath
	selfName string // "host/org/repo" of the repo being parsed

	// User-declared sysroot prefixes. Any file path under one of these is
	// bucketed under module `cstdlib`. Set via the abcoder `--sysroot`
	// flag (repeatable). Order doesn't matter — first match wins.
	sysroots []string

	repoMu   sync.Mutex
	repoMods map[string]repoInfo // dir containing .git -> resolved repo info

	// clangd often reports Range covering only the identifier, so alias /
	// base-class detection falls back to reading source. declCache memoises
	// declarationText per (URI, line, col).
	srcMu     sync.Mutex
	srcLines  map[string][]string
	declCache map[declKey]string
}

type declKey struct {
	uri  lsp.DocumentURI
	line int
	col  int
}

type repoInfo struct {
	root string // repository root on disk
	name string // "org/repo" derived from remote.origin.url, falls back to base(root)
}

func (c *CppSpec) ProtectedSymbolKinds() []lsp.SymbolKind {
	return []lsp.SymbolKind{lsp.SKFunction, lsp.SKMethod, lsp.SKVariable, lsp.SKConstant, lsp.SKClass, lsp.SKStruct}
}

func NewCppSpec() *CppSpec {
	return &CppSpec{
		repoMods:  map[string]repoInfo{},
		srcLines:  map[string][]string{},
		declCache: map[declKey]string{},
	}
}

// SetSysroots registers path prefixes whose contents are classified as
// the `cstdlib` module. Entries are realpath-resolved to match clangd URIs.
func (c *CppSpec) SetSysroots(roots []string) {
	c.sysroots = c.sysroots[:0]
	for _, r := range roots {
		if r = strings.TrimSpace(r); r == "" {
			continue
		}
		c.sysroots = append(c.sysroots, canonicalizeAbs(r))
	}
}

// canonicalizeAbs returns an absolute, realpath-resolved version of p.
// Either step is allowed to fail; the remaining transformations still
// apply. Used so path-prefix comparisons (repo, sysroots) match what
// clangd reports in sym.Location.URI.
func canonicalizeAbs(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	if real, err := filepath.EvalSymlinks(p); err == nil && real != "" {
		p = real
	}
	return p
}

// sourceLine returns line N (0-based) from the file at uri, reading and
// caching the file content on first access. Returns "" on any error or if
// N is out of range. Safe for concurrent use.
func (c *CppSpec) sourceLine(uri lsp.DocumentURI, n int) string {
	path := uri.File()
	c.srcMu.Lock()
	lines, ok := c.srcLines[path]
	c.srcMu.Unlock()
	if !ok {
		b, err := os.ReadFile(path)
		if err != nil {
			return ""
		}
		lines = strings.Split(string(b), "\n")
		c.srcMu.Lock()
		c.srcLines[path] = lines
		c.srcMu.Unlock()
	}
	if n < 0 || n >= len(lines) {
		return ""
	}
	return lines[n]
}

// declarationText returns the declaration source from sym's start line up to
// the next `;` or `{` at template depth 0. clangd's Range can start at the
// identifier or on a preceding comment line, so we read from the start of the
// line and skip line-comment-only lines.
func (c *CppSpec) declarationText(sym lsp.DocumentSymbol) string {
	startLine := sym.Location.Range.Start.Line
	key := declKey{uri: sym.Location.URI, line: startLine, col: sym.Location.Range.Start.Character}
	c.srcMu.Lock()
	cached, ok := c.declCache[key]
	c.srcMu.Unlock()
	if ok {
		return cached
	}
	const maxLines = 8
	var buf strings.Builder
	depth := 0
	wroteContent := false
	for off := 0; off < maxLines; off++ {
		raw := c.sourceLine(sym.Location.URI, startLine+off)
		if raw == "" && off > 0 {
			if wroteContent {
				buf.WriteByte('\n')
			}
			continue
		}
		line := raw
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		if strings.TrimSpace(line) == "" {
			if wroteContent {
				buf.WriteByte('\n')
			}
			continue
		}
		for i := 0; i < len(line); i++ {
			ch := line[i]
			switch ch {
			case '<':
				depth++
			case '>':
				if depth > 0 {
					depth--
				}
			case ';', '{':
				if depth == 0 {
					buf.WriteByte(ch)
					out := buf.String()
					c.srcMu.Lock()
					c.declCache[key] = out
					c.srcMu.Unlock()
					return out
				}
			}
			buf.WriteByte(ch)
		}
		wroteContent = true
		buf.WriteByte('\n')
	}
	out := buf.String()
	c.srcMu.Lock()
	c.declCache[key] = out
	c.srcMu.Unlock()
	return out
}

func (c *CppSpec) FileImports(content []byte) ([]uniast.Import, error) {
	return nil, nil
}

// XXX: maybe multi module support for C++?
func (c *CppSpec) WorkSpace(root string) (map[string]string, error) {
	if _, err := filepath.Abs(root); err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}
	// Path comparisons must match clangd's URI form (realpath).
	c.repo = canonicalizeAbs(root)
	absPath := c.repo

	// Derive a module name for this repo from its own .git/config; fall back
	// to the directory basename so the JSON never gets the generic "current".
	name := ""
	if info, err := os.Stat(filepath.Join(absPath, ".git")); err == nil && info.IsDir() {
		name = readGitOriginOrgRepo(filepath.Join(absPath, ".git", "config"))
	}
	if name == "" {
		name = filepath.Base(absPath)
	}
	c.selfName = name

	return map[string]string{name: absPath}, nil
}

// returns: modname, pathpath, error
// Multiple symbols with the same name could occur (for example in the Linux kernel).
// The identify is mod::pkg::name. So we use the pkg (the file name) to distinguish them.
func (c *CppSpec) NameSpace(path string, file *uniast.File) (string, string, error) {
	// Build-time generated artifacts (IDL/proto/blade glue under
	// build64_release). Route to a dedicated module so they're
	// distinguishable from hand-written project sources.
	if i := strings.Index(path, "/build64_release/"); i >= 0 {
		return "build_generated", path[i+len("/build64_release/"):], nil
	}
	if hasPathPrefix(path, c.repo) {
		rel, _ := filepath.Rel(c.repo, path)
		return c.selfName, rel, nil
	}
	// User-declared sysroot(s): bucket every header/source under them as
	// `cstdlib`. Stripping the sysroot prefix keeps pkg paths stable across
	// machines that install toolchains in different locations.
	for _, sr := range c.sysroots {
		if sr == "" {
			continue
		}
		if hasPathPrefix(path, sr) {
			rel, _ := filepath.Rel(sr, path)
			return "cstdlib", rel, nil
		}
	}
	if info, ok := c.lookupExternalRepo(path); ok {
		relpath, err := filepath.Rel(info.root, path)
		if err != nil {
			relpath = path
		}
		return info.name, relpath, nil
	}
	return "external", path, nil
}

// hasPathPrefix is HasPrefix that respects path boundaries: "/a/foo" is not a
// prefix of "/a/foobar". This avoids false matches when one repo's path is a
// textual prefix of another's (e.g. /freq vs /freq_service).
func hasPathPrefix(p, root string) bool {
	if !strings.HasPrefix(p, root) {
		return false
	}
	if len(p) == len(root) {
		return true
	}
	return p[len(root)] == filepath.Separator
}

// lookupExternalRepo walks upward from path until it finds a directory holding
// a .git entry, parses remote.origin.url to derive "org/repo", and caches the
// result. Returns ok=false if no enclosing git repo is found.
func (c *CppSpec) lookupExternalRepo(path string) (repoInfo, bool) {
	dir := filepath.Dir(path)
	visited := []string{}
	for {
		if dir == "" || dir == "/" || dir == "." {
			break
		}
		c.repoMu.Lock()
		info, hit := c.repoMods[dir]
		c.repoMu.Unlock()
		if hit {
			c.cacheChain(visited, info)
			if info.root == "" {
				return repoInfo{}, false
			}
			return info, true
		}
		gitPath := filepath.Join(dir, ".git")
		if fi, err := os.Stat(gitPath); err == nil && fi.IsDir() {
			name := readGitOriginOrgRepo(filepath.Join(gitPath, "config"))
			if name == "" {
				name = filepath.Base(dir)
			}
			info = repoInfo{root: dir, name: name}
			c.repoMu.Lock()
			c.repoMods[dir] = info
			c.repoMu.Unlock()
			c.cacheChain(visited, info)
			return info, true
		}
		visited = append(visited, dir)
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// negative-cache so future lookups under the same path don't re-walk
	neg := repoInfo{}
	c.repoMu.Lock()
	for _, d := range visited {
		c.repoMods[d] = neg
	}
	c.repoMu.Unlock()
	return repoInfo{}, false
}

func (c *CppSpec) cacheChain(dirs []string, info repoInfo) {
	if len(dirs) == 0 {
		return
	}
	c.repoMu.Lock()
	defer c.repoMu.Unlock()
	for _, d := range dirs {
		c.repoMods[d] = info
	}
}

// readGitOriginOrgRepo parses a git config file and returns "host/org/repo"
// derived from the [remote "origin"] url. Supports both ssh and https forms:
//
//	git@code.byted.org:data/cppservice          -> code.byted.org/data/cppservice
//	https://code.byted.org/data-arch/feathub.git -> code.byted.org/data-arch/feathub
//
// When the host can't be determined, returns "org/repo" without prefix.
func readGitOriginOrgRepo(cfgPath string) string {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return ""
	}
	inOrigin := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "[") {
			inOrigin = strings.Contains(line, `remote "origin"`)
			continue
		}
		if !inOrigin {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 || strings.TrimSpace(line[:eq]) != "url" {
			continue
		}
		u := strings.TrimSpace(line[eq+1:])
		u = strings.TrimSuffix(u, ".git")
		host := ""
		// Prefer scheme-based parse first so `https://...` isn't
		// mistaken for the ssh `user@host:path` form (both contain `:`).
		if i := strings.Index(u, "://"); i >= 0 {
			// https://host/org/repo
			rest := u[i+3:]
			if at := strings.LastIndex(rest, "@"); at >= 0 {
				rest = rest[at+1:] // strip user@
			}
			if slash := strings.IndexByte(rest, '/'); slash >= 0 {
				host = rest[:slash]
				u = rest[slash+1:]
			} else {
				u = rest
			}
		} else if i := strings.LastIndex(u, ":"); i >= 0 && !strings.Contains(u[:i], "/") {
			// ssh: git@host:org/repo
			head := u[:i]
			if at := strings.LastIndex(head, "@"); at >= 0 {
				host = head[at+1:]
			} else {
				host = head
			}
			u = u[i+1:]
		}
		parts := strings.Split(u, "/")
		var orgRepo string
		if n := len(parts); n >= 2 {
			orgRepo = parts[n-2] + "/" + parts[n-1]
		} else {
			orgRepo = u
		}
		if host != "" {
			return host + "/" + orgRepo
		}
		return orgRepo
	}
	return ""
}

func (c *CppSpec) ShouldSkip(path string) bool {
	// Build-time generated artifacts (IDL/proto/blade codegen under
	// build64_release). Skipping at scanner level avoids the heavy
	// DocumentSymbols + per-sym Locate/SemanticTokens cost. Edges from
	// in-repo source that reference these headers still resolve via the
	// fake-Unknown fallback in getSymbolByLocation.
	if strings.Contains(path, "/build64_release/") {
		return true
	}
	if (strings.HasSuffix(path, ".cpp") && !strings.HasSuffix(path, "_test.cpp")) || strings.HasSuffix(path, ".h") {
		return false
	}
	return true
}

func (c *CppSpec) IsDocToken(tok lsp.Token) bool {
	return tok.Type == tokComment
}

func (c *CppSpec) DeclareTokenOfSymbol(sym lsp.DocumentSymbol) int {
	for i, t := range sym.Tokens {
		if c.IsDocToken(t) {
			continue
		}
		for _, m := range t.Modifiers {
			if m == modDeclaration {
				return i
			}
		}
	}
	return -1
}

func (c *CppSpec) IsEntityToken(tok lsp.Token) bool {
	for _, m := range tok.Modifiers {
		if m == modDeclaration || m == modDefinition {
			return false
		}
	}
	return tok.Type == tokClass || tok.Type == tokFunction || tok.Type == tokMethod || tok.Type == tokVariable
}

func (c *CppSpec) IsStdToken(tok lsp.Token) bool {
	for _, m := range tok.Modifiers {
		if m == modDefaultLibrary {
			return true
		}
	}
	return false
}

func (c *CppSpec) TokenKind(tok lsp.Token) lsp.SymbolKind {
	switch tok.Type {
	case tokClass:
		return lsp.SKClass
	case tokEnum:
		return lsp.SKEnum
	case tokEnumMember:
		return lsp.SKEnumMember
	case tokFunction, tokMacro:
		return lsp.SKFunction
	// rust spec does not treat parameter as a variable
	case tokParameter:
		return lsp.SKVariable
	case tokTypeParameter:
		return lsp.SKTypeParameter
	case tokMethod:
		return lsp.SKMethod
	case tokNamespace:
		return lsp.SKNamespace
	case tokVariable:
		return lsp.SKVariable
	case tokInterface, tokConcept, tokModifier, tokType, tokBracket, tokComment, tokLabel, tokOperator, tokProperty, tokUnknown:
		return lsp.SKUnknown
	}
	panic(fmt.Sprintf("Weird token type: %s at %+v\n", tok.Type, tok.Location))
}

func (c *CppSpec) IsMainFunction(sym lsp.DocumentSymbol) bool {
	return sym.Kind == lsp.SKFunction && sym.Name == "main"
}

func (c *CppSpec) IsEntitySymbol(sym lsp.DocumentSymbol) bool {
	typ := sym.Kind
	return typ == lsp.SKFunction || typ == lsp.SKMethod || typ == lsp.SKVariable || typ == lsp.SKConstant || typ == lsp.SKClass || typ == lsp.SKStruct
}

func (c *CppSpec) IsPublicSymbol(sym lsp.DocumentSymbol) bool {
	id := c.DeclareTokenOfSymbol(sym)
	if id == -1 {
		return false
	}
	for _, m := range sym.Tokens[id].Modifiers {
		if m == modGlobalScope {
			return true
		}
	}
	return false
}

func (c *CppSpec) HasImplSymbol() bool {
	return true
}

func (c *CppSpec) ImplSymbol(sym lsp.DocumentSymbol) (int, int, int) {
	inter := -1
	fn := -1

	// Only treat class/struct as impl container in C++
	if sym.Kind != lsp.SKClass && sym.Kind != lsp.SKStruct {
		return inter, -1, fn
	}

	want := cppShortTypeName(sym.Name)
	if want == "" {
		return inter, -1, fn
	}

	// Prefer type-ish tokens that match the receiver name.
	for i, tok := range sym.Tokens {
		if tok.Text != want {
			continue
		}
		switch tok.Type {
		case tokClass, tokStruct:
			return inter, i, fn
		}
	}

	return inter, -1, fn
}

// IsTypedefSymbol reports whether the symbol is a `typedef X Y;` declaration.
// Typedefs stay in the AST as Type entries with TypeKind=typedef.
func (c *CppSpec) IsTypedefSymbol(sym lsp.DocumentSymbol) bool {
	return strings.HasPrefix(strings.TrimSpace(c.declarationText(sym)), "typedef ")
}

// IsUsingAlias reports whether the symbol is one of the two C++ alias forms:
//   - `using Y = X;`        — pure rename, introduces no new type
//   - `using NS::Name;`     — using-declaration that imports `NS::Name`
//     into the current scope
//
// References to either should resolve to the underlying type X / NS::Name;
// the alias itself must be dropped from the AST (otherwise inheritance
// edges of the form `class D : public Y` would produce a phantom Type
// node `<scope>::Y::Name` instead of pointing at the real base).
//
// `using namespace foo;` (a namespace-import directive) is NOT an alias;
// it doesn't introduce a name binding into the type system.
func (c *CppSpec) IsUsingAlias(sym lsp.DocumentSymbol) bool {
	trim := strings.TrimSpace(c.declarationText(sym))
	if !strings.HasPrefix(trim, "using ") {
		return false
	}
	// using namespace foo; — name-import directive, not an alias.
	if strings.HasPrefix(trim, "using namespace ") {
		return false
	}
	// using Y = X;  — always a type alias (introduces new type binding).
	if strings.Contains(trim, "=") {
		return true
	}
	// using NS::Name; — using-declaration. Only treat it as an alias when
	// the imported entity is a TYPE (class/struct/enum/typeParameter/
	// interface). For non-type imports like `using Base::foo;` (member
	// function) or `using std::swap;` (free function), this is name-
	// import only — collapsing it to an alias and dropping the symbol
	// makes those overloads disappear from the AST.
	switch sym.Kind {
	case lsp.SKClass, lsp.SKStruct, lsp.SKEnum, lsp.SKTypeParameter, lsp.SKInterface:
		return true
	}
	return false
}

// AliasTargetTokenIndex returns the token index of the aliased type's
// reference for a `using Y = X;` symbol — the token whose Definition leads
// to the real type. Returns -1 when not a using-alias or the target can't
// be located.
func (c *CppSpec) AliasTargetTokenIndex(sym lsp.DocumentSymbol) int {
	if !c.IsUsingAlias(sym) {
		return -1
	}
	// First non-declaration class/struct/type token. Namespaces in front
	// (`::ns::Foo::`) are namespace-kind and naturally skipped.
	for i, tok := range sym.Tokens {
		if isReferenceTypeToken(tok) {
			return i
		}
	}
	return -1
}

// baseSpan marks the byte range [s,e) of one base specifier within
// `declarationText`. Shared between BaseClassTokens and BaseClassRefs.
type baseSpan struct{ s, e int }

// parseBaseSpecifiers walks a class symbol's declaration text, isolates
// the base clause (between the first `:` and the body-opening `{`), and
// splits it on commas at template depth 0. Each entry in the returned
// slice points at one specifier like "public NS::Base<T,U>". Returns
// (decl, nil) when there's no base clause (forward decl, plain class).
func (c *CppSpec) parseBaseSpecifiers(sym lsp.DocumentSymbol) (string, []baseSpan) {
	if sym.Kind != lsp.SKClass && sym.Kind != lsp.SKStruct {
		return "", nil
	}
	decl := c.declarationText(sym)
	if decl == "" {
		return "", nil
	}
	bodyStart := strings.IndexByte(decl, '{')
	if bodyStart < 0 {
		return decl, nil
	}
	colon := strings.IndexByte(decl, ':')
	if colon < 0 || colon >= bodyStart {
		return decl, nil
	}
	var specs []baseSpan
	depth := 0
	specStart := colon + 1
	for i := colon + 1; i < bodyStart; i++ {
		switch decl[i] {
		case '<':
			depth++
		case '>':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				specs = append(specs, baseSpan{specStart, i})
				specStart = i + 1
			}
		}
	}
	specs = append(specs, baseSpan{specStart, bodyStart})
	return decl, specs
}

// BaseClassTokens returns indices of tokens naming each base class of a
// class/struct symbol (`class Foo : public Bar, protected Baz<T>`).
// Returns nil for forward decls or non-class symbols. Template
// arguments (`Req`/`Rsp` in `Provider<Req,Rsp>`) are filtered out by
// depth-0 enforcement.
func (c *CppSpec) BaseClassTokens(sym lsp.DocumentSymbol) []int {
	if len(sym.Tokens) == 0 {
		return nil
	}
	decl, specs := c.parseBaseSpecifiers(sym)
	if len(specs) == 0 {
		return nil
	}

	// Token offsets are relative to sym.Location.Range.Start. declarationText
	// is anchored at the same start, so offsets line up.
	lines := utils.CountLinesPooled(decl)
	bases := make([]int, 0, len(specs))
	for _, sp := range specs {
		// Pick the first reference token of type class/struct/type at
		// template depth 0 within this specifier.
		for i, tok := range sym.Tokens {
			if !isReferenceTypeToken(tok) {
				continue
			}
			if l := tok.Location.Range.Start.Line - sym.Location.Range.Start.Line; l < 0 || l >= len(*lines) {
				continue
			}
			off := lsp.RelativePostionWithLines(*lines, sym.Location.Range.Start, tok.Location.Range.Start)
			if off < sp.s || off >= sp.e {
				continue
			}
			// Verify template depth at this offset (relative to specifier).
			d := 0
			for j := sp.s; j < off; j++ {
				switch decl[j] {
				case '<':
					d++
				case '>':
					if d > 0 {
						d--
					}
				}
			}
			if d != 0 {
				continue
			}
			bases = append(bases, i)
			break // one base per specifier
		}
	}
	return bases
}

// BaseClassRef names one base class plus the file position of its name,
// usable for textDocument/definition. Source-text-based; works when
// clangd's semantic-tokens stream omits the base (typical for unresolved
// external templated bases).
type BaseClassRef struct {
	Name string
	Loc  lsp.Location
}

// BaseClassRefs parses a class's base clause directly from declarationText
// rather than relying on sym.Tokens (which clangd may not populate for
// unresolved templated bases).
func (c *CppSpec) BaseClassRefs(sym lsp.DocumentSymbol) []BaseClassRef {
	decl, specs := c.parseBaseSpecifiers(sym)
	if len(specs) == 0 {
		return nil
	}

	// declarationText starts at column 0 of sym's first line (sourceLine
	// returns the full line), so a char-offset in decl maps to file
	// (sym.Location.Range.Start.Line + line, col) directly.
	lineStarts := []int{0}
	for i, b := range []byte(decl) {
		if b == '\n' {
			lineStarts = append(lineStarts, i+1)
		}
	}
	posOf := func(off int) lsp.Position {
		line := 0
		for li, st := range lineStarts {
			if st > off {
				break
			}
			line = li
		}
		col := off - lineStarts[line]
		return lsp.Position{
			Line:      sym.Location.Range.Start.Line + line,
			Character: col,
		}
	}

	accessKW := map[string]bool{"public": true, "protected": true, "private": true, "virtual": true}
	isIdentStart := func(b byte) bool {
		return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
	}
	isIdentCont := func(b byte) bool {
		return isIdentStart(b) || (b >= '0' && b <= '9')
	}

	var out []BaseClassRef
	for _, sp := range specs {
		d := 0
		i := sp.s
		for i < sp.e {
			switch decl[i] {
			case '<':
				d++
				i++
				continue
			case '>':
				if d > 0 {
					d--
				}
				i++
				continue
			}
			if !isIdentStart(decl[i]) {
				i++
				continue
			}
			start := i
			for i < sp.e && isIdentCont(decl[i]) {
				i++
			}
			word := decl[start:i]
			if d != 0 {
				continue
			}
			if accessKW[word] {
				continue
			}
			// Follow `::Foo` segments. The recorded position must point at
			// the LAST name (the actual type — that's the token clangd
			// can Definition-resolve), but the recorded Name preserves
			// the FULL `ns::Sub::Foo` qualifier so unresolved fallbacks
			// keep the namespace distinction (without it, two distinct
			// bases like `third_party::Provider` and `other_pkg::Provider`
			// would collapse to the same bare "Provider").
			fullStart := start
			lastStart := start
			fullEnd := i
			for i+1 < sp.e && decl[i] == ':' && decl[i+1] == ':' {
				i += 2
				if i >= sp.e || !isIdentStart(decl[i]) {
					break
				}
				segStart := i
				for i < sp.e && isIdentCont(decl[i]) {
					i++
				}
				lastStart = segStart
				fullEnd = i
			}
			// Strip a leading "::" (root-namespace prefix) so the recorded
			// name matches what clangd / collect / export compare against.
			fullName := decl[fullStart:fullEnd]
			out = append(out, BaseClassRef{
				Name: fullName,
				Loc: lsp.Location{
					URI:   sym.Location.URI,
					Range: lsp.Range{Start: posOf(lastStart), End: posOf(lastStart)},
				},
			})
			break
		}
	}
	return out
}

func cppShortTypeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	// Drop namespace qualifiers
	if i := strings.LastIndex(name, "::"); i >= 0 {
		name = name[i+2:]
	}

	// Drop template args
	if i := strings.IndexByte(name, '<'); i >= 0 {
		name = name[:i]
	}

	// Drop leading keywords if they leak into Name (rare)
	name = strings.TrimPrefix(name, "class ")
	name = strings.TrimPrefix(name, "struct ")
	name = strings.TrimSpace(name)

	return name
}

func (c *CppSpec) GetUnloadedSymbol(from lsp.Token, define lsp.Location) (string, error) {
	return "", nil
}

func (c *CppSpec) FunctionSymbol(sym lsp.DocumentSymbol) (int, []int, []int, []int) {
	// C++: function or method (and sometimes qualified names are still SKFunction)
	if sym.Kind != lsp.SKFunction && sym.Kind != lsp.SKMethod && !strings.Contains(sym.Name, "::") {
		return -1, nil, nil, nil
	}

	receiver := -1
	typeParams := make([]int, 0, 4)
	inputParams := make([]int, 0, 8)
	outputs := make([]int, 0, 4)

	lines := utils.CountLinesPooled(sym.Text)

	// 1) type params
	for i, tok := range sym.Tokens {
		if tok.Type == tokTypeParameter {
			typeParams = append(typeParams, i)
		}
	}

	// 2) find name token (method/function)
	nameTokIdx := -1
	for i, tok := range sym.Tokens {
		if tok.Type == tokMethod || tok.Type == tokFunction {
			nameTokIdx = i
			break
		}
	}
	if nameTokIdx < 0 {
		return -1, typeParams, nil, nil
	}

	// 3) receiver: parse from qualified name "Person::SayHi" -> "Person"
	recvShort := receiverShortName(sym.Name)
	if recvShort != "" {
		for i := 0; i < nameTokIdx; i++ { // receiver must be before method name in signature
			tok := sym.Tokens[i]
			if tok.Text != recvShort {
				continue
			}
			// prefer type-ish token kinds for receiver
			if tok.Type == tokClass || tok.Type == tokStruct || tok.Type == tokType {
				receiver = i
				break
			}
		}
		if receiver < 0 {
			for i := 0; i < nameTokIdx; i++ {
				tok := sym.Tokens[i]
				if tok.Text == recvShort && c.IsEntityToken(tok) {
					receiver = i
					break
				}
			}
		}
	}

	nameOff := lsp.RelativePostionWithLines(*lines, sym.Location.Range.Start, sym.Tokens[nameTokIdx].Location.Range.Start)

	// 4) find params bounds
	paramL, paramR := -1, -1
	if nameOff >= 0 && nameOff < len(sym.Text) {
		open := strings.Index(sym.Text[nameOff:], "(")
		if open >= 0 {
			paramL = nameOff + open
			paramR = findMatchingParen(sym.Text, paramL)
		}
	}

	// 5) classify tokens
	for i, tok := range sym.Tokens {
		if !c.IsEntityToken(tok) {
			continue
		}
		if tok.Type == tokTypeParameter || tok.Type == tokNamespace {
			continue
		}

		off := lsp.RelativePostionWithLines(*lines, sym.Location.Range.Start, tok.Location.Range.Start)

		// inputs
		if paramL >= 0 && paramR >= 0 && off > paramL && off <= paramR {
			inputParams = append(inputParams, i)
			continue
		}

		// outputs: before name token, excluding receiver
		if off >= 0 && off < nameOff {
			if i == receiver {
				continue
			}
			outputs = append(outputs, i)
		}
	}

	return receiver, typeParams, inputParams, outputs
}

func receiverShortName(qualified string) string {
	parts := strings.Split(qualified, "::")
	if len(parts) < 2 {
		return ""
	}
	recv := parts[len(parts)-2]
	if i := strings.IndexByte(recv, '<'); i >= 0 { // Foo<T> -> Foo
		recv = recv[:i]
	}
	return strings.TrimSpace(recv)
}

func findMatchingParen(s string, openIdx int) int {
	if openIdx < 0 || openIdx >= len(s) || s[openIdx] != '(' {
		return -1
	}
	depth := 0
	for i := openIdx; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
