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

package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// count files and total size with specific subfix in a directory recursively
func CountFiles(dir string, subfix string, skipdir string) (int, int) {
	count := 0
	size := 0
	skipdir, _ = filepath.Abs(filepath.Join(dir, skipdir))
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if strings.HasPrefix(path, skipdir) {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == subfix {
			count++
			size += int(info.Size())
		}
		return nil
	})
	return count, size
}

// find the first file with specific subfix in a directory recursively
func FirstFile(dir string, subfix string, skipdir string) string {
	var ret string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if strings.HasPrefix(path, skipdir) {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == subfix {
			ret = path
			return filepath.SkipDir
		}
		return nil
	})
	return ret
}
