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
	"testing"
)

func TestNewIdentityFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Identity
	}{
		{
			name:     "standard format",
			input:    "mod?pkg#name",
			expected: Identity{ModPath: "mod", PkgPath: "pkg", Name: "name"},
		},
		{
			name:     "name with question mark - Java wildcard",
			input:    "mod?pkg#name<?>",
			expected: Identity{ModPath: "mod", PkgPath: "pkg", Name: "name<?>"},
		},
		{
			name:     "name with multiple question marks",
			input:    "mod?pkg#Map<?,?>",
			expected: Identity{ModPath: "mod", PkgPath: "pkg", Name: "Map<?,?>"},
		},
		{
			name:     "no ModPath",
			input:    "pkg#name",
			expected: Identity{ModPath: "", PkgPath: "pkg", Name: "name"},
		},
		{
			name:     "no PkgPath and ModPath",
			input:    "name",
			expected: Identity{ModPath: "", PkgPath: "", Name: "name"},
		},
		{
			name:     "complex Java generic",
			input:    "mod?pkg#Function<? super T, ? extends R>",
			expected: Identity{ModPath: "mod", PkgPath: "pkg", Name: "Function<? super T, ? extends R>"},
		},
		{
			name:     "with version number",
			input:    "mod@v1.0?pkg#name",
			expected: Identity{ModPath: "mod@v1.0", PkgPath: "pkg", Name: "name"},
		},
		{
			name:     "Java method with generic parameters",
			input:    "com.example@1.0?com.example.utils#process<?>",
			expected: Identity{ModPath: "com.example@1.0", PkgPath: "com.example.utils", Name: "process<?>"},
		},
		{
			name:     "nested generics",
			input:    "mod?pkg#List<Map<String, ?>>",
			expected: Identity{ModPath: "mod", PkgPath: "pkg", Name: "List<Map<String, ?>>"},
		},
		{
			name:     "capture wildcard",
			input:    "mod?pkg#capture of ?",
			expected: Identity{ModPath: "mod", PkgPath: "pkg", Name: "capture of ?"},
		},
		{
			name:     "ModPath with empty PkgPath and Name",
			input:    "mod?#",
			expected: Identity{ModPath: "mod", PkgPath: "", Name: ""},
		},
		{
			name:     "only PkgPath separator",
			input:    "pkg#",
			expected: Identity{ModPath: "", PkgPath: "pkg", Name: ""},
		},
		{
			name:     "both separators but empty parts",
			input:    "?#",
			expected: Identity{ModPath: "", PkgPath: "", Name: ""},
		},
		{
			name:     "ModPath with version and question mark in name",
			input:    "mod@v1.0?pkg#method<?>",
			expected: Identity{ModPath: "mod@v1.0", PkgPath: "pkg", Name: "method<?>"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: Identity{ModPath: "", PkgPath: "", Name: ""},
		},
		{
			"java example 1",
			`com.bytedance.ea.travel:travel-web:1.0.0-SNAPSHOT?com.bytedance.ea.travel.web.controller#CommonInfoController.public Result<?> allCountries(@RequestParam(name = "language", required = false) String language)`,
			Identity{ModPath: "com.bytedance.ea.travel:travel-web:1.0.0-SNAPSHOT", PkgPath: "com.bytedance.ea.travel.web.controller", Name: "CommonInfoController.public Result<?> allCountries(@RequestParam(name = \"language\", required = false) String language)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewIdentityFromString(tt.input)
			if result != tt.expected {
				t.Errorf("NewIdentityFromString(%q) = %+v, want %+v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIdentity_Full(t *testing.T) {
	tests := []struct {
		name     string
		identity Identity
		expected string
	}{
		{
			name:     "all parts present",
			identity: Identity{ModPath: "mod", PkgPath: "pkg", Name: "name"},
			expected: "mod?pkg#name",
		},
		{
			name:     "no ModPath",
			identity: Identity{ModPath: "", PkgPath: "pkg", Name: "name"},
			expected: "?pkg#name",
		},
		{
			name:     "no PkgPath",
			identity: Identity{ModPath: "mod", PkgPath: "", Name: "name"},
			expected: "mod?#name",
		},
		{
			name:     "only Name",
			identity: Identity{ModPath: "", PkgPath: "", Name: "name"},
			expected: "?#name",
		},
		{
			name:     "all empty",
			identity: Identity{ModPath: "", PkgPath: "", Name: ""},
			expected: "?#",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.identity.Full(); got != tt.expected {
				t.Errorf("Identity.Full() = %v, want %v", got, tt.expected)
			}
			// Round-trip test
			parsed := NewIdentityFromString(tt.expected)
			if parsed != tt.identity {
				t.Errorf("NewIdentityFromString(Full()) = %+v, want %+v", parsed, tt.identity)
			}
		})
	}
}
