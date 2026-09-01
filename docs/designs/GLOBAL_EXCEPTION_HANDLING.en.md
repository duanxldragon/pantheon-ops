---
title: Global Exception Handling Design
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-06-26
---

# Global Exception Handling Design

Chinese version: [GLOBAL_EXCEPTION_HANDLING.md](./GLOBAL_EXCEPTION_HANDLING.md)

## 1. Background & Goals

Current backend lacks unified exception handling:

- Handlers return error responses directly
- Repeated error -> HTTP response conversion code
- Panic causing unhandled crashes
- Missing unified error logging, monitoring, and alerting

This document defines the minimum viable solution for global exception handling.

---

## 2. Core Design

### 2.1 Exception Type System

```go
// pkg/errors/errors.go
package errors

type AppError struct {
    Code       string `json:"code"`
    Message    string `json:"message"`
    Detail     string `json:"detail,omitempty"`
    Cause      error  `json:"-"`
    HTTPStatus int    `json:"-"`
}

func (e *AppError) Error() string
func (e *AppError) Unwrap() error

// Factory functions
func NewBadRequest(code string) *AppError
func NewNotFound(code string) *AppError
func NewForbidden(code string) *AppError
func NewInternal(code string) *AppError
func NewUnauthorized(code string) *AppError
```

### 2.2 Middleware Stack

```
Request
  ↓
RecoveryMiddleware      ← Capture panic, return 500
  ↓
ErrorHandlerMiddleware ← Unified error response, logging
  ↓
TokenAuthMiddleware
  ↓
Business Handler
  ↓
Response
```

### 2.3 Unified Response Format

```go
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
```

---

## 3. Migration Path

| Phase   | Tasks                                           |
| ------- | ----------------------------------------------- |
| Phase 1 | Create `pkg/errors/errors.go`, middleware files |
| Phase 2 | Unified response package                        |
| Phase 3 | i18n integration                                |
| Phase 4 | Monitoring & alerting                           |

---

## 4. Acceptance Criteria

- [ ] Panic does not crash the process
- [ ] All API errors return unified format
- [ ] Error logs include trace_id
- [ ] i18n keys are configurable
- [ ] Unit tests cover error paths
