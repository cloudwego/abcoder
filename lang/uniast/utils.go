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
)

func InsertDependency(ids []Dependency, id Dependency) []Dependency {
	for _, i := range ids {
		if i.Identity == id.Identity {
			return ids
		}
	}
	return append(ids, id)
}

func InserImport(ids []Import, id Import) []Import {
	for _, i := range ids {
		if i.Path == id.Path {
			return ids
		}
	}
	return append(ids, id)
}

func InsertRelation(ids []Relation, id Relation) []Relation {
	for _, i := range ids {
		if i.Identity == id.Identity {
			return ids
		}
	}
	return append(ids, id)
}

func LoadRepo(path string) (*Repository, error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var repo Repository
	if err := json.Unmarshal(bs, &repo); err != nil {
		return nil, err
	}
	repo.AllNodesSetRepo()
	return &repo, nil
}
