package main

import (
	"bytes"
	"encoding/json"
	"errors"
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
		if err != nil || !info.IsDir() || shouldIgnoreDir(path) {
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
				if def != nil {
					if def.Methods == nil {
						def.Methods = map[string]Identity{}
					}
					names := strings.Split(f.Name, ".")
					var name = names[0]
					if len(names) > 1 {
						name = names[1]
					}
					def.Methods[name] = Identity{f.PkgPath, f.Name}
				}
			}
		}
	}
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

// GetNode get a AST node from cache if parsed, or parse corresponding package and get the node
func (p *goParser) GetNode(id Identity) (*Function, *Struct) {
	if def := p.repo.GetFunction(id); def != nil {
		return def, nil
	}
	if def := p.repo.GetType(id); def != nil {
		return nil, def
	}
	dir := p.pkgPathToABS(id.PkgPath)
	println(dir)
	if err := p.ParseDir(dir); err != nil {
		return nil, nil
	}
	pkg := p.repo.Packages[id.PkgPath]
	for _, v := range pkg.Functions {
		if v.Name == id.Name {
			return v, nil
		}
	}
	for _, v := range pkg.Types {
		if v.Name == id.Name {
			return nil, v
		}
	}
	return nil, nil
}

var errStop = errors.New("")

// GetMain get main func on demands
func (p *goParser) GetMain(depth int) (*MainStream, *Function) {
	var mainFile string
	err := filepath.Walk(p.homePageDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil || info.IsDir() || shouldIgnoreFile(path) {
			return nil
		}
		file, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if hasMain(file) {
			mainFile = path
			return errStop
		}
		return nil
	})
	if err != errStop {
		return nil, nil
	}

	// parse main dir and get root
	mainDir := filepath.Dir(mainFile)
	if err := p.ParseDir(mainDir); err != nil {
		return nil, nil
	}
	pkg := p.repo.Packages[p.pkgPathFromABS(mainDir)]
	var mainFunc *Function
	for _, v := range pkg.Functions {
		if v.Name == "main" {
			mainFunc = v
			break
		}
	}

	// trace from root
	var m = &MainStream{}
	visited := cache(map[interface{}]bool{})
	p.fillRelatedContent(depth, mainFunc, &m.RelatedFunctions, &m.RelatedStruct, visited, p.GetNode)
	return m, mainFunc
}

// trace depends from bottom to top
// Notice: an AST node may be undefined on parse ondemands
func (p *goParser) fillRelatedContent(depth int, f *Function, fl *[]SingleFunction, sl *[]SingleStruct, visited cache, traceUndefined func(id Identity) (*Function, *Struct)) {
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
			if traceUndefined == nil {
				panic("undefiend function: " + ff.String())
			} else {
				nf, _ := traceUndefined(ff)
				if nf == nil {
					panic("undefiend function: " + ff.String())
				}
				def = nf
			}
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
			if traceUndefined == nil {
				panic("undefiend function: " + ff.String())
			} else {
				nf, _ := traceUndefined(ff)
				if nf == nil {
					panic("undefiend function: " + ff.String())
				}
				def = nf
			}
		}
		content := def.Content

		var st *Struct
		if def.AssociatedStruct != nil {
			st = p.repo.GetType(*def.AssociatedStruct)
			if st == nil {
				if traceUndefined == nil {
					panic("undefiend type: " + def.AssociatedStruct.String())
				} else {
					_, ns := traceUndefined(ff)
					if ns == nil {
						panic("undefiend type: " + def.AssociatedStruct.String())
					}
					st = ns
				}
			}
			// for method which has been associated with struct, push the struct
			if st.Content != "" {
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

		s := SingleFunction{
			CallName: call,
			Content:  content,
		}
		*fl = append(*fl, s)
		next = append(next, def)
	}

	for _, ff := range next {
		p.fillRelatedContent(depth-1, ff, fl, sl, visited, traceUndefined)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Missing home dir of the project to parse.")
		os.Exit(1)
	}
	//

	homeDir := os.Args[1]

	p := newGoParser("", homeDir)
	if err := p.ParseRepo(); err != nil {
		fmt.Println("Error parsing go files:", err)
		os.Exit(1)
	}

	//p.generateStruct()
	//m, _ := p.getMain(-1)

	out := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(p.repo)
	if err != nil {
		fmt.Println("Error marshalling functions to JSON:", err)
		os.Exit(1)
	}

	fmt.Println(out.String())
}
