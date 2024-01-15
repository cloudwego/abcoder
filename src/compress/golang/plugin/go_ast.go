package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//---------------- Golang Parser -----------------

// golang parser, used parse multle packages from the entire project
type goParser struct {
	modName     string
	homePageDir string
	visited     map[string]bool
	repo        Repository
}

// newGoParser
func newGoParser(modName, homePageDir string) *goParser {
	abs, err := filepath.Abs(homePageDir)
	if err != nil {
		panic(fmt.Sprintf("cannot get absolute path form homePageDir:%v", err))
	}

	if modName == "" {
		var err error
		modName, err = getModuleName(homePageDir + "/go.mod")
		if err != nil {
			panic(err.Error())
		}
	}

	p := &goParser{
		modName:     modName,
		homePageDir: abs,
		visited:     map[string]bool{},
		repo:        NewRepository(modName),
	}
	return p
}

// ParseRepo parse the entiry repo from homePageDir recursively until end
func (p *goParser) ParseRepo() error {
	err := filepath.Walk(p.homePageDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil || !info.IsDir() || shouldIgnore(path) {
			return nil
		}

		if err := p.ParseDir(path); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	p.associateStructWithMethods()
	return nil
}

// GetRepo return currently parsed golang AST
// Notice: To get completely parsed repo, you'd better call goParser.ParseRepo() before this
func (p *goParser) GetRepo() Repository {
	if len(p.repo.Packages) == 0 {
		_ = p.ParseRepo()
	}
	return p.repo
}

func shouldIgnore(path string) bool {
	return strings.Contains(path, ".git")
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
	if rel, _ := filepath.Rel(p.homePageDir, path); rel != "" {
		path = rel
	}
	return filepath.Join(p.modName, path)
}

func (p *goParser) associateStructWithMethods() {
	for _, fs := range p.repo.Packages {
		for _, f := range fs.Functions {
			if f.IsMethod && f.AssociatedStruct != nil {
				def := p.repo.GetType(*f.AssociatedStruct)
				// entrue the Struct has been visted
				if def != nil && def.FilePath != "" {
					if def.Methods == nil {
						def.Methods = map[string]Identity{}
					}
					def.Methods[strings.Split(f.Name, ".")[1]] = Identity{f.PkgPath, f.Name}
				}
			}
		}
	}
}

// TODO: Parallel transformation
// ParseTilTheEnd parse the all go files from the starDir,
// and their related go files in the project recursively
func (p *goParser) ParseTilTheEnd(startDir string) error {
	if err := p.ParseDir(startDir); err != nil {
		return err
	}
	for path, pkg := range p.repo.Packages {
		// ignore third-party packages
		if !strings.Contains(path, p.modName) {
			continue
		}
		for _, f := range pkg.Functions {
			// Notice: local funcs has been parsed in ParseDir
			for _, fc := range f.InternalFunctionCalls {
				if p.visited[fc.PkgPath] {
					continue
				}
				if err := p.ParseTilTheEnd(p.pkgPathToABS(fc.PkgPath)); err != nil {
					return err
				}
			}
			for _, fc := range f.InternalMethodCalls {
				if p.visited[fc.PkgPath] {
					continue
				}
				if err := p.ParseTilTheEnd(p.pkgPathToABS(fc.PkgPath)); err != nil {
					return err
				}
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
	CallName string
	Content  string
}

type cache map[interface{}]bool

func (c cache) Visited(val interface{}) bool {
	ok := c[val]
	if !ok {
		c[val] = true
	}
	return ok
}

func (p *goParser) getMain(depth int) (*MainStream, *Function) {
	m := &MainStream{
		RelatedFunctions: make([]SingleFunction, 0),
	}

	var mainFunc *Function

	for _, v := range p.repo.Packages {
		for _, vv := range v.Functions {
			if vv.Name == "main" {
				mainFunc = vv
				m.MainFunc = vv.Content
				break
			}
		}
	}
	visited := cache(map[interface{}]bool{})
	p.fillRelatedContent(depth, mainFunc, &m.RelatedFunctions, &m.RelatedStruct, visited)
	return m, mainFunc
}

func (p *goParser) fillRelatedContent(depth int, f *Function, fl *[]SingleFunction, sl *[]SingleStruct, visited cache) {
	if depth == 0 {
		return
	}
	if f == nil {
		return
	}

	var next []*Function
	for call, ff := range f.InternalFunctionCalls {
		if visited.Visited(ff) {
			continue
		}
		def := p.repo.GetFunction(ff)
		if def == nil {
			// continue // TODO: fixme
			panic("undefiend function: " + ff.String())
		}
		s := SingleFunction{
			CallName: call,
			Content:  def.Content,
		}
		*fl = append(*fl, s)
		next = append(next, def)
	}

	for call, ff := range f.InternalMethodCalls {
		if visited.Visited(ff) {
			continue
		}
		def := p.repo.GetFunction(ff)
		if def == nil {
			// continue
			// TODO: fixme
			panic("undefiend function: " + ff.String())
		}
		content := def.Content
		var st *Struct
		if def.AssociatedStruct != nil {
			st = p.repo.GetType(Identity{def.AssociatedStruct.PkgPath, def.AssociatedStruct.Name})
		}
		if st != nil && st.TypeKind != TypeKindStruct {
			content = st.Content
		}
		s := SingleFunction{
			CallName: call,
			Content:  content,
		}
		*fl = append(*fl, s)
		next = append(next, def)

		// for method which has been associated with struct, push the struct
		if st != nil && st.Content != "" {
			if visited.Visited(st) {
				continue
			}
			ss := SingleStruct{
				CallName: call,
				Content:  st.Content,
			}
			*sl = append(*sl, ss)
		}
	}

	for _, ff := range next {
		p.fillRelatedContent(depth-1, ff, fl, sl, visited)
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
	m, _ := p.getMain(-1)

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
