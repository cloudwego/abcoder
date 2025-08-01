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
	"fmt"
	"strings"

	"github.com/cloudwego/abcoder/lang/log"
	"github.com/joyme123/thrift-ls/parser"
)

// getRealContent extracts the content from the source bytes using precise start and end offsets.
func (p *ThriftParser) getRealContent(source []byte, startOffset, endOffset int) string {
	if source == nil {
		return ""
	}
	sourceLen := len(source)
	if startOffset < 0 || endOffset > sourceLen || startOffset > endOffset {
		log.Error("Invalid content offset. Start: %d, End: %d, Source Length: %d", startOffset, endOffset, sourceLen)
		return ""
	}
	return string(source[startOffset:endOffset])
}

// getRealStructPositions returns a Struct node's real start and end positions (from 'struct' keyword to '}').
func (p *ThriftParser) getRealStructPositions(s *parser.Struct) (sp, ep parser.Position) {
	if s == nil {
		return parser.InvalidPosition, parser.InvalidPosition
	}
	startPos := s.Location.StartPos
	endPos := s.Location.EndPos

	if !p.opts.CollectComment {
		if s.StructKeyword != nil {
			startPos = s.StructKeyword.Pos()
		}
		if s.RCurKeyword != nil {
			endPos = s.RCurKeyword.End()
		}
	}
	return startPos, endPos
}

// getRealEnumPositions returns an Enum node's real start and end positions.
func (p *ThriftParser) getRealEnumPositions(e *parser.Enum) (sp, ep parser.Position) {
	if e == nil {
		return parser.InvalidPosition, parser.InvalidPosition
	}
	startPos := e.Location.StartPos
	endPos := e.Location.EndPos

	if !p.opts.CollectComment {
		if e.EnumKeyword != nil {
			startPos = e.EnumKeyword.Pos()
		}
		if e.RCurKeyword != nil {
			endPos = e.RCurKeyword.End()
		}
	}

	return startPos, endPos
}

// getRealServicePositions returns a Service node's real start and end positions.
func (p *ThriftParser) getRealServicePositions(s *parser.Service) (sp, ep parser.Position) {
	if s == nil {
		return parser.InvalidPosition, parser.InvalidPosition
	}
	startPos := s.Location.StartPos
	endPos := s.Location.EndPos

	if !p.opts.CollectComment {
		if s.ServiceKeyword != nil {
			startPos = s.ServiceKeyword.Pos()
		}
		if s.RCurKeyword != nil {
			endPos = s.RCurKeyword.End()
		}
	}

	return startPos, endPos
}

// getRealExceptionPositions returns an Exception node's real start and end positions.
func (p *ThriftParser) getRealExceptionPositions(e *parser.Exception) (sp, ep parser.Position) {
	if e == nil {
		return parser.InvalidPosition, parser.InvalidPosition
	}
	startPos := e.Location.StartPos
	endPos := e.Location.EndPos
	if !p.opts.CollectComment {
		if e.ExceptionKeyword != nil {
			startPos = e.ExceptionKeyword.Pos()
		}
		if e.RCurKeyword != nil {
			endPos = e.RCurKeyword.End()
		}
	}
	return startPos, endPos
}

// getRealUnionPositions returns a Union node's real start and end positions.
func (p *ThriftParser) getRealUnionPositions(u *parser.Union) (sp, ep parser.Position) {
	if u == nil {
		return parser.InvalidPosition, parser.InvalidPosition
	}
	startPos := u.Location.StartPos
	endPos := u.Location.EndPos

	if !p.opts.CollectComment {
		if u.UnionKeyword != nil {
			startPos = u.UnionKeyword.Pos()
		}
		if u.RCurKeyword != nil {
			endPos = u.RCurKeyword.End()
		}
	}

	return startPos, endPos
}

// getRealTypedefPositions returns a Typedef node's real start and end positions.
func (p *ThriftParser) getRealTypedefPositions(t *parser.Typedef) (sp, ep parser.Position) {
	if t == nil {
		return parser.InvalidPosition, parser.InvalidPosition
	}
	startPos := t.Location.StartPos
	endPos := t.Location.EndPos

	if !p.opts.CollectComment {
		if t.TypedefKeyword != nil {
			startPos = t.TypedefKeyword.Pos()
		}
		if t.Alias != nil {
			endPos = t.Alias.End()
		}
	}

	return startPos, endPos
}

// getRealConstPositions returns a Const definition's real start and end positions.
func (p *ThriftParser) getRealConstPositions(c *parser.Const) (sp, ep parser.Position) {
	if c == nil {
		return parser.InvalidPosition, parser.InvalidPosition
	}
	startPos := c.Location.StartPos
	endPos := c.Location.EndPos

	if !p.opts.CollectComment {
		if c.ConstKeyword != nil {
			startPos = c.ConstKeyword.Pos()
		}
		if c.ListSeparatorKeyword != nil {
			endPos = c.ListSeparatorKeyword.End()
		} else if c.Value != nil {
			endPos = c.Value.End()
		}
	}

	return startPos, endPos
}

func (p *ThriftParser) getRealFuncEndOffset(fn *parser.Function, collectSignature bool) int {
	if fn == nil {
		return -1
	}

	if !p.opts.CollectComment || collectSignature {
		if fn.ListSeparatorKeyword != nil {
			return fn.ListSeparatorKeyword.End().Offset
		}
		if fn.Annotations != nil {
			return fn.Annotations.End().Offset
		}
		if fn.Throws != nil {
			return fn.Throws.End().Offset
		}
		if fn.RParKeyword != nil {
			return fn.RParKeyword.End().Offset
		}
	}

	return fn.End().Offset
}

func (p *ThriftParser) getFuncStartOffset(fn *parser.Function, collectSignature bool) int {
	if fn == nil {
		return -1
	}
	if !p.opts.CollectComment || collectSignature {
		if fn.Oneway != nil {
			return fn.Oneway.Pos().Offset
		}
		if fn.Void != nil {
			return fn.Void.Pos().Offset
		}
		if fn.FunctionType != nil {
			return fn.FunctionType.Pos().Offset
		}
	}
	return fn.Pos().Offset
}

func (p *ThriftParser) getRealFuncStartLine(fn *parser.Function) int {
	if fn != nil && fn.Name != nil && fn.Name.Name != nil {
		return fn.Name.Name.Pos().Line
	}
	if fn != nil {
		return fn.Pos().Line
	}
	return -1
}

func (p *ThriftParser) getFuncSignature(function *parser.Function, source []byte) (string, error) {
	if function == nil || source == nil {
		return "", fmt.Errorf("function node or source is nil")
	}

	startOffset := p.getFuncStartOffset(function, true)
	endOffset := p.getRealFuncEndOffset(function, true)

	sourceLen := len(source)
	if startOffset < 0 || endOffset > sourceLen || startOffset > endOffset {
		return "", fmt.Errorf("invalid offset range for function '%s'. Start: %d, End: %d", function.Name.Name.Text, startOffset, endOffset)
	}

	signatureBytes := source[startOffset:endOffset]
	signature := strings.TrimSpace(string(signatureBytes))

	if strings.HasSuffix(signature, ",") || strings.HasSuffix(signature, ";") {
		signature = signature[:len(signature)-1]
		signature = strings.TrimSpace(signature)
	}

	return signature, nil
}

func (p *ThriftParser) getRealFieldTypePositions(ft *parser.FieldType) (sp, ep parser.Position) {
	if ft == nil {
		return parser.InvalidPosition, parser.InvalidPosition
	}
	startPos := ft.Location.StartPos
	endPos := ft.Location.EndPos

	if !p.opts.CollectComment {
		if ft.TypeName != nil {
			startPos = ft.TypeName.Pos()
		}

		if ft.Annotations != nil {
			endPos = ft.Annotations.End()
		} else if ft.RPointKeyword != nil {
			endPos = ft.RPointKeyword.End()
		} else if ft.TypeName != nil {
			endPos = ft.TypeName.End()
		}
	}

	return startPos, endPos
}

func (p *ThriftParser) getRealFieldTypeLine(ft *parser.FieldType) int {
	if ft == nil {
		return -1
	}

	if ft.TypeName != nil {
		return ft.TypeName.Pos().Line
	}

	return ft.Pos().Line
}

func (p *ThriftParser) getRealStructLine(s *parser.Struct) int {
	if s != nil && s.Identifier != nil && s.Identifier.Name != nil {
		return s.Identifier.Name.Pos().Line
	}
	if s != nil {
		return s.Pos().Line
	}
	return -1
}

func (p *ThriftParser) getRealEnumLine(e *parser.Enum) int {
	if e != nil && e.Name != nil && e.Name.Name != nil {
		return e.Name.Name.Pos().Line
	}
	if e != nil {
		return e.Pos().Line
	}
	return -1
}

func (p *ThriftParser) getRealServiceLine(s *parser.Service) int {
	if s != nil && s.Name != nil && s.Name.Name != nil {
		return s.Name.Name.Pos().Line
	}
	if s != nil {
		return s.Pos().Line
	}
	return -1
}

func (p *ThriftParser) getRealExceptionLine(e *parser.Exception) int {
	if e != nil && e.Name != nil && e.Name.Name != nil {
		return e.Name.Name.Pos().Line
	}
	if e != nil {
		return e.Pos().Line
	}
	return -1
}

func (p *ThriftParser) getRealUnionLine(u *parser.Union) int {
	if u != nil && u.Name != nil && u.Name.Name != nil {
		return u.Name.Name.Pos().Line
	}
	if u != nil {
		return u.Pos().Line
	}
	return -1
}

func (p *ThriftParser) getRealTypedefLine(t *parser.Typedef) int {
	if t != nil && t.Alias != nil && t.Alias.Name != nil {
		return t.Alias.Name.Pos().Line
	}
	if t != nil {
		return t.Pos().Line
	}
	return -1
}

func (p *ThriftParser) getRealConstLine(c *parser.Const) int {
	if c != nil && c.Name != nil && c.Name.Name != nil {
		return c.Name.Name.Pos().Line
	}
	if c != nil {
		return c.Pos().Line
	}
	return -1
}
