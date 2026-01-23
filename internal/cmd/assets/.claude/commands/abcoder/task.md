---
name: ABCoder: Task
description: Create a SPEC-driven CODE_TASK document from task context.
category: ABCoder
tags: [abcoder, task, creation]
---
根据任务上下文创建一个由 SPEC 驱动的 CODE_TASK 文档。

<!-- ABCODER:START -->
**Guardrails**
- 必须提供任务名称，格式为 `/task <任务名称>`。若未提供，根据任务上下文推荐一个名词（确保清晰、简洁）。
- 文件路径使用 `./task/{{MMDD}}/{{NAME}}__CODE_TASK.md` 格式。
- 严格遵循 CODE_TASK 模板格式和要求。
- 清晰列出所有外部依赖包名称（如有）。
- 创建完成后停止操作，不进行额外工作。

**Steps**
Track these steps as TODOs and complete them one by one.
1. 验证用户提供了任务名称，如未提供则提示使用格式 `/task <任务名称>`。
2. 执行 `d=$(date +%m%d) && mkdir -p "./task/$d/"` 创建 `./task/{{MMDD}}/` 目录。
3. 读取模板文件 `{{CLAUDE_HOME_PATH}}/.claude/tmpls/ABCODER_CODE_TASK.md`。
4. 根据任务上下文和名称，按照模板格式填充内容，生成新文件 `./task/{{MMDD}}/{{NAME}}__CODE_TASK.md`。
5. 检查并清晰列出 CODE_TASK 包含的外部依赖包（如有）。
6. 验证生成的文件格式正确，包含所有必要字段。
7. 告知用户文件已创建成功，包含文件路径和外部依赖信息（如有），停止操作。

**Reference**
- 模板文件：`{{CLAUDE_HOME_PATH}}/.claude/tmpls/ABCODER_CODE_TASK.md`
- 示例：
  - `/task Feature_Auth` → 创建 `./task/1013/Feature_Auth__CODE_TASK.md`
  - `/task Bugfix-Api` → 创建 `./task/1013/Bugfix_Api__CODE_TASK.md`
<!-- ABCODER:END -->
