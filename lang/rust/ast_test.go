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

package rust

import (
	"fmt"
	"testing"
)

func TestRustDependencyTree(t *testing.T) {
	useStatements, err := ParseUseStatements(`use http::Method as HttpMethod;
use http::{Server, Request as HttpRequest, Response::{IntoResponse, StatusCode as HttpStatusCode}};
`)
	if err != nil {
		fmt.Printf("Error parsing use statements: %v\n", err)
		return
	}
	dependencyTree := BuildDependencyTree(useStatements)
	if len(dependencyTree.Children) != 1 {
		t.Fatalf("Expected 1 child, got %d", len(dependencyTree.Children))
	}
	uses := ConvertTreeToUse(dependencyTree.Children[0], "")
	usePaths := make([]string, len(uses))
	for i, u := range uses {
		usePaths[i] = u.Path
	}
	expectedPaths := []string{
		"use http::Method as HttpMethod;",
		"use http::Server;",
		"use http::Request as HttpRequest;",
		"use http::Response::IntoResponse;",
		"use http::Response::StatusCode as HttpStatusCode;",
	}
	if len(usePaths) != len(expectedPaths) {
		t.Fatalf("Expected %d paths, got %d", len(expectedPaths), len(usePaths))
	}
	for i := range usePaths {
		if usePaths[i] != expectedPaths[i] {
			t.Fatalf("Index %d, Expected path\n%s\nbut got\n%s", i, expectedPaths[i], usePaths[i])
		}
	}
}
