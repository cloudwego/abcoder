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

	sitter "github.com/smacker/go-tree-sitter"
)

// DebugNode is a Go-native representation of a tree-sitter node for easy debugging.
// It contains the essential information about a node in a format that is friendly
// to Go's debugging tools.

type DebugNode struct {
	Type     string       `json:"type"`
	Content  string       `json:"content"`
	Start    sitter.Point `json:"start"`
	End      sitter.Point `json:"end"`
	Children []*DebugNode `json:"children"`
}

// NewDebugTree parses the source code and builds a Go-native debug tree.
// This function is designed to be used in testing and debugging scenarios.
func NewDebugTree(ctx context.Context, content []byte) (*DebugNode, error) {
	tree, err := Parse(ctx, content)
	if err != nil {
		return nil, err
	}

	rootSitterNode := tree.RootNode()
	debugRoot := buildDebugNode(rootSitterNode, content)

	return debugRoot, nil
}

// buildDebugNode is a recursive helper function that converts a sitter.Node
// into a DebugNode.
func buildDebugNode(node *sitter.Node, content []byte) *DebugNode {
	if node == nil {
		return nil
	}

	children := make([]*DebugNode, 0, node.ChildCount())
	for i := 0; i < int(node.ChildCount()); i++ {
		childSitterNode := node.Child(i)
		children = append(children, buildDebugNode(childSitterNode, content))
	}

	return &DebugNode{
		Type:     node.Type(),
		Content:  node.Content(content),
		Start:    node.StartPoint(),
		End:      node.EndPoint(),
		Children: children,
	}
}
