# CODE_TASK 模板

## 使用方法
描述你的编码需求时引用此文件，Claude Code 会：
1. 解析任务列表
2. 使用 TodoWrite 工具跟踪进度
3. 用文档提供的参数直接调用`mcp__abcoder__get_ast_node`

## 格式示例

```markdown
1. [ ] 为Flight数据模型添加comfortInfo字段
   - file: models/flight.go
   - action: modify

2. [ ] 创建ComfortInfo数据源的客户端
   - file: clients/comfort_client.go
   - action: create
   - 涉及`go-redis`的`newClient` Method
``
mcp__abcoder__get_ast_node "repo_name" '[{"mod_path":"node/mod/path","pkg_path":"node/pkg/path","name":"node_name"}]'
``

3. [ ] 在FlightService中增加调用comfort_client的逻辑
   - file: services/flight_service.go
   - action: modify

4. [ ] 更新GetFlightDetails API的返回值
   - file: controllers/flight_controller.go
   - action: modify
   - 返回值IDL结构：
``
type FlightDetailsResponse struct {
    // 航班基础信息
    FlightNumber string `json:"flight_number"` // 航班号，如 "CA1234"
    Airline      string `json:"airline"`      // 航空公司，如 "中国国际航空"
    Status struct {
        Code    string `json:"code"`    // 状态码，如 "ON_TIME", "DELAYED", "CANCELLED"
        Message string `json:"message"` // 状态描述，如 "航班准点起飞"
    } `json:"status"`
}
``

5. [ ] 为新增逻辑编写单元测试
   - file: services/flight_service_test.go
   - action: create
```

### 验证方法
#### 接口验证
1. 调用curl
```
curl -X POST 'localhost:4379/mcp' \
  --header 'Content-Type: application/json' \
  --header 'mcp-session-id: {{mcp-session}}' \
  --data '{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "get_flight_details",
    "arguments": {
      "flight_number": {{flight_number}}
    }
  }
}'
```
#### 日志验证
调用上面的接口验证时，Log应该输出下面的信息
```
[INFO] FlightDetailsResponse: xxx
```

```

```
## 注意事项
IMPORTANT: 保持格式简单，让 Claude Code 专注于执行
- action 可选: create/modify/delete
- 文件路径要准确
- 任务描述要明确具体。对于涉及外部服务调用或 SDK 使用的任务，必须提供详细的技术规格；提醒禁止简化实现或MOCK，必须使用真实 SDK/curl 调用
  - 涉及SDK：需要明确指定具体的 SDK Package、Method名称
  - 涉及curl：需要明确指定完整的 curl 命令，包括请求头、请求体、URL、Resp结构（JSON/IDL）
- 如果外部SDK内容较多，请在文档提供`mcp__abcoder__get_ast_node`调用参数（请确保参数能正确返回结果），保持文档内容简洁；让subAgent可直接调用
- 在关键Step处提醒`build`修复语法错误；避免最后出现大量语法问题
- 必须提供直接、清晰、具体的验证方法

## 核心理念
**直接使用原则**：编写CODE_TASK时进行充分的`mcp__abcoder__get_ast_node`分析，提供完整的上下文信息、`get_ast_node`调用参数，确保无需`mcp__abcoder`重新分析仓库即可直接执行任务。

指导原则：
- 技术规格预分析：在编写任务时就提供完整的技术实现背景、分支逻辑
- 代码结构预理解：通过`mcp__abcoder`工具预先分析代码结构和依赖关系
- 避免重复工作：SubAgent应专注于执行，而非重新理解和分析
