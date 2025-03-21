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
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/abcoder/src/uniast"
	. "github.com/cloudwego/abcoder/src/uniast"
	"golang.org/x/tools/go/packages"
)

var errSysImport = fmt.Errorf("sys import")

// The go file's context. Used to pass information between ast node handlers
type fileContext struct {
	repoDir     string
	filePath    string
	module      *Module
	pkgPath     PkgPath
	bs          []byte
	fset        *token.FileSet
	imports     *importInfo
	pkgTypeInfo *types.Info
	deps        map[string]*packages.Package
}

func (ctx *fileContext) FileLine(node ast.Node) FileLine {
	pos := ctx.fset.Position((node).Pos())
	rel, _ := filepath.Rel(ctx.repoDir, pos.Filename)
	return FileLine{File: rel, Line: pos.Line}
}

func isExternalID(id *Identity, curmod string) bool {
	return strings.Contains(id.ModPath, "@") || id.ModPath != curmod ||
		// NOTICE: regard kitex_gen and hertz_gen as external
		strings.Contains(id.PkgPath, "/kitex_gen/") || strings.Contains(id.PkgPath, "/hertz_gen/")
}

func newModule(mod string, dir string) *Module {
	ret := uniast.NewModule(mod, dir)
	ret.Language = Golang
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
	for i, fpath := range pkg.GoFiles {
		file := pkg.Syntax[i]
		bs := p.getFileBytes(fpath)
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
	// try find self first
	if strings.HasPrefix(impt, ctx.module.Name) {
		return ctx.module.Name, nil
	}
	// try find in go.mod
	for dep, ver := range ctx.module.Dependencies {
		if strings.HasPrefix(impt, dep) {
			return ver, nil
		}
	}
	return "", fmt.Errorf("not found mod: %s", impt)
}

func (ctx *fileContext) GetRawContent(node ast.Node) []byte {
	return GetRawContent(ctx.fset, ctx.bs, node)
}

func GetRawContent(fset *token.FileSet, file []byte, node ast.Node) []byte {
	var doc = bytes.Buffer{}
	switch v := node.(type) {
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

// FIXME: for complex type like map[XX]YY , we only extract first-meet type here
func (ctx *fileContext) GetTypeId(typ ast.Expr) (x Identity, isPointer bool, isStdOrBuiltin bool) {
	if tinfo, ok := ctx.pkgTypeInfo.Types[typ]; ok {
		return ctx.getIdFromType(tinfo.Type)
	} else {
		panic("cannot find type info for " + string(ctx.GetRawContent(typ)))
	}
}

func (ctx *fileContext) collectFields(fields []*ast.Field, m *[]Dependency) {
	for _, fieldDecl := range fields {
		id, _, isStdOrBuiltin := ctx.GetTypeId(fieldDecl.Type)
		if isStdOrBuiltin || id.PkgPath == "" {
			continue
		}
		*m = append(*m, Dependency{
			Identity: id,
			FileLine: ctx.FileLine(fieldDecl),
		})
	}
	return
}

type importInfo struct {
	SysImports        map[string]string
	ProjectImports    map[string]string
	ThirdPartyImports map[string][2]string // 0-mod, 1-import
	Origins           []string
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
			st.Line = fset.Position(typ.Pos()).Line // not real
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
	id, _, isGoBuiltins := ctx.GetTypeId(typ)
	dep := NewDependency(id, ctx.FileLine(typ))
	if isGoBuiltins || id.PkgPath == "" {
		return
	}
	if err := p.referCodes(ctx, &id, p.opts.ReferCodeDepth); err != nil {
		fmt.Fprintf(os.Stderr, "failed to get refer code for %s: %v\n", id.Name, err)
	}
	if inlined {
		st.InlineStruct = append(st.InlineStruct, dep)
	} else {
		st.SubStruct = append(st.SubStruct, dep)
	}
}

var compositeTypePrefixs = []string{"[]", "map[", "chan ", "<-chan", "chan<-", "func("}

// get type id and tells if it is std or builtin
func (ctx *fileContext) getIdFromType(typ types.Type) (x Identity, isPointer bool, isStrOrBuiltin bool) {
	if tobj, isPointer := getNamedType(typ); tobj != nil {
		if isGoBuiltins(tobj.Name()) {
			return Identity{Name: tobj.Name()}, isPointer, true
		}
		name := tobj.Name()
		// NOTICE: filter composite type (map[] slice func chan ...)
		// TODO: support extract sub named type
		for _, prefix := range compositeTypePrefixs {
			if strings.HasPrefix(name, prefix) {
				return Identity{Name: name}, isPointer, true
			}
		}
		// get mod and pkg from tobj.Pkg()
		tp := tobj.Pkg()
		if tp == nil {
			return NewIdentity(ctx.module.Name, ctx.pkgPath, name), isPointer, false
		}
		mod, err := ctx.GetMod(tp.Path())
		if err == errSysImport {
			return Identity{Name: name, PkgPath: tp.Path()}, isPointer, true
		} else if err != nil {
			return Identity{Name: name}, isPointer, false
		}
		return NewIdentity(mod, tp.Path(), tobj.Name()), isPointer, false
	} else {
		typStr := typ.String()
		isPointer := strings.HasPrefix(typStr, "*")
		typStr = strings.TrimPrefix(typStr, "*")
		if isGoBuiltins(typStr) {
			return Identity{Name: typStr}, isPointer, true
		}
		for _, prefix := range compositeTypePrefixs {
			if strings.HasPrefix(typStr, prefix) {
				return Identity{Name: typStr}, isPointer, true
			}
		}
		if idx := strings.LastIndex(typStr, "."); idx > 0 {
			pkg := typStr[:idx]
			if isSysPkg(pkg) {
				return Identity{Name: typStr[idx+1:], PkgPath: pkg}, isPointer, true
			}
			// FIXME: some types (ex: return type of a func-calling) cannot be found go mod here.
			// Ignore empty mod for now.
			mod, _ := ctx.GetMod(pkg)
			return NewIdentity(mod, pkg, typStr[idx+1:]), isPointer, false
		} else {
			return NewIdentity(ctx.module.Name, ctx.pkgPath, typStr), isPointer, false
		}
	}
}

func (ctx *fileContext) IsSysImport(alias string) bool {
	_, ok := ctx.imports.SysImports[alias]
	return ok
}
