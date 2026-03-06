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

package writer

import (
	"strings"

	"github.com/cloudwego/abcoder/lang/uniast"
)

func writeImport(sb *strings.Builder, impts []uniast.Import) {
	for _, imp := range impts {
		writeSingleImport(sb, imp)
	}
	sb.WriteString("\n")
}

func writeSingleImport(sb *strings.Builder, v uniast.Import) {
	sb.WriteString("import ")
	if v.Alias != nil && *v.Alias == "static" {
		sb.WriteString("static ")
	}
	sb.WriteString(v.Path)
	sb.WriteString(";\n")
}

// mergeImports merges two import slices, deduplicating by path.
// priors take precedence (because they may contain aliases).
func mergeImports(priors []uniast.Import, subs []uniast.Import) (ret []uniast.Import) {
	visited := make(map[string]bool, len(priors)+len(subs))
	ret = make([]uniast.Import, 0, len(priors)+len(subs))
	for _, v := range priors {
		if visited[v.Path] {
			continue
		}
		visited[v.Path] = true
		ret = append(ret, v)
	}
	for _, v := range subs {
		if visited[v.Path] {
			continue
		}
		visited[v.Path] = true
		ret = append(ret, v)
	}
	return
}
