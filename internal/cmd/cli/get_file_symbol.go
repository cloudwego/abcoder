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

	"github.com/spf13/cobra"
)

func newGetFileSymbolCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get_file_symbol <repo_name> <file_path> <name>",
		Short: "Get detailed symbol information",
		Long: `Get detailed information about a symbol including code, dependencies, and references.

Returns the symbol's code, type, line number, and relationship with other symbols.`,
		Example: `abcoder cli get_file_symbol myrepo src/main.go MyFunction`,
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			astsDir, err := getASTsDir(cmd)
			if err != nil {
				return err
			}

			repoName := args[0]
			filePath := args[1]
			symbolName := args[2]

			repoFile := findRepoFile(astsDir, repoName)
			if repoFile == "" {
				return fmt.Errorf("repo not found: %s", repoName)
			}

			// 加载 data（用于后续按需读取）
			data, err := loadRepoFileData(repoFile)
			if err != nil {
				return err
			}

			// 1. 定位 pkgPath（极致按需：只读取 File 字段验证）
			modPath, pkgPath, err := findPkgPathByFile(data, filePath)
			if err != nil {
				return fmt.Errorf("symbol '%s' not found in file '%s'", symbolName, filePath)
			}

			// 2. 读取 symbol 完整内容
			sym, err := getSymbolByFileFull(data, modPath, pkgPath, filePath, symbolName)
			if err != nil {
				return fmt.Errorf("symbol '%s' not found in file '%s'", symbolName, filePath)
			}

			// 找到 symbol，构建返回结构
			nodeType := "FUNC"
			if t, ok := sym["node_type"].(string); ok {
				nodeType = t
			}

			signature := ""
			if s, ok := sym["Signature"].(string); ok {
				signature = s
			}
			content := ""
			if c, ok := sym["Content"].(string); ok {
				content = c
			}

			// 3. 按需读取 Graph References
			refs, err := getSymbolReferences(data, modPath, pkgPath, symbolName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "DEBUG: getSymbolReferences error: %v\n", err)
				return err
			}

			// 按 Kind 分类
			var deps, refsOnly []map[string]string
			for _, r := range refs {
				if r["kind"] == "Dependency" {
					deps = append(deps, r)
				} else {
					refsOnly = append(refsOnly, r)
				}
			}

			node := map[string]interface{}{
				"name":         symbolName,
				"type":         nodeType,
				"file":         filePath,
				"line":         int(sym["Line"].(float64)),
				"codes":        content,
				"signature":    signature,
				"dependencies": deps,
				"references":   refsOnly,
			}

			resp := map[string]interface{}{
				"node": node,
			}

			b, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Fprintf(os.Stdout, "%s\n", b)
			return nil
		},
	}
}
