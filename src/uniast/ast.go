/**
 * Copyright 2025 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package uniast

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func (p *Module) GetDependency(pkg string) string {
	// // search internal library first
	// if lib := p.Libraries[mod]; lib != nil {
	// 	return lib
	// }
	// match the prefix of name for each repo.Dependencies
	for k, v := range p.Dependencies {
		if strings.HasPrefix(pkg, k) {
			return v
		}
	}
	// FIXME: return value's dependency may not explicitly defined in go.mod, thus may not be found
	// fmt.Fprintf(os.Stderr, "Error: not found dependency for %v", pkg)
	return ""
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

// Function holds the information about a function
type Function struct {
	Exported bool

	IsMethod bool // If the function is a method
	Identity      // unique identity in a repo
	FileLine
	Content string // Content of the function, including functiion signature and body

	Receiver *Receiver  `json:",omitempty"` // Method receiver
	Params   []Identity `json:",omitempty"` // function parameters, key is the parameter name
	Results  []Identity `json:",omitempty"` // function results, key is the result name or type name

	// call to in-the-project functions, key is {{pkgAlias.funcName}} or {{funcName}}
	FunctionCalls []Identity `json:",omitempty"`

	// call to internal methods,
	// NOTICE: method name may be duplicated, so we collect according to the SEQUENCE of APPEARANCE
	MethodCalls []Identity `json:",omitempty"`

	Types       []Identity `json:",omitempty"` // types used in the function
	GolobalVars []Identity `json:",omitempty"` // global vars used in the function

	// func llm compress result
	CompressData *string `json:"compress_data,omitempty"`
}

type Receiver struct {
	IsPointer bool
	Type      Identity
	Name      string
}

// FileLine represents a filename and line number
type FileLine struct {
	File string

	// NOTICE: line number start from 1
	Line int
}

type TypeKind int

const (
	TypeKindStruct    = 0 // type struct
	TypeKindInterface = 1 // type interface
	TypeKindNamed     = 2 // type NamedXXX other..
	TypeKindEnum      = 3 // type NamedXXX other..
)

// Type holds the information about a struct
type Type struct {
	Exported bool // if the struct is exported

	TypeKind // type Kind: Struct / Interface / Typedef
	Identity // unique id in a repo
	FileLine
	Content string // struct declaration content

	// field type (not include basic types), type name => type id
	SubStruct []Identity `json:",omitempty"`

	// inline field type (not include basic types)
	InlineStruct []Identity `json:",omitempty"`

	// methods defined on the Struct, not including inlined type's method
	Methods map[string]Identity `json:",omitempty"`

	// Implemented interfaces
	Implements []Identity `json:",omitempty"`

	// functions defined in fields, key is type name, val is the function Signature
	// FieldFunctions map[string]string

	CompressData *string `json:"compress_data,omitempty"` // struct llm compress result
}

type Var struct {
	IsExported bool
	IsConst    bool
	Identity
	FileLine
	Type    *Identity `json:",omitempty"`
	Content string

	CompressData *string `json:"compress_data,omitempty"`
}

type Dependency struct {
	Range [2]int32
	Identity
}
