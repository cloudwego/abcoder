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
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/cloudwego/abcoder/lang/collect"
	"github.com/cloudwego/abcoder/lang/testutils"
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/pkg/errors"
)

func defaultOptions(lang string) ParseOptions {
	lsp := map[string]string{
		"rust": "rust-analyzer",
	}
	return ParseOptions{
		LSP:     lsp[lang],
		Verbose: false,
		CollectOption: collect.CollectOption{
			Language:           uniast.Language(lang),
			LoadExternalSymbol: false,
			NeedStdSymbol:      false,
			NoNeedComment:      true,
			NotNeedTest:        true,
			Excludes:           []string{},
		},
		RepoID:     "",
		LspOptions: make(map[string]string),
	}
}

func readFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}
	defer file.Close()
	return io.ReadAll(file)
}

// Compare func.Content with file[startOffset:endOffset].
func matchWithImplhead(lang, expected string, fn *uniast.Function) bool {
	actual := fn.Content
	// compare by verbatim for non-methods
	if !fn.IsMethod {
		return actual == expected
	}
	// compare with implhead for methods
	ptnFrags := map[string][]string{
		"rust": {
			`impl[^{]*\{`,
			`(type[^;]*;)*`,
			regexp.QuoteMeta(expected),
			`\}`,
		},
	}[lang]
	ptn := strings.Join(ptnFrags, `\s*`)
	reptn := regexp.MustCompile(ptn)
	return reptn.MatchString(actual)
}

func checkFunctionConsistency(t *testing.T, lang string, fn *uniast.Function, workspace string) {
	println("checking fn ", fn.Name, "in", workspace)
	file := filepath.Join(workspace, fn.File)
	fileContents, err := readFile(file)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", file, err)
	}
	expectedContent := string(fileContents[fn.StartOffset:fn.EndOffset])
	if !matchWithImplhead(lang, expectedContent, fn) {
		t.Fatalf("Function %s content mismatch: expected\n%q\ngot\n%q\nfn: %#v\n", fn.Name, expectedContent, fn.Content, fn)
	}
}

func checkTypeConsistency(t *testing.T, lang string, ty *uniast.Type, workspace string) {
	println("checking ty ", ty.Name, "in", workspace)
	file := filepath.Join(workspace, ty.File)
	fileContents, err := readFile(file)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", file, err)
	}
	expectedContent := string(fileContents[ty.StartOffset:ty.EndOffset])
	if expectedContent != ty.Content {
		t.Fatalf("Type %s content mismatch: expected\n%q\ngot\n%q\nty: %#v\n", ty.Name, expectedContent, ty.Content, ty)
	}
}

func checkVarConsistency(t *testing.T, lang string, va *uniast.Var, workspace string) {
	println("checking var ", va.Name, "in", workspace)
	file := filepath.Join(workspace, va.File)
	fileContents, err := readFile(file)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", file, err)
	}
	expectedContent := string(fileContents[va.StartOffset:va.EndOffset])
	if expectedContent != va.Content {
		t.Fatalf("Var %s content mismatch: expected\n%q\ngot\n%q\nva: %#v\n", va.Name, expectedContent, va.Content, va)
	}
}

// Checks for all Functions/Types/Vars:
//
//	their Content, Start/EndOffset, Line fields are consistent
func checkRepoConsistency(t *testing.T, lang string, repo *uniast.Repository, workspace string) {
	t.Helper()
	for _, mod := range repo.Modules {
		for _, pkg := range mod.Packages {
			for _, fn := range pkg.Functions {
				checkFunctionConsistency(t, lang, fn, workspace)
			}
			for _, ty := range pkg.Types {
				checkTypeConsistency(t, lang, ty, workspace)
			}
			for _, va := range pkg.Vars {
				checkVarConsistency(t, lang, va, workspace)
			}
		}
	}
}

func TestParser_NodeFieldsConsistency(t *testing.T) {
	checked_languages := []string{"rust", "go"}
	for _, lang := range checked_languages {
		testCase := testutils.FirstTest(lang)
		repobytes, err := Parse(context.Background(), testCase, defaultOptions(lang))
		if err != nil {
			t.Fatalf("Parse() failed: %v", err)
		}
		var repo uniast.Repository
		if err := json.Unmarshal(repobytes, &repo); err != nil {
			t.Fatalf("Unmarshal() failed: %v", err)
		}
		checkRepoConsistency(t, lang, &repo, testCase)
	}
}
