// Copyright 2025 ByteDance Inc.
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

package patch

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/cloudwego/abcoder/src/lang/golang/writer"
	"github.com/cloudwego/abcoder/src/lang/utils"
	"github.com/cloudwego/abcoder/src/uniast"
)

// PatchModule patches the ast Nodes onto module files

type Patch struct {
	Id        uniast.Identity
	Codes     string
	File      string
	Type      uniast.NodeType
	AddedDeps []uniast.Identity
}

type patchNode struct {
	uniast.Identity
	uniast.FileLine
	Codes string
	File  *uniast.File
}

type Patcher struct {
	Options
	repo    *uniast.Repository
	patches map[string][]patchNode
	ps      map[uniast.Language]uniast.Writer
}

type Options struct {
	RepoDir string
	OutDir  string
}

func NewPatcher(opts Options) *Patcher {
	return &Patcher{
		Options: opts,
	}
}

func (p *Patcher) SetRepo(repo *uniast.Repository) {
	p.repo = repo
}

func (p *Patcher) Patch(patch Patch) error {
	// find package
	node := p.repo.GetNode(patch.Id)
	if node == nil {
		node = p.repo.SetNode(patch.Id, patch.Type)
	}
next_dep:
	for _, dep := range patch.AddedDeps {
		for _, r := range node.Dependencies {
			if r.Target == dep {
				continue next_dep
			}
		}
		node.Dependencies = append(node.Dependencies, uniast.Relation{
			Target: dep,
			Kind:   uniast.DEPENDENCY,
		})
	}
	f := p.repo.GetFileById(patch.Id)
	if f == nil {
		mod := p.repo.GetModule(patch.Id.ModPath)
		f = uniast.NewFile(patch.File)
		mod.SetFile(patch.File, f)
		node.SetFile(patch.File)
	}

	for _, dep := range patch.AddedDeps {
		f.Imports = uniast.InserImport(f.Imports, uniast.Import{
			Alias: nil,
			Path:  dep.PkgPath,
		})
	}
	n := patchNode{
		FileLine: node.FileLine(),
		Codes:    patch.Codes,
		File:     f,
	}
	if err := p.patch(n); err != nil {
		return fmt.Errorf("patch file %s failed: %v", f.Path, err)
	}
	return nil
}

func (p *Patcher) patch(n patchNode) error {
	if p.patches == nil {
		p.patches = make(map[string][]patchNode)
	}
	if n.StartOffset < 1 {
		n.StartOffset = math.MaxInt
	}
	p.patches[n.FileLine.File] = append(p.patches[n.FileLine.File], n)
	return nil
}

func (p *Patcher) Flush() error {
	// write pathes
	for fpath, ns := range p.patches {

		path := filepath.Join(p.RepoDir, fpath)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read file %s failed: %v", path, err)
		}

		// sort by offset
		sort.SliceStable(ns, func(i, j int) bool {
			return ns[i].StartOffset < ns[j].StartOffset
		})

		var offset int
		for _, n := range ns {
			if n.StartOffset >= len(data) {
				data = append(append(data, '\n'), []byte(n.Codes)...)
				continue
			}
			tmp := append(data[:offset+n.StartOffset:offset+n.StartOffset], []byte(n.Codes)...)
			data = append(tmp, data[offset+n.EndOffset:]...)
			offset += (len(n.Codes) - (n.EndOffset - n.StartOffset))
		}

		if err := utils.MustWriteFile(filepath.Join(p.OutDir, fpath), data); err != nil {
			return fmt.Errorf("write file %s failed: %v", fpath, err)
		}

		// patch imports
		if len(ns) > 0 {
			n := ns[0]
			mod := p.repo.GetModule(n.Identity.ModPath)
			if mod == nil {
				return fmt.Errorf("module %s not found", n.Identity.ModPath)
			}
			ip := p.getLangPatcher(mod.Language)
			if ip == nil {
				return fmt.Errorf("unsupported language %s", mod.Language)
			}
			data, err := ip.PatchImports(&uniast.File{
				Path:    fpath,
				Imports: n.File.Imports,
			})
			if err != nil {
				return fmt.Errorf("patch imports failed: %v", err)
			}
			if err := utils.MustWriteFile(filepath.Join(p.OutDir, fpath), data); err != nil {
				return fmt.Errorf("write file %s failed: %v", fpath, err)
			}
		}
	}

	// write origins
	for _, mod := range p.repo.Modules {
		for _, f := range mod.Files {
			if p.patches[f.Path] != nil {
				continue
			}
			fpath := filepath.Join(p.RepoDir, f.Path)
			bs, err := os.ReadFile(fpath)
			if err != nil {
				return fmt.Errorf("read file %s failed: %v", fpath, err)
			}
			fpath = filepath.Join(p.OutDir, f.Path)
			if err := utils.MustWriteFile(fpath, bs); err != nil {
				return fmt.Errorf("write file %s failed: %v", fpath, err)
			}
		}
	}
	return nil
}

func (p *Patcher) getLangPatcher(lang uniast.Language) uniast.Writer {
	switch lang {
	case uniast.Golang:
		return writer.NewWriter(writer.Options{
			RepoDir: p.RepoDir,
		})
	default:
		return nil
	}
}
