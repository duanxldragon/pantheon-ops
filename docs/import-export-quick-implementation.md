# Business 模块导入导出功能 - 快速实施方案

**策略**: 直接复制 CMDB Host 的实现模式，逐模块适配

## 实施顺序

1. **CMDB Label** (最简单) - 0.3天
2. **BizScope** (简单) - 0.4天  
3. **CMDB Group** (中等，树形) - 0.5天
4. **Deploy Package** (中等) - 0.6天
5. **Deploy Template** (复杂，JSON) - 0.8天
6. **Deploy Task** (只读导出) - 0.3天

## CMDB Label 实施清单（第一个）

### 后端文件

**1. 创建 `backend/modules/business/cmdb/label/label_import_export.go`**

直接复制 `host/host_import_export.go` 的结构，修改：
- 结构体：Host → Label
- 字段映射：
  ```go
  // CSV 字段
  Key         string // key (必填，唯一)
  Name        string // name (必填)
  Category    string // category (base/network/business/custom)
  ValueMode   string // valueMode (free/enum/dict)
  DictCode    string // dictCode (valueMode=dict时必填)
  Options     string // options (valueMode=enum时必填，逗号分隔)
  Required    string // required (true/false)
  Status      string // status (enabled/disabled)
  Description string // description
  ```
- 验证逻辑：
  - 必填字段：Key, Name, Category, ValueMode
  - 枚举验证：Category, ValueMode, Status
  - 业务逻辑：
    - valueMode=enum 时 Options 必填
    - valueMode=dict 时 DictCode 必填
- 唯一性判断：基于 `LOWER(key)` 匹配

**2. 修改 `backend/modules/business/cmdb/label/label_handler.go`**

在 `RegisterRoutes` 方法中添加：
```go
labels.GET("/export", middleware.RequirePermission("business:cmdb:label:export"), h.Export)
labels.GET("/import-template", h.DownloadImportTemplate)
labels.POST("/import", middleware.RequirePermission("business:cmdb:label:import"), h.Import)
```

添加三个 Handler 方法（直接复制 host_handler.go 的对应方法）：
- `Export(c *gin.Context)`
- `DownloadImportTemplate(c *gin.Context)`
- `Import(c *gin.Context)`

**3. 修改 `backend/modules/business/cmdb/module.go`**

在 `labelPermissions` 数组中添加：
```go
{Key: "operations-cmdb-label-export", ParentKey: "operations-cmdb-label-list", TitleKey: "operations.cmdb.label.export", Perms: "business:cmdb:label:export", Type: "F", Module: cmdbModuleKey, Sort: 3},
{Key: "operations-cmdb-label-import", ParentKey: "operations-cmdb-label-list", TitleKey: "operations.cmdb.label.import", Perms: "business:cmdb:label:import", Type: "F", Module: cmdbModuleKey, Sort: 4},
```

### 前端文件

**4. 修改 `frontend/src/modules/business/cmdb/label/api.ts`**

添加三个函数（复制 `host/api.ts` 的对应函数）：
```typescript
export function exportLabels(params?: LabelListQuery) {
  return downloadFile({
    url: '/business/cmdb/labels/export',
    method: 'get',
    params: params as Record<string, unknown> | undefined,
    filename: 'cmdb-label-export.csv',
  });
}

export function downloadLabelImportTemplate() {
  return downloadFile({
    url: '/business/cmdb/labels/import-template',
    method: 'get',
    filename: 'cmdb-label-import-template.csv',
  });
}

export function importLabels(file: File) {
  return uploadImportFile('/business/cmdb/labels/import', file);
}
```

**5. 修改 `frontend/src/modules/business/cmdb/label/CmdbLabelSchemaList.tsx`**

添加导入：
```typescript
import { ImportCsvButton } from '../../../../components';
import { IconDownload } from '@arco-design/web-react/icon';
```

添加处理函数（复制 `CmdbHostList.tsx` 的对应函数）：
```typescript
const handleExport = async () => { /* ... */ };
const handleDownloadTemplate = async () => { /* ... */ };
const handleImport = async (file: File) => { /* ... */ };
```

在 `ListHeaderActions` 中添加按钮：
```typescript
<Button
  icon={<IconDownload />}
  onClick={handleDownloadTemplate}
>
  {t('common.downloadTemplate')}
</Button>
<Button
  icon={<IconDownload />}
  onClick={handleExport}
  disabled={!hasPerm('business:cmdb:label:export')}
>
  {t('common.export')}
</Button>
<ImportCsvButton
  onImport={handleImport}
  disabled={!hasPerm('business:cmdb:label:import')}
/>
```

**6. 修改国际化文件**

`frontend/src/modules/business/cmdb/locales/zh-CN.json`:
```json
{
  "operations.cmdb.label.export": "导出标签",
  "operations.cmdb.label.import": "导入标签",
  "business.cmdb.label.import.keyRequired": "标签键不能为空",
  "business.cmdb.label.import.nameRequired": "标签名称不能为空",
  "business.cmdb.label.import.categoryRequired": "分类不能为空",
  "business.cmdb.label.import.valueModeRequired": "取值模式不能为空",
  "business.cmdb.label.import.invalidCategory": "分类必须是 base, network, business, custom 之一",
  "business.cmdb.label.import.invalidValueMode": "取值模式必须是 free, enum, dict 之一",
  "business.cmdb.label.import.invalidStatus": "状态必须是 enabled 或 disabled",
  "business.cmdb.label.import.enumOptionsRequired": "取值模式为 enum 时，选项列表不能为空",
  "business.cmdb.label.import.dictCodeRequired": "取值模式为 dict 时，字典编码不能为空"
}
```

对应添加英文版到 `en-US.json`。

## 关键点

1. **直接复制**: 99% 的代码可以直接从 `host/` 复制，只需修改结构体名称和字段
2. **最小改动**: 不要重新设计，完全遵循 Host 的模式
3. **快速验证**: 实现后立即测试导入导出流程
4. **迭代推进**: Label 完成后，再用相同方式处理下一个模块

## 验收标准

- [ ] 后端 API 可以访问（/labels/export, /labels/import-template, /labels/import）
- [ ] 前端 UI 显示导入导出按钮
- [ ] 可以导出现有标签数据
- [ ] 可以下载导入模板
- [ ] 可以导入 CSV 文件（创建 + 更新）
- [ ] 权限控制生效
- [ ] 国际化文案正常显示

---

**下一步**: 完成 Label 后，复制相同模式到 BizScope
