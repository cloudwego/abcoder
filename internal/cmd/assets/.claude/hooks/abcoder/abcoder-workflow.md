# ABCoder代码分析工作流

## AST 层次结构
- **模块(Module)**: 仓库中的编译单元，由 `mod_path` 标识；例如: "github.com/cloudwego/kitex"
- **包(Package)**: 符号的命名空间，由 `pkg_path` 标识；例如: "github.com/cloudwego/kitex/pkg/generic"
- **文件(File)**: 代码文件，由 `file_path` 标识；例如: "pkg/generic/closer.go"
- **AST节点**: 语法单元(函数、类型、变量)，由 `NodeID` 标识；例如:
```
{
  "mod_path": "github.com/cloudwego/kitex",
  "pkg_path": "github.com/cloudwego/kitex/pkg/generic",
  "name": "Closer"
}
```

## ABCoder工作SOP
1. **问题分析**:
   - 基于用户问题分析相关关键词
   - 必须使用 `list_repos` 确认repo_name

2. **代码定位** (repo→package→node→ast node relationship):
   - 2.1 **定位package**: 基于 `get_repo_structure` 返回的package list选择目标package
   - 2.2 **定位文件**: 通过 `get_package_structure` 返回的file信息，确认目标文件
   - 2.3 **定位节点**: 通过 `get_files_structure` 返回的node信息，确认目标节点
   - 2.4 **确认node详情**: 递归调用 `get_ast_node` 获取node详细（dependencies, references, inheritance, implementation, grouping）

3. **自我反思**:
   - 理解完整的code calling-chain、contextual-relationship
   - 如果无法清楚解释机制，使用`sequential_thinking` 来帮助分解问题并记录信息，调整选择并重复步骤2

## 注意事项
- 回复应列出相关代码的准确 metadata，包括 AST node（或 package）的 identity、file location 和代码。**必须提供确切的 file location（包括 line numbers）！**
