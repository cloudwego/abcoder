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

The repositories are loaded from .repo_index.json or *.json files in the --asts-dir directory.`,
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

			// 尝试从 .repo_index.json 读取映射
			indexFile := filepath.Join(astsDir, ".repo_index.json")
			var repoNames []string
			var currentRepo string

			if data, err := os.ReadFile(indexFile); err == nil {
				// 用 sonic 解析 mappings
				mappingsVal, err := sonic.Get(data, "mappings")
				if err == nil {
					mappings, err := mappingsVal.Map()
					if err == nil {
						for name, v := range mappings {
							repoNames = append(repoNames, name)
							// 检查当前目录是否匹配 (mappings value 是文件名，需要检查对应的 json 文件)
							if pathMatchesCwd(astsDir, v.(string), cwd) {
								currentRepo = name
							}
						}
					}
				}
			}

			// 扫描 JSON 文件，使用 sonic 快速读取
			repoNamesMap := make(map[string]struct{})
			files, err := filepath.Glob(filepath.Join(astsDir, "*.json"))
			if err != nil {
				return err
			}
			for _, f := range files {
				// 跳过 _repo_index.json
				if strings.HasSuffix(f, "_repo_index.json") || strings.HasSuffix(f, ".repo_index.json") {
					continue
				}
				// 使用 sonic 快速读取 id 字段，避免加载整个 JSON
				if data, err := os.ReadFile(f); err == nil {
					val, err := sonic.Get(data, "id")
					if err == nil {
						id, err := val.String()
						if err == nil && id != "" {
							repoNamesMap[id] = struct{}{}
						}
					}
					// 尝试读取 Path 字段，检查是否匹配当前目录
					if currentRepo == "" {
						val, err := sonic.Get(data, "Path")
						if err == nil {
							path, err := val.String()
							if err == nil && path == cwd {
								// 从 id 字段获取名称
								val, err := sonic.Get(data, "id")
								if err == nil {
									id, err := val.String()
									if err == nil && id != "" {
										currentRepo = id
									}
								}
							}
						}
					}
				}
			}
			repoNames = maps.Keys(repoNamesMap)

			type ListReposOutput struct {
				RepoNames    []string `json:"repo_names"`
				CurrentRepo  string   `json:"current_repo,omitempty"`
			}
			resp := ListReposOutput{
				RepoNames:   repoNames,
				CurrentRepo: currentRepo,
			}
			b, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Fprintf(os.Stdout, "%s\n", b)
			return nil
		},
	}
}
