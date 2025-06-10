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

var countCache = sync.Map{}

// USE AND ONLY USE in pairs with CountLinesPooled.
func PutCount(count *[]int) {
	*count = (*count)[:0]
	countPool.Put(count)
}

// The cached version of CountLines.
// Avoids redundant computations for the same text.
// Use when the same text is processed multiple times, such as contents of a file.
//
// The key MUST uniquely identify the text i.e. for any two invocations
//
//	CountLinesCached(key1, text1) and CountLinesCached(key2, text2),
//
// if key1 == key2, then text1 must be equal to text2 (and also vice versa).
func CountLinesCached(key string, text string) *[]int {
	if cached, ok := countCache.Load(key); ok {
		res := cached.([]int)
		return &res
	}

	lines := CountLines(text)
	countCache.Store(key, lines)
	return &lines
}

// The pooled version of CountLines.
// Eases burden on allocation and GC.
// Use when invocation on small text is frequent.
//
// MUST manually invoke `PutCount` when done with the result to return the slice to the pool.
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

// CountLines calculates the starting offsets of lines in a given text.
// Each offset marks the byte position of the character immediately following a newline character,
// or 0 for the very beginning of the text.
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
