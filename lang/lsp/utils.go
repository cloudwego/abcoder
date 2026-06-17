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

package lsp

import (
	"strings"

	"github.com/cloudwego/abcoder/lang/log"
	"github.com/cloudwego/abcoder/lang/utils"
)

func GetDistance(text string, start Position, pos Position) int {
	lines := utils.CountLinesPooled(text)
	defer utils.PutCount(lines)
	// find the line of the position
	l := pos.Line - start.Line
	// Defensive: out-of-range line (degenerate clangd range) -> -1, which
	// ChunkHead already treats as "no text". Avoids an index-out-of-range
	// panic mid-collect/export.
	if l < 0 || l >= len(*lines) {
		return -1
	}
	return (*lines)[l] + pos.Character - start.Character
}

// calculate the relative index of a position to a text
func ChunkHead(text string, textPos Position, pos Position) string {
	distance := GetDistance(text, textPos, pos)
	if distance < 0 || distance >= len(text) {
		return ""
	}
	return text[:distance]
}

// calculate the relative index of a position to a text
func RelativePostionWithLines(lines []int, basePos Position, pos Position) int {
	// find the line of the position
	l := pos.Line - basePos.Line
	// Defensive: clangd can report a position whose line is outside the
	// cached line table (degenerate / macro-expanded / stale ranges), which
	// would otherwise panic here with index-out-of-range and abort the whole
	// parse (seen in both CppSpec.FunctionSymbol during collect and
	// fileLine→PositionOffset during export). Return -1, the same "unknown
	// offset" sentinel PositionOffset already produces for invalid input;
	// callers either guard on `< 0` or just record a -1 offset.
	if l < 0 || l >= len(lines) {
		return -1
	}
	return lines[l] + pos.Character - basePos.Character
}

func PositionOffset(file_uri string, text string, pos Position) int {
	if pos.Line < 0 || pos.Character < 0 {
		log.Error("invalid text position: %+v", pos)
		return -1
	}
	lines := utils.CountLinesCached(file_uri, text)
	return RelativePostionWithLines(*lines, Position{Line: 0, Character: 0}, pos)
}

// FindSingle finds the single char's left token index in a text
// start and end is the limit range of tokens
func FindSingle(text string, lines []int, textPos Position, tokens []Token, sep string, start int, end int) int {
	if start < 0 {
		start = 0
	}
	if end >= len(tokens) {
		end = len(tokens) - 1
	}
	if start >= len(tokens) {
		return -1
	}
	sPos := RelativePostionWithLines(lines, textPos, tokens[start].Location.Range.Start)
	ePos := RelativePostionWithLines(lines, textPos, tokens[end].Location.Range.End)
	pos := strings.Index(text[sPos:ePos], sep)
	if pos == -1 {
		return -1
	}
	pos += sPos
	for i := start; i <= end && i < len(tokens); i++ {
		rel := RelativePostionWithLines(lines, textPos, tokens[i].Location.Range.Start)
		if rel > pos {
			return i - 1
		}
	}
	return -1
}

// FindPair finds the right token index of lchar and left token index of rchar in a text
// start and end is the limit range of tokens
// notAllow is the character that not allow in the range
func FindPair(text string, lines []int, textPos Position, tokens []Token, lchar rune, rchar rune, start int, end int, notAllow rune) (int, int) {
	if start < 0 {
		start = 0
	}
	if end >= len(tokens) {
		end = len(tokens) - 1
	}
	if start >= len(tokens) {
		return -1, -1
	}

	startIndex := RelativePostionWithLines(lines, textPos, tokens[start].Location.Range.Start)

	lArrow := -1
	lCount := 0
	rArrow := -1
	notAllowCount := 0
	ctext := text[startIndex:]
	for i, c := range ctext {
		if c == notAllow && lCount == 0 {
			return -1, -1
		} else if c == lchar && notAllowCount == 0 {
			lCount++
			if lCount == 1 {
				lArrow = i
			}
		} else if c == rchar && notAllowCount == 0 {
			if rchar == '>' && ctext[i-1] == '-' {
				// notice: -> is not a pair in Rust
				continue
			}
			lCount--
			if lCount == 0 {
				rArrow = i
				break
			}
		}
	}
	if lArrow == -1 || rArrow == -1 {
		return -1, -1
	}
	lArrow += startIndex
	rArrow += startIndex

	s := -1
	e := -1
	for i := start; i <= end && i < len(tokens); i++ {
		rel := RelativePostionWithLines(lines, textPos, tokens[i].Location.Range.Start)
		if rel >= lArrow && s == -1 {
			s = i
		}
		if rel > rArrow {
			e = i - 1
			break
		}
	}

	return s, e
}
