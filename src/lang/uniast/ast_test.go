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
)

var testdata = "../../testdata"

func TestRepository_BuildGraph(t *testing.T) {
	var r Repository
	data, err := os.ReadFile(testdata + "/ast/localsession.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	if err := r.BuildGraph(); err != nil {
		t.Fatal(err)
	}
	if js, err := json.Marshal(r); err != nil {
		t.Fatal(err)
	} else {
		if err := os.WriteFile(testdata+"/ast/localsession_g.json", js, os.FileMode(0644)); err != nil {
			t.Fatal(err)
		}
	}
}
