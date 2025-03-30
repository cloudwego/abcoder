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

package rust

import (
	"os"
	"path/filepath"

	"github.com/cloudwego/abcoder/src/lang/log"
	"github.com/cloudwego/abcoder/src/lang/uniast"
)

type RustModulePatcher struct {
	Root string
}

func (p *RustModulePatcher) Patch(ast *uniast.Module) {
	filepath.Walk(p.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if filepath.Base(path) == "target" || filepath.Base(path) == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		relpath, err := filepath.Rel(p.Root, path)
		if err != nil {
			log.Error("get relative path of %s failed: %v", path, err)
			return nil
		}
		file := ast.Files[relpath]
		if file == nil {
			file = uniast.NewFile(path)
			ast.Files[relpath] = file
		}
		if filepath.Ext(path) != ".rs" {
			return nil
		}
		// 解析use语句
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		uses, err := ParseUseStatements(string(content))
		if err != nil {
			log.Error("parse file %s use statements failed: %v", path, err)
			return nil
		}
		file.Imports = uses
		return nil
	})
}
