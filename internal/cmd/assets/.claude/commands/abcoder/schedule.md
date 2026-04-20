---
name: ABCoder: Schedule
description: Design implementation plan using skill__abcoder analysis and code exploration.
category: ABCoder
tags: [abcoder, schedule, planning]
---

<!-- ABCODER:START -->
使用`skill__abcoder`分析相关仓库（下钻到`skill__abcoder__get_file_symbol`查看细节），帮助用户设计实现方案。

## Guardrails
IMPORTANT: 默认保持向后兼容；但是如果新需求和旧代码存在冲突，请你清晰具体告知，质询用户
IMPORTANT: 制定系统、根本性的解决方案
- 最大化复用项目已有功能、优先使用外部成熟库；不重复造轮子。
- 学习项目/外部库已有最佳实践；保持风格一致。
- 优先采用直接、最小改动的实现方式，只有在用户明确要求时才增加复杂度。
- 严格限制修改影响面在必要但**全面**范围。
- 找出任何模糊或含糊不清的细节，并在`abcoder:spec`前提出必要的后续问题。
- 在Schedule阶段禁止编写代码，禁止使用agent。
IMPORTANT: MUST 从`skill__abcoder__list_repos`开始, 下钻到`skill__abcoder__get_file_symbol`

在开始任何分析前，先问自己：
1. "这是个真问题还是臆想出来的？" - 拒绝过度设计
2. "有更简单的方法吗？" - 永远寻找最简方案  
3. "会破坏什么吗？" - 向后兼容是铁律

结构化问题分解思考
   第一层：数据结构分析. "Bad programmers worry about the code. Good programmers worry about data structures."
   - 核心数据IDL是什么？它们的关系如何？类型是否兼容
   - 数据流向哪里？谁拥有它？谁修改它？
   - 有没有不必要的数据复制或转换？
   第二层：特殊情况识别. "好代码没有特殊情况"
   - 找出所有 if/else 分支
   - 哪些是真正的业务逻辑？哪些是糟糕设计的补丁？
   - 能否重新设计数据结构来消除这些分支？
   第三层：复杂度审查. "如果实现需要超过3层缩进，重新设计它"
   - 这个功能的本质是什么？（一句话说清）
   - 当前方案用了多少概念来解决？
   - 能否减少到一半？再少一半？
   第四层：破坏性分析. "Never break userspace" - 向后兼容是铁律
   - 列出所有可能受影响的现有功能
   - 哪些依赖会被破坏？
   - 如何在不破坏任何东西的前提下改进？
   第五层：实用性验证. "Theory and practice sometimes clash. Theory loses. Every single time."
   - 这个问题在生产环境真实存在吗？
   - 有多少用户真正遇到这个问题？
   - 解决方案的复杂度是否与问题的严重性匹配？  

## Style
- 面向E2E用户，隐藏实现细节（一句话总结）
- 仅暴露必要的SDK/API/Method出入参数
- 保持简洁, 回复保持在500字以内；除非用户明确要求，不要包含代码函数体，仅透出signature、IDL数据流向

**Steps**
Track these steps as TODOs and complete them one by one.
1. 从 `skill__abcoder__tree_repo` 开始，获取目标仓库结构。
2. 根据任务描述，生成pattern, `skill__abcoder__search_symbol` 定位相关的 symbol。
3. 使用 `skill__abcoder__get_file_structure` 获取 file 所有 symbol。
4. 使用 `skill__abcoder__get_file_symbol` 获取 symbol 源代码/dependence/reference, 使用depend/refer的file-path和name, 继续调用 `get_file_symbol`, 追溯数据流向。
5. 分析依赖关系、数据流向调用链、类型信息等。
6. 设计实现方案，确保最大化复用已有功能、最小化改动。
7. 找出任何模糊或缺失的技术细节，并向用户提出后续问题。
8. 输出清晰的技术方案，包括修改范围、涉及的文件、关键实现步骤。

**Reference**
- `skill__abcoder__list_repos` - 列出所有可用仓库
- `skill__abcoder__tree_repo` - 获取仓库结构（必须作为第一步）
- `skill__abcoder__search_symbol` - 搜索仓库的相关symbol(支持regex pattern)
- `skill__abcoder__get_file_structure` - 获取 file 的所有 symbol
- `skill__abcoder__get_file_symbol` - 获取 symbol 的 源代码/dependence/reference
<!-- ABCODER:END -->
