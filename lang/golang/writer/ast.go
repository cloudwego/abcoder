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

package writer

import (
	"strconv"
	"strings"

	"github.com/cloudwego/abcoder/lang/uniast"
)

func writeImport(sb *strings.Builder, impts []uniast.Import) {
	if len(impts) == 0 {
		return
	}
	sb.WriteString("import ")
	if len(impts) == 1 {
		writeSingleImport(sb, impts[0])
		return
	}
	sb.WriteString("(\n")
	for i := 0; i < len(impts); i++ {
		sb.WriteString("\t")
		writeSingleImport(sb, impts[i])
	}
	sb.WriteString(")\n")
}

func writeSingleImport(sb *strings.Builder, v uniast.Import) {
	if v.Alias != nil {
		sb.WriteString(*v.Alias)
		sb.WriteString(" ")
	}
	sb.WriteString(strconv.Quote(v.Path))
	sb.WriteString("\n")
}

// merge the imports of file and nodes, and return the merged imports
// file is in priority (because it contains alias)
func mergeImports(priors []uniast.Import, subs []uniast.Import) (ret []uniast.Import) {
	visited := make(map[string]bool, len(priors)+len(subs))
	ret = make([]uniast.Import, 0, len(priors)+len(subs))
	for _, v := range priors {

		if visited[v.Path] {
			continue
		} else {
			visited[v.Path] = true
			ret = append(ret, v)
		}
	}
	for _, v := range subs {
		if visited[v.Path] {
			continue
		} else {
			visited[v.Path] = true
			ret = append(ret, v)
		}
	}
	return
}
