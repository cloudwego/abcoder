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

	"github.com/cloudwego/abcoder/lang/golang/writer"
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/cloudwego/abcoder/lang/utils"
)

// PatchModule patches the ast Nodes onto module files

type Patch struct {
	Id        uniast.Identity
	Codes     string
	File      string
	Type      uniast.NodeType
	AddedDeps []uniast.Identity
}

type PatchNode struct {
	uniast.Identity
	uniast.FileLine
	Codes string
	File  *uniast.File
}

type Patcher struct {
	Options
	repo    *uniast.Repository
	patches Patches
}

type Patches map[string][]PatchNode

func (p *Patcher) GetPatchNodes() Patches {
	return p.patches
}

func (p *Patcher) SetPatchNodes(ps Patches) {
	p.patches = ps
}

type Options struct {
	RepoDir        string
	OutDir         string
	DefaultLanuage uniast.Language
}

func NewPatcher(repo *uniast.Repository, opts Options) *Patcher {
	return &Patcher{
		Options: opts,
		repo:    repo,
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
			if r.Identity == dep {
				continue next_dep
			}
		}
		node.Dependencies = append(node.Dependencies, uniast.Relation{
			Identity: dep,
			Kind:     uniast.DEPENDENCY,
		})
	}

	mod := p.repo.GetModule(patch.Id.ModPath)
	if mod == nil {
		mod = uniast.NewModule(patch.Id.ModPath, "", p.DefaultLanuage)
		p.repo.SetModule(patch.Id.ModPath, mod)
	}

	f := mod.GetFile(patch.File)
	if f == nil {
		f = uniast.NewFile(patch.File)
		mod.SetFile(patch.File, f)
	}

	fl := node.FileLine()
	if fl.File != patch.File {
		node.SetFileLine(uniast.FileLine{
			File: patch.File,
			Line: 0,
		})
		fl = node.FileLine()
	}

	w := p.getLangWriter(mod.Language)
	if w == nil {
		return fmt.Errorf("unsupported language %s writer", mod.Language)
	}

	for _, dep := range patch.AddedDeps {
		impt, err := w.IdToImport(dep)
		if err != nil {
			return fmt.Errorf("convert identity %s to import failed: %v", dep.Full(), err)
		}
		f.Imports = uniast.InserImport(f.Imports, impt)
	}
	n := PatchNode{
		Identity: patch.Id,
		FileLine: fl,
		Codes:    patch.Codes,
		File:     f,
	}
	if err := p.patch(n); err != nil {
		return fmt.Errorf("patch file %s failed: %v", f.Path, err)
	}
	return nil
}

func (p *Patcher) patch(n PatchNode) error {
	if p.patches == nil {
		p.patches = make(map[string][]PatchNode)
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
		if len(ns) == 0 {
			continue
		}
		mod := p.repo.GetModule(ns[0].Identity.ModPath)
		writer := p.getLangWriter(mod.Language)
		if writer == nil {
			return fmt.Errorf("unsupported language %s writer", mod.Language)
		}

		data, err := os.ReadFile(filepath.Join(p.RepoDir, fpath))
		if err != nil {
			fi := mod.GetFile(fpath)
			data, err = writer.CreateFile(fi, mod)
			if err != nil {
				return fmt.Errorf("create file %s failed: %v", fpath, err)
			}
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
			data, err := writer.PatchImports(n.File.Imports, data)
			if err != nil {
				return fmt.Errorf("patch imports failed: %v", err)
			}
			if err := utils.MustWriteFile(filepath.Join(p.OutDir, fpath), data); err != nil {
				return fmt.Errorf("write file %s failed: %v", fpath, err)
			}
		}
	}

	// write origins
	for _, mod := range p.repo.InternalModules() {
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

func (p *Patcher) getLangWriter(lang uniast.Language) uniast.Writer {
	if lang == "" || lang == uniast.Unknown {
		lang = p.DefaultLanuage
	}
	switch lang {
	case uniast.Golang:
		return writer.NewWriter(writer.Options{})
	default:
		return nil
	}
}
