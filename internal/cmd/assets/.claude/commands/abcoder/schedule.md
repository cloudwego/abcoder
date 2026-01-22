---
name: ABCoder: Schedule
description: Design implementation plan using mcp__abcoder analysis and code exploration.
category: ABCoder
tags: [abcoder, schedule, planning]
---
使用mcp__abcoder分析相关仓库（下钻到mcp__abcoder__get_ast_node查看细节），帮助用户设计实现方案。

<!-- ABCODER:START -->
**Guardrails**
- 最大化复用项目已有功能，避免重复造轮子。
- 优先采用直接、最小改动的实现方式，只有在用户明确要求时才增加复杂度。
- 严格限制修改影响面在所请求的结果范围内。
- 找出任何模糊或含糊不清的细节，并在修改文件前提出必要的后续问题。
- 在 Schedule 阶段禁止编写代码，禁止使用 agent。

**Steps**
Track these steps as TODOs and complete them one by one.
1. 从 `mcp__abcoder__get_repo_structure` 开始，获取目标仓库结构。
2. 根据任务描述，定位相关的 package。
3. 使用 `mcp__abcoder__get_package_structure` 获取 package 内的文件和节点列表。
4. 使用 `mcp__abcoder__get_ast_node` 深入分析相关代码节点，理解现有实现模式。
5. 分析依赖关系、调用链、类型信息等。
6. 设计实现方案，确保最大化复用已有功能、最小化改动。
7. 找出任何模糊或缺失的技术细节，并向用户提出后续问题。
8. 输出清晰的技术方案，包括修改范围、涉及的文件、关键实现步骤。

**Reference**
- `mcp__abcoder__list_repos` - 列出所有可用仓库
- `mcp__abcoder__get_repo_structure` - 获取仓库结构（必须作为第一步）
- `mcp__abcoder__get_package_structure` - 获取 package 结构
- `mcp__abcoder__get_file_structure` - 获取文件结构
- `mcp__abcoder__get_ast_node` - 获取 AST 节点详情（下钻分析）
<!-- ABCODER:END -->
