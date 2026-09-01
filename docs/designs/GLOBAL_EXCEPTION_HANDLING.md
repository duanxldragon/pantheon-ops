---
title: 全局异常处理设计
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-06-26
---

# 全局异常处理设计

English version: [GLOBAL_EXCEPTION_HANDLING.en.md](./GLOBAL_EXCEPTION_HANDLING.en.md)

## 1. 背景与目标

当前后端缺乏统一的异常处理机制：

- 各 handler 直接返回错误响应
- 重复的 error -> HTTP response 转换代码
- panic 导致的未处理崩溃
- 缺少统一的错误日志、监控和告警

本文定义全局异常处理的最小可行方案。

---

## 2. 核心设计

### 2.1 异常类型体系

```go
// pkg/errors/errors.go
package errors

type AppError struct {
    Code    string      `json:"code"`    // 业务错误码
    Message string      `json:"message"` // 用户可见消息（i18n key）
    Detail  string     `json:"detail,omitempty"` // 内部调试信息
    Cause   error      `json:"-"`
    HTTPStatus int     `json:"-"`       // HTTP 状态码
}

func (e *AppError) Error() string {
    return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
}

func (e *AppError) Unwrap() error {
    return e.Cause
}

// 工厂函数
func NewBadRequest(code string) *AppError
func NewNotFound(code string) *AppError
func NewForbidden(code string) *AppError
func NewInternal(code string) *AppError
func NewUnauthorized(code string) *AppError

// 带详情
func NewBadRequestf(code string, detail string, args ...any) *AppError
```

### 2.2 中间件栈

```
Request
  ↓
RecoveryMiddleware      ← 捕获 panic，转为 500
  ↓
ErrorHandlerMiddleware  ← 统一错误响应、日志
  ↓
TokenAuthMiddleware
  ↓
Business Handler
  ↓
Response
```

### 2.3 RecoveryMiddleware

```go
// internal/middleware/recovery.go
func RecoveryMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if r := recover(); r != nil {
                log.Error("panic recovered",
                    "path", c.Request.URL.Path,
                    "error", r,
                    "stack", string(debug.Stack()))

                // 发送告警（可选）
                alertPanic(c, r)

                c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{
                    Code:    "system.internal_error",
                    Message: "system.internal_error",
                })
            }
        }()
        c.Next()
    }
}
```

### 2.4 ErrorHandlerMiddleware

```go
// internal/middleware/error_handler.go
func ErrorHandlerMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()

        // 处理 c.Errors 中的错误
        if len(c.Errors) > 0 {
            err := c.Errors.Last().Err
            handleError(c, err)
        }
    }
}

func handleError(c *gin.Context, err error) {
    var appErr *AppError
    if errors.As(err, &appErr) {
        c.JSON(appErr.HTTPStatus, ErrorResponse{
            Code:    appErr.Code,
            Message: appErr.Message,
        })
        return
    }

    // 未知错误
    log.Error("unhandled error", "error", err)
    c.JSON(http.StatusInternalServerError, ErrorResponse{
        Code:    "system.internal_error",
        Message: "system.internal_error",
    })
}
```

### 2.5 统一响应格式

```go
// pkg/response/response.go
package response

type SuccessResponse struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Detail  string `json:"detail,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, SuccessResponse{
        Code:    0,
        Message: "success",
        Data:    data,
    })
}

func Error(c *gin.Context, err error) {
    // 复用 handleError 逻辑
}
```

---

## 3. i18n 集成

错误消息通过 i18n key 获取本地化文本：

```json
{
  "system.internal_error": "系统内部错误，请稍后重试",
  "dept.not_found": "部门不存在",
  "user.not_found": "用户不存在"
}
```

前端根据 `code` 查询 i18n 显示用户友好消息。

---

## 4. 监控与告警

### 4.1 日志结构

```json
{
  "level": "error",
  "timestamp": "2026-06-26T10:00:00Z",
  "path": "/api/system/user",
  "method": "POST",
  "error_code": "dept.not_found",
  "error_message": "部门不存在",
  "trace_id": "abc123",
  "user_id": 1
}
```

### 4.2 告警规则

| 条件                    | 级别     | 动作     |
| ----------------------- | -------- | -------- |
| 5分钟内 500 错误 > 10次 | Warning  | 日志告警 |
| 5分钟内 panic > 1次     | Critical | 立即告警 |
| 特定错误码频率异常      | Warning  | 日志告警 |

---

## 5. 迁移路径

### Phase 1: 中间件基础设施

- [ ] 创建 `pkg/errors/errors.go`
- [ ] 创建 `internal/middleware/recovery.go`
- [ ] 创建 `internal/middleware/error_handler.go`
- [ ] 在 `main.go` 注册中间件

### Phase 2: 统一响应

- [ ] 创建 `pkg/response/response.go`
- [ ] 改造现有 handler 返回方式

### Phase 3: i18n 集成

- [ ] 添加错误消息 i18n key
- [ ] 前端统一错误展示组件

### Phase 4: 监控

- [ ] 结构化日志集成
- [ ] 告警规则配置

---

## 6. 约束与原则

1. **不暴露内部细节**：生产环境 `detail` 字段仅对特定角色可见
2. **不丢失错误链**：`errors.Wrap` 保留完整堆栈
3. **幂等性考虑**：GET 请求错误应可安全重试
4. **性能影响**：异常处理路径应尽量减少内存分配

---

## 7. 相关文件

| 文件                                   | 作用         |
| -------------------------------------- | ------------ |
| `pkg/errors/errors.go`                 | 错误类型定义 |
| `internal/middleware/recovery.go`      | Panic 恢复   |
| `internal/middleware/error_handler.go` | 错误处理     |
| `pkg/response/response.go`             | 响应格式化   |
| `backend/cmd/server/main.go`           | 中间件注册   |

---

## 8. 验收标准

- [ ] Panic 不导致进程崩溃
- [ ] 所有 API 错误返回统一格式
- [ ] 错误日志包含 trace_id
- [ ] i18n key 可配置
- [ ] 单元测试覆盖错误路径
