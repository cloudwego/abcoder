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

// PkgPath is the import path of a package, it is either absolute path or url
type PkgPath = string

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
	ThirdPartyFunctionCalls map[string]*ThirdPartyCall

	// call to internal methods, key is the {{object.funcName}}
	InternalMethodCalls map[string]*Function

	// call to thrid-party methods, key is the {{object.funcName}}
	ThirdPartyMethodCalls map[string]*ThirdPartyCall
}

// ThirdPartyCall holds location information about a third party declaration
type ThirdPartyCall struct {
	PkgPath  string // Import Path of the third party package
	Identity string // Unique Name of declaration (FunctionName, or StructName.MethodName etc)
}

// Struct holds the information about a struct
type Struct struct {
	Name    string // Name of the struct
	PkgPath        // Path to the package where the struct is defined
	Content string // struct declaration content

	// related local structs in fields, key is {{pkgName.typName}} or {{typeName}}, val is declaration of the struct
	InternalStructs map[string]*Struct

	// related third party structs in fields,
	// ex: type A struct { B pkg.B }, pkg.B is a child of A, key is "pkg.B"
	ThirdPartyChildren map[string]ThirdPartyCall

	// method name to Function
	Methods map[string]*Function
}

// ToABS converts a local package path to absolute path
// If the path is not a local package, return empty string
func (p *goParser) pkgPathToABS(path PkgPath) string {
	if !strings.HasPrefix(string(path), p.modName) {
		return ""
	} else {
		return filepath.Join(p.homePageDir, strings.TrimPrefix(string(path), p.modName))
	}
}

// FromABS converts an absolute path to local mod path
func (p *goParser) pkgPathFromABS(path string) PkgPath {
	if rel, _ := filepath.Rel(p.homePageDir, path); rel == "" {
		return ""
	} else {
		return filepath.Join(p.modName, rel)
	}
}

// parseFile parse single go file and return all functions in it
// warning: this function has no cache, do not call it with repeated file
func (p *goParser) parseFile(filePath string) (map[string]*Function, error) {
	fset := token.NewFileSet()

	bs, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	f, err := parser.ParseFile(fset, filePath, bs, 0)
	if err != nil {
		return nil, err
	}

	thirdPartyImports := make(map[string]string)
	projectImports := make(map[string]string)
	sysImports := make(map[string]string)
	for _, imp := range f.Imports {
		importPath := imp.Path.Value[1 : len(imp.Path.Value)-1] // remove the quotes
		var importAlias string
		if imp.Name != nil {
			importAlias = imp.Name.Name
		} else {
			importAlias = filepath.Base(importPath) // use the base name as alias by default
		}

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
				thirdPartyImports[importAlias] = importPath
			}
		}
	}

	pkgPath := p.pkgPathFromABS(filepath.Dir(filePath))
	fileFuncs := map[string]*Function{}

	ast.Inspect(f, func(node ast.Node) bool {
		funcDecl, ok := node.(*ast.FuncDecl)
		if ok {
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
				associatedStruct = p.getOrSetStruct(p.modName, structName)
			}

			pos := fset.PositionFor(node.Pos(), false).Offset
			end := fset.PositionFor(node.End(), false).Offset
			content := string(bs[pos:end])

			var thirdPartyMethodCalls, thirdPartyFunctionCalls = map[string]*ThirdPartyCall{}, map[string]*ThirdPartyCall{}
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
						if impt, ok := projectImports[x.(*ast.Ident).Name]; ok {
							functionCalls[funcName] = p.getOrSetFunc(impt, expr.Sel.Name)
							return true
						}
						// third-party function calls
						if impt, ok := thirdPartyImports[x.(*ast.Ident).Name]; ok {
							thirdPartyFunctionCalls[funcName] = &ThirdPartyCall{PkgPath: impt, Identity: expr.Sel.Name}
							return true
						}
						// WHY: skip sys imports?
						if _, ok := sysImports[x.(*ast.Ident).Name]; ok {
							// internalFunctionCalls[funcName] = p.getOrSetFunc(impt, expr.Sel.Name)
							return true
						}

						// Fallback must be method calls
						// FIXME: get type info of object
						f := p.getOrSetFunc(pkgPath, funcName)
						f.IsMethod = true
						f.AssociatedStruct = &Struct{Name: x.(*ast.Ident).Name}
						methodCalls[funcName] = f
						// TODO: seperate internal method and third-party method

						return true
					case *ast.Ident:
						funcName = expr.Name
						if !isGoBuiltinFunc(funcName) {
							functionCalls[funcName] = p.getOrSetFunc(pkgPath, funcName)
						}
						return true
					}
				}
				return true
			})

			// update detailed function call info
			f := p.getOrSetFunc(pkgPath, funcDecl.Name.Name)
			*f = Function{
				Name:                    funcDecl.Name.Name,
				PkgPath:                 pkgPath,
				FilePath:                filePath,
				IsMethod:                isMethod,
				AssociatedStruct:        associatedStruct,
				Content:                 content,
				InternalFunctionCalls:   functionCalls,
				ThirdPartyFunctionCalls: thirdPartyFunctionCalls,
				InternalMethodCalls:     methodCalls,
				ThirdPartyMethodCalls:   thirdPartyMethodCalls,
			}
			fileFuncs[funcDecl.Name.Name] = f
		}
		return true
	})

	return fileFuncs, nil
}

type goParser struct {
	modName               string
	homePageDir           string
	visited               map[string]bool
	processedPkgFunctions map[PkgPath]map[string]*Function
	processedPkgStruct    map[PkgPath]map[string]*Struct
}

func newGoParser(modName, homePageDir string) *goParser {
	abs, _ := filepath.Abs(homePageDir)
	return &goParser{
		modName:               modName,
		homePageDir:           abs,
		processedPkgFunctions: map[PkgPath]map[string]*Function{},
		processedPkgStruct:    map[PkgPath]map[string]*Struct{},
		visited:               map[string]bool{},
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

func (p *goParser) ParseDir(dir string) (map[string]*Function, error) {
	// unify dir ./xxx/xxx -> xxx/xxx
	if !strings.HasPrefix(dir, "/") {
		dir = filepath.Join(p.homePageDir, dir)
	}
	pkgPath := p.pkgPathFromABS(dir)
	if p.visited[pkgPath] {
		return p.processedPkgFunctions[pkgPath], nil
	}
	for _, f := range getGoFilesInDir(dir) {
		_, err := p.parseFile(f)
		if err != nil {
			return nil, err
		}
	}
	p.visited[pkgPath] = true
	return p.processedPkgFunctions[pkgPath], nil
}

// TODO: Parallel transformation
func (p *goParser) ParseTilTheEnd(startDir string) error {
	if p.modName == "" {
		var err error
		p.modName, err = getModuleName(p.homePageDir + "/go.mod")
		if err != nil {
			return err
		}
	}
	functionList, err := p.ParseDir(startDir)
	if err != nil {
		return err
	}
	for _, f := range functionList {
		// Notice: local funcs has been parsed in ParseDir
		for _, fc := range f.InternalFunctionCalls {
			if fc.FilePath != "" || fc.IsMethod {
				continue
			}
			if err := p.ParseTilTheEnd(p.pkgPathToABS(fc.PkgPath)); err != nil {
				return err
			}
		}
	}
	return nil
}

type MainStream struct {
	MainFunc string

	RelatedFunctions []SingleFunction
}

type SingleFunction struct {
	CallName string
	Content  string
}

// TODO: generate struct
// func (p *goParser) generateStruct() {
// 	processedStruct := make(map[string]*Struct)
// 	for pkgName, fs := range p.processedPkgFunctions {
// 		if len(fs) == 0 {
// 			continue
// 		}
// 		for _, f := range fs {
// 			if !f.IsMethod {
// 				continue
// 			}
// 			if processedStruct[f.AssociatedStruct] == nil {
// 				st := &Struct{Name: f.AssociatedStruct, methods: make(map[string]Function)}
// 				st.methods[f.Name] = f
// 				processedStruct[f.AssociatedStruct] = st
// 				continue
// 			}

// 			processedStruct[f.AssociatedStruct].methods[f.Name] = f
// 			continue
// 		}

// 		if len(processedStruct) == 0 {
// 			continue
// 		}

// 		structList := make([]Struct, 0, len(processedStruct))

// 		for _, s := range processedStruct {
// 			structList = append(structList, *s)
// 		}

// 		p.processedPkgStruct[pkgName] = structList
// 	}
// }

func (p *goParser) getMain() *MainStream {
	m := &MainStream{
		RelatedFunctions: make([]SingleFunction, 0),
	}

	var functionCalledInMain = map[string]*Function{}

Out:
	for _, v := range p.processedPkgFunctions {
		for _, vv := range v {
			if vv.Name == "main" {
				m.MainFunc = vv.Content
				for k, v := range vv.InternalFunctionCalls {
					functionCalledInMain[k] = v
				}
				for k, v := range vv.InternalMethodCalls {
					functionCalledInMain[k] = v
				}
				//TODO: add methods
				break Out
			}
		}
	}
	p.fillFunctionContent(functionCalledInMain, &m.RelatedFunctions)
	return m
}

func (p *goParser) fillFunctionContent(f map[string]*Function, fl *[]SingleFunction) {
	for call, ff := range f {
		s := SingleFunction{
			CallName: call,
			Content:  ff.Content,
		}

		*fl = append(*fl, s)

		if len(ff.InternalFunctionCalls) != 0 {
			p.fillFunctionContent(ff.InternalFunctionCalls, fl)
		}
		//TODO: add methods
	}
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

	p := newGoParser("", homeDir)
	if err := p.ParseTilTheEnd(p.homePageDir); err != nil {
		fmt.Println("Error parsing go files:", err)
		os.Exit(1)
	}

	// p.generateStruct()
	m := p.getMain()

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
