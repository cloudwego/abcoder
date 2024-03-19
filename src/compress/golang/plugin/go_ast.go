package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//---------------- Golang Parser -----------------

// golang parser, used parse multle packages from the entire project
type goParser struct {
	homePageDir string           // absolute path to the home page of the repo
	visited     map[PkgPath]bool // visited packages
	modules     [][2]string      //  [name, abs-path] of modules, sorted by path length in descending order
	repo        Repository
}

// newGoParser
func newGoParser(name string, homePageDir string) *goParser {
	abs, err := filepath.Abs(homePageDir)
	if err != nil {
		panic(fmt.Sprintf("cannot get absolute path form homePageDir:%v", err))
	}

	p := &goParser{
		homePageDir: abs,
		visited:     map[string]bool{},
		repo:        NewRepository(name),
	}

	if err := p.collectGoMods(p.homePageDir); err != nil {
		panic(err)
	}
	return p
}

func (p *goParser) collectGoMods(startDir string) error {
	err := filepath.Walk(startDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, "go.mod") {
			return nil
		}
		name, content, err := getModuleName(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(p.homePageDir, filepath.Dir(path))
		if err != nil {
			return fmt.Errorf("module path %v is not in the repo", path)
		}
		p.repo.Modules[name] = NewModule(name, rel)
		deps, err := parseModuleFile(content)
		if err != nil {
			return err
		}
		for k, v := range deps {
			p.repo.Modules[name].Dependencies[k] = v
		}
		return nil
	})
	if err != nil {
		return err
	}

	var libs [][2]string
	for _, v := range p.repo.Modules {
		libs = append(libs, [2]string{v.Name, filepath.Join(p.homePageDir, v.Dir)})
	}
	// sort by dir in descending order
	sort.SliceStable(libs, func(i, j int) bool {
		return len(libs[i][1]) >= len(libs[j][1])
	})
	p.modules = libs
	return nil
}

// ParseRepo parse the entiry repo from homePageDir recursively until end
func (p *goParser) ParseRepo() error {
	for _, lib := range p.repo.Modules {
		startDir := filepath.Join(p.homePageDir, lib.Dir)
		filepath.WalkDir(startDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() || shouldIgnoreDir(path) {
				return nil
			}
			pkgPath := p.pkgPathFromABS(path)
			if err := p.ParsePackage(pkgPath); err != nil {
				return err
			}
			return nil
		})
	}

	p.associateStructWithMethods()
	return nil
}

// GetRepo return currently parsed golang AST
// Notice: To get completely parsed repo, you'd better call goParser.ParseRepo() before this
func (p *goParser) GetRepo() Repository {
	return p.repo
}

func (p *Module) GetDependency(mod string) string {
	// // search internal library first
	// if lib := p.Libraries[mod]; lib != nil {
	// 	return lib
	// }
	// match the prefix of name for each repo.Dependencies
	for k, v := range p.Dependencies {
		if strings.HasPrefix(mod, k) {
			return v
		}
	}
	return ""
}

// ToABS converts a local package path to absolute path
// If the path is not a local package, return empty string
// func (p *goParser) pkgPathToABS(path PkgPath) string {
// 	if !strings.HasPrefix(string(path), p.curMod) {
// 		return ""
// 	} else {
// 		return filepath.Join(p.homePageDir, strings.TrimPrefix(string(path), p.modName))
// 	}
// }

func (p *goParser) associateStructWithMethods() {
	for _, lib := range p.repo.Modules {
		for _, fs := range lib.Packages {
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
						def.Methods[name] = f.Identity
					}
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
func (p *goParser) GetNode(id Identity) (*Function, *Type, error) {
	if def := p.repo.GetFunction(id); def != nil {
		return def, nil, nil
	}
	if def := p.repo.GetType(id); def != nil {
		return nil, def, nil
	}

	lib := p.repo.Modules[id.ModPath]
	if lib == nil {
		return nil, nil, fmt.Errorf("library not defined: %v", id.ModPath)
	}
	if err := p.ParsePackage(id.PkgPath); err != nil {
		return nil, nil, err
	}

	pkg := lib.Packages[id.PkgPath]
	if pkg == nil {
		return nil, nil, fmt.Errorf("package not defined: %v", id.PkgPath)
	}
	for _, v := range pkg.Functions {
		if v.Name == id.Name {
			return v, nil, nil
		}
	}
	for _, v := range pkg.Types {
		if v.Name == id.Name {
			return nil, v, nil
		}
	}
	return nil, nil, nil
}

var errStop = errors.New("")

type EntityKind int

const (
	EKindError EntityKind = iota
	EKindFunc
	EKindType
	EKindConst
	EKindVar
)

type IE struct {
	Identity
	Kind EntityKind
}

func (p *goParser) SearchName(name string) (ids []Identity, err error) {
	filepath.Walk(p.homePageDir, func(path string, info fs.FileInfo, e error) error {
		if e != nil || info.IsDir() || shouldIgnoreFile(path) || shouldIgnoreDir(filepath.Dir(path)) || !strings.HasSuffix(path, ".go") {
			return nil
		}
		// go AST parse file
		fset := token.NewFileSet()
		fcontent, e := os.ReadFile(path)
		if e != nil {
			err = e
			return nil
		}
		f, e := parser.ParseFile(fset, path, fcontent, parser.SkipObjectResolution)
		if e != nil {
			err = e
			return nil
		}
		mod, _ := p.getModuleFromPath(filepath.Dir(path))
		pkg := p.pkgPathFromABS(filepath.Dir(path))
		// fmt.Printf("mod:%v, pkg:%v\n", mod, pkg)
		// match name
		for _, decl := range f.Decls {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				dname := decl.Name.Name
				if decl.Recv != nil {
					var tys []Identity
					getTypeName(fset, fcontent, decl.Recv.List[0].Type, &tys)
					if len(tys) > 0 {
						dname = fmt.Sprintf("%v.%v", tys[0].Name, dname)
					}
				}
				if dname == name {
					ids = append(ids, NewIdentity(mod, pkg, name))
				}
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						if spec.Name.Name == name {
							ids = append(ids, NewIdentity(mod, pkg, name))
						}
					case *ast.ValueSpec:
						for _, n := range spec.Names {
							if n.Name == name {
								ids = append(ids, NewIdentity(mod, pkg, name))
							}
						}

					}
				}
			}
		}
		return nil
	})
	return
}

// GetMain get main func on demands
func (p *goParser) GetMain(depth int) (*MainStream, *Function, error) {
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
		return nil, nil, err
	}

	// parse main dir and get root
	pkgPath := p.pkgPathFromABS(filepath.Dir(mainFile))
	if err := p.ParsePackage(pkgPath); err != nil {
		return nil, nil, err
	}

	var mainFunc *Function
	for _, lib := range p.repo.Modules {
		for _, pkg := range lib.Packages {
			for _, v := range pkg.Functions {
				if v.Name == "main" {
					mainFunc = v
					break
				}
			}
		}
	}

	// trace from root
	var m = &MainStream{}
	visited := cache(map[interface{}]bool{})
	err = p.fillRelatedContent(depth, mainFunc, &m.RelatedFunctions, &m.RelatedStruct, visited, p.GetNode)
	return m, mainFunc, err
}

// trace depends from bottom to top
// Notice: an AST node may be undefined on parse ondemands
func (p *goParser) fillRelatedContent(depth int, f *Function, fl *[]SingleFunction, sl *[]SingleStruct, visited cache, traceUndefined func(id Identity) (*Function, *Type, error)) error {
	if depth == 0 {
		return nil
	}
	if f == nil {
		return nil
	}

	var next []*Function
	for call, ff := range f.FunctionCalls {
		if visited.Visited(ff) {
			continue
		}
		if ff.ModPath != "" {
			continue
		}
		def := p.repo.GetFunction(ff)
		if def == nil {
			if traceUndefined == nil {
				return fmt.Errorf("undefiend InternalFunctionCalls %v for %v", ff.String(), f.Identity.String())
			} else {
				nf, _, err := traceUndefined(ff)
				if nf == nil {
					return fmt.Errorf("undefiend InternalFunctionCalls %v for %v", ff.String(), f.Identity.String())
				}
				if err != nil {
					return err
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

	for call, ff := range f.MethodCalls {
		if visited.Visited(ff) {
			continue
		}
		if ff.ModPath != "" {
			continue
		}
		def := p.repo.GetFunction(ff)
		if def == nil {
			if traceUndefined == nil {
				return fmt.Errorf("undefiend InternalMethodCalls: %v for %v", ff.String(), f.Identity.String())
			} else {
				nf, _, err := traceUndefined(ff)
				if nf == nil {
					return fmt.Errorf("undefiend InternalMethodCalls: %v for %v", ff.String(), f.Identity.String())
				}
				if err != nil {
					return err
				}
				def = nf
			}
		}
		content := def.Content

		var st *Type
		if def.AssociatedStruct != nil {
			st = p.repo.GetType(*def.AssociatedStruct)
			if st == nil {
				if traceUndefined == nil {
					return fmt.Errorf("undefiend AssociatedStruct: %v for %v", def.AssociatedStruct.String(), def.Identity.String())
				} else {
					_, ns, err := traceUndefined(ff)
					if ns == nil {
						return fmt.Errorf("undefiend Associated Struct: %v for %v", def.AssociatedStruct.String(), def.Identity.String())
					}
					if err != nil {
						return err
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

	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Missing home dir of the project to parse.")
		os.Exit(1)
	}

	homeDir := os.Args[1]

	search := ""
	if len(os.Args) >= 3 {
		search = os.Args[2]
	}

	p := newGoParser(homeDir, homeDir)
	var out = NewRepository(homeDir)

	if search == "" {
		// parse whole repo
		if err := p.ParseRepo(); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing repo: %v", err)
			os.Exit(1)
		}
		out = p.GetRepo()
	} else {
		// SPEC: seperate the packagepath and entity name by #
		ids := strings.Split(search, "#")

		if len(ids) == 1 {
			// parse pacakge
			pkgPath := ids[0]
			if err := p.ParsePackage(pkgPath); err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing package %v: %v", pkgPath, err)
				os.Exit(1)
			}
			repo := p.GetRepo()
			k, _ := p.getModuleFromPkg(pkgPath)
			out.Modules[k] = NewModule(repo.Modules[k].Name, repo.Modules[k].Dir)
			out.Modules[k].Packages[pkgPath] = repo.Modules[k].Packages[pkgPath]
		} else if len(ids) == 2 {
			if ids[0] == "" {
				//search mode
				idss, err := p.SearchName(ids[1])
				fmt.Fprintf(os.Stderr, "Error search %v:%v", ids[1], err)
				for _, id := range idss {
					loadNode(p, id.PkgPath, id.Name, &out)
				}
			} else {
				// parse entity
				pkgPath, name := ids[0], ids[1]
				loadNode(p, pkgPath, name, &out)
			}
		}
	}

	buf := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error marshalling functions to JSON:", err)
		os.Exit(1)
	}

	fmt.Println(buf.String())
}

func loadNode(p *goParser, pkgPath string, name string, out *Repository) {
	mod, _ := p.getModuleFromPkg(pkgPath)
	fp, sp, err := p.GetNode(NewIdentity(mod, PkgPath(pkgPath), name))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting node: %v", err)
		os.Exit(1)
	}
	repo := p.GetRepo()
	if out.Modules[mod] == nil {
		out.Modules[mod] = NewModule(repo.Modules[mod].Name, repo.Modules[mod].Dir)
	}
	if out.Modules[mod].Packages[pkgPath] == nil {
		out.Modules[mod].Packages[pkgPath] = NewPacakge(pkgPath)
	}
	if fp != nil {
		out.Modules[mod].Packages[pkgPath].Functions[name] = fp
	}
	if sp != nil {
		out.Modules[mod].Packages[pkgPath].Types[name] = sp
	}
}
