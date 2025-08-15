package parser

import (
	"path/filepath"
	"testing"
)

func TestParseMavenProject(t *testing.T) {
	projectRootPath := "testdata/my-app"
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
