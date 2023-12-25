package main

import (
	"bufio"
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
	sysImports := make(map[string]string)
	for _, imp := range f.Imports {
		importPath := imp.Path.Value[1 : len(imp.Path.Value)-1] // remove the quotes
		importBaseName := filepath.Base(importPath)
		importAlias := importBaseName // use the base name as alias by default

		// Check if user has defined an alias for current import
		if imp.Name != nil {
			importAlias = imp.Name.Name // update the alias
		}

		isSysPkg := !strings.Contains(strings.Split(importPath, "/")[0], ".")

		if isSysPkg {
			sysImports[importAlias] = importPath
		}

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
						funcName = x.(*ast.Ident).Name + "." + expr.Sel.Name
						if importName, ok := projectImports[x.(*ast.Ident).Name]; ok {
							// TODO: build.Import didn't work here. Calculate manually.
							suffix := strings.TrimPrefix(importName, p.modName)
							pkgDir := filepath.Join(p.homePageDir, suffix)
							functionCalls = append(functionCalls, Function{Name: expr.Sel.Name, CallName: funcName, PkgDir: pkgDir})
							return true
						}
						if _, ok = thirdPartyImports[x.(*ast.Ident).Name]; ok {
							thirdPartyFunctionCalls = append(thirdPartyFunctionCalls, funcName)
							return true
						}

						// skip sys imports
						if _, ok = sysImports[x.(*ast.Ident).Name]; ok {
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
	modName      string
	homePageDir  string
	processedPkg map[string][]Function
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

func (p *goParser) ParseDir(dir string) ([]Function, bool) {
	// unify dir ./xxx/xxx -> xxx/xxx
	relativePrefix := "./"
	if strings.HasPrefix(dir, relativePrefix) {
		dir = strings.TrimPrefix(dir, relativePrefix)
	}

	if p.processedPkg[dir] != nil {
		return p.processedPkg[dir], false
	}
	functionList := make([]Function, 0)
	for _, f := range getGoFilesInDir(dir) {
		funcs, err := p.parseFile(f)
		if err != nil {
			fmt.Println("Error parsing file:", err)
			continue
		}
		functionList = append(functionList, funcs...)
	}
	p.processedPkg[dir] = functionList
	return functionList, true
}

// TODO: Parallel transformation
func (p *goParser) ParseTilTheEnd(startDir string) {
	if p.modName == "" {
		var err error
		p.modName, err = getModuleName(p.homePageDir + "/go.mod")
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}
	functionList, _ := p.ParseDir(startDir)
	for _, f := range functionList {
		for _, fc := range f.FunctionCalls {
			if p.processedPkg[fc.PkgDir] != nil {
				continue
			}
			p.ParseTilTheEnd(fc.PkgDir)
		}
	}
	return
}

type MainStream struct {
	MainFunc string

	RelatedFunctions []SingleFunction
}

type SingleFunction struct {
	CallName string
	Content  string
}

func (p *goParser) generate() *MainStream {
	m := &MainStream{
		RelatedFunctions: make([]SingleFunction, 0),
	}

	var functionCalledInMain []Function

Out:
	for _, v := range p.processedPkg {
		for _, vv := range v {
			if vv.Name == "main" {
				m.MainFunc = vv.Content
				functionCalledInMain = vv.FunctionCalls
				break Out
			}
		}
	}

	p.fillFunctionContent(functionCalledInMain, &m.RelatedFunctions)
	return m
}

func (p *goParser) fillFunctionContent(f []Function, fl *[]SingleFunction) {
	for _, ff := range f {
		for _, pf := range p.processedPkg[ff.PkgDir] {
			if pf.IsMethod {
				// Skip method here
				continue
			}
			if ff.Name == pf.Name {
				s := SingleFunction{
					CallName: ff.CallName,
					Content:  pf.Content,
				}
				*fl = append(*fl, s)

				if len(pf.FunctionCalls) != 0 {
					p.fillFunctionContent(pf.FunctionCalls, fl)
				}
			}
		}
	}
}

func getModuleName(modFilePath string) (string, error) {
	file, err := os.Open(modFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "module") {
			// Assuming 'module' keyword is followed by module name
			parts := strings.Split(line, " ")
			if len(parts) > 1 {
				return parts[1], nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to scan file: %v", err)
	}

	return "", nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Missing home dir of the project to parse.")
		os.Exit(1)
	}
	//

	homeDir := os.Args[1]

	p := &goParser{modName: "", homePageDir: homeDir, processedPkg: make(map[string][]Function)}
	p.ParseTilTheEnd(p.homePageDir)

	m := p.generate()

	out := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(m)
	if err != nil {
		fmt.Println("Error marshalling functions to JSON:", err)
		os.Exit(1)
	}

	fmt.Println(out.String())
}
