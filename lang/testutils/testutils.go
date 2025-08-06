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

package testutils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
)

func GetTestDataRoot() string {
	rootDir, err := filepath.Abs("../../testdata")
	if err != nil {
		panic("Failed to get absolute path of testdata: " + err.Error())
	}
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		log.Fatalf("Test data directory does not exist: %s", rootDir)
	}
	return rootDir
}

func MakeTmpTestdir(reset bool) string {
	rootDir := GetTestDataRoot()
	tmpDir := filepath.Join(rootDir, "tmp")
	if reset {
		if err := os.RemoveAll(tmpDir); err != nil {
			panic("Failed to remove old tmp directory: " + err.Error())
		}
	}
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		if err := os.Mkdir(tmpDir, 0755); err != nil {
			panic("Failed to create tmp directory: " + err.Error())
		}
	}
	return tmpDir
}

func GetTestAstFile(name string) string {
	rootDir := GetTestDataRoot()
	astFile := filepath.Join(rootDir, "asts", name+".json")
	if _, err := os.Stat(astFile); os.IsNotExist(err) {
		panic(fmt.Sprintf("AST file does not exist: %s", astFile))
	}
	return astFile
}

func ListTests(lang string) []string {
	var testcases []string
	test_root := filepath.Join(GetTestDataRoot(), lang)
	entries, err := os.ReadDir(test_root)
	if err != nil || len(entries) == 0 {
		panic(fmt.Sprintf("Failed to read test directory %s: %v", test_root, err))
	}
	for _, entry := range entries {
		if entry.IsDir() {
			testcases = append(testcases, filepath.Join(test_root, entry.Name()))
		}
	}
	sort.Slice(testcases, func(i, j int) bool {
		return filepath.Base(testcases[i]) < filepath.Base(testcases[j])
	})
	return testcases
}

func FirstTest(lang string) string {
	return ListTests(lang)[0]
}
