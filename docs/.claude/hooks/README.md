# ABCoder Hooks

Claude Code 钩子系统，用于自动化 ABCoder 代码分析工作流。

## 配置文件

`settings.json` 定义了两个钩子事件：

- **PreToolUse**: 在工具调用前触发
- **PostToolUse**: 在工具调用后触发

## 钩子说明

### 1. parse.sh (PreToolUse)

**匹配工具**: `get_repo_structure`, `get_file_structure`, `get_package_structure`, `get_ast_node`

**功能**: 自动检测项目语言并生成最新 AST 到 `~/.asts/` 目录（ `~/.claude.json` 需要配置abcoder目录 `abcoder mcp ${HOME}/.asts` ）

- 自动检测 Go/TypeScript 项目
- 执行 `abcoder parse go/ts . -o ~/.asts/repo.json` 生成 AST 文件

**检测规则**:
- Go: 存在 `go.mod` 或 `main.go`
- TypeScript: 存在 `package.json`, `tsconfig.json` 或 `.ts/.tsx` 文件

### 2. prompt.sh (PostToolUse)

**匹配工具**: `list_repos`

**功能**: 显示 ABCoder 工作流程 SOP

在列出仓库后，自动提示用户遵循正确的分析流程。

### 3. reminder.sh (PostToolUse)

**匹配工具**: `get_repo_structure`, `get_package_structure`, `get_file_structure`

**功能**: 提醒递归调用 `get_ast_node`

在定位目标节点后，提醒用户必须使用 `get_ast_node` 递归获取完整的节点信息（类型、代码、位置、依赖、引用等）。

## ABCoder 工作流 SOP

```
1. 问题分析
   └── list_repos (确认 repo_name)

2. 代码定位 (repo -> package -> node -> ast)
   ├── get_repo_structure (定位 package)
   ├── get_package_structure (定位 node)
   └── get_ast_node (递归获取节点关系)

3. 自我反思
   └── sequential_thinking (理解调用链和上下文)
```

## AST 层次结构

| 层级 | 标识 | 示例 |
|------|------|------|
| Module | mod_path | github.com/cloudwego/kitex |
| Package | pkg_path | github.com/cloudwego/kitex/pkg/generic |
| File | file_path | pkg/generic/closer.go |
| AST Node | {mod_path, pkg_path, name} | {"mod_path": "...", "pkg_path": "...", "name": "Closer"} |

## 安装位置

```
~/.claude/hooks/abcoder/
├── parse.sh
├── prompt.sh
├── reminder.sh
└── abcoder-workflow.md
```
