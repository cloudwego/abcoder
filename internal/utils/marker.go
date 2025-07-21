/**
 * Copyright 2024 ByteDance Inc.
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

package utils

import (
	"strings"

	"github.com/cloudwego/abcoder/lang/uniast"
)

const (
	MARKDOWN_CODE_BLOCK = "```"
)

func ExtractMDCodes(resp string, lang string) (js string, realLang uniast.Language, last int) {
	if idx := strings.Index(resp, MARKDOWN_CODE_BLOCK+lang); idx >= 0 {
		idx = idx + 3 + len(lang)
		idx2 := strings.Index(resp[idx:], MARKDOWN_CODE_BLOCK)
		if idx2 < 0 {
			return "", "", -1
		}
		return resp[idx : idx+idx2], MDBlockToLang(lang), idx + idx2 + 3
	} else if idx := strings.Index(resp, MARKDOWN_CODE_BLOCK); idx >= 0 {
		idx = idx + 3
		s := idx
		for ; idx < len(resp) && resp[idx] != '\n'; idx++ {
		}
		realLang = MDBlockToLang(resp[s:idx])
		idx2 := strings.Index(resp[idx:], MARKDOWN_CODE_BLOCK)
		if idx2 < 0 {
			return "", "", -1
		}
		return resp[idx : idx+idx2], realLang, idx + idx2 + 3
	}
	return
}

func LangToMDBlock(lang uniast.Language) string {
	switch lang {
	case uniast.Golang:
		return "go"
	case uniast.Rust:
		return "rust"
	default:
		return ""
	}
}

func MDBlockToLang(lang string) uniast.Language {
	switch lang {
	case "go":
		return uniast.Golang
	case "rust":
		return uniast.Rust
	default:
		return uniast.Language(lang)
	}
}

func ExtractXMLBlock(text string, tag string) (string, int) {
	idx := strings.Index(text, "<"+tag+">")
	if idx < 0 {
		return "", -1
	}
	idx2 := strings.Index(text[idx:], "</"+tag+">")
	if idx2 < 0 {
		return "", -1
	}
	return text[idx+len(tag)+2 : idx+idx2], idx + idx2 + len(tag) + 3
}

func IterateXMLBlock(text string, tag string, hook func(block string) bool) {
	cur := text
	// extracts research tasks
	for cur != "" {
		block, next := ExtractXMLBlock(cur, tag)
		if next >= 0 {
			if !hook(block) {
				return
			}
			if next > len(cur) {
				next = len(cur)
			}
			cur = cur[next:]
		} else {
			return
		}
	}
}
