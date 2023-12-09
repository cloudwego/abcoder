package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Function struct {
	Name                    string
	IsMethod                bool
	AssociatedStruct        string
	Content                 string
	FunctionCalls           []string // Holds internal function calls
	ThirdPartyFunctionCalls []string // Holds third party function calls
	MethodCalls             []string
}

func ParseFile(filePath string, modName string) ([]Function, error) {
	fset := token.NewFileSet()

	bs, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	f, err := parser.ParseFile(fset, filePath, bs, 0)
	if err != nil {
		return nil, err
	}

	thirdPartyImports := make(map[string]struct{})
	projectImports := make(map[string]struct{})
	for _, imp := range f.Imports {
		importPath := imp.Path.Value[1 : len(imp.Path.Value)-1] // remove the quotes
		importBaseName := filepath.Base(importPath)
		importAlias := importBaseName // use the base name as alias by default

		// Check if user has defined an alias for current import
		if imp.Name != nil {
			importAlias = imp.Name.Name // update the alias
		}

		isSysPkg := !strings.Contains(strings.Split(importPath, "/")[0], ".")

		// Ignoring golang standard libraries（like net/http）
		if !isSysPkg {
			// Distinguish between project packages and third party packages
			if strings.HasPrefix(importPath, modName) {
				projectImports[importAlias] = struct{}{}
			} else {
				thirdPartyImports[importAlias] = struct{}{}
			}
		}
	}

	var funcs []Function
	ast.Inspect(f, func(node ast.Node) bool {
		funcDecl, ok := node.(*ast.FuncDecl)
		if ok {
			var associatedStruct string
			isMethod := funcDecl.Recv != nil
			if isMethod {
				associatedStruct = funcDecl.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
			}

			pos := fset.PositionFor(node.Pos(), false).Offset
			end := fset.PositionFor(node.End(), false).Offset
			content := string(bs[pos:end])

			var functionCalls, thirdPartyFunctionCalls, methodCalls []string
			ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if ok {
					var funcName string
					switch expr := call.Fun.(type) {
					case *ast.SelectorExpr:
						funcName = expr.X.(*ast.Ident).Name + "." + expr.Sel.Name
						if _, ok = projectImports[expr.X.(*ast.Ident).Name]; ok {
							functionCalls = append(functionCalls, funcName)
							return true
						}
						if _, ok = thirdPartyImports[expr.X.(*ast.Ident).Name]; ok {
							thirdPartyFunctionCalls = append(thirdPartyFunctionCalls, funcName)
							return true
						}

						methodCalls = append(methodCalls, funcName)
						return true
					case *ast.Ident:
						funcName = expr.Name
						functionCalls = append(functionCalls, funcName)
						return true
					}
				}
				return true
			})

			funcs = append(funcs, Function{
				Name:                    funcDecl.Name.Name,
				IsMethod:                isMethod,
				AssociatedStruct:        associatedStruct,
				Content:                 content,
				FunctionCalls:           functionCalls,
				ThirdPartyFunctionCalls: thirdPartyFunctionCalls,
				MethodCalls:             methodCalls,
			})
		}
		return true
	})

	return funcs, nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Missing filepath argument or module name")
		os.Exit(1)
	}

	funcs, err := ParseFile(os.Args[1], os.Args[2])

	if err != nil {
		fmt.Println("Error parsing file:", err)
		os.Exit(1)
	}

	bs, err := json.Marshal(funcs)
	if err != nil {
		fmt.Println("Error marshalling functions to JSON:", err)
		os.Exit(1)
	}

	fmt.Println(string(bs))
}
