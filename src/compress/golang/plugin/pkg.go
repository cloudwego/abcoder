/**
 * Copyright 2024 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

// Repository
type Repository struct {
	Name    string             // go module name
	Modules map[string]*Module // module name => Library
}

func NewRepository(name string) Repository {
	ret := Repository{
		Name:    name,
		Modules: map[string]*Module{},
	}
	return ret
}

type Module struct {
	Name         string               // go module name
	Dir          string               // relative path to repo
	Packages     map[PkgPath]*Package // pkage import path => Package
	Dependencies map[string]string    // module name => module_path@version
}

func NewModule(name string, dir string) *Module {
	ret := Module{
		Name:         name,
		Dir:          dir,
		Packages:     map[PkgPath]*Package{},
		Dependencies: map[string]string{},
	}
	return &ret
}

// Package
type Package struct {
	PkgPath
	// Dependencies []PkgPath
	Functions map[string]*Function // Function name (may be {{func}} or {{struct.method}}) => Function
	Types     map[string]*Type     // type name => type define
}

func NewPacakge(pkgPath PkgPath) *Package {
	ret := Package{
		PkgPath:   pkgPath,
		Functions: map[string]*Function{},
		Types:     map[string]*Type{},
	}
	return &ret
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

func (p *Repository) SetFunction(id Identity, f *Function) {
	lib := p.Modules[id.ModPath]
	if lib == nil {
		lib = &Module{
			Name: id.ModPath,
		}
		p.Modules[id.ModPath] = lib
	}
	pp, ok := lib.Packages[id.PkgPath]
	if !ok {
		pp = &Package{
			Functions: map[string]*Function{},
			Types:     map[string]*Type{},
		}
		lib.Packages[id.PkgPath] = pp
	}
	if pp.Functions[id.Name] != nil {
		// FIXME
		panic("duplicated function:" + id.String())
	}
	pp.Functions[id.Name] = f
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

func (p *Repository) SetType(id Identity, f *Type) {
	lib := p.Modules[id.ModPath]
	if lib == nil {
		lib = &Module{
			Name: id.ModPath,
		}
		p.Modules[id.ModPath] = lib
	}
	pp, ok := lib.Packages[id.PkgPath]
	if !ok {
		pp = &Package{
			Functions: map[string]*Function{},
			Types:     map[string]*Type{},
		}
		lib.Packages[id.PkgPath] = pp
	}
	if pp.Types[id.PkgPath] != nil {
		panic("duplicated type:" + id.String())
	}
	pp.Types[id.Name] = f
}

func (p *goParser) getModuleFromPkg(pkg PkgPath) (name string, dir string) {
	for _, m := range p.modules {
		if strings.HasPrefix(pkg, m[0]) {
			return m[0], m[1]
		}
	}
	return "", ""
}

// path is absolute path
func (p *goParser) getModuleFromPath(path string) (name string, dir string) {
	for _, m := range p.modules {
		if strings.HasPrefix(path, m[1]) {
			return m[0], m[1]
		}
	}
	return "", ""
}

// FromABS converts an absolute path to local mod path
func (p *goParser) pkgPathFromABS(path string) PkgPath {
	mod, dir := p.getModuleFromPath(path)
	if mod == "" {
		panic("not found package from " + path)
	}
	if rel, err := filepath.Rel(dir, path); err != nil {
		panic("path " + path + " is not relative from mod path " + dir)
	} else {
		return filepath.Join(mod, rel)
	}
}

func (p *goParser) ParsePackage(pkgPath PkgPath) (err error) {
	mod, dir := p.getModuleFromPkg(pkgPath)
	if mod == "" {
		return fmt.Errorf("not found module for package %s", pkgPath)
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
	// fmt.Println("[ParsePackage] mod:", mod, "dir:", dir, "pkgPath:", pkgPath)

	// slow-path: load packages in the dir, including sub pakcages
	fset := token.NewFileSet()
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports,
		Fset: fset,
		Dir:  dir,
	}, pkgPath)
	if err != nil {
		return fmt.Errorf("load path '%s' failed: %v", dir, err)
	}

	for _, pkg := range pkgs {
		//TODO: only path single main package at present
		if pkg.ID != pkgPath {
			continue
		}
		for idx, file := range pkg.Syntax {
			filePath := pkg.GoFiles[idx]
			bs, err := ioutil.ReadFile(filePath)
			if err != nil {
				return err
			}
			sysImports, projectImports, thirdPartyImports, err := p.seprateImports(lib, file.Imports)
			if err != nil {
				return err
			}
			ctx := &fileContext{
				filePath:          filePath,
				module:            lib,
				pkgPath:           pkgPath,
				bs:                bs,
				fset:              fset,
				sysImports:        sysImports,
				projectImports:    projectImports,
				thirdPartyImports: thirdPartyImports,
				pkgTypeInfo:       pkg.TypesInfo,
			}
			if _, _, err := p.inspectFile(ctx, file); err != nil {
				return err
			}
		}
		if obj := lib.Packages[pkgPath]; obj != nil {
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

func getModuleName(modFilePath string) (string, []byte, error) {
	file, err := os.Open(modFilePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read file: %v", err)
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "module") {
			// Assuming 'module' keyword is followed by module name
			parts := strings.Split(line, " ")
			if len(parts) > 1 {
				return parts[1], data, nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", data, fmt.Errorf("failed to scan file: %v", err)
	}

	return "", data, nil
}

// parse go.mod and get a map of module name to module_path@version
func parseModuleFile(data []byte) (map[string]string, error) {
	ast, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod file: %v", err)
	}
	modules := make(map[string]string)
	for _, req := range ast.Require {
		modules[req.Mod.Path] = req.Mod.Path + "@" + req.Mod.Version
	}
	// replaces
	for _, replace := range ast.Replace {
		modules[replace.Old.Path] = replace.New.Path + "@" + replace.New.Version
	}
	return modules, nil
}
