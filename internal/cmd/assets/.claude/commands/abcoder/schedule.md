---
name: ABCoder: Schedule
description: Design implementation plan using mcp__abcoder analysis and code exploration. Output SCHEDULE document for CODE_TASK decomposition.
category: ABCoder
tags: [abcoder, schedule, planning]
---
使用 mcp__abcoder 分析相关仓库（下钻到 mcp__abcoder__get_ast_node 查看细节），完成需求理解、技术上下文收集、方案设计和任务拆解，输出可直接拆解为 CODE_TASK 的 SCHEDULE 文档。

<!-- ABCODER:START -->

## Guardrails

- **最大化复用已有功能**：避免重复造轮子，优先采用直接、最小改动的实现方式
- **需求澄清优先**：在开始技术分析前，必须先理解需求并主动澄清模糊点
- **严格限制影响面**：所有修改限制在所请求的结果范围内
- **禁止编码和 agent**：Schedule 阶段只做分析和设计，严禁编写代码、使用 subAgent
- **输出必须可拆解**：SCHEDULE 文档必须包含完整规格，确保能直接拆解为可执行的 CODE_TASK

## 关键要求

### 需求理解阶段
- **需求提取**：从用户输入中提取核心需求关键词、技术栈、业务场景、功能边界
- **主动澄清**：使用 `AskUserQuestion` 工具主动提出关键决策问题，确保需求明确
- **需求确认**：输出需求摘要，与用户对齐后再进入技术分析阶段
- **需求留档**：将用户最终确认完毕后的需求保存到 `./task/{{MMDD}}/{{NAME}}__FEATURE_REQUEST.md`
   - 使用 `d=$(date +%m%d)` 获取当前日期（如 0127）
   - NAME 使用清晰的需求名称（如：添加用户认证、优化查询性能等）
   - 使用 `mkdir -p ./task/${d}` 确保目录存在
   - 如果用户的需求是一个完整的文档，在`./task/${d}/${NAME}__FEATURE_REQUEST.md`中记录其完整的绝对路径即可，无需重复复制内容。

### 上下文收集阶段
必须收集以下 6 类 context，确保实现信息完整：

1. **structural_context**（结构信息）
   - 通过 `get_repo_structure` 获取仓库整体结构
   - 通过 `get_package_structure` 定位相关 package 和文件
   - 输出：目标 repo、相关 packages、文件组织结构

2. **implementation_context**（实现细节）
   - 通过 `get_ast_node` 分析相关代码节点（**每批≤3个节点**，避免超限）
   - 分析依赖关系、调用链、类型信息
   - 输出：核心数据结构、计算逻辑、工具函数、接口定义

3. **pattern_context**（代码模式）
   - 搜索相似功能代码，理解项目最佳实践
   - 分析设计模式、编码规范、错误处理方式
   - 输出：可复用的代码模式、项目约定

4. **demo_code_context**（示例代码分析 - 🔴 关键步骤）
   - **扫描需求中所有demo代码的import语句**，提取所有依赖
   - 区分三类依赖：
     - A. **新增外部SDK**：需要go get安装（如Metrics Query SDK）
     - B. **内部工具库**：已在项目中使用，只需import
     - C. **标准库**：Go标准库，无需安装
   - **验证每个依赖在目标仓库中的存在性**：使用Grep搜索确认是否已引入
   - 输出：完整依赖清单（库名、是否已存在、引入方式、使用位置）

5. **sdk_definition_context**（SDK/API 定义）
   - 收集涉及的 SDK 方法定义（如有）
   - 收集外部 API 的 JSON/IDL 定义
   - 输出：SDK Method 签名、请求/响应格式

6. **config_context**（配置信息）
   - 收集数据库、中间件、环境配置信息
   - 输出：配置文件位置、关键配置项

### 任务拆解阶段
每个任务必须包含完整规格：

- **描述**：清晰的任务描述
- **前置依赖**：依赖的前置任务列表
- **输入规格**：函数参数、数据输入格式
- **输出规格**：返回值、数据输出格式
- **涉及文件**：具体文件路径和行号
- **关键函数/类**：涉及的核心代码元素
- **验证方式**：build（必需）、test、output、file_exists（按需添加）
- **三方依赖**：引入的三方库及引入方式

### 输出结构要求
SCHEDULE 文档必须采用以下标准结构，确保可直接拆解为 CODE_TASK：

```markdown
# [需求名称] 设计方案

## 1. 需求摘要
- **用户原始需求**：xxx
- **需求澄清记录**：xxx
- **最终需求确认**：xxx

## 2. 技术上下文分析

### 2.1 仓库结构
- **目标 repo**：xxx
- **相关 packages**：xxx

### 2.2 现有实现分析
- **相似功能定位**：xxx
- **核心数据结构**：xxx
- **关键工具函数**：xxx
- **SDK/API 定义**：xxx（如有）

### 2.3 依赖关系
- **三方库依赖**：完整依赖清单，格式如下：
  ```markdown
  | 依赖库 | 类型 | 是否已引入 | 引入方式 | 使用位置 |
  |--------|------|-----------|---------|---------|
  | code.byted.org/inf/metrics-query | 新增外部SDK | ❌ 否 | import mq "..." | Task 1.2 |
  | code.byted.org/lang/gg/gslice | 内部工具库 | ✅ 是 | import "..." | Task 1.2 (计算平均值) |
  | github.com/cloudwego/hertz | 内部工具库 | ✅ 是 | import "..." | HTTP框架 |
  ```
  - 🔴 **必须区分**：新增外部SDK vs 内部工具库
  - 每个依赖需包含：库名、是否已引入、引入方式、使用位置
- **内部模块依赖**：xxx

### 2.4 代码模式分析
- **可复用模式**：xxx
- **项目约定**：xxx

## 3. 实现方案设计

### 3.1 整体架构
- **架构说明**：xxx
- **核心组件**：xxx
- **调用链路**：xxx

### 3.2 关键实现细节
- **数据模型设计**：xxx
- **核心算法/逻辑**：xxx
- **接口设计**：xxx

## 4. 任务拆解（分阶段）

### Phase 1: MVP 核心功能

#### Task 1.1: [任务名称]
- **描述**：xxx
- **前置依赖**：无
- **输入规格**：xxx
- **输出规格**：xxx
- **涉及文件**：`xxx:行号`
- **关键函数/类**：xxx
- **验证方式**：build/test/manual
- **三方依赖**：xxx

### Phase 2: 完善功能

#### Task 2.1: [任务名称]
- **描述**：xxx
- **前置依赖**：Task 1.1
- **输入规格**：xxx
- **输出规格**：xxx
- **涉及文件**：`xxx:行号`
- **关键函数/类**：xxx
- **验证方式**：build/test/manual
- **三方依赖**：xxx

### Phase 3: 优化完善

#### Task 3.1: [任务名称]
- **描述**：xxx
- **前置依赖**：Task 2.1
- **输入规格**：xxx
- **输出规格**：xxx
- **涉及文件**：`xxx:行号`
- **关键函数/类**：xxx
- **验证方式**：build/test/manual
- **三方依赖**：xxx

## 5. 待确认问题
- **问题1**：xxx
- **问题2**：xxx

## 6. 质量检查清单
- [ ] 需求理解完整，已与用户确认
- [ ] 技术上下文收集充分（覆盖 6 类 context：structural, implementation, pattern, demo_code, sdk_definition, config）
- [ ] 🔴 demo_code_context已完整扫描需求中的所有import语句
- [ ] 🔴 三方依赖已分类（新增外部SDK / 内部工具库 / 标准库）
- [ ] 🔴 每个依赖已验证在目标仓库中的存在性（使用Grep确认）
- [ ] 每个任务都有明确的输入/输出规格
- [ ] 每个任务都有明确的验证标准（至少包含 build）
- [ ] 依赖关系清晰，无循环依赖
- [ ] 任务拆解粒度合理，每个任务可独立验证
```

## 执行步骤

按以下顺序执行，使用 TodoWrite 跟踪进度：

### Phase 1: 需求理解与澄清
1. **分析用户输入**：提取核心需求关键词、技术栈、业务场景、功能边界
2. **识别模糊点**：列出需要澄清的技术细节
3. **主动澄清**：使用 `AskUserQuestion` 工具提出关键决策问题
4. **输出需求摘要**：与用户对齐需求，确认后进入下一阶段

### Phase 2: 技术上下文收集
5. **获取仓库结构**：使用 `mcp__abcoder__get_repo_structure`（必须第一步）
6. **定位相关 package**：根据需求选择目标 package
7. **收集 structural_context**：使用 `get_package_structure` 获取 package 内文件和节点
8. **收集 implementation_context**：使用 `get_ast_node` 深入分析相关节点（**每批≤3个**，避免超限）
9. **收集 pattern_context**：搜索相似功能代码，分析项目模式
10. **收集 demo_code_context**（🔴 关键步骤）：
    - 扫描需求文档中所有demo代码的import语句
    - 使用正则表达式提取所有import路径
    - 区分依赖类型：外部SDK、内部工具库、标准库
    - 使用Grep验证每个依赖在目标仓库中的存在性
    - 记录依赖的使用位置和场景
11. **收集 sdk_definition_context**（如需要）：收集 SDK/API 定义
12. **收集 config_context**（如需要）：收集配置信息

### Phase 3: 方案设计
13. **分析依赖关系**：梳理三方库依赖和内部模块依赖（整合demo_code_context和implementation_context的依赖信息）
14. **设计整体架构**：明确核心组件、调用链路
15. **设计关键实现**：数据模型、核心算法、接口设计

### Phase 4: 任务拆解
16. **分阶段拆解**：按 MVP → 完善阶段 → 优化阶段拆解
17. **细化每个任务**：为每个任务填写完整规格
18. **梳理依赖关系**：确保任务依赖清晰、无循环依赖

### Phase 5: 文档输出与质量检查
19. **保存 SCHEDULE 文档**：保存到 `./task/{{MMDD}}/{{NAME}}__SCHEDULE.md`
   - 使用 `d=$(date +%m%d)` 获取当前日期（如 0127）
   - NAME 使用清晰的需求名称（如：添加用户认证、优化查询性能等）
   - 使用 `mkdir -p ./task/${d}` 确保目录存在
20. **执行质量检查**：对照质量检查清单逐项检查（重点检查：三方依赖声明完整性）
21. **用户确认**：与用户确认 SCHEDULE 文档，确保可拆解为 CODE_TASK

## Reference

### ABCoder 工具链
- `mcp__abcoder__list_repos` - 列出所有可用仓库
- `mcp__abcoder__get_repo_structure` - 获取仓库结构（**必须第一步**）
- `mcp__abcoder__get_package_structure` - 获取 package 结构
- `mcp__abcoder__get_file_structure` - 获取文件结构
- `mcp__abcoder__get_ast_node` - 获取 AST 节点详情（**每批≤3个节点**）

### 辅助工具
- `AskUserQuestion` - 主动澄清需求
- `TodoWrite` - 跟踪执行进度
- `Grep` - 搜索代码模式
- `Glob` - 查找文件

<!-- ABCODER:END -->
