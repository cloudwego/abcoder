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

package java

import (
	"path/filepath"
	"strings"

	javaparser "github.com/cloudwego/abcoder/lang/java/parser"
	lsp "github.com/cloudwego/abcoder/lang/lsp"
	"github.com/cloudwego/abcoder/lang/uniast"
	sitter "github.com/smacker/go-tree-sitter"
)

type JavaSpec struct {
	repo    string
	rootMod *javaparser.ModuleInfo
	// 新增索引
	nameToMod map[string]*javaparser.ModuleInfo // 目录绝对路径 -> module 路径
	dirToPkg  map[string]JavaPkg                // 目录绝对路径 -> package 路径
}

func (c *JavaSpec) ProtectedSymbolKinds() []lsp.SymbolKind {
	// Java methods/vars are nested inside class ranges; keep them during Export filterLocalSymbols.
	return []lsp.SymbolKind{lsp.SKFunction, lsp.SKMethod, lsp.SKVariable, lsp.SKConstant}
}

type JavaPkg struct {
	Name string
	Path string
}

func NewJavaSpec(reop string) *JavaSpec {
	rootPomPath := filepath.Join(reop, "pom.xml")
	rootModule, err := javaparser.ParseMavenProject(rootPomPath)
	if err != nil {
		return &JavaSpec{
			repo:      reop,
			rootMod:   rootModule,
			nameToMod: make(map[string]*javaparser.ModuleInfo),
			dirToPkg:  make(map[string]JavaPkg),
		}
	}
	nameToMod := javaparser.GetModuleStructMap(rootModule)

	return &JavaSpec{
		repo:      reop,
		rootMod:   rootModule,
		nameToMod: nameToMod,
		dirToPkg:  make(map[string]JavaPkg),
	}

}

func (c *JavaSpec) FileImports(content []byte) ([]uniast.Import, error) {
	//   Java import parsing by tree-setting
	panic("Java import parsing by tree-setting")
}

func (c *JavaSpec) WorkSpace(root string) (map[string]string, error) {
	rets := javaparser.GetModuleMap(c.rootMod)
	return rets, nil
}

func (c *JavaSpec) PathToMod(path string) *javaparser.ModuleInfo {

	var maxPathmatchMods *javaparser.ModuleInfo

	for _, modInfo := range c.nameToMod {
		if strings.HasPrefix(path, modInfo.Path) {
			if maxPathmatchMods == nil {
				maxPathmatchMods = modInfo
			} else if len(modInfo.Path) > len(maxPathmatchMods.Path) {
				maxPathmatchMods = modInfo
			}
		}
	}
	return maxPathmatchMods
}

func (c *JavaSpec) NameSpace(path string, file *uniast.File) (string, string, error) {
	if !strings.HasPrefix(path, c.repo) {
		// External library
		return "external", "external", nil
	}

	modName := ""
	modInfo := c.PathToMod(path)
	if modInfo != nil {
		modName = modInfo.Coordinates
	}
	return modName, file.Package, nil
}

func (c *JavaSpec) ShouldSkip(path string) bool {
	// UT 文件不处理
	return !strings.HasSuffix(path, ".java") || c.IsTest(path) || c.IsTarget(path)
}

func (c *JavaSpec) IsTest(path string) bool {
	for _, moduleInfo := range c.nameToMod {
		if strings.HasPrefix(path, moduleInfo.TestSourcePath) {
			return true
		}
	}
	return false
}
func (c *JavaSpec) IsTarget(path string) bool {
	for _, moduleInfo := range c.nameToMod {
		if strings.HasPrefix(path, moduleInfo.TargetPath) {
			return true
		}
	}
	return false
}

func (c *JavaSpec) IsDocToken(tok lsp.Token) bool {
	return tok.Type == "comment"
}

func (c *JavaSpec) DeclareTokenOfSymbol(sym lsp.DocumentSymbol) int {
	for i, t := range sym.Tokens {
		if c.IsDocToken(t) {
			continue
		}
		for _, m := range t.Modifiers {
			if m == "declaration" {
				return i
			}
		}
	}
	return -1
}

func (c *JavaSpec) IsEntityToken(tok lsp.Token) bool {
	// TODO: refine for Java
	return tok.Type == "class_declaration" || tok.Type == "interface_declaration" || tok.Type == "method_declaration" || tok.Type == "static_method_invocation" || tok.Type == "method_invocation"
}

func (c *JavaSpec) IsStdToken(tok lsp.Token) bool {
	// TODO: implement for Java std lib
	return tok.Type == "generic_type" || tok.Type == "interface_declaration" || tok.Type == "method_declaration"
}

func (c *JavaSpec) TokenKind(tok lsp.Token) lsp.SymbolKind {
	return NodeTypeToSymbolKind(tok.Type)
}

func (c *JavaSpec) IsMainFunction(sym lsp.DocumentSymbol) bool {
	// A simple heuristic for Java main method
	return (sym.Kind == lsp.SKMethod || sym.Kind == lsp.SKFunction) && sym.Name == "main(String[])"
}

func (c *JavaSpec) IsEntitySymbol(sym lsp.DocumentSymbol) bool {
	typ := sym.Kind
	return typ == lsp.SKMethod || typ == lsp.SKFunction || typ == lsp.SKVariable || typ == lsp.SKClass || typ == lsp.SKInterface || typ == lsp.SKEnum
}

func (c *JavaSpec) IsPublicSymbol(sym lsp.DocumentSymbol) bool {
	// 使用tree-sitter节点获取modifiers字段，支持类、接口、方法、字段等各种符号类型
	if sym.Node == nil {
		return false
	}

	// 根据不同符号类型，查找对应的modifiers节点
	var modifiersNode *sitter.Node

	// 处理不同类型的Java符号
	switch sym.Kind {
	case lsp.SKClass, lsp.SKInterface, lsp.SKEnum:
		// 类、接口、枚举声明
		modifiersNode = sym.Node.ChildByFieldName("modifiers")
		if modifiersNode == nil {
			// 尝试从父节点获取modifiers
			if sym.Node.Type() == "class_declaration" ||
				sym.Node.Type() == "interface_declaration" ||
				sym.Node.Type() == "enum_declaration" {
				modifiersNode = sym.Node.ChildByFieldName("modifiers")
			}
		}

	case lsp.SKMethod, lsp.SKConstructor:
		// 方法、构造函数
		modifiersNode = sym.Node.ChildByFieldName("modifiers")
		if modifiersNode == nil && sym.Node.Type() == "method_declaration" {
			modifiersNode = sym.Node.ChildByFieldName("modifiers")
		}

	case lsp.SKVariable, lsp.SKField:
		// 字段、变量
		modifiersNode = sym.Node.ChildByFieldName("modifiers")
		if modifiersNode == nil && sym.Node.Type() == "field_declaration" {
			modifiersNode = sym.Node.ChildByFieldName("modifiers")
		}

	default:
		// 其他类型，尝试通用方式
		modifiersNode = sym.Node.ChildByFieldName("modifiers")
	}

	// 如果找到modifiers节点，检查是否包含public
	if modifiersNode != nil {
		// 遍历所有modifier子节点
		for i := 0; i < int(modifiersNode.ChildCount()); i++ {
			modifier := modifiersNode.Child(i)
			if modifier != nil && modifier.Type() == "modifier" {
				modifierText := modifier.Content([]byte(sym.Text))
				if strings.Contains(strings.ToLower(modifierText), "public") {
					return true
				}
			}
		}

		// 直接检查modifiers节点文本
		modifiersText := modifiersNode.Content([]byte(sym.Text))
		return strings.Contains(strings.ToLower(modifiersText), "public")
	}

	// 如果没有modifiers节点，检查整个符号文本
	// 处理一些特殊情况，如接口方法默认public
	symbolText := strings.ToLower(sym.Text)

	// 接口中的方法默认是public的
	if sym.Kind == lsp.SKMethod && strings.Contains(symbolText, "interface") {
		return true
	}

	// 检查是否包含public关键字
	return strings.Contains(symbolText, "public")
}

func (c *JavaSpec) HasImplSymbol() bool {
	// For Java `implements` and `extends`
	return false
}

func (c *JavaSpec) ImplSymbol(sym lsp.DocumentSymbol) (int, int, int) {
	// Java中的继承和接口实现关系识别
	return -1, -1, -1
}

func (c *JavaSpec) FunctionSymbol(sym lsp.DocumentSymbol) (int, []int, []int, []int) {
	// TODO: Implement for Java
	return -1, nil, nil, nil
}

func (c *JavaSpec) GetUnloadedSymbol(from lsp.Token, define lsp.Location) (string, error) {
	return "", nil
}

// NodeTypeToSymbolKind maps a tree-sitter node type to the corresponding LSP SymbolKind.
// The mapping is based on the official LSP specification and the tree-sitter-java grammar.
// Ref: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#symbolKind
func NodeTypeToSymbolKind(nodeType string) lsp.SymbolKind {
	switch nodeType {
	case "class_declaration":
		return lsp.SKClass
	case "method_declaration":
		return lsp.SKMethod
	case "constructor_declaration":
		return lsp.SKConstructor
	case "field_declaration":
		return lsp.SKField
	case "enum_declaration":
		return lsp.SKEnum
	case "enum_constant":
		return lsp.SKEnumMember
	case "interface_declaration", "super_interfaces":
		return lsp.SKInterface
	case "annotation_type_declaration":
		// Annotations are a form of interface in Java.
		return lsp.SKInterface
	case "module_declaration":
		return lsp.SKModule
	case "package_declaration":
		return lsp.SKPackage
	case "variable_declarator":
		// This can be a local variable or a field. Context is needed to be more specific.
		// Defaulting to SKVariable.
		return lsp.SKVariable
	case "type_parameter":
		return lsp.SKTypeParameter
	// Add more mappings as needed for other node types.
	case "type_identifier":
		return lsp.SKClass
	case "generic_type":
		return lsp.SKTypeParameter
	default:
		return lsp.SKUnknown
	}
}
