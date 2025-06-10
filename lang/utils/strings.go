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

package utils

import "sync"

var countPool = sync.Pool{
	New: func() interface{} {
		ret := make([]int, 0, 256)
		return &ret
	},
}

func PutCount(count *[]int) {
	*count = (*count)[:0]
	countPool.Put(count)
}

var cachedLines = sync.Map{}

func CountLinesCached(ident string, text string) *[]int {
	if v, ok := cachedLines.Load(ident); ok {
		res := v.([]int)
		return &res
	}
	tmp := CountLines(text)
	cachedLines.Store(ident, tmp)
	return &tmp
}

func CountLinesPooled(text string) *[]int {
	tmp := countPool.Get().(*[]int)
	*tmp = append(*tmp, 0)
	for i, c := range text {
		if c == '\n' {
			*tmp = append(*tmp, i+1)
		}
	}
	return tmp
}

func CountLines(text string) []int {
	var ret []int
	ret = append(ret, 0)
	for i, c := range text {
		if c == '\n' {
			ret = append(ret, i+1)
		}
	}
	return ret
}

func DedupSlice[T comparable](s []T) []T {
	seen := make(map[T]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}
