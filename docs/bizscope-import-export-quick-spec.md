# BizScope 导入导出快速实施指令

**模块**: 业务域 (BizScope)  
**复杂度**: 简单  
**预计时间**: 0.4天

## 参考文件
- 后端参考: `backend/modules/business/cmdb/host/host_import_export.go`
- 前端参考: `frontend/src/modules/business/cmdb/host/CmdbHostList.tsx`

## CSV 字段定义

```csv
code,name,owner,environment,status,description
```

**字段说明**:
- `code` (必填，唯一): 业务域编码，如 "ecommerce-prod"
- `name` (必填): 业务域名称，如 "电商生产环境"
- `owner`: 负责人
- `environment`: 环境类型 (dev/test/prod)
- `status`: 状态 (active/inactive)
- `description`: 描述

## 需要修改的文件

### 后端 (6个文件)

1. **创建 `backend/modules/business/bizscope/bizscope_import_export.go`**
   - 复制 `cmdb/host/host_import_export.go` 的完整结构
   - 修改：`Host` → `BizScope`, `HostService` → `BizScopeService`
   - CSV 字段映射到 `BizScope` 结构体
   - 唯一性判断：基于 `LOWER(code)`

2. **修改 `backend/modules/business/bizscope/bizscope_handler.go`**
   - 在 `RegisterRoutes` 添加 3 个路由：
     ```go
     bizscope.GET("/export", middleware.RequirePermission("business:bizscope:export"), h.Export)
     bizscope.GET("/import-template", h.DownloadImportTemplate)
     bizscope.POST("/import", middleware.RequirePermission("business:bizscope:import"), h.Import)
     ```
   - 添加 3 个 Handler 方法

3. **修改 `backend/modules/business/bizscope/bizscope_service.go`**
   - 添加 `GetByCodeCaseInsensitive(code string)` 方法（用于导入时查重）

4. **修改 `backend/modules/business/bizscope/module.go`**
   - 在 permissions 数组添加：
     ```go
     {Key: "operations-bizscope-export", ParentKey: "operations-bizscope-list", TitleKey: "operations.bizscope.export", Perms: "business:bizscope:export", Type: "F", Module: bizscopeModuleKey, Sort: 3},
     {Key: "operations-bizscope-import", ParentKey: "operations-bizscope-list", TitleKey: "operations.bizscope.import", Perms: "business:bizscope:import", Type: "F", Module: bizscopeModuleKey, Sort: 4},
     ```

### 前端 (3个文件)

5. **修改 `frontend/src/modules/business/bizscope/api.ts`**
   - 添加：
     ```typescript
     export function exportBizScopes(params?: BizScopeListQuery) {
       return downloadFile({
         url: '/business/bizscope/export',
         method: 'get',
         params: params as Record<string, unknown> | undefined,
         filename: 'bizscope-export.csv',
       });
     }
     
     export function downloadBizScopeImportTemplate() {
       return downloadFile({
         url: '/business/bizscope/import-template',
         method: 'get',
         filename: 'bizscope-import-template.csv',
       });
     }
     
     export function importBizScopes(file: File) {
       return uploadImportFile('/business/bizscope/import', file);
     }
     ```

6. **修改 `frontend/src/modules/business/bizscope/BizScopeList.tsx`**
   - 导入组件：
     ```typescript
     import { ImportCsvButton } from '../../../components';
     import { IconDownload } from '@arco-design/web-react/icon';
     ```
   - 添加 3 个处理函数：`handleExport`, `handleDownloadTemplate`, `handleImport`
   - 在 `ListHeaderActions` 添加导入导出按钮

7. **修改国际化文件**
   - `frontend/src/modules/business/bizscope/locales/zh-CN.json`:
     ```json
     {
       "operations.bizscope.export": "导出业务域",
       "operations.bizscope.import": "导入业务域",
       "business.bizscope.import.codeRequired": "业务域编码不能为空",
       "business.bizscope.import.nameRequired": "业务域名称不能为空",
       "business.bizscope.import.invalidEnvironment": "环境类型必须是 dev, test, prod 之一",
       "business.bizscope.import.invalidStatus": "状态必须是 active 或 inactive"
     }
     ```
   - 对应添加 `en-US.json` 英文版

## 验证逻辑

### 必填字段
- `code` (必填)
- `name` (必填)

### 枚举验证
- `environment` ∈ {dev, test, prod, ""} (空表示未设置)
- `status` ∈ {active, inactive, ""} (空表示默认 active)

### 唯一性
- 基于 `LOWER(code)` 判断是创建还是更新
- SQL: `SELECT * FROM biz_business_scope WHERE LOWER(code) = LOWER(?)`

### 业务逻辑
- 导入时如果 code 已存在，更新记录
- 导入时如果 code 不存在，创建新记录
- 不删除绑定的主机（与导出无关）

## 实施步骤

1. 复制 `cmdb/host/host_import_export.go` 到 `bizscope/bizscope_import_export.go`
2. 全局替换：
   - `Host` → `BizScope`
   - `host` → `bizscope`
   - `HostService` → `BizScopeService`
   - `hostname` → `code`
3. 修改 CSV 字段映射（9个字段 → 6个字段）
4. 修改验证逻辑（环境类型、状态枚举）
5. 修改 Handler、权限、前端 API
6. 测试导入导出流程

## Codex 执行指令（待 Label 完成后使用）

```bash
codex exec "为 BizScope (业务域) 模块添加导入导出功能，复制 CMDB Host 的实现模式。

参考: backend/modules/business/cmdb/host/host_import_export.go

CSV字段: code,name,owner,environment,status,description
唯一性: LOWER(code)
枚举: environment(dev/test/prod), status(active/inactive)

详见 docs/bizscope-import-export-quick-spec.md" -C pantheon-ops
```

---

**优先级**: P1（在 CMDB Label 完成后立即执行）
