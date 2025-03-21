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

	"github.com/cloudwego/abcoder/src/lang/log"
	"github.com/cloudwego/abcoder/src/lang/lsp"
	. "github.com/cloudwego/abcoder/src/lang/lsp"
	parse "github.com/cloudwego/abcoder/src/uniast"
)

type dependency struct {
	Location Location        `json:"location"`
	Symbol   *DocumentSymbol `json:"symbol"`
}

func (d dependency) FileLine() parse.FileLine {
	return parse.FileLine{
		File: d.Location.URI.File(),
		Line: d.Location.Range.Start.Line,
	}
}

func newModule(name string, dir string) *parse.Module {
	ret := parse.NewModule(name, dir)
	ret.Language = parse.Rust
	return ret
}

func (c *Collector) Export(ctx context.Context) (*parse.Repository, error) {
	// recursively read all go files in repo
	repo := parse.NewRepository(c.repo)
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
		repo.Modules[name] = newModule(name, rel)
	}

	// export symbols
	for _, symbol := range c.syms {
		visited := make(map[*lsp.DocumentSymbol]*parse.Identity)
		_, err := c.exportSymbol(&repo, symbol, "", visited)
		if err != nil {
			log.Info("export symbol %s failed: %v\n", symbol, err)
		}
	}

	// patch module
	for p, m := range repo.Modules {
		if p == "" || strings.Contains(p, "@") {
			continue
		}
		c.modPatcher.Patch(m)
	}

	return &repo, nil
}

func (c *Collector) exportSymbol(repo *parse.Repository, symbol *DocumentSymbol, refName string, visited map[*DocumentSymbol]*parse.Identity) (*parse.Identity, error) {
	if symbol == nil {
		return nil, errors.New("symbol is nil")
	}
	if id, ok := visited[symbol]; ok {
		return id, nil
	}

	name := symbol.Name
	if name == "" {
		if refName == "" {
			return nil, fmt.Errorf("both symbol %v name and refname is empty", symbol)
		}
		// NOTICE: use refName as id when symbol name is missing
		name = refName
	}
	file := symbol.Location.URI.File()
	mod, path, err := c.spec.NameSpace(file)
	if err != nil {
		return nil, err
	}
	id := parse.NewIdentity(mod, path, name)
	visited[symbol] = &id

	// Load eternal symbol on demands
	if !c.LoadExternalSymbol && (!c.internal(symbol.Location) || symbol.Kind == SKUnknown) {
		return &id, nil
	}

	if repo.Modules[mod] == nil {
		repo.Modules[mod] = newModule(mod, "")
	}
	module := repo.Modules[mod]
	if repo.Modules[mod].Packages[path] == nil {
		repo.Modules[mod].Packages[path] = parse.NewPackage(path)
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
	fileLine := parse.FileLine{
		File: relfile,
		Line: symbol.Location.Range.Start.Line + 1,
	}
	// collect files
	if module.Files[relfile] == nil {
		module.Files[relfile] = parse.NewFile(relfile)
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
		obj := &parse.Function{
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
					log.Error("export input symbol %s failed: %v\n", input.Symbol, err)
					continue
				}
				dep := parse.NewDependency(*tyid, input.FileLine())
				obj.Types = parse.Dedup(obj.Types, dep)
			}
		}
		if info.Inputs != nil {
			for _, input := range info.InputsSorted {
				tok, _ := c.cli.Locate(input.Location)
				tyid, err := c.exportSymbol(repo, input.Symbol, tok, visited)
				if err != nil {
					log.Error("export input symbol %s failed: %v\n", input.Symbol, err)
					continue
				}
				dep := parse.NewDependency(*tyid, input.FileLine())
				obj.Params = parse.Dedup(obj.Params, dep)
			}
		}
		if info.Outputs != nil {
			for _, output := range info.OutputsSorted {
				tok, _ := c.cli.Locate(output.Location)
				tyid, err := c.exportSymbol(repo, output.Symbol, tok, visited)
				if err != nil {
					log.Error("export output symbol %s failed: %v\n", output.Symbol, err)
					continue
				}
				dep := parse.NewDependency(*tyid, output.FileLine())
				obj.Results = parse.Dedup(obj.Results, dep)
			}
		}
		if info.Method != nil && info.Method.Receiver.Symbol != nil {
			tok, _ := c.cli.Locate(info.Method.Receiver.Location)
			rid, err := c.exportSymbol(repo, info.Method.Receiver.Symbol, tok, visited)
			if err == nil {
				obj.Receiver = &parse.Receiver{
					Type: *rid,
					Name: rid.Name,
				}
				obj.IsMethod = true
				id.Name = rid.Name
				// NOTICE: check if the method is a trait method
				// if true, type = trait<receiver>
				if info.Method.Interface != nil {
					itok, _ := c.cli.Locate(info.Method.Interface.Location)
					iid, err := c.exportSymbol(repo, info.Method.Interface.Symbol, itok, visited)
					if err != nil {
						log.Error("export interface symbol %s failed: %v\n", info.Method.Interface.Symbol, err)
					} else {
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
			} else {
				log.Error("export receiver symbol %s failed: %v\n", info.Method.Receiver.Symbol, err)
			}
		}
		// collect deps
		if deps := c.deps[symbol]; deps != nil {
			for _, dep := range deps {
				tok, _ := c.cli.Locate(dep.Location)
				depid, err := c.exportSymbol(repo, dep.Symbol, tok, visited)
				if err != nil {
					log.Error("export dep symbol %s failed: %v\n", dep.Symbol, err)
					continue
				}
				pdep := parse.NewDependency(*depid, dep.FileLine())
				switch dep.Symbol.Kind {
				case lsp.SKFunction:
					obj.FunctionCalls = parse.Dedup(obj.FunctionCalls, pdep)
				case lsp.SKMethod:
					if obj.MethodCalls == nil {
						obj.MethodCalls = make([]parse.Dependency, 0, len(deps))
					}
					// NOTICE: use loc token as key here, to make it more readable
					obj.MethodCalls = parse.Dedup(obj.MethodCalls, pdep)
				case lsp.SKVariable, lsp.SKConstant:
					if obj.GolobalVars == nil {
						obj.GolobalVars = make([]parse.Dependency, 0, len(deps))
					}
					obj.GolobalVars = parse.Dedup(obj.GolobalVars, pdep)
				case lsp.SKStruct, lsp.SKTypeParameter, lsp.SKInterface, lsp.SKEnum:
					if obj.Types == nil {
						obj.Types = make([]parse.Dependency, 0, len(deps))
					}
					obj.Types = parse.Dedup(obj.Types, pdep)
				default:
					log.Error("dep symbol %s not collected for %v\n", dep.Symbol, id)
				}
			}
		}
		obj.Identity = id
		pkg.Functions[id.Name] = obj

	// Type
	case lsp.SKStruct, lsp.SKTypeParameter, lsp.SKInterface, lsp.SKEnum:
		obj := &parse.Type{
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
					log.Error("export dep symbol %s failed: %v\n", dep.Symbol, err)
					continue
				}
				switch dep.Symbol.Kind {
				case lsp.SKStruct, lsp.SKTypeParameter, lsp.SKInterface, lsp.SKEnum:
					obj.SubStruct = append(obj.SubStruct, parse.NewDependency(*depid, dep.FileLine()))
				default:
					log.Error("dep symbol %s not collected for \n", dep.Symbol, id)
				}
			}
		}
		// collect methods
		if rec := receivers[symbol]; rec != nil {
			obj.Methods = make(map[string]parse.Identity, len(rec))
			for _, method := range rec {
				tok, _ := c.cli.Locate(method.Location)
				mid, err := c.exportSymbol(repo, method, tok, visited)
				if err != nil {
					log.Error("export method symbol %s failed: %v\n", method, err)
					continue
				}
				// NOTICE: use method name as key here
				obj.Methods[method.Name] = *mid
			}
		}
		obj.Identity = id
		pkg.Types[id.Name] = obj
	// Vars
	case lsp.SKConstant, lsp.SKVariable:
		obj := &parse.Var{
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
			} else {
				log.Error("export var type symbol %s failed: %v\n", ty.Symbol, err)
			}
		}
		obj.Identity = id
		pkg.Vars[id.Name] = obj
	default:
		log.Error("symbol %s not collected\n", symbol)
	}

	return &id, nil
}

func mapKind(kind lsp.SymbolKind) parse.TypeKind {
	switch kind {
	case lsp.SKStruct:
		return parse.TypeKindStruct
	case lsp.SKTypeParameter:
		return parse.TypeKindNamed
	case lsp.SKInterface:
		return parse.TypeKindInterface
	case lsp.SKEnum:
		return parse.TypeKindEnum
	default:
		panic(fmt.Sprintf("unexpected kind %v", kind))
	}
}
