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
	return &cobra.Command{
		Use:   "search_symbol <repo_name> <query>",
		Short: "Search symbols by name",
		Long: `Search symbols in a repository by name pattern.
Supports substring match, prefix match (query*), suffix match (*query), and wildcard (*query*).`,
		Example: `abcoder cli search_symbol myrepo GetUser
abcoder cli search_symbol myrepo "*User"
abcoder cli search_symbol myrepo "Get*"`,
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

			// 尝试加载索引
			idx, err := loadSymbolIndex(astsDir, repoName, repoFile)
			if err != nil {
				return fmt.Errorf("failed to load index: %w", err)
			}

			// 如果索引不存在或过期，需要从 JSON 构建
			if idx == nil {
				fmt.Fprintf(os.Stderr, "Index not found or outdated, rebuilding...\n")
				data, err := os.ReadFile(repoFile)
				if err != nil {
					return fmt.Errorf("failed to read repo file: %w", err)
				}

				modsVal, err := sonic.Get(data, "Modules")
				if err != nil {
					return fmt.Errorf("failed to get modules: %w", err)
				}

				mods, err := modsVal.Map()
				if err != nil {
					return fmt.Errorf("failed to parse modules: %w", err)
				}

				indexData := make(map[string][]NameMatch)

				for _, modVal := range mods {
					mod, ok := modVal.(map[string]interface{})
					if !ok {
						continue
					}

					pkgs, ok := mod["Packages"].(map[string]interface{})
					if !ok {
						continue
					}

					for _, pkgVal := range pkgs {
						pkg, ok := pkgVal.(map[string]interface{})
						if !ok {
							continue
						}

						// Functions
						if fns, ok := pkg["Functions"].(map[string]interface{}); ok {
							for _, fnVal := range fns {
								fn, ok := fnVal.(map[string]interface{})
								if !ok {
									continue
								}
								name := fn["Name"].(string)
								file := fn["File"].(string)
								indexData[name] = append(indexData[name], NameMatch{File: file, Type: "FUNC"})
							}
						}

						// Types
						if types, ok := pkg["Types"].(map[string]interface{}); ok {
							for _, typeVal := range types {
								t, ok := typeVal.(map[string]interface{})
								if !ok {
									continue
								}
								name := t["Name"].(string)
								file := t["File"].(string)
								indexData[name] = append(indexData[name], NameMatch{File: file, Type: "TYPE"})
							}
						}

						// Vars
						if vars, ok := pkg["Vars"].(map[string]interface{}); ok {
							for _, varVal := range vars {
								v, ok := varVal.(map[string]interface{})
								if !ok {
									continue
								}
								name := v["Name"].(string)
								file := v["File"].(string)
								indexData[name] = append(indexData[name], NameMatch{File: file, Type: "VAR"})
							}
						}
					}
				}

				// 保存索引
				if err := saveSymbolIndex(astsDir, repoName, repoFile, indexData); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to save index: %v\n", err)
				}

				idx = &SymbolIndex{
					Mtime: 0, // 临时使用
					Data:  indexData,
				}
			}

			// 搜索
			results := make(map[string]map[string][]string)
			for name, matches := range idx.Data {
				if matchName(name, query) {
					for _, m := range matches {
						if results[m.File] == nil {
							results[m.File] = map[string][]string{
								"FUNC": {},
								"TYPE": {},
								"VAR":  {},
							}
						}
						results[m.File][m.Type] = append(results[m.File][m.Type], name)
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
}
