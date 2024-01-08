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
	"path/filepath"
	"strings"
)

// PkgPath is the import path of a package, it is either absolute path or url
type PkgPath = string

// ThirdPartyIdentity holds identity information about a third party declaration
type ThirdPartyIdentity struct {
	PkgPath         // Import Path of the third party package
	Identity string // Unique Name of declaration (FunctionName, StructName.MethodName, or StructName)
}

// Function holds the information about a function
type Function struct {
	IsMethod         bool    // If the function is a method
	Name             string  // Name of the function
	PkgPath                  // import path to the package where the function is defined
	FilePath         string  // File where the function is defined, empty if the function declaration is not scanned
	Content          string  // Content of the function, including functiion signature and body
	AssociatedStruct *Struct // Method receiver

	// call to in-the-project functions, key is {{pkgAlias.funcName}} or {{funcName}}
	InternalFunctionCalls map[string]*Function

	// call to third-party function calls, key is the {{pkgAlias.funcName}}
	// ex: http.Get() -> {"http.Get":{PkgDir: "net/http", Name: "Get"}}
	ThirdPartyFunctionCalls map[string]*ThirdPartyIdentity

	// call to internal methods, key is the {{object.funcName}}
	InternalMethodCalls map[string]*Function

	// call to thrid-party methods, key is the {{object.funcName}}
	ThirdPartyMethodCalls map[string]*ThirdPartyIdentity
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

// getOrSetFunc get a function in the map, or alloc and set a new one if not exists
func (p *goParser) getOrSetFunc(pkg, name string) *Function {
	pkgFuncs := p.processedPkgFunctions[pkg]
	if pkgFuncs == nil {
		pkgFuncs = make(map[string]*Function)
		p.processedPkgFunctions[pkg] = pkgFuncs
	}
	if pkgFuncs[name] == nil {
		f := &Function{Name: name, PkgPath: pkg}
		pkgFuncs[name] = f
		return f
	}
	return pkgFuncs[name]
}

// getOrSetStruct get a struct in the map, or alloc and set a new one if not exists
func (p *goParser) getOrSetStruct(pkg, name string) *Struct {
	pkgStructs := p.processedPkgStruct[pkg]
	if pkgStructs == nil {
		pkgStructs = make(map[string]*Struct)
		p.processedPkgStruct[pkg] = pkgStructs
	}
	if pkgStructs[name] == nil {
		s := &Struct{Name: name, PkgPath: pkg}
		pkgStructs[name] = s
		return s
	}
	return pkgStructs[name]
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

	var associatedStruct *Struct
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
		associatedStruct = p.getOrSetStruct(ctx.pkgPath, structName)
	}

	pos := ctx.fset.PositionFor(funcDecl.Pos(), false).Offset
	end := ctx.fset.PositionFor(funcDecl.End(), false).Offset
	content := string(ctx.bs[pos:end])

	var thirdPartyMethodCalls, thirdPartyFunctionCalls = map[string]*ThirdPartyIdentity{}, map[string]*ThirdPartyIdentity{}
	var functionCalls, methodCalls = map[string]*Function{}, map[string]*Function{}

	ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if ok {
			var funcName string
			switch expr := call.Fun.(type) {
			case *ast.SelectorExpr:
				funcName := ""
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
				// fixme: in closure like func(importName StructX) { ... }, importName is not in projectImports
				funcName = x.(*ast.Ident).Name + "." + expr.Sel.Name
				// internal function calls
				if impt, ok := ctx.projectImports[x.(*ast.Ident).Name]; ok {
					functionCalls[funcName] = p.getOrSetFunc(impt, expr.Sel.Name)
					return true
				}
				// third-party function calls
				if impt, ok := ctx.thirdPartyImports[x.(*ast.Ident).Name]; ok {
					thirdPartyFunctionCalls[funcName] = &ThirdPartyIdentity{PkgPath: impt, Identity: expr.Sel.Name}
					return true
				}
				// WHY: skip sys imports?
				if _, ok := ctx.sysImports[x.(*ast.Ident).Name]; ok {
					// internalFunctionCalls[funcName] = p.getOrSetFunc(impt, expr.Sel.Name)
					return true
				}

				// Fallback must be method calls
				// FIXME: get type info of object
				f := p.getOrSetFunc(ctx.pkgPath, funcName)
				f.IsMethod = true
				f.AssociatedStruct = &Struct{Name: x.(*ast.Ident).Name}
				methodCalls[funcName] = f
				// TODO: seperate internal method and third-party method

				return true
			case *ast.Ident:
				funcName = expr.Name
				if !isGoBuiltinFunc(funcName) {
					functionCalls[funcName] = p.getOrSetFunc(ctx.pkgPath, funcName)
				}
				return true
			}
		}
		return true
	})
	name := funcDecl.Name.Name
	if isMethod {
		name = associatedStruct.Name + "." + name
	}
	// update detailed function call info
	f := p.getOrSetFunc(ctx.pkgPath, name)
	*f = Function{
		Name:                    funcDecl.Name.Name,
		PkgPath:                 ctx.pkgPath,
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

// Struct holds the information about a struct
type Struct struct {
	Name     string // Name of the struct
	PkgPath         // Path to the package where the struct is defined
	FilePath string // File where the struct is defined
	Content  string // struct declaration content

	// related local structs in fields, key is {{pkgName.typName}} or {{typeName}}, val is declaration of the struct
	InternalStructs map[string]*Struct

	// related third party structs in fields,
	// ex: type A struct { B pkg.B }, pkg.B is a child of A, key is "pkg.B"
	ThirdPartyStructs map[string]*ThirdPartyIdentity

	// method name to Function
	Methods map[string]*Function

	// functions defined in fields, key is fieldName, val is the functionSignature
	FieldFunctions map[string]string
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
}

// parse a ast.StructType node and renturn allocated *Struct
func (p *goParser) parseStruct(ctx *fileContext, struName string, struDecl *ast.StructType) (*Struct, bool) {
	pkgPath := p.pkgPathFromABS(filepath.Dir(ctx.filePath))
	st := p.getOrSetStruct(pkgPath, struName)
	st.FilePath = ctx.filePath

	pos := ctx.fset.PositionFor(struDecl.Pos(), false).Offset
	end := ctx.fset.PositionFor(struDecl.End(), false).Offset
	st.Content = string(ctx.bs[pos:end])

	inStructs := map[string]*Struct{}
	exStructs := map[string]*ThirdPartyIdentity{}
	fieldFuncs := map[string]string{}

	ast.Inspect(struDecl.Fields, func(n ast.Node) bool {
		fieldDecl, ok := n.(*ast.Field)
		if !ok {
			return true
		}
		name := ""
		if len(fieldDecl.Names) > 0 {
			// TODO: combine all names
			name = fieldDecl.Names[0].String()
		} else {
			name = string(ctx.bs[fieldDecl.Type.Pos():fieldDecl.Type.End()])
		}

		types := []ThirdPartyIdentity{}
		isFunc := getTypeName(ctx.bs, fieldDecl.Type, &types)

		for _, ty := range types {
			if isFunc {
				fieldFuncs[name] = ty.Identity
			}
			// local structs
			if ty.PkgPath != "" {
				if _, ok := ctx.sysImports[ty.PkgPath]; ok {
					// std package
					continue
				}
				if impt, ok := ctx.projectImports[ty.PkgPath]; ok {
					// internal package
					sub := p.getOrSetStruct(impt, ty.Identity)
					inStructs[name] = sub
				} else if impt, ok := ctx.thirdPartyImports[ty.PkgPath]; ok {
					// thrid-party package
					ty.PkgPath = impt
					exStructs[name] = &ty
				}
			} else {
				// local package
				sub := p.getOrSetStruct(pkgPath, ty.Identity)
				inStructs[name] = sub
			}

		}

		return true
	})

	st.InternalStructs = inStructs
	st.ThirdPartyStructs = exStructs
	st.FieldFunctions = fieldFuncs

	return st, true
}

// handle typ expr and return not-builtin type identity and return if the type if a func signature.
// ret is used to store results.
func getTypeName(file []byte, typ ast.Expr, ret *[]ThirdPartyIdentity) bool {
	switch ty := typ.(type) {
	case *ast.Ident:
		if !isGoBuiltinFunc(ty.Name) {
			*ret = append(*ret, ThirdPartyIdentity{Identity: ty.Name})
		}
		return false
	case *ast.StarExpr:
		return getTypeName(file, ty.X, ret)
	case *ast.ArrayType:
		return getTypeName(file, ty.Elt, ret)
	case *ast.MapType:
		a := getTypeName(file, ty.Key, ret)
		b := getTypeName(file, ty.Value, ret)
		return a || b
	case *ast.ChanType:
		return getTypeName(file, ty.Value, ret)
	case *ast.SelectorExpr:
		pkg, ok := ty.X.(*ast.Ident)
		if ok {
			*ret = append(*ret, ThirdPartyIdentity{Identity: ty.Sel.Name, PkgPath: pkg.Name})
		}
		return false
	case *ast.FuncType:
		name := string(file[ty.Func:typ.End()])
		*ret = append(*ret, ThirdPartyIdentity{Identity: name})
		return true
	case *ast.InterfaceType:
		name := string(file[ty.Interface:typ.End()])
		*ret = append(*ret, ThirdPartyIdentity{Identity: name})
		return false
	}
	return false
}
