# BizScope 导入导出功能实现规范

**模块**: Business > BizScope (业务域管理)  
**优先级**: P1  
**预计工期**: 0.4 天

## 数据模型

```go
type BizScope struct {
    ID          uint64
    Code        string         // 业务域编码（唯一，必填）
    Name        string         // 业务域名称（必填）
    Owner       string         // 负责人
    Environment string         // 环境（dev/test/prod，必填）
    Status      string         // 状态（active/inactive，必填）
    Remark      string         // 备注
}
```

**表名**: `biz_business_scope`

## CSV 字段定义

| CSV Header | 字段 | 类型 | 必填 | 说明 |
|------------|------|------|------|------|
| code | Code | string | ✅ | 业务域编码，英文字母/数字/下划线/短横线，唯一 |
| name | Name | string | ✅ | 业务域名称 |
| owner | Owner | string | ❌ | 负责人 |
| environment | Environment | string | ✅ | 环境：dev/test/prod |
| status | Status | string | ✅ | 状态：active/inactive |
| remark | Remark | string | ❌ | 备注 |

## 验证规则

### 必填字段验证
- `code`: 必填，唯一性检查
- `name`: 必填
- `environment`: 必填
- `status`: 必填

### 枚举值验证
- `environment` ∈ {dev, test, prod}
- `status` ∈ {active, inactive}

### 业务逻辑验证
1. `code` 在系统中唯一（不区分大小写）
2. `code` 格式：仅允许字母、数字、下划线、短横线
3. 删除检查：如果业务域已绑定主机，不允许删除（但导入更新时不受此限制）

### 唯一性规则
- **创建**: code 不能与现有记录重复
- **更新**: 基于 code 匹配现有记录，允许更新其他字段
- **CSV 内**: 同一个 code 在 CSV 文件中只能出现一次

## 后端实现

### 1. 创建文件: `backend/modules/business/bizscope/bizscope_import_export.go`

```go
package bizscope

import (
    "fmt"
    "regexp"
    "strings"
    "time"

    "pantheon-base/pkg/common"
    "pantheon-base/pkg/impexp"

    "gorm.io/gorm"
)

var maxBizScopeExportRows = 10000

var bizscopeValidEnvironments = map[string]struct{}{
    "dev":  {},
    "test": {},
    "prod": {},
}

var bizscopeValidStatuses = map[string]struct{}{
    "active":   {},
    "inactive": {},
}

var bizscopeCodePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type bizscopeImportRow struct {
    Code        string
    Name        string
    Owner       string
    Environment string
    Status      string
    Remark      string
    Existing    *BizScope
}

func bizscopeImportExportHeaders() []string {
    return []string{
        "code",
        "name",
        "owner",
        "environment",
        "status",
        "remark",
    }
}

// Export 导出业务域
func (s *Service) Export(query *BizScopeListQuery, dataScope *common.DataScopeReq) (*impexp.CSVFile, error) {
    if s.db == nil {
        return nil, common.NewBadRequest("database.not_initialized")
    }

    db := s.db.Model(&BizScope{})
    
    // 应用筛选条件
    if query != nil {
        if query.Code != "" {
            db = db.Where("code LIKE ?", "%"+common.EscapeLikePattern(query.Code)+"%")
        }
        if query.Name != "" {
            db = db.Where("name LIKE ?", "%"+common.EscapeLikePattern(query.Name)+"%")
        }
        if query.Owner != "" {
            db = db.Where("owner LIKE ?", "%"+common.EscapeLikePattern(query.Owner)+"%")
        }
        if query.Environment != "" {
            db = db.Where("environment = ?", query.Environment)
        }
        if query.Status != "" {
            db = db.Where("status = ?", query.Status)
        }
    }

    var scopes []BizScope
    if err := db.Order("id DESC").Limit(maxBizScopeExportRows).Find(&scopes).Error; err != nil {
        return nil, err
    }

    rows := make([][]string, 0, len(scopes))
    for i := range scopes {
        rows = append(rows, []string{
            scopes[i].Code,
            scopes[i].Name,
            scopes[i].Owner,
            scopes[i].Environment,
            scopes[i].Status,
            scopes[i].Remark,
        })
    }

    return &impexp.CSVFile{
        Filename: "bizscope-export.csv",
        Headers:  bizscopeImportExportHeaders(),
        Rows:     rows,
    }, nil
}

// BuildImportTemplate 生成导入模板
func (s *Service) BuildImportTemplate() *impexp.CSVFile {
    return &impexp.CSVFile{
        Filename: "bizscope-import-template.csv",
        Headers:  bizscopeImportExportHeaders(),
        Rows: [][]string{
            {"#说明：code/name/environment/status 必填；code唯一且仅允许字母/数字/下划线/短横线；environment取值dev/test/prod；status取值active/inactive", "", "", "", "", ""},
            {"volcano-prod", "火山引擎生产环境", "ops-team", "prod", "active", "核心业务生产环境"},
            {"volcano-test", "火山引擎测试环境", "qa-team", "test", "active", "功能测试环境"},
            {"internal-dev", "内部开发环境", "dev-team", "dev", "active", "日常开发环境"},
        },
    }
}

// Import 导入业务域
func (s *Service) Import(records [][]string, dataScope *common.DataScopeReq, createdBy string) (*impexp.ImportResult, error) {
    result := &impexp.ImportResult{
        Applied: false,
        Errors:  []impexp.ImportError{},
    }
    if s.db == nil {
        return nil, common.ErrDatabaseNotInitialized
    }
    if len(records) == 0 {
        impexp.AppendImportError(result, 0, "file", "import.file.empty")
        return result, nil
    }

    // 步骤1: 验证表头
    headerIndex := make(map[string]int, len(records[0]))
    for index, header := range records[0] {
        headerIndex[strings.TrimSpace(header)] = index
    }
    for _, header := range []string{"code", "name", "environment", "status"} {
        if _, ok := headerIndex[header]; !ok {
            impexp.AppendImportError(result, 0, header, "import.header.missing")
        }
    }
    if result.Failed > 0 {
        return result, nil
    }

    // 步骤2: 加载现有业务域
    existingByCode, err := s.loadBizScopesForImport()
    if err != nil {
        return nil, err
    }

    // 步骤3: 解析并验证每一行
    rows := make([]bizscopeImportRow, 0, len(records)-1)
    seenCodes := make(map[string]int, len(records)-1)
    for rowIndex := 1; rowIndex < len(records); rowIndex++ {
        record := records[rowIndex]
        if impexp.IsCSVRecordEmpty(record) {
            continue
        }
        rowNumber := rowIndex + 1
        row := parseBizScopeImportRow(record, headerIndex, rowNumber, result)
        
        // 检查 CSV 内重复
        if firstRow, ok := seenCodes[strings.ToLower(row.Code)]; ok && row.Code != "" {
            impexp.AppendImportError(result, rowNumber, "code", fmt.Sprintf("import.duplicate.row.%d", firstRow))
        } else if row.Code != "" {
            seenCodes[strings.ToLower(row.Code)] = rowNumber
        }
        
        row.Existing = existingByCode[strings.ToLower(row.Code)]
        rows = append(rows, row)
    }

    // 步骤4: 如果有错误，提前返回
    if result.Failed > 0 {
        return result, nil
    }

    // 步骤5: 应用导入
    if err := s.applyBizScopeImportRows(result, rows, createdBy); err != nil {
        return nil, err
    }

    result.Applied = true
    return result, nil
}

func (s *Service) loadBizScopesForImport() (map[string]*BizScope, error) {
    var scopes []BizScope
    if err := s.db.Find(&scopes).Error; err != nil {
        return nil, err
    }
    existingByCode := make(map[string]*BizScope, len(scopes))
    for i := range scopes {
        existingByCode[strings.ToLower(scopes[i].Code)] = &scopes[i]
    }
    return existingByCode, nil
}

func parseBizScopeImportRow(
    record []string,
    headerIndex map[string]int,
    rowNumber int,
    result *impexp.ImportResult,
) bizscopeImportRow {
    row := bizscopeImportRow{
        Code:        strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "code")),
        Name:        strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "name")),
        Owner:       strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "owner")),
        Environment: strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "environment")),
        Status:      strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "status")),
        Remark:      strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "remark")),
    }

    // 必填字段验证
    if row.Code == "" {
        impexp.AppendImportError(result, rowNumber, "code", "business.bizscope.code_required")
    }
    if row.Name == "" {
        impexp.AppendImportError(result, rowNumber, "name", "business.bizscope.name_required")
    }
    if row.Environment == "" {
        impexp.AppendImportError(result, rowNumber, "environment", "business.bizscope.environment_required")
    }
    if row.Status == "" {
        impexp.AppendImportError(result, rowNumber, "status", "business.bizscope.status_required")
    }

    // 格式验证
    if row.Code != "" && !bizscopeCodePattern.MatchString(row.Code) {
        impexp.AppendImportError(result, rowNumber, "code", "business.bizscope.code_format_invalid")
    }

    // 枚举值验证
    if row.Environment != "" {
        if _, ok := bizscopeValidEnvironments[row.Environment]; !ok {
            impexp.AppendImportError(result, rowNumber, "environment", "business.bizscope.environment.invalid")
        }
    }
    if row.Status != "" {
        if _, ok := bizscopeValidStatuses[row.Status]; !ok {
            impexp.AppendImportError(result, rowNumber, "status", "business.bizscope.status.invalid")
        }
    }

    return row
}

func (s *Service) applyBizScopeImportRows(result *impexp.ImportResult, rows []bizscopeImportRow, createdBy string) error {
    return s.db.Transaction(func(tx *gorm.DB) error {
        for i := range rows {
            if rows[i].Existing != nil {
                // 更新
                updates := map[string]interface{}{
                    "name":        rows[i].Name,
                    "owner":       rows[i].Owner,
                    "environment": rows[i].Environment,
                    "status":      rows[i].Status,
                    "remark":      rows[i].Remark,
                    "updated_at":  time.Now(),
                }
                if err := tx.Model(rows[i].Existing).Updates(updates).Error; err != nil {
                    return err
                }
                result.Updated++
            } else {
                // 创建
                scope := BizScope{
                    Code:        rows[i].Code,
                    Name:        rows[i].Name,
                    Owner:       rows[i].Owner,
                    Environment: rows[i].Environment,
                    Status:      rows[i].Status,
                    Remark:      rows[i].Remark,
                }
                if err := tx.Create(&scope).Error; err != nil {
                    return err
                }
                result.Created++
            }
        }
        return nil
    })
}
```

### 2. 修改 `backend/modules/business/bizscope/bizscope_handler.go`

在 `RegisterRoutes` 中添加：

```go
func (h *Handler) RegisterRoutes(r gin.IRoutes) {
    r.GET("/bizscope", h.List)
    r.GET("/bizscope/options", h.ListOptions)
    r.POST("/bizscope", h.Create)
    r.GET("/bizscope/:id", h.Get)
    r.PUT("/bizscope/:id", h.Update)
    r.DELETE("/bizscope/:id", h.Delete)
    r.POST("/bizscope/:id/hosts/bind", h.BindHosts)
    r.DELETE("/bizscope/:id/hosts/:hostId", h.UnbindHost)
    
    // 导入导出 ⭐ 新增
    r.GET("/bizscope/export", h.Export)
    r.GET("/bizscope/import-template", h.DownloadImportTemplate)
    r.POST("/bizscope/import", h.Import)
}

func (h *Handler) Export(c *gin.Context) {
    var query BizScopeListQuery
    if err := c.ShouldBindQuery(&query); err != nil {
        common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
        return
    }
    
    dataScope := common.GetDataScope(c)
    csvFile, err := h.svc.Export(&query, dataScope)
    if err != nil {
        common.FailWithError(c, common.CodeError, err, "business.bizscope.export_failed")
        return
    }
    
    impexp.WriteCSVResponse(c, csvFile)
}

func (h *Handler) DownloadImportTemplate(c *gin.Context) {
    csvFile := h.svc.BuildImportTemplate()
    impexp.WriteCSVResponse(c, csvFile)
}

func (h *Handler) Import(c *gin.Context) {
    file, err := c.FormFile("file")
    if err != nil {
        common.Fail(c, common.CodeParamInvalid, "import.file.missing")
        return
    }
    
    records, err := impexp.ParseUploadedCSV(file)
    if err != nil {
        common.FailWithError(c, common.CodeParamInvalid, err, "import.file.invalid")
        return
    }
    
    dataScope := common.GetDataScope(c)
    createdBy := strconv.FormatUint(common.GetUserID(c), 10)
    
    common.SetAuditMetadata(c, "business.bizscope.import", common.BusinessTypeImport)
    result, err := h.svc.Import(records, dataScope, createdBy)
    if err != nil {
        common.FailWithError(c, common.CodeError, err, "business.bizscope.import_failed")
        return
    }
    
    common.Success(c, result)
}
```

### 3. 修改 `backend/modules/business/bizscope/module.go`

在权限定义中添加：

```go
"business:bizscope:export",
"business:bizscope:import",
```

## 前端实现

### 1. 修改 `frontend/src/modules/business/bizscope/api.ts`

添加导入导出函数：

```typescript
import { downloadFile, uploadImportFile } from '../../../api/importExport';

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

### 2. 修改 `frontend/src/modules/business/bizscope/BizScopeList.tsx`

添加导入导出处理：

```typescript
import { ImportCsvButton } from '../../../components';
import { showImportResult } from '../../../api/importExport';
import { IconDownload } from '@arco-design/web-react/icon';

// 添加 handlers
const handleExport = async () => {
  try {
    await exportBizScopes(buildQueryParams());
    Message.success(t('common.exportSuccess'));
  } catch (error) {
    Message.error(t('common.exportFailed'));
  }
};

const handleDownloadTemplate = async () => {
  try {
    await downloadBizScopeImportTemplate();
    Message.success(t('common.downloadSuccess'));
  } catch (error) {
    Message.error(t('common.downloadFailed'));
  }
};

const handleImport = async (file: File) => {
  try {
    const result = await importBizScopes(file);
    showImportResult(result, t, {
      errorFileName: 'bizscope-import-errors.csv',
      autoDownloadErrors: true,
    });
    if (result.applied && result.failed === 0) {
      reload();
    }
  } catch (error) {
    Message.error(t('common.importFailed'));
  }
};

// 在 ListHeaderActions 中添加按钮
<ListHeaderActions>
  <ImportCsvButton
    onImport={handleImport}
    onDownloadTemplate={handleDownloadTemplate}
    disabled={!hasPerm('business:bizscope:import')}
  />
  <Button
    type="secondary"
    icon={<IconDownload />}
    onClick={handleExport}
    disabled={!hasPerm('business:bizscope:export')}
  >
    {t('common.export')}
  </Button>
  <PermissionAction permissionKey="business:bizscope:create">
    <Button type="primary" icon={<IconPlus />} onClick={handleCreate}>
      {t('common.create')}
    </Button>
  </PermissionAction>
</ListHeaderActions>
```

## 国际化

### 中文 (`frontend/src/modules/business/bizscope/locales/zh-CN.json`)

```json
{
  "business.bizscope.code_required": "业务域编码不能为空",
  "business.bizscope.name_required": "业务域名称不能为空",
  "business.bizscope.environment_required": "环境不能为空",
  "business.bizscope.status_required": "状态不能为空",
  "business.bizscope.code_format_invalid": "编码格式无效（仅允许字母/数字/下划线/短横线）",
  "business.bizscope.environment.invalid": "环境值无效（dev/test/prod）",
  "business.bizscope.status.invalid": "状态值无效（active/inactive）",
  "business.bizscope.export_failed": "导出失败",
  "business.bizscope.import_failed": "导入失败"
}
```

### 英文 (`frontend/src/modules/business/bizscope/locales/en-US.json`)

```json
{
  "business.bizscope.code_required": "Code is required",
  "business.bizscope.name_required": "Name is required",
  "business.bizscope.environment_required": "Environment is required",
  "business.bizscope.status_required": "Status is required",
  "business.bizscope.code_format_invalid": "Invalid code format (only letters/digits/underscore/dash allowed)",
  "business.bizscope.environment.invalid": "Invalid environment (dev/test/prod)",
  "business.bizscope.status.invalid": "Invalid status (active/inactive)",
  "business.bizscope.export_failed": "Export failed",
  "business.bizscope.import_failed": "Import failed"
}
```

## 验收标准

- [ ] 后端 3 个接口正常工作（/export, /import-template, /import）
- [ ] 前端 UI 正确集成（导出按钮 + 导入按钮）
- [ ] 权限控制正确（export/import 权限）
- [ ] 导出功能正常（CSV 格式正确，含 BOM）
- [ ] 导入功能正常（创建/更新逻辑正确）
- [ ] Code 格式验证正确
- [ ] 枚举值验证正确
- [ ] 错误处理完善（行号、字段、错误信息清晰）
- [ ] 国际化文案完整（中英文）
- [ ] 审计日志记录正确

---

**准备就绪，等待 CMDB Label 完成后实施。**
