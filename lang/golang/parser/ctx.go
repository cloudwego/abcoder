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
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/abcoder/lang/uniast"
	. "github.com/cloudwego/abcoder/lang/uniast"
	"golang.org/x/tools/go/packages"
)

var errSysImport = fmt.Errorf("sys import")

// The go file's context. Used to pass information between ast node handlers
type fileContext struct {
	repoDir        string
	filePath       string
	module         *Module
	pkgPath        PkgPath
	bs             []byte
	fset           *token.FileSet
	imports        *importInfo
	pkgTypeInfo    *types.Info
	deps           map[string]*packages.Package
	collectComment bool
}

func isExternalID(id *Identity, curmod string) bool {
	return strings.Contains(id.ModPath, "@") || id.ModPath != curmod ||
		// NOTICE: regard kitex_gen and hertz_gen as external
		strings.Contains(id.PkgPath, "/kitex_gen/") || strings.Contains(id.PkgPath, "/hertz_gen/")
}

const (
	StdLanguage = "go"
)

func newModule(mod string, dir string) (ret *Module) {
	ret = uniast.NewModule(mod, dir, Golang)
	return ret
}

func (p *GoParser) referCodes(ctx *fileContext, id *Identity, depth int) (err error) {
	if depth == 0 || id.PkgPath == "" || !isExternalID(id, ctx.module.Name) {
		return nil
	}
	// var kg bool
	// if strings.Contains(id.PkgPath, "/kitex_gen/") {
	// 	println("refering kitex_gen", id.Full())
	// 	kg = true
	// }
	// defer func() {
	// 	if err != nil && kg {
	// 		panic(err)
	// 	}
	// }()
	mod := p.repo.Modules[id.ModPath]
	if mod == nil {
		mod = newModule(id.ModPath, "")
		mod.Language = uniast.Golang
		p.repo.Modules[id.ModPath] = mod
	}
	// fmt.Printf("refer code for %v\n", id.Full())
	pkg := ctx.deps[id.PkgPath]
	if pkg == nil {
		return fmt.Errorf("cannot find package %s", id.PkgPath)
	}

	var files []string
	if len(p.cgoPkgs) > 0 {
		files = pkg.CompiledGoFiles
	} else {
		files = pkg.GoFiles
	}

	for _, fpath := range files {
		bs := p.getFileBytes(fpath)
		file, err := parser.ParseFile(pkg.Fset, fpath, bs, parser.ParseComments)
		if err != nil {
			return err
		}
		impts, e := p.parseImports(pkg.Fset, bs, mod, file.Imports)
		if e != nil {
			err = e
			continue
		}
		// println("search file", fpath)
		_, e = p.searchOnFile(file, pkg.Fset, bs, id.ModPath, pkg.ID, impts, id.Name)
		if e != nil {
			err = e
			continue
		}
	}
	return
}

func (p *GoParser) getFileBytes(path string) []byte {
	if bs, ok := p.files[path]; ok {
		return bs
	}
	bs, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	p.files[path] = bs
	return bs
}

func (ctx *fileContext) GetImportPath(alias string) (string, string, error) {
	return ctx.imports.GetImportPath(alias, ctx.module.Name)
}

func (impt importInfo) GetImportPath(alias string, mod string) (string, string, error) {
	if im, ok := impt.SysImports[alias]; ok {
		return im, "", errSysImport
	} else if im, ok := impt.ProjectImports[alias]; ok {
		return im, mod, nil
	} else if ims, ok := impt.ThirdPartyImports[alias]; ok {
		return ims[1], ims[0], nil
	} else {
		return "", "", fmt.Errorf("not found pkg: %s", alias)
	}
}

func (ctx *fileContext) GetMod(impt string) (string, error) {
	if impt == ctx.pkgPath {
		return ctx.module.Name, nil
	}
	if isSysPkg(impt) {
		return "", errSysImport
	}

	// fileContext 中的 import 信息只有**当前文件的引用路径**，但是存在一种场景就是实际调用的节点在另外的一个Package，导致漏解析
	// 常见于"链式调用"、"另一个 pkg 的全局变量的类型在另外一个 pkg 下"
	if ctx.module != nil && ctx.module.Packages != nil {
		if _, exist := ctx.module.Packages[impt]; exist {
			return ctx.module.Name, nil
		}
	}
	for _, ims := range ctx.imports.ProjectImports {
		if ims == impt {
			return ctx.module.Name, nil
		}
	}
	for _, ims := range ctx.imports.ThirdPartyImports {
		if ims[1] == impt {
			return ims[0], nil
		}
	}
	mod, _ := matchMod(impt, ctx.module.Dependencies)
	if mod == "" {
		return "", fmt.Errorf("not found mod for %s", impt)
	}
	return mod, nil
}

func (ctx *fileContext) FileLine(node ast.Node) FileLine {
	pos := ctx.fset.Position((node).Pos())
	rel, _ := filepath.Rel(ctx.repoDir, pos.Filename)
	end := ctx.fset.Position((node).End())
	ret := FileLine{File: rel, Line: pos.Line, StartOffset: pos.Offset, EndOffset: end.Offset}
	if _, ok := node.(*ast.TypeSpec); ok {
		// NOTICE: type spec is not the start of the type definition
		// so we need to adjust the offset = len("type ")
		ret.StartOffset -= 5
	}
	if ctx.collectComment {
		fset := ctx.fset
		switch v := node.(type) {
		case *ast.Field:
			if v.Doc != nil {
				ret.StartOffset = fset.Position(v.Doc.Pos()).Offset
			}
		case *ast.GenDecl:
			if v.Doc != nil {
				ret.StartOffset = fset.Position(v.Doc.Pos()).Offset
			}
		case *ast.TypeSpec:
			if v.Doc != nil {
				ret.StartOffset = fset.Position(v.Doc.Pos()).Offset
			}
		case *ast.ValueSpec:
			if v.Doc != nil {
				ret.StartOffset = fset.Position(v.Doc.Pos()).Offset
			}
		case *ast.FuncDecl:
			if v.Doc != nil {
				ret.StartOffset = fset.Position(v.Doc.Pos()).Offset
			}
		}
	}
	return ret
}

func (ctx *fileContext) GetRawContent(node ast.Node) []byte {
	return GetRawContent(ctx.fset, ctx.bs, node, ctx.collectComment)
}

func GetRawContent(fset *token.FileSet, file []byte, node ast.Node, collectComment bool) []byte {
	var doc = bytes.Buffer{}
	switch v := node.(type) {
	case *ast.Field:
		if collectComment && v.Doc != nil {
			doc.Write(file[fset.Position(v.Doc.Pos()).Offset:fset.Position(v.Doc.End()).Offset])
			doc.WriteByte('\n')
		}
	case *ast.GenDecl:
		if collectComment && v.Doc != nil {
			doc.Write(file[fset.Position(v.Doc.Pos()).Offset:fset.Position(v.Doc.End()).Offset])
			doc.WriteByte('\n')
		}
	case *ast.TypeSpec:
		if collectComment && v.Doc != nil {
			doc.Write(file[fset.Position(v.Doc.Pos()).Offset:fset.Position(v.Doc.End()).Offset])
			doc.WriteByte('\n')
		}
		// NOTICE: type spec is not the start of the type definition
		// so we need to add "type "
		doc.WriteString("type ")
	case *ast.ValueSpec:
		if collectComment && v.Doc != nil {
			doc.Write(file[fset.Position(v.Doc.Pos()).Offset:fset.Position(v.Doc.End()).Offset])
			doc.WriteByte('\n')
		}
	case *ast.FuncDecl:
		if collectComment && v.Doc != nil {
			doc.Write(file[fset.Position(v.Doc.Pos()).Offset:fset.Position(v.Doc.End()).Offset])
			doc.WriteByte('\n')
		}
	}
	doc.Write(file[fset.Position(node.Pos()).Offset:fset.Position(node.End()).Offset])
	return doc.Bytes()
}

// func (ctx *fileContext) GetRaw(from token.Pos, to token.Pos) []byte {
// 	return ctx.bs[ctx.fset.Position(from).Offset:ctx.fset.Position(to).Offset]
// }

type typeInfo struct {
	Id             Identity
	IsNamed        bool
	IsPointer      bool
	IsStdOrBuiltin bool
	Deps           []Identity
	Ty             types.Type
}

// FIXME: for complex type like map[XX]YY , we only extract first-meet type here
func (ctx *fileContext) GetTypeInfo(typ ast.Expr) typeInfo {
	if tinfo, ok := ctx.pkgTypeInfo.Types[typ]; ok {
		return ctx.getTypeinfo(tinfo.Type)
	} else {
		// NOTICE: for unloaded type, we only mock the type name
		fmt.Fprintf(os.Stderr, "cannot find type info for %s\n", ctx.GetRawContent(typ))
		return ctx.mockType(typ)
	}
}

func (ctx *fileContext) mockType(typ ast.Expr) (ti typeInfo) {
	switch ty := typ.(type) {
	case *ast.StarExpr:
		ti = ctx.mockType(ty.X)
		ti.IsPointer = true
		return
	case *ast.CallExpr:
		// try get func type
		ti = ctx.mockType(ty.Fun)
		ti.IsPointer = false
		return
	case *ast.SelectorExpr:
		// try get import path
		switch xx := ty.X.(type) {
		case *ast.Ident:
			impt, mod, err := ctx.imports.GetImportPath(xx.Name, "")
			if err != nil {
				goto fallback
			}
			ti.Id = NewIdentity(mod, PkgPath(impt), ty.Sel.Name)
			return
		case *ast.SelectorExpr:
			// recurse
			ti = ctx.mockType(xx)
			ti.Id.Name = ty.Sel.Name
			ti.IsPointer = false
			return ti
		}
	}

fallback:
	ti.Id = NewIdentity("UNLOADED", ctx.pkgPath, string(ctx.GetRawContent(typ)))
	ti.IsStdOrBuiltin = true
	return
}

func (ctx *fileContext) collectFields(fields []*ast.Field, m *[]Dependency) {
	for _, fieldDecl := range fields {
		ti := ctx.GetTypeInfo(fieldDecl.Type)
		if !ti.IsStdOrBuiltin && ti.Id.ModPath != "" {
			*m = InsertDependency(*m, Dependency{
				Identity: ti.Id,
				FileLine: ctx.FileLine(fieldDecl),
			})
		}
		for _, dep := range ti.Deps {
			*m = InsertDependency(*m, Dependency{
				Identity: dep,
				FileLine: ctx.FileLine(fieldDecl),
			})
		}
	}
}

type importInfo struct {
	SysImports        map[string]string
	ProjectImports    map[string]string
	ThirdPartyImports map[string][2]string // 0-mod, 1-import
	Origins           []Import
}

func (p *GoParser) mockTypes(typ ast.Expr, m map[string]Identity, file []byte, fset *token.FileSet, fpath string, mod string, pkg string, impts *importInfo) (name string, isPointer bool) {
	ids, _, isP := getTypeName(fset, file, typ)
	for _, id := range ids {
		// NOTICE: mock all types in the module
		if id.PkgPath != "" {
			impt, m, err := impts.GetImportPath(id.PkgPath, mod)
			if err != nil && err != errSysImport {
				fmt.Fprintf(os.Stderr, "cannot get import path for "+ids[0].PkgPath+": "+err.Error())
			} else if err == errSysImport {
				continue
			}
			id.PkgPath = impt
			// FIXME: cannot get the third-party mod here
			if m != "" {
				id.ModPath = m
			} else {
				id.ModPath = mod
			}
		} else {
			id.PkgPath = pkg
			id.ModPath = mod
		}
		// println("add new type", id.Full())
		// may not be within the same package, thus set receiver too
		if n := p.repo.GetType(id); n == nil {
			st := p.newType(id.ModPath, id.PkgPath, id.Name)
			st.Exported = isUpperCase(id.Name[0])
			st.File = fpath
			st.Line = fset.Position(typ.Pos()).Line - 1 // not real
			// FIXME: cannot get specific entity's definition unless load the whole package
			st.Content = "type " + id.Name + " struct{}"
		}

		// not real
		if name == "" && id.Name != "" {
			name = id.Name
		}
		m[id.Name] = id
	}
	return name, isP
}

// handle typ expr and return not-builtin type identity and return if the type if a func signature.
// ret is used to store results.
func getTypeName(fset *token.FileSet, file []byte, typ ast.Expr) (ret []Identity, isFunc bool, isPointer bool) {
	switch ty := typ.(type) {
	case *ast.Ident:
		if !isGoBuiltins(ty.Name) {
			ret = append(ret, Identity{Name: ty.Name})
		}
		return
	case *ast.IndexExpr: // generic type parameter
		ret = append(ret, Identity{Name: ty.X.(*ast.Ident).Name})
	case *ast.IndexListExpr: // generic type parameter
		ret = append(ret, Identity{Name: ty.X.(*ast.Ident).Name})
	case *ast.StarExpr:
		id, _, _ := getTypeName(fset, file, ty.X)
		ret = append(ret, id...)
		isPointer = true
		return
	case *ast.ArrayType:
		id, _, _ := getTypeName(fset, file, ty.Elt)
		ret = append(ret, id...)
		return
	case *ast.MapType:
		a, _, _ := getTypeName(fset, file, ty.Key)
		ret = append(ret, a...)
		b, _, _ := getTypeName(fset, file, ty.Value)
		ret = append(ret, b...)
		return
	case *ast.ChanType:
		id, _, _ := getTypeName(fset, file, ty.Value)
		ret = append(ret, id...)
		return
	case *ast.SelectorExpr:
		pkg, ok := ty.X.(*ast.Ident)
		if ok {
			ret = append(ret, Identity{Name: ty.Sel.Name, PkgPath: pkg.Name})
		}
		return
	case *ast.FuncType:
		start := ty.Pos()
		name := string(file[fset.Position(start).Offset:fset.Position(typ.End()).Offset])
		ret = append(ret, Identity{Name: name})
		isFunc = true
		return
	case *ast.InterfaceType:
		name := string(file[fset.Position(ty.Interface).Offset:fset.Position(typ.End()).Offset])
		if name == "interface{}" {
			return
		}
		ret = append(ret, Identity{Name: name})
		return
	case *ast.Ellipsis:
		id, _, _ := getTypeName(fset, file, ty.Elt)
		ret = append(ret, id...)
		return
	}
	return
}

func (p *GoParser) collectTypes(ctx *fileContext, typ ast.Expr, st *Type, inlined bool) {
	ti := ctx.GetTypeInfo(typ)
	if !ti.IsStdOrBuiltin && ti.Id.ModPath != "" {
		dep := NewDependency(ti.Id, ctx.FileLine(typ))
		if err := p.referCodes(ctx, &ti.Id, p.opts.ReferCodeDepth); err != nil {
			fmt.Fprintf(os.Stderr, "failed to get refer code for %s: %v\n", ti.Id, err)
		}
		if inlined {
			st.InlineStruct = InsertDependency(st.InlineStruct, dep)
		} else {
			st.SubStruct = InsertDependency(st.SubStruct, dep)
		}
	}
	for _, dep := range ti.Deps {
		if err := p.referCodes(ctx, &dep, p.opts.ReferCodeDepth); err != nil {
			fmt.Fprintf(os.Stderr, "failed to get refer code for %s: %v\n", dep, err)
		}
		st.SubStruct = InsertDependency(st.SubStruct, NewDependency(dep, ctx.FileLine(typ)))
	}
}

// get type id and tells if it is std or builtin
func (ctx *fileContext) getTypeinfo(typ types.Type) (ti typeInfo) {
	visited := make(map[types.Type]bool)
	tobjs, isPointer, isNamed := getNamedTypes(typ, visited)
	ti.IsPointer = isPointer
	ti.Ty = typ
	ti.IsNamed = isNamed
	// NOTICE: only get full id for Named type
	if isNamed {
		tobj := tobjs[0]
		if tp := tobj.Pkg(); tp != nil {
			mod, err := ctx.GetMod(tp.Path())
			if err == errSysImport {
				ti.Id = Identity{"", tp.Path(), tobj.Name()}
				ti.IsStdOrBuiltin = true
			} else if err != nil || mod == "" {
				// unloaded type, mark it
				ti.Id = Identity{"", tp.Path(), tobj.Name()}
				ti.IsStdOrBuiltin = false
			} else {
				ti.Id = NewIdentity(mod, tp.Path(), tobj.Name())
				ti.IsStdOrBuiltin = false
			}
		} else {
			if isGoBuiltins(tobj.Name()) {
				ti.Id = Identity{Name: tobj.Name()}
				ti.IsStdOrBuiltin = true
			} else {
				// unloaded type, mark it
				ti.Id = Identity{"", ctx.pkgPath, tobj.Name()}
				ti.IsStdOrBuiltin = false
			}
		}
	} else {
		// Notice: for Composite type like map, slice, regard it as builtin
		ti.Id = Identity{"", "", typ.String()}
		ti.IsStdOrBuiltin = true
	}
	// collect sub Named type here
	i := 0
	if isNamed {
		i = 1
	}
	for ; i < len(tobjs); i++ {
		tobj := tobjs[i]
		if isGoBuiltins(tobj.Name()) {
			continue
		}
		// get mod and pkg from tobj.Pkg()
		tp := tobj.Pkg()
		if tp == nil {
			continue
		}
		mod, err := ctx.GetMod(tp.Path())
		if err != nil || mod == "" {
			continue
		}
		ti.Deps = append(ti.Deps, NewIdentity(mod, tp.Path(), tobj.Name()))
	}
	return
}

func (ctx *fileContext) IsSysImport(alias string) bool {
	_, ok := ctx.imports.SysImports[alias]
	return ok
}
