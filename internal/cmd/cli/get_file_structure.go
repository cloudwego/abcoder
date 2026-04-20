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

func newGetFileStructureCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get_file_structure <repo_name> <file_path>",
		Short: "Get symbol names of a file",
		Long: `Get the symbol names and signatures of a file in the repository.

Returns a list of functions, types, and variables defined in the file.`,
		Example: `abcoder cli get_file_structure myrepo src/main.go`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			astsDir, err := getASTsDir(cmd)
			if err != nil {
				return err
			}

			repoName := args[0]
			filePath := args[1]

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
				return fmt.Errorf("file '%s' not found in repo", filePath)
			}

			// 2. 读取该文件所有 symbols
			syms, err := getFileSymbolsByFile(data, modPath, pkgPath, filePath)
			if err != nil || len(syms) == 0 {
				return fmt.Errorf("no symbols found in file '%s'", filePath)
			}

			type Node struct {
				Name      string `json:"name"`
				Line     int    `json:"line"`
				Signature string `json:"signature,omitempty"`
				TypeKind  string `json:"typeKind,omitempty"` // class, typedef, struct, enum, interface
			}

			var nodes []Node
			for _, sym := range syms {
				n := Node{
					Name: sym["Name"].(string),
					Line: int(sym["Line"].(float64)),
				}
				if sig, ok := sym["Signature"].(string); ok {
					n.Signature = sig
				}
				// 添加 TypeKind (class, typedef, struct, enum, interface)
				if tk, ok := sym["TypeKind"].(string); ok && tk != "" {
					n.TypeKind = tk
				}
				nodes = append(nodes, n)
			}

			resp := map[string]interface{}{
				"file_path": filePath,
				"mod_path":  modPath,
				"pkg_path":  pkgPath,
				"nodes":     nodes,
			}

			b, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Fprintf(os.Stdout, "%s\n", b)
			return nil
		},
	}
}
