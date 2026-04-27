package parser

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"path/filepath"

	. "github.com/cloudwego/abcoder/lang/uniast"
)

func (p *GoParser) GetModuleFromPathForCLI(path string) (string, string, string) {
	return p.getModuleFromPath(path)
}

func (p *GoParser) PkgPathFromABSForCLI(path string) PkgPath {
	return p.pkgPathFromABS(path)
}

func (p *GoParser) ParsePackageOnlyForCLI(pkgPath PkgPath) error {
	return p.parsePackage(pkgPath)
}

func (p *GoParser) GetRepoForCLI() *Repository {
	repo := p.getRepo()
	return &repo
}

func (p *GoParser) ParseSymbolInFile(absFilePath, symbolName string) (Identity, error) {
	mod, _, _ := p.getModuleFromPath(absFilePath)
	if mod == "" {
		return Identity{}, fmt.Errorf("file not in repo modules: %s", absFilePath)
	}
	pkgPath := p.pkgPathFromABS(filepath.Dir(absFilePath))
	if err := p.parsePackage(pkgPath); err != nil {
		return Identity{}, err
	}

	m := p.repo.Modules[mod]
	if m == nil {
		return Identity{}, fmt.Errorf("module not found: %s", mod)
	}
	fset := token.NewFileSet()
	fcontent := p.getFileBytes(absFilePath)
	file, err := goparser.ParseFile(fset, absFilePath, fcontent, goparser.SkipObjectResolution)
	if err != nil {
		return Identity{}, err
	}
	impts, err := p.parseImports(fset, fcontent, m, file.Imports)
	if err != nil {
		return Identity{}, err
	}
	ids, err := p.searchOnFile(file, fset, fcontent, mod, string(pkgPath), impts, symbolName, "")
	if err != nil {
		return Identity{}, err
	}
	if len(ids) == 0 {
		return Identity{}, fmt.Errorf("symbol '%s' not found in file '%s'", symbolName, absFilePath)
	}
	for _, id := range ids {
		node, _ := p.getNode(id)
		switch n := node.(type) {
		case *Function:
			if n.File == relOrSame(p.homePageDir, absFilePath) {
				return id, nil
			}
		case *Type:
			if n.FileLine.File == relOrSame(p.homePageDir, absFilePath) {
				return id, nil
			}
		case *Var:
			if n.FileLine.File == relOrSame(p.homePageDir, absFilePath) {
				return id, nil
			}
		}
	}
	return ids[0], nil
}

func relOrSame(root, abs string) string {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return abs
	}
	return rel
}

func _noopAST(_ ast.Node) {}
