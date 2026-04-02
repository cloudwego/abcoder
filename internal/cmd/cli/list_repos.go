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
	"sort"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

func newListReposCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list_repos",
		Short: "List available repositories",
		Long: `List all available repositories in the AST directory.

The repositories are loaded from *.json files in the --asts-dir directory.`,
		Example: `abcoder cli list-repos`,
		RunE: func(cmd *cobra.Command, args []string) error {
			astsDir, err := getASTsDir(cmd)
			if err != nil {
				return err
			}

			// 获取当前工作目录
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] cwd: %s\n", cwd)
			}

			// 扫描所有 JSON 文件，读取 id 和 path
			repoNamesMap := make(map[string]struct{})
			var currentRepos []string
			type pathItem struct {
				id   string
				path string
			}
			var pathItems []pathItem

			files, err := filepath.Glob(filepath.Join(astsDir, "*.json"))
			if err != nil {
				return err
			}
			if verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] found %d json files\n", len(files))
			}
			for _, f := range files {
				// 跳过 _repo_index.json
				if strings.HasSuffix(f, "_repo_index.json") || strings.HasSuffix(f, ".repo_index.json") {
					continue
				}
				// 使用 sonic 快速读取 id 和 path 字段
				if data, err := os.ReadFile(f); err == nil {
					// 读取 id
					idVal, err := sonic.Get(data, "id")
					if err != nil {
						continue
					}
					id, err := idVal.String()
					if err != nil || id == "" {
						continue
					}
					repoNamesMap[id] = struct{}{}

					// 读取 path
					pathVal, err := sonic.Get(data, "Path")
					if err == nil {
						path, err := pathVal.String()
						if err == nil && path != "" {
							pathItems = append(pathItems, pathItem{id: id, path: path})
						}
					}
				}
			}

			// 按 path 排序，用于前缀匹配时提前退出
			sort.Slice(pathItems, func(i, j int) bool {
				return pathItems[i].path < pathItems[j].path
			})

			// 查找 cwd 前缀匹配的 repo
			for _, item := range pathItems {
				if verbose {
					fmt.Fprintf(os.Stderr, "[VERBOSE] checking: id=%s, path=%s\n", item.id, item.path)
				}
				// 如果 path 比 cwd 短，不可能匹配，提前退出
				if len(item.path) < len(cwd) {
					if verbose {
						fmt.Fprintf(os.Stderr, "[VERBOSE] early exit: path shorter than cwd\n")
					}
					continue
				}
				if strings.HasPrefix(item.path, cwd) {
					currentRepos = append(currentRepos, item.id)
					if verbose {
						fmt.Fprintf(os.Stderr, "[VERBOSE] MATCH: id=%s, path=%s\n", item.id, item.path)
					}
				}
			}

			repoNames := maps.Keys(repoNamesMap)

			type ListReposOutput struct {
				RepoNames    []string `json:"repo_names"`
				CurrentRepos []string `json:"current_repo,omitempty"`
			}
			resp := ListReposOutput{
				RepoNames:    repoNames,
				CurrentRepos: currentRepos,
			}
			b, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Fprintf(os.Stdout, "%s\n", b)
			return nil
		},
	}
}
