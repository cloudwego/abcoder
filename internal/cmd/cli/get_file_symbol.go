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
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/cloudwego/abcoder/llm/tool"
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
			name := args[2]
			tools := tool.NewASTReadTools(tool.ASTReadToolsOptions{
				RepoASTsDir: astsDir,
				DisableWatch: true,
			})
			resp, err := tools.GetFileSymbol(context.Background(), tool.GetFileSymbolReq{
				RepoName: repoName,
				FilePath: filePath,
				Name:     name,
			})
			if err != nil {
				return fmt.Errorf("failed to get file symbol: %w", err)
			}

			b, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Fprintf(os.Stdout, "%s\n", b)
			return nil
		},
	}
}
