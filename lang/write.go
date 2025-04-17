/**
 * Copyright 2025 ByteDance Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package lang

import (
	"context"
	"fmt"

	"github.com/cloudwego/abcoder/lang/golang/writer"
	"github.com/cloudwego/abcoder/lang/uniast"
)

// Write writes the AST to the output directory.
type WriteOptions struct {
	// OutputDir is the output directory.
	OutputDir string
	// Compiler path
	Compiler string
}

// Write writes the AST to the output directory.
func Write(ctx context.Context, repo *uniast.Repository, args WriteOptions) error {
	for mpath, m := range repo.Modules {
		if m.IsExternal() {
			continue
		}
		var w uniast.Writer
		switch m.Language {
		case uniast.Golang:
			w = writer.NewWriter(writer.Options{CompilerPath: args.Compiler})
		default:
			return fmt.Errorf("unsupported language: %s", m.Language)
		}
		if err := w.WriteModule(repo, mpath, args.OutputDir); err != nil {
			return err
		}
	}
	return nil
}
