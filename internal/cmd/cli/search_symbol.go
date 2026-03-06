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
	"regexp"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/spf13/cobra"
)

// SearchResult 搜索结果
type SearchResult struct {
	RepoName string                    `json:"repo_name"`
	Query    string                    `json:"query"`
	Results  map[string]map[string][]string `json:"results"` // file -> type -> names
}

const indexDir = ".index"

type SymbolIndex struct {
	Mtime int64                  `json:"mtime"`
	Data  map[string][]NameMatch `json:"data"` // name -> []NameMatch
}

type NameMatch struct {
	File string `json:"file"`
	Type string `json:"type"`
}

// loadSymbolIndex 加载符号索引
func loadSymbolIndex(astsDir, repoName, repoFile string) (*SymbolIndex, error) {
	idxPath := filepath.Join(astsDir, indexDir, repoName+".idx")

	// 检查索引文件是否存在
	if _, err := os.Stat(idxPath); err != nil {
		// 索引不存在，返回 nil，让调用者知道需要生成
		return nil, nil
	}

	// 读取索引
	data, err := os.ReadFile(idxPath)
	if err != nil {
		return nil, err
	}

	var idx SymbolIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}

	// 检查 mtime 是否一致
	repoInfo, err := os.Stat(repoFile)
	if err != nil {
		return nil, err
	}
	if idx.Mtime != repoInfo.ModTime().UnixMilli() {
		// mtime 不一致，返回 nil，让调用者知道需要重新生成
		return nil, nil
	}

	return &idx, nil
}

// hasRegexMetaChars 检查是否包含正则元字符
func hasRegexMetaChars(s string) bool {
	return strings.ContainsAny(s, ".+?{}[]|\\^$")
}

// matchName 检查 name 是否匹配 query
// 支持: ripgrep 正则语法
// - 普通字符串: 包含匹配 (*query*)
// - 含 * : 通配符匹配 (转为 .*)
// - 含其他正则元字符: 正则包含匹配 (.*query.*)
func matchName(name, query string) bool {
	// 如果包含 * (通配符)
	if strings.Contains(query, "*") {
		// 转为 .* 并做包含匹配
		pattern := strings.ReplaceAll(query, "*", ".*")
		pattern = ".*" + pattern + ".*"
		matched, _ := regexp.MatchString(pattern, name)
		return matched
	}

	// 如果包含正则元字符，使用正则包含匹配
	if hasRegexMetaChars(query) {
		pattern := ".*" + query + ".*"
		matched, err := regexp.MatchString(pattern, name)
		if err != nil {
			// 回退到子串包含匹配
			return strings.Contains(name, query)
		}
		return matched
	}

	// 普通字符串：包含匹配 (*query*)
	pattern := ".*" + query + ".*"
	matched, _ := regexp.MatchString(pattern, name)
	return matched
}

func newSearchSymbolCmd() *cobra.Command {
	var pathFilter string

	cmd := &cobra.Command{
		Use:   "search_symbol <repo_name> <query>",
		Short: "Search symbols by name",
		Long: `Search symbols in a repository by name pattern.
Supports substring match, prefix match (query*), suffix match (*query), and wildcard (*query*).`,
		Example: `abcoder cli search_symbol myrepo GetUser
abcoder cli search_symbol myrepo "*User"
abcoder cli search_symbol myrepo "Get*"
abcoder cli search_symbol myrepo "Graph" --path "src/main/java/com/uniast/parser"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			astsDir, err := getASTsDir(cmd)
			if err != nil {
				return err
			}

			repoName := args[0]
			query := args[1]

			repoFile := findRepoFile(astsDir, repoName)
			if repoFile == "" {
				return fmt.Errorf("repo not found: %s", repoName)
			}

			// 读取 JSON 文件
			data, err := os.ReadFile(repoFile)
			if err != nil {
				return fmt.Errorf("failed to read repo file: %w", err)
			}

			var results = make(map[string]map[string][]string)

			// 方式1: 使用 NameToLocations（新增字段，O(1)）
			nameToLocsVal, err := sonic.Get(data, "NameToLocations")
			if err == nil && nameToLocsVal.Exists() {
				nameToLocs, err := nameToLocsVal.Map()
				if err == nil {
					if verbose {
						fmt.Fprintf(os.Stderr, "[VERBOSE] using NameToLocations (new)\n")
					}
					for name := range nameToLocs {
						if matchName(name, query) {
							// 支持两种格式：
							// 1. 数组格式: {"GetUser": ["src/user.rs"]}
							// 2. 对象格式: {"GetUser": {"Files": ["src/user.rs"]}}
							locVal := nameToLocsVal.Get(name)
							var files []interface{}
							filesVal := locVal.Get("Files")
							if filesVal.Exists() {
								files, _ = filesVal.Array()
							} else {
								// 尝试数组格式
								files, _ = locVal.Array()
							}
							if len(files) > 0 {
								for _, f := range files {
									fileStr, _ := f.(string)
									// path 前缀过滤
									if pathFilter != "" && !strings.HasPrefix(fileStr, pathFilter) {
										continue
									}
									if results[fileStr] == nil {
										results[fileStr] = map[string][]string{
											"FUNC": {},
											"TYPE": {},
											"VAR":  {},
										}
									}
									results[fileStr]["FUNC"] = append(results[fileStr]["FUNC"], name)
								}
							}
						}
					}

					// 无论是否有结果都直接返回
					output := SearchResult{
						RepoName: repoName,
						Query:    query,
						Results:  results,
					}
					b, _ := json.MarshalIndent(output, "", "  ")
					fmt.Fprintf(os.Stdout, "%s\n", b)
					return nil
				}
			}

			// 方式2: 没有 NameToLocations，构建并写回 JSON
			if verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] building NameToLocations\n")
			}

			// 使用公共函数构建
			nameToTypeFiles, err := buildNameToLocations(data, pathFilter)
			if err != nil {
				return err
			}

			// 写回 JSON（使用完整 path 构建，否则后续搜索会丢失数据）
			fullNameToTypeFiles, err := buildNameToLocations(data, "")
			if err == nil {
				// 转换为 name -> []file 格式
				fullNameToLocsMap := make(map[string][]string)
				for name, typeFiles := range fullNameToTypeFiles {
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
					fullNameToLocsMap[name] = fileList
				}
				if err := saveNameToLocations(repoFile, fullNameToLocsMap); err != nil {
					if verbose {
						fmt.Fprintf(os.Stderr, "Warning: failed to save NameToLocations: %v\n", err)
					}
				} else if verbose {
					fmt.Fprintf(os.Stderr, "[VERBOSE] saved NameToLocations to %s\n", repoFile)
				}
			}

			// 转换为输出格式（全部归为 FUNC，因为 JSON 里没存 type）
			for name, typeFiles := range nameToTypeFiles {
				for _, fileSet := range typeFiles {
					for file := range fileSet {
						if results[file] == nil {
							results[file] = map[string][]string{
								"FUNC": {},
								"TYPE": {},
								"VAR":  {},
							}
						}
						results[file]["FUNC"] = append(results[file]["FUNC"], name)
					}
				}
			}

			output := SearchResult{
				RepoName: repoName,
				Query:    query,
				Results:  results,
			}

			b, _ := json.MarshalIndent(output, "", "  ")
			fmt.Fprintf(os.Stdout, "%s\n", b)
			return nil
		},
	}

	cmd.Flags().StringVar(&pathFilter, "path", "", "filter by file path prefix (e.g., src/main/java/com/uniast/parser)")

	return cmd
}
