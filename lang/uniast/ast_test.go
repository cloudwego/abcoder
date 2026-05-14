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

// TestRepository_BuildGraph_Deterministic ensures BuildGraph yields a byte-stable
// JSON repeatedly. Relation slices (References, Dependencies, etc.) are filled
// via map iteration, so without an explicit canonical sort each run produced a
// different order and downstream diffs lit up spurious changes.
func TestRepository_BuildGraph_Deterministic(t *testing.T) {
	astFile := testutils.GetTestAstFile("localsession")

	var prev []byte
	for i := 0; i < 5; i++ {
		r, err := LoadRepo(astFile)
		if err != nil {
			t.Fatalf("iter %d: load repo: %v", i, err)
		}
		if err := r.BuildGraph(); err != nil {
			t.Fatalf("iter %d: build graph: %v", i, err)
		}
		js, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("iter %d: marshal: %v", i, err)
		}
		if i == 0 {
			prev = js
			continue
		}
		if string(prev) != string(js) {
			t.Fatalf("iter %d: BuildGraph output is not byte-stable across runs (len %d vs %d)", i, len(prev), len(js))
		}
	}
}

func BenchmarkRepository_BuildGraph(b *testing.B) {
	astFile := testutils.GetTestAstFile("large_ast")
	r, err := LoadRepo(astFile)
	if err != nil {
		b.Fatalf("failed to load repo: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := r.BuildGraph(); err != nil {
			b.Fatalf("failed to build graph: %v", err)
		}
	}
}
