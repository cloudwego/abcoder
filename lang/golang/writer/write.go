/**
 * Copyright 2025 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package writer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/cloudwego/abcoder/lang/utils"
)

var _ uniast.Writer = (*Writer)(nil)
var testPkgPathRegex = regexp.MustCompile(`^(.+?) \[(.+)\]$`)

type Options struct {
	// RepoDir   string
	// OutDir    string
	CompilerPath string
}

type Writer struct {
	Options
	visited map[string]map[string]*fileNode
}

type fileNode struct {
	chunks []chunk
	impts  []uniast.Import
}

type chunk struct {
	codes string
	line  int
}

const localVersion = "v0.0.0"

func NewWriter(opts Options) *Writer {
	if opts.CompilerPath == "" {
		opts.CompilerPath = "go"
	}
	return &Writer{
		Options: opts,
		visited: make(map[string]map[string]*fileNode),
	}
}

func (w *Writer) WriteRepo(repo *uniast.Repository, outDir string) error {
	for m, mod := range repo.Modules {
		if strings.Contains(m, "@") {
			continue
		}
		if err := w.WriteModule(repo, m, outDir); err != nil {
			return fmt.Errorf("write module %s failed: %v", mod.Name, err)
		}
	}
	return nil
}

// sanitizePkgPath sanitize the package path, remove the suffix in brackets
func sanitizePkgPath(pkgPath string) string {
	matches := testPkgPathRegex.FindStringSubmatch(pkgPath)
	// matches should be 3 elements:
	// 1. The full string
	// 2. The package name
	// 3. The content inside the brackets
	if len(matches) == 3 {
		packageName := matches[1]
		testName := matches[2]
		if testName == packageName+".test" {
			return packageName
		}
	}
	return pkgPath
}
func (w *Writer) WriteModule(repo *uniast.Repository, modPath string, outDir string) error {
	mod := repo.Modules[modPath]
	if mod == nil {
		return fmt.Errorf("module %s not found", modPath)
	}
	for _, pkg := range mod.Packages {
		if err := w.appendPackage(repo, pkg); err != nil {
			return fmt.Errorf("write package %s failed: %v", pkg.PkgPath, err)
		}
	}

	outdir := filepath.Join(outDir, mod.Dir)
	for dir, pkg := range w.visited {
		// sanitize the package path
		cleanDir := sanitizePkgPath(dir)
		rel := strings.TrimPrefix(cleanDir, mod.Name)
		pkgDir := filepath.Join(outdir, rel)
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			return fmt.Errorf("mkdir %s failed: %v", pkgDir, err)
		}

		for fpath, f := range pkg {

			var sb strings.Builder
			sb.WriteString("package ")
			if p := mod.Packages[dir]; p != nil && p.IsMain {
				sb.WriteString("main")
			} else {
				sb.WriteString(filepath.Base(dir))
			}
			sb.WriteString("\n\n")

			var fimpts []uniast.Import
			if fi, ok := mod.Files[filepath.Join(mod.Dir, rel, fpath)]; ok && fi.Imports != nil {
				fimpts = fi.Imports
			}
			impts := mergeImports(fimpts, f.impts)
			if len(impts) > 0 {
				writeImport(&sb, impts)
			}

			sort.SliceStable(f.chunks, func(i, j int) bool {
				return f.chunks[i].line < f.chunks[j].line
			})
			for _, c := range f.chunks {
				sb.WriteString(c.codes)
				sb.WriteString("\n\n")
			}
			fpath = filepath.Join(pkgDir, fpath)
			if err := os.WriteFile(fpath, []byte(sb.String()), 0644); err != nil {
				return fmt.Errorf("write file %s failed: %v", fpath, err)
			}
		}
	}

	// create go mod
	var bs strings.Builder
	bs.WriteString("module ")
	bs.WriteString(mod.Name)
	bs.WriteString("\n\ngo ")
	goVersion, err := w.GetGoVersion()
	if err != nil {
		goVersion = "1.21"
	}
	bs.WriteString(goVersion)
	bs.WriteString("\n\n")

	replaces := make(map[string]string)
	if len(mod.Dependencies) > 0 {
		bs.WriteString("require (\n")
		for name, dep := range mod.Dependencies {
			bs.WriteString("\t")
			bs.WriteString(name)
			sp := strings.Split(dep, "@")
			if len(sp) == 2 {
				if sp[1] == "" {
					bs.WriteString(" ")
					bs.WriteString(localVersion)
					replaces[name] = sp[0]
				} else {
					bs.WriteString(" ")
					bs.WriteString(sp[1])
				}
			}
			bs.WriteString("\n")
		}
		bs.WriteString(")\n\n")
	}

	for name, dep := range replaces {
		bs.WriteString("replace ")
		bs.WriteString(name)
		bs.WriteString(" => ")
		bs.WriteString(dep)
		bs.WriteString("\n")
	}

	if err := os.WriteFile(filepath.Join(outdir, "go.mod"), []byte(bs.String()), 0644); err != nil {
		return fmt.Errorf("write go.mod failed: %v", err)
	}

	// go mod tidy
	cmd := exec.Command(w.Options.CompilerPath, "mod", "tidy")
	cmd.Dir = outdir
	if err := cmd.Run(); err != nil {
		log.Error("go mod tidy failed: %v", err)
	}
	return nil
}

var goVersionRegex = regexp.MustCompile(`go(\d+\.\d+(\.\d+)?)`)

func (w *Writer) GetGoVersion() (string, error) {
	cmd := exec.Command(w.Options.CompilerPath, "version")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("get go version failed: %v", err)
	}
	// extract with regexp
	matches := goVersionRegex.FindStringSubmatch(string(out))
	if len(matches) == 0 {
		return "", fmt.Errorf("get go version failed: %v", err)
	}
	return matches[1], nil
}

func (w *Writer) appendPackage(repo *uniast.Repository, pkg *uniast.Package) error {
	for _, v := range pkg.Vars {
		n := repo.GetNode(v.Identity)
		if err := w.appendNode(n, pkg.PkgPath, pkg.IsMain, v.File, v.Line, v.Content); err != nil {
			return fmt.Errorf("append chunk for var %s failed: %v", v.Name, err)
		}
	}
	for _, f := range pkg.Functions {
		if f.IsInterfaceMethod {
			// NOTICE: interface method and it has already been written in Interface Decl
			continue
		}
		n := repo.GetNode(f.Identity)
		if err := w.appendNode(n, pkg.PkgPath, pkg.IsMain, f.File, f.Line, f.Content); err != nil {
			return fmt.Errorf("append chunk for function %s failed: %v", f.Name, err)
		}
	}
	for _, t := range pkg.Types {
		n := repo.GetNode(t.Identity)
		if err := w.appendNode(n, pkg.PkgPath, pkg.IsMain, t.File, t.Line, t.Content); err != nil {
			return fmt.Errorf("append chunk for type %s failed: %v", t.Name, err)
		}
	}
	return nil
}

func (w *Writer) appendNode(node *uniast.Node, pkg string, isMain bool, file string, line int, src string) error {
	p := w.visited[pkg]
	if p == nil {
		p = make(map[string]*fileNode)
		w.visited[pkg] = p
	}
	var fpath string
	if file == "" {
		if isMain {
			fpath = "main.go"
		} else {
			fpath = "lib.go"
		}
	} else {
		fpath = filepath.Base(file)
	}
	// codes, impts, err := SplitGoImportsAndCodes(src)
	// if err != nil {
	// 	return fmt.Errorf("split go imports and codes failed: %v", err)
	// }
	fs := p[fpath]
	if fs == nil {
		fs = &fileNode{
			chunks: make([]chunk, 0, len(node.Dependencies)),
			impts:  make([]uniast.Import, 0, len(node.Dependencies)),
		}
		p[fpath] = fs
	}
	for _, v := range node.Dependencies {
		if v.PkgPath == "" || v.PkgPath == pkg {
			continue
		}
		fs.impts = append(fs.impts, uniast.Import{Path: strconv.Quote(v.PkgPath)})
	}

	// 检查是否有imports
	if cs, impts, err := w.SplitImportsAndCodes(src); err == nil {
		src = cs
		for _, v := range impts {
			fs.impts = append(fs.impts, v)
		}
	}

	fs.chunks = append(fs.chunks, chunk{
		codes: src,
		line:  line,
	})
	return nil
}

// receive a piece of golang code, parse it and splits the imports and codes
func (w Writer) SplitImportsAndCodes(src string) (codes string, imports []uniast.Import, err error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.SkipObjectResolution)
	if err != nil {
		// NOTICE: if parse failed, just return the src
		return src, nil, nil
	}
	for _, imp := range f.Imports {
		var alias string
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		imports = append(imports, uniast.Import{Path: imp.Path.Value, Alias: &alias})
	}
	start := 0
	for _, s := range f.Decls {
		if gen, ok := s.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			continue
		}
		start = fset.Position(s.Pos()).Offset
		break
	}
	return src[start:], imports, nil
}

func (w *Writer) IdToImport(id uniast.Identity) (uniast.Import, error) {
	return uniast.Import{Path: strconv.Quote(id.PkgPath)}, nil
}

func (p *Writer) PatchImports(impts []uniast.Import, file []byte) ([]byte, error) {

	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "default.go", file, parser.ImportsOnly)
	if err != nil {
		return nil, utils.WrapError(err, "fail parse file %s", file)
	}

	old := make([]uniast.Import, 0, len(f.Imports))
	for _, imp := range f.Imports {
		i := uniast.Import{
			Path: imp.Path.Value,
		}
		if imp.Name != nil {
			tmp := imp.Name.Name
			i.Alias = &tmp
		}
		old = append(old, i)
	}

	impts = mergeImports(old, impts)
	if len(impts) == len(old) {
		return file, nil
	}

	var sb strings.Builder
	writeImport(&sb, impts)
	final := sb.String()

	imptStart := fs.Position(f.Name.End()).Offset + 1
	if len(f.Imports) > 0 {
		for imptStart < len(file) && file[imptStart] != 'i' {
			imptStart++
		}
	}
	imptEnd := imptStart
	if len(f.Imports) > 1 {
		imptEnd = fs.Position(f.Imports[len(f.Imports)-1].End()).Offset
		for len(old) > 1 && imptEnd < len(file) && (file[imptEnd] != ')') {
			imptEnd++
		}
		imptEnd += 2 // for `)`
	}
	r1 := append(file[:imptStart:imptStart], final...)
	ret := append(r1, file[imptEnd:]...)
	return ret, nil
}

func (p *Writer) CreateFile(fi *uniast.File, mod *uniast.Module) ([]byte, error) {
	var sb strings.Builder
	sb.WriteString("package ")
	pkgName := filepath.Base(filepath.Dir(fi.Path))
	if fi.Package != "" {
		pkg := mod.Packages[fi.Package]
		if pkg != nil {
			if pkg.IsMain {
				pkgName = "main"
			} else {
				pkgName = filepath.Base(pkg.PkgPath)
			}
		}
	}
	if pkgName == "" {
		return nil, fmt.Errorf("package name is empty")
	}
	sb.WriteString(pkgName)
	sb.WriteString("\n\n")

	if len(fi.Imports) > 0 {
		writeImport(&sb, fi.Imports)
	}

	bs := sb.String()
	return []byte(bs), nil
}
