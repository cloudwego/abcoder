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
	"github.com/cloudwego/abcoder/llm/tool"
	"github.com/spf13/cobra"
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

			// 尝试从 .repo_index.json 读取映射
			indexFile := filepath.Join(astsDir, ".repo_index.json")
			var repoNames []string

			if data, err := os.ReadFile(indexFile); err == nil {
				// 解析 repo_index.json
				var index struct {
					Mappings map[string]string `json:"mappings"`
				}
				if err := json.Unmarshal(data, &index); err == nil && index.Mappings != nil {
					for name := range index.Mappings {
						repoNames = append(repoNames, name)
					}
				}
			}

			// 如果没有从 index 读取到，回退到扫描 JSON 文件，使用 sonic 快速读取 id
			if len(repoNames) == 0 {
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
								repoNames = append(repoNames, id)
							}
						}
					}
				}
			}

			resp := tool.ListReposResp{RepoNames: repoNames}
			b, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Fprintf(os.Stdout, "%s\n", b)
			return nil
		},
	}
}
