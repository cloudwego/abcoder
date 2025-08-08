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
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
)

// Hack to get the project root directory from go tests.
// Go tests start from the directory where the test file is located,
// causing the relative path to testdata files to be unstable.
func GetTestDataRoot() string {
	_, currentFilePath, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get caller information")
	}
	projectRoot := filepath.Dir(currentFilePath)
	for {
		goModPath := filepath.Join(projectRoot, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			break
		}
		parentDir := filepath.Dir(projectRoot)
		if parentDir == projectRoot {
			panic("could not find project root (go.mod not found)")
		}
		projectRoot = parentDir
	}
	rootDir, err := filepath.Abs(filepath.Join(projectRoot, "testdata"))
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

func GitCloneFast(repoURL, dir, branch string) (string, error) {
	rootDir := GetTestDataRoot()
	repoDir := filepath.Join(rootDir, "repos", dir)
	if _, err := os.Stat(repoDir); !os.IsNotExist(err) {
		cmd := exec.Command("git", "-C", repoDir, "status")
		if err := cmd.Run(); err == nil {
			return repoDir, nil
		} else {
			return "", fmt.Errorf("bad existing repo %s: %w", repoDir, err)
		}
	}
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create repo directory: %w", err)
	}
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", branch, "https://"+repoURL, repoDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git clone failed: %w", err)
	}
	return repoDir, nil
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

func TestPath(name, lang string) string {
	testcases := ListTests(lang)
	for _, test := range testcases {
		ptn := fmt.Sprintf("^\\d+_%s$", regexp.QuoteMeta(name))
		matched, _ := regexp.MatchString(ptn, filepath.Base(test))
		if matched {
			return test
		}
	}
	panic(fmt.Sprintf("Test case %s not found in language %s, available: %v", name, lang, testcases))
}

func FirstTest(lang string) string {
	return ListTests(lang)[0]
}
