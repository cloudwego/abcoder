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

package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	. "github.com/cloudwego/abcoder/lang/uniast"
	"golang.org/x/tools/go/packages"
)

func (p *GoParser) parseImports(fset *token.FileSet, file []byte, mod *Module, impts []*ast.ImportSpec) (*importInfo, error) {
	thirdPartyImports := make(map[string][2]string)
	projectImports := make(map[string]string)
	sysImports := make(map[string]string)
	ret := &importInfo{}
	for _, imp := range impts {
		importPath := imp.Path.Value[1 : len(imp.Path.Value)-1] // remove the quotes
		importAlias := ""
		// Check if user has defined an alias for current import
		if imp.Name != nil {
			importAlias = imp.Name.Name // update the alias
			ret.Origins = append(ret.Origins, Import{Path: imp.Path.Value, Alias: &importAlias})
		} else {
			importAlias = getPackageAlias(importPath)
			ret.Origins = append(ret.Origins, Import{Path: imp.Path.Value})
		}

		// Fix: module name may also be like this?
		if isSysPkg(importPath) {
			// Ignoring golang standard libraries（like net/http）
			sysImports[importAlias] = importPath
		} else {
			// Distinguish between project packages and third party packages
			if strings.HasPrefix(importPath, mod.Name) {
				projectImports[importAlias] = importPath
			} else {
				mod := mod.GetDependency(importPath)
				thirdPartyImports[importAlias] = [2]string{mod, importPath}
			}
		}
	}
	ret.SysImports = sysImports
	ret.ProjectImports = projectImports
	ret.ThirdPartyImports = thirdPartyImports
	return ret, nil
}

func (p *GoParser) ParseNode(pkgPath string, name string) (Repository, error) {
	out := NewRepository(p.repo.Name)
	if pkgPath == "" {
		//search mode
		idss, err := p.searchName(name)
		if err != nil {
			return Repository{}, fmt.Errorf("Error search %v:%v", name, err)
		}

		for _, id := range idss {
			if err := loadNode(p, id.PkgPath, id.Name, &out); err != nil {
				return out, err
			}
		}
	} else {
		// parse entity
		pkgPath, name := pkgPath, name
		if err := loadNode(p, pkgPath, name, &out); err != nil {
			return out, err
		}
	}
	return out, nil
}

func (p *GoParser) associateImplements() {
	for typ, tid := range p.types {
		for iface, iid := range p.interfaces {
			if types.Implements(typ, iface) {
				tobj := p.getRepo().GetType(tid)
				tobj.Implements = Append(tobj.Implements, iid)
			}
			// 另外检查 typ 的指针类型是否实现了 iface
			if types.Implements(types.NewPointer(typ), iface) {
				tobj := p.getRepo().GetType(tid)
				tobj.Implements = Append(tobj.Implements, iid)
			}
		}
	}
}

func (p *GoParser) ParsePackage(pkgPath PkgPath) (Repository, error) {
	if err := p.parsePackage(pkgPath); err != nil {
		return Repository{}, err
	}
	repo := p.getRepo()
	k, _ := p.getModuleFromPkg(pkgPath)
	var out = NewRepository(repo.Name)
	out.Modules[k] = newModule(repo.Modules[k].Name, repo.Modules[k].Dir)
	out.Modules[k].Packages[pkgPath] = repo.Modules[k].Packages[pkgPath]
	return out, nil
}

func (p *GoParser) parsePackage(pkgPath PkgPath) (err error) {
	mod, dir := p.getModuleFromPkg(pkgPath)
	if mod == "" {
		return fmt.Errorf("not found module for package %s", pkgPath)
	}
	if dir == "" {
		// NOTICE: external package should set the dir to the one of its referer
		for _, m := range p.repo.Modules {
			if m.GetDependency(pkgPath) != "" {
				dir = filepath.Join(p.homePageDir, m.Dir)
				break
			}
		}
	}
	// fast-path: check cache first
	if p.visited[pkgPath] {
		return nil
	}
	p.visited[pkgPath] = true

	lib := p.repo.Modules[mod]
	if lib == nil {
		return fmt.Errorf("module not load: %s", mod)
	}
	// fmt.Println("[parsePackage] mod:", mod, "dir:", dir, "pkgPath:", pkgPath, p.opts.ReferCodeDepth)

	return p.loadPackages(lib, filepath.Join(p.homePageDir, lib.Dir), pkgPath)
}

var loadCount = 0

func (p *GoParser) loadPackages(mod *Module, dir string, pkgPath PkgPath) (err error) {
	if mm := p.repo.Modules[mod.Name]; mm != nil && (*mm).Packages[pkgPath] != nil {
		return nil
	}
	fmt.Fprintf(os.Stderr, "[loadPackages] mod: %s, dir: %s, pkgPath: %s\n", mod.Name, dir, pkgPath)
	fset := token.NewFileSet()
	loadCount++
	// slow-path: load packages in the dir, including sub pakcages
	opts := packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports
	cfg := &packages.Config{
		Mode: opts,
		Fset: fset,
		Dir:  dir,
	}
	if p.opts.ReferCodeDepth != 0 {
		opts |= packages.NeedDeps
	}
	if p.opts.NeedTest {
		opts |= packages.NeedForTest
		cfg.Tests = true
	}
	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return fmt.Errorf("load path '%s' failed: %v", dir, err)
	}
	for _, pkg := range pkgs {
		if mm := p.repo.Modules[mod.Name]; mm != nil && (*mm).Packages[pkg.ID] != nil {
			continue
		}
		if pp, ok := mod.Packages[pkg.ID]; ok && pp != nil {
			continue
		}
	next_file:
		for idx, file := range pkg.Syntax {
			if idx >= len(pkg.GoFiles) {
				fmt.Fprintf(os.Stderr, "skip file %s by loader\n", file.Name)
				continue
			}
			filePath := pkg.GoFiles[idx]
			for _, exclude := range p.exclues {
				if exclude.MatchString(filePath) {
					fmt.Fprintf(os.Stderr, "skip file %s\n", filePath)
					continue next_file
				}
			}
			bs := p.getFileBytes(filePath)
			ctx := &fileContext{
				repoDir:        p.homePageDir,
				filePath:       filePath,
				module:         mod,
				pkgPath:        pkg.ID,
				bs:             bs,
				fset:           fset,
				pkgTypeInfo:    pkg.TypesInfo,
				deps:           pkg.Imports,
				collectComment: p.opts.CollectComment,
			}
			imports, err := p.parseImports(ctx.fset, ctx.bs, mod, file.Imports)
			if err != nil {
				return err
			}
			ctx.imports = imports
			relpath, _ := filepath.Rel(p.homePageDir, filePath)
			f := mod.Files[relpath]
			if f == nil {
				f = NewFile(relpath)
				mod.Files[relpath] = f
			}
			pkgid := pkg.ID
			f.Package = &pkgid
			f.Imports = imports.Origins
			if err := p.parseFile(ctx, file); err != nil {
				return err
			}
		}
		if obj := mod.Packages[pkg.ID]; obj != nil {
			// obj.Dependencies = make([]PkgPath, 0, len(pkg.Imports))
			// for _, imp := range pkg.Imports {
			// 	if isSysPkg(imp.ID) {
			// 		continue
			// 	}
			// 	obj.Dependencies = append(obj.Dependencies, imp.ID)
			// }
			obj.PkgPath = pkg.ID
			if strings.HasSuffix(obj.PkgPath, ".test]") {
				obj.IsTest = true
			}
			if strings.HasSuffix(obj.PkgPath, ".test") {
				delete(mod.Packages, obj.PkgPath)
			}
		}
	}
	return
}

func IsTestPackage(pkgPath string) bool {
	return strings.HasSuffix(pkgPath, ".test") || strings.HasSuffix(pkgPath, ".test]")
}
