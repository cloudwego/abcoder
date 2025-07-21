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

package tool

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	abutil "github.com/cloudwego/abcoder/internal/utils"
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/cloudwego/abcoder/llm/log"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/fsnotify/fsnotify"
)

const (
	ToolListRepos           = "list_repos"
	DescListRepos           = "list all repositories"
	ToolGetRepoStructure    = "get_repo_structure"
	DescGetRepoStructure    = "get the repository structure, including package list and file list"
	ToolGetPackageStructure = "get_package_structure"
	DescGetPackageStructure = "get the package (NameSpace) structure, including file list and node-id list"
	ToolGetFileStructure    = "get_file_structure"
	DescGetFileStructure    = "get the file structure, including node (id,signature,type) list"
	ToolGetASTNode          = "get_ast_node"
	DescGetASTNode          = "precisely get the codes, type, location of a specific ast node, as well as the identities of related (Dependend/Reference/Implement/Inherit/Group) nodes"
	// ToolWriteASTNode        = "write_ast_node"
)

var (
	SchemaListRepos           = GetJSONSchema(ListReposReq{})
	SchemaGetRepoStructure    = GetJSONSchema(GetRepoStructReq{})
	SchemaGetPackageStructure = GetJSONSchema(GetPackageStructReq{})
	SchemaGetFileStructure    = GetJSONSchema(GetFileStructReq{})
	SchemaGetASTNode          = GetJSONSchema(GetASTNodeReq{})
)

type ASTReadToolsOptions struct {
	// PatchOptions patch.Options
	RepoASTsDir string
}

type ASTReadTools struct {
	opts  ASTReadToolsOptions
	repos sync.Map
	tools map[string]tool.InvokableTool
}

func NewASTReadTools(opts ASTReadToolsOptions) *ASTReadTools {
	ret := &ASTReadTools{
		opts: opts,
		// patcher: patch.NewPatcher(repo, opts.PatchOptions),
		tools: map[string]tool.InvokableTool{},
	}

	// read all *.json files in opts.RepoASTsDir
	files, err := filepath.Glob(filepath.Join(opts.RepoASTsDir, "*.json"))
	if err != nil {
		panic(err)
	}
	for _, f := range files {
		// parse json
		if repo, err := uniast.LoadRepo(f); err != nil {
			panic("Load Uniast JSON file failed: " + err.Error())
		} else {
			ret.repos.Store(repo.Name, repo)
		}
	}

	// add a file watch on the RepoASTsDir
	abutil.WatchDir(opts.RepoASTsDir, func(op fsnotify.Op, file string) {
		if !strings.HasSuffix(file, ".json") {
			return
		}
		if op&fsnotify.Write != 0 || op&fsnotify.Create != 0 {
			if repo, err := uniast.LoadRepo(file); err != nil {
				log.Error("Load Uniast JSON file failed: %v", err)
			} else {
				ret.repos.Store(repo.Name, repo)
			}
		} else if op&fsnotify.Remove != 0 {
			ret.repos.Delete(filepath.Base(file))
		}
	})

	tt, err := utils.InferTool(string(ToolListRepos),
		DescListRepos,
		ret.ListRepos, utils.WithMarshalOutput(func(ctx context.Context, output interface{}) (string, error) {
			return abutil.MarshalJSONIndent(output)
		}))
	if err != nil {
		panic(err)
	}
	ret.tools[ToolListRepos] = tt

	tt, err = utils.InferTool(ToolGetRepoStructure,
		DescGetRepoStructure,
		ret.GetRepoStructure, utils.WithMarshalOutput(func(ctx context.Context, output interface{}) (string, error) {
			return abutil.MarshalJSONIndent(output)
		}))
	if err != nil {
		panic(err)
	}
	ret.tools[ToolGetRepoStructure] = tt

	tt, err = utils.InferTool(string(ToolGetPackageStructure),
		string(DescGetPackageStructure),
		ret.GetPackageStructure, utils.WithMarshalOutput(func(ctx context.Context, output interface{}) (string, error) {
			return abutil.MarshalJSONIndent(output)
		}))
	if err != nil {
		panic(err)
	}
	ret.tools[ToolGetPackageStructure] = tt

	tt, err = utils.InferTool(string(ToolGetFileStructure),
		string(DescGetFileStructure),
		ret.GetFileStructure, utils.WithMarshalOutput(func(ctx context.Context, output interface{}) (string, error) {
			return abutil.MarshalJSONIndent(output)
		}))
	if err != nil {
		panic(err)
	}
	ret.tools[ToolGetFileStructure] = tt

	tt, err = utils.InferTool(ToolGetASTNode,
		string(DescGetASTNode),
		ret.GetASTNode, utils.WithMarshalOutput(func(ctx context.Context, output interface{}) (string, error) {
			return abutil.MarshalJSONIndent(output)
		}))
	if err != nil {
		panic(err)
	}
	ret.tools[ToolGetASTNode] = tt
	return ret
}

func (t ASTReadTools) GetTools() []Tool {
	ret := make([]Tool, 0, len(t.tools))
	for _, tt := range t.tools {
		ret = append(ret, tt)
	}
	return ret
}

func (t ASTReadTools) GetTool(name string) Tool {
	return t.tools[name]
}

type ListReposReq struct {
}

type ListReposResp struct {
	RepoNames []string `json:"repo_names" jsonschema:"description=the names of the repositories"`
}

func (t ASTReadTools) ListRepos(ctx context.Context, req ListReposReq) (*ListReposResp, error) {
	ret := ListReposResp{}
	t.repos.Range(func(key, value interface{}) bool {
		ret.RepoNames = append(ret.RepoNames, key.(string))
		return true
	})
	return &ret, nil
}

type GetRepoStructReq struct {
	RepoName string `json:"repo_name" jsonschema:"description=the name of the repository"`
}

type GetRepoStructResp struct {
	Modules []ModuleStruct `json:"modules" jsonschema:"description=the module structure of the repository"`
}

type ModuleStruct struct {
	uniast.ModPath `json:"mod_path" jsonschema:"description=the mod path of the module"`
	Packages       []PackageStruct `json:"packages,omitempty" jsonschema:"description=the package structures of the module"`
}

type PackageStruct struct {
	uniast.PkgPath `json:"pkg_path" jsonschema:"description=the path of the package"`
	Files          []FileStruct `json:"files,omitempty" jsonschema:"description=the file structures of the package"`
}

type FileStruct struct {
	FilePath string          `json:"file_path" jsonschema:"description=the path of the file"`
	Imports  []uniast.Import `json:"imports,omitempty" jsonschema:"description=the imports of the file"`
	Nodes    []NodeStruct    `json:"nodes,omitempty" jsonschema:"description=the node structs of the file"`
}

type NodeStruct struct {
	Name         string   `json:"name" jsonschema:"description=the name of the node"`
	Type         string   `json:"type,omitempty" jsonschema:"description=the type of the node"`
	Signature    string   `json:"signature,omitempty" jsonschema:"description=the func signature of the node"`
	File         string   `json:"file,omitempty" jsonschema:"description=the file path of the node"`
	Line         int      `json:"line,omitempty" jsonschema:"description=the line of the node"`
	Codes        string   `json:"codes,omitempty" jsonschema:"description=the codes of the node"`
	Dependencies []NodeID `json:"dependencies,omitempty" jsonschema:"description=the dependencies of the node"`
	References   []NodeID `json:"references,omitempty" jsonschema:"description=the references of the node"`
	Implements   []NodeID `json:"implements,omitempty" jsonschema:"description=the implements of the node"`
	Groups       []NodeID `json:"groups,omitempty" jsonschema:"description=the groups of the node"`
	Inherits     []NodeID `json:"inherits,omitempty" jsonschema:"description=the inherits of the node"`
}

type NodeID struct {
	ModPath uniast.ModPath `json:"mod_path" jsonschema:"description=the mod path of the node"`
	PkgPath uniast.PkgPath `json:"pkg_path" jsonschema:"description=the package path of the node"`
	Name    string         `json:"name" jsonschema:"description=the name of the node"`
}

func NewNodeID(id uniast.Identity) NodeID {
	return NodeID{
		ModPath: id.ModPath,
		PkgPath: id.PkgPath,
		Name:    id.Name,
	}
}

func (t *ASTReadTools) getRepoAST(repoName string) (*uniast.Repository, error) {
	repo, ok := t.repos.Load(repoName)
	if !ok {
		candis := []string{}
		t.repos.Range(func(key, value interface{}) bool {
			if strings.Contains(key.(string), repoName) {
				candis = append(candis, key.(string))
			}
			return true
		})
		if len(candis) == 1 {
			repo, ok = t.repos.Load(candis[0])
			if !ok {
				return nil, fmt.Errorf("repo '%s' not found", candis[0])
			}
			return repo.(*uniast.Repository), nil
		} else if len(candis) > 1 {
			return nil, fmt.Errorf("repo '%s' is ambiguous, maybe you want one of %v", repoName, candis)
		} else {
			return nil, fmt.Errorf("repo '%s' not found", repoName)
		}
	}
	return repo.(*uniast.Repository), nil
}

// GetRepoStructure list the packages and file-paths
func (t *ASTReadTools) GetRepoStructure(_ context.Context, req GetRepoStructReq) (*GetRepoStructResp, error) {
	log.Debug("get repo structure, req: %v", abutil.MarshalJSONIndentNoError(req))
	repo, err := t.getRepoAST(req.RepoName)
	if err != nil {
		return nil, err
	}

	resp := new(GetRepoStructResp)
	for _, mod := range repo.Modules {
		if mod.IsExternal() {
			continue
		}
		mm := ModuleStruct{
			ModPath: mod.Name,
		}
		for p := range mod.Packages {
			pp := PackageStruct{
				PkgPath: p,
			}
			files := mod.GetPkgFiles(p)
			for _, f := range files {
				pp.Files = append(pp.Files, FileStruct{
					FilePath: f.Path,
				})
			}
			mm.Packages = append(mm.Packages, pp)
		}
		resp.Modules = append(resp.Modules, mm)
	}
	log.Debug("get repo structure, resp: %v", abutil.MarshalJSONIndentNoError(resp))
	return resp, nil
}

type GetPackageStructReq struct {
	RepoName string         `json:"repo_name" jsonschema:"description=the name of the repository"`
	ModPath  uniast.ModPath `json:"mod_path" jsonschema:"description=the module path"`
	PkgPath  uniast.PkgPath `json:"package_path" jsonschema:"description=the package path"`
}

type GetPackageStructResp struct {
	Files []FileStruct `json:"files" jsonschema:"description=the file structures"`
}

func (t *ASTReadTools) getPkgFiles(ctx context.Context, pkg *uniast.Package, repo string) []FileStruct {
	files := make(map[string]bool, 8)
	for _, f := range pkg.Functions {
		files[f.File] = true
	}
	for _, f := range pkg.Types {
		files[f.File] = true
	}
	for _, f := range pkg.Vars {
		files[f.File] = true
	}
	ret := make([]FileStruct, 0, len(files))
	for f := range files {
		resp, err := t.getFileStructure(ctx, GetFileStructReq{
			RepoName: repo,
			FilePath: f,
		}, false)
		if err != nil {
			continue
		}
		ret = append(ret, resp.FileStruct)
	}
	return ret
}

// GetPackageStruct get package structure
func (t *ASTReadTools) GetPackageStructure(ctx context.Context, req GetPackageStructReq) (*GetPackageStructResp, error) {
	log.Debug("get package structure, req: %v", abutil.MarshalJSONIndentNoError(req))
	repo, err := t.getRepoAST(req.RepoName)
	if err != nil {
		return nil, err
	}

	resp := new(GetPackageStructResp)
	if req.ModPath == "" {
		for _, mod := range repo.Modules {
			if pkg, ok := mod.Packages[req.PkgPath]; ok {
				fs := t.getPkgFiles(ctx, pkg, repo.Name)
				resp.Files = append(resp.Files, fs...)
			}
		}
	} else {
		mod := repo.GetModule(req.ModPath)
		if mod != nil {
			if pkg, ok := mod.Packages[req.PkgPath]; ok {
				fs := t.getPkgFiles(ctx, pkg, repo.Name)
				resp.Files = append(resp.Files, fs...)
			}
		}
	}
	log.Debug("get repo structure, resp: %v", abutil.MarshalJSONIndentNoError(resp))
	return resp, nil
}

type GetFileStructReq struct {
	RepoName string `json:"repo_name" jsonschema:"description=the name of the repository"`
	FilePath string `json:"file_paths" jsonschema:"description=the file paths"`
}

type GetFileStructResp struct {
	FileStruct
}

// GetFileStruct get node list, each node only includes ID\Type\Signature
func (t *ASTReadTools) GetFileStructure(_ context.Context, req GetFileStructReq) (*GetFileStructResp, error) {
	log.Debug("get file structure, req: %v", abutil.MarshalJSONIndentNoError(req))
	resp, err := t.getFileStructure(context.Background(), req, true)
	log.Debug("get repo structure, resp: %v", abutil.MarshalJSONIndentNoError(resp))
	return resp, err
}

func (t *ASTReadTools) getFileStructure(_ context.Context, req GetFileStructReq, needNodeDetail bool) (*GetFileStructResp, error) {
	repo, err := t.getRepoAST(req.RepoName)
	if err != nil {
		return nil, err
	}

	resp := new(GetFileStructResp)
	file, _ := repo.GetFile(req.FilePath)
	if file == nil {
		return nil, fmt.Errorf("file '%s' not found", req.FilePath)
	}
	nodes := repo.GetFileNodes(req.FilePath)
	ff := FileStruct{
		FilePath: req.FilePath,
	}
	if needNodeDetail {
		ff.Imports = file.Imports
	}
	for _, n := range nodes {
		nn := NodeStruct{
			Name: n.Identity.Name,
		}
		if needNodeDetail {
			nn.Type = n.Type.String()
			nn.Signature = n.Signature()
			nn.Line = n.FileLine().Line
		}
		ff.Nodes = append(ff.Nodes,
			nn,
		)
	}
	resp.FileStruct = ff
	return resp, nil
}

type GetASTNodeReq struct {
	RepoName string            `json:"repo_name" jsonschema:"description=the name of the repository"`
	NodeIDs  []uniast.Identity `json:"node_ids" jsonschema:"description=the identities of the ast node"`
}

type GetASTNodeResp struct {
	Nodes []NodeStruct `json:"nodes" jsonschema:"description=the ast nodes"`
}

func (t *ASTReadTools) GetASTNode(_ context.Context, params GetASTNodeReq) (*GetASTNodeResp, error) {
	log.Debug("get ast node, req: %v", abutil.MarshalJSONIndentNoError(params))

	repo, err := t.getRepoAST(params.RepoName)
	if err != nil {
		return nil, err
	}

	resp := new(GetASTNodeResp)
	for _, id := range params.NodeIDs {
		log.Debug("query ast node %v", id.Full())
		node := repo.GetNode(id)
		if node == nil {
			continue
		}
		var desp []NodeID
		for _, dep := range node.Dependencies {
			desp = append(desp, NewNodeID(dep.Identity))
		}
		var refs []NodeID
		for _, ref := range node.References {
			refs = append(refs, NewNodeID(ref.Identity))
		}
		var imps []NodeID
		for _, imp := range node.Implements {
			imps = append(imps, NewNodeID(imp.Identity))
		}
		var inhs []NodeID
		for _, inh := range node.Inherits {
			inhs = append(inhs, NewNodeID(inh.Identity))
		}
		var grps []NodeID
		for _, grp := range node.Groups {
			grps = append(grps, NewNodeID(grp.Identity))
		}
		resp.Nodes = append(resp.Nodes, NodeStruct{
			Name:         node.Identity.Name,
			Type:         node.Type.String(),
			Codes:        node.Content(),
			File:         node.FileLine().File,
			Line:         node.FileLine().Line,
			Dependencies: desp,
			References:   refs,
			Implements:   imps,
			Inherits:     inhs,
			Groups:       grps,
		})
	}
	log.Debug("get repo structure, resp: %v", abutil.MarshalJSONIndentNoError(resp))
	return resp, nil
}
