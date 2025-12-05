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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/cloudwego/abcoder/lang/uniast"
)

//---------------- Golang Parser -----------------

// golang parser, used parse multle packages from the entire project
type GoParser struct {
	homePageDir string           // absolute path to the home page of the repo
	visited     map[PkgPath]bool // visited packages
	modules     []moduleInfo     //  [name, abs-path] of modules, sorted by path length in descending order
	repo        Repository
	opts        Options
	interfaces  map[*types.Interface]Identity
	types       map[types.Type]Identity
	files       map[string][]byte
	exclues     []*regexp.Regexp
}

type moduleInfo struct {
	name string
	dir  string
	path string
}

func newModuleInfo(name string, dir string, path string) moduleInfo {
	return moduleInfo{
		name: name,
		dir:  dir,
		path: path,
	}
}

func NewParser(name string, homePageDir string, o Options) *GoParser {
	return newGoParser(name, homePageDir, o)
}

// newGoParser
func newGoParser(name string, homePageDir string, opts Options) *GoParser {
	abs, err := filepath.Abs(homePageDir)
	if err != nil {
		panic(fmt.Sprintf("cannot get absolute path form homePageDir:%v", err))
	}

	p := &GoParser{
		homePageDir: abs,
		visited:     map[string]bool{},
		repo:        NewRepository(name),
		interfaces:  map[*types.Interface]Identity{},
		types:       map[types.Type]Identity{},
		files:       map[string][]byte{},
	}

	if opts.Excludes != nil {
		p.exclues = compileExcludes(opts.Excludes)
	}

	if err := p.collectGoMods(p.homePageDir); err != nil {
		panic(err)
	}

	p.opts = opts
	return p
}

func (p *GoParser) collectGoMods(startDir string) error {
	err := filepath.Walk(startDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, "go.mod") {
			return nil
		}

		name, err := getModuleName(path)
		if err != nil {
			return fmt.Errorf("failed to get module name: %w", err)
		}

		rel, err := filepath.Rel(p.homePageDir, filepath.Dir(path))
		if err != nil {
			return fmt.Errorf("module path %v is not in the repo", path)
		}
		p.repo.Modules[name] = newModule(name, rel)
		p.modules = append(p.modules, newModuleInfo(name, rel, name))

		deps, err := getDeps(filepath.Dir(path))
		if err != nil {
			return err
		}
		for k, v := range deps {
			p.repo.Modules[name].Dependencies[k] = v
			p.modules = append(p.modules, newModuleInfo(k, "", v))
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

type replace struct {
	Path    string `json:"Path"`
	Version string `json:"Version"`
	Dir     string `json:"Dir"`
	GoMod   string `json:"GoMod"`
}

type dep struct {
	Module struct {
		Path     string   `json:"Path"`
		Version  string   `json:"Version"`
		Replace  *replace `json:"Replace,omitempty"`
		Indirect bool     `json:"Indirect"`
		Dir      string   `json:"Dir"`
		GoMod    string   `json:"GoMod"`
	} `json:"Module"`
}

func getDeps(dir string) (map[string]string, error) {
	// run go mod tidy first to ensure all dependencies are resolved
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute 'go mod tidy', err: %v, output: %s", err, string(output))
	}

	if hasNoDeps(filepath.Join(dir, "go.mod")) {
		return map[string]string{}, nil
	}

	cmd = exec.Command("go", "list", "-json", "all")
	cmd.Dir = dir
	output, err = cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute 'go list -json all', err: %v, output: %s", err, string(output))
	}

	deps := make(map[string]string)
	decoder := json.NewDecoder(bytes.NewReader(output))
	for {
		var mod dep
		if err := decoder.Decode(&mod); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to decode json: %v", err)
		}
		module := mod.Module
		// golang internal package, ignore it.
		if module.Path == "" {
			continue
		}
		if module.Replace != nil {
			deps[module.Path] = module.Replace.Path + "@" + module.Replace.Version
		} else {
			if module.Version != "" {
				deps[module.Path] = module.Path + "@" + module.Version
			} else {
				// If no version, it's a local package. So we use local commit as version
				commit, err := getCommitHash(dir)
				if err != nil {
					deps[module.Path] = module.Path
				} else {
					deps[module.Path] = module.Path + "@" + commit
				}
			}
		}
	}

	return deps, nil
}

// ParseRepo parse the entiry repo from homePageDir recursively until end
func (p *GoParser) ParseRepo() (Repository, error) {
	for _, lib := range p.modules {
		if strings.Contains(lib.path, "@") {
			continue
		}
		mod := p.repo.Modules[lib.name]
		if err := p.ParseModule(mod, filepath.Join(p.homePageDir, mod.Dir)); err != nil {
			return p.getRepo(), err
		}
	}
	p.associateStructWithMethods()
	p.associateImplements()
	fmt.Fprintf(os.Stderr, "total call packages.Load %d times\n", loadCount)
	return p.getRepo(), nil
}

func (p *GoParser) ParseModule(mod *Module, dir string) (err error) {
	// run go mod tidy before parse
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	buf := bytes.NewBuffer(nil)
	cmd.Stderr = buf
	cmd.Stdout = buf
	go func() {
		sc := bufio.NewScanner(buf)
		// scan and print
		for sc.Scan() {
			fmt.Fprintln(os.Stderr, sc.Text())
		}
	}()
	fmt.Fprintf(os.Stderr, "running go mod tidy in %s ...\n", dir)
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "run go mod tidy failed in %s: %v\n", dir, buf.String())
	}

	filepath.Walk(dir, func(path string, info fs.FileInfo, e error) error {
		if info != nil && info.IsDir() && filepath.Base(path) == ".git" {
			return filepath.SkipDir
		}
		if e != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(p.homePageDir, path)
		mod.Files[rel] = NewFile(rel)
		return nil
	})

	if p.opts.LoadByPackages {
		var errs []error
		filepath.Walk(dir, func(path string, info fs.FileInfo, e error) error {
			if e != nil || !info.IsDir() || shouldIgnoreDir(path) {
				return nil
			}
			for _, exclude := range p.exclues {
				if exclude.MatchString(path) {
					return nil
				}
			}
			if err := p.parsePackage(p.pkgPathFromABS(path)); err != nil {
				errs = append(errs, err)
			}
			return nil
		})
		if len(errs) > 0 {
			return fmt.Errorf("parse package failed: %v", errs)
		}
		return nil
	} else {
		return p.loadPackages(mod, dir, "./...")
	}
}

// getRepo return currently parsed golang AST
// Notice: To get completely parsed repo, you'd better call goParser.ParseRepo() before this
func (p *GoParser) getRepo() Repository {
	return p.repo
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

func (p *GoParser) associateStructWithMethods() {
	for _, lib := range p.repo.Modules {
		for _, fs := range lib.Packages {
			for _, f := range fs.Functions {
				if f.IsMethod && f.Receiver != nil {
					def := p.repo.GetType(f.Receiver.Type)
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

// getNode get a AST node from cache if parsed, or parse corresponding package and get the node
func (p *GoParser) getNode(id Identity) (interface{}, error) {
	if def := p.repo.GetFunction(id); def != nil {
		return def, nil
	}
	if def := p.repo.GetType(id); def != nil {
		return def, nil
	}
	if def := p.repo.GetVar(id); def != nil {
		return def, nil
	}

	lib := p.repo.Modules[id.ModPath]
	if lib == nil {
		lib = newModule(id.ModPath, "")
		p.repo.Modules[id.ModPath] = lib
	}
	if err := p.parsePackage(id.PkgPath); err != nil {
		return nil, err
	}

	pkg := lib.Packages[id.PkgPath]
	if pkg == nil {
		return nil, fmt.Errorf("package not defined: %v", id.PkgPath)
	}
	if def := pkg.Functions[id.Name]; def != nil {
		return def, nil
	}
	if def := pkg.Types[id.Name]; def != nil {
		return def, nil
	}
	if def := pkg.Vars[id.Name]; def != nil {
		return def, nil
	}
	return nil, nil
}

func (p *GoParser) searchName(name string) (ids []Identity, err error) {
	filepath.Walk(p.homePageDir, func(path string, info fs.FileInfo, e error) error {
		if e != nil || info.IsDir() || shouldIgnoreFile(path) || shouldIgnoreDir(filepath.Dir(path)) || !strings.HasSuffix(path, ".go") {
			return nil
		}
		mod := p.pkgPathFromABS(path)
		m := p.repo.Modules[mod]
		if m == nil {
			dir, _ := filepath.Rel(p.homePageDir, path)
			m = newModule(mod, dir)
			p.repo.Modules[mod] = m
		}
		pkg := p.pkgPathFromABS(filepath.Dir(path))
		// go AST parse file
		fset := token.NewFileSet()
		fcontent := p.getFileBytes(path)
		file, e := parser.ParseFile(fset, path, fcontent, parser.SkipObjectResolution)
		if e != nil {
			err = e
			return nil
		}
		impts, err := p.parseImports(fset, fcontent, m, file.Imports)
		if err != nil {
			return err
		}
		tids, e := p.searchOnFile(file, fset, fcontent, m.Name, pkg, impts, name)
		if e != nil {
			err = e
			return nil
		}
		ids = append(ids, tids...)
		return nil
	})
	return
}

// getRelativeOrBasePath returns the relative path if possible, otherwise the base path.
func getRelativeOrBasePath(homePageDir string, fset *token.FileSet, pos token.Pos) string {
	relp, err := filepath.Rel(homePageDir, fset.Position(pos).Filename)
	if err == nil {
		return relp
	}
	return filepath.Base(fset.Position(pos).Filename)
}

func (p *GoParser) exportFileLine(fset *token.FileSet, decl ast.Node) (ret FileLine) {
	ret.File = getRelativeOrBasePath(p.homePageDir, fset, decl.Pos())
	ret.Line = fset.Position(decl.Pos()).Line
	return
}

func (p *GoParser) searchOnFile(file *ast.File, fset *token.FileSet, fcontent []byte, mod string, pkg string, impt *importInfo, name string) (ids []Identity, err error) {
	for _, decl := range file.Decls {
		// println(string(GetRawContent(fset, fcontent, decl)))
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			dname := decl.Name.Name
			var receiver *Receiver
			if decl.Recv != nil && strings.Contains(name, ".") {
				var m = map[string]Identity{}
				tname, isPointer := p.mockTypes(decl.Recv.List[0].Type, m, fcontent, fset, getRelativeOrBasePath(p.homePageDir, fset, decl.Pos()), mod, pkg, impt)
				if tname == "" {
					fmt.Fprintf(os.Stderr, "Error: cannot get type from receiver %v", decl.Recv.List[0].Type)
					continue
				}
				dname = fmt.Sprintf("%v.%v", tname, dname)
				// mock type
				id := Identity{
					ModPath: mod,
					PkgPath: pkg,
					Name:    tname,
				}
				receiver = &Receiver{
					Type:      id,
					IsPointer: isPointer,
					// Name:      name,
				}
			}
			if dname == name {
				ids = append(ids, newIdentity(mod, pkg, name))
				fn := p.newFunc(mod, pkg, name)
				fn.Content = string(GetRawContent(fset, fcontent, decl, p.opts.CollectComment))
				fn.FileLine = p.exportFileLine(fset, decl)
				fn.IsMethod = decl.Recv != nil
				fn.Receiver = receiver
				// if decl.Type.Params != nil {
				// 	params := map[string]Identity{}
				// 	for _, fdec := range decl.Type.Params.List {
				// 		p.mockTypes(fdec.Type, params, fcontent, fset, fn.File, mod, pkg, impt)
				// 	}
				// 	fn.Params = params
				// }
				// if decl.Type.Results != nil {
				// 	results := map[string]Identity{}
				// 	for _, fdec := range decl.Type.Results.List {
				// 		p.mockTypes(fdec.Type, results, fcontent, fset, fn.File, mod, pkg, impt)
				// 	}
				// 	fn.Results = results
				// }
			}
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					// NOTICE: collect every types to avoid missing when searching from other refs
					// OPTIMIZE: only collect the type with the name
					var st *Type
					if spec.Name.Name == name {
						st = p.newType(mod, pkg, spec.Name.Name)
						st.Content = string(GetRawContent(fset, fcontent, spec, p.opts.CollectComment))
						st.FileLine = p.exportFileLine(fset, spec)
						st.TypeKind = getTypeKind(spec.Type)
						ids = append(ids, newIdentity(mod, pkg, name))
					}
					if inter, ok := spec.Type.(*ast.InterfaceType); st != nil && ok {
						// interface type may be a method called
						for _, m := range inter.Methods.List {
							if len(m.Names) == 0 {
								continue
							}
							mname := spec.Name.Name + "." + m.Names[0].Name
							if mname == name {
								// collect the method
								ids = append(ids, newIdentity(mod, pkg, name))
								fn := p.newFunc(mod, pkg, name)
								fn.Content = string(GetRawContent(fset, fcontent, m, p.opts.CollectComment))
								fn.FileLine = p.exportFileLine(fset, m)
								fn.IsMethod = true
								fn.IsInterfaceMethod = true
								fn.Receiver = &Receiver{
									Type:      st.Identity,
									IsPointer: true,
									// Name:      name,
								}
								// if m.Type != nil {
								// 	// collect method's func params
								// 	params := map[string]Identity{}
								// 	for _, fdec := range m.Type.(*ast.FuncType).Params.List {
								// 		p.mockTypes(fdec.Type, params, fcontent, fset, fn.File, mod, pkg, impt)
								// 	}
								// 	fn.Params = params
								// 	results := map[string]Identity{}
								// 	for _, fdec := range m.Type.(*ast.FuncType).Results.List {
								// 		p.mockTypes(fdec.Type, results, fcontent, fset, fn.File, mod, pkg, impt)
								// 	}
								// 	fn.Results = results
								// }
							}
						}
					}
				case *ast.ValueSpec:
					var lastType *Identity
					for _, n := range spec.Names {
						if n.Name == name {
							ids = append(ids, newIdentity(mod, pkg, name))
							v := p.newVar(mod, pkg, name, decl.Tok == token.CONST)
							v.Content = string(GetRawContent(fset, fcontent, spec, p.opts.CollectComment))
							v.FileLine = p.exportFileLine(fset, spec)
							if spec.Type != nil {
								var m = map[string]Identity{}
								// NOTICE: collect all types
								tname, _ := p.mockTypes(spec.Type, m, fcontent, fset, v.File, mod, pkg, impt)
								id := NewIdentity(mod, pkg, tname)
								v.Type = &id
							} else {
								v.Type = lastType
							}
						}
					}
				}
			}
		}
	}
	return
}

func (p *GoParser) getModuleFromPkg(pkg PkgPath) (name string, dir string) {
	for _, m := range p.modules {
		if strings.HasPrefix(pkg, m.name) && len(m.name) > len(name) {
			name = m.name
			dir = m.dir
		}
	}
	return
}

// path is absolute path
func (p *GoParser) getModuleFromPath(path string) (name string, dir string, rel string) {
	for _, m := range p.modules {
		if m.dir == "" {
			continue
		}
		dir := filepath.Join(p.homePageDir, m.dir)
		if strings.HasPrefix(path, dir) {
			rel, _ = filepath.Rel(dir, path)
			return m.name, m.dir, rel
		}
	}
	return "", "", ""
}

// FromABS converts an absolute path to local mod path
func (p *GoParser) pkgPathFromABS(path string) PkgPath {
	mod, _, rel := p.getModuleFromPath(path)
	if mod == "" {
		panic("not found package from " + path)
	}
	if rel != "" && rel != "." {
		return mod + "/" + rel
	} else {
		return mod
	}
}
