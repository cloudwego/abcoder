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
)

func newTreeRepoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tree_repo <repo_name>",
		Short: "Get file tree of a repository",
		Long: `Get the file tree structure of a repository.

Returns a map of directories to file lists.`,
		Example: `abcoder cli tree_repo myrepo`,
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

			data, err := loadRepoFileData(repoFile)
			if err != nil {
				return err
			}

			// 获取所有 mod keys
			modKeys, err := getModuleKeys(data)
			if err != nil {
				return err
			}

			// 收集所有文件，按目录聚合
			files := make(map[string][]string)
			for _, modPath := range modKeys {
				// 跳过外部模块（通过 IsExternal 字段判断）
				isExtVal, _ := sonic.Get(data, "Modules", modPath, "IsExternal")
				if isExt, _ := isExtVal.Bool(); isExt {
					continue
				}

				// 只遍历 Files 的 keys（极致按需：不加载 value）
				filePaths, err := iterModFiles(data, modPath)
				if err != nil {
					continue
				}

				for _, path := range filePaths {
					// 过滤掉非当前仓库的文件
					if strings.HasPrefix(path, "..") {
						continue
					}

					// 获取目录路径
					dir := filepath.Dir(path)
					if dir == "." {
						dir = "./"
					}
					// 添加 '/' 后缀
					if dir != "" && dir != "./" && !strings.HasSuffix(dir, "/") {
						dir = dir + "/"
					}

					// 获取文件名
					name := filepath.Base(path)
					files[dir] = append(files[dir], name)
				}
			}

			// 排序
			for dir := range files {
				sort.Strings(files[dir])
			}

			resp := map[string]interface{}{
				"files": files,
			}
			b, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Fprintf(os.Stdout, "%s\n", b)
			return nil
		},
	}
}
