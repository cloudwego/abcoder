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
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
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
	DescListRepos           = "[DISCOVERY] Step 1/4: List available repositories. Always the first step in ABCoder workflow. You MUST call `tree_repo` later."
	ToolTreeRepo            = "tree_repo"
	DescTreeRepo            = "[STRUCTURE] Step 2/4: Get available file_paths of a repo. Input: repo_name from `list_repos` output. Output: available file_paths. You MUST call `get_file_structure` later."
	ToolGetFileStructure    = "get_file_structure"
	DescGetFileStructure    = "[STRUCTURE] Step 3/4: Get available symbol names of a file. Input: repo_name, file_path from `tree_repo` output. Output: symbol names with signatures. You MUST call `get_file_symbol` later."
	ToolGetFileSymbol       = "get_file_symbol"
	DescGetFileSymbol       = "[ANALYSIS] Step 4/4: Get symbol's code, dependencies and references; use refer/depend's file_path and name as next `get_file_symbol` input. Input: repo_name, file_path, name. Output: codes, dependencies, references. You MUST call `get_file_symbol` with refers/depends file_path and name to check its code, call-chain or data-flow detail."

	ToolGetRepoStructure    = "get_repo_structure"
	DescGetRepoStructure    = "[STRUCTURE] level2/4: Get repository structure. Input: repo_name from list_repos output. Output: modules with packages and files."
	ToolGetPackageStructure = "get_package_structure"
	DescGetPackageStructure = "[STRUCTURE] level3/4: Get package structure with node_ids. Input: repo_name, mod_path, pkg_path from get_repo_structure output. Output: files with node_ids."
	ToolGetASTNode          = "get_ast_node"
	DescGetASTNode          = "[ANALYSIS] level4/4: Get detailed AST node info. Input: repo_name, node_ids from previous calls. Output: codes, dependencies, references, implementations."
	// ToolWriteASTNode        = "write_ast_node"
)

var (
	SchemaListRepos           = GetJSONSchema(ListReposReq{})
	SchemaGetRepoStructure    = GetJSONSchema(GetRepoStructReq{})
	SchemaGetPackageStructure = GetJSONSchema(GetPackageStructReq{})
	SchemaGetFileStructure    = GetJSONSchema(GetFileStructReq{})
	SchemaGetASTNode          = GetJSONSchema(GetASTNodeReq{})
	SchemaGetFileSymbol       = GetJSONSchema(GetFileSymbolReq{})
	SchemaTreeRepo            = GetJSONSchema(TreeRepoReq{})
)

type ASTReadToolsOptions struct {
	// PatchOptions patch.Options
	RepoASTsDir  string
	DisableWatch bool
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

	// add a file watch on the RepoASTsDir (unless disabled)
	if !opts.DisableWatch {
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
	}

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

	tt, err = utils.InferTool(ToolGetFileSymbol,
		string(DescGetFileSymbol),
		ret.GetFileSymbol, utils.WithMarshalOutput(func(ctx context.Context, output interface{}) (string, error) {
			return abutil.MarshalJSONIndent(output)
		}))
	if err != nil {
		panic(err)
	}
	ret.tools[ToolGetFileSymbol] = tt

	tt, err = utils.InferTool(ToolTreeRepo,
		DescTreeRepo,
		ret.TreeRepo, utils.WithMarshalOutput(func(ctx context.Context, output interface{}) (string, error) {
			return abutil.MarshalJSONIndent(output)
		}))
	if err != nil {
		panic(err)
	}
	ret.tools[ToolTreeRepo] = tt
	return ret
}

func (t *ASTReadTools) GetTools() []Tool {
	ret := make([]Tool, 0, len(t.tools))
	for _, tt := range t.tools {
		ret = append(ret, tt)
	}
	return ret
}

func (t *ASTReadTools) GetTool(name string) Tool {
	return t.tools[name]
}

type ListReposReq struct{}

type ListReposResp struct {
	RepoNames []string `json:"repo_names" jsonschema:"description=the names of the repositories"`
}

func (t *ASTReadTools) ListRepos(ctx context.Context, req ListReposReq) (*ListReposResp, error) {
	ret := ListReposResp{}
	t.repos.Range(func(key, value interface{}) bool {
		ret.RepoNames = append(ret.RepoNames, key.(string))
		return true
	})
	return &ret, nil
}

type GetRepoStructReq struct {
	RepoName string `json:"repo_name" jsonschema:"description=the name of the repository (output of list_repos tool)"`
}

type GetRepoStructResp struct {
	Modules []ModuleStruct `json:"modules" jsonschema:"description=the module structure of the repository"`
	Error   string         `json:"error,omitempty" jsonschema:"description=the error message"`
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
	ModPath   uniast.ModPath `json:"mod_path,omitempty" jsonschema:"description=the module path"`
	PkgPath   uniast.PkgPath `json:"pkg_path,omitempty" jsonschema:"description=the package path"`
	Imports   []uniast.Import `json:"imports,omitempty" jsonschema:"description=the imports of the file"`
	Nodes     []NodeStruct    `json:"nodes,omitempty" jsonschema:"description=the node structs of the file"`
}

type NodeStruct struct {
	ModPath      uniast.ModPath `json:"mod_path,omitempty" jsonschema:"description=the module path"`
	PkgPath      uniast.PkgPath `json:"pkg_path,omitempty" jsonschema:"description=the package path"`
	Name         string         `json:"name" jsonschema:"description=the name of the node"`
	Type         string         `json:"type,omitempty" jsonschema:"description=the type of the node"`
	Signature    string         `json:"signature,omitempty" jsonschema:"description=the func signature of the node"`
	File         string         `json:"file,omitempty" jsonschema:"description=the file path of the node"`
	Line         int            `json:"line,omitempty" jsonschema:"description=the line of the node"`
	Codes        string         `json:"codes,omitempty" jsonschema:"description=the codes of the node"`
	Dependencies []NodeID       `json:"dependencies,omitempty" jsonschema:"description=the dependencies of the node"`
	References   []NodeID       `json:"references,omitempty" jsonschema:"description=the references of the node"`
	Implements   []NodeID       `json:"implements,omitempty" jsonschema:"description=the implements of the node"`
	Groups       []NodeID       `json:"groups,omitempty" jsonschema:"description=the groups of the node"`
	Inherits     []NodeID       `json:"inherits,omitempty" jsonschema:"description=the inherits of the node"`
}

type NodeID struct {
	ModPath uniast.ModPath `json:"mod_path" jsonschema:"description=module path of the node (from get_repo_structure)"`
	PkgPath uniast.PkgPath `json:"pkg_path" jsonschema:"description=package path of the node (from get_repo_structure)"`
	Name    string         `json:"name" jsonschema:"description=name of the node (from get_package_structure or get_file_structure)"`
}

func NewNodeID(id uniast.Identity) NodeID {
	return NodeID{
		ModPath: id.ModPath,
		PkgPath: id.PkgPath,
		Name:    id.Name,
	}
}

func (n NodeID) Identity() uniast.Identity {
	return uniast.Identity{
		ModPath: n.ModPath,
		PkgPath: n.PkgPath,
		Name:    n.Name,
	}
}

// FileNodeID 文件节点标识（用于 get_file_symbol 输出）
type FileNodeID struct {
	FilePath string `json:"file_path" jsonschema:"description=file path relative to repo root"`
	Name     string `json:"name" jsonschema:"description=symbol name in the file"`
}

// fileNodeIDGroupByPath 控制是否按 file_path 分组输出（默认 true）
var fileNodeIDGroupByPath = true

// SetFileNodeIDGroupByPath 设置是否按 file_path 分组输出
func SetFileNodeIDGroupByPath(group bool) {
	fileNodeIDGroupByPath = group
}

// fileNodeGroup 用于聚合相同 file_path 的 name（私有）
type fileNodeGroup struct {
	FilePath string   `json:"file_path"`
	Names    []string `json:"names"`
}

// groupFileNodeIDs 将 []FileNodeID 转换为 []fileNodeGroup
func groupFileNodeIDs(nodeIDs []FileNodeID) []fileNodeGroup {
	groupMap := make(map[string]*fileNodeGroup)

	for _, nid := range nodeIDs {
		key := nid.FilePath

		if group, exists := groupMap[key]; exists {
			group.Names = append(group.Names, nid.Name)
		} else {
			groupMap[key] = &fileNodeGroup{
				FilePath: nid.FilePath,
				Names:    []string{nid.Name},
			}
		}
	}

	result := make([]fileNodeGroup, 0, len(groupMap))
	for _, group := range groupMap {
		result = append(result, *group)
	}

	return result
}

type _FileNodeID FileNodeID

func (f FileNodeID) MarshalJSON() ([]byte, error) {
	if fileNodeIDGroupByPath {
		return json.Marshal(fileNodeGroup{
			FilePath: f.FilePath,
			Names:    []string{f.Name},
		})
	}
	return json.Marshal(_FileNodeID(f))
}

// convertNodeIDs 将 uniast.Relation 转换为 FileNodeID
func convertNodeIDs(repo *uniast.Repository, relations []uniast.Relation) []FileNodeID {
	result := make([]FileNodeID, 0, len(relations))
	for _, rel := range relations {
		if n := repo.GetNode(rel.Identity); n != nil {
			fl := n.FileLine()
			result = append(result, FileNodeID{
				FilePath: fl.File,
				Name:     rel.Identity.Name,
			})
		}
	}
	return result
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
				return nil, fmt.Errorf("repo '%s' not found. Use `list_repos` to get valid repo_name", candis[0])
			}
			return repo.(*uniast.Repository), nil
		} else if len(candis) > 1 {
			return nil, fmt.Errorf("repo '%s' is ambiguous, maybe you want one of %v", repoName, candis)
		} else {
			return nil, fmt.Errorf("repo '%s' not found. Use `list_repos` to get valid repo_name", repoName)
		}
	}
	return repo.(*uniast.Repository), nil
}

// GetRepoStructure list the packages and file-paths
func (t *ASTReadTools) GetRepoStructure(_ context.Context, req GetRepoStructReq) (*GetRepoStructResp, error) {
	log.Debug("get repo structure, req: %v", abutil.MarshalJSONIndentNoError(req))
	repo, err := t.getRepoAST(req.RepoName)
	if err != nil {
		return &GetRepoStructResp{
			Error: err.Error(),
		}, nil
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
	RepoName string         `json:"repo_name" jsonschema:"description=the name of the repository (output of list_repos tool)"`
	ModPath  uniast.ModPath `json:"mod_path" jsonschema:"description=the module path (output of get_repo_structure tool)"`
	PkgPath  uniast.PkgPath `json:"package_path" jsonschema:"description=the package path (output of get_repo_structure tool)"`
}

type GetPackageStructResp struct {
	Files []FileStruct `json:"files" jsonschema:"description=the file structures"`
	Error string       `json:"error,omitempty" jsonschema:"description=the error message"`
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
		return &GetPackageStructResp{
			Error: err.Error(),
		}, nil
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

	if len(resp.Files) == 0 {
		candidates := []string{}
		if mod, ok := repo.Modules[req.ModPath]; ok {
			for p := range mod.Packages {
				candidates = append(candidates, p)
			}
		}
		resp.Error = fmt.Sprintf("package '%s' not found, maybe you want one of %v", req.PkgPath, candidates)
	}

	log.Debug("get repo structure, resp: %v", abutil.MarshalJSONIndentNoError(resp))
	return resp, nil
}

type GetFileStructReq struct {
	RepoName string `json:"repo_name" jsonschema:"description=the name of the repository (output of list_repos tool)"`
	FilePath string `json:"file_path" jsonschema:"description=relative file path (output of get_repo_structure tool, e.g., 'src/main.go')"`
}

type GetFileStructResp struct {
	FileStruct
	Error string `json:"error,omitempty" jsonschema:"description=the error message"`
}

// GetFileStruct get node list, each node only includes ID\Type\Signature
func (t *ASTReadTools) GetFileStructure(_ context.Context, req GetFileStructReq) (*GetFileStructResp, error) {
	log.Debug("get file structure, req: %v", abutil.MarshalJSONIndentNoError(req))
	resp, err := t.getFileStructure(context.Background(), req, true)
	if err != nil {
		return &GetFileStructResp{
			Error: err.Error(),
		}, nil
	}
	log.Debug("get repo structure, resp: %v", abutil.MarshalJSONIndentNoError(resp))
	return resp, nil
}

func (t *ASTReadTools) getFileStructure(_ context.Context, req GetFileStructReq, needNodeDetail bool) (*GetFileStructResp, error) {
	repo, err := t.getRepoAST(req.RepoName)
	if err != nil {
		return nil, err
	}

	resp := new(GetFileStructResp)
	file, mod := repo.GetFile(req.FilePath)
	if file == nil {
		return nil, fmt.Errorf("file '%s' not found. Use 'get_repo_structure' to get valid file paths", req.FilePath)
	}

	nodes := repo.GetFileNodes(req.FilePath)
	ff := FileStruct{
		FilePath: req.FilePath,
		ModPath:  mod.Name,
		PkgPath:  file.Package,
	}
	if needNodeDetail {
		ff.Imports = file.Imports
	}
	// If nodes count > 500, only show name + line
	simplifiedOutput := len(nodes) > 500
	for _, n := range nodes {
		nn := NodeStruct{
			Name: n.Identity.Name,
		}
		if needNodeDetail {
			nn.Line = n.FileLine().Line
			if !simplifiedOutput && n.Type != uniast.VAR {
				nn.Signature = n.Signature()
			}
		}
		ff.Nodes = append(ff.Nodes,
			nn,
		)
	}
	resp.FileStruct = ff
	return resp, nil
}

type GetASTNodeReq struct {
	RepoName string   `json:"repo_name" jsonschema:"description=the name of the repository (output of list_repos tool)"`
	NodeIDs  []NodeID `json:"node_ids" jsonschema:"description=the identities of the ast node (output of get_package_structure or get_file_structure tool)"`
}

type GetASTNodeResp struct {
	Nodes []NodeStruct `json:"nodes" jsonschema:"description=the ast nodes"`
	Error string       `json:"error,omitempty" jsonschema:"description=the error message"`
}

type GetFileSymbolReq struct {
	RepoName string `json:"repo_name" jsonschema:"description=the name of the repository (output of list_repos tool)"`
	FilePath string `json:"file_path" jsonschema:"description=the file path (output of get_repo_structure tool)"`
	Name     string `json:"name" jsonschema:"description=the name of the symbol (function, type, or variable) to query"`
}

type GetFileSymbolResp struct {
	Node  FileNodeStruct `json:"node" jsonschema:"description=the ast node"`
	Error string         `json:"error,omitempty" jsonschema:"description=the error message"`
}

// FileNodeStruct 文件节点结构（使用 FileNodeID）
type FileNodeStruct struct {
	Name         string       `json:"name" jsonschema:"description=the name of the node"`
	Type         string       `json:"type,omitempty" jsonschema:"description=the type of the node"`
	Signature    string       `json:"signature,omitempty" jsonschema:"description=the func signature of the node"`
	File         string       `json:"file,omitempty" jsonschema:"description=the file path of the node"`
	Line         int          `json:"line,omitempty" jsonschema:"description=the line of the node"`
	Codes        string       `json:"codes,omitempty" jsonschema:"description=the codes of the node"`
	Dependencies []FileNodeID `json:"dependencies,omitempty" jsonschema:"description=the dependencies of the node"`
	References   []FileNodeID `json:"references,omitempty" jsonschema:"description=the references of the node"`
	Implements   []FileNodeID `json:"implements,omitempty" jsonschema:"description=the implements of the node"`
	Groups       []FileNodeID `json:"groups,omitempty" jsonschema:"description=the groups of the node"`
	Inherits     []FileNodeID `json:"inherits,omitempty" jsonschema:"description=the inherits of the node"`
}

// MarshalJSON 自定义JSON序列化，实现所有关系字段的分组输出
type _FileNodeStruct FileNodeStruct

func (n FileNodeStruct) MarshalJSON() ([]byte, error) {
	if fileNodeIDGroupByPath {
		aux := &struct {
			Dependencies []fileNodeGroup `json:"dependencies,omitempty"`
			References   []fileNodeGroup `json:"references,omitempty"`
			Implements   []fileNodeGroup `json:"implements,omitempty"`
			Groups       []fileNodeGroup `json:"groups,omitempty"`
			Inherits     []fileNodeGroup `json:"inherits,omitempty"`
			*_FileNodeStruct
		}{
			Dependencies: groupFileNodeIDs(n.Dependencies),
			References:   groupFileNodeIDs(n.References),
			Implements:   groupFileNodeIDs(n.Implements),
			Groups:       groupFileNodeIDs(n.Groups),
			Inherits:     groupFileNodeIDs(n.Inherits),
			_FileNodeStruct: (*_FileNodeStruct)(&n),
		}
		return json.Marshal(aux)
	}
	return json.Marshal(_FileNodeStruct(n))
}

func (t *ASTReadTools) GetASTNode(_ context.Context, params GetASTNodeReq) (*GetASTNodeResp, error) {
	log.Debug("get ast node, req: %v", abutil.MarshalJSONIndentNoError(params))

	repo, err := t.getRepoAST(params.RepoName)
	if err != nil {
		return &GetASTNodeResp{
			Error: err.Error(),
		}, nil
	}

	resp := new(GetASTNodeResp)
	for _, nid := range params.NodeIDs {
		id := nid.Identity()
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
			ModPath:      node.Identity.ModPath,
			PkgPath:      node.Identity.PkgPath,
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

	if len(resp.Nodes) == 0 {
		resp.Error = "node not found. Use `get_package_structure` to list all valid nodes. If it cannot be confirmed, call `get_file_structure` for detailed node information"
	}

	log.Debug("get repo structure, resp: %v", abutil.MarshalJSONIndentNoError(resp))
	return resp, nil
}

// GetFileSymbol get detailed AST node info by file path and symbol name
func (t *ASTReadTools) GetFileSymbol(_ context.Context, req GetFileSymbolReq) (*GetFileSymbolResp, error) {
	log.Debug("get file symbol, req: %v", abutil.MarshalJSONIndentNoError(req))

	// 加载仓库
	repo, err := t.getRepoAST(req.RepoName)
	if err != nil {
		return &GetFileSymbolResp{
			Error: err.Error(),
		}, nil
	}

	// 查找文件
	file, _ := repo.GetFile(req.FilePath)
	if file == nil {
		return &GetFileSymbolResp{
			Error: fmt.Errorf("file '%s' not found. Use 'get_repo_structure' to get valid file paths", req.FilePath).Error(),
		}, nil
	}

	// 在文件中查找符号
	nodes := repo.GetFileNodes(req.FilePath)
	var targetNode *uniast.Node
	var found bool

	for _, node := range nodes {
		if node.Identity.Name == req.Name {
			targetNode = node
			found = true
			break
		}
	}

	if !found {
		return &GetFileSymbolResp{
			Error: fmt.Sprintf("symbol '%s' not found in file '%s'. Use 'get_file_structure' to list all symbols in the file", req.Name, req.FilePath),
		}, nil
	}

	// 构建 FileNodeStruct
	fl := targetNode.FileLine()
	nodeStruct := FileNodeStruct{
		Name:      targetNode.Identity.Name,
		Type:      targetNode.Type.String(),
		File:      fl.File,
		Line:      fl.Line,
		Codes:     targetNode.Content(),
		Signature: targetNode.Signature(),
		// 使用抽象函数转换所有关系字段
		Dependencies: convertNodeIDs(repo, targetNode.Dependencies),
		References:   convertNodeIDs(repo, targetNode.References),
		Implements:   convertNodeIDs(repo, targetNode.Implements),
		Inherits:     convertNodeIDs(repo, targetNode.Inherits),
		Groups:       convertNodeIDs(repo, targetNode.Groups),
	}

	log.Debug("get file symbol, resp: %v", abutil.MarshalJSONIndentNoError(&GetFileSymbolResp{Node: nodeStruct}))
	return &GetFileSymbolResp{
		Node: nodeStruct,
	}, nil
}

type TreeRepoReq struct {
	RepoName string `json:"repo_name" jsonschema:"description=the name of the repository (output of list_repos tool)"`
}

type TreeRepoResp struct {
	Files map[string][]string `json:"files" jsonschema:"description=map of directory path to file list (directories end with '/')"`
	Error string              `json:"error,omitempty" jsonschema:"description=the error message"`
}

// TreeRepo returns a map of package paths to file lists, with directories ending in '/'
func (t *ASTReadTools) TreeRepo(_ context.Context, req TreeRepoReq) (*TreeRepoResp, error) {
	log.Debug("tree repo, req: %v", abutil.MarshalJSONIndentNoError(req))
	repo, err := t.getRepoAST(req.RepoName)
	if err != nil {
		return &TreeRepoResp{
			Error: err.Error(),
		}, nil
	}

	// 收集所有文件，按目录聚合
	files := make(map[string][]string)
	for _, mod := range repo.Modules {
		if mod.IsExternal() {
			continue
		}
		for _, file := range mod.Files {
			if file.Package == "" {
				continue
			}
			// 过滤掉非当前仓库的文件（以 .. 开头或包含 ..）
			if strings.HasPrefix(file.Path, "..") {
				continue
			}
			// 获取文件的目录路径
			dir := filepath.Dir(file.Path)
			if dir == "." {
				dir = "./"
			}
			// 添加 '/' 后缀
			if dir != "" && dir != "./" && !strings.HasSuffix(dir, "/") {
				dir = dir + "/"
			}
			// 获取文件名
			name := filepath.Base(file.Path)
			files[dir] = append(files[dir], name)
		}
	}

	// 对每个目录下的文件列表进行排序
	for dir := range files {
		sort.Strings(files[dir])
	}

	log.Debug("tree repo, resp: %v", abutil.MarshalJSONIndentNoError(&TreeRepoResp{Files: files}))
	return &TreeRepoResp{Files: files}, nil
}
