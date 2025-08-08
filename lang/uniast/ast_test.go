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

package uniast

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/cloudwego/abcoder/lang/testutils"
)

func TestRepository_BuildGraph(t *testing.T) {
	astFile := testutils.GetTestAstFile("localsession")
	r, err := LoadRepo(astFile)
	if err != nil {
		t.Fatalf("failed to load repo: %v", err)
	}
	if err := r.BuildGraph(); err != nil {
		t.Fatalf("failed to build graph: %v", err)
	}
	if js, err := json.Marshal(r); err != nil {
		t.Fatalf("failed to marshal repo: %v", err)
	} else {
		astFileWithGraph := testutils.GetTestDataRoot() + "/asts/localsession_g.json"
		if err := os.WriteFile(astFileWithGraph, js, 0644); err != nil {
			t.Fatalf("failed to write repo with graph: %v", err)
		}
	}
}
