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

func Test_Rust(t *testing.T) {
	useStatements, err := ParseUseStatements(`use http::Method as HttpMethod;
use http::{Server, Request as HttpRequest, Response::{IntoResponse, StatusCode as HttpStatusCode}};
`)
	if err != nil {
		fmt.Printf("Error parsing use statements: %v\n", err)
		return
	}
	dependencyTree := BuildDependencyTree(useStatements)
	//PrintTree(dependencyTree, "")
	for _, r := range dependencyTree.Children {
		uses := ConvertTreeToUse(r, "")
		for _, u := range uses {
			fmt.Println(u)
		}
	}
}
