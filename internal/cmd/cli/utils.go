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

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/spf13/cobra"
)

// getASTsDir returns the ASTs directory path from command flags or default location.
func getASTsDir(cmd *cobra.Command) (string, error) {
	astsDir, err := cmd.Flags().GetString("asts-dir")
	if err != nil {
		return "", err
	}
	if astsDir == "" {
		astsDir = filepath.Join(os.Getenv("HOME"), ".asts")
	}
	if _, err := os.Stat(astsDir); os.IsNotExist(err) {
		return "", fmt.Errorf("ASTs directory does not exist: %s", astsDir)
	}
	return astsDir, nil
}

// findRepoFile 查找 repo 对应的 JSON 文件
func findRepoFile(astsDir, repoName string) string {
	// 先尝试直接匹配文件名
	patterns := []string{
		repoName + ".json",
		repoName,
	}

	// 处理特殊字符
	encoded := strings.ReplaceAll(repoName, "/", "-")
	encoded = strings.ReplaceAll(encoded, ":", "-")
	patterns = append(patterns, encoded+".json")

	// glob 模式
	patterns = append(patterns, "*-"+encoded+".json")

	for _, pattern := range patterns {
		if match, err := filepath.Glob(filepath.Join(astsDir, pattern)); err == nil {
			for _, f := range match {
				return f
			}
		}
	}

	// 遍历所有文件匹配
	files, _ := filepath.Glob(filepath.Join(astsDir, "*.json"))
	for _, f := range files {
		if strings.HasSuffix(f, "_repo_index.json") || strings.HasSuffix(f, ".repo_index.json") {
			continue
		}
		// 读取 id 字段匹配
		if data, err := os.ReadFile(f); err == nil {
			if val, err := sonic.Get(data, "id"); err == nil {
				if id, err := val.String(); err == nil && id == repoName {
					return f
				}
			}
		}
	}

	return ""
}

// loadRepoModules 用 sonic 读取 repo 的 Modules 结构
func loadRepoModules(repoFile string) (map[string]interface{}, error) {
	data, err := os.ReadFile(repoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read repo file: %w", err)
	}

	modsVal, err := sonic.Get(data, "Modules")
	if err != nil {
		return nil, fmt.Errorf("failed to get modules: %w", err)
	}
	mods, err := modsVal.Map()
	if err != nil {
		return nil, fmt.Errorf("failed to parse modules: %w", err)
	}
	return mods, nil
}

// pathMatchesCwd 检查 mappings 中的文件名对应的 json 文件的 Path 字段是否匹配 cwd
func pathMatchesCwd(astsDir, filename, cwd string) bool {
	repoFile := filepath.Join(astsDir, filename)
	data, err := os.ReadFile(repoFile)
	if err != nil {
		return false
	}
	val, err := sonic.Get(data, "Path")
	if err != nil {
		return false
	}
	path, err := val.String()
	if err != nil {
		return false
	}
	return path == cwd
}

// loadRepoFileData 读取整个 repo JSON 文件，返回 raw data 供后续 sonic.Get 按需读取
func loadRepoFileData(repoFile string) ([]byte, error) {
	return os.ReadFile(repoFile)
}

// getModuleKeys 获取 Modules 的所有 key（不加载 value）
func getModuleKeys(data []byte) ([]string, error) {
	val, err := sonic.Get(data, "Modules")
	if err != nil {
		return nil, err
	}
	var keys []string
	iter, err := val.Properties()
	if err != nil {
		return nil, err
	}
	var p ast.Pair
	for iter.Next(&p) {
		keys = append(keys, p.Key)
	}
	return keys, nil
}

// getPackageKeys 获取指定 module 下 Packages 的所有 key（不加载 value）
func getPackageKeys(data []byte, modPath string) ([]string, error) {
	val, err := sonic.Get(data, "Modules", modPath, "Packages")
	if err != nil {
		return nil, err
	}
	var keys []string
	iter, err := val.Properties()
	if err != nil {
		return nil, err
	}
	var p ast.Pair
	for iter.Next(&p) {
		keys = append(keys, p.Key)
	}
	return keys, nil
}

// iterModFiles 遍历指定 mod 的 Files，只返回 file path（不加载 value）
func iterModFiles(data []byte, modPath string) ([]string, error) {
	val, err := sonic.Get(data, "Modules", modPath, "Files")
	if err != nil {
		return nil, err
	}
	iter, err := val.Properties()
	if err != nil {
		return nil, err
	}
	var keys []string
	var p ast.Pair
	for iter.Next(&p) {
		keys = append(keys, p.Key)
	}
	return keys, nil
}

// iterSymbolNameFile 遍历指定 category (Functions/Types/Vars) 的所有 symbol
// 只读取 Name 和 File 字段，不读取完整内容
// 返回: [][]string{{name, file}, ...}
func iterSymbolNameFile(data []byte, modPath, pkgPath, category string) ([][]string, error) {
	val, err := sonic.Get(data, "Modules", modPath, "Packages", pkgPath, category)
	if err != nil {
		return nil, err
	}
	if !val.Exists() {
		return nil, nil
	}
	iter, err := val.Properties()
	if err != nil {
		return nil, err
	}
	var results [][]string
	var p ast.Pair
	for iter.Next(&p) {
		symName := p.Key
		// 只读取 File 字段
		fileVal, err := sonic.Get(data, "Modules", modPath, "Packages", pkgPath, category, symName, "File")
		if err != nil || !fileVal.Exists() {
			continue
		}
		filePath, err := fileVal.String()
		if err != nil {
			continue
		}
		results = append(results, []string{symName, filePath})
	}
	return results, nil
}

// findPkgPathByFile 通过 filePath 查找 pkgPath
// 返回: modPath, pkgPath
// 使用 File.ModPath + File.PkgPath 实现 O(1) 查找
func findPkgPathByFile(data []byte, filePath string) (string, string, error) {
	if verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] findPkgPathByFile: filePath=%s\n", filePath)
	}

	// 1. 遍历 Modules，尝试直接通过 Files[filePath] 找到 File
	modsVal, err := sonic.Get(data, "Modules")
	if err != nil {
		return "", "", err
	}
	modsIter, err := modsVal.Properties()
	if err != nil {
		return "", "", err
	}
	var modPair ast.Pair

	for modsIter.Next(&modPair) {
		modPath := modPair.Key

		// 直接查找 Module.Files[filePath]
		fileVal, err := sonic.Get(data, "Modules", modPath, "Files", filePath)
		if err != nil || !fileVal.Exists() {
			continue
		}

		// 读取 File.ModPath 和 File.PkgPath（JSON 字段名是大写）
		modPathVal, _ := fileVal.Get("ModPath").String()
		pkgPathVal, _ := fileVal.Get("PkgPath").String()

		if modPathVal != "" && pkgPathVal != "" {
			if verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] HIT via Files: modPath=%s, pkgPath=%s\n", modPathVal, pkgPathVal)
			}
			return modPathVal, pkgPathVal, nil
		}
	}

	// 2. 回退：使用旧的推导方式（兼容旧数据）
	if verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] fallback to derived path\n")
	}
	return findPkgPathByFileDerived(data, filePath)
}

// findPkgPathByFileDerived 通过推导查找 pkgPath（旧逻辑，兼容）
func findPkgPathByFileDerived(data []byte, filePath string) (string, string, error) {
	derivedPkg := filepath.Dir(filePath)
	if derivedPkg == "." {
		derivedPkg = ""
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] findPkgPathByFileDerived: filePath=%s, derivedPkg=%s\n", filePath, derivedPkg)
	}

	modsVal, err := sonic.Get(data, "Modules")
	if err != nil {
		return "", "", err
	}
	modsIter, err := modsVal.Properties()
	if err != nil {
		return "", "", err
	}
	var modPair ast.Pair

	for modsIter.Next(&modPair) {
		modPath := modPair.Key

		var fullPkgPath string
		if derivedPkg == "" {
			fullPkgPath = modPath
		} else {
			fullPkgPath = modPath + "/" + derivedPkg
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] trying direct: modPath=%s, fullPkgPath=%s\n", modPath, fullPkgPath)
		}

		if matched, _ := pkgHasFile(data, modPath, fullPkgPath, filePath, "", "Functions"); matched {
			if verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] HIT via direct: modPath=%s, fullPkgPath=%s\n", modPath, fullPkgPath)
			}
			return modPath, fullPkgPath, nil
		}
		if matched, _ := pkgHasFile(data, modPath, fullPkgPath, filePath, "", "Types"); matched {
			if verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] HIT via direct: modPath=%s, fullPkgPath=%s\n", modPath, fullPkgPath)
			}
			return modPath, fullPkgPath, nil
		}
		if matched, _ := pkgHasFile(data, modPath, fullPkgPath, filePath, "", "Vars"); matched {
			if verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] HIT via direct: modPath=%s, fullPkgPath=%s\n", modPath, fullPkgPath)
			}
			return modPath, fullPkgPath, nil
		}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] fallback to findPkgPathByFileFullLoad\n")
	}
	return findPkgPathByFileFullLoad(data, filePath)
}

// findPkgPathByFileFullLoad 全量加载方案：一次性加载 Modules.Packages，建立 file→{modPath,pkgPath} 索引
func findPkgPathByFileFullLoad(data []byte, filePath string) (string, string, error) {
	// 一次性反序列化 Modules.Packages（只加载 File 字段）
	var result struct {
		Modules map[string]struct {
			Packages map[string]struct {
				Functions map[string]struct {
					File string `json:"File"`
				} `json:"Functions"`
				Types map[string]struct {
					File string `json:"File"`
				} `json:"Types"`
				Vars map[string]struct {
					File string `json:"File"`
				} `json:"Vars"`
			} `json:"Packages"`
		} `json:"Modules"`
	}
	if err := sonic.Unmarshal(data, &result); err != nil {
		return "", "", fmt.Errorf("unmarshal failed: %w", err)
	}

	// 遍历建立 file → {modPath, pkgPath} 索引
	fileIndex := make(map[string][2]string)
	for modPath, mod := range result.Modules {
		for pkgPath, pkg := range mod.Packages {
			for _, fn := range pkg.Functions {
				if fn.File != "" {
					fileIndex[fn.File] = [2]string{modPath, pkgPath}
				}
			}
			for _, t := range pkg.Types {
				if t.File != "" {
					fileIndex[t.File] = [2]string{modPath, pkgPath}
				}
			}
			for _, v := range pkg.Vars {
				if v.File != "" {
					fileIndex[v.File] = [2]string{modPath, pkgPath}
				}
			}
		}
	}

	// 直接查找
	if info, ok := fileIndex[filePath]; ok {
		return info[0], info[1], nil
	}

	return "", "", fmt.Errorf("file not found: %s", filePath)
}

// pkgHasFile 检查指定 category (Functions/Types/Vars) 中是否有匹配的 file
// 如果 symbolName 不为空，则同时匹配 symbolName
func pkgHasFile(data []byte, modPath, pkgPath, filePath, symbolName, category string) (bool, error) {
	categoryVal, err := sonic.Get(data, "Modules", modPath, "Packages", pkgPath, category)
	if err != nil {
		return false, err
	}
	if !categoryVal.Exists() {
		return false, nil
	}

	iter, err := categoryVal.Properties()
	if err != nil {
		return false, err
	}
	var pair ast.Pair

	for iter.Next(&pair) {
		symName := pair.Key

		// 如果指定了 symbolName，则只检查该 symbol
		if symbolName != "" && symName != symbolName {
			continue
		}

		// 只读取 File 字段进行比对
		fileVal, err := sonic.Get(data, "Modules", modPath, "Packages", pkgPath, category, symName, "File")
		if err != nil {
			continue
		}
		if !fileVal.Exists() {
			continue
		}
		fnFile, err := fileVal.String()
		if err != nil {
			continue
		}

		if fnFile == filePath {
			return true, nil
		}
	}

	return false, nil
}

// getSymbolByFileFull 完整读取 package 内容后匹配 symbol
// 在找到目标 pkg 后调用此函数读取完整内容
func getSymbolByFileFull(data []byte, modPath, pkgPath, filePath, symbolName string) (map[string]interface{}, error) {
	// 读取目标 package 的内容
	pkgVal, err := sonic.Get(data, "Modules", modPath, "Packages", pkgPath)
	if err != nil {
		return nil, fmt.Errorf("sonic.Get(Packages) failed for %s/%s: %w", modPath, pkgPath, err)
	}
	if !pkgVal.Exists() {
		return nil, fmt.Errorf("Packages does not exist for %s/%s", modPath, pkgPath)
	}
	pkg, err := pkgVal.Map()
	if err != nil {
		return nil, fmt.Errorf("pkgVal.Map() failed: %w", err)
	}

	// 检查 Functions
	if fns, ok := pkg["Functions"].(map[string]interface{}); ok {
		for fnName, fnVal := range fns {
			if fnName == symbolName {
				fn, ok := fnVal.(map[string]interface{})
				if !ok {
					continue
				}
				if fn["File"] == filePath {
					fn["node_type"] = "FUNC"
					return fn, nil
				}
			}
		}
	}
	// 检查 Types
	if types, ok := pkg["Types"].(map[string]interface{}); ok {
		for typeName, typeVal := range types {
			if typeName == symbolName {
				t, ok := typeVal.(map[string]interface{})
				if !ok {
					continue
				}
				if t["File"] == filePath {
					t["node_type"] = "TYPE"
					return t, nil
				}
			}
		}
	}
	// 检查 Vars
	if vars, ok := pkg["Vars"].(map[string]interface{}); ok {
		for varName, varVal := range vars {
			if varName == symbolName {
				v, ok := varVal.(map[string]interface{})
				if !ok {
					continue
				}
				if v["File"] == filePath {
					v["node_type"] = "VAR"
					return v, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("symbol not found")
}

// getFileSymbolsByFile 按需读取: modPath → pkgPath → 获取该文件所有 symbols
func getFileSymbolsByFile(data []byte, modPath, pkgPath, filePath string) ([]map[string]interface{}, error) {
	// 读取目标 package 的内容
	pkgVal, err := sonic.Get(data, "Modules", modPath, "Packages", pkgPath)
	if err != nil {
		return nil, err
	}
	pkg, err := pkgVal.Map()
	if err != nil {
		return nil, err
	}

	var nodes []map[string]interface{}

	// 检查 Functions
	if fns, ok := pkg["Functions"].(map[string]interface{}); ok {
		for _, fnVal := range fns {
			fn, ok := fnVal.(map[string]interface{})
			if !ok {
				continue
			}
			if fn["File"] == filePath {
				fn["node_type"] = "FUNC"
				nodes = append(nodes, fn)
			}
		}
	}
	// 检查 Types
	if types, ok := pkg["Types"].(map[string]interface{}); ok {
		for _, typeVal := range types {
			t, ok := typeVal.(map[string]interface{})
			if !ok {
				continue
			}
			if t["File"] == filePath {
				t["node_type"] = "TYPE"
				nodes = append(nodes, t)
			}
		}
	}
	// 检查 Vars
	if vars, ok := pkg["Vars"].(map[string]interface{}); ok {
		for _, varVal := range vars {
			v, ok := varVal.(map[string]interface{})
			if !ok {
				continue
			}
			if v["File"] == filePath {
				v["node_type"] = "VAR"
				nodes = append(nodes, v)
			}
		}
	}
	return nodes, nil
}

// getSymbolReferences 用 sonic 按需读取 Graph 节点的 Dependencies 和 References
// Identity 格式: {ModPath}?{PkgPath}#{Name}
func getSymbolReferences(data []byte, modPath, pkgPath, symbolName string) ([]map[string]string, error) {
	// Graph key 格式: {ModPath}?{PkgPath}#{Name}
	// Python 根目录文件的 PkgPath 是 "."，需要映射为 "."
	graphKey := modPath + "?" + pkgPath + "#" + symbolName

	// 使用嵌套 Get 避免特殊字符 (?#) 处理问题
	graphVal, err := sonic.Get(data, "Graph")
	if err != nil {
		return nil, err
	}
	nodeVal := graphVal.Get(graphKey)
	if !nodeVal.Exists() {
		return nil, nil // 没有 Graph 节点，返回空
	}

	var refs []map[string]string

	// 读取 Dependencies（当前节点依赖的）
	depsVal := nodeVal.Get("Dependencies")
	if depsVal.Exists() {
		deps, err := parseRelationItems(*depsVal, "Dependency")
		if err != nil {
			return nil, err
		}
		refs = append(refs, deps...)
	}

	// 读取 References（引用当前节点的）
	refsVal := nodeVal.Get("References")
	if refsVal.Exists() {
		refsItems, err := parseRelationItems(*refsVal, "Reference")
		if err != nil {
			return nil, err
		}
		refs = append(refs, refsItems...)
	}

	return refs, nil
}

// parseRelationItems 解析关系数组，添加 kind 来源标记
func parseRelationItems(val ast.Node, kind string) ([]map[string]string, error) {
	arr, err := val.Array()
	if err != nil {
		return nil, err
	}

	var refs []map[string]string
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		ref := make(map[string]string)
		// 使用固定的 kind，不依赖 JSON 中的 Kind 字段
		ref["kind"] = kind
		if v, ok := m["Name"].(string); ok {
			ref["name"] = v
		}
		if v, ok := m["ModPath"].(string); ok {
			ref["mod_path"] = v
		}
		if v, ok := m["PkgPath"].(string); ok {
			ref["pkg_path"] = v
		}
		if f, ok := m["File"].(string); ok {
			ref["file"] = f
		}
		if n, ok := m["Line"].(float64); ok {
			ref["line"] = fmt.Sprintf("%d", int(n))
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

// findSymbolFile 通过 ModPath + PkgPath + Name 反向查找 FilePath
// 路径格式: .Modules[ModPath].Packages[PkgPath].Functions[Name].File
func findSymbolFile(data []byte, modPath, pkgPath, name string) string {
	if modPath == "" || pkgPath == "" || name == "" {
		return ""
	}

	// 尝试 Functions
	fileVal, _ := sonic.Get(data, "Modules", modPath, "Packages", pkgPath, "Functions", name, "File")
	if fileVal.Exists() {
		if f, err := fileVal.String(); err == nil {
			return f
		}
	}

	// 尝试 Types
	fileVal, _ = sonic.Get(data, "Modules", modPath, "Packages", pkgPath, "Types", name, "File")
	if fileVal.Exists() {
		if f, err := fileVal.String(); err == nil {
			return f
		}
	}

	// 尝试 Vars
	fileVal, _ = sonic.Get(data, "Modules", modPath, "Packages", pkgPath, "Vars", name, "File")
	if fileVal.Exists() {
		if f, err := fileVal.String(); err == nil {
			return f
		}
	}

	return ""
}
