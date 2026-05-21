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

// lightIdentityForExternal builds a Identity for an external symbol we don't
// want to fully export (no Type/Function entry produced) but still want to
// reference from the current symbol's Implements / similar edge fields.
// Returns nil if not enough info is available.
func (c *Collector) lightIdentityForExternal(sym *DocumentSymbol) *uniast.Identity {
	if sym == nil || sym.Name == "" {
		return nil
	}
	file := sym.Location.URI.File()
	mod, pkg, err := c.spec.NameSpace(file, nil)
	if err != nil || mod == "" {
		return nil
	}
	name := sym.Name
	// For C++ classes, the short name from clangd doesn't include namespace;
	// best-effort prepend the lexical scope from sibling symbols. Skip if it
	// already qualifies.
	if c.Language == uniast.Cpp {
		name = applyCppScopePrefix(c.scopePrefix(sym), name)
	}
	id := uniast.NewIdentity(mod, pkg, name)
	return &id
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

// dedupCppFunction handles header-decl-vs-cpp-def collapsing for a single
// emitted Function. Returns true when the caller should drop `obj` (the
// existing entry already won, possibly with body fields copied over from
// `obj`). Returns false when `obj` should be written into pkg.Functions
// as the canonical version.
func (c *Collector) dedupCppFunction(repo *uniast.Repository, symbol *DocumentSymbol, name, mod, path, content string, obj *uniast.Function) bool {
	// Decide hasBody from clangd's AST (presence of CompoundStmt under a
	// FunctionDecl). When the server doesn't support textDocument/ast we
	// can't safely distinguish definition from declaration; default to
	// `false` (treat as declaration) so the .cpp body still wins via the
	// "no prev body" branch when a .cpp emit follows the .h decl.
	hasBody := false
	if c.exportCtx != nil {
		if v, ok := c.cppASTHasBody(c.exportCtx, symbol); ok {
			hasBody = v
		}
	}
	isHeader := isCppHeaderPkg(path)
	cur := cppFnLoc{mod: mod, pkg: path, hasBody: hasBody, isHeader: isHeader}
	prev, hasPrev := c.cppFnEmitted[name]
	if !hasPrev {
		c.cppFnEmitted[name] = cur
		return false
	}
	prevPkg := func() *uniast.Package {
		if pm := repo.Modules[prev.mod]; pm != nil {
			return pm.Packages[prev.pkg]
		}
		return nil
	}
	switch {
	case prev.hasBody && !hasBody && prev.isHeader:
		return true // header already has body
	case prev.hasBody && !hasBody && !prev.isHeader:
		// Move .cpp body into the .h entry we're about to emit.
		if pp := prevPkg(); pp != nil {
			if src, ok := pp.Functions[name]; ok {
				copyFnBodyFields(obj, src)
				delete(pp.Functions, name)
			}
		}
	case hasBody && !prev.hasBody && prev.isHeader:
		// .h decl already emitted — copy our body fields into it.
		if pp := prevPkg(); pp != nil {
			if hdr, ok := pp.Functions[name]; ok {
				copyFnBodyFields(hdr, obj)
				prev.hasBody = true
				c.cppFnEmitted[name] = prev
			}
		}
		return true
	case hasBody && !prev.hasBody && !prev.isHeader:
		if pp := prevPkg(); pp != nil {
			delete(pp.Functions, name)
		}
	default:
		// Same body status — prefer the header.
		if prev.isHeader && !isHeader {
			return true
		}
		if isHeader && !prev.isHeader {
			if pp := prevPkg(); pp != nil {
				delete(pp.Functions, name)
			}
			break
		}
		if prev.mod == mod && prev.pkg == path {
			return true
		}
	}
	c.cppFnEmitted[name] = cur
	return false
}

// copyFnBodyFields copies the body-derived fields of src onto dst,
// leaving dst's identity / receiver / location-anchoring intact. Used by
// dedupCppFunction to merge a .cpp definition into a .h declaration.
func copyFnBodyFields(dst, src *uniast.Function) {
	dst.Content = src.Content
	dst.FunctionCalls = src.FunctionCalls
	dst.MethodCalls = src.MethodCalls
	dst.GlobalVars = src.GlobalVars
	dst.Types = src.Types
	dst.Params = src.Params
	dst.Results = src.Results
	dst.Signature = src.Signature
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
	// Stash ctx so recursive exportSymbol-and-friends can issue LSP
	// requests (e.g. textDocument/ast for cpp kind classification)
	// without threading a new parameter through every helper.
	c.exportCtx = ctx
	defer func() { c.exportCtx = nil }()
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

	// External C++ methods skip processSymbol (gated by needProcessExternal)
	// so c.funcs has no entry and Type.Methods stays empty. Recover the
	// receiver from the documentSymbol parent.
	if c.Language == uniast.Cpp && c.cli != nil {
		for _, sym := range c.syms {
			if sym.Kind != SKMethod {
				continue
			}
			if fi, ok := c.funcs[sym]; ok && fi.Method != nil && fi.Method.Receiver.Symbol != nil {
				continue
			}
			p := c.cli.GetParent(sym)
			if p == nil || (p.Kind != SKClass && p.Kind != SKStruct && p.Kind != SKInterface) {
				continue
			}
			c.receivers[p] = append(c.receivers[p], sym)
			fi := c.funcs[sym]
			if fi.Method == nil {
				fi.Method = &methodInfo{}
			}
			fi.Method.Receiver = dependency{Location: p.Location, Symbol: p}
			c.funcs[sym] = fi
		}
	}

	log.Info("Export: exporting %d symbols...\n", len(c.syms))
	visited := make(map[*DocumentSymbol]*uniast.Identity)
	for _, symbol := range c.syms {
		_, _ = c.exportSymbol(&repo, symbol, "", visited)
	}

	// Synthesize inherited methods per derived class so the call graph can
	// be walked through NVI/virtual dispatch. Outgoing this-edges in the
	// synthesized body are devirtualized to derived overrides where present.
	if c.Language == uniast.Cpp {
		log.Info("Export: synthesizing inherited C++ methods...\n")
		c.synthesizeInheritedMethodsCpp(&repo)
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

	// Drop packages that ended up empty after dedup / method-relocation.
	// For C++ this commonly happens to .cpp packages whose only entries
	// were method definitions relocated into their .h owner package.
	for _, m := range repo.Modules {
		if m == nil {
			continue
		}
		for pkgPath, pkg := range m.Packages {
			if pkg == nil {
				delete(m.Packages, pkgPath)
				continue
			}
			if len(pkg.Functions) == 0 && len(pkg.Types) == 0 && len(pkg.Vars) == 0 {
				delete(m.Packages, pkgPath)
			}
		}
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

	if c.Language == uniast.Cpp && c.exportCtx != nil {
		// AST-only classification — no text fallback. clangd uses short
		// kind names (strips the `Decl` suffix):
		//   `TypeAlias` / `TypeAliasTemplate` -> alias of a type; drop
		//     the symbol, references resolve to the target.
		//   `UsingDirective` (`using namespace foo;`) -> NOT an alias.
		//   `Using` / `UsingShadow` / `UsingPack` -> name-import; only
		//     drop when importing a TYPE (so phantom Class nodes like
		//     `using common::Provider;` get redirected to the real
		//     type). LSP Kind says: SKClass/SKStruct/SKEnum/SKInterface
		//     /SKTypeParameter are types.
		isAlias := false
		if kind, ok := c.cppASTKind(c.exportCtx, symbol); ok {
			switch kind {
			case ASTKindTypeAlias, ASTKindTypeAliasTemplate:
				isAlias = true
			case ASTKindUsing, ASTKindUsingShadow, ASTKindUsingPack:
				switch symbol.Kind {
				case SKClass, SKStruct, SKEnum, SKInterface, SKTypeParameter:
					isAlias = true
				}
			}
		}
		if isAlias {
			e = ErrExternalSymbol
			return
		}
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

		// join name with namespace + class chain
		name = applyCppScopePrefix(c.scopePrefix(symbol), name)
	}

	tmp := uniast.NewIdentity(mod, path, name)
	id = &tmp

	// Eagerly prefix Identity.Name for methods so a cyclic visit
	// (receiver Type -> receivers map -> back to this method via the
	// visited cache) reads the final name, not the bare one. Without
	// this, Type.Methods[k] = *mid value-copies a partially-built id
	// non-deterministically. Cpp finalizes name in the SKMethod branch
	// because it needs extractCppCallSig + namespace munging.
	if c.Language != uniast.Cpp && (symbol.Kind == SKMethod || symbol.Kind == SKFunction) {
		if mi := c.funcs[symbol].Method; mi != nil && mi.Receiver.Symbol != nil {
			recvName := mi.Receiver.Symbol.Name
			if mi.Interface != nil && mi.Interface.Symbol != nil {
				recvName = mi.Interface.Symbol.Name + "<" + recvName + ">"
			}
			sep := "."
			if symbol.Kind == SKFunction {
				sep = "::"
			}
			id.Name = recvName + sep + name
		}
	}

	// Save to visited ONLY WHEN no errors occur
	visited[symbol] = id

	// cstdlib (sysroot) and build_generated (codegen) modules carry
	// only edges by design — collect already drops these syms from
	// c.syms (see addSymbol), so this branch only fires for *recursive*
	// emission via dep edges (Function.FunctionCalls / Types / etc.).
	// Return the Identity so the edge still resolves; skip body emission.
	if c.Language == uniast.Cpp && (mod == "cstdlib" || mod == "build_generated") {
		return
	}

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
		info := c.funcs[symbol]
		// Detect interface-method via LSP parent first (cli.files populated).
		// Java IPC mode does not populate cli.files, so fall back to the
		// receiver symbol kind recorded by the scanner.
		isInterfaceMethod := false
		if c.cli != nil {
			if p := c.cli.GetParent(symbol); p != nil && p.Kind == SKInterface {
				isInterfaceMethod = true
			}
		}
		if isInterfaceMethod {
			// NOTICE: no need collect interface method for non-Java langs.
			// Java still collects it but flags IsInterfaceMethod.
			break
		}
		if info.Method != nil && info.Method.Receiver.Symbol != nil &&
			info.Method.Receiver.Symbol.Kind == SKInterface {
			isInterfaceMethod = true
		}
		obj := &uniast.Function{
			FileLine:          fileLine,
			Content:           content,
			Exported:          public,
			IsInterfaceMethod: isInterfaceMethod,
		}
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

				// cpp get method name without class name and namespace
				if c.Language == uniast.Cpp && rid != nil {
					p := strings.IndexByte(name, '(')
					head, tail := name, ""
					if p >= 0 {
						head, tail = name[:p], name[p:]
					}

					if idx := strings.LastIndex(head, "::"); idx >= 0 {
						head = head[idx+2:]
					}
					name = head + tail
				}

				if k == SKFunction || c.Language == uniast.Cpp {
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
					// Preserve external call edges as lightweight Identities
					// so the call graph remains walkable in default mode.
					if errors.Is(err, ErrExternalSymbol) &&
						(dep.Symbol.Kind == SKFunction || dep.Symbol.Kind == SKMethod) {
						if ext := c.lightIdentityForExternal(dep.Symbol); ext != nil {
							pdep := uniast.NewDependency(*ext, c.fileLine(dep.Location))
							if dep.Symbol.Kind == SKFunction {
								obj.FunctionCalls = uniast.InsertDependency(obj.FunctionCalls, pdep)
							} else {
								if obj.MethodCalls == nil {
									obj.MethodCalls = make([]uniast.Dependency, 0, len(deps))
								}
								obj.MethodCalls = uniast.InsertDependency(obj.MethodCalls, pdep)
							}
						}
					}
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
		// C++: a method is reported by clangd as two DocumentSymbols, one
		// at the header declaration and one at the .cpp definition. We
		// want exactly one Function in the AST per NodeID, preferring the
		// one that carries the body (and thus FC/MC edges). When the
		// definition is unreachable (external base classes, pure-virtual
		// C++ dedup: clangd reports the .h declaration and .cpp definition
		// as two DocumentSymbols. Collapse them into a single entry under
		// the .h pkg (the "owner") carrying the .cpp body's edges.
		// `content` is raw symbol text — ImplHead-wrapped obj.Content can
		// look like it has a body even when the original was a decl.
		dropForDup := false
		if c.Language == uniast.Cpp && (k == SKMethod || k == SKFunction) {
			dropForDup = c.dedupCppFunction(repo, symbol, id.Name, mod, path, content, obj)
		}
		if !dropForDup {
			pkg.Functions[id.Name] = obj
		}

	// Type
	case SKStruct, SKTypeParameter, SKInterface, SKEnum, SKClass:
		// Forward declarations (`class X;` / `struct X;`) are reported by
		// clangd as separate DocumentSymbols. Skip them — the real
		// definition is emitted by a different pass under its own pkg.
		// (typedef X Y; also goes through SKClass/SKStruct in clangd but
		// must NOT be filtered — it never starts with the class/struct
		// keyword.)
		if c.Language == uniast.Cpp && (k == SKClass || k == SKStruct) &&
			!strings.Contains(content, "{") {
			trim := strings.TrimSpace(content)
			if strings.HasPrefix(trim, "class ") || strings.HasPrefix(trim, "struct ") {
				break
			}
		}
		tkind := mapKind(k)
		// C++ typedefs are reported by clangd with SymbolKind Class/Struct,
		// but they're aliases. Tag them as Typedef when clang AST says so.
		// No text-prefix fallback — when ast is unavailable we just leave
		// it as struct/class.
		// clangd's textDocument/ast returns short kind names (no "Decl"
		// suffix): `Typedef` for `typedef X Y;` and `TypeAlias` for
		// `using Y = X;`.
		if c.Language == uniast.Cpp && c.exportCtx != nil {
			if kind, ok := c.cppASTKind(c.exportCtx, symbol); ok &&
				(kind == ASTKindTypedef || kind == ASTKindTypeAlias) {
				tkind = uniast.TypeKindTypedef
			}
		}
		obj := &uniast.Type{
			FileLine: fileLine,
			Content:  content,
			TypeKind: tkind,
			Exported: public,
		}
		// Implements relationship is preserved as a first-class field rather
		// than blended into the generic SubStruct dependency list.
		implSyms := map[*DocumentSymbol]bool{}
		if rels := c.implementsRel[symbol]; rels != nil {
			for _, rel := range rels {
				tok := ""
				if c.cli != nil {
					tok, _ = c.cli.Locate(rel.Location)
				}
				iid, err := c.exportSymbol(repo, rel.Symbol, tok, visited)
				if err != nil {
					// External base classes are valuable signal even when
					// the full external symbol isn't loaded — emit a
					// lightweight Identity so consumers still see the
					// inheritance edge. (Only do this when we actually
					// have enough info to form a stable Identity.)
					if errors.Is(err, ErrExternalSymbol) {
						if ext := c.lightIdentityForExternal(rel.Symbol); ext != nil {
							obj.Implements = append(obj.Implements, *ext)
							implSyms[rel.Symbol] = true
						}
					}
					continue
				}
				obj.Implements = append(obj.Implements, *iid)
				implSyms[rel.Symbol] = true
			}
		}
		// collect deps
		if deps := c.deps[symbol]; deps != nil {
			for _, dep := range deps {
				if implSyms[dep.Symbol] {
					continue
				}
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
				// NOTICE: use method name as key here.
				// For C++: derive the key from the constructed Identity
				// Name (mid.Name), which includes the full signature
				// (`ns::Class::handle(int x)`), then strip the receiver
				// scope via cppBaseName. This produces "handle(int x)"
				// vs "handle(long x)" so overloads don't collapse onto a
				// single short-name key.
				if c.Language == uniast.Cpp {
					methodName := c.cppBaseName(mid.Name)
					_, methodExist := obj.Methods[methodName]
					isHeaderMethod := strings.HasSuffix(method.Location.URI.File(), ".h")
					if methodExist && isHeaderMethod {
						continue
					}
					obj.Methods[methodName] = *mid
				} else {
					obj.Methods[method.Name] = *mid
				}
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
		// Walk over enclosing namespaces AND class/struct/interface scopes so
		// inline methods of external classes get the full receiver qualifier
		// (e.g. `cppservice::ApiHandler::process`). Without the class hop,
		// every method in an external header collapses to `ns::method` and
		// distinct methods of distinct classes clash.
		switch p.Kind {
		case lsp.SKNamespace, lsp.SKClass, lsp.SKStruct, lsp.SKInterface:
			if p.Name != "" {
				n := p.Name
				if i := strings.IndexByte(n, '<'); i >= 0 { // strip template args: Foo<T> -> Foo
					n = n[:i]
				}
				parts = append([]string{n}, parts...)
			}
		}
		cur = p
	}
	return strings.Join(parts, "::") // "a::b"
}

// applyCppScopePrefix prepends the lexical scope (namespace+class chain) to
// the bare symbol name, but only the portion that's actually missing.
//
// Examples (prefix = "cppservice::ApiHandler"):
//
//	"process(...)"                       -> "cppservice::ApiHandler::process(...)"
//	"ApiHandler::process(...)"           -> "cppservice::ApiHandler::process(...)"
//	"cppservice::ApiHandler::process()"  -> unchanged
func applyCppScopePrefix(prefix, name string) string {
	if prefix == "" {
		return name
	}
	parts := strings.Split(prefix, "::")
	for i := 0; i < len(parts); i++ {
		suffix := strings.Join(parts[i:], "::") + "::"
		if strings.HasPrefix(name, suffix) {
			if i == 0 {
				return name
			}
			return strings.Join(parts[:i], "::") + "::" + name
		}
	}
	return prefix + "::" + name
}

// synthesizeInheritedMethodsCpp walks every C++ Type with `Implements` and
// emits a derived-class view of each inherited method, mirroring the way
// Model B describes inheritance: a synthesized `D::m` symbol that owns
// the body of `B::m` but is bound to `D`'s vtable. Virtual dispatch inside
// the inherited body is devirtualized — calls to `B::n` get rewritten to
// `D::n` when `D` overrides `n` — so a walker can step from
// `D::inherited` straight into `D::override` without losing the chain at
// `B::n`.
//
// We deliberately do not rewrite caller-side edges here: the receiver's
// static type at each call site is not currently propagated through the
// collector, so we cannot reliably choose the right `D` per call. That
// rewrite is a follow-up; the synthesized symbols are still useful on
// their own (they materialize the inheritance closure in the AST, and
// downstream tooling can map B::m → {D::m} via the Implements graph).
func (c *Collector) synthesizeInheritedMethodsCpp(repo *uniast.Repository) {
	// Build a flat index of every concrete function in the repo keyed by
	// its Identity, so we can look up methods of a base class B in O(1).
	type funcSlot struct {
		mod *uniast.Module
		pkg *uniast.Package
		fn  *uniast.Function
	}
	byReceiver := map[uniast.Identity][]*funcSlot{}
	// typesById lets a synthesised method back-fill the derived class's
	// Type.Methods map.
	typesById := map[uniast.Identity]*uniast.Type{}
	for _, mod := range repo.Modules {
		for _, pkg := range mod.Packages {
			for _, fn := range pkg.Functions {
				if fn.IsMethod && fn.Receiver != nil {
					byReceiver[fn.Receiver.Type] = append(byReceiver[fn.Receiver.Type], &funcSlot{mod: mod, pkg: pkg, fn: fn})
				}
			}
			for _, ty := range pkg.Types {
				typesById[ty.Identity] = ty
			}
		}
	}

	// methodLocalSig returns the receiver-scope-stripped signature
	// "method(params)" — overload-distinguishing key for dedup so e.g.
	// `Base::foo(int)` and `Base::foo(string)` don't collide.
	methodLocalSig := func(qualified string) string {
		head := qualified
		tail := ""
		if i := strings.Index(qualified, "("); i >= 0 {
			head = qualified[:i]
			tail = qualified[i:]
		}
		if i := strings.LastIndex(head, "::"); i >= 0 {
			head = head[i+2:]
		}
		return head + tail
	}
	// methodScope strips the trailing "::method(params)" giving the
	// receiver namespace+class prefix.
	methodScope := func(qualified string) string {
		s := qualified
		if i := strings.Index(s, "("); i >= 0 {
			s = s[:i]
		}
		if i := strings.LastIndex(s, "::"); i >= 0 {
			return s[:i]
		}
		return ""
	}
	// Replace the receiver scope of a qualified method name with `newScope`,
	// keeping the trailing "::method(params)" intact. Used to rewrite
	// `B::n(args)` -> `D::n(args)` inside synthesized bodies.
	withScope := func(qualified, newScope string) string {
		head := qualified
		tail := ""
		if i := strings.Index(head, "("); i >= 0 {
			tail = head[i:]
			head = head[:i]
		}
		short := head
		if i := strings.LastIndex(head, "::"); i >= 0 {
			short = head[i+2:]
		}
		if newScope == "" {
			return short + tail
		}
		return newScope + "::" + short + tail
	}

	// Collect transitive bases of a type (BFS over Implements).
	transitiveBases := func(t *uniast.Type) []uniast.Identity {
		seen := map[uniast.Identity]bool{}
		var out []uniast.Identity
		queue := append([]uniast.Identity(nil), t.Implements...)
		for len(queue) > 0 {
			b := queue[0]
			queue = queue[1:]
			if seen[b] {
				continue
			}
			seen[b] = true
			out = append(out, b)
			// follow base's bases via repo
			if bm := repo.Modules[b.ModPath]; bm != nil {
				if bp := bm.Packages[b.PkgPath]; bp != nil {
					if bt := bp.Types[b.Name]; bt != nil {
						queue = append(queue, bt.Implements...)
					}
				}
			}
		}
		return out
	}

	// Iterate over each Type in each module/package; types-snapshot first
	// to avoid mutating the map while iterating.
	type typeSlot struct {
		mod *uniast.Module
		pkg *uniast.Package
		ty  *uniast.Type
	}
	var allTypes []typeSlot
	for _, mod := range repo.Modules {
		for _, pkg := range mod.Packages {
			for _, ty := range pkg.Types {
				if len(ty.Implements) > 0 {
					allTypes = append(allTypes, typeSlot{mod, pkg, ty})
				}
			}
		}
	}

	for _, ts := range allTypes {
		D := ts.ty
		// D's own methods, indexed by local signature (short name + param
		// list) so overloads like `foo(int)` vs `foo(string)` are treated
		// as distinct.
		dOwnBySig := map[string]bool{}
		for _, f := range byReceiver[D.Identity] {
			dOwnBySig[methodLocalSig(f.fn.Name)] = true
		}
		// Methods already synthesized this round, keyed by local sig,
		// to avoid duplicates when both diamond bases provide the same
		// overload.
		synthBySig := map[string]bool{}

		dScope := D.Identity.Name // "ns::ClassD"

		for _, baseID := range transitiveBases(D) {
			baseMethods := byReceiver[baseID]
			// C++ headers + .cpp produce two records with identical
			// Identity.Name (declaration in header, definition in cpp).
			// Pick the one with a richer call graph so the synthesis
			// downstream isn't sensitive to map-iteration order. Keyed
			// by local sig so overloads survive.
			bestBySig := map[string]*funcSlot{}
			callCount := func(f *uniast.Function) int {
				return len(f.MethodCalls) + len(f.FunctionCalls) + len(f.GlobalVars) + len(f.Types)
			}
			for _, bf := range baseMethods {
				s := methodLocalSig(bf.fn.Name)
				cur, ok := bestBySig[s]
				if !ok || callCount(bf.fn) > callCount(cur.fn) {
					bestBySig[s] = bf
				}
			}
			for sig, bf := range bestBySig {
				if dOwnBySig[sig] || synthBySig[sig] {
					continue
				}
				// Construct synthesized Identity: "<dScope>::<methodTail>".
				// Search for the receiver-name `::` only in the head (before
				// the parameter list) — parameter types like `const
				// std::string&` contain `::` of their own and would otherwise
				// trick LastIndex into picking the wrong split point.
				tail := bf.fn.Name
				head := bf.fn.Name
				paramStart := len(bf.fn.Name)
				if j := strings.Index(bf.fn.Name, "("); j >= 0 {
					head = bf.fn.Name[:j]
					paramStart = j
				}
				if i := strings.LastIndex(head, "::"); i >= 0 {
					tail = bf.fn.Name[i+2:]
				} else {
					tail = bf.fn.Name[paramStart:] // method has no scope at all
					tail = head + tail
				}
				newName := dScope + "::" + tail
				newId := uniast.NewIdentity(ts.mod.Name, ts.pkg.PkgPath, newName)
				if _, exists := ts.pkg.Functions[newName]; exists {
					synthBySig[sig] = true
					continue
				}

				// Deep-copy the function, devirtualizing this-edges where D
				// overrides the called signature. Match by local sig so we
				// don't redirect `Base::foo(string)` to `D::foo(string)`
				// when D only overrode `foo(int)`.
				rewriteScope := func(target uniast.Identity) uniast.Identity {
					if methodScope(target.Name) != baseID.Name {
						return target
					}
					if !dOwnBySig[methodLocalSig(target.Name)] {
						return target
					}
					// D has its own override of this signature: redirect
					// target to D::<sig>.
					return uniast.NewIdentity(D.Identity.ModPath, D.Identity.PkgPath, withScope(target.Name, dScope))
				}
				cloneDeps := func(in []uniast.Dependency) []uniast.Dependency {
					if len(in) == 0 {
						return nil
					}
					out := make([]uniast.Dependency, 0, len(in))
					for _, d := range in {
						nd := d
						nd.Identity = rewriteScope(d.Identity)
						out = append(out, nd)
					}
					return out
				}

				synth := &uniast.Function{
					FileLine:      bf.fn.FileLine,
					Content:       bf.fn.Content,
					Signature:     bf.fn.Signature,
					Exported:      bf.fn.Exported,
					IsMethod:      true,
					Receiver:      &uniast.Receiver{IsPointer: false, Type: D.Identity},
					MethodCalls:   cloneDeps(bf.fn.MethodCalls),
					FunctionCalls: cloneDeps(bf.fn.FunctionCalls),
					GlobalVars:    cloneDeps(bf.fn.GlobalVars),
					Types:         cloneDeps(bf.fn.Types),
					Params:        cloneDeps(bf.fn.Params),
					Results:       cloneDeps(bf.fn.Results),
					Identity:      newId,
				}
				ts.pkg.Functions[newId.Name] = synth
				synthBySig[sig] = true
				// Back-fill the derived Type's Methods map so consumers
				// can discover the inherited method through D.Methods.
				// Key derived from the synthesised Identity Name via
				// cppBaseName so it matches the native-method key style
				// ("handle(int x)") and overloads don't collapse.
				if dt := typesById[D.Identity]; dt != nil {
					if dt.Methods == nil {
						dt.Methods = map[string]uniast.Identity{}
					}
					key := c.cppBaseName(newId.Name)
					if _, exists := dt.Methods[key]; !exists {
						dt.Methods[key] = newId
					}
				}
			}
		}
	}
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

// extractCppCallSig returns "sym.Name(params)".
//
// Fast path: clangd populates DocumentSymbol.Detail with the canonical
// signature (e.g. "void(int x)") for methods and functions — use that
// directly. Fall back to text-level extraction only when Detail is empty
// (older LSPs or non-clangd providers).
func (c *Collector) extractCppCallSig(sym *lsp.DocumentSymbol) (ret string) {
	name := strings.TrimSpace(sym.Name)
	if name == "" {
		return ""
	}
	if detail := strings.TrimSpace(sym.Detail); detail != "" {
		// clangd's Detail is the bare type "void(int x)" — we want
		// "name(int x)". Find the params parenthesis group.
		if i := strings.IndexByte(detail, '('); i >= 0 {
			params := detail[i:]
			// Trim trailing return-type-style annotations after a
			// matching ')' close (e.g. " const" / " noexcept" / " -> X").
			depth := 0
			end := -1
			for j := 0; j < len(params); j++ {
				switch params[j] {
				case '(':
					depth++
				case ')':
					depth--
					if depth == 0 {
						end = j + 1
					}
				}
				if end > 0 {
					break
				}
			}
			if end > 0 {
				return name + params[:end]
			}
		}
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
