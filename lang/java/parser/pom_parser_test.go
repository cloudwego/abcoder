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
	"testing"
)

func TestParseMavenProject(t *testing.T) {
	projectRootPath := "../../../testdata/java/3_java_pom"
	rootPomPath := filepath.Join(projectRootPath, "pom.xml")

	rootModule, err := ParseMavenProject(rootPomPath)
	if err != nil {
		t.Fatalf("Error parsing root project: %v", err)
	}

	if rootModule.ArtifactID != "my-app" {
		t.Errorf("Expected artifactId to be 'my-app', but got '%s'", rootModule.ArtifactID)
	}

	if len(rootModule.SubModules) != 1 {
		t.Fatalf("Expected 1 submodule, but got %d", len(rootModule.SubModules))
	}

	subModule := rootModule.SubModules[0]
	if subModule.ArtifactID != "my-app-sub" {
		t.Errorf("Expected submodule artifactId to be 'my-app-sub', but got '%s'", subModule.ArtifactID)
	}

	// Print the tree to visually verify the structure
	PrintProjectTree(rootModule, "")
}
