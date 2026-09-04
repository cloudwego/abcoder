# MCP Server Handler 并发安全性 (Concurrency Safety)

## 概述 (Overview)

ABCoder MCP 服务器的请求处理器（request handler）中的上下文（context）是**并发安全的**。

The request context in the ABCoder MCP server handler is **concurrency-safe**.

## 并发安全性保证 (Concurrency Safety Guarantees)

### 1. 上下文隔离 (Context Isolation)

每个请求都会收到一个独立的 `context.Context` 实例，由 MCP 框架提供：

Each request receives an isolated `context.Context` instance provided by the MCP framework:

```go
Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // ctx is unique per request
    var req R
    if err := request.BindArguments(&req); err != nil {
        return nil, err
    }
    // handler execution with isolated context
    resp, err := handler(ctx, req)
    if err != nil {
        // handle error
    }
    // process response
}
```

### 2. 无上下文修改 (No Context Mutation)

代码库中不使用 `context.WithValue()` 或其他上下文修改操作，消除了上下文值修改引起的并发问题：

The codebase does not use `context.WithValue()` or other context mutation operations, eliminating concurrency issues from context value modifications:

- ✅ 无全局上下文状态 (No global context state)
- ✅ 上下文仅用于传递和取消 (Context used only for propagation and cancellation)
- ✅ 每个处理器调用都获得新的上下文 (Each handler invocation gets a fresh context)

### 3. 共享状态的线程安全 (Thread-Safe Shared State)

应用程序级别的共享状态使用 `sync.Map` 进行保护：

Application-level shared state is protected using `sync.Map`:

```go
type ASTReadTools struct {
    opts  ASTReadToolsOptions
    repos sync.Map  // Thread-safe for concurrent access
    tools map[string]tool.InvokableTool
}
```

### 4. 框架级并发处理 (Framework-Level Concurrency Handling)

底层 MCP-Go 框架的 `Listen()` 方法正确处理并发请求，为每个请求提供独立的上下文：

The underlying MCP-Go framework's `Listen()` method properly handles concurrent requests, providing isolated contexts for each request:

```go
// Server handles multiple concurrent requests
go func() {
    err := stdioServer.Listen(ctx, stdinReader, stdoutWriter)
    // Each request gets its own context and goroutine
}()
```

## 测试验证 (Testing Verification)

项目包含专门的并发测试来验证上下文安全性：

The project includes dedicated concurrency tests to verify context safety:

### 测试：TestConcurrentHandlerInvocations

此测试验证：
- 20个并发请求同时调用处理器
- 每个请求都有自己的上下文
- 所有请求都成功完成，没有竞争条件

This test verifies:
- 20 concurrent requests invoke handlers simultaneously
- Each request has its own context
- All requests complete successfully without race conditions

```bash
# 运行测试 (Run test)
go test -v ./llm/mcp -run TestConcurrentHandlerInvocations

# 使用竞态检测器运行 (Run with race detector)
go test -race -v ./llm/mcp -run TestConcurrentHandlerInvocations
```

测试结果：
```
=== RUN   TestConcurrentHandlerInvocations
    server_test.go:222: Successfully handled 20 concurrent requests without context conflicts
--- PASS: TestConcurrentHandlerInvocations (0.02s)
PASS
```

## 结论 (Conclusion)

✅ **请求上下文完全并发安全**，因为：

The request context is completely concurrency-safe because:

1. 每个请求接收一个隔离的上下文实例
2. 从不创建或修改上下文值（不使用 `WithValue()`）
3. 上下文从不在并发处理器之间共享
4. 底层 MCP 框架正确处理请求路由
5. 共享应用程序状态（repos）使用 `sync.Map` 保护

**不存在与请求上下文处理相关的竞态条件。**

**No race conditions exist related to request context handling.**

## 参考资料 (References)

- [handler.go](handler.go) - 处理器实现 (Handler implementation)
- [server.go](server.go) - 服务器实现 (Server implementation)
- [server_test.go](server_test.go) - 并发测试 (Concurrency tests)
- [ast_read.go](../tool/ast_read.go) - 线程安全的共享状态 (Thread-safe shared state)
