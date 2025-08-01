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

package parser

import (
	"github.com/cloudwego/abcoder/lang/uniast"
	"github.com/joyme123/thrift-ls/parser"
	"github.com/joyme123/thrift-ls/utils"
)

func newFileLine(relFilePath string, line int, startPos, endPos parser.Position) uniast.FileLine {
	return uniast.FileLine{
		File:        relFilePath,
		Line:        line,
		StartOffset: startPos.Offset,
		EndOffset:   endPos.Offset,
	}
}

func removeAllComments(doc *parser.Document) {
	if doc == nil {
		return
	}
	removeCommentsRecursive(doc)
}

// removeCommentsRecursive 是一个递归函数，用于遍历 AST 并清除注释。
func removeCommentsRecursive(node parser.Node) {
	if utils.IsNil(node) {
		return
	}

	switch n := node.(type) {
	case *parser.Document:
		n.Comments = nil

	case *parser.Include:
		n.Comments = nil
		n.EndLineComments = nil
		if n.IncludeKeyword != nil {
			n.IncludeKeyword.Comments = nil
		}
	case *parser.CPPInclude:
		n.Comments = nil
		n.EndLineComments = nil
		if n.CPPIncludeKeyword != nil {
			n.CPPIncludeKeyword.Comments = nil
		}
	case *parser.Namespace:
		n.Comments = nil
		n.EndLineComments = nil
		if n.NamespaceKeyword != nil {
			n.NamespaceKeyword.Comments = nil
		}

	case *parser.Struct:
		n.Comments = nil
		n.EndLineComments = nil
		if n.StructKeyword != nil {
			n.StructKeyword.Comments = nil
		}
		if n.LCurKeyword != nil {
			n.LCurKeyword.Comments = nil
		}
		if n.RCurKeyword != nil {
			n.RCurKeyword.Comments = nil
		}
	case *parser.Union:
		n.Comments = nil
		n.EndLineComments = nil
		if n.UnionKeyword != nil {
			n.UnionKeyword.Comments = nil
		}
		if n.LCurKeyword != nil {
			n.LCurKeyword.Comments = nil
		}
		if n.RCurKeyword != nil {
			n.RCurKeyword.Comments = nil
		}
	case *parser.Exception:
		n.Comments = nil
		n.EndLineComments = nil
		if n.ExceptionKeyword != nil {
			n.ExceptionKeyword.Comments = nil
		}
		if n.LCurKeyword != nil {
			n.LCurKeyword.Comments = nil
		}
		if n.RCurKeyword != nil {
			n.RCurKeyword.Comments = nil
		}
	case *parser.Service:
		n.Comments = nil
		n.EndLineComments = nil
		if n.ServiceKeyword != nil {
			n.ServiceKeyword.Comments = nil
		}
		if n.ExtendsKeyword != nil {
			n.ExtendsKeyword.Comments = nil
		}
		if n.LCurKeyword != nil {
			n.LCurKeyword.Comments = nil
		}
		if n.RCurKeyword != nil {
			n.RCurKeyword.Comments = nil
		}
	case *parser.Enum:
		n.Comments = nil
		n.EndLineComments = nil
		if n.EnumKeyword != nil {
			n.EnumKeyword.Comments = nil
		}
		if n.LCurKeyword != nil {
			n.LCurKeyword.Comments = nil
		}
		if n.RCurKeyword != nil {
			n.RCurKeyword.Comments = nil
		}
	case *parser.Typedef:
		n.Comments = nil
		n.EndLineComments = nil
		if n.TypedefKeyword != nil {
			n.TypedefKeyword.Comments = nil
		}
	case *parser.Const:
		n.Comments = nil
		n.EndLineComments = nil
		if n.ConstKeyword != nil {
			n.ConstKeyword.Comments = nil
		}
		if n.EqualKeyword != nil {
			n.EqualKeyword.Comments = nil
		}
		if n.ListSeparatorKeyword != nil {
			n.ListSeparatorKeyword.Comments = nil
		}

	case *parser.Field:
		n.Comments = nil
		n.EndLineComments = nil
		if n.Index != nil {
			n.Index.Comments = nil
			if n.Index.ColonKeyword != nil {
				n.Index.ColonKeyword.Comments = nil
			}
		}
		if n.RequiredKeyword != nil {
			n.RequiredKeyword.Comments = nil
		}
		if n.EqualKeyword != nil {
			n.EqualKeyword.Comments = nil
		}
		if n.ListSeparatorKeyword != nil {
			n.ListSeparatorKeyword.Comments = nil
		}
	case *parser.Function:
		n.Comments = nil
		n.EndLineComments = nil
		if n.Oneway != nil {
			n.Oneway.Comments = nil
		}
		if n.Void != nil {
			n.Void.Comments = nil
		}
		if n.LParKeyword != nil {
			n.LParKeyword.Comments = nil
		}
		if n.RParKeyword != nil {
			n.RParKeyword.Comments = nil
		}
		if n.ListSeparatorKeyword != nil {
			n.ListSeparatorKeyword.Comments = nil
		}
	case *parser.EnumValue:
		n.Comments = nil
		n.EndLineComments = nil
		if n.EqualKeyword != nil {
			n.EqualKeyword.Comments = nil
		}
		if n.ListSeparatorKeyword != nil {
			n.ListSeparatorKeyword.Comments = nil
		}
	case *parser.Identifier:
		n.Comments = nil
	case *parser.Literal:
		n.Comments = nil
	case *parser.ConstValue:
		n.Comments = nil
	case *parser.FieldType:
		if n.TypeName != nil {
			n.TypeName.Comments = nil
		}
	case *parser.TypeName:
		n.Comments = nil
	}

	for _, child := range node.Children() {
		removeCommentsRecursive(child)
	}
}
