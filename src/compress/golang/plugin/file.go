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
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// PkgPath is the import path of a package, it is either absolute path or url
type PkgPath = string

// Identity holds identity information about a third party declaration
type Identity struct {
	ModPath string // ModPath is the module which the package belongs to
	PkgPath        // Import Path of the third party package
	Name    string // Unique Name of declaration (FunctionName, StructName.MethodName, or StructName)
}

func NewIdentity(mod, pkg, name string) Identity {
	if mod == "" {
		panic("module name cannot be empty: " + pkg + "." + name)
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

// Function holds the information about a function
type Function struct {
	Exported bool

	IsMethod         bool      // If the function is a method
	Identity                   // unique identity in a repo
	FilePath         string    `json:"-"` // File where the function is defined, empty if the function declaration is not scanned
	Content          string    // Content of the function, including functiion signature and body
	AssociatedStruct *Identity // Method receiver

	// call to in-the-project functions, key is {{pkgAlias.funcName}} or {{funcName}}
	FunctionCalls map[string]Identity

	// call to internal methods, key is the {{object.funcName}}
	MethodCalls map[string]Identity
}

func isGoBuiltinFunc(name string) bool {
	switch name {
	case "append", "cap", "close", "complex", "copy", "delete", "imag", "len", "make", "new", "panic", "print", "println", "real", "recover":
		return true
	case "string", "bool", "byte", "complex64", "complex128", "error", "float32", "float64", "int", "int8", "int16", "int32", "int64", "rune", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return true
	default:
		return false
	}
}

func (p *goParser) inspectFile(ctx *fileContext, f *ast.File) (map[string]*Function, map[string]*Type, error) {

	fileStructs := map[string]*Type{}
	fileFuncs := map[string]*Function{}
	cont := true
	ast.Inspect(f, func(node ast.Node) bool {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "panic: %v in %s:%d\n", r, ctx.filePath, ctx.fset.Position(node.Pos()).Line)
				return
			}
		}()
		if funcDecl, ok := node.(*ast.FuncDecl); ok {
			// parse funcs
			f, ct := p.parseFunc(ctx, funcDecl)
			fileFuncs[f.Name] = f
			cont = ct
		} else if typDecl, ok := node.(*ast.TypeSpec); ok {
			name := typDecl.Name.Name
			var st *Type
			var ct bool

			switch ty := typDecl.Type.(type) {
			case *ast.StructType:
				// struct decl
				st, ct = p.parseStruct(ctx, name, ty)
			case *ast.InterfaceType:
				// interface decl
				st, ct = p.parseInterface(ctx, name, ty)
			default:
				// typedef, ex: type Str StructA
				st := p.newStruct(ctx.module.Name, ctx.pkgPath, name)
				st.Exported = isUpperCase(name[0])
				st.TypeKind = TypeKindNamed
				st.Content = "type " + string(ctx.GetRaw(typDecl.Name.Pos(), typDecl.End()))
				p.collectTypes(ctx, "", typDecl.Type, st, typDecl.Assign.IsValid())
				ct = true
			}

			fileStructs[name] = st
			cont = ct
		}
		return cont
	})
	return fileFuncs, fileStructs, nil
}

// newFunc allocate a function in the repo
func (p *goParser) newFunc(mod, pkg, name string) *Function {
	ret := &Function{Identity: NewIdentity(mod, pkg, name)}
	p.repo.SetFunction(ret.Identity, ret)
	return ret
}

// newStruct allocate a struct in the repo
func (p *goParser) newStruct(mod, pkg, name string) *Type {
	ret := &Type{Identity: NewIdentity(mod, pkg, name)}
	p.repo.SetType(ret.Identity, ret)
	return ret
}

func (p *goParser) seprateImports(mod *Module, impts []*ast.ImportSpec) (map[string]string, map[string]string, map[string][2]string, error) {
	thirdPartyImports := make(map[string][2]string)
	projectImports := make(map[string]string)
	sysImports := make(map[string]string)
	for _, imp := range impts {
		importPath := imp.Path.Value[1 : len(imp.Path.Value)-1] // remove the quotes
		importAlias := getPackageAlias(importPath)
		// Check if user has defined an alias for current import
		if imp.Name != nil {
			importAlias = imp.Name.Name // update the alias
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
				if mod == "" {
					return nil, nil, nil, fmt.Errorf("unknown third party package: " + importPath)
				}
				thirdPartyImports[importAlias] = [2]string{mod, importPath}
			}
		}
	}
	return sysImports, projectImports, thirdPartyImports, nil
}

// parseFunc parses all function declaration in one file
func (p *goParser) parseFunc(ctx *fileContext, funcDecl *ast.FuncDecl) (*Function, bool) {
	var associatedStruct *Identity
	isMethod := funcDecl.Recv != nil
	if isMethod {
		var structName []Identity
		// TODO: reserve the pointer message?
		_ = getTypeName(ctx.fset, ctx.bs, funcDecl.Recv.List[0].Type, &structName)
		if len(structName) == 0 {
			panic("cannot get receiver's type:" + string(ctx.GetRawContent(funcDecl.Recv.List[0].Type)))
		}
		tm := NewIdentity(ctx.module.Name, ctx.pkgPath, structName[0].Name)
		associatedStruct = &tm
	}

	content := string(ctx.GetRawContent(funcDecl))

	var functionCalls, methodCalls = map[string]Identity{}, map[string]Identity{}

	if funcDecl.Body == nil {
		goto set_func
	}

	ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if ok {
			var funcName string
			switch expr := call.Fun.(type) {
			case *ast.SelectorExpr:
				// TODO: not the best but works, optimize it later.
				x := expr.X
				for {
					if _, ok := x.(*ast.Ident); !ok {
						seleExp, ok := x.(*ast.SelectorExpr)
						if !ok {
							return false
						}
						x = seleExp.X
						continue
					}
					break
				}
				funcName = x.(*ast.Ident).Name + "." + expr.Sel.Name
				// check if it's method calls
				if sel, ok := ctx.pkgTypeInfo.Selections[expr]; ok && (sel.Kind() == types.MethodExpr || sel.Kind() == types.MethodVal) {
					// builtin or std libs, just ignore
					m, ok := sel.Obj().(*types.Func)
					if !ok || m.Pkg() == nil || ctx.IsSysImport(m.Pkg().Name()) {
						return true
					}
					sig := m.Type().(*types.Signature)
					rname := sig.Recv().Type().String()
					recv := getTypeNamed(sig.Recv().Type())
					if recv != nil {
						rname = recv.Name()
					}
					if rname == "" || strings.ContainsAny(rname, "{(") {
						// must be a local declaration, ignore it
						return true
					}
					mpkg := m.Pkg().Path()
					// NOTICE: skip sys imports?
					if isSysPkg(mpkg) {
						return true
					}
					//NOTICE: use {structName.methodName} as method key
					mname := rname + "." + m.Name()
					mod := ctx.module.Name
					if !strings.HasPrefix(mpkg, ctx.module.Name) {
						// external pkg
						mod = ctx.module.GetDependency(mpkg)
						if mod == "" {
							fmt.Fprintf(os.Stderr, "cannot find module for %s.%s in %s\n", mpkg, mname, ctx.filePath)
							return true
						}
					}
					methodCalls[funcName] = NewIdentity(mod, mpkg, mname)
					return true
				}
				// check if it's a function calls
				if use, ok := ctx.pkgTypeInfo.Uses[x.(*ast.Ident)]; ok {
					pkg, ok := use.(*types.PkgName)
					if !ok || pkg.Imported() == nil {
						return true
					}
					typ, ok := ctx.pkgTypeInfo.Types[expr]
					if !ok {
						return true
					}
					// expr type must be func signature
					if _, ok := typ.Type.(*types.Signature); !ok {
						return true
					}
					path := pkg.Imported().Path()
					mod := ctx.module.Name
					if isSysPkg(path) {
						return true
					} else if !strings.HasPrefix(path, ctx.module.Name) {
						// external
						mod = ctx.module.GetDependency(path)
						if mod == "" {
							fmt.Fprintf(os.Stderr, "cannot find module for %s.%s in %s\n", path, expr.Sel.Name, ctx.filePath)
							return true
						}
					}
					functionCalls[funcName] = NewIdentity(mod, path, expr.Sel.Name)
				}
			case *ast.Ident:
				funcName = expr.Name
				if isGoBuiltinFunc(funcName) {
					return true
				}
				typ, ok := ctx.pkgTypeInfo.Types[expr]
				if !ok {
					return true
				}
				// TODO: we can't handle variant func (closure) at present
				obj := ctx.pkgTypeInfo.Defs[expr]
				if _, isVar := obj.(*types.Var); isVar {
					return true
				}
				obj = ctx.pkgTypeInfo.Uses[expr]
				if _, isVar := obj.(*types.Var); isVar {
					return true
				}
				// expr type must be func signature
				if _, ok := typ.Type.(*types.Signature); !ok {
					return true
				}
				functionCalls[funcName] = NewIdentity(ctx.module.Name, ctx.pkgPath, funcName)
				return true
			}
		}
		return true
	})

set_func:
	name := funcDecl.Name.Name
	exported := isUpperCase(name[0])
	if isMethod {
		name = associatedStruct.Name + "." + name
	}
	if name == "init" && p.repo.GetFunction(NewIdentity(ctx.module.Name, ctx.pkgPath, name)) != nil {
		// according to https://go.dev/ref/spec#Program_initialization_and_execution,
		// duplicated init() is allowed and never be referenced, thus add a subfix
		name += "_" + strconv.Itoa(int(funcDecl.Pos()))
	}
	// update detailed function call info
	f := p.newFunc(ctx.module.Name, ctx.pkgPath, name)
	f.Exported = exported
	f.FilePath = ctx.filePath
	f.Content = content
	f.FunctionCalls = functionCalls
	f.MethodCalls = methodCalls
	f.IsMethod = isMethod
	f.AssociatedStruct = associatedStruct
	return f, true
}

func getTypeNamed(typ types.Type) types.Object {
	if pt, ok := typ.(*types.Pointer); ok {
		typ = pt.Elem()
	}
	name, ok := typ.(*types.Named)
	if ok {
		return name.Obj()
	}
	return nil
}

func (ctx *fileContext) IsSysImport(alias string) bool {
	_, ok := ctx.sysImports[alias]
	return ok
}

type TypeKind int

const (
	TypeKindStruct    = 0 // type struct
	TypeKindInterface = 1 // type interface
	TypeKindNamed     = 2 // type NamedXXX other..
)

// Type holds the information about a struct
type Type struct {
	Exported bool // if the struct is exported

	TypeKind        // type Kind: Struct / Interface / Typedef
	Identity        // unique id in a repo
	FilePath string `json:"-"` // File where the struct is defined
	Content  string // struct declaration content

	// field type (not include basic types), type name => type id
	SubStruct map[string]Identity

	// inline field type (not include basic types)
	InlineStruct map[string]Identity

	// methods defined on the Struct, not including inlined type's method
	Methods map[string]Identity

	// functions defined in fields, key is type name, val is the function Signature
	// FieldFunctions map[string]string
}

// The go file's context. Used to pass information between ast node handlers
type fileContext struct {
	filePath          string
	module            *Module
	pkgPath           PkgPath
	bs                []byte
	fset              *token.FileSet
	sysImports        map[string]string
	projectImports    map[string]string
	thirdPartyImports map[string][2]string
	pkgTypeInfo       *types.Info
}

func isUpperCase(c byte) bool {
	return c >= 'A' && c <= 'Z'
}

// parse a ast.StructType node and renturn allocated *Struct
func (p *goParser) parseStruct(ctx *fileContext, struName string, struDecl *ast.StructType) (*Type, bool) {
	st := p.newStruct(ctx.module.Name, ctx.pkgPath, struName)
	st.FilePath = ctx.filePath
	st.TypeKind = TypeKindStruct
	st.Content = "type " + st.Name + " " + string(ctx.GetRaw(struDecl.Struct, struDecl.End()))
	st.Exported = isUpperCase(struName[0])
	if struDecl.Fields == nil {
		return st, true
	}

	for _, fieldDecl := range struDecl.Fields.List {
		inlined := len(fieldDecl.Names) == 0
		fieldname := string(ctx.GetRawContent(fieldDecl.Type))
		if !inlined {
			// Fixme: join names?
			fieldname = fieldDecl.Names[0].Name
		}
		p.collectTypes(ctx, fieldname, fieldDecl.Type, st, inlined)
	}
	return st, true
}

func (p *goParser) collectTypes(ctx *fileContext, field string, typ ast.Expr, st *Type, inlined bool) {
	types := []Identity{}
	isFunc := getTypeName(ctx.fset, ctx.bs, typ, &types)

	for _, ty := range types {
		// regard func-typed field as a method on the struct
		if isFunc {
			// Fix: multiple types use the same mname
			// ex:  type FuncMap  map[func()]func()
			if len(types) > 1 {
				continue
			}
			mname := st.Name
			if field != "" {
				mname += "." + field
			}
			f := p.newFunc(ctx.module.Name, ctx.pkgPath, mname)
			id := NewIdentity(ctx.module.Name, ctx.pkgPath, st.Name)
			f.AssociatedStruct = &id
			f.IsMethod = true
			f.FilePath = ctx.filePath
			// NOTICE: content is only func signature
			f.Content = "func " + field + ty.Name
			if st.Methods == nil {
				st.Methods = map[string]Identity{}
			}
			st.Methods[field] = NewIdentity(ctx.module.Name, ctx.pkgPath, mname)
		} else {
			impt := ctx.pkgPath
			mod := ctx.module.Name
			if ty.PkgPath != "" {
				var err error
				impt, mod, err = ctx.GetImportPath(ty.PkgPath)
				if err == errSysImport {
					continue
				} else if err != nil {
					panic(err)
				}
			}
			if inlined {
				if st.InlineStruct == nil {
					st.InlineStruct = map[string]Identity{}
				}
				st.InlineStruct[ty.CallName()] = NewIdentity(mod, impt, ty.Name)
			} else {
				if st.SubStruct == nil {
					st.SubStruct = map[string]Identity{}
				}
				st.SubStruct[ty.CallName()] = NewIdentity(mod, impt, ty.Name)
			}
		}
	}
}

var errSysImport = fmt.Errorf("sys import")

func (ctx *fileContext) GetImportPath(alias string) (string, string, error) {
	if _, ok := ctx.sysImports[alias]; ok {
		return "", "", errSysImport
	} else if im, ok := ctx.projectImports[alias]; ok {
		return im, ctx.module.Name, nil
	} else if ims, ok := ctx.thirdPartyImports[alias]; ok {
		return ims[1], ims[0], nil
	} else {
		return "", "", fmt.Errorf("not found pkg: %s", alias)
	}
}

func (ctx *fileContext) GetRawContent(node ast.Node) []byte {
	return ctx.bs[ctx.fset.Position(node.Pos()).Offset:ctx.fset.Position(node.End()).Offset]
}

func (ctx *fileContext) GetRaw(from token.Pos, to token.Pos) []byte {
	return ctx.bs[ctx.fset.Position(from).Offset:ctx.fset.Position(to).Offset]
}

func (p *goParser) parseInterface(ctx *fileContext, name string, decl *ast.InterfaceType) (*Type, bool) {
	if decl == nil || decl.Incomplete || decl.Methods == nil {
		return nil, true
	}

	st := p.newStruct(ctx.module.Name, ctx.pkgPath, name)
	st.Exported = isUpperCase(name[0])
	st.FilePath = ctx.filePath
	st.TypeKind = TypeKindInterface
	st.Content = "type " + st.Name + " " + string(ctx.GetRaw(decl.Interface, decl.End()))

	for _, fieldDecl := range decl.Methods.List {
		inlined := len(fieldDecl.Names) == 0
		fieldname := string(ctx.GetRawContent(fieldDecl.Type))
		if !inlined {
			// Fixme: join names?
			fieldname = fieldDecl.Names[0].Name
		}
		p.collectTypes(ctx, fieldname, fieldDecl.Type, st, inlined)
	}

	return st, true
}

// handle typ expr and return not-builtin type identity and return if the type if a func signature.
// ret is used to store results.
func getTypeName(fset *token.FileSet, file []byte, typ ast.Expr, ret *[]Identity) bool {
	switch ty := typ.(type) {
	case *ast.Ident:
		if !isGoBuiltinFunc(ty.Name) {
			*ret = append(*ret, Identity{Name: ty.Name})
		}
		return false
	case *ast.IndexExpr: // generic type parameter
		*ret = append(*ret, Identity{Name: ty.X.(*ast.Ident).Name})
	case *ast.IndexListExpr: // generic type parameter
		*ret = append(*ret, Identity{Name: ty.X.(*ast.Ident).Name})
	case *ast.StarExpr:
		return getTypeName(fset, file, ty.X, ret)
	case *ast.ArrayType:
		return getTypeName(fset, file, ty.Elt, ret)
	case *ast.MapType:
		a := getTypeName(fset, file, ty.Key, ret)
		b := getTypeName(fset, file, ty.Value, ret)
		return a || b
	case *ast.ChanType:
		return getTypeName(fset, file, ty.Value, ret)
	case *ast.SelectorExpr:
		pkg, ok := ty.X.(*ast.Ident)
		if ok {
			*ret = append(*ret, Identity{Name: ty.Sel.Name, PkgPath: pkg.Name})
		}
		return false
	case *ast.FuncType:
		start := ty.Params.Pos()
		if ty.TypeParams != nil {
			start = ty.TypeParams.Pos()
		}
		name := string(file[fset.Position(start).Offset:fset.Position(typ.End()).Offset])
		*ret = append(*ret, Identity{Name: name})
		return true
	case *ast.InterfaceType:
		name := string(file[fset.Position(ty.Interface).Offset:fset.Position(typ.End()).Offset])
		*ret = append(*ret, Identity{Name: name})
		return false
	}
	return false
}
