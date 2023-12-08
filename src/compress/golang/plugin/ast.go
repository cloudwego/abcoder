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
)

type Function struct {
	Name             string
	IsMethod         bool
	AssociatedStruct string
	Content          string
	FunctionCalls    []string
	MethodCalls      []string
}

func ParseFile(filePath string) ([]Function, error) {
	fset := token.NewFileSet()

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("Error getting absolute path: %w", err)
	}

	bs, err := ioutil.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("Error reading file: %w", err)
	}
	content := string(bs)

	f, err := parser.ParseFile(fset, absPath, nil, parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("Parser error: %w", err)
	}

	var funcs []Function
	importedPackages := make(map[string]struct{})
	for _, imp := range f.Imports {
		importPath := imp.Path.Value
		importName := filepath.Base(importPath[1 : len(importPath)-1])
		importedPackages[importName] = struct{}{}
	}

	ast.Inspect(f, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		var associatedStruct string
		isMethod := funcDecl.Recv != nil
		if isMethod {
			t := funcDecl.Recv.List[0].Type
			if starExpr, ok := t.(*ast.StarExpr); ok {
				associatedStruct = starExpr.X.(*ast.Ident).Name
			} else {
				associatedStruct = t.(*ast.Ident).Name
			}
		}

		funcContent := content[fset.PositionFor(funcDecl.Pos(), false).Offset:fset.PositionFor(funcDecl.End(), false).Offset]

		var functionCalls, methodCalls []string
		ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
			callExpr, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			switch expr := callExpr.Fun.(type) {
			case *ast.SelectorExpr:
				ident, ok := expr.X.(*ast.Ident)
				if !ok {
					return true
				}
				funcName := ident.Name + "." + expr.Sel.Name
				if _, ok := importedPackages[ident.Name]; ok {
					functionCalls = append(functionCalls, funcName)
				} else {
					methodCalls = append(methodCalls, funcName)
				}
			case *ast.Ident:
				functionCalls = append(functionCalls, expr.Name)
			default:
				return true
			}

			return true
		})

		funcs = append(funcs, Function{
			Name:             funcDecl.Name.Name,
			IsMethod:         isMethod,
			AssociatedStruct: associatedStruct,
			Content:          funcContent,
			FunctionCalls:    functionCalls,
			MethodCalls:      methodCalls,
		})

		return true
	})

	return funcs, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Missing filepath argument")
		os.Exit(1)
	}

	filePath := os.Args[1]

	funcs, err := ParseFile(filePath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	jsonFuncs, err := json.Marshal(funcs)
	if err != nil {
		fmt.Println("Error converting to JSON:", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonFuncs))
}
