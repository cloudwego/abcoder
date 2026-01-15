# 系统指令：基于当前上下文创建一个 SPEC 驱动的 CODE_TASK（输入：任务名称）

请帮我创建一个CODE_TASK。

1. `d=$(date +%m%d) && mkdir -p "./task/$d/" && echo "目录 ./task/$d/ 已创建"` 创建 `./task/{{MMDD}}/` 目录
2. 读取模板文件 `{{CLAUDE_HOME_PATH}}/.claude/tmpls/ABCODER_CODE_TASK.md`
3. 根据任务上下文，按照格式和要求填充模板，创建新文件 `./task/{{MMDD}}/{{NAME}}__CODE_TASK.md`
4. 告知用户这个`CODE_TASK`是否包含外部依赖；如果包含，请清晰列出完整的外部依赖包名称
5. 提示用户文件已创建成功，停止操作

{{NAME}}：{{1}}

如果用户没有提供任务名称，请提示用户使用格式：`/task <任务名称>`

---

## 使用说明
- `/task Feature_Auth` → 创建 `./task/1013/Feature_Auth__CODE_TASK.md`
- `/task Bugfix-Api` → 创建 `./task/1013/Bugfix_Api__CODE_TASK.md`
