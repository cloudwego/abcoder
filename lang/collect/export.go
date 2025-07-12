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
	"path/filepath"
	"strings"

	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/lang/lsp"
	. "github.com/cloudwego/abcoder/lang/lsp"
	"github.com/cloudwego/abcoder/lang/uniast"
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
	text := c.cli.GetFile(loc.URI).Text
	file_uri := string(loc.URI)
	return uniast.FileLine{
		File:        rel,
		Line:        loc.Range.Start.Line + 1,
		StartOffset: lsp.PositionOffset(file_uri, text, loc.Range.Start),
		EndOffset:   lsp.PositionOffset(file_uri, text, loc.Range.End),
	}
}

func newModule(name string, dir string, lang uniast.Language) *uniast.Module {
	ret := uniast.NewModule(name, dir, lang)
	return ret
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
	c.filterLocalSymbols()

	// export symbols
	visited := make(map[*lsp.DocumentSymbol]*uniast.Identity)
	for _, symbol := range c.syms {
		_, _ = c.exportSymbol(&repo, symbol, "", visited)
	}

	// patch module
	if c.modPatcher != nil {
		for p, m := range repo.Modules {
			if p == "" || strings.Contains(p, "@") {
				continue
			}
			c.modPatcher.Patch(m)
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
				delete(c.syms, loc1)
				break
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
	if id, ok := visited[symbol]; ok {
		return id, nil
	}

	name := symbol.Name
	if name == "" {
		if refName == "" {
			e = fmt.Errorf("both symbol %v name and refname is empty", symbol)
			return
		}
		// NOTICE: use refName as id when symbol name is missing
		name = refName
	}
	file := symbol.Location.URI.File()
	mod, path, err := c.spec.NameSpace(file)
	if err != nil {
		e = err
		return
	}

	if !c.NeedStdSymbol && mod == "" {
		e = ErrStdSymbol
		return
	}

	tmp := uniast.NewIdentity(mod, path, name)
	id = &tmp
	visited[symbol] = id

	// Load eternal symbol on demands
	if !c.LoadExternalSymbol && (!c.internal(symbol.Location) || symbol.Kind == SKUnknown) {
		e = ErrExternalSymbol
		return
	}

	if repo.Modules[mod] == nil {
		repo.Modules[mod] = newModule(mod, "", c.Language)
	}
	module := repo.Modules[mod]
	if repo.Modules[mod].Packages[path] == nil {
		repo.Modules[mod].Packages[path] = uniast.NewPackage(path)
	}
	pkg := repo.Modules[mod].Packages[path]
	if c.spec.IsMainFunction(*symbol) {
		pkg.IsMain = true
	}

	var relfile string
	if c.internal(symbol.Location) {
		relfile, _ = filepath.Rel(c.repo, file)
	} else {
		relfile = filepath.Base(file)
	}
	fileLine := c.fileLine(symbol.Location)
	// collect files
	if module.Files[relfile] == nil {
		module.Files[relfile] = uniast.NewFile(relfile)
	}

	content := symbol.Text
	public := c.spec.IsPublicSymbol(*symbol)

	// map receiver to methods
	receivers := make(map[*DocumentSymbol][]*DocumentSymbol, len(c.funcs)/4)
	for method, rec := range c.funcs {
		if method.Kind == lsp.SKMethod && rec.Method != nil && rec.Method.Receiver.Symbol != nil {
			receivers[rec.Method.Receiver.Symbol] = append(receivers[rec.Method.Receiver.Symbol], method)
		}
	}

	switch k := symbol.Kind; k {
	// Function
	case lsp.SKFunction, lsp.SKMethod:
		if p := c.cli.GetParent(symbol); p != nil && p.Kind == lsp.SKInterface {
			// NOTICE: no need collect interface method
			break
		}
		obj := &uniast.Function{
			FileLine: fileLine,
			Content:  content,
			Exported: public,
		}
		info := c.funcs[symbol]
		// NOTICE: type parames collect into types
		if info.TypeParams != nil {
			for _, input := range info.TypeParamsSorted {
				tok, _ := c.cli.Locate(input.Location)
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
				tok, _ := c.cli.Locate(input.Location)
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
				tok, _ := c.cli.Locate(output.Location)
				tyid, err := c.exportSymbol(repo, output.Symbol, tok, visited)
				if err != nil {
					continue
				}
				dep := uniast.NewDependency(*tyid, c.fileLine(output.Location))
				obj.Results = uniast.InsertDependency(obj.Results, dep)
			}
		}
		if info.Method != nil && info.Method.Receiver.Symbol != nil {
			tok, _ := c.cli.Locate(info.Method.Receiver.Location)
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
					itok, _ := c.cli.Locate(info.Method.Interface.Location)
					iid, err := c.exportSymbol(repo, info.Method.Interface.Symbol, itok, visited)
					if err == nil {
						id.Name = iid.Name + "<" + id.Name + ">"
					}
				}
				if k == lsp.SKFunction {
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
				tok, _ := c.cli.Locate(dep.Location)
				depid, err := c.exportSymbol(repo, dep.Symbol, tok, visited)
				if err != nil {
					continue
				}
				pdep := uniast.NewDependency(*depid, c.fileLine(dep.Location))
				switch dep.Symbol.Kind {
				case lsp.SKFunction:
					obj.FunctionCalls = uniast.InsertDependency(obj.FunctionCalls, pdep)
				case lsp.SKMethod:
					if obj.MethodCalls == nil {
						obj.MethodCalls = make([]uniast.Dependency, 0, len(deps))
					}
					// NOTICE: use loc token as key here, to make it more readable
					obj.MethodCalls = uniast.InsertDependency(obj.MethodCalls, pdep)
				case lsp.SKVariable, lsp.SKConstant:
					if obj.GlobalVars == nil {
						obj.GlobalVars = make([]uniast.Dependency, 0, len(deps))
					}
					obj.GlobalVars = uniast.InsertDependency(obj.GlobalVars, pdep)
				case lsp.SKStruct, lsp.SKTypeParameter, lsp.SKInterface, lsp.SKEnum, lsp.SKClass:
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
	case lsp.SKStruct, lsp.SKTypeParameter, lsp.SKInterface, lsp.SKEnum, lsp.SKClass:
		obj := &uniast.Type{
			FileLine: fileLine,
			Content:  content,
			TypeKind: mapKind(k),
			Exported: public,
		}
		// collect deps
		if deps := c.deps[symbol]; deps != nil {
			for _, dep := range deps {
				tok, _ := c.cli.Locate(dep.Location)
				depid, err := c.exportSymbol(repo, dep.Symbol, tok, visited)
				if err != nil {
					continue
				}
				switch dep.Symbol.Kind {
				case lsp.SKStruct, lsp.SKTypeParameter, lsp.SKInterface, lsp.SKEnum, lsp.SKClass:
					obj.SubStruct = uniast.InsertDependency(obj.SubStruct, uniast.NewDependency(*depid, c.fileLine(dep.Location)))
				default:
					log.Error("dep symbol %s not collected for \n", dep.Symbol, id)
				}
			}
		}
		// collect methods
		if rec := receivers[symbol]; rec != nil {
			obj.Methods = make(map[string]uniast.Identity, len(rec))
			for _, method := range rec {
				tok, _ := c.cli.Locate(method.Location)
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
	case lsp.SKConstant, lsp.SKVariable:
		obj := &uniast.Var{
			FileLine:   fileLine,
			Content:    content,
			IsExported: public,
			IsConst:    k == lsp.SKConstant,
		}
		if ty, ok := c.vars[symbol]; ok {
			tok, _ := c.cli.Locate(ty.Location)
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

func mapKind(kind lsp.SymbolKind) uniast.TypeKind {
	switch kind {
	case lsp.SKStruct:
		return "struct"
	// XXX: C++ should use class instead of struct
	case lsp.SKClass:
		return "struct"
	case lsp.SKTypeParameter:
		return "type-parameter"
	case lsp.SKInterface:
		return "interface"
	case lsp.SKEnum:
		return "enum"
	default:
		panic(fmt.Sprintf("unexpected kind %v", kind))
	}
}
