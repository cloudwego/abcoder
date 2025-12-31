# Claude Code Commands

Claude Code 斜杠命令定义，用于规范化和自动化开发工作流。

## 命令说明

### /task - 创建编码任务

创建 CODE_TASK 文档，规范化编码需求描述。

**执行流程**:
1. 创建 `./task/{{MMDD}}/` 目录
2. 读取模板 `~/.claude/tmpls/CODE_TASK.md`
3. 根据任务上下文填充模板，生成 `./task/{{MMDD}}/{{NAME}}__CODE_TASK.md`
4. 列出外部依赖包（如有）
5. 提示创建成功

**使用示例**:
```
/task Feature_Auth     → 创建 ./task/1231/Feature_Auth__CODE_TASK.md
/task Bugfix-Api       → 创建 ./task/1231/Bugfix_Api__CODE_TASK.md
```

**CODE_TASK 格式要求**:
- action: create/modify/delete
- 文件路径准确
- 涉及 SDK 时指定 Package/Method 名称
- 涉及 curl 时提供具体命令（请求头、请求体、URL、响应结构）
- 提供具体`get_ast_node`入参、验证方法

---

### /schd - 设计实现方案

使用 mcp__abcoder 分析代码库，设计技术实现方案。

**执行流程**:
1. 从 `mcp__abcoder__get_repo_structure` 开始
2. 下钻到 `mcp__abcoder__get_ast_node` 查看细节
3. 分析依赖关系和调用链
4. 设计最小改动、最大化复用的方案

**Guardrails**:
- 最大化复用已有功能
- 优先采用最小改动方式
- 限制修改影响面
- 找出模糊细节并提出问题
- 禁止编写代码，禁止使用 agent

---

### /recheck - 技术方案核对

批判性检查 CODE_TASK 的技术可行性。

**执行流程**:
1. 从 `mcp__abcoder__get_repo_structure` 开始
2. 使用 mcp__abcoder 核对技术细节
3. 下钻到 `mcp__abcoder__get_ast_node` 粒度验证

**检查项**:
- 方案是否可实现需求
- 是否存在技术风险
- 是否最大化复用已有功能
- 是否最小化改动

## 模板文件

### CODE_TASK.md

编码任务模板，定义任务清单格式。

**核心要素**:
```
1. [ ] 任务描述
   - file: 文件路径
   - action: create/modify/delete
   - 技术规格（SDK Method / curl JSON / IDL结构 / `get_ast_node` 调用参数）
```

**核心理念 - 直接使用原则**:
- 编写 CODE_TASK 时预先使用 `mcp__abcoder` 分析
- 提供完整的 `get_ast_node` 调用参数
- SubAgent 直接执行，无需重新分析

## 工作流示意

```
用户需求
    │
    ▼
/schd ──────────────→ 设计方案（abcoder分析）
    │                     │
    ▼                     ▼
/task ────────→ CODE_TASK（含技术规格）
    │                     │
    ▼                     ▼
/recheck ─────→ 方案核对（abcoder验证）
    │                     │
    ▼                     ▼
coding-executor ─→ 执行实现
```

## 文件位置

```
~/.claude/
├── commands/
│   ├── task.md
│   ├── schd.md
│   └── recheck.md
└── tmpls/
    ├── CODE_TASK.md
```
