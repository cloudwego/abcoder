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

package collect

import "testing"

func TestCppFnHasBody(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    bool
	}{
		// Declarations
		{"plain decl", `void f();`, false},
		{"decl with default arg", `int g(int x = 0);`, false},
		{"decl with lambda default — the headline repro case",
			`void f(std::function<void()> cb = []{});`, false},
		{"decl with brace-init default",
			`void f(std::vector<int> v = {1, 2, 3});`, false},
		{"decl with templated function-type default",
			`void f(std::function<int(int)> cb = [](int x) { return x*2; });`, false},
		{"decl with trailing override/noexcept", `int handle(int x) const noexcept;`, false},
		{"comment containing brace", `// f { ...
void f();`, false},
		{"block comment containing brace", `/* {} */ void f();`, false},
		{"string literal containing brace", `void f(const char* s = "hello {world}");`, false},

		// Definitions
		{"plain definition", `void f() {}`, true},
		{"definition with body", `int g(int x) { return x + 1; }`, true},
		{"ctor with member init list",
			`Foo::Foo(int x) : member_(x) {}`, true},
		{"definition with lambda default + body",
			`void f(std::function<void()> cb = []{}) { cb(); }`, true},
		{"definition with brace-init default + body",
			`void f(std::vector<int> v = {1, 2, 3}) { use(v); }`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cppFnHasBody(tc.content)
			if got != tc.want {
				t.Errorf("cppFnHasBody(%q) = %v, want %v", tc.content, got, tc.want)
			}
		})
	}
}
