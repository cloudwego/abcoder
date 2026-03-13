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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	rootProjectNameRegex = regexp.MustCompile(`rootProject\.name\s*=\s*['"]([^'"]+)['"]`)
	includeRegex         = regexp.MustCompile(`include\s*\(?([^)\n]+)\)?`)
	includeItemRegex     = regexp.MustCompile(`['"]([^'"]+)['"]`)
	groupRegex           = regexp.MustCompile(`(?m)^\s*group\s*=\s*['"]([^'"]+)['"]`)
	versionRegex         = regexp.MustCompile(`(?m)^\s*version\s*=\s*['"]([^'"]+)['"]`)
	gradlePropRegex      = regexp.MustCompile(`(?m)^(\w[\w.]*)\s*=\s*(.+)$`)
)

// ParseGradleProject parses a Gradle project from the given root directory.
// It reads settings.gradle(.kts) for subproject includes, and build.gradle(.kts) for
// group/version information. Returns a ModuleInfo tree compatible with the Maven parser output.
func ParseGradleProject(rootDir string) (*ModuleInfo, error) {
	// Find settings file
	settingsContent, err := readGradleFile(rootDir, "settings.gradle", "settings.gradle.kts")
	if err != nil {
		// No settings file — try single-project build
		_, buildErr := readGradleFile(rootDir, "build.gradle", "build.gradle.kts")
		if buildErr != nil {
			return nil, fmt.Errorf("no Gradle build files found in %s", rootDir)
		}
		return parseSingleGradleProject(rootDir)
	}

	// Read optional gradle.properties for property substitution
	properties := readGradleProperties(rootDir)

	// Extract root project name
	rootName := filepath.Base(rootDir)
	if m := rootProjectNameRegex.FindStringSubmatch(settingsContent); len(m) > 1 {
		rootName = m[1]
	}

	// Parse root build.gradle for group/version
	group, version := "com.example", "1.0.0"
	if buildContent, err := readGradleFile(rootDir, "build.gradle", "build.gradle.kts"); err == nil {
		g, v := extractGroupVersion(buildContent, properties)
		if g != "" {
			group = g
		}
		if v != "" {
			version = v
		}
	}

	rootModule := &ModuleInfo{
		ArtifactID:     rootName,
		GroupID:        group,
		Version:        version,
		Coordinates:    fmt.Sprintf("%s:%s:%s", group, rootName, version),
		Path:           rootDir,
		SourcePath:     filepath.Join(rootDir, "src", "main", "java"),
		TestSourcePath: filepath.Join(rootDir, "src", "test", "java"),
		TargetPath:     filepath.Join(rootDir, "build"),
		SubModules:     []*ModuleInfo{},
		Properties:     properties,
	}

	// Extract included subprojects
	subprojects := extractSubprojects(settingsContent)
	sort.Strings(subprojects)

	for _, sub := range subprojects {
		// Gradle uses ":" as separator, e.g. ":app" or ":core:utils"
		subDir := strings.ReplaceAll(strings.TrimPrefix(sub, ":"), ":", string(filepath.Separator))
		subPath := filepath.Join(rootDir, subDir)

		subGroup := group
		subVersion := version
		if buildContent, err := readGradleFile(subPath, "build.gradle", "build.gradle.kts"); err == nil {
			g, v := extractGroupVersion(buildContent, properties)
			if g != "" {
				subGroup = g
			}
			if v != "" {
				subVersion = v
			}
		}

		artifactID := filepath.Base(subDir)
		subModule := &ModuleInfo{
			ArtifactID:     artifactID,
			GroupID:        subGroup,
			Version:        subVersion,
			Coordinates:    fmt.Sprintf("%s:%s:%s", subGroup, artifactID, subVersion),
			Path:           subPath,
			SourcePath:     filepath.Join(subPath, "src", "main", "java"),
			TestSourcePath: filepath.Join(subPath, "src", "test", "java"),
			TargetPath:     filepath.Join(subPath, "build"),
			SubModules:     []*ModuleInfo{},
			Properties:     properties,
		}
		rootModule.SubModules = append(rootModule.SubModules, subModule)
	}

	return rootModule, nil
}

func parseSingleGradleProject(rootDir string) (*ModuleInfo, error) {
	properties := readGradleProperties(rootDir)
	group, version := "com.example", "1.0.0"
	rootName := filepath.Base(rootDir)

	if buildContent, err := readGradleFile(rootDir, "build.gradle", "build.gradle.kts"); err == nil {
		g, v := extractGroupVersion(buildContent, properties)
		if g != "" {
			group = g
		}
		if v != "" {
			version = v
		}
	}

	return &ModuleInfo{
		ArtifactID:     rootName,
		GroupID:        group,
		Version:        version,
		Coordinates:    fmt.Sprintf("%s:%s:%s", group, rootName, version),
		Path:           rootDir,
		SourcePath:     filepath.Join(rootDir, "src", "main", "java"),
		TestSourcePath: filepath.Join(rootDir, "src", "test", "java"),
		TargetPath:     filepath.Join(rootDir, "build"),
		SubModules:     []*ModuleInfo{},
		Properties:     properties,
	}, nil
}

func readGradleFile(dir string, names ...string) (string, error) {
	for _, name := range names {
		p := filepath.Join(dir, name)
		data, err := os.ReadFile(p)
		if err == nil {
			return string(data), nil
		}
	}
	return "", fmt.Errorf("no gradle file found in %s", dir)
}

func readGradleProperties(dir string) map[string]string {
	props := make(map[string]string)
	data, err := os.ReadFile(filepath.Join(dir, "gradle.properties"))
	if err != nil {
		return props
	}
	for _, m := range gradlePropRegex.FindAllStringSubmatch(string(data), -1) {
		props[m[1]] = strings.TrimSpace(m[2])
	}
	return props
}

func extractGroupVersion(content string, props map[string]string) (group, version string) {
	if m := groupRegex.FindStringSubmatch(content); len(m) > 1 {
		group = resolveGradleProperty(m[1], props)
	}
	if m := versionRegex.FindStringSubmatch(content); len(m) > 1 {
		version = resolveGradleProperty(m[1], props)
	}
	return
}

func resolveGradleProperty(value string, props map[string]string) string {
	// Resolve ${property} references
	return propRegex.ReplaceAllStringFunc(value, func(match string) string {
		key := match[2 : len(match)-1]
		if val, ok := props[key]; ok {
			return val
		}
		return match
	})
}

func extractSubprojects(settingsContent string) []string {
	var subs []string
	seen := make(map[string]bool)
	for _, m := range includeRegex.FindAllStringSubmatch(settingsContent, -1) {
		items := includeItemRegex.FindAllStringSubmatch(m[1], -1)
		for _, item := range items {
			name := item[1]
			if !seen[name] {
				seen[name] = true
				subs = append(subs, name)
			}
		}
	}
	return subs
}
