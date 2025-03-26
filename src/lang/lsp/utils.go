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
	"github.com/cloudwego/abcoder/src/lang/log"
	"github.com/cloudwego/abcoder/src/lang/utils"
)

func GetDistance(text string, start Position, pos Position) int {
	lines := utils.CountLinesCached(text)
	defer utils.PutCount(lines)
	// find the line of the position
	return (*lines)[pos.Line-start.Line] + pos.Character - start.Character
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
func RelativePostionWithLines(lines []int, textPos Position, pos Position) int {
	// find the line of the position
	l := pos.Line - textPos.Line

	return lines[l] + pos.Character - textPos.Character
}

func PositionOffset(text string, pos Position) int {
	if pos.Line < 1 || pos.Character < 1 {
		log.Error("invalid text position: %+v", pos)
		return -1
	}
	lines := utils.CountLinesCached(text)
	defer utils.PutCount(lines)

	return RelativePostionWithLines(*lines, Position{Line: 1, Character: 1}, pos)
}
