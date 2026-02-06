# CODE_TASK 格式规范

## 设计原则

1. **显式依赖声明**：每个任务必须明确声明依赖的task_id
2. **结构化上下文指令**：使用标准化的语法指定需要收集的上下文
3. **明确验证标准**：每个任务必须有清晰的验证步骤
4. **可自动解析**：所有内容使用标准化格式，便于Parser提取

## 文件结构

```markdown
# [任务标题]

## 任务概述
对整个需求的简要描述

## 全局配置
```yaml
repo_name: {{REPO_NAME}}
max_retries: 3
execution_mode: auto  # auto | interactive
```

## 任务定义

### 任务组：阶段名称
```task
id: task_1_1
title: 任务标题
stage: 1
priority: high  # high | medium | low

depends_on: []  # 显式声明依赖的task_id列表

files:
  - path: pkg/clients/metrics/client.go
    action: modify  # create | modify | delete
    description: 添加Metrics SDK Client初始化方法

context:
  - type: ast_node  # ast_node | file | pattern | dependency_context | sdk_definition | config
    target: pkg/clients/metrics?/#Client
    purpose: 学习现有Client结构
    node_ids:  # ast_node特有
      - mod_path: code.byted.org/namespace/mod_name
        pkg_path: code.byted.org/namespace/mod_name/pkg/database
        name: Client

  - type: file
    path: pkg/server/server.go
    lines: [100, 150]  # 可选，行范围
    purpose: 参考路由注册模式

  - type: pattern
    pattern_name: route_registration  # 可选
    purpose: 参考项目中其他路由的注册方式
    search_in:  # 可选，指定搜索范围
      - pkg/server

  - type: dependency_context
    from_task: task_1_1
    purpose: 使用SDK依赖配置
    files:  # 可选，只引用特定文件
      - pkg/clients/metrics/client.go

  - type: sdk_definition
    sdk: code.byted.org/inf/metrics-query
    method: NewClient
    purpose: 学习SDK使用方法
    example: |
      import mq "code.byted.org/inf/metrics-query"
      client, err := mq.NewClient(appName, appSecret, opts...)

  - type: config
    config_type: database | metrics | redis | ...
    purpose: 获取配置信息

implementation: |
  具体实现要求：
  1. 使用Metrics SDK的NewClient方法
  2. App Name: sinf.bytecgc.task
  3. 配置China-North和China-BOE endpoints

verification:
  - type: build  # build | test | output | file_exists | compile_check
    required: true

  - type: test
    command: go test ./pkg/clients/metrics/
    required: false

  - type: output
    contains: "Metrics client initialized"
    required: true
```

### 任务组：阶段2名称
```task
id: task_2_1
title: 定义API模型
stage: 2
priority: high

depends_on:
  - task_1_1

files:
  - path: pkg/models/api_models/api.go
    action: modify
    description: 添加ResourceUsageListRequest等模型

context:
  - type: file
    path: pkg/models/api_models/api.go
    purpose: 学习现有API模型定义模式

implementation: |
  新增以下结构体：
  1. ResourceUsageListRequest
     - service_type: []string (rds, redis，空或传空表示全部)
     - page: int
     - page_size: int

  2. ResourceUsageItem
     - psm: string
     - service_type: string (rds, redis)
     - biz_domain: string
     - owners: []string
     - cpu: float64
     - mem: float64
     - disk: float64

  3. ResourceUsageListResponse和ResourceUsageListData

verification:
  - type: build
    required: true
```
```

## 上下文类型说明

### 1. ast_node - AST节点分析
使用 `mcp__abcoder__get_ast_node` 获取AST节点信息。

```yaml
context:
  - type: ast_node
    target: 包名.结构体/函数名
    purpose: 收集目的说明
    node_ids:
      - mod_path: 模块路径
        pkg_path: 包路径
        name: 节点名称
```

**用途**：
- 学习代码结构
- 理解调用关系
- 获取类型定义
- 分析实现模式

### 2. file - 文件内容
读取文件内容，可指定行范围。

```yaml
context:
  - type: file
    path: 文件相对路径
    purpose: 收集目的说明
    lines: [start, end]  # 可选，不指定则读取整个文件
```

**用途**：
- 参考现有代码实现
- 学习代码风格
- 理解项目结构

### 3. pattern - 代码模式
参考项目中类似的代码模式。

```yaml
context:
  - type: pattern
    pattern_name: 模式名称
    purpose: 收集目的说明
    search_in:  # 可选：指定搜索范围
      - pkg/server
      - pkg/dispatcher
```

**用途**：
- 学习特定的代码模式（如路由注册、DAO操作）
- 避免重复造轮子
- 保持代码风格一致

### 4. dependency_context - 依赖任务上下文
引用已完成任务的上下文。

```yaml
context:
  - type: dependency_context
    from_task: task_1_1
    purpose: 使用前置任务的输出
    files:  # 可选：只引用特定文件
      - pkg/clients/metrics/client.go
```

**用途**：
- 复用前置任务生成的代码
- 理解数据流向
- 保持上下文连续性

### 5. sdk_definition - SDK定义
外部SDK的定义和使用示例。

```yaml
context:
  - type: sdk_definition
    sdk: code.byted.org/inf/metrics-query
    method: NewClient
    purpose: 学习SDK使用方法
    example: |
      import mq "code.byted.org/inf/metrics-query"

      client, err := mq.NewClient("sinf.bytecgc.task", "60560c36-9edb-4a78-be56-96a4a4900ed7",
          mq.WithPeriodicTokenUpdate,
          mq.WithEntry(mq.Cluster("China-North"), "http://openapi-metrics-cn.byted.org"))
      if err != nil {
          return nil, err
      }
```

**用途**：
- 提供SDK使用规范
- 避免错误的API调用
- 统一SDK使用方式

### 6. config - 配置信息
项目配置信息。

```yaml
context:
  - type: config
    config_type: database | metrics | redis | ...
    purpose: 获取配置信息
```

**用途**：
- 获取必要的配置
- 理解配置结构
- 确保配置正确使用

## 验证类型说明

### 1. build - 编译验证
```yaml
verification:
  - type: build
    required: true  # true | false
```

### 2. test - 测试验证
```yaml
verification:
  - type: test
    command: go test ./pkg/...
    required: true
    expected_output: "PASS"  # 可选
```

### 3. output - 输出验证
```yaml
verification:
  - type: output
    contains: "expected string"  # 必须包含
    not_contains: "error string"  # 不能包含
    required: true
```

### 4. file_exists - 文件存在验证
```yaml
verification:
  - type: file_exists
    path: pkg/new_file.go
    required: true
```

### 5. compile_check - 编译检查
```yaml
verification:
  - type: compile_check
    required: false
```


## 注意事项
1. **task_id唯一性**: 确保每个task_id在全局唯一
2. **依赖无环**: depends_on不能形成循环依赖
3. **context完备性**: 确保每个任务的context包含所有必要的信息
4. **verification完整性**: 每个任务至少要有build验证
5. **implementation清晰性**: 详细描述实现要求，便于LLM理解
