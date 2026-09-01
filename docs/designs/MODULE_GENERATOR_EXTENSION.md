---
title: 模块生成模板扩展设计
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-06-26
---

# 模块生成模板扩展设计

English version: [MODULE_GENERATOR_EXTENSION.en.md](./MODULE_GENERATOR_EXTENSION.en.md)

## 1. 背景

当前 `backend/internal/scaffold/` 模块生成器生成的代码缺少最佳实践：

- 不自动导入 `pkg/common` 中的工具函数
- 不使用 `StatusEnabled/StatusDisabled` 常量
- 不包含分页、导入导出的通用工具

本文档定义生成器模板的扩展规范。

---

## 2. 当前生成器能力

### 2.1 已支持的功能

| 功能         | 文件                                      | 说明         |
| ------------ | ----------------------------------------- | ------------ |
| 后端模块注册 | `generated_module.go`                     | 模块初始化   |
| 后端 CRUD    | `*_service.go`                            | 基础增删改查 |
| 后端 Handler | `*_handler.go`                            | HTTP 路由    |
| 前端模块注册 | `frontend/src/modules/*/index.ts`         | 路由注册     |
| 前端组件注册 | `frontend/src/modules/*/components/*.tsx` | 组件懒加载   |

### 2.2 缺失的最佳实践

| 缺失项            | 影响                               |
| ----------------- | ---------------------------------- |
| `pkg/common` 导入 | 状态常量、错误处理                 |
| 分页工具          | `NormalizePageQuery`               |
| 导入导出          | `uploadImportFile`, `downloadFile` |
| 软删除支持        | `deleted_at` 过滤                  |
| 审计字段          | `created_by`, `updated_by`         |

---

## 3. 扩展方案

### 3.1 后端模板扩展

#### 3.1.1 导入块扩展

```go
// 生成模板中新增 import
import (
    "pantheon-platform/backend/pkg/common"  // 新增
    "pantheon-platform/backend/pkg/impexp"   // 新增
)
```

#### 3.1.2 Service 模板扩展

```go
// 新增辅助函数使用
func (s *{{.ServiceName}}Service) Create{{.ModelName}}(req *{{.ModelName}}CreateReq) (*{{.ModelName}}Resp, error) {
    // 使用状态常量
    status := common.NormalizeEnabledStatus(req.Status)
    // ...
}

// 新增状态常量定义区域
const (
    // Status constants are defined in pkg/common
    // Use common.StatusEnabled and common.StatusDisabled
)

// 新增批量 ID 去重
ids := common.NormalizeUint64IDs(req.IDs)
```

#### 3.1.3 分页查询扩展

```go
// 使用统一分页工具
func (s *{{.ServiceName}}Service) List{{.ModelName}}(query *{{.ModelName}}ListQuery) (*ListResponse, error) {
    page, pageSize := common.NormalizePageQuery(query.Page, query.PageSize)

    var total int64
    var items []{{.ModelName}}
    // ...
}
```

### 3.2 前端模板扩展

#### 3.2.1 导入块扩展

```typescript
// 新增通用工具导入
import { normalizePageQuery } from "@/utils/pagination";
import { downloadFile, uploadImportFile } from "@/api";
import { useNotification } from "@/hooks/useNotification";
```

#### 3.2.2 API 调用扩展

```typescript
// 新增导入导出方法
export const download{{.ModelName}}Template = () => downloadFile('/{{.ModulePath}}/import-template');
export const export{{.ModelName}}s = (params) => downloadFile('/{{.ModulePath}}/export', params);
export const import{{.ModelName}}s = (file: File) => uploadImportFile('/{{.ModulePath}}/import', file);
```

---

## 4. 实现清单

### 4.1 后端

| 文件           | 修改内容                                    |
| -------------- | ------------------------------------------- |
| `workspace.go` | 新增 `generatedBackendServiceTemplate` 扩展 |
| `types.go`     | 新增 `CommonHelpers bool` 配置项            |

### 4.2 前端

| 文件                                   | 修改内容            |
| -------------------------------------- | ------------------- |
| `frontend/scripts/generate-module.mjs` | 新增 common imports |

---

## 5. 验收标准

- [ ] 新生成的模块自动导入 `pkg/common`
- [ ] 状态字段使用 `common.StatusEnabled/StatusDisabled`
- [ ] 批量操作使用 `common.NormalizeUint64IDs`
- [ ] 前端 API 自动包含导入导出方法
- [ ] 生成代码通过 lint 检查

---

## 6. 示例输出

### 6.1 生成后端 Service

```go
package business_example

import (
    "errors"
    "pantheon-platform/backend/pkg/common"  // 新增
    "pantheon-platform/backend/pkg/impexp"  // 新增
    "time"

    "gorm.io/gorm"
)

type ExampleService struct {
    db *gorm.DB
}

func (s *ExampleService) CreateExample(req *ExampleCreateReq) (*ExampleResp, error) {
    // 使用状态常量
    status := common.NormalizeEnabledStatus(req.Status)

    example := Example{
        Name:   req.Name,
        Status: status,
    }
    if err := s.db.Create(&example).Error; err != nil {
        return nil, err
    }
    return toExampleResp(example), nil
}

func (s *ExampleService) BatchUpdateStatus(ids []uint64, status int) (int, error) {
    // 使用批量去重
    normalizedIDs := common.NormalizeUint64IDs(ids)
    if len(normalizedIDs) == 0 {
        return 0, common.NewBadRequest("param.invalid")
    }

    if err := s.db.Model(&Example{}).
        Where("id IN ?", normalizedIDs).
        Updates(map[string]any{"status": common.NormalizeEnabledStatus(status)}).Error; err != nil {
        return 0, err
    }
    return len(normalizedIDs), nil
}
```

### 6.2 生成前端 API

```typescript
// 自动生成的 API 文件
import { request } from "@/core/request";
import { downloadFile, uploadImportFile } from "@/api"; // 新增

export const getExampleList = (params: ExampleListQuery) =>
  request.get<{ list: Example[]; total: number }>("/business/example/list", {
    params,
  });

export const createExample = (data: ExampleCreateReq) =>
  request.post<Example>("/business/example", data);

export const downloadExampleTemplate = () =>
  downloadFile("/business/example/import-template");

export const exportExamples = (params?: ExampleExportQuery) =>
  downloadFile("/business/example/export", params);

export const importExamples = (file: File) =>
  uploadImportFile("/business/example/import", file);
```

---

## 7. 迁移路径

| 阶段    | 任务             |
| ------- | ---------------- |
| Phase 1 | 更新后端生成模板 |
| Phase 2 | 更新前端生成模板 |
| Phase 3 | 文档和验收       |
