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

package parse

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Repository
type Repository struct {
	Name    string             `json:"id"` // go module name
	Modules map[string]*Module // module name => Library
}

func NewRepository(name string) Repository {
	ret := Repository{
		Name:    name,
		Modules: map[string]*Module{},
	}
	return ret
}

type File struct {
	Name    string
	Imports []string
}

func NewFile(path string) *File {
	ret := File{
		Name: filepath.Base(path),
	}
	return &ret
}

type Module struct {
	Name         string               // go module name
	Dir          string               // relative path to repo
	Packages     map[PkgPath]*Package // pkage import path => Package
	Dependencies map[string]string    `json:",omitempty"` // module name => module_path@version
	Files        map[string]*File     `json:",omitempty"` // relative path => file info
}

func NewModule(name string, dir string) *Module {
	if strings.Contains(name, "@") {
		name = strings.Split(name, "@")[0]
	}
	ret := Module{
		Name:         name,
		Dir:          dir,
		Packages:     map[PkgPath]*Package{},
		Dependencies: map[string]string{},
		Files:        map[string]*File{},
	}
	return &ret
}

// Package
type Package struct {
	IsMain bool
	PkgPath
	Functions    map[string]*Function // Function name (may be {{func}} or {{struct.method}}) => Function
	Types        map[string]*Type     // type name => type define
	Vars         map[string]*Var      // var name => var define
	CompressData *string              `json:"compress_data,omitempty"` // package compress info
}

func NewPackage(pkgPath PkgPath) *Package {
	ret := Package{
		PkgPath:   pkgPath,
		Functions: map[string]*Function{},
		Types:     map[string]*Type{},
		Vars:      map[string]*Var{},
	}
	return &ret
}

// PkgPath is the import path of a package, it is either absolute path or url
type PkgPath = string

// Identity holds identity information about a third party declaration
type Identity struct {
	ModPath string // ModPath is the module which the package belongs to
	PkgPath        // Import Path of the third party package
	Name    string // Unique Name of declaration (FunctionName, TypeName.MethodName, InterfaceName<TypeName>.MethodName, or TypeName)
}

func NewIdentity(mod, pkg, name string) Identity {
	if mod == "" {
		fmt.Fprintf(os.Stderr, "module name cannot be empty: %s.%s\n", pkg, name)
		// panic(fmt.Sprintf("module name cannot be empty: %s.%s", pkg, name))
	}
	return Identity{ModPath: mod, PkgPath: pkg, Name: name}
}

func newIdentity(mod, pkg, name string) Identity {
	return Identity{ModPath: mod, PkgPath: pkg, Name: name}
}

// return full packagepath.name
func (i Identity) String() string {
	return i.PkgPath + "#" + i.Name
}

// return packagename.name
func (i Identity) CallName() string {
	if i.PkgPath != "" {
		return filepath.Base(i.PkgPath) + "." + i.Name
	}
	return i.Name
}

func (i Identity) Full() string {
	return i.ModPath + "?" + i.PkgPath + "#" + i.Name
}

// GetFunction the function identified by id.
// if id indicates a method, it will try traceinto inlined sub structs to get the named method
func (p Repository) GetFunction(id Identity) *Function {
	lib := p.Modules[id.ModPath]
	if lib == nil {
		return nil
	}
	if pkg, ok := lib.Packages[id.PkgPath]; ok {
		if f := pkg.Functions[id.Name]; f != nil {
			return f
		}
	}
	return nil
}

func (p *Repository) SetFunction(id Identity, f *Function) *Function {
	lib := p.Modules[id.ModPath]
	if lib == nil {
		lib = NewModule(id.ModPath, "")
		p.Modules[id.ModPath] = lib
	}
	pp, ok := lib.Packages[id.PkgPath]
	if !ok {
		pp = NewPackage(id.PkgPath)
		lib.Packages[id.PkgPath] = pp
	}
	if pp.Functions[id.Name] == nil {
		pp.Functions[id.Name] = f
	}
	if id.Name == "main" {
		pp.IsMain = true
	}
	return pp.Functions[id.Name]
}

func (p Repository) GetType(id Identity) *Type {
	lib := p.Modules[id.ModPath]
	if lib == nil {
		return nil
	}
	if pkg, ok := lib.Packages[id.PkgPath]; ok {
		return pkg.Types[id.Name]
	}
	return nil
}

func (p *Repository) SetType(id Identity, f *Type) *Type {
	lib := p.Modules[id.ModPath]
	if lib == nil {
		lib = NewModule(id.ModPath, "")
		p.Modules[id.ModPath] = lib
	}
	pp, ok := lib.Packages[id.PkgPath]
	if !ok {
		pp = NewPackage(id.PkgPath)
		lib.Packages[id.PkgPath] = pp
	}
	if pp.Types[id.Name] == nil {
		pp.Types[id.Name] = f
	}
	return pp.Types[id.Name]
}

func (p *Repository) GetVar(id Identity) *Var {
	lib := p.Modules[id.ModPath]
	if lib == nil {
		return nil
	}
	if pkg, ok := lib.Packages[id.PkgPath]; ok {
		return pkg.Vars[id.Name]
	}
	return nil
}

func (p *Repository) SetVar(id Identity, v *Var) *Var {
	lib := p.Modules[id.ModPath]
	if lib == nil {
		lib = NewModule(id.ModPath, "")
		p.Modules[id.ModPath] = lib
	}
	pp, ok := lib.Packages[id.PkgPath]
	if !ok {
		pp = NewPackage(id.PkgPath)
		lib.Packages[id.PkgPath] = pp
	}
	if pp.Vars[id.Name] == nil {
		pp.Vars[id.Name] = v
	}
	return pp.Vars[id.Name]
}

func (p *goParser) parseImports(fset *token.FileSet, file []byte, mod *Module, impts []*ast.ImportSpec) (*importInfo, error) {
	thirdPartyImports := make(map[string][2]string)
	projectImports := make(map[string]string)
	sysImports := make(map[string]string)
	ret := &importInfo{}
	for _, imp := range impts {
		ret.Origins = append(ret.Origins, string(GetRawContent(fset, file, imp.Path)))
		importPath := imp.Path.Value[1 : len(imp.Path.Value)-1] // remove the quotes
		importAlias := ""
		// Check if user has defined an alias for current import
		if imp.Name != nil {
			importAlias = imp.Name.Name // update the alias
		} else {
			importAlias = getPackageAlias(importPath)
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

func (p *goParser) ParseNode(pkgPath string, name string) (Repository, error) {
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

func (p *goParser) associateImplements() {
	for typ, tid := range p.types {
		for iface, iid := range p.interfaces {
			if types.Implements(typ, iface) {
				tobj := p.getRepo().GetType(tid)
				tobj.Implements = append(tobj.Implements, iid)
			}
			// 另外检查 typ 的指针类型是否实现了 iface
			if types.Implements(types.NewPointer(typ), iface) {
				tobj := p.getRepo().GetType(tid)
				tobj.Implements = append(tobj.Implements, iid)
			}
		}
	}
}

func (p *goParser) ParsePackage(pkgPath PkgPath) (Repository, error) {
	if err := p.parsePackage(pkgPath); err != nil {
		return Repository{}, err
	}
	repo := p.getRepo()
	k, _ := p.getModuleFromPkg(pkgPath)
	var out = NewRepository(repo.Name)
	out.Modules[k] = NewModule(repo.Modules[k].Name, repo.Modules[k].Dir)
	out.Modules[k].Packages[pkgPath] = repo.Modules[k].Packages[pkgPath]
	return out, nil
}

func (p *goParser) parsePackage(pkgPath PkgPath) (err error) {
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

	return p.loadPackages(lib, dir, pkgPath)
}

var loadCount = 0

func (p *goParser) loadPackages(mod *Module, dir string, pkgPath PkgPath) (err error) {
	if mm := p.repo.Modules[mod.Name]; mm != nil && (*mm).Packages[pkgPath] != nil {
		return nil
	}
	fmt.Fprintf(os.Stderr, "[loadPackages] mod: %s, dir: %s, pkgPath: %s", mod.Name, dir, pkgPath)
	loadCount++
	// slow-path: load packages in the dir, including sub pakcages
	opts := packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports
	if p.opts.ReferCodeDepth != 0 {
		opts |= packages.NeedDeps
	}
	fset := token.NewFileSet()
	pkgs, err := packages.Load(&packages.Config{
		Mode: opts,
		Fset: fset,
		Dir:  dir,
	}, pkgPath)
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

		for idx, file := range pkg.Syntax {
			filePath := pkg.GoFiles[idx]
			bs := p.getFileBytes(filePath)
			ctx := &fileContext{
				repoDir:     p.homePageDir,
				filePath:    filePath,
				module:      mod,
				pkgPath:     pkg.ID,
				bs:          bs,
				fset:        fset,
				pkgTypeInfo: pkg.TypesInfo,
				deps:        pkg.Imports,
			}
			imports, err := p.parseImports(ctx.fset, ctx.bs, mod, file.Imports)
			if err != nil {
				return err
			}
			ctx.imports = imports
			relpath, _ := filepath.Rel(p.homePageDir, filePath)
			f := NewFile(relpath)
			mod.Files[relpath] = f
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
		}
	}
	return
}
