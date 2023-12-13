package main

import (
	"bytes"
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
	CallName                string // Has meaning only for its parent nodes
	PkgDir                  string
	IsMethod                bool
	AssociatedStruct        string
	Content                 string
	FunctionCalls           []Function // Holds internal function calls
	ThirdPartyFunctionCalls []string   // Holds third party function calls
	MethodCalls             []string
}

func (p *goParser) parseFile(filePath string) ([]Function, error) {
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
	projectImports := make(map[string]string)
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
			if strings.HasPrefix(importPath, p.modName) {
				projectImports[importAlias] = importPath
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

			var thirdPartyFunctionCalls, methodCalls []string
			var functionCalls []Function
			ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if ok {
					var funcName string
					switch expr := call.Fun.(type) {
					case *ast.SelectorExpr:
						funcName = expr.X.(*ast.Ident).Name + "." + expr.Sel.Name
						if importName, ok := projectImports[expr.X.(*ast.Ident).Name]; ok {
							// TODO: build.Import didn't work here. Calculate manually.
							suffix := strings.TrimPrefix(importName, p.modName)
							pkgDir := filepath.Join(p.homePageDir, suffix)
							functionCalls = append(functionCalls, Function{Name: expr.Sel.Name, CallName: funcName, PkgDir: pkgDir})
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
						functionCalls = append(functionCalls, Function{Name: funcName, CallName: funcName, PkgDir: filepath.Dir(filePath)})
						return true
					}
				}
				return true
			})

			funcs = append(funcs, Function{
				Name:                    funcDecl.Name.Name,
				PkgDir:                  filepath.Dir(filePath),
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

type goParser struct {
	modName     string
	homePageDir string
}

func getGoFilesInDir(dir string) []string {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	goFiles := make([]string, 0)
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".go") {
			goFiles = append(goFiles, filepath.Join(dir, file.Name()))
		}
	}
	return goFiles
}

func (p *goParser) ParseDir(dir string) []Function {
	functionList := make([]Function, 0)
	for _, f := range getGoFilesInDir(dir) {
		funcs, err := p.parseFile(f)
		if err != nil {
			fmt.Println("Error parsing file:", err)
			continue
		}
		functionList = append(functionList, funcs...)
	}
	return functionList
}

func main() {
	//if len(os.Args) < 3 {
	//	fmt.Println("Missing filepath argument or module name")
	//	os.Exit(1)
	//}
	//
	//funcs, err := parseFile(os.Args[1], os.Args[2])
	p := &goParser{modName: "a.com/b/c", homePageDir: "./tmp/demo"}
	functionList := make([]Function, 0)
	for _, f := range getGoFilesInDir("./tmp/demo") {
		funcs, err := p.parseFile(f)
		if err != nil {
			fmt.Println("Error parsing file:", err)
			continue
		}
		functionList = append(functionList, funcs...)
	}

	out := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(functionList)
	if err != nil {
		fmt.Println("Error marshalling functions to JSON:", err)
		os.Exit(1)
	}

	fmt.Println(out.String())
}
