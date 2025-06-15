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

package lsp

import (
	"testing"
)

func TestPositionOffset(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		pos      Position
		expected int
	}{
		{
			name:     "Single line text, position at start",
			text:     "Hello, World!",
			pos:      Position{Line: 0, Character: 0},
			expected: 0,
		},
		{
			name:     "Single line text, position at end",
			text:     "Hello, World!",
			pos:      Position{Line: 0, Character: 13},
			expected: 13,
		},
		{
			name:     "Multi-line text, position at start of first line",
			text:     "Line 1\nLine 2\nLine 3",
			pos:      Position{Line: 0, Character: 0},
			expected: 0,
		},
		{
			name:     "Multi-line text, position at end of first line",
			text:     "Line 1\nLine 2\nLine 3",
			pos:      Position{Line: 0, Character: 6},
			expected: 6,
		},
		{
			name:     "Multi-line text, position at start of second line",
			text:     "Line 1\nLine 2\nLine 3",
			pos:      Position{Line: 1, Character: 0},
			expected: 7,
		},
		{
			name:     "Multi-line text, position at end of last line",
			text:     "Line 1\nLine 2\nLine 3",
			pos:      Position{Line: 2, Character: 6},
			expected: 20,
		},
		{
			name:     "Empty text, position at start",
			text:     "",
			pos:      Position{Line: 0, Character: 0},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PositionOffset(tt.name, tt.text, tt.pos)
			if result != tt.expected {
				t.Errorf("PositionOffset() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
