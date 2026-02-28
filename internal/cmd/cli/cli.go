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
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,.
// See the License either express or implied for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"github.com/spf13/cobra"
)

// NewCliCmd returns the parent command for CLI operations.
func NewCliCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cli",
		Short: "CLI commands for AST analysis",
		Long: `CLI commands for directly querying AST data without MCP protocol.

These commands provide direct access to repository, file, and symbol information.`,
		Example: `abcoder cli list-repos`,
	}

	// Add subcommands
	cmd.AddCommand(newListReposCmd())
	cmd.AddCommand(newTreeRepoCmd())
	cmd.AddCommand(newGetFileStructureCmd())
	cmd.AddCommand(newGetFileSymbolCmd())

	return cmd
}
