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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Language string

const (
	Golang  Language = "go"
	Rust    Language = "rust"
	Cxx     Language = "cxx"
	Unknown Language = ""
)

func (l Language) String() string {
	switch l {
	case Rust:
		return "rust"
	case Golang:
		return "go"
	case Cxx:
		return "cxx"
	default:
		return string(l)
	}
}

func NewLanguage(lang string) (l Language) {
	// sp := strings.Split(lang, "@")
	// if len(sp) > 1 {
	// 	lang = sp[0]
	// 	version = sp[1]
	// }
	switch strings.ToLower(lang) {
	case "go", "golang":
		return Golang
	case "rust":
		return Rust
	case "cxx":
		return Cxx
	default:
		return Unknown
	}
}

// Node ID (string) => Node
// Node ID comes from Identity.Full()
type NodeGraph map[string]*Node

// Repository
type Repository struct {
	Name    string             `json:"id"` // module name
	Modules map[string]*Module // module name => module
	Graph   NodeGraph          // node id => node
}

func (r Repository) ID() string {
	return r.Name
}

func (r Repository) InternalModules() []*Module {
	var ret []*Module
	for _, v := range r.Modules {
		if !v.IsExternal() {
			ret = append(ret, v)
		}
	}
	return ret
}

func NewRepository(name string) Repository {
	ret := Repository{
		Name:    name,
		Modules: map[string]*Module{},
		Graph:   map[string]*Node{},
	}
	return ret
}

type File struct {
	Path    string
	Imports []Import `json:",omitempty"`
	Package *PkgPath `json:",omitempty"`
}

type Import struct {
	Alias *string `json:",omitempty"`
	Path  string
}

func (i *Import) UnmarshalJSON(data []byte) error {
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		v, e := strconv.Unquote(string(data))
		if e != nil {
			return e
		}
		i.Path = v
		return nil
	} else {
		var ii importObj
		err := json.Unmarshal(data, &ii)
		if err != nil {
			return err
		}
		i.Alias = ii.Alias
		i.Path = ii.Path
		return nil
	}
}

type importObj Import

func NewImport(alias *string, path string) Import {
	return Import{
		Alias: alias,
		Path:  path,
	}
}

func (i Import) Equals(other Import) bool {
	if i.Alias == nil && other.Alias == nil {
		return i.Path == other.Path
	}
	if i.Alias == nil || other.Alias == nil {
		return false
	}
	return *i.Alias == *other.Alias && i.Path == other.Path
}

func NewFile(path string) *File {
	// abs, _ := filepath.Abs(path)
	ret := File{
		Path: path,
	}
	return &ret
}

func (m Module) SetFile(path string, file *File) {
	if m.Files == nil {
		m.Files = map[string]*File{}
	}
	if m.Files[path] == nil {
		m.Files[path] = file
	}
}

type Module struct {
	Language     Language
	Version      string
	Name         string               // go module name
	Dir          string               // relative path to repo
	Packages     map[PkgPath]*Package // pkage import path => Package
	Dependencies map[string]string    // module name => module_path@version
	Files        map[string]*File     // relative path => file info
	CompressData *string              `json:"compress_data,omitempty"` // module compress info
}

// func (r Repository) GetFileById(id Identity) *File {
// 	mod := r.Modules[id.ModPath]
// 	if mod == nil {
// 		return nil
// 	}
// 	node := r.GetNode(id)
// 	if node == nil {
// 		return nil
// 	}
// 	return mod.Files[node.FileLine().File]
// }

func (m Module) GetFile(path string) *File {
	return m.Files[path]
}

func (m Module) IsExternal() bool {
	return m.Dir == ""
}

func NewModule(name string, dir string, language Language) *Module {
	var v string
	sp := strings.Split(name, "@")
	name = sp[0]
	if len(sp) > 1 {
		v = sp[1]
	}
	ret := Module{
		Version:      v,
		Language:     language,
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
	IsTest bool
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

type ModPath = string

func ModPathName(mod ModPath) string {
	if strings.Contains(mod, "@") {
		return strings.Split(mod, "@")[0]
	}
	return mod
}

// Identity is the universal-unique for an ast node.
type Identity struct {
	// module id, must be unique within a repo
	ModPath `json:"ModPath" jsonschema:"description=the compiling module of the ast node, the format is {ModName} or {ModName}@{Version}"`

	// path id, must be unique within a module
	PkgPath `json:"PkgPath" jsonschema:"description=the namespace of the ast node"`

	// symbol id , must be unique within a package
	Name string `json:"Name" jsonschema:"description=unique name of the ast node, the format is one of {FunctionName}, {TypeName}.{MethodName}, {InterfaceName}<{TypeName}>.{MethodName}, {TypeName}"`
}

func NewIdentity(mod, pkg, name string) Identity {
	if mod == "" {
		fmt.Fprintf(os.Stderr, "module name cannot be empty: %s#%s\n", pkg, name)
		// panic(fmt.Sprintf("module name cannot be empty: %s.%s", pkg, name))
	}
	return Identity{ModPath: mod, PkgPath: pkg, Name: name}
}

func NewIdentityFromString(str string) (ret Identity) {
	sp := strings.Split(str, "?")
	if len(sp) == 2 {
		ret.ModPath = sp[0]
		str = sp[1]
	}
	sp = strings.Split(str, "#")
	if len(sp) == 2 {
		ret.PkgPath = sp[0]
		str = sp[1]
	}
	ret.Name = str
	return ret
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
		panic(fmt.Sprintf("must set module before set func"))
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

func (p *Repository) SetModule(path string, mod *Module) {
	if p.Modules == nil {
		p.Modules = map[string]*Module{}
	}
	p.Modules[path] = mod
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
		panic(fmt.Sprintf("must set module before set type"))
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
		panic(fmt.Sprintf("must set module before set var"))
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

	IsMethod          bool // If the function is a method
	IsInterfaceMethod bool // If is a empty interface method stub
	Identity               // unique identity in a repo
	FileLine
	Content string // Content of the function, including functiion signature and body

	Receiver *Receiver    `json:",omitempty"` // Method receiver
	Params   []Dependency `json:",omitempty"` // function parameters, key is the parameter name
	Results  []Dependency `json:",omitempty"` // function results, key is the result name or type name

	// call to in-the-project functions, key is {{pkgAlias.funcName}} or {{funcName}}
	FunctionCalls []Dependency `json:",omitempty"`

	// call to internal methods,
	// NOTICE: method name may be duplicated, so we collect according to the SEQUENCE of APPEARANCE
	MethodCalls []Dependency `json:",omitempty"`

	Types      []Dependency `json:",omitempty"` // types used in the function
	GlobalVars []Dependency `json:",omitempty"` // global vars used in the function

	// func llm compress result
	CompressData *string `json:"compress_data,omitempty"`
}

type Dependency struct {
	Identity
	FileLine `json:",omitempty"`
}

func (d Dependency) Id() Identity {
	return d.Identity
}

func NewDependency(id Identity, fl FileLine) Dependency {
	return Dependency{
		Identity: id,
		FileLine: fl,
	}
}

type Receiver struct {
	IsPointer bool
	Type      Identity
}

// FileLine represents a filename and line number
type FileLine struct {
	File string

	// NOTICE: start line. line number start from 1
	Line int

	// start offset in file
	StartOffset int

	// end offset in file
	EndOffset int
}

type TypeKind string

const (
	TypeKindStruct    TypeKind = "struct"
	TypeKindInterface TypeKind = "interface"
	TypeKindTypedef   TypeKind = "typedef"
	TypeKindEnum      TypeKind = "enum"
)

func (t *TypeKind) UnmarshalJSON(data []byte) error {
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		*t = TypeKind(data[1 : len(data)-1])
		return nil
	}

	// 兼容历史go ast
	switch string(data) {
	case "0":
		*t = TypeKindStruct
	case "1":
		*t = TypeKindInterface
	default:
		*t = TypeKindTypedef
	}
	return nil
}

// const (
// 	TypeKindStruct    = 0 // type struct
// 	TypeKindInterface = 1 // type interface
// 	TypeKindNamed     = 2 // type NamedXXX other..
// 	TypeKindEnum      = 3 // type NamedXXX other..
// )

// Type holds the information about a struct
type Type struct {
	Exported bool // if the struct is exported

	TypeKind TypeKind // type Kind: Struct / Interface / Typedef

	Identity // unique id in a repo
	FileLine
	Content string // struct declaration content

	// field type (not include basic types), type name => type id
	SubStruct []Dependency `json:",omitempty"`

	// inline field type (not include basic types)
	InlineStruct []Dependency `json:",omitempty"`

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

	IsConst   bool
	IsPointer bool // if its Type is a pointer type
	Identity
	FileLine
	Type         *Identity `json:",omitempty"`
	Content      string
	Dependencies []Dependency `json:",omitempty"`

	CompressData *string `json:"compress_data,omitempty"`
}
