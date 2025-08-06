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

package patch

import (
	"os"
	"os/exec"
	"testing"

	"github.com/cloudwego/abcoder/lang/testutils"
	"github.com/cloudwego/abcoder/lang/uniast"
)

// Expected to fail because the AST file contains local paths.
func TestPatcher(t *testing.T) {
	// Load AST
	t.Logf("Loading AST file for localsession...")
	astFile := testutils.GetTestAstFile("localsession")
	repo, err := uniast.LoadRepo(astFile)
	if err != nil {
		t.Fatalf("failed to load repo: %v", err)
	}

	// Load repo from git
	tmproot := testutils.MakeTmpTestdir(true)
	repoURL := "github.com/cloudwego/localsession"
	repoDir := tmproot + "/localsession"
	t.Logf("Cloning repo %s to %s...", repoURL, repoDir)
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", "main", "https://"+repoURL, repoDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git clone failed: %v", err)
	}

	// Create patcher with options
	patcher := NewPatcher(repo, Options{
		RepoDir:        repoDir,
		OutDir:         tmproot + "/localsession2",
		DefaultLanuage: uniast.Golang,
	})

	// Create a test patch
	testPatches := []Patch{
		{
			Id: uniast.Identity{ModPath: repoURL, PkgPath: repoURL + "/backup", Name: "DefaultOptions"},
			Codes: `func DefaultOptions() Options {
	ret := Options{
		Enable:         false,
		ManagerOptions: localsession.DefaultManagerOptions(),
		SonicConfig:    sonic.ConfigDefault,
	}
	return ret
}`,
			File: "backup/metainfo.go",
			Type: uniast.FUNC,
			AddedDeps: []uniast.Identity{
				{ModPath: "github.com/bytedance/sonic@v1.12.1", PkgPath: "github.com/bytedance/sonic", Name: "ConfigDefault"},
			},
		},
		{
			Id: uniast.Identity{ModPath: repoURL, PkgPath: repoURL + "/backup", Name: "DefaultOptions2"},
			Codes: `func DefaultOptions2() Options {
	ret := Options{
		Enable:         false,
		ManagerOptions: localsession.DefaultManagerOptions(),
	}
	return ret
}`,
			File: "backup/metainfo.go",
			Type: uniast.FUNC,
		},
		{
			Id: uniast.Identity{ModPath: repoURL, PkgPath: repoURL + "/backup", Name: "TestCase"},
			Codes: `type TestCase struct {
				Enable bool
			}`,
			File: "backup/abcoder_test.go",
			Type: uniast.TYPE,
		},
		{
			Id: uniast.Identity{ModPath: repoURL, PkgPath: repoURL + "/backup", Name: "TestFunc"},
			Codes: `
			func TestFunc(t *testing.T) {}`,
			File: "backup/abcoder_test.go",
			Type: uniast.FUNC,
			AddedDeps: []uniast.Identity{
				{ModPath: "", PkgPath: "testing", Name: "T"},
			},
		},
	}

	// Apply the patches
	for _, testPatch := range testPatches {
		if err := patcher.Patch(testPatch); err != nil {
			t.Fatalf("failed to patch: %v", err)
		}
	}

	// Flush changes
	if err := patcher.Flush(); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}

	// TODO: check patching work as expected
}
