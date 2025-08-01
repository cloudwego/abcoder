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
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/cloudwego/abcoder/lang/cxx"
	"github.com/cloudwego/abcoder/lang/log"
	. "github.com/cloudwego/abcoder/lang/lsp"
	"github.com/cloudwego/abcoder/lang/python"
	"github.com/cloudwego/abcoder/lang/rust"
	"github.com/cloudwego/abcoder/lang/uniast"
)

type CollectOption struct {
	Language           uniast.Language
	LoadExternalSymbol bool
	NeedStdSymbol      bool
	NoNeedComment      bool
	NotNeedTest        bool
	Excludes           []string
	LoadByPackages     bool
	IDLPkgType         string
}

type Collector struct {
	cli  *LSPClient
	spec LanguageSpec

	repo string

	syms map[Location]*DocumentSymbol

	//  symbol => (receiver,impl,func)
	funcs map[*DocumentSymbol]functionInfo

	// 	symbol => [deps]
	deps map[*DocumentSymbol][]dependency

	// variable (or const) => type
	vars map[*DocumentSymbol]dependency

	files map[string]*uniast.File

	// modPatcher ModulePatcher

	CollectOption
}

type methodInfo struct {
	Receiver  dependency  `json:"receiver"`
	Interface *dependency `json:"implement,omitempty"` // which interface it implements
	ImplHead  string      `json:"implHead,omitempty"`
}

type functionInfo struct {
	Method           *methodInfo        `json:"method,omitempty"`
	TypeParams       map[int]dependency `json:"typeParams,omitempty"`
	TypeParamsSorted []dependency       `json:"-"`
	Inputs           map[int]dependency `json:"inputs,omitempty"`
	InputsSorted     []dependency       `json:"-"`
	Outputs          map[int]dependency `json:"outputs,omitempty"`
	OutputsSorted    []dependency       `json:"-"`
	Signature        string             `json:"signature,omitempty"`
}

func switchSpec(l uniast.Language) LanguageSpec {
	switch l {
	case uniast.Rust:
		return rust.NewRustSpec()
	case uniast.Cxx:
		return cxx.NewCxxSpec()
	case uniast.Python:
		return python.NewPythonSpec()
	default:
		panic(fmt.Sprintf("unsupported language %s", l))
	}
}

func NewCollector(repo string, cli *LSPClient) *Collector {
	ret := &Collector{
		repo:  repo,
		cli:   cli,
		spec:  switchSpec(cli.ClientOptions.Language),
		syms:  map[Location]*DocumentSymbol{},
		funcs: map[*DocumentSymbol]functionInfo{},
		deps:  map[*DocumentSymbol][]dependency{},
		vars:  map[*DocumentSymbol]dependency{},
		files: map[string]*uniast.File{},
	}
	// if cli.Language == uniast.Rust {
	// 	ret.modPatcher = &rust.RustModulePatcher{Root: repo}
	// }
	return ret
}

func (c *Collector) configureLSP(ctx context.Context) {
	// XXX: should be put in language specification
	if c.Language == uniast.Python {
		if !c.NeedStdSymbol {
			if c.Language == uniast.Python {
				conf := map[string]interface{}{
					"settings": map[string]interface{}{
						"pylsp": map[string]interface{}{
							"plugins": map[string]interface{}{
								"jedi_definition": map[string]interface{}{
									"follow_builtin_definitions": false,
								},
							},
						},
					},
				}
				c.cli.Notify(ctx, "workspace/didChangeConfiguration", conf)
			}
		}
	}
}

func (c *Collector) Collect(ctx context.Context) error {
	c.configureLSP(ctx)
	excludes := make([]string, len(c.Excludes))
	for i, e := range c.Excludes {
		if !filepath.IsAbs(e) {
			excludes[i] = filepath.Join(c.repo, e)
		} else {
			excludes[i] = e
		}
	}

	// scan all files
	root_syms := make([]*DocumentSymbol, 0, 1024)
	scanner := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		for _, e := range excludes {
			if strings.HasPrefix(path, e) {
				return nil
			}
		}

		if c.spec.ShouldSkip(path) {
			return nil
		}

		file := c.files[path]
		if file == nil {
			file = uniast.NewFile(path)
			c.files[path] = file
		}

		// 解析use语句
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		uses, err := c.spec.FileImports(content)
		if err != nil {
			log.Error("parse file %s use statements failed: %v", path, err)
		}
		file.Imports = uses

		// collect symbols
		uri := NewURI(path)
		symbols, err := c.cli.DocumentSymbols(ctx, uri)
		if err != nil {
			return err
		}
		// file := filepath.Base(path)
		for _, sym := range symbols {
			// collect content
			content, err := c.cli.Locate(sym.Location)
			if err != nil {
				return err
			}
			// collect tokens
			tokens, err := c.cli.SemanticTokens(ctx, sym.Location)
			if err != nil {
				return err
			}
			sym.Text = content
			sym.Tokens = tokens
			c.syms[sym.Location] = sym
			root_syms = append(root_syms, sym)
		}

		return nil
	}
	if err := filepath.Walk(c.repo, scanner); err != nil {
		return err
	}

	// collect some extra metadata
	entity_syms := make([]*DocumentSymbol, 0, len(root_syms))
	for _, sym := range root_syms {
		// only language entity symbols need to be collect on next
		if c.spec.IsEntitySymbol(*sym) {
			entity_syms = append(entity_syms, sym)
		}
		c.processSymbol(ctx, sym, 1)
	}

	// collect internal references
	// for _, sym := range syms {
	// 	i := c.spec.DeclareTokenOfSymbol(*sym)
	// 	if i < 0 {
	// 		log.Error("declare token of symbol %s failed\n", sym)
	// 		continue
	// 	}
	// 	refs, err := c.cli.References(ctx, sym.Tokens[i].Location)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	for _, rloc := range refs {
	// 		// remove child symbol
	// 		if sym.Location.Include(rloc) {
	// 			continue
	// 		}
	// 		rsym, err := c.getSymbolByLocation(ctx, rloc)
	// 		if err != nil || rsym == nil {
	// 			log.Error("symbol not found for location %v\n", rloc)
	// 			continue
	// 		}
	// 		// remove external or parent symbol
	// 		if !c.internal(rsym.Location) || rsym.Location.Include(sym.Location) {
	// 			continue
	// 		}
	// 		c.deps[rsym] = append(c.deps[rsym], Dependency{
	// 			Location: rloc,
	// 			Symbol:   sym,
	// 		})
	// 	}
	// }

	// collect dependencies
	for _, sym := range entity_syms {
	next_token:

		for i, token := range sym.Tokens {
			// only entity token need to be collect (std token is only collected when NeedStdSymbol is true)
			if !c.spec.IsEntityToken(token) {
				continue
			}

			// skip function's params
			if sym.Kind == SKFunction || sym.Kind == SKMethod {
				if finfo, ok := c.funcs[sym]; ok {
					if finfo.Method != nil {
						if finfo.Method.Receiver.Location.Include(token.Location) {
							continue next_token
						}
					}
					if finfo.Inputs != nil {
						if _, ok := finfo.Inputs[i]; ok {
							continue next_token
						}
					}
					if finfo.Outputs != nil {
						if _, ok := finfo.Outputs[i]; ok {
							continue next_token
						}
					}
					if finfo.TypeParams != nil {
						if _, ok := finfo.TypeParams[i]; ok {
							continue next_token
						}
					}
				}
			}
			// skip variable's type
			if sym.Kind == SKVariable || sym.Kind == SKConstant {
				if dep, ok := c.vars[sym]; ok {
					if dep.Location.Include(token.Location) {
						continue next_token
					}
				}
			}

			// go to definition
			dep, err := c.getSymbolByToken(ctx, token)
			if err != nil || dep == nil {
				log.Error("dep token %v not found: %v\n", token, err)
				continue
			}

			// NOTICE: some internal symbols may not been get by DocumentSymbols, thus we let Unknown symbol pass
			if dep.Kind == SKUnknown && c.internal(dep.Location) {
				// try get symbol kind by token
				sk := c.spec.TokenKind(token)
				if sk != SKUnknown {
					dep.Kind = sk
					dep.Name = token.Text
				}
			}

			// remove local symbols
			if sym.Location.Include(dep.Location) {
				continue
			} else {
				c.syms[dep.Location] = dep
			}

			c.deps[sym] = append(c.deps[sym], dependency{
				Location: token.Location,
				Symbol:   dep,
			})

		}
	}

	return nil
}

func (c *Collector) internal(loc Location) bool {
	return strings.HasPrefix(loc.URI.File(), c.repo)
}

func (c *Collector) getSymbolByToken(ctx context.Context, tok Token) (*DocumentSymbol, error) {
	return c.getSymbolByTokenWithLimit(ctx, tok, 1)
}

func (c *Collector) getSymbolByTokenWithLimit(ctx context.Context, tok Token, depth int) (*DocumentSymbol, error) {
	// get definition symbol
	defs, err := c.cli.Definition(ctx, tok.Location.URI, tok.Location.Range.Start)
	if err != nil {
		return nil, err
	}
	if len(defs) == 0 {
		return nil, fmt.Errorf("definition of token %s not found", tok)
	}
	if len(defs) > 1 {
		log.Error("definition of token %s not unique", tok)
	}
	return c.getSymbolByLocation(ctx, defs[0], depth, tok)
}

// Find the symbol (from the symbol list) that matches the location.
// It is the smallest (most specific) entity symbol that contains the location.
//
// Parameters:
//
//	@syms: the list of symbols to search in
//	@loc: the location to find the symbol for
//
// Returns:
//
//	*DocumentSymbol: the most specific entity symbol that contains the location.
//	If no such symbol is found, it returns nil.
func (c *Collector) findMatchingSymbolIn(loc Location, syms []*DocumentSymbol) *DocumentSymbol {
	var most_specific *DocumentSymbol
	for _, sym := range syms {
		if !sym.Location.Include(loc) || !c.spec.IsEntitySymbol(*sym) {
			continue
		}
		// now we have a candidate (containing loc && entity), check if it is the most specific
		if most_specific == nil {
			most_specific = sym
			continue
		}
		if most_specific.Location.Include(sym.Location) {
			// use sym, which is more specific than most_specific
			most_specific = sym
			continue
		}
		if sym.Location.Include(most_specific.Location) {
			// remain current choice
			continue
		}
		// Indicates a bad usage, sym contains unstructured symbols.
		log.Error("getMostSpecificEntitySymbol: cannot decide between symbols %s (at %+v) and %s (at %+v)\n",
			most_specific.Name, most_specific.Location,
			sym.Name, sym.Location)
	}
	return most_specific
}

// return a language entity symbol
//   - loaded: just return loaded symbol
//   - not loaded but set option LoadExternalSymbol: load external symbol and return
//   - otherwise: return a Unknown symbol
func (c *Collector) getSymbolByLocation(ctx context.Context, loc Location, depth int, from Token) (*DocumentSymbol, error) {
	// already loaded
	// if sym, ok := c.syms[loc]; ok {
	// 	return sym, nil
	// }

	// 1. already loaded
	if sym := c.findMatchingSymbolIn(loc, slices.Collect(maps.Values(c.syms))); sym != nil {
		return sym, nil
	}

	if c.LoadExternalSymbol && !c.internal(loc) && (c.NeedStdSymbol || !c.spec.IsStdToken(from)) {
		// 2. load external symbol from its file
		syms, err := c.cli.DocumentSymbols(ctx, loc.URI)
		if err != nil {
			return nil, err
		}
		// load the other external symbols in that file
		for _, sym := range syms {
			// save symbol first
			if _, ok := c.syms[sym.Location]; !ok {
				content, err := c.cli.Locate(sym.Location)
				if err != nil {
					return nil, err
				}
				sym.Text = content
				c.syms[sym.Location] = sym
			}
		}
		// load more external symbols if depth permits
		if depth >= 0 {
			// process target symbol
			for _, sym := range syms {
				// check if need process
				if c.needProcessExternal(sym) {
					// collect tokens before process
					tokens, err := c.cli.SemanticTokens(ctx, sym.Location)
					if err != nil {
						return nil, err
					}
					sym.Tokens = tokens
					c.processSymbol(ctx, sym, depth-1)
				}
			}
		}
		rsym := c.findMatchingSymbolIn(loc, slices.Collect(maps.Values(syms)))
		return rsym, nil
	} else {
		// external symbol, just locate the content
		var text string
		if c.internal(loc) {
			// maybe internal symbol not loaded, like `lazy_static!` in Rust
			// use the before and after symbol as text
			var left, right *DocumentSymbol
			syms, err := c.cli.DocumentSymbols(ctx, loc.URI)
			if err != nil {
				if c.cli.ClientOptions.Verbose {
					log.Error("locate %v failed: %v\n", loc, err)
				}
				goto finally
			}
			for _, sym := range syms {
				if sym.Location.Range.End.Less(loc.Range.Start) {
					if left == nil || left.Location.Range.End.Less(sym.Location.Range.End) {
						left = sym
					}
				}
				if loc.Range.End.Less(sym.Location.Range.Start) {
					if right == nil || sym.Location.Range.Start.Less(right.Location.Range.Start) {
						right = sym
					}
				}
			}
			if left == nil {
				left = &DocumentSymbol{
					Location: Location{
						URI: loc.URI,
						Range: Range{
							Start: Position{
								Line:      0,
								Character: 0,
							},
							End: Position{
								Line:      0,
								Character: 0,
							},
						},
					},
				}
			}
			if right == nil {
				lines := c.cli.LineCounts(loc.URI)
				right = &DocumentSymbol{
					Location: Location{
						URI: loc.URI,
						Range: Range{
							Start: Position{
								Line:      len(lines),
								Character: 1,
							},
							End: Position{
								Line:      len(lines),
								Character: 1,
							},
						},
					},
				}
			}
			var end int
			line := c.cli.Line(loc.URI, right.Location.Range.Start.Line-1)
			for i := 0; i < len(line); i++ {
				if unicode.IsSpace(rune(line[i])) {
					end = i
					break
				}
			}
			txt, err := c.cli.Locate(Location{
				URI: loc.URI,
				Range: Range{
					Start: Position{
						Line:      left.Location.Range.End.Line + 1,
						Character: 0,
					},
					End: Position{
						Line:      right.Location.Range.Start.Line - 1,
						Character: end,
					},
				},
			})
			if err != nil {
				if c.cli.ClientOptions.Verbose {
					log.Error("locate %v failed: %v\n", loc, err)
				}
				goto finally
			}
			text = txt
		}
	finally:
		if text == "" {
			txt, err := c.cli.Locate(loc)
			if err != nil {
				if c.cli.ClientOptions.Verbose {
					log.Error("locate %v failed: %v\n", loc, err)
				}
			}
			text = txt
		}
		// not loaded, make a fake Unknown symbol
		tmp := &DocumentSymbol{
			Name:     from.Text,
			Kind:     c.spec.TokenKind(from),
			Location: loc,
			Text:     text,
		}
		c.syms[loc] = tmp
		return tmp, nil
	}
}

func (c *Collector) getDepsWithLimit(ctx context.Context, sym *DocumentSymbol, tps []int, depth int) (map[int]dependency, []dependency) {
	var tsyms = make(map[int]dependency, len(tps))
	var sorted = make([]dependency, 0, len(tps))
	for _, tp := range tps {
		dep, err := c.getSymbolByTokenWithLimit(ctx, sym.Tokens[tp], depth)
		if err != nil || sym == nil {
			log.Error_skip(1, "token %v not found its symbol: %v", tp, err)
		} else {
			d := dependency{sym.Tokens[tp].Location, dep}
			tsyms[tp] = d
			sorted = append(sorted, d)
		}
	}
	return tsyms, sorted
}

func (c *Collector) collectImpl(ctx context.Context, sym *DocumentSymbol, depth int) {
	// method info: receiver, implementee
	inter, rec, fn := c.spec.ImplSymbol(*sym)
	if rec < 0 {
		return
	}
	var rd, ind *dependency
	var err error
	rsym, err := c.getSymbolByTokenWithLimit(ctx, sym.Tokens[rec], depth)
	if err != nil || rsym == nil {
		log.Error("get receiver symbol for token %v failed: %v\n", rec, err)
		return
	}
	rd = &dependency{sym.Tokens[rec].Location, rsym}
	if inter >= 0 {
		isym, err := c.getSymbolByToken(ctx, sym.Tokens[inter])
		if err != nil || isym == nil {
			log.Error("get implement symbol for token %v failed: %v\n", inter, err)
		} else {
			ind = &dependency{sym.Tokens[inter].Location, isym}
		}
	}
	var impl string
	// HACK: impl head for Rust.
	if fn > 0 && fn < len(sym.Tokens) {
		impl = ChunkHead(sym.Text, sym.Location.Range.Start, sym.Tokens[fn].Location.Range.Start)
	}
	// HACK: implhead for Python. Should actually be provided by the language spec.
	if impl == "" || len(impl) < len(sym.Name) {
		impl = fmt.Sprintf("class %s {\n", sym.Name)
	}
	// search all methods
	for _, method := range c.syms {
		// NOTICE: some class method (ex: XXType::new) are SKFunction, but still collect its receiver
		if (method.Kind == SKMethod || method.Kind == SKFunction) && sym.Location.Include(method.Location) {
			if _, ok := c.funcs[method]; !ok {
				c.funcs[method] = functionInfo{}
			}
			f := c.funcs[method]
			f.Method = &methodInfo{
				Receiver:  *rd,
				Interface: ind,
				ImplHead:  impl,
			}
			c.funcs[method] = f
		}
	}
}

func (c *Collector) needProcessExternal(sym *DocumentSymbol) bool {
	return (c.spec.HasImplSymbol() && sym.Kind == SKObject) || (!c.spec.HasImplSymbol() && sym.Kind == SKMethod)
}

func (c *Collector) processSymbol(ctx context.Context, sym *DocumentSymbol, depth int) {
	// method info: receiver, implementee
	hasImpl := c.spec.HasImplSymbol()
	if hasImpl {
		c.collectImpl(ctx, sym, depth)
	}

	// function info: type params, inputs, outputs, receiver (if !needImpl)
	if sym.Kind == SKFunction || sym.Kind == SKMethod {
		var rsym *dependency
		rec, tps, ips, ops := c.spec.FunctionSymbol(*sym)

		if !hasImpl && rec >= 0 {
			rsym, err := c.getSymbolByTokenWithLimit(ctx, sym.Tokens[rec], depth)
			if err != nil || rsym == nil {
				log.Error("get receiver symbol for token %v failed: %v\n", rec, err)
			}
		}
		tsyms, ts := c.getDepsWithLimit(ctx, sym, tps, depth-1)
		ipsyms, is := c.getDepsWithLimit(ctx, sym, ips, depth-1)
		opsyms, os := c.getDepsWithLimit(ctx, sym, ops, depth-1)

		//get last token of params for get signature
		lastToken := rec
		for _, t := range tps {
			if t > lastToken {
				lastToken = t
			}
		}
		for _, t := range ips {
			if t > lastToken {
				lastToken = t
			}
		}
		for _, t := range ops {
			if t > lastToken {
				lastToken = t
			}
		}

		c.updateFunctionInfo(sym, tsyms, ipsyms, opsyms, ts, is, os, rsym, lastToken)
	}

	// variable info: type
	if sym.Kind == SKVariable || sym.Kind == SKConstant {
		i := c.spec.DeclareTokenOfSymbol(*sym)
		// find first entity token
		for i = i + 1; i < len(sym.Tokens); i++ {
			if c.spec.IsEntityToken(sym.Tokens[i]) {
				break
			}
		}
		if i < 0 || i >= len(sym.Tokens) {
			log.Error("get type token of variable symbol %s failed\n", sym)
			return
		}
		tsym, err := c.getSymbolByTokenWithLimit(ctx, sym.Tokens[i], depth-1)
		if err != nil || tsym == nil {
			log.Error("get type symbol for token %s failed:%v\n", sym.Tokens[i], err)
			return
		}
		c.vars[sym] = dependency{
			Location: sym.Tokens[i].Location,
			Symbol:   tsym,
		}
	}
}

func (c *Collector) updateFunctionInfo(sym *DocumentSymbol, tsyms, ipsyms, opsyms map[int]dependency, ts, is, os []dependency, rsym *dependency, lastToken int) {
	if _, ok := c.funcs[sym]; !ok {
		c.funcs[sym] = functionInfo{}
	}
	f := c.funcs[sym]
	f.TypeParams = tsyms
	f.TypeParamsSorted = ts
	f.Inputs = ipsyms
	f.InputsSorted = is
	f.Outputs = opsyms
	f.OutputsSorted = os
	if rsym != nil {
		if f.Method == nil {
			f.Method = &methodInfo{}
		}
		f.Method.Receiver = *rsym
	}

	// ctruncate the function signature text
	if lastToken >= 0 && lastToken < len(sym.Tokens)-1 {
		lastPos := sym.Tokens[lastToken+1].Location.Range.Start
		f.Signature = ChunkHead(sym.Text, sym.Location.Range.Start, lastPos)
	}

	c.funcs[sym] = f
}
