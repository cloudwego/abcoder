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
	"fmt"
	"go/token"
	"io/ioutil"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Repository
type Repository struct {
	ModName  string               // go module name
	Packages map[PkgPath]*Package // pkage import path => Package
}

func NewRepository(mod string) Repository {
	return Repository{ModName: mod, Packages: map[PkgPath]*Package{}}
}

// Package
type Package struct {
	Functions map[string]*Function // Function name (may be {{func}} or {{struct.method}}) => Function
	Types     map[string]*Struct   // type name => type define
}

// GetFunction the function identified by id.
// if id indicates a method, it will try traceinto inlined sub structs to get the named method
func (p Repository) GetFunction(id Identity) *Function {
	if pkg, ok := p.Packages[id.PkgPath]; ok {
		if f := pkg.Functions[id.Name]; f != nil {
			return f
		}
	}
	return nil
}

func (p *Repository) SetFunction(id Identity, f *Function) {
	pp, ok := p.Packages[id.PkgPath]
	if !ok {
		pp = &Package{
			Functions: map[string]*Function{},
			Types:     map[string]*Struct{},
		}
		p.Packages[id.PkgPath] = pp
	}
	if pp.Functions[id.Name] != nil {
		// FIXME
		panic("duplicated function:" + id.String())
	}
	pp.Functions[id.Name] = f
}

func (p Repository) GetType(id Identity) *Struct {
	if pkg, ok := p.Packages[id.PkgPath]; ok {
		return pkg.Types[id.Name]
	}
	return nil
}

func (p *Repository) SetType(id Identity, f *Struct) {
	pp, ok := p.Packages[id.PkgPath]
	if !ok {
		pp = &Package{
			Functions: map[string]*Function{},
			Types:     map[string]*Struct{},
		}
		p.Packages[id.PkgPath] = pp
	}
	if pp.Types[id.PkgPath] != nil {
		panic("duplicated type:" + id.String())
	}
	pp.Types[id.Name] = f
}

func (p *goParser) ParseDir(dir string) (err error) {
	if !strings.HasPrefix(dir, "/") {
		dir = filepath.Join(p.homePageDir, dir)
	}
	// fast-path: check cache first
	pkgPath := p.pkgPathFromABS(dir)
	if p.visited[pkgPath] {
		return nil
	}
	p.visited[pkgPath] = true

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
			sysImports, projectImports, thirdPartyImports := p.seprateImports(file.Imports)
			ctx := &fileContext{
				filePath:          filePath,
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
	}
	return
}
