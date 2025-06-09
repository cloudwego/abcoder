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
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"strconv"
	"strings"

	. "github.com/cloudwego/abcoder/lang/uniast"
)

func (p *GoParser) parseFile(ctx *fileContext, f *ast.File) error {
	cont := true
	ast.Inspect(f, func(node ast.Node) bool {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "panic: %v in %s:%d\n", r, ctx.filePath, ctx.fset.Position(node.Pos()).Line)
				cont = false
				return
			}
		}()
		if funcDecl, ok := node.(*ast.FuncDecl); ok {
			// parse funcs
			_, ct := p.parseFunc(ctx, funcDecl)
			// fileFuncs[f.Name] = f
			cont = ct
		} else if decl, ok := node.(*ast.GenDecl); ok {
			var doc *ast.CommentGroup
			if ctx.collectComment && decl.Doc != nil {
				doc = decl.Doc
			}
			var ct = true
			switch decl.Tok {
			case token.TYPE:
				for _, spec := range decl.Specs {
					typDecl := spec.(*ast.TypeSpec)
					_, ct = p.parseType(ctx, typDecl, doc)
				}
			case token.VAR:
				var firstVal *float64
				for _, spec := range decl.Specs {
					vspec, ok := spec.(*ast.ValueSpec)
					if ok {
						_, firstVal = p.parseVar(ctx, vspec, false, nil, firstVal, doc)
					}
				}
			case token.CONST:
				var firstType *Identity
				var firstVal *float64
				for _, spec := range decl.Specs {
					vspec, ok := spec.(*ast.ValueSpec)
					if ok {
						firstType, firstVal = p.parseVar(ctx, vspec, true, firstType, firstVal, doc)
					}
				}
			}
			cont = ct
		}
		return cont
	})
	return nil
}

func (p *GoParser) newVar(mod string, pkg string, name string, isConst bool) *Var {
	ret := &Var{
		Identity:   NewIdentity(mod, pkg, name),
		IsConst:    isConst,
		IsExported: isUpperCase(name[0]),
	}
	return p.repo.SetVar(ret.Identity, ret)
}

func (p *GoParser) parseVar(ctx *fileContext, vspec *ast.ValueSpec, isConst bool, lastType *Identity, lastValue *float64, doc *ast.CommentGroup) (*Identity, *float64) {
	var typ *Identity
	var val *ast.Expr
	for i, name := range vspec.Names {
		if name.Name == "_" {
			// igore anonymous var
			continue
		}
		if vspec.Values != nil {
			val = &vspec.Values[i]
		}
		v := p.newVar(ctx.module.Name, ctx.pkgPath, name.Name, isConst)
		v.FileLine = ctx.FileLine(vspec)
		if vspec.Type != nil {
			ti := ctx.GetTypeInfo(vspec.Type)
			v.Type = &ti.Id
			v.IsPointer = ti.IsPointer
			for _, dep := range ti.Deps {
				v.Dependencies = InsertDependency(v.Dependencies, NewDependency(dep, ctx.FileLine(vspec.Type)))
			}
		} else if val != nil && !isConst {
			ti := ctx.GetTypeInfo(*val)
			v.Type = &ti.Id
			v.IsPointer = ti.IsPointer
			for _, dep := range ti.Deps {
				v.Dependencies = InsertDependency(v.Dependencies, NewDependency(dep, ctx.FileLine(*val)))
			}
		} else {
			v.Type = typ
		}
		// NOTICE: for `const ( a Type, b )` b can inherit the a's type
		if isConst && v.Type == nil {
			v.Type = lastType
		}
		var varType string
		if v.Type != nil {
			if v.Type.PkgPath == ctx.pkgPath {
				varType = v.Type.Name
			} else {
				varType = v.Type.CallName()
			}
			if v.IsPointer {
				varType = "*" + varType
			}
		}

		if !isConst {
			v.Content = fmt.Sprintf("var %s %s", name.Name, varType)
		} else {
			if varType != "" {
				v.Content = fmt.Sprintf("const %s %s", name.Name, varType)
			} else {
				v.Content = fmt.Sprintf("const %s", name.Name)
			}
		}

		var comment string
		if ctx.collectComment && doc != nil {
			comment += string(ctx.GetRawContent(doc)) + "\n"
		}
		if ctx.collectComment && vspec.Doc != nil {
			comment += string(ctx.GetRawContent(vspec.Doc)) + "\n"
			v.FileLine.StartOffset = ctx.fset.Position(vspec.Pos()).Offset
		}
		v.Content = comment + v.Content

		var finalVal string
		if val != nil {
			// refer codes
			if sel, ok := (*val).(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok {
					if pkg, ok := ctx.pkgTypeInfo.Uses[x]; ok {
						if pkg, ok := pkg.(*types.PkgName); ok {
							path := pkg.Imported().Path()
							mod, err := ctx.GetMod(path)
							if err == errSysImport {
								continue
							}
							id := NewIdentity(mod, path, sel.Sel.Name)
							// refer val's define
							if err := p.referCodes(ctx, &id, p.opts.ReferCodeDepth); err != nil {
								fmt.Fprintf(os.Stderr, "failed to get refer code for %s: %v\n", id.Name, err)
							}
						}
					}
				}
			}
			finalVal = string(ctx.GetRawContent(*val))
			// NOTICE: handle `iota`
			if strings.Contains(finalVal, "iota") {
				// parset the val expr to int value
				tmp, err := parseExpr(finalVal)
				if err == nil {
					if v, ok := tmp.(float64); ok {
						lastValue = &v
						finalVal = strconv.FormatFloat(v, 'f', -1, 64)
					}
				}
			}
		} else if lastValue != nil {
			tmp := (*lastValue + 1)
			lastValue = &tmp
			finalVal = strconv.FormatFloat(tmp, 'f', -1, 64)
		}

		if finalVal != "" {
			v.Content += " = " + finalVal
		}

		typ = v.Type
	}
	return typ, lastValue
}

// newFunc allocate a function in the repo
func (p *GoParser) newFunc(mod, pkg, name string) *Function {
	ret := &Function{Identity: NewIdentity(mod, pkg, name), Exported: isUpperCase(name[0])}
	return p.repo.SetFunction(ret.Identity, ret)
}

// newType allocate a struct in the repo
func (p *GoParser) newType(mod, pkg, name string) *Type {
	ret := &Type{Identity: NewIdentity(mod, pkg, name), Exported: isUpperCase(name[0])}
	return p.repo.SetType(ret.Identity, ret)
}

func (p *GoParser) parseSelector(ctx *fileContext, expr *ast.SelectorExpr, infos collectInfos) (cont bool) {
	// println("[parseFunc] ast.SelectorExpr:", string(ctx.GetRawContent(expr)))
	// TODO: not the best but works, optimize it later.
	if ident, ok := expr.X.(*ast.Ident); ok {
		if use, ok := ctx.pkgTypeInfo.Uses[ident]; ok {
			if pkg, ok := use.(*types.PkgName); ok {

				// pkg.funccall
				// callName := string(ctx.GetRawContent(expr))
				path := pkg.Imported().Path()
				mod, err := ctx.GetMod(path)
				if err == errSysImport {
					return false
				}
				id := NewIdentity(mod, path, expr.Sel.Name)
				dep := NewDependency(id, ctx.FileLine(expr.Sel))

				// NOTICE: refer external codes for convinience
				if err := p.referCodes(ctx, &id, p.opts.ReferCodeDepth); err != nil {
					fmt.Fprintf(os.Stderr, "failed to get refer code for %s: %v\n", id.Name, err)
				}

				if v := ctx.pkgTypeInfo.Uses[expr.Sel]; v != nil {
					// type name
					if _, isNamed := v.(*types.TypeName); isNamed {
						// id, ok := ctx.getTypeId(tn.Type())
						// if !ok {
						// 	// fmt.Fprintf(os.Stderr, "failed to get type id for %s\n", expr.Name)
						// 	return false
						// }
						*infos.tys = InsertDependency(*infos.tys, dep)
						// global var
					} else if _, ok := v.(*types.Const); ok {
						*infos.globalVars = InsertDependency(*infos.globalVars, dep)
						// external const
					} else if _, ok := v.(*types.Var); ok {
						*infos.globalVars = InsertDependency(*infos.globalVars, dep)
						// external function
					} else if _, ok := v.(*types.Func); ok {
						*infos.functionCalls = InsertDependency(*infos.functionCalls, dep)
					}
					return false
				}
				return false
			}
		}
	} else if sel, ok := expr.X.(*ast.SelectorExpr); ok {
		// recurse call
		cont = p.parseSelector(ctx, sel, infos)
	} else {
		// try to get type info of field first
		if ti := ctx.GetTypeInfo(expr); ti.Ty != nil {
			if _, ok := ti.Ty.(*types.Signature); ok {
				// collect method call
				// method call
				rev := ctx.GetTypeInfo(expr.X)
				if !rev.IsStdOrBuiltin {
					id := NewIdentity(rev.Id.ModPath, rev.Id.PkgPath, rev.Id.Name+"."+expr.Sel.Name)
					dep := NewDependency(id, ctx.FileLine(expr.Sel))
					*infos.methodCalls = InsertDependency(*infos.methodCalls, dep)
				}
			}
		}
		return true
	}

	// method calls
	// ex: `obj.Method()`
	if sel, ok := ctx.pkgTypeInfo.Selections[expr]; ok && (sel.Kind() == types.MethodExpr || sel.Kind() == types.MethodVal) {
		// println("[parseFunc] method call:", callName)
		// builtin or std libs, just ignore
		m, ok := sel.Obj().(*types.Func)
		if !ok || m.Pkg() == nil || ctx.IsSysImport(m.Pkg().Name()) {
			return false
		}
		pkg := m.Pkg().Path()
		mod, err := ctx.GetMod(pkg)
		if err == errSysImport {
			return false
		}
		// callName := string(ctx.GetRawContent(expr))
		// get receiver type name
		// var rname string
		rev := ctx.getTypeinfo(sel.Recv())
		// if rev == nil {
		// 	rname = extractName(sel.Recv().String())
		// } else {
		if !rev.IsStdOrBuiltin && rev.Id.ModPath != "" {
			id := NewIdentity(mod, pkg, rev.Id.Name+"."+expr.Sel.Name)
			dep := NewDependency(id, ctx.FileLine(expr.Sel))
			if err := p.referCodes(ctx, &id, p.opts.ReferCodeDepth); err != nil {
				fmt.Fprintf(os.Stderr, "failed to get refer code for %s: %v\n", id.Name, err)
			}
			*infos.methodCalls = InsertDependency(*infos.methodCalls, dep)
		}
		return false
	}

	return cont
}

type collectInfos struct {
	functionCalls, methodCalls *[]Dependency
	tys, globalVars            *[]Dependency
}

// parseFunc parses all function declaration in one file
func (p *GoParser) parseFunc(ctx *fileContext, funcDecl *ast.FuncDecl) (*Function, bool) {
	// method receiver
	var receiver *Receiver
	isMethod := funcDecl.Recv != nil
	if isMethod {
		// TODO: reserve the pointer message?
		ti := ctx.GetTypeInfo(funcDecl.Recv.List[0].Type)
		// name := "self"
		// if len(funcDecl.Recv.List[0].Names) > 0 {
		// 	name = funcDecl.Recv.List[0].Names[0].Name
		// }
		receiver = &Receiver{
			Type:      ti.Id,
			IsPointer: ti.IsPointer,
			// Name:      name,
		}
	}

	fname := funcDecl.Name.Name
	if isMethod {
		fname = receiver.Type.Name + "." + fname
	}

	// collect parameters
	var params []Dependency
	if funcDecl.Type.Params != nil {
		ctx.collectFields(funcDecl.Type.Params.List, &params)
	}
	// collect results
	var results []Dependency
	if funcDecl.Type.Results != nil {
		ctx.collectFields(funcDecl.Type.Results.List, &results)
	}

	// collect content
	content := string(ctx.GetRawContent(funcDecl))

	var functionCalls, globalVars, tys, methodCalls []Dependency

	if funcDecl.Body == nil {
		goto set_func
	}

	ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
		switch expr := node.(type) {
		case *ast.SelectorExpr:
			return p.parseSelector(ctx, expr, collectInfos{&functionCalls, &methodCalls, &tys, &globalVars})
		case *ast.Ident:
			callName := expr.Name
			// println("[parseFunc] ast.Ident:", callName)
			if isGoBuiltins(callName) {
				return false
			}

			// // collect Named types of defines
			// // ex: `var x NamedType`
			// if def, ok := ctx.pkgTypeInfo.Defs[expr]; ok {
			// 	println("[parseFunc] def:", def.String())
			// 	if tn, isNamed := def.(*types.TypeName); isNamed {
			// 		id, ok := ctx.getTypeId(tn.Type())
			// 		if !ok {
			// 			// fmt.Fprintf(os.Stderr, "failed to get type id for %s\n", expr.Name)
			// 			return false
			// 		}
			// 		tys[expr.Name] = id
			// 	}
			// 	return false
			// }
			if use, ok := ctx.pkgTypeInfo.Uses[expr]; ok {
				id := NewIdentity(ctx.module.Name, ctx.pkgPath, callName)
				dep := NewDependency(id, ctx.FileLine(expr))
				// type name
				if _, isNamed := use.(*types.TypeName); isNamed {
					// id, ok := ctx.getTypeId(tn.Type())
					// if !ok {
					// 	// fmt.Fprintf(os.Stderr, "failed to get type id for %s\n", expr.Name)
					// 	return false
					// }
					tys = InsertDependency(tys, dep)
					// global var
				} else if v, ok := use.(*types.Var); ok {
					// NOTICE: the Parent of global scope is nil?
					if isPkgScope(v.Parent()) {
						globalVars = InsertDependency(globalVars, dep)
					}
					// global const
				} else if c, ok := use.(*types.Const); ok {
					if isPkgScope(c.Parent()) {
						globalVars = InsertDependency(globalVars, dep)
					}
					return false
					// function
				} else if f, ok := use.(*types.Func); ok {
					// exclude method
					if f.Type().(*types.Signature).Recv() == nil {
						functionCalls = InsertDependency(functionCalls, dep)
					}
				}
			}
		}
		return true
	})

set_func:

	if fname == "init" && p.repo.GetFunction(NewIdentity(ctx.module.Name, ctx.pkgPath, fname)) != nil {
		// according to https://go.dev/ref/spec#Program_initialization_and_execution,
		// duplicated init() is allowed and never be referenced, thus add a subfix
		fname += "_" + strconv.Itoa(int(funcDecl.Pos()))
	}

	// update detailed function call info
	f := p.newFunc(ctx.module.Name, ctx.pkgPath, fname)
	f.FileLine = ctx.FileLine(funcDecl)
	f.Content = content
	f.FunctionCalls = functionCalls
	f.MethodCalls = methodCalls
	f.IsMethod = isMethod
	f.Receiver = receiver
	f.Params = params
	f.Results = results
	f.GlobalVars = globalVars
	f.Types = tys
	return f, false
}

func (p *GoParser) parseType(ctx *fileContext, typDecl *ast.TypeSpec, doc *ast.CommentGroup) (st *Type, ct bool) {
	switch decl := typDecl.Type.(type) {
	case *ast.StructType:
		st, ct = p.parseStruct(ctx, typDecl.Name.Name, typDecl.Name, decl)
	case *ast.InterfaceType:
		st, ct = p.parseInterface(ctx, typDecl.Name, decl)
	default:
		// typedef, ex: type Str StructA
		st = p.newType(ctx.module.Name, ctx.pkgPath, typDecl.Name.Name)
		st.TypeKind = "typedef"
		p.collectTypes(ctx, typDecl.Type, st, typDecl.Assign.IsValid())
		ct = false
		// check if it implements any parser.interfaces
		if obj, ok := ctx.pkgTypeInfo.Defs[typDecl.Name]; ok {
			if t := obj.Type(); t != nil {
				p.types[t] = st.Identity
			}
		}
	}

	st.FileLine = ctx.FileLine(typDecl)
	st.Content = string(ctx.GetRawContent(typDecl))
	if ctx.collectComment && doc != nil {
		st.Content = string(ctx.GetRawContent(doc)) + "\n" + string(ctx.GetRawContent(typDecl))
	}
	return
}

// parse a ast.StructType node and renturn allocated *Struct
func (p *GoParser) parseStruct(ctx *fileContext, struName string, name *ast.Ident, struDecl *ast.StructType) (*Type, bool) {
	st := p.newType(ctx.module.Name, ctx.pkgPath, struName)
	st.TypeKind = "struct"
	if struDecl.Fields == nil {
		return st, false
	}
	for _, fieldDecl := range struDecl.Fields.List {
		inlined := len(fieldDecl.Names) == 0
		fieldname := string(ctx.GetRawContent(fieldDecl.Type))
		if !inlined {
			// Fixme: join names?
			fieldname = fieldDecl.Names[0].Name
		}
		if stru, ok := fieldDecl.Type.(*ast.StructType); ok {
			// anonymous struct. parse it
			as, _ := p.parseStruct(ctx, "_"+fieldname, nil, stru)
			// move out substructs of the anonymous struct
			for _, dep := range as.SubStruct {
				st.SubStruct = InsertDependency(st.SubStruct, dep)
			}
			for _, dep := range as.InlineStruct {
				st.SubStruct = InsertDependency(st.InlineStruct, dep)
			}
			// remove the anonymous struct from the repo
			delete(p.repo.GetPackage(as.ModPath, as.PkgPath).Types, as.Name)
		} else {
			p.collectTypes(ctx, fieldDecl.Type, st, inlined)
		}
	}
	// check if it implements any parser.interfaces
	if name != nil {
		// check if it implements any parser.interfaces
		if obj, ok := ctx.pkgTypeInfo.Defs[name]; ok {
			if t := obj.Type(); t != nil {
				p.types[t] = st.Identity
			}
		}
	}

	return st, false
}

func (p *GoParser) parseInterface(ctx *fileContext, name *ast.Ident, decl *ast.InterfaceType) (*Type, bool) {
	if decl == nil || decl.Incomplete || decl.Methods == nil {
		return nil, true
	}

	st := p.newType(ctx.module.Name, ctx.pkgPath, name.Name)
	st.TypeKind = "interface"

	for _, fieldDecl := range decl.Methods.List {
		inlined := len(fieldDecl.Names) == 0
		// fieldname := string(ctx.GetRawContent(fieldDecl.Type))
		// if !inlined {
		// 	// Fixme: join names?
		// 	fieldname = fieldDecl.Names[0].Name
		// }
		if _, ok := fieldDecl.Type.(*ast.FuncType); ok {
			// method decl
			id := NewIdentity(ctx.module.Name, ctx.pkgPath, name.Name+"."+fieldDecl.Names[0].Name)
			if st.Methods == nil {
				st.Methods = make(map[string]Identity)
			}
			st.Methods[fieldDecl.Names[0].Name] = id
			fn := p.newFunc(ctx.module.Name, ctx.pkgPath, id.Name)
			fn.Content = string(ctx.GetRawContent(fieldDecl))
			fn.FileLine = ctx.FileLine(fieldDecl)
			fn.IsMethod = true
			fn.IsInterfaceMethod = true
		}
		p.collectTypes(ctx, fieldDecl.Type, st, inlined)
	}

	// get types.Interface and store in parser
	if obj := ctx.pkgTypeInfo.Defs[name]; obj != nil {
		if named, ok := obj.Type().(*types.Named); ok {
			iface := named.Underlying().(*types.Interface)
			p.interfaces[iface] = st.Identity
		}
	}

	return st, false
}
