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
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// getASTsDir returns the ASTs directory path from command flags or default location.
func getASTsDir(cmd *cobra.Command) (string, error) {
	astsDir, err := cmd.Flags().GetString("asts-dir")
	if err != nil {
		return "", err
	}
	if astsDir == "" {
		astsDir = filepath.Join(os.Getenv("HOME"), ".asts")
	}
	if _, err := os.Stat(astsDir); os.IsNotExist(err) {
		return "", fmt.Errorf("ASTs directory does not exist: %s", astsDir)
	}
	return astsDir, nil
}
