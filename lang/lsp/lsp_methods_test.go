// Copyright 2025 CloudWeGo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lsp

import (
	"testing"

	"github.com/cloudwego/abcoder/lang/utils"
)

func TestLocateOutOfBounds(t *testing.T) {
	const uri = DocumentURI("file:///tmp/sample.py")
	text := "def f():\n    return 1\n"
	cli := &LSPClient{
		files: map[DocumentURI]*TextDocumentItem{
			uri: {
				URI:        uri,
				Text:       text,
				LineCounts: utils.CountLines(text),
			},
		},
	}

	loc := func(sl, sc, el, ec int) Location {
		l := Location{URI: uri}
		l.Range.Start.Line, l.Range.Start.Character = sl, sc
		l.Range.End.Line, l.Range.End.Character = el, ec
		return l
	}

	got, err := cli.Locate(loc(0, 0, 0, 3))
	if err != nil {
		t.Fatalf("valid range returned error: %v", err)
	}
	if got != "def" {
		t.Fatalf("valid range = %q, want %q", got, "def")
	}

	cases := map[string]Location{
		"line past EOF":      loc(13, 0, 13, 6),
		"inverted offsets":   loc(1, 6, 0, 1),
		"character past EOF": loc(0, 0, 1, 1000),
	}
	for name, l := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := cli.Locate(l); err == nil {
				t.Fatalf("expected error for %s, got nil", name)
			}
		})
	}
}
