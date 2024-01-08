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

//---------------- Golang Parser -----------------

// golang parser, used parse multle packages from the entire project
type goParser struct {
	modName               string
	homePageDir           string
	visited               map[string]bool
	processedPkgFunctions map[PkgPath]map[string]*Function
	processedPkgStruct    map[PkgPath]map[string]*Struct
}

// newGoParser
func newGoParser(modName, homePageDir string) *goParser {
	abs, err := filepath.Abs(homePageDir)
	if err != nil {
		panic(fmt.Sprintf("cannot get absolute path form homePageDir:%v", err))
	}
	return &goParser{
		modName:               modName,
		homePageDir:           abs,
		processedPkgFunctions: map[PkgPath]map[string]*Function{},
		processedPkgStruct:    map[PkgPath]map[string]*Struct{},
		visited:               map[string]bool{},
	}
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

func (p *goParser) associateStructWithMethods() {
	for _, fs := range p.processedPkgFunctions {
		for _, f := range fs {
			if f.IsMethod && f.AssociatedStruct != nil {
				// entrue the Struct has been visted
				if f.AssociatedStruct.FilePath != "" {
					if f.AssociatedStruct.Methods == nil {
						f.AssociatedStruct.Methods = map[string]*Function{}
					}
					f.AssociatedStruct.Methods[f.Name] = f
				}
			}
		}
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

	sysImports, projectImports, thirdPartyImports := p.seprateImports(f.Imports)
	pkgPath := p.pkgPathFromABS(filepath.Dir(filePath))
	ctx := &fileContext{
		filePath:          filePath,
		pkgPath:           pkgPath,
		bs:                bs,
		fset:              fset,
		sysImports:        sysImports,
		projectImports:    projectImports,
		thirdPartyImports: thirdPartyImports,
	}

	fileFuncs, _, err := p.inspectFile(ctx, f)
	return fileFuncs, err
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
			// parse structs
			struName := typDecl.Name.Name
			struDecl, ok := typDecl.Type.(*ast.StructType)
			if ok {
				st, ct := p.parseStruct(ctx, struName, struDecl)
				fileStructs[struName] = st
				cont = ct
			}
		}
		return cont
	})
	return fileFuncs, fileStructs, nil
}

// ParseDir parse a single go package in the dir
// If the dir has been visted, return cache
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
// ParseTilTheEnd parse the all go files from the starDir,
// and their related go files in the project recursively
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

	p.associateStructWithMethods()
	return nil
}

type MainStream struct {
	MainFunc string

	RelatedFunctions []SingleFunction

	RelatedStruct []SingleStruct
}

type SingleFunction struct {
	CallName string
	Content  string
}

type SingleStruct struct {
	Name    string
	Content string
}

func (p *goParser) getMain() (*MainStream, *Function) {
	m := &MainStream{
		RelatedFunctions: make([]SingleFunction, 0),
	}

	var mainFunc *Function

	for _, v := range p.processedPkgFunctions {
		for _, vv := range v {
			if vv.Name == "main" {
				mainFunc = vv
				m.MainFunc = vv.Content
				break
			}
		}
	}
	p.fillRelatedContent(mainFunc, &m.RelatedFunctions, &m.RelatedStruct)
	return m, mainFunc
}

func (p *goParser) fillRelatedContent(f *Function, fl *[]SingleFunction, sl *[]SingleStruct) {
	for call, ff := range f.InternalFunctionCalls {
		s := SingleFunction{
			CallName: call,
			Content:  ff.Content,
		}
		*fl = append(*fl, s)
		p.fillRelatedContent(ff, fl, sl)
	}

	for call, ff := range f.InternalMethodCalls {
		s := SingleFunction{
			CallName: call,
			Content:  ff.Content,
		}
		*fl = append(*fl, s)
		p.fillRelatedContent(ff, fl, sl)
		// for method which has been associated with struct, push the struct
		if ff.AssociatedStruct != nil && ff.AssociatedStruct.Content != "" {
			ss := SingleStruct{
				Name:    ff.AssociatedStruct.Name,
				Content: ff.AssociatedStruct.Content,
			}
			*sl = append(*sl, ss)
		}
		p.fillRelatedContent(ff, fl, sl)
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
	m, _ := p.getMain()

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
