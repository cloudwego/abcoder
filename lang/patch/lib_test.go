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
	"testing"

	"github.com/cloudwego/abcoder/lang/uniast"
)

var root = "../../../tmp"

func TestPatcher(t *testing.T) {
	// Load repository
	repo, err := uniast.LoadRepo(root + "/localsession.json")
	if err != nil {
		t.Errorf("failed to load repo: %v", err)
	}

	// Create patcher with options
	patcher := NewPatcher(repo, Options{
		RepoDir:        root + "/localsession",
		OutDir:         root + "/localsession2",
		DefaultLanuage: uniast.Golang,
	})

	// Create a test patch
	testPatches := []Patch{
		{
			Id: uniast.Identity{ModPath: "github.com/cloudwego/localsession", PkgPath: "github.com/cloudwego/localsession/backup", Name: "DefaultOptions"},
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
			Id: uniast.Identity{ModPath: "github.com/cloudwego/localsession", PkgPath: "github.com/cloudwego/localsession/backup", Name: "DefaultOptions2"},
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
			Id: uniast.Identity{ModPath: "github.com/cloudwego/localsession", PkgPath: "github.com/cloudwego/localsession/backup", Name: "TestCase"},
			Codes: `type TestCase struct {
				Enable bool
			}`,
			File: "backup/abcoder_test.go",
			Type: uniast.TYPE,
		},
		{
			Id: uniast.Identity{ModPath: "github.com/cloudwego/localsession", PkgPath: "github.com/cloudwego/localsession/backup", Name: "TestFunc"},
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
			t.Errorf("failed to patch: %v", err)
		}
	}

	// Flush changes
	if err := patcher.Flush(); err != nil {
		t.Errorf("failed to flush: %v", err)
	}
}
