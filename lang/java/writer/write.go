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

package writer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/cloudwego/abcoder/lang/uniast"
)

var _ uniast.Writer = (*Writer)(nil)

var importLineRegex = regexp.MustCompile(`^\s*import\s+(static\s+)?[\w.*]+\s*;\s*$`)

type Options struct{}

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

func NewWriter(opts Options) *Writer {
	return &Writer{
		Options: opts,
		visited: make(map[string]map[string]*fileNode),
	}
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
	for pkgPath, pkg := range w.visited {
		// Convert Java package path to directory: com.example.core → com/example/core
		rel := strings.ReplaceAll(string(pkgPath), ".", "/")
		pkgDir := filepath.Join(outdir, "src", "main", "java", rel)
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			return fmt.Errorf("mkdir %s failed: %v", pkgDir, err)
		}

		for fpath, f := range pkg {
			var sb strings.Builder
			sb.WriteString("package ")
			sb.WriteString(string(pkgPath))
			sb.WriteString(";\n\n")

			var fimpts []uniast.Import
			// mod.Files keys are relative paths (e.g. src/main/java/org/example/Cat.java),
			// but fpath is a basename (e.g. Cat.java). Match by basename.
			for fk, fi := range mod.Files {
				if filepath.Base(fk) == fpath && fi.Imports != nil {
					fimpts = fi.Imports
					break
				}
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
			outPath := filepath.Join(pkgDir, fpath)
			if err := os.WriteFile(outPath, []byte(sb.String()), 0644); err != nil {
				return fmt.Errorf("write file %s failed: %v", outPath, err)
			}
		}
	}

	// Generate pom.xml
	if err := w.writePom(mod, outdir); err != nil {
		return fmt.Errorf("write pom.xml failed: %v", err)
	}
	return nil
}

func (w *Writer) appendPackage(repo *uniast.Repository, pkg *uniast.Package) error {
	for _, v := range pkg.Vars {
		n := repo.GetNode(v.Identity)
		if err := w.appendNode(n, pkg.PkgPath, v.File, v.Line, v.Content); err != nil {
			return fmt.Errorf("append chunk for var %s failed: %v", v.Name, err)
		}
	}
	for _, f := range pkg.Functions {
		if f.IsMethod || f.IsInterfaceMethod {
			// Java methods are inside Type.Content, skip standalone method entries
			continue
		}
		n := repo.GetNode(f.Identity)
		if err := w.appendNode(n, pkg.PkgPath, f.File, f.Line, f.Content); err != nil {
			return fmt.Errorf("append chunk for function %s failed: %v", f.Name, err)
		}
	}
	for _, t := range pkg.Types {
		n := repo.GetNode(t.Identity)
		if err := w.appendNode(n, pkg.PkgPath, t.File, t.Line, t.Content); err != nil {
			return fmt.Errorf("append chunk for type %s failed: %v", t.Name, err)
		}
	}
	return nil
}

func (w *Writer) appendNode(node *uniast.Node, pkg uniast.PkgPath, file string, line int, src string) error {
	p := w.visited[string(pkg)]
	if p == nil {
		p = make(map[string]*fileNode)
		w.visited[string(pkg)] = p
	}
	var fpath string
	if file == "" {
		fpath = "Lib.java"
	} else {
		fpath = filepath.Base(file)
	}
	fs := p[fpath]
	if fs == nil {
		depLen := 0
		if node != nil {
			depLen = len(node.Dependencies)
		}
		fs = &fileNode{
			chunks: make([]chunk, 0, depLen),
			impts:  make([]uniast.Import, 0, depLen),
		}
		p[fpath] = fs
	}
	if node != nil {
		for _, v := range node.Dependencies {
			if v.PkgPath == "" || v.PkgPath == pkg {
				continue
			}
			imp, _ := w.IdToImport(v.Identity)
			// Skip java.lang imports (auto-imported)
			if strings.HasPrefix(imp.Path, "java.lang.") {
				continue
			}
			fs.impts = append(fs.impts, imp)
		}
	}

	// Check for embedded imports in source
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

func (w *Writer) SplitImportsAndCodes(src string) (codes string, imports []uniast.Import, err error) {
	lines := strings.Split(src, "\n")
	lastImportIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "package ") {
			lastImportIdx = i
			continue
		}
		if importLineRegex.MatchString(line) {
			imp := parseImportLine(trimmed)
			if imp.Path != "" {
				imports = append(imports, imp)
			}
			lastImportIdx = i
		}
	}
	if lastImportIdx < 0 {
		return src, nil, nil
	}
	// Return everything after the last import/package line
	remaining := strings.Join(lines[lastImportIdx+1:], "\n")
	return strings.TrimLeft(remaining, "\n"), imports, nil
}

func parseImportLine(line string) uniast.Import {
	// "import static com.example.Foo;" or "import com.example.Foo;"
	s := strings.TrimPrefix(line, "import ")
	s = strings.TrimSuffix(s, ";")
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "static ") {
		path := strings.TrimPrefix(s, "static ")
		path = strings.TrimSpace(path)
		alias := "static"
		return uniast.Import{Path: path, Alias: &alias}
	}
	return uniast.Import{Path: s}
}

func (w *Writer) IdToImport(id uniast.Identity) (uniast.Import, error) {
	return uniast.Import{Path: string(id.PkgPath) + "." + id.Name}, nil
}

func (w *Writer) PatchImports(impts []uniast.Import, file []byte) ([]byte, error) {
	lines := strings.Split(string(file), "\n")

	// Find package line end and import block boundaries
	packageEnd := -1
	importStart := -1
	importEnd := -1
	var oldImports []uniast.Import

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "package ") {
			packageEnd = i
			continue
		}
		if importLineRegex.MatchString(line) {
			if importStart < 0 {
				importStart = i
			}
			importEnd = i
			oldImports = append(oldImports, parseImportLine(trimmed))
		}
	}

	merged := mergeImports(oldImports, impts)
	if len(merged) == len(oldImports) {
		return file, nil
	}

	// Build import lines without trailing blank line
	var sb strings.Builder
	for _, imp := range merged {
		writeSingleImport(&sb, imp)
	}
	newImportBlock := sb.String()

	var result strings.Builder
	if importStart >= 0 {
		// Replace existing import block
		for i := 0; i < importStart; i++ {
			result.WriteString(lines[i])
			result.WriteString("\n")
		}
		result.WriteString(newImportBlock)
		for i := importEnd + 1; i < len(lines); i++ {
			result.WriteString(lines[i])
			if i < len(lines)-1 {
				result.WriteString("\n")
			}
		}
	} else {
		// No existing imports; insert after package line
		insertAfter := packageEnd
		for i := 0; i <= insertAfter; i++ {
			result.WriteString(lines[i])
			result.WriteString("\n")
		}
		result.WriteString("\n")
		result.WriteString(newImportBlock)
		for i := insertAfter + 1; i < len(lines); i++ {
			result.WriteString(lines[i])
			if i < len(lines)-1 {
				result.WriteString("\n")
			}
		}
	}
	return []byte(result.String()), nil
}

func (w *Writer) CreateFile(fi *uniast.File, mod *uniast.Module) ([]byte, error) {
	var sb strings.Builder
	pkgName := string(fi.Package)
	if pkgName == "" {
		return nil, fmt.Errorf("package name is empty")
	}
	sb.WriteString("package ")
	sb.WriteString(pkgName)
	sb.WriteString(";\n\n")

	if len(fi.Imports) > 0 {
		writeImport(&sb, fi.Imports)
	}

	return []byte(sb.String()), nil
}

func (w *Writer) writePom(mod *uniast.Module, outDir string) error {
	// Parse mod.Name as groupId:artifactId:version
	parts := strings.SplitN(mod.Name, ":", 3)
	groupId := "com.example"
	artifactId := "project"
	version := "1.0.0"
	if len(parts) >= 1 && parts[0] != "" {
		groupId = parts[0]
	}
	if len(parts) >= 2 && parts[1] != "" {
		artifactId = parts[1]
	}
	if len(parts) >= 3 && parts[2] != "" {
		version = parts[2]
	}

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString("\n")
	sb.WriteString(`<project xmlns="http://maven.apache.org/POM/4.0.0"`)
	sb.WriteString("\n")
	sb.WriteString(`         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`)
	sb.WriteString("\n")
	sb.WriteString(`         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">`)
	sb.WriteString("\n")
	sb.WriteString("    <modelVersion>4.0.0</modelVersion>\n\n")
	sb.WriteString("    <groupId>")
	sb.WriteString(groupId)
	sb.WriteString("</groupId>\n")
	sb.WriteString("    <artifactId>")
	sb.WriteString(artifactId)
	sb.WriteString("</artifactId>\n")
	sb.WriteString("    <version>")
	sb.WriteString(version)
	sb.WriteString("</version>\n")

	if len(mod.Dependencies) > 0 {
		sb.WriteString("\n    <dependencies>\n")
		for name, dep := range mod.Dependencies {
			depParts := strings.SplitN(name, ":", 2)
			depGroupId := depParts[0]
			depArtifactId := ""
			if len(depParts) >= 2 {
				depArtifactId = depParts[1]
			}
			depVersion := dep
			if idx := strings.Index(dep, "@"); idx >= 0 {
				depVersion = dep[idx+1:]
			}
			sb.WriteString("        <dependency>\n")
			sb.WriteString("            <groupId>")
			sb.WriteString(depGroupId)
			sb.WriteString("</groupId>\n")
			sb.WriteString("            <artifactId>")
			sb.WriteString(depArtifactId)
			sb.WriteString("</artifactId>\n")
			if depVersion != "" {
				sb.WriteString("            <version>")
				sb.WriteString(depVersion)
				sb.WriteString("</version>\n")
			}
			sb.WriteString("        </dependency>\n")
		}
		sb.WriteString("    </dependencies>\n")
	}

	sb.WriteString("</project>\n")

	return os.WriteFile(filepath.Join(outDir, "pom.xml"), []byte(sb.String()), 0644)
}
