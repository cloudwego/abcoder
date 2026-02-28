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
			tools := tool.NewASTReadTools(tool.ASTReadToolsOptions{
				RepoASTsDir: astsDir,
				DisableWatch: true,
			})
			resp, err := tools.TreeRepo(context.Background(), tool.TreeRepoReq{RepoName: repoName})
			if err != nil {
				return fmt.Errorf("failed to tree repo: %w", err)
			}

			b, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Fprintf(os.Stdout, "%s\n", b)
			return nil
		},
	}
}
