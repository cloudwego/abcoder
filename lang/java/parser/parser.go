package parser

import (
	"context"
	"log"
	"sync"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/cloudwego/abcoder/lang/uniast"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

var (
	once   sync.Once
	parser *sitter.Parser
)

func NewParser() *sitter.Parser {
	once.Do(func() {
		parser = sitter.NewParser()
		parser.SetLanguage(java.GetLanguage())
	})
	return parser
}

func GetLanguage(l uniast.Language) *sitter.Language {
	switch l {
	case uniast.Java:
		return java.GetLanguage()
	}
	return nil
}

func Parse(ctx context.Context, content []byte) (*sitter.Tree, error) {
	p := NewParser()
	tree, err := p.ParseCtx(ctx, nil, content)
	if err != nil {
		log.Printf("Error parsing content: %v", err)
		return nil, err
	}
	return tree, nil
}

func Utf8ToUtf16Position(content []byte, row, byteColumn uint32) (line, character int) {
	// 计算到指定行的起始位置
	lineStart := 0
	currentLine := uint32(0)

	for i := 0; i < len(content); {
		if currentLine == row {
			lineStart = i
			break
		}
		if content[i] == '\n' {
			currentLine++
		}
		i++
	}

	// 计算UTF-16字符位置
	utf16Pos := 0
	for i := lineStart; i < lineStart+int(byteColumn); {
		r, size := utf8.DecodeRune(content[i:])
		if r == utf8.RuneError {
			break
		}
		utf16Pos += utf16.RuneLen(r)
		i += size
	}

	return int(row), utf16Pos
}

func FindChildIdentifier(node *sitter.Node) *sitter.Node {
	var pkgNameNode *sitter.Node
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" || child.Type() == "scoped_identifier" {
			pkgNameNode = child
			break
		}
	}
	return pkgNameNode
}

func FindChildByType(node *sitter.Node, typeString string) *sitter.Node {
	var pkgNameNode *sitter.Node
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == typeString {
			pkgNameNode = child
			break
		}
	}
	return pkgNameNode
}
