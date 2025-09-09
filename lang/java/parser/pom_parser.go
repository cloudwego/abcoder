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
	"log"
	"path/filepath"
	"regexp"

	"github.com/vifraa/gopom"
)

// ModuleInfo stores information about a Maven module.
type ModuleInfo struct {
	ArtifactID     string
	GroupID        string
	Version        string
	Coordinates    string
	Path           string
	SourcePath     string
	TestSourcePath string
	TargetPath     string
	SubModules     []*ModuleInfo
	Properties     map[string]string
}

// ParseMavenProject recursively parses a module and its submodules.
// pomPath: The path to the pom.xml file to parse.
func ParseMavenProject(pomPath string) (*ModuleInfo, error) {
	return parseMavenProject(pomPath, nil)
}

var propRegex = regexp.MustCompile(`\$\{(.+?)\}`)

func resolveProperty(value string, properties map[string]string) string {
	resolvedValue := value
	for i := 0; i < 10; i++ { // Limit iterations to prevent infinite loops
		newValue := propRegex.ReplaceAllStringFunc(resolvedValue, func(match string) string {
			key := match[2 : len(match)-1]
			if val, ok := properties[key]; ok {
				return val
			}
			return match
		})
		if newValue == resolvedValue {
			return newValue
		}
		resolvedValue = newValue
	}
	return resolvedValue
}

func parseMavenProject(pomPath string, parent *ModuleInfo) (*ModuleInfo, error) {
	// 1. Parse the pom.xml file using gopom.
	project, err := gopom.Parse(pomPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", pomPath, err)
	}

	// Collect properties from parent and current pom
	properties := make(map[string]string)
	if parent != nil && parent.Properties != nil {
		for k, v := range parent.Properties {
			properties[k] = v
		}
	}
	if project.Properties != nil && project.Properties.Entries != nil {
		for k, v := range project.Properties.Entries {
			properties[k] = v
		}
	}

	var groupID, version string
	if project.GroupID != nil {
		groupID = *project.GroupID
	} else if parent != nil {
		groupID = parent.GroupID
	}

	if project.Version != nil {
		version = *project.Version
	} else if parent != nil {
		version = parent.Version
	}

	// Resolve properties in version and groupID
	version = resolveProperty(version, properties)
	groupID = resolveProperty(groupID, properties)

	if project.ArtifactID == nil {
		return nil, fmt.Errorf("artifactId is missing in %s", pomPath)
	}

	// Determine source and test source directories
	modulePath := filepath.Dir(pomPath)
	sourcePath := filepath.Join(modulePath, "src", "main", "java")
	testSourcePath := filepath.Join(modulePath, "src", "test", "java")
	targetPath := filepath.Join(modulePath, "target")
	if project.Build != nil {
		if project.Build.SourceDirectory != nil {
			sourcePath = filepath.Join(modulePath, *project.Build.SourceDirectory)
		}
		if project.Build.TestSourceDirectory != nil {
			testSourcePath = filepath.Join(modulePath, *project.Build.TestSourceDirectory)
		}
		if project.Build.OutputDirectory != nil {
			targetPath = filepath.Join(modulePath, *project.Build.OutputDirectory)
		}
	}

	// 2. Create a struct to store our module information.
	currentModule := &ModuleInfo{
		ArtifactID:     *project.ArtifactID,
		GroupID:        groupID,
		Version:        version,
		Coordinates:    fmt.Sprintf("%s:%s:%s", groupID, *project.ArtifactID, version),
		Path:           modulePath,
		SourcePath:     sourcePath,
		TestSourcePath: testSourcePath,
		TargetPath:     targetPath,
		SubModules:     []*ModuleInfo{},
		Properties:     properties,
	}

	// 3. If a <modules> section exists, recursively parse the submodules.
	if project.Modules != nil && len(*project.Modules) > 0 {
		for _, moduleName := range *project.Modules {
			// Construct the path to the submodule's pom.xml.
			subPomPath := filepath.Join(currentModule.Path, moduleName, "pom.xml")

			// Recursive call.
			subModuleInfo, err := parseMavenProject(subPomPath, currentModule)
			if err != nil {
				// If parsing a submodule fails, we can log it and skip.
				log.Printf("Warning: failed to parse submodule %s: %v", subPomPath, err)
				continue
			}
			currentModule.SubModules = append(currentModule.SubModules, subModuleInfo)
		}
	}

	return currentModule, nil
}

func GetModuleMap(root *ModuleInfo) map[string]string {
	rets := map[string]string{}
	var queue []*ModuleInfo
	if root != nil {
		queue = append(queue, root)
	}
	for len(queue) > 0 {
		module := queue[0]
		queue = queue[1:]
		rets[module.Coordinates] = module.Path
		for _, subModule := range module.SubModules {
			queue = append(queue, subModule)
		}
	}
	return rets
}

func GetModuleStructMap(root *ModuleInfo) map[string]*ModuleInfo {
	rets := map[string]*ModuleInfo{}
	var queue []*ModuleInfo
	if root != nil {
		queue = append(queue, root)
	}
	for len(queue) > 0 {
		module := queue[0]
		queue = queue[1:]
		rets[module.Coordinates] = module
		for _, subModule := range module.SubModules {
			queue = append(queue, subModule)
		}
	}
	return rets
}

func GetModulePaths(root *ModuleInfo) []string {
	var paths []string
	moduleMap := GetModuleMap(root)
	for _, path := range moduleMap {
		paths = append(paths, path)
	}
	return paths
}

// PrintProjectTree prints the project structure in a hierarchical format.
func PrintProjectTree(module *ModuleInfo, indent string) {
	if module == nil {
		return
	}
	// Print current module info.
	fmt.Printf("%s- %s\n", indent, module.Coordinates)

	// Recursively print submodules.
	for _, subModule := range module.SubModules {
		PrintProjectTree(subModule, indent+"  ")
	}
}
