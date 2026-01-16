# AST-Driven Coding 配置

Claude Code 的 AST 驱动开发配置，通过 MCP 工具、钩子和斜杠命令实现无幻觉的代码分析和精确执行。

## 目录结构

```
.claude/
├── CLAUDE.md          # 核心指令：AST-Driven Coder 角色定义
├── settings.json      # 钩子配置：PreToolUse/PostToolUse
├── hooks/             # 钩子脚本
│   ├── parse.sh       # 自动检测语言并生成 AST
│   ├── prompt.sh      # 显示工作流程 SOP
│   └── reminder.sh    # 提醒递归调用 get_ast_node
├── commands/          # 斜杠命令定义
│   ├── abcoder:task.md        # /abcoder:task - 创建编码任务
│   ├── abcoder:schedule.md        # /abcoder:schedule - 设计实现方案
│   └── abcoder:recheck.md     # /abcoder:recheck - 技术方案核对
└── tmpls/             # 文档模板
    └── CODE_TASK.md   # 编码任务模板
```

## 核心理念

**AST-Driven Coding**: 基于 UniAST + LSP 的无幻觉代码分析

| 原则 | 说明 |
|------|------|
| 绝不假设 | 不确定代码 MUST 通过 `mcp__abcoder__get_ast_node` 验证 |
| 代码分析优先级 | `mcp__abcoder` > Read/Search |
| 直接使用原则 | 预先分析提供完整上下文，SubAgent 直接执行无需重分析 |
| 分阶段开发 | MVP → 完善 → 优化 |

## MCP 工具

### mcp__abcoder

本地代码深度分析工具，提供四层 AST 结构：

```
list_repos → get_repo_structure → get_package_structure → get_file_structure → get_ast_node
     │              │                      │                       │                    │
     └── repo_name  └── mod/pkg list       └── file list           └── node list        └── dependencies/references/inheritance
```

**SOP 流程**:
1. 问题分析 → `list_repos` 确认 repo_name
2. 定位 package → `get_repo_structure` 选择目标 package
3. 定位 node → `get_package_structure` 确认目标 node
4. 获取关系 → `get_ast_node` 递归获取完整信息

### mcp__sequential_thinking

多步骤问题的系统化思考工具，用于复杂问题分解和模糊需求质询。

## 钩子系统

| 钩子 | 事件 | 匹配工具 | 作用 |
|------|------|----------|------|
| parse.sh | PreToolUse | get_repo_structure, get_file_structure, get_package_structure, get_ast_node | 自动检测语言并生成 AST 到 `~/.asts/` 目录 |
| prompt.sh | PostToolUse | list_repos | 显示 ABCoder 工作流程 SOP |
| reminder.sh | PostToolUse | get_repo_structure, get_package_structure, get_file_structure | 提醒递归调用 get_ast_node |

## 斜杠命令

### /abcoder:task <任务名称>

创建 CODE_TASK 文档，生成 `./task/{{MMDD}}/{{NAME}}__CODE_TASK.md`

**格式要求**:
- action: create/modify/delete
- 涉及 SDK: 指定 Package/Method 名称
- 涉及 curl: 提供完整命令和响应结构
- 提供具体验证方法

### /abcoder:schedule <任务描述>

使用 mcp__abcoder 设计实现方案

**Guardrails**:
- 最大化复用已有功能
- 优先最小改动
- 禁止编写代码、禁止使用 agent

### /abcoder:recheck <任务名称>

批判性检查 CODE_TASK 技术可行性

**检查项**:
- 方案可实现性
- 技术风险
- 代码复用和改动最小化

## 工作流

```
用户需求
    │
    ▼
/abcoder:schedule ──────────────→ 设计方案（abcoder分析）
    │                            │
    ▼                            ▼
/abcoder:task ────────→ CODE_TASK（含技术规格）
    │                            │
    ▼                            ▼
/abcoder:recheck ─────→ 方案核对（abcoder验证）
    │                            │
    ▼                            ▼
coding-executor ──────→ 执行实现
```

## 安装

1. 复制 `.claude/` 目录到项目根目录或用户主目录
2. 确保 `settings.json` 中的钩子路径正确
3. 确保 `~/.claude/tmpls/` 目录存在且包含模板文件

## 依赖

- Claude Code CLI
- abcoder MCP 服务器（提供 mcp__abcoder 工具）
- sequential-thinking MCP 服务器（提供 mcp__sequential_thinking 工具）

## 相关文档

- [hooks/README.md](../internal/cmd/assets/.claude/hooks/README.md) - 钩子系统详解
- [commands/README.md](../internal/cmd/assets/.claude/commands/README.md) - 斜杠命令详解
