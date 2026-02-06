---
name: ABCoder: Task
description: Create a SPEC-driven CODE_TASK document from task context.
category: ABCoder
tags: [abcoder, task, creation]
---
根据任务上下文，或者用户给出的 SCHEDULE 文件，创建一个由 SPEC 驱动的 CODE_TASK 文档。

<!-- ABCODER:START -->
**Guardrails**
- 必须提供任务名称，格式为 `/abcoder:task <任务名称>`。若未提供，根据任务上下文推荐一个名词（确保清晰、简洁）。
- 文件路径使用 `./task/{{MMDD}}/{{NAME}}__CODE_TASK.md` 格式。
- 严格遵循 CODE_TASK 模板格式和要求。
- 确保 CODE_TASK 中子任务的上下文完备，每一个```task```定义必须包含完整的任务信息、依赖、文件、上下文、实现要求等，确保每一个子任务可以独立执行。
- 创建完成后停止操作，不进行额外工作。

**Steps**
Track these steps as TODOs and complete them one by one.
1. 读取模板文件 `{{CLAUDE_HOME_PATH}}/.claude/tmpls/ABCODER_CODE_TASK.md`。
2. 根据任务上下文和名称，按照模板格式填充内容，生成新文件 `./task/{{MMDD}}/{{NAME}}__CODE_TASK.md`。
3. 针对每个task，逐个校验其context是否完备，类型上：ast_node|file|pattern|dependency_context|sdk_definition|config 无遗漏。对于 List 类型，校验是否包含所有必要的信息。

**Reference**
- 模板文件：`{{CLAUDE_HOME_PATH}}/.claude/tmpls/ABCODER_CODE_TASK.md`
- 示例：
  - `/abcoder:task Feature_Auth` → 创建 `./task/1013/Feature_Auth__CODE_TASK.md`
  - `/abcoder:task Bugfix-Api` → 创建 `./task/1013/Bugfix_Api__CODE_TASK.md`
<!-- ABCODER:END -->
