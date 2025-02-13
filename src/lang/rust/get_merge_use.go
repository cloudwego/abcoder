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

package rust

import (
	"bufio"
	"fmt"
	"strings"
)

// UseNode represents a module node in the dependency tree
type UseNode struct {
	Name     string
	Children []*UseNode
}

func ParseUseStatements(fileContent string) ([]string, error) {
	file := strings.NewReader(fileContent)
	var useStatements []string
	var currentStatement strings.Builder
	scanner := bufio.NewScanner(file)

	inUseBlock := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "use ") || inUseBlock {
			inUseBlock = true
			currentStatement.WriteString(line)
			if len(line) > 0 && line[len(line)-1] == ';' {
				useStatements = append(useStatements, currentStatement.String())
				currentStatement.Reset()
				inUseBlock = false
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return useStatements, nil
}

func BuildDependencyTree(useStatements []string) *UseNode {
	root := &UseNode{Name: "root"}
	stmts := []string{}
	for _, s := range useStatements {
		ss := splitUseStatement(s)
		stmts = append(stmts, ss...)
	}
	for _, stmt := range stmts {
		stmt = strings.TrimPrefix(stmt, "use ")
		stmt = strings.TrimSuffix(stmt, ";")
		parseAndAddModules(root, stmt)
	}
	return root
}

func parseAndAddModules(root *UseNode, stmt string) {
	parts := strings.Split(stmt, "::")
	current := root

	for i := 0; i < len(parts); i++ {
		part := parts[i]
		// Regular module, just navigate down the tree
		current = getOrCreateChild(current, part)
	}
}

func getOrCreateChild(parent *UseNode, name string) *UseNode {
	for _, child := range parent.Children {
		if child.Name == name {
			return child
		}
	}

	newNode := &UseNode{Name: name}
	parent.Children = append(parent.Children, newNode)
	return newNode
}

func splitUseStatement(useStmt string) []string {
	// Remove the prefix "use " and the suffix ";"
	useStmt = strings.TrimPrefix(useStmt, "use ")
	useStmt = strings.TrimSuffix(useStmt, ";")

	openBracePos := strings.Index(useStmt, "{")
	closeBracePos := strings.LastIndex(useStmt, "}")

	// 没有花括号的case
	if openBracePos == closeBracePos {
		return []string{useStmt}
	}
	if openBracePos < 0 || closeBracePos < 0 {
		return []string{useStmt}
	}

	basePath := strings.TrimSpace(useStmt[:openBracePos])
	modulesStr := useStmt[openBracePos+1 : closeBracePos]
	modules := splitModules(modulesStr)

	var simpleUseStmts []string
	for _, module := range modules {
		simpleUseStmts = append(simpleUseStmts, generateSimpleUseStatements(basePath, module)...)
	}

	return simpleUseStmts
}

func splitModules(modulesStr string) []string {
	var modules []string
	var currentModule strings.Builder
	bracesLevel := 0

	for _, char := range modulesStr {
		if char == '{' {
			bracesLevel++
		}
		if char == '}' {
			bracesLevel--
		}
		if char == ',' && bracesLevel == 0 {
			modules = append(modules, strings.TrimSpace(currentModule.String()))
			currentModule.Reset()
		} else {
			currentModule.WriteRune(char)
		}
	}

	if currentModule.Len() > 0 {
		modules = append(modules, strings.TrimSpace(currentModule.String()))
	}

	return modules
}

func generateSimpleUseStatements(basePath, module string) []string {
	var simpleUseStmts []string

	if strings.Contains(module, "{") {
		//	 Handle nested modules
		openBracePos := strings.Index(module, "{")
		closeBracePos := strings.LastIndex(module, "}")
		if closeBracePos < 0 || openBracePos+1 >= closeBracePos {
			return simpleUseStmts
		}
		newBasePath := fmt.Sprintf("%s%s", basePath, strings.TrimSpace(module[:openBracePos]))
		nestedModulesStr := module[openBracePos+1 : closeBracePos]
		nestedModules := splitModules(nestedModulesStr)
		for _, nestedModule := range nestedModules {
			simpleUseStmts = append(simpleUseStmts, generateSimpleUseStatements(newBasePath, nestedModule)...)
		}
	} else {
		// Regular module
		simpleUseStmts = append(simpleUseStmts, fmt.Sprintf("use %s%s;", basePath, module))
	}

	return simpleUseStmts
}

func ConvertTreeToUse(node *UseNode, prefix string) []string {
	if node == nil {
		return nil
	}

	newPrefix := node.Name
	if prefix != "" {
		newPrefix = prefix + "::" + node.Name
	}

	if len(node.Children) == 0 {
		return []string{"use " + newPrefix + ";"}
	}

	var childStatements []string
	for _, child := range node.Children {
		childStatements = append(childStatements, ConvertTreeToUse(child, newPrefix)...)
	}

	return childStatements
}

func GetAndMergeUse(fileContents []string) ([]string, error) {
	// 解析出所有文件中的 use 声明
	var useStatements []string
	for _, fc := range fileContents {
		us, err := ParseUseStatements(fc)
		if err != nil {
			return nil, fmt.Errorf("error parsing use statements: %v", err)
		}
		useStatements = append(useStatements, us...)
	}

	// 将所有的 use 声明构建成依赖树，从而实现去重效果
	dependencyTree := BuildDependencyTree(useStatements)

	// 获取最终改 mod 下的所有 use 声明(去重且唯一)
	var ret []string
	for _, r := range dependencyTree.Children {
		uses := ConvertTreeToUse(r, "")
		ret = append(ret, uses...)
	}

	return ret, nil
}

func GetRustContentDefine(name, fileContent string) (string, error) {
	if len(fileContent) == 0 {
		return "", nil
	}
	scanner := bufio.NewScanner(strings.NewReader(fileContent))
	useCounts := 0
	for scanner.Scan() {
		line := scanner.Text()
		trimLine := strings.TrimSpace(line)
		// 处理use语句块
		if strings.HasPrefix(trimLine, "use ") { // bugfix: 匹配 "use "避免误匹配
			useCounts++
			if strings.HasSuffix(trimLine, "{") {
				bracketCount := 1
				for bracketCount > 0 && scanner.Scan() {
					nextLine := scanner.Text()
					bracketCount += strings.Count(nextLine, "{")
					bracketCount -= strings.Count(nextLine, "}")
				}
			}
		}
	}

	scanner = bufio.NewScanner(strings.NewReader(fileContent))
	var buffer strings.Builder
	if useCounts == 0 {
		for scanner.Scan() {
			_, err := buffer.WriteString(scanner.Text() + "\n")
			if err != nil {
				fmt.Printf("[GetRustContentDefine]scan err: %s\n", err)
			}
		}

		return buffer.String(), nil
	}
	for useCounts > 0 && scanner.Scan() {
		line := scanner.Text()
		trimLine := strings.TrimSpace(line)
		// 处理use语句块
		if strings.HasPrefix(trimLine, "use ") {
			useCounts--
			if strings.HasSuffix(trimLine, "{") {
				bracketCount := 1
				for bracketCount > 0 && scanner.Scan() {
					nextLine := scanner.Text()
					bracketCount += strings.Count(nextLine, "{")
					bracketCount -= strings.Count(nextLine, "}")
				}
			}
		}
		if useCounts == 0 {
			for scanner.Scan() {
				_, err := buffer.WriteString(scanner.Text() + "\n")
				if err != nil {
					fmt.Printf("[GetRustContentDefine]scan err: %s\n", err)
				}
			}
		}
	}

	return buffer.String(), nil
}
