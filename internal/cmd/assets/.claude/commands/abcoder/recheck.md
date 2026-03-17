---
name: ABCoder: Recheck
description: Validate CODE_TASK technical feasibility with critical analysis using mcp__abcoder.
category: ABCoder
tags: [abcoder, recheck, validation]
---
从原始需求出发，使用 mcp__abcoder 进行批判性分析，以验证 CODE_TASK 的技术可行性。

<!-- ABCODER:START -->
**Guardrails**
- 批判性思考，保持诚实：确保方案可实现需求、无技术风险、最大化复用已有功能、最小化改动。
- 严格使用 `mcp__abcoder` 验证技术细节，禁止假设。
- 下钻到 `mcp__abcoder__get_ast_node` 粒度进行验证。

**Steps**
Track these steps as TODOs and complete them one by one.
1. 从 `mcp__abcoder__get_repo_structure` 开始，获取目标仓库结构。
2. 定位相关的 package 和 node。
3. 使用 `mcp__abcoder__get_ast_node` 深入分析相关代码节点，验证方案的可行性。
4. 检查方案是否可以实现 CODE_TASK 中的所有需求。
5. 识别并报告任何潜在的技术风险。
6. 验证是否最大化复用了已有功能，是否最小化了改动。
7. 如发现问题，提出具体的修改建议或后续问题。
8. 总结分析结果，明确指出方案是否可以执行。

**Reference**
- `mcp__abcoder__list_repos` - 列出所有可用仓库
- `mcp__abcoder__get_repo_structure` - 获取仓库结构（必须作为第一步）
- `mcp__abcoder__get_package_structure` - 获取 package 结构
- `mcp__abcoder__get_file_structure` - 获取文件结构
- `mcp__abcoder__get_ast_node` - 获取 AST 节点详情（下钻验证）
<!-- ABCODER:END -->
