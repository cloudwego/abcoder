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

package parser

import (
	"context"
	"io/ioutil"
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	fileToAnalyze := "../../../testdata/java/0_simple/HelloWorld.java"

	content, err := ioutil.ReadFile(fileToAnalyze)
	assert.NoError(t, err)

	tree, err := Parse(context.Background(), content)
	assert.NoError(t, err)
	assert.NotNil(t, tree)

	root := tree.RootNode()
	assert.NotNil(t, root)

	// A simple check to see if we have a reasonable root node type
	assert.Equal(t, "program", root.Type())
}

func TestPrintTree(t *testing.T) {
	fileToAnalyze := "../../../testdata/java/0_simple/HelloWorld.java"

	content, err := ioutil.ReadFile(fileToAnalyze)
	assert.NoError(t, err)

	tree, err := Parse(context.Background(), content)
	assert.NoError(t, err)

	var printNode func(*sitter.Node, int)
	printNode = func(node *sitter.Node, level int) {
		if node == nil {
			return
		}

		indent := strings.Repeat("  ", level)
		contentType := node.Type()
		contentStr := strings.ReplaceAll(node.Content(content), "\n", "\\n")

		t.Logf("%s%s (%s) [%d:%d - %d:%d] `%s`",
			indent,
			contentType,
			node.Type(),
			node.StartPoint().Row, node.StartPoint().Column,
			node.EndPoint().Row, node.EndPoint().Column,
			contentStr,
		)

		for i := 0; i < int(node.ChildCount()); i++ {
			printNode(node.Child(i), level+1)
		}
	}

	t.Log("--- Syntax Tree --- ")
	printNode(tree.RootNode(), 0)
	t.Log("--- End Syntax Tree ---")
}

func TestDebugTree(t *testing.T) {
	fileToAnalyze := "../../../testdata/java/0_simple/HelloWorld.java"

	content, err := ioutil.ReadFile(fileToAnalyze)
	assert.NoError(t, err)

	debugTree, err := NewDebugTree(context.Background(), content)
	assert.NoError(t, err)
	assert.NotNil(t, debugTree)

	// <<<--- PLACE A BREAKPOINT ON THE LINE BELOW ---<<< //
	t.Log("Successfully built the debug tree. You can now inspect the 'debugTree' variable.")
}
