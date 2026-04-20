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
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/abcoder/lang/utils"
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/spf13/cobra"
)

// buildNameToLocations 从 JSON 数据构建 NameToLocations
// 如果 pathFilter 不为空，则只收集匹配前缀的 file
// 返回: name -> type -> fileSet (去重)
func buildNameToLocations(data []byte, pathFilter string) (map[string]map[string]map[string]bool, error) {
	// 一次性反序列化整个 Modules
	var result struct {
		Modules map[string]*uniast.Module `json:"modules"`
	}
	if err := sonic.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	// name -> type -> files (去重)
	nameToTypeFiles := make(map[string]map[string]map[string]bool)

	// 遍历所有模块
	for _, mod := range result.Modules {
		// 跳过外部模块
		if mod.IsExternal() {
			continue
		}

		// 遍历所有包
		for _, pkg := range mod.Packages {
			// 提取 Functions
			for name, fn := range pkg.Functions {
				if pathFilter != "" && !strings.HasPrefix(fn.File, pathFilter) {
					continue
				}
				if nameToTypeFiles[name] == nil {
					nameToTypeFiles[name] = make(map[string]map[string]bool)
				}
				if nameToTypeFiles[name]["FUNC"] == nil {
					nameToTypeFiles[name]["FUNC"] = make(map[string]bool)
				}
				nameToTypeFiles[name]["FUNC"][fn.File] = true
			}

			// 提取 Types
			for name, typ := range pkg.Types {
				if pathFilter != "" && !strings.HasPrefix(typ.FileLine.File, pathFilter) {
					continue
				}
				if nameToTypeFiles[name] == nil {
					nameToTypeFiles[name] = make(map[string]map[string]bool)
				}
				if nameToTypeFiles[name]["TYPE"] == nil {
					nameToTypeFiles[name]["TYPE"] = make(map[string]bool)
				}
				nameToTypeFiles[name]["TYPE"][typ.FileLine.File] = true
			}

			// 提取 Vars
			for name, v := range pkg.Vars {
				if pathFilter != "" && !strings.HasPrefix(v.FileLine.File, pathFilter) {
					continue
				}
				if nameToTypeFiles[name] == nil {
					nameToTypeFiles[name] = make(map[string]map[string]bool)
				}
				if nameToTypeFiles[name]["VAR"] == nil {
					nameToTypeFiles[name]["VAR"] = make(map[string]bool)
				}
				nameToTypeFiles[name]["VAR"][v.FileLine.File] = true
			}
		}
	}

	return nameToTypeFiles, nil
}

// saveNameToLocations 写回 NameToLocations 到 JSON 文件
func saveNameToLocations(repoFile string, nameToLocs map[string][]string) error {
	data, err := os.ReadFile(repoFile)
	if err != nil {
		return err
	}

	// 使用标准库 JSON 反序列化
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	// 添加 NameToLocations
	result["NameToLocations"] = nameToLocs

	// 重新Marshal（保持缩进格式）
	prettyJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	// 写入 .tmp 再 rename
	tmpPath := repoFile + ".tmp"
	if err := utils.MustWriteFile(tmpPath, prettyJSON); err != nil {
		return err
	}
	return os.Rename(tmpPath, repoFile)
}

type Symbol struct {
	Name string `json:"name"`
	File string `json:"file"`
	Type string `json:"type"` // FUNC, TYPE, VAR
}

type ExtractResult struct {
	RepoName string                              `json:"repo_name"`
	Files    map[string]map[string][]string      `json:"files"` // file -> type -> names
}

func newExtractSymbolCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "extract_symbol <repo_name>",
		Short: "Extract all symbols from repo JSON",
		Long: `Extract all symbol names and file paths from a repository's uniast JSON.
Only extracts filepath + name (no content), for use with search_node.`,
		Example: `abcoder cli extract_symbol myrepo`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			astsDir, err := getASTsDir(cmd)
			if err != nil {
				return err
			}

			repoName := args[0]

			repoFile := findRepoFile(astsDir, repoName)
			if repoFile == "" {
				return fmt.Errorf("repo not found: %s", repoName)
			}

			data, err := os.ReadFile(repoFile)
			if err != nil {
				return fmt.Errorf("failed to read repo file: %w", err)
			}

			// 方式1: 优先用 sonic 读取 NameToLocations
			nameToLocsVal, err := sonic.Get(data, "NameToLocations")
			if err == nil && nameToLocsVal.Exists() {
				if verbose {
					fmt.Fprintf(os.Stderr, "[VERBOSE] using existing NameToLocations\n")
				}

				// 获取所有 name keys
				nameToLocsMap, _ := nameToLocsVal.Map()

				// 转换为输出格式: file -> type -> names
				files := make(map[string]map[string][]string)
				for name := range nameToLocsMap {
					filesVal, _ := sonic.Get(data, "NameToLocations", name, "Files")
					if filesVal.Exists() {
						fileList, err := filesVal.Array()
						if err == nil {
							for _, f := range fileList {
								fileStr, _ := f.(string)
								if files[fileStr] == nil {
									files[fileStr] = map[string][]string{
										"FUNC": {},
										"TYPE": {},
										"VAR":  {},
									}
								}
								// NameToLocations 不区分类型，都归为 FUNC
								files[fileStr]["FUNC"] = append(files[fileStr]["FUNC"], name)
							}
						}
					}
				}

				result := ExtractResult{
					RepoName: repoName,
					Files:    files,
				}
				b, _ := json.MarshalIndent(result, "", "  ")
				fmt.Fprintf(os.Stdout, "%s\n", b)
				return nil
			}

			// 方式2: 没有 NameToLocations，遍历提取并写回 JSON
			if verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] building NameToLocations\n")
			}

			// 使用公共函数构建
			nameToTypeFiles, err := buildNameToLocations(data, "")
			if err != nil {
				return err
			}

			// 转换为 NameToLocations 格式: name -> []file
			// 拍平 type，只保留 files
			nameToLocsMap := make(map[string][]string)
			for name, typeFiles := range nameToTypeFiles {
				fileSet := make(map[string]bool)
				for _, files := range typeFiles {
					for file := range files {
						fileSet[file] = true
					}
				}
				var fileList []string
				for file := range fileSet {
					fileList = append(fileList, file)
				}
				nameToLocsMap[name] = fileList
			}

			// 写回 JSON
			if err := saveNameToLocations(repoFile, nameToLocsMap); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save NameToLocations: %v\n", err)
			} else if verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] saved NameToLocations to %s\n", repoFile)
			}

			// 转换为输出格式: file -> type -> names
			files := make(map[string]map[string][]string)
			for name, typeFiles := range nameToTypeFiles {
				for typ, fileSet := range typeFiles {
					for file := range fileSet {
						if files[file] == nil {
							files[file] = map[string][]string{
								"FUNC": {},
								"TYPE": {},
								"VAR":  {},
							}
						}
						files[file][typ] = append(files[file][typ], name)
					}
				}
			}

			// 过滤掉空的 TYPE 和 VAR
			for file, types := range files {
				if len(types["TYPE"]) == 0 {
					delete(types, "TYPE")
				}
				if len(types["VAR"]) == 0 {
					delete(types, "VAR")
				}
				if len(types["FUNC"]) == 0 {
					delete(files, file)
				}
			}

			result := ExtractResult{
				RepoName: repoName,
				Files:    files,
			}

			b, _ := json.MarshalIndent(result, "", "  ")
			fmt.Fprintf(os.Stdout, "%s\n", b)
			return nil
		},
	}
}
