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
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/cloudwego/abcoder/src/uniast"
)

var _ uniast.Writer = (*Writer)(nil)

type Options struct {
	OutDir    string
	GoVersion string
}

type Writer struct {
	Options
	visited map[string]map[string]*fileNode
}

type fileNode struct {
	chunks []chunk
	impts  []string
}

type chunk struct {
	codes string
	line  int
}

func NewWriter(opts Options) *Writer {
	return &Writer{
		Options: opts,
		visited: make(map[string]map[string]*fileNode),
	}
}

func (w *Writer) WriteRepo(repo *uniast.Repository) error {
	for m, mod := range repo.Modules {
		if strings.Contains(m, "@") {
			continue
		}
		if err := w.WriteModule(repo, m); err != nil {
			return fmt.Errorf("write module %s failed: %v", mod.Name, err)
		}
	}
	return nil
}

func (w *Writer) WriteModule(repo *uniast.Repository, modPath string) error {
	mod := repo.Modules[modPath]
	if mod == nil {
		return fmt.Errorf("module %s not found", modPath)
	}
	for _, pkg := range mod.Packages {
		if err := w.appendPackage(repo, pkg); err != nil {
			return fmt.Errorf("write package %s failed: %v", pkg.PkgPath, err)
		}
	}

	outdir := filepath.Join(w.Options.OutDir, mod.Dir)
	for dir, pkg := range w.visited {
		rel := strings.TrimPrefix(dir, mod.Name)
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

			var fimpts []string
			if fi, ok := mod.Files[filepath.Join(mod.Dir, rel, fpath)]; ok && fi.Imports != nil {
				fimpts = fi.Imports
			}
			impts := w.mergeImports(fimpts, f.impts)
			if len(impts) > 0 {
				sb.WriteString("import (\n")
				for _, v := range impts {
					sb.WriteString("\t")
					sb.WriteString(v)
					sb.WriteString("\n")
				}
				sb.WriteString(")\n\n")
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

	// go mod
	var bs strings.Builder
	bs.WriteString("module ")
	bs.WriteString(mod.Name)
	bs.WriteString("\n\ngo ")
	bs.WriteString(w.Options.GoVersion)
	bs.WriteString("\n\n")
	if len(mod.Dependencies) > 0 {
		bs.WriteString("require (\n")
		for name, dep := range mod.Dependencies {
			bs.WriteString("\t")
			bs.WriteString(name)
			sp := strings.Split(dep, "@")
			if len(sp) == 2 {
				bs.WriteString(" ")
				bs.WriteString(sp[1])
			}
			bs.WriteString("\n")
		}
		bs.WriteString(")\n\n")
	}
	if err := os.WriteFile(filepath.Join(outdir, "go.mod"), []byte(bs.String()), 0644); err != nil {
		return fmt.Errorf("write go.mod failed: %v", err)
	}

	return nil
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
			impts:  make([]string, 0, len(node.Dependencies)),
		}
		p[fpath] = fs
	}
	for _, v := range node.Dependencies {
		if v.Target.PkgPath == "" || v.Target.PkgPath == pkg {
			continue
		}
		fs.impts = append(fs.impts, strconv.Quote(v.Target.PkgPath))
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
func (w Writer) SplitImportsAndCodes(src string) (codes string, imports []string, err error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.SkipObjectResolution)
	if err != nil {
		// NOTICE: if parse failed, just return the src
		return src, nil, nil
	}
	for _, imp := range f.Imports {
		var impt = imp.Path.Value
		if imp.Name != nil {
			impt = fmt.Sprintf("%s %s", imp.Name.Name, impt)
		}
		imports = append(imports, impt)
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

func (w *Writer) IdToImport(id uniast.Identity) (string, error) {
	return strconv.Quote(id.PkgPath), nil
}

// merge the imports of file and nodes, and return the merged imports
// file is in priority (because it contains alias)
func (w *Writer) mergeImports(priors []string, subs []string) (ret []string) {
	visited := make(map[string]bool, len(priors)+len(subs))
	ret = make([]string, 0, len(priors)+len(subs))
	for _, v := range priors {
		sp := strings.Split(v, " ")
		var impt = sp[0]
		if len(sp) >= 2 {
			impt = sp[1]
		}
		key, _ := strconv.Unquote(impt)
		if visited[key] {
			continue
		} else {
			visited[key] = true
			ret = append(ret, v)
		}
	}
	for _, v := range subs {
		sp := strings.Split(v, " ")
		var impt = sp[0]
		if len(sp) >= 2 {
			impt = sp[1]
		}
		key, _ := strconv.Unquote(impt)
		if visited[key] {
			continue
		} else {
			visited[key] = true
			ret = append(ret, v)
		}
	}
	return
}
