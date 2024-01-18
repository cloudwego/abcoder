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
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
)

// PkgPath is the import path of a package, it is either absolute path or url
type PkgPath = string

// Identity holds identity information about a third party declaration
type Identity struct {
	PkgPath        // Import Path of the third party package
	Name    string // Unique Name of declaration (FunctionName, StructName.MethodName, or StructName)
}

// return full packagepath.name
func (i Identity) String() string {
	return i.PkgPath + "." + i.Name
}

// return packagename.name
func (i Identity) CallName() string {
	if i.PkgPath != "" {
		return filepath.Base(i.PkgPath) + "." + i.Name
	}
	return i.Name
}

// Function holds the information about a function
type Function struct {
	IsMethod         bool      // If the function is a method
	Identity                   // unique identity in a repo
	FilePath         string    `json:"-"` // File where the function is defined, empty if the function declaration is not scanned
	Content          string    // Content of the function, including functiion signature and body
	AssociatedStruct *Identity // Method receiver

	// call to in-the-project functions, key is {{pkgAlias.funcName}} or {{funcName}}
	InternalFunctionCalls map[string]Identity

	// call to third-party function calls, key is the {{pkgAlias.funcName}}
	// ex: http.Get() -> {"http.Get":{PkgDir: "net/http", Name: "Get"}}
	ThirdPartyFunctionCalls map[string]Identity

	// call to internal methods, key is the {{object.funcName}}
	InternalMethodCalls map[string]Identity

	// call to thrid-party methods, key is the {{object.funcName}}
	ThirdPartyMethodCalls map[string]Identity
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

func (p *goParser) inspectFile(ctx *fileContext, f *ast.File) (map[string]*Function, map[string]*Struct, error) {
	fileStructs := map[string]*Struct{}
	fileFuncs := map[string]*Function{}
	cont := true
	ast.Inspect(f, func(node ast.Node) bool {
		if funcDecl, ok := node.(*ast.FuncDecl); ok {
			// parse funcs
			f, ct := p.parseFunc(ctx, funcDecl)
			fileFuncs[f.Name] = f
			cont = ct
		} else if typDecl, ok := node.(*ast.TypeSpec); ok {
			name := typDecl.Name.Name
			var st *Struct
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
				st := p.newStruct(ctx.pkgPath, name)
				st.TypeKind = TypeKindNamed
				st.Content = string(ctx.GetRawContent(typDecl))
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
func (p *goParser) newFunc(pkg, name string) *Function {
	ret := &Function{Identity: Identity{pkg, name}}
	p.repo.SetFunction(ret.Identity, ret)
	return ret
}

// newStruct allocate a struct in the repo
func (p *goParser) newStruct(pkg, name string) *Struct {
	ret := &Struct{Identity: Identity{pkg, name}}
	p.repo.SetType(ret.Identity, ret)
	return ret
}

func (p *goParser) seprateImports(impts []*ast.ImportSpec) (map[string]string, map[string]string, map[string]string) {
	thirdPartyImports := make(map[string]string)
	projectImports := make(map[string]string)
	sysImports := make(map[string]string)
	for _, imp := range impts {
		importPath := imp.Path.Value[1 : len(imp.Path.Value)-1] // remove the quotes
		importAlias := filepath.Base(importPath)                // use the base name as alias by default
		// Check if user has defined an alias for current import
		if imp.Name != nil {
			importAlias = imp.Name.Name // update the alias
		}

		// Fix: module name may also be like this?
		isSysPkg := !strings.Contains(strings.Split(importPath, "/")[0], ".")
		if isSysPkg {
			// Ignoring golang standard libraries（like net/http）
			sysImports[importAlias] = importPath
		} else {
			// Distinguish between project packages and third party packages
			if strings.HasPrefix(importPath, p.modName) {
				projectImports[importAlias] = importPath
			} else {
				thirdPartyImports[importAlias] = importPath
			}
		}
	}
	return sysImports, projectImports, thirdPartyImports
}

// parseFunc parses all function declaration in one file
func (p *goParser) parseFunc(ctx *fileContext, funcDecl *ast.FuncDecl) (*Function, bool) {
	var associatedStruct *Identity
	isMethod := funcDecl.Recv != nil
	if isMethod {
		var structName string
		// TODO: reserve the pointer message?
		switch x := funcDecl.Recv.List[0].Type.(type) {
		case *ast.Ident:
			structName = x.Name
		case *ast.StarExpr:
			structName = x.X.(*ast.Ident).Name
		}
		associatedStruct = &Identity{ctx.pkgPath, structName}
	}

	pos := ctx.fset.PositionFor(funcDecl.Pos(), false).Offset
	end := ctx.fset.PositionFor(funcDecl.End(), false).Offset
	content := string(ctx.bs[pos:end])

	var thirdPartyMethodCalls, thirdPartyFunctionCalls = map[string]Identity{}, map[string]Identity{}
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
					mpkg := m.Pkg().Path()
					//NOTICE: use {structName.methodName} as method key
					mname := rname + "." + m.Name()
					if strings.HasPrefix(mpkg, p.modName) {
						// internal pkg
						methodCalls[funcName] = Identity{mpkg, mname}
					} else {
						// external pkg
						thirdPartyMethodCalls[funcName] = Identity{mpkg, mname}
					}
					return true
				}
				// check if it's a package reference
				if use, ok := ctx.pkgTypeInfo.Uses[x.(*ast.Ident)]; ok {
					pkg, ok := use.(*types.PkgName)
					if !ok || pkg.Imported() == nil {
						return true
					}
					// NOTICE: skip sys imports?
					if ctx.IsSysImport(pkg.Imported().Name()) {
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
					// internal function calls
					if impt, ok := ctx.projectImports[pkg.Imported().Name()]; ok {
						functionCalls[funcName] = Identity{impt, expr.Sel.Name}
						return true
					}
					// third-party function calls
					if impt, ok := ctx.thirdPartyImports[pkg.Imported().Name()]; ok {
						thirdPartyFunctionCalls[funcName] = Identity{PkgPath: impt, Name: expr.Sel.Name}
						return true
					}
					return true
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
				functionCalls[funcName] = Identity{ctx.pkgPath, funcName}
				return true
			}
		}
		return true
	})

set_func:
	name := funcDecl.Name.Name
	if isMethod {
		name = associatedStruct.Name + "." + name
	}
	// update detailed function call info
	f := p.newFunc(ctx.pkgPath, name)
	*f = Function{
		Identity: Identity{
			Name:    name,
			PkgPath: ctx.pkgPath,
		},
		FilePath:                ctx.filePath,
		IsMethod:                isMethod,
		AssociatedStruct:        associatedStruct,
		Content:                 content,
		InternalFunctionCalls:   functionCalls,
		ThirdPartyFunctionCalls: thirdPartyFunctionCalls,
		InternalMethodCalls:     methodCalls,
		ThirdPartyMethodCalls:   thirdPartyMethodCalls,
	}
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

// Struct holds the information about a struct
type Struct struct {
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
	pkgPath           PkgPath
	bs                []byte
	fset              *token.FileSet
	sysImports        map[string]string
	projectImports    map[string]string
	thirdPartyImports map[string]string
	pkgTypeInfo       *types.Info
}

// parse a ast.StructType node and renturn allocated *Struct
func (p *goParser) parseStruct(ctx *fileContext, struName string, struDecl *ast.StructType) (*Struct, bool) {
	st := p.newStruct(ctx.pkgPath, struName)
	st.FilePath = ctx.filePath
	st.TypeKind = TypeKindStruct

	pos := ctx.fset.PositionFor(struDecl.Pos(), false).Offset
	end := ctx.fset.PositionFor(struDecl.End(), false).Offset
	st.Content = string(ctx.bs[pos:end])

	ast.Inspect(struDecl.Fields, func(n ast.Node) bool {
		fieldDecl, ok := n.(*ast.Field)
		if !ok {
			return true
		}
		inlined := len(fieldDecl.Names) == 0
		fieldname := string(ctx.GetRawContent(fieldDecl.Type))
		if !inlined {
			// Fixme: join names?
			fieldname = fieldDecl.Names[0].Name
		}
		p.collectTypes(ctx, fieldname, fieldDecl.Type, st, inlined)
		return true
	})
	return st, true
}

func (p *goParser) collectTypes(ctx *fileContext, field string, typ ast.Expr, st *Struct, inlined bool) {
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
			f := p.newFunc(ctx.pkgPath, mname)
			f.AssociatedStruct = &Identity{ctx.pkgPath, st.Name}
			f.IsMethod = true
			f.FilePath = ctx.filePath
			// NOTICE: content is only func signature
			f.Content = ty.Name
			if st.Methods == nil {
				st.Methods = map[string]Identity{}
			}
			st.Methods[field] = Identity{ctx.pkgPath, mname}
		} else {
			impt := ctx.pkgPath
			if ty.PkgPath != "" {
				if _, ok := ctx.sysImports[ty.PkgPath]; ok {
					continue
				} else if im, ok := ctx.projectImports[ty.PkgPath]; ok {
					impt = im
				} else if im, ok := ctx.thirdPartyImports[ty.PkgPath]; ok {
					impt = im
				} else {
					panic("not found pkg: " + impt)
				}
			}
			if inlined {
				if st.InlineStruct == nil {
					st.InlineStruct = map[string]Identity{}
				}
				st.InlineStruct[ty.CallName()] = Identity{impt, ty.Name}
			} else {
				if st.SubStruct == nil {
					st.SubStruct = map[string]Identity{}
				}
				st.SubStruct[ty.CallName()] = Identity{impt, ty.Name}
			}
		}
	}
}

func (ctx *fileContext) GetRawContent(node ast.Node) []byte {
	return ctx.bs[ctx.fset.Position(node.Pos()).Offset:ctx.fset.Position(node.End()).Offset]
}

func (p *goParser) parseInterface(ctx *fileContext, name string, decl *ast.InterfaceType) (*Struct, bool) {
	if decl == nil || decl.Incomplete {
		return nil, true
	}

	st := p.newStruct(ctx.pkgPath, name)
	st.FilePath = ctx.filePath
	st.TypeKind = TypeKindInterface
	st.Content = string(ctx.GetRawContent(decl))

	ast.Inspect(decl.Methods, func(n ast.Node) bool {
		fieldDecl, ok := n.(*ast.Field)
		if !ok {
			return true
		}
		inlined := len(fieldDecl.Names) == 0
		fieldname := string(ctx.GetRawContent(fieldDecl.Type))
		if !inlined {
			// Fixme: join names?
			fieldname = fieldDecl.Names[0].Name
		}
		p.collectTypes(ctx, fieldname, fieldDecl.Type, st, inlined)
		return true
	})

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
		name := string(file[fset.Position(ty.Func).Offset:fset.Position(typ.End()).Offset])
		*ret = append(*ret, Identity{Name: name})
		return true
	case *ast.InterfaceType:
		name := string(file[fset.Position(ty.Interface).Offset:fset.Position(typ.End()).Offset])
		*ret = append(*ret, Identity{Name: name})
		return false
	}
	return false
}
