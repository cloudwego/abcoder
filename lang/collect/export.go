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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/lang/lsp"
	. "github.com/cloudwego/abcoder/lang/lsp"
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/cloudwego/abcoder/lang/utils"
)

type dependency struct {
	Location Location        `json:"location"`
	Symbol   *DocumentSymbol `json:"symbol"`
}

func (c *Collector) fileLine(loc Location) uniast.FileLine {
	var rel string
	if c.internal(loc) {
		rel, _ = filepath.Rel(c.repo, loc.URI.File())
	} else {
		rel = filepath.Base(loc.URI.File())
	}
	fileURI := string(loc.URI)
	filePath := loc.URI.File()

	text := ""
	// 1. Try LSP client files
	if c.cli != nil {
		if f := c.cli.GetFile(loc.URI); f != nil {
			text = f.Text
		}
	}

	// 2. Try internal cache
	if text == "" {
		if cached, ok := c.fileContentCache[filePath]; ok {
			text = cached
		}
	}

	// 3. Fallback to OS ReadFile and update cache
	if text == "" {
		fd, err := os.ReadFile(filePath)
		if err != nil {
			return uniast.FileLine{File: rel, Line: loc.Range.Start.Line + 1}
		}
		text = string(fd)
		c.fileContentCache[filePath] = text
	}

	return uniast.FileLine{
		File:        rel,
		Line:        loc.Range.Start.Line + 1,
		StartOffset: PositionOffset(fileURI, text, loc.Range.Start),
		EndOffset:   PositionOffset(fileURI, text, loc.Range.End),
	}
}

func newModule(name string, dir string, lang uniast.Language) *uniast.Module {
	ret := uniast.NewModule(name, dir, lang)
	return ret
}

func (c *Collector) ExportLocalFunction() map[Location]*DocumentSymbol {
	if len(c.localFunc) == 0 {
		c.localFunc = make(map[Location]*DocumentSymbol)
		for symbol := range c.funcs {
			c.localFunc[symbol.Location] = symbol
		}
	}
	return c.localFunc
}

func (c *Collector) Export(ctx context.Context) (*uniast.Repository, error) {
	// recursively read all go files in repo
	repo := uniast.NewRepository(c.repo)
	modules, err := c.spec.WorkSpace(c.repo)
	if err != nil {
		return nil, err
	}

	// set modules on repo
	for name, path := range modules {
		rel, err := filepath.Rel(c.repo, path)
		if err != nil {
			return nil, err
		}
		repo.Modules[name] = newModule(name, rel, c.Language)
	}

	// not allow local symbols inside another symbol
	log.Info("Export: filtering local symbols...\n")

	//c.filterLocalSymbols()
	c.filterLocalSymbolsByCache()

	// Pre-compute receivers map to avoid O(N^2) complexity in exportSymbol recursion
	log.Info("Export: pre-computing receivers map...\n")
	c.receivers = make(map[*DocumentSymbol][]*DocumentSymbol, len(c.funcs)/4)
	for method, rec := range c.funcs {
		if (method.Kind == SKMethod) && rec.Method != nil && rec.Method.Receiver.Symbol != nil {
			c.receivers[rec.Method.Receiver.Symbol] = append(c.receivers[rec.Method.Receiver.Symbol], method)
		}

		if (method.Kind == SKFunction && c.Language == uniast.Java) && rec.Method != nil && rec.Method.Receiver.Symbol != nil {
			c.receivers[rec.Method.Receiver.Symbol] = append(c.receivers[rec.Method.Receiver.Symbol], method)
		}
	}

	// export symbols
	log.Info("Export: exporting %d symbols...\n", len(c.syms))
	visited := make(map[*DocumentSymbol]*uniast.Identity)
	for _, symbol := range c.syms {
		_, _ = c.exportSymbol(&repo, symbol, "", visited)
	}

	log.Info("Export: connecting files to packages...\n")
	for fp, f := range c.files {
		rel, err := filepath.Rel(c.repo, fp)
		if err != nil {
			continue
		}

		modpath, pkgpath, err := c.spec.NameSpace(fp, f)
		if err != nil {
			continue
		}

		// connect file to package
		if modpath == "" || strings.Contains(modpath, "@") {
			continue
		}
		m, ok := repo.Modules[modpath]
		if !ok {
			continue
		}

		m.Files[rel] = f
		if pkgpath == "" || f.Package != "" {
			continue
		}
		if _, ok := m.Packages[pkgpath]; !ok {
			continue
		}
		f.Package = pkgpath
	}

	return &repo, nil
}

var (
	ErrStdSymbol      = errors.New("std symbol")
	ErrExternalSymbol = errors.New("external symbol")
)

// NOTICE: for rust and golang, each entity has separate location
// TODO: some language may allow local symbols inside another symbol,
func (c *Collector) filterLocalSymbols() {
	// filter symbols
	for loc1 := range c.syms {
		for loc2 := range c.syms {
			if loc1 == loc2 {
				continue
			}
			if loc2.Include(loc1) {
				if utils.Contains(c.spec.ProtectedSymbolKinds(), c.syms[loc1].Kind) {
					break
				}
				delete(c.syms, loc1)
				break
			}
		}
	}
}

func (c *Collector) filterLocalSymbolsByCache() {
	if len(c.syms) == 0 {
		return
	}

	// Group symbols by file URI to reduce comparison scope
	symsByFile := make(map[DocumentURI][]*DocumentSymbol)
	for loc, sym := range c.syms {
		symsByFile[loc.URI] = append(symsByFile[loc.URI], sym)
	}

	for _, fileSyms := range symsByFile {
		if len(fileSyms) <= 1 {
			continue
		}

		// Sort symbols in the same file:
		// 1. By start offset (ascending)
		// 2. By end offset (descending) - larger range first
		// This ensures that if symbol A contains symbol B, A appears before B.
		sort.Slice(fileSyms, func(i, j int) bool {
			locI, locJ := fileSyms[i].Location, fileSyms[j].Location
			if locI.Range.Start.Line != locJ.Range.Start.Line {
				return locI.Range.Start.Line < locJ.Range.Start.Line
			}
			if locI.Range.Start.Character != locJ.Range.Start.Character {
				return locI.Range.Start.Character < locJ.Range.Start.Character
			}
			if locI.Range.End.Line != locJ.Range.End.Line {
				return locI.Range.End.Line > locJ.Range.End.Line
			}
			return locI.Range.End.Character > locJ.Range.End.Character
		})

		// Use a stack-like approach or simple active parent tracking
		// Since we sorted by start ASC and end DESC, a candidate parent always comes first.
		var activeParents []*DocumentSymbol
		for _, sym := range fileSyms {
			isNested := false
			// Check if current symbol is nested within any of the active parents
			// We only need to check the most recent ones that could still contain it
			for i := len(activeParents) - 1; i >= 0; i-- {
				parent := activeParents[i]
				if parent.Location.Include(sym.Location) {
					if !utils.Contains(c.spec.ProtectedSymbolKinds(), sym.Kind) {
						isNested = true
						break
					}
				} else if parent.Location.Range.End.Less(sym.Location.Range.Start) {
					// This parent can no longer contain any future symbols (since we're sorted by start)
					// But we don't necessarily need to remove it from the slice here for correctness.
				}
			}

			if isNested {
				delete(c.syms, sym.Location)
			} else {
				activeParents = append(activeParents, sym)
			}
		}
	}
}

func (c *Collector) exportSymbol(repo *uniast.Repository, symbol *DocumentSymbol, refName string, visited map[*DocumentSymbol]*uniast.Identity) (id *uniast.Identity, e error) {
	defer func() {
		if e != nil && e != ErrStdSymbol && e != ErrExternalSymbol {
			log.Info("export symbol %s failed: %v\n", symbol, e)
		}
	}()

	if symbol == nil {
		e = errors.New("symbol is nil")
		return
	}

	// 判断是否为本地符号
	// 只有符号是“定义”，或者符号是“本地方法”时，才需要完整导出
	// 其他情况（如外部引用、或对本地非顶层符号的引用）都只导出标识符
	isDefinition := symbol.Role == DEFINITION
	_, isLocalMethod := c.funcs[symbol]
	_, isLocalSymbol := c.syms[symbol.Location]
	if !isDefinition {
		if isLocalSymbol {
			//引用类型符号，把引用类型符号替换为local 符号
			symbol = c.syms[symbol.Location]
		} else {
			if symbol.Kind == SKFunction || symbol.Kind == SKMethod {
				documentSymbol := c.ExportLocalFunction()[symbol.Location]
				if documentSymbol != nil {
					symbol = documentSymbol
				}
			}

		}
	}

	if id, ok := visited[symbol]; ok {
		return id, nil
	}

	// Check NeedStdSymbol
	file := symbol.Location.URI.File()
	mod, path, err := c.spec.NameSpace(file, c.files[file])
	if err != nil {
		e = err
		return
	}

	//// Java IPC mode: external/JDK/third-party symbols
	//// For external symbols, we set the module and continue with normal export flow
	isJavaIPC := c.Language == uniast.Java && c.javaIPC != nil

	if isJavaIPC && !c.internal(symbol.Location) {
		// Determine module name based on URI path
		fp := symbol.Location.URI.File()
		if strings.Contains(fp, "abcoder-jdk") {
			mod = "jdk"
		} else if strings.Contains(fp, "abcoder-unknown") {
			mod = "unknown"
		}
	}
	if !c.NeedStdSymbol && mod == "" {
		e = ErrStdSymbol
		return
	}

	// Load external symbol on demands
	if !c.LoadExternalSymbol && (!c.internal(symbol.Location) || symbol.Kind == SKUnknown) {
		e = ErrExternalSymbol
		return
	}

	// Construct Identity and save to visited
	name := symbol.Name
	if name == "" {
		if refName == "" {
			e = fmt.Errorf("both symbol %v name and refname is empty", symbol)
			return
		}
		// NOTICE: use refName as id when symbol name is missing
		name = refName
	}

	if c.Language == uniast.Cpp {
		// for function override, use call signature as id
		if symbol.Kind == SKMethod || symbol.Kind == SKFunction {
			name = c.extractCppCallSig(symbol)
		}

		// join name with namespace
		if ns := c.scopePrefix(symbol); ns != "" {
			if !strings.HasPrefix(name, ns+"::") {
				name = ns + "::" + name
			}
		}
	}

	tmp := uniast.NewIdentity(mod, path, name)
	id = &tmp
	// Save to visited ONLY WHEN no errors occur
	visited[symbol] = id

	// Walk down from repo struct
	if repo.Modules[mod] == nil {
		repo.Modules[mod] = newModule(mod, "", c.Language)
	}
	module := repo.Modules[mod]
	if module.Packages[path] == nil {
		module.Packages[path] = uniast.NewPackage(path)
	}
	pkg := repo.Modules[mod].Packages[path]
	if c.spec.IsMainFunction(*symbol) {
		pkg.IsMain = true
	}

	fileLine := c.fileLine(symbol.Location)

	content := symbol.Text
	public := c.spec.IsPublicSymbol(*symbol)

	if !isDefinition && !isLocalMethod && !isLocalSymbol {
		// In Java IPC mode we never rely on LSP Definition.
		if !isJavaIPC {
			if c.cli == nil {
				return id, fmt.Errorf("LSP client is nil")
			}
			defs, err := c.cli.Definition(context.Background(), symbol.Location.URI, symbol.Location.Range.Start)
			if err != nil || len(defs) == 0 {
				// 意味着引用为外部符号，LSP 无法查询到符号定位
				return id, err
			}
		}
	}

	// map receiver to methods
	// Using pre-computed receivers map from c.receivers
	receivers := c.receivers

	switch k := symbol.Kind; k {
	// Function
	case SKFunction, SKMethod:
		if c.cli != nil {
			if p := c.cli.GetParent(symbol); p != nil && p.Kind == SKInterface {
				// NOTICE: no need collect interface method
				break
			}
		}
		obj := &uniast.Function{
			FileLine: fileLine,
			Content:  content,
			Exported: public,
		}
		info := c.funcs[symbol]
		obj.Signature = info.Signature
		// NOTICE: type parames collect into types
		if info.TypeParams != nil {
			for _, input := range info.TypeParamsSorted {
				tok := ""
				if c.cli != nil {
					tok, _ = c.cli.Locate(input.Location)
				}
				tyid, err := c.exportSymbol(repo, input.Symbol, tok, visited)
				if err != nil {
					continue
				}
				dep := uniast.NewDependency(*tyid, c.fileLine(input.Location))
				obj.Types = uniast.InsertDependency(obj.Types, dep)
			}
		}
		if info.Inputs != nil {
			for _, input := range info.InputsSorted {
				tok := ""
				if c.cli != nil {
					tok, _ = c.cli.Locate(input.Location)
				}
				tyid, err := c.exportSymbol(repo, input.Symbol, tok, visited)
				if err != nil {
					continue
				}
				dep := uniast.NewDependency(*tyid, c.fileLine(input.Location))
				obj.Params = uniast.InsertDependency(obj.Params, dep)
			}
		}
		if info.Outputs != nil {
			for _, output := range info.OutputsSorted {
				tok := ""
				if c.cli != nil {
					tok, _ = c.cli.Locate(output.Location)
				}
				tyid, err := c.exportSymbol(repo, output.Symbol, tok, visited)
				if err != nil {
					continue
				}
				dep := uniast.NewDependency(*tyid, c.fileLine(output.Location))
				obj.Results = uniast.InsertDependency(obj.Results, dep)
			}
		}
		if info.Method != nil && info.Method.Receiver.Symbol != nil {
			tok := ""
			if c.cli != nil {
				tok, _ = c.cli.Locate(info.Method.Receiver.Location)
			}
			rid, err := c.exportSymbol(repo, info.Method.Receiver.Symbol, tok, visited)
			if err == nil {
				obj.Receiver = &uniast.Receiver{
					Type: *rid,
					// Name: rid.Name,
				}
				obj.IsMethod = true
				id.Name = rid.Name
				// NOTICE: check if the method is a trait method
				// if true, type = trait<receiver>
				if info.Method.Interface != nil {
					itok := ""
					if c.cli != nil {
						itok, _ = c.cli.Locate(info.Method.Interface.Location)
					}
					iid, err := c.exportSymbol(repo, info.Method.Interface.Symbol, itok, visited)
					if err == nil {
						id.Name = iid.Name + "<" + id.Name + ">"
					}
				}

				// cpp get method name without class name
				if c.Language == uniast.Cpp && rid != nil {
					rec := strings.TrimSpace(rid.Name)
					if rec != "" {
						searchStr := rec + "::"
						if idx := strings.Index(name, searchStr); idx >= 0 {
							name = name[idx+len(searchStr):]
						}
					}
				}

				if k == SKFunction {
					// NOTICE: class static method name is: type::method
					id.Name += "::" + name
				} else {
					// NOTICE: object method name is: type.method
					id.Name += "." + name
				}
				// NOTICE: keep impl codes to the content
				if info.Method.ImplHead != "" {
					obj.Content = info.Method.ImplHead + obj.Content + "\n}"
				}
			}
		}
		// collect deps
		if deps := c.deps[symbol]; deps != nil {
			for _, dep := range deps {
				tok := ""
				if c.cli != nil {
					tok, _ = c.cli.Locate(dep.Location)
				}
				depid, err := c.exportSymbol(repo, dep.Symbol, tok, visited)
				if err != nil {
					continue
				}
				pdep := uniast.NewDependency(*depid, c.fileLine(dep.Location))
				switch dep.Symbol.Kind {
				case SKFunction:
					obj.FunctionCalls = uniast.InsertDependency(obj.FunctionCalls, pdep)
				case SKMethod:
					if obj.MethodCalls == nil {
						obj.MethodCalls = make([]uniast.Dependency, 0, len(deps))
					}
					// NOTICE: use loc token as key here, to make it more readable
					obj.MethodCalls = uniast.InsertDependency(obj.MethodCalls, pdep)
				case SKVariable, SKConstant:
					if obj.GlobalVars == nil {
						obj.GlobalVars = make([]uniast.Dependency, 0, len(deps))
					}
					obj.GlobalVars = uniast.InsertDependency(obj.GlobalVars, pdep)
				case SKStruct, SKTypeParameter, SKInterface, SKEnum, SKClass:
					if obj.Types == nil {
						obj.Types = make([]uniast.Dependency, 0, len(deps))
					}
					obj.Types = uniast.InsertDependency(obj.Types, pdep)
				default:
					log.Error("dep symbol %s not collected for %v\n", dep.Symbol, id)
				}
			}
		}
		obj.Identity = *id
		pkg.Functions[id.Name] = obj

	// Type
	case SKStruct, SKTypeParameter, SKInterface, SKEnum, SKClass:
		obj := &uniast.Type{
			FileLine: fileLine,
			Content:  content,
			TypeKind: mapKind(k),
			Exported: public,
		}
		// collect deps
		// collect deps
		if deps := c.deps[symbol]; deps != nil {
			for _, dep := range deps {
				tok := ""
				if c.cli != nil {
					tok, _ = c.cli.Locate(dep.Location)
				}
				depid, err := c.exportSymbol(repo, dep.Symbol, tok, visited)
				if err != nil {
					continue
				}
				switch dep.Symbol.Kind {
				case SKStruct, SKTypeParameter, SKInterface, SKEnum, SKClass:
					obj.SubStruct = uniast.InsertDependency(obj.SubStruct, uniast.NewDependency(*depid, c.fileLine(dep.Location)))
				case SKConstant, SKVariable:
				default:
					log.Error("dep symbol %s not collected for \n", dep.Symbol, id)
				}
			}
		}
		// collect methods
		if rec := receivers[symbol]; rec != nil {
			obj.Methods = make(map[string]uniast.Identity, len(rec))
			for _, method := range rec {
				tok := ""
				if c.cli != nil {
					tok, _ = c.cli.Locate(method.Location)
				}
				mid, err := c.exportSymbol(repo, method, tok, visited)
				if err != nil {
					continue
				}
				// NOTICE: use method name as key here
				obj.Methods[method.Name] = *mid
			}
		}
		obj.Identity = *id
		pkg.Types[id.Name] = obj
	// Vars
	case SKConstant, SKVariable:
		obj := &uniast.Var{
			FileLine:   fileLine,
			Content:    content,
			IsExported: public,
			IsConst:    k == SKConstant,
		}
		if ty, ok := c.vars[symbol]; ok {
			tok := ""
			if c.cli != nil {
				tok, _ = c.cli.Locate(ty.Location)
			}
			tyid, err := c.exportSymbol(repo, ty.Symbol, tok, visited)
			if err == nil {
				obj.Type = tyid
			}
		}
		obj.Identity = *id
		pkg.Vars[id.Name] = obj
	default:
		log.Error("symbol %s not collected\n", symbol)
	}

	return
}

func mapKind(kind SymbolKind) uniast.TypeKind {
	switch kind {
	case SKStruct:
		return "struct"
	// XXX: C++ should use class instead of struct
	case SKClass:
		return "struct"
	case SKTypeParameter:
		return "type-parameter"
	case SKInterface:
		return "interface"
	case SKEnum:
		return "enum"
	default:
		panic(fmt.Sprintf("unexpected kind %v", kind))
	}
}

func (c *Collector) scopePrefix(sym *DocumentSymbol) string {
	parts := []string{}
	cur := sym
	for {
		p := c.cli.GetParent(cur)
		if p == nil {
			break
		}
		if p.Kind == lsp.SKNamespace {
			if p.Name != "" {
				parts = append([]string{p.Name}, parts...)
			}
		}
		cur = p
	}
	return strings.Join(parts, "::") // "a::b"
}

func (c *Collector) cppBaseName(n string) string {
	n = strings.TrimSpace(n)
	if i := strings.LastIndex(n, "::"); i >= 0 {
		n = n[i+2:]
	}
	n = strings.TrimSpace(n)
	// optional: strip template args on the function name itself: foo<T> -> foo
	if j := strings.IndexByte(n, '<'); j >= 0 {
		n = n[:j]
	}
	return strings.TrimSpace(n)
}

// extractCppCallSig returns "sym.Name(params)" where params is extracted from sym.Text.
func (c *Collector) extractCppCallSig(sym *lsp.DocumentSymbol) (ret string) {
	name := strings.TrimSpace(sym.Name)
	if name == "" {
		return ""
	}
	text := sym.Text
	if text == "" {
		return name + "()"
	}

	want := c.cppBaseName(name)
	if want == "" {
		want = name
	}
	fallback := name + "()"

	isIdent := func(b byte) bool {
		return (b >= 'a' && b <= 'z') ||
			(b >= 'A' && b <= 'Z') ||
			(b >= '0' && b <= '9') ||
			b == '_'
	}
	isWholeIdentAt := func(s string, pos int, w string) bool {
		if pos < 0 || pos+len(w) > len(s) || s[pos:pos+len(w)] != w {
			return false
		}
		if pos > 0 && isIdent(s[pos-1]) {
			return false
		}
		if pos+len(w) < len(s) && isIdent(s[pos+len(w)]) {
			return false
		}
		return true
	}
	findMatchingParenIn := func(s string, openIdx int, end int) int {
		if openIdx < 0 || openIdx >= len(s) || s[openIdx] != '(' {
			return -1
		}
		if end > len(s) {
			end = len(s)
		}
		depth := 0
		for i := openIdx; i < end; i++ {
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

	headerEnd := len(text)
	if i := strings.IndexByte(text, '{'); i >= 0 && i < headerEnd {
		headerEnd = i
	}
	if i := strings.IndexByte(text, ';'); i >= 0 && i < headerEnd {
		headerEnd = i
	}
	header := text[:headerEnd]

	namePos := -1
	for i := 0; i+len(want) <= len(header); i++ {
		if isWholeIdentAt(header, i, want) {
			namePos = i
			break
		}
	}
	if namePos < 0 {
		return fallback
	}

	openIdx := -1
	for i := namePos + len(want); i < len(header); i++ {
		if header[i] == '(' {
			openIdx = i
			break
		}
	}
	if openIdx < 0 {
		return fallback
	}

	closeIdx := findMatchingParenIn(header, openIdx, len(header))
	if closeIdx < 0 {
		return fallback
	}

	return name + header[openIdx:closeIdx+1]
}
