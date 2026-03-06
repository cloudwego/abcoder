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

package parser

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testdataDir() string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "..", "..", "..", "testdata", "java")
}

func TestParseGradleProject(t *testing.T) {
	rootDir := filepath.Join(testdataDir(), "5_gradle_project")
	mod, err := ParseGradleProject(rootDir)
	require.NoError(t, err)
	require.NotNil(t, mod)

	assert.Equal(t, "my-gradle-app", mod.ArtifactID)
	assert.Equal(t, "com.example", mod.GroupID)
	assert.Equal(t, "1.0.0", mod.Version)
	assert.Equal(t, "com.example:my-gradle-app:1.0.0", mod.Coordinates)
	assert.Equal(t, filepath.Join(rootDir, "src", "main", "java"), mod.SourcePath)
	assert.Equal(t, filepath.Join(rootDir, "src", "test", "java"), mod.TestSourcePath)
	assert.Equal(t, filepath.Join(rootDir, "build"), mod.TargetPath)

	// Should have 2 submodules: app and core
	require.Len(t, mod.SubModules, 2)

	// SubModules are sorted by include order (alphabetical after sort)
	appMod := mod.SubModules[0]
	assert.Equal(t, "app", appMod.ArtifactID)
	assert.Equal(t, "com.example", appMod.GroupID)
	assert.Equal(t, "1.0.0", appMod.Version)
	assert.Equal(t, filepath.Join(rootDir, "app", "src", "main", "java"), appMod.SourcePath)

	coreMod := mod.SubModules[1]
	assert.Equal(t, "core", coreMod.ArtifactID)
	assert.Equal(t, "com.example", coreMod.GroupID)
	assert.Equal(t, "1.0.0", coreMod.Version)
	assert.Equal(t, filepath.Join(rootDir, "core", "src", "main", "java"), coreMod.SourcePath)
}

func TestParseGradleProject_ModulePaths(t *testing.T) {
	rootDir := filepath.Join(testdataDir(), "5_gradle_project")
	mod, err := ParseGradleProject(rootDir)
	require.NoError(t, err)

	paths := GetModulePaths(mod)
	assert.NotEmpty(t, paths)

	modMap := GetModuleStructMap(mod)
	assert.Len(t, modMap, 3) // root + 2 submodules
}

func TestParseGradleProject_NotFound(t *testing.T) {
	mod, err := ParseGradleProject("/nonexistent/path")
	assert.Error(t, err)
	assert.Nil(t, mod)
}

func TestExtractSubprojects(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "single include",
			content:  `include ':app'`,
			expected: []string{":app"},
		},
		{
			name:     "multiple includes",
			content:  "include ':app', ':core'",
			expected: []string{":app", ":core"},
		},
		{
			name: "separate include lines",
			content: `include ':app'
include ':core'`,
			expected: []string{":app", ":core"},
		},
		{
			name:     "kotlin style parentheses",
			content:  `include(":app", ":core")`,
			expected: []string{":app", ":core"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractSubprojects(tc.content)
			assert.Equal(t, tc.expected, result)
		})
	}
}
