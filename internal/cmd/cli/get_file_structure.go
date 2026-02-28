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
			tools := tool.NewASTReadTools(tool.ASTReadToolsOptions{
				RepoASTsDir: astsDir,
				DisableWatch: true,
			})
			resp, err := tools.GetFileStructure(context.Background(), tool.GetFileStructReq{
				RepoName: repoName,
				FilePath: filePath,
			})
			if err != nil {
				return fmt.Errorf("failed to get file structure: %w", err)
			}

			b, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Fprintf(os.Stdout, "%s\n", b)
			return nil
		},
	}
}
