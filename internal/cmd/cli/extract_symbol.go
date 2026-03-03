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
	"path/filepath"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/abcoder/lang/utils"
	"github.com/spf13/cobra"
)

const indexDir = ".index"

type SymbolIndex struct {
	Mtime int64                  `json:"mtime"`
	Data  map[string][]NameMatch `json:"data"` // name -> []NameMatch
}

type NameMatch struct {
	File string `json:"file"`
	Type string `json:"type"`
}

// saveSymbolIndex 保存符号索引到 ~/.asts/.index/{repo}.idx
func saveSymbolIndex(astsDir, repoName, repoFile string, data map[string][]NameMatch) error {
	// 获取 repo 文件的 mtime
	info, err := os.Stat(repoFile)
	if err != nil {
		return fmt.Errorf("stat repo file: %w", err)
	}
	mtime := info.ModTime().UnixMilli()

	// 检查现有索引
	idxPath := filepath.Join(astsDir, indexDir, repoName+".idx")
	if _, err := os.Stat(idxPath); err == nil {
		// 读取现有索引的 mtime
		if oldData, err := os.ReadFile(idxPath); err == nil {
			var oldIdx SymbolIndex
			if json.Unmarshal(oldData, &oldIdx) == nil && oldIdx.Mtime == mtime {
				return nil // mtime 一致，无需更新
			}
		}
	}

	// 创建索引
	idx := SymbolIndex{
		Mtime: mtime,
		Data:  data,
	}

	// 写入 .tmp 再 rename
	idxPathTmp := idxPath + ".tmp"
	b, err := json.Marshal(idx)
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}
	if err := utils.MustWriteFile(idxPathTmp, b); err != nil {
		return fmt.Errorf("write index: %w", err)
	}
	if err := os.Rename(idxPathTmp, idxPath); err != nil {
		return fmt.Errorf("rename index: %w", err)
	}
	return nil
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

			// 获取所有 mod keys（只遍历 keys）
			modKeys, err := getModuleKeys(data)
			if err != nil {
				return err
			}

			var files = make(map[string]map[string][]string)
			var indexData = make(map[string][]NameMatch)

			// 遍历所有模块
			for _, modPath := range modKeys {
				// 跳过外部模块
				isExtVal, _ := sonic.Get(data, "Modules", modPath, "IsExternal")
				if isExt, _ := isExtVal.Bool(); isExt {
					continue
				}

				// 获取所有 package keys（只遍历 keys）
				pkgKeys, err := getPackageKeys(data, modPath)
				if err != nil {
					continue
				}

				// 遍历所有包
				for _, pkgPath := range pkgKeys {
					// 提取 Functions: 只读取 Name + File（极致按需）
					if results, err := iterSymbolNameFile(data, modPath, pkgPath, "Functions"); err == nil {
						for _, r := range results {
							name, file := r[0], r[1]
							if files[file] == nil {
								files[file] = map[string][]string{
									"FUNC": {},
									"TYPE": {},
									"VAR":  {},
								}
							}
							files[file]["FUNC"] = append(files[file]["FUNC"], name)
							indexData[name] = append(indexData[name], NameMatch{File: file, Type: "FUNC"})
						}
					}

					// 提取 Types
					if results, err := iterSymbolNameFile(data, modPath, pkgPath, "Types"); err == nil {
						for _, r := range results {
							name, file := r[0], r[1]
							if files[file] == nil {
								files[file] = map[string][]string{
									"FUNC": {},
									"TYPE": {},
									"VAR":  {},
								}
							}
							files[file]["TYPE"] = append(files[file]["TYPE"], name)
							indexData[name] = append(indexData[name], NameMatch{File: file, Type: "TYPE"})
						}
					}

					// 提取 Vars
					if results, err := iterSymbolNameFile(data, modPath, pkgPath, "Vars"); err == nil {
						for _, r := range results {
							name, file := r[0], r[1]
							if files[file] == nil {
								files[file] = map[string][]string{
									"FUNC": {},
									"TYPE": {},
									"VAR":  {},
								}
							}
							files[file]["VAR"] = append(files[file]["VAR"], name)
							indexData[name] = append(indexData[name], NameMatch{File: file, Type: "VAR"})
						}
					}
				}
			}

			// 保存索引文件
			if err := saveSymbolIndex(astsDir, repoName, repoFile, indexData); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save index: %v\n", err)
			}

			// 过滤掉空的 TYPE 和 VAR
			for file, types := range files {
				if len(types["TYPE"]) == 0 {
					delete(types, "TYPE")
				}
				if len(types["VAR"]) == 0 {
					delete(types, "VAR")
				}
				// 如果 FUNC 也空，删除整个文件
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
