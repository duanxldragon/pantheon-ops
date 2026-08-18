# CMDB Label 导入导出功能实现规范

**模块**: Business > CMDB > Label (标签架构)  
**优先级**: P0 (最简单，先突破)  
**预计工期**: 0.4 天

## 数据模型

```go
type LabelSchema struct {
    ID          uint64
    Key         string         // 标签键（唯一，必填）
    Name        string         // 显示名称（必填）
    Category    string         // 分类（base/network/business/custom，必填）
    ValueMode   string         // 值模式（free/enum/dict，必填）
    DictCode    string         // 字典编码（valueMode=dict时必填）
    Options     datatypes.JSON // 选项列表（valueMode=enum时必填）
    Required    bool           // 是否必填
    Status      string         // 状态（enabled/disabled）
    Description string         // 描述
}
```

**表名**: `biz_cmdb_label_schema`

## CSV 字段定义

| CSV Header | 字段 | 类型 | 必填 | 说明 |
|------------|------|------|------|------|
| key | Key | string | ✅ | 标签键，英文字母/数字/下划线，唯一 |
| name | Name | string | ✅ | 显示名称 |
| category | Category | string | ✅ | 分类：base/network/business/custom |
| valueMode | ValueMode | string | ✅ | 值模式：free/enum/dict |
| dictCode | DictCode | string | ❌ | 字典编码（valueMode=dict时必填） |
| options | Options | string | ❌ | 枚举选项（valueMode=enum时必填），格式：option1;option2;option3 |
| required | Required | string | ❌ | 是否必填：true/false，默认 false |
| status | Status | string | ❌ | 状态：enabled/disabled，默认 enabled |
| description | Description | string | ❌ | 描述 |

## 验证规则

### 必填字段验证
- `key`: 必填，唯一性检查
- `name`: 必填
- `category`: 必填
- `valueMode`: 必填

### 枚举值验证
- `category` ∈ {base, network, business, custom}
- `valueMode` ∈ {free, enum, dict}
- `status` ∈ {enabled, disabled}
- `required` ∈ {true, false, ""}

### 业务逻辑验证
1. 如果 `valueMode = enum`，则 `options` 必填且非空
2. 如果 `valueMode = dict`，则 `dictCode` 必填且非空
3. `key` 在系统中唯一（不区分大小写）
4. `options` 格式：用分号分隔的选项列表

### 唯一性规则
- **创建**: key 不能与现有记录重复
- **更新**: 基于 key 匹配现有记录，允许更新其他字段
- **CSV 内**: 同一个 key 在 CSV 文件中只能出现一次

## 后端实现

### 1. 创建文件: `backend/modules/business/cmdb/label/label_import_export.go`

```go
package label

import (
    "encoding/json"
    "fmt"
    "strings"
    "time"

    "pantheon-base/pkg/common"
    "pantheon-base/pkg/impexp"

    "gorm.io/datatypes"
    "gorm.io/gorm"
)

var maxLabelExportRows = 10000

var labelValidCategories = map[string]struct{}{
    "base":     {},
    "network":  {},
    "business": {},
    "custom":   {},
}

var labelValidValueModes = map[string]struct{}{
    "free": {},
    "enum": {},
    "dict": {},
}

var labelValidStatuses = map[string]struct{}{
    "enabled":  {},
    "disabled": {},
}

type labelImportRow struct {
    Key         string
    Name        string
    Category    string
    ValueMode   string
    DictCode    string
    Options     []string
    Required    bool
    Status      string
    Description string
    Existing    *LabelSchema
}

func labelImportExportHeaders() []string {
    return []string{
        "key",
        "name",
        "category",
        "valueMode",
        "dictCode",
        "options",
        "required",
        "status",
        "description",
    }
}

// Export 导出标签架构
func (s *LabelService) Export(query LabelSchemaQuery) (*impexp.CSVFile, error) {
    if s.db == nil {
        return nil, common.NewBadRequest("database.not_initialized")
    }

    db := s.db.Model(&LabelSchema{})
    
    // 应用筛选条件
    if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
        like := "%" + common.EscapeLikePattern(keyword) + "%"
        db = db.Where("`key` LIKE ? OR name LIKE ?", like, like)
    }
    if query.Status != "" {
        db = db.Where("status = ?", query.Status)
    }
    if query.Category != "" {
        db = db.Where("category = ?", query.Category)
    }

    var labels []LabelSchema
    if err := db.Order("category ASC, id DESC").Limit(maxLabelExportRows).Find(&labels).Error; err != nil {
        return nil, err
    }

    rows := make([][]string, 0, len(labels))
    for i := range labels {
        rows = append(rows, []string{
            labels[i].Key,
            labels[i].Name,
            labels[i].Category,
            labels[i].ValueMode,
            labels[i].DictCode,
            formatLabelOptions(labels[i].Options),
            formatBool(labels[i].Required),
            labels[i].Status,
            labels[i].Description,
        })
    }

    return &impexp.CSVFile{
        Filename: "cmdb-label-export.csv",
        Headers:  labelImportExportHeaders(),
        Rows:     rows,
    }, nil
}

func formatLabelOptions(raw datatypes.JSON) string {
    if len(raw) == 0 {
        return ""
    }
    var options []string
    if err := json.Unmarshal(raw, &options); err != nil {
        return ""
    }
    return strings.Join(options, "; ")
}

func formatBool(value bool) string {
    if value {
        return "true"
    }
    return "false"
}

// BuildImportTemplate 生成导入模板
func (s *LabelService) BuildImportTemplate() *impexp.CSVFile {
    return &impexp.CSVFile{
        Filename: "cmdb-label-import-template.csv",
        Headers:  labelImportExportHeaders(),
        Rows: [][]string{
            {"#说明：key/name/category/valueMode 必填；category取值base/network/business/custom；valueMode取值free/enum/dict；options多个用\"; \"分隔；required取值true/false；status取值enabled/disabled", "", "", "", "", "", "", "", ""},
            {"env", "环境", "base", "enum", "", "dev; test; staging; prod", "true", "enabled", "运行环境标签"},
            {"region", "区域", "network", "free", "", "", "false", "enabled", "地理区域"},
            {"team", "团队", "business", "dict", "dept_teams", "", "false", "enabled", "所属团队（字典）"},
        },
    }
}

// Import 导入标签架构
func (s *LabelService) Import(records [][]string, createdBy string) (*impexp.ImportResult, error) {
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
    for _, header := range []string{"key", "name", "category", "valueMode"} {
        if _, ok := headerIndex[header]; !ok {
            impexp.AppendImportError(result, 0, header, "import.header.missing")
        }
    }
    if result.Failed > 0 {
        return result, nil
    }

    // 步骤2: 加载现有标签
    existingByKey, err := s.loadLabelsForImport()
    if err != nil {
        return nil, err
    }

    // 步骤3: 解析并验证每一行
    rows := make([]labelImportRow, 0, len(records)-1)
    seenKeys := make(map[string]int, len(records)-1)
    for rowIndex := 1; rowIndex < len(records); rowIndex++ {
        record := records[rowIndex]
        if impexp.IsCSVRecordEmpty(record) {
            continue
        }
        rowNumber := rowIndex + 1
        row := parseLabelImportRow(record, headerIndex, rowNumber, result)
        
        // 检查 CSV 内重复
        if firstRow, ok := seenKeys[strings.ToLower(row.Key)]; ok && row.Key != "" {
            impexp.AppendImportError(result, rowNumber, "key", fmt.Sprintf("import.duplicate.row.%d", firstRow))
        } else if row.Key != "" {
            seenKeys[strings.ToLower(row.Key)] = rowNumber
        }
        
        row.Existing = existingByKey[strings.ToLower(row.Key)]
        rows = append(rows, row)
    }

    // 步骤4: 如果有错误，提前返回
    if result.Failed > 0 {
        return result, nil
    }

    // 步骤5: 应用导入
    if err := s.applyLabelImportRows(result, rows, createdBy); err != nil {
        return nil, err
    }

    result.Applied = true
    return result, nil
}

func (s *LabelService) loadLabelsForImport() (map[string]*LabelSchema, error) {
    var labels []LabelSchema
    if err := s.db.Find(&labels).Error; err != nil {
        return nil, err
    }
    existingByKey := make(map[string]*LabelSchema, len(labels))
    for i := range labels {
        existingByKey[strings.ToLower(labels[i].Key)] = &labels[i]
    }
    return existingByKey, nil
}

func parseLabelImportRow(
    record []string,
    headerIndex map[string]int,
    rowNumber int,
    result *impexp.ImportResult,
) labelImportRow {
    row := labelImportRow{
        Key:         strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "key")),
        Name:        strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "name")),
        Category:    strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "category")),
        ValueMode:   strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "valueMode")),
        DictCode:    strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "dictCode")),
        Status:      strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "status")),
        Description: strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "description")),
    }

    // 必填字段验证
    if row.Key == "" {
        impexp.AppendImportError(result, rowNumber, "key", "cmdblabel.key_required")
    }
    if row.Name == "" {
        impexp.AppendImportError(result, rowNumber, "name", "cmdblabel.name_required")
    }
    if row.Category == "" {
        impexp.AppendImportError(result, rowNumber, "category", "cmdblabel.category_required")
    }
    if row.ValueMode == "" {
        impexp.AppendImportError(result, rowNumber, "valueMode", "cmdblabel.valueMode_required")
    }

    // 枚举值验证
    if row.Category != "" {
        if _, ok := labelValidCategories[row.Category]; !ok {
            impexp.AppendImportError(result, rowNumber, "category", "cmdblabel.category.invalid")
        }
    }
    if row.ValueMode != "" {
        if _, ok := labelValidValueModes[row.ValueMode]; !ok {
            impexp.AppendImportError(result, rowNumber, "valueMode", "cmdblabel.valueMode.invalid")
        }
    }
    if row.Status != "" {
        if _, ok := labelValidStatuses[row.Status]; !ok {
            impexp.AppendImportError(result, rowNumber, "status", "cmdblabel.status.invalid")
        }
    }

    // 解析 options
    optionsRaw := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "options"))
    if optionsRaw != "" {
        row.Options = parseLabelOptions(optionsRaw)
    }

    // 解析 required
    requiredRaw := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "required"))
    if requiredRaw != "" {
        row.Required = strings.ToLower(requiredRaw) == "true"
    }

    // 业务逻辑验证
    if row.ValueMode == "enum" && len(row.Options) == 0 {
        impexp.AppendImportError(result, rowNumber, "options", "cmdblabel.options_required_for_enum")
    }
    if row.ValueMode == "dict" && row.DictCode == "" {
        impexp.AppendImportError(result, rowNumber, "dictCode", "cmdblabel.dictCode_required_for_dict")
    }

    // 默认值
    if row.Status == "" {
        row.Status = "enabled"
    }

    return row
}

func parseLabelOptions(raw string) []string {
    options := make([]string, 0)
    for _, item := range strings.FieldsFunc(raw, func(r rune) bool {
        return r == ';' || r == ','
    }) {
        item = strings.TrimSpace(item)
        if item != "" {
            options = append(options, item)
        }
    }
    return options
}

func (s *LabelService) applyLabelImportRows(result *impexp.ImportResult, rows []labelImportRow, createdBy string) error {
    return s.db.Transaction(func(tx *gorm.DB) error {
        for i := range rows {
            var optionsJSON datatypes.JSON
            if len(rows[i].Options) > 0 {
                data, err := json.Marshal(rows[i].Options)
                if err != nil {
                    return err
                }
                optionsJSON = datatypes.JSON(data)
            }

            if rows[i].Existing != nil {
                // 更新
                updates := map[string]interface{}{
                    "name":        rows[i].Name,
                    "category":    rows[i].Category,
                    "value_mode":  rows[i].ValueMode,
                    "dict_code":   rows[i].DictCode,
                    "options":     optionsJSON,
                    "required":    rows[i].Required,
                    "status":      rows[i].Status,
                    "description": rows[i].Description,
                    "updated_at":  time.Now(),
                }
                if err := tx.Model(rows[i].Existing).Updates(updates).Error; err != nil {
                    return err
                }
                result.Updated++
            } else {
                // 创建
                label := LabelSchema{
                    Key:         rows[i].Key,
                    Name:        rows[i].Name,
                    Category:    rows[i].Category,
                    ValueMode:   rows[i].ValueMode,
                    DictCode:    rows[i].DictCode,
                    Options:     optionsJSON,
                    Required:    rows[i].Required,
                    Status:      rows[i].Status,
                    Description: rows[i].Description,
                }
                if err := tx.Create(&label).Error; err != nil {
                    return err
                }
                result.Created++
            }
        }
        return nil
    })
}
```

### 2. 修改 `backend/modules/business/cmdb/label/label_handler.go`

在 `RegisterRoutes` 中添加：

```go
func (h *LabelHandler) RegisterRoutes(r gin.IRoutes) {
    r.GET("/labels", h.List)
    r.GET("/labels/options", h.ListOptions)
    r.POST("/labels", h.Create)
    r.PUT("/labels/:id", h.Update)
    r.DELETE("/labels/:id", h.Delete)
    
    // 导入导出 ⭐ 新增
    r.GET("/labels/export", h.Export)
    r.GET("/labels/import-template", h.DownloadImportTemplate)
    r.POST("/labels/import", h.Import)
}

func (h *LabelHandler) Export(c *gin.Context) {
    var query LabelSchemaQuery
    if err := c.ShouldBindQuery(&query); err != nil {
        common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
        return
    }
    
    csvFile, err := h.svc.Export(query)
    if err != nil {
        common.FailWithError(c, common.CodeError, err, "cmdblabel.export_failed")
        return
    }
    
    impexp.WriteCSVResponse(c, csvFile)
}

func (h *LabelHandler) DownloadImportTemplate(c *gin.Context) {
    csvFile := h.svc.BuildImportTemplate()
    impexp.WriteCSVResponse(c, csvFile)
}

func (h *LabelHandler) Import(c *gin.Context) {
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
    
    createdBy := strconv.FormatUint(common.GetUserID(c), 10)
    
    common.SetAuditMetadata(c, "cmdblabel.import", common.BusinessTypeImport)
    result, err := h.svc.Import(records, createdBy)
    if err != nil {
        common.FailWithError(c, common.CodeError, err, "cmdblabel.import_failed")
        return
    }
    
    common.Success(c, result)
}
```

### 3. 修改 `backend/modules/business/cmdb/module.go`

在权限定义中添加：

```go
"business:cmdb:label:export",
"business:cmdb:label:import",
```

## 前端实现

### 1. 修改 `frontend/src/modules/business/cmdb/label/api.ts`

添加导入导出函数：

```typescript
import { downloadFile, uploadImportFile } from '../../../../api/importExport';

export function exportLabels(params?: LabelSchemaQuery) {
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

### 2. 修改 `frontend/src/modules/business/cmdb/label/LabelList.tsx`

在组件中添加导入导出处理：

```typescript
import { ImportCsvButton } from '../../../../components';
import { showImportResult } from '../../../../api/importExport';
import { IconDownload } from '@arco-design/web-react/icon';

// 添加 handlers
const handleExport = async () => {
  try {
    await exportLabels(buildQueryParams());
    Message.success(t('common.exportSuccess'));
  } catch (error) {
    Message.error(t('common.exportFailed'));
  }
};

const handleDownloadTemplate = async () => {
  try {
    await downloadLabelImportTemplate();
    Message.success(t('common.downloadSuccess'));
  } catch (error) {
    Message.error(t('common.downloadFailed'));
  }
};

const handleImport = async (file: File) => {
  try {
    const result = await importLabels(file);
    showImportResult(result, t, {
      errorFileName: 'cmdb-label-import-errors.csv',
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
    disabled={!hasPerm('business:cmdb:label:import')}
  />
  <Button
    type="secondary"
    icon={<IconDownload />}
    onClick={handleExport}
    disabled={!hasPerm('business:cmdb:label:export')}
  >
    {t('common.export')}
  </Button>
  <PermissionAction permissionKey="business:cmdb:label:create">
    <Button type="primary" icon={<IconPlus />} onClick={handleCreate}>
      {t('common.create')}
    </Button>
  </PermissionAction>
</ListHeaderActions>
```

## 国际化

### 中文 (`frontend/src/modules/business/cmdb/locales/zh-CN.json`)

```json
{
  "cmdblabel.key_required": "标签键不能为空",
  "cmdblabel.name_required": "名称不能为空",
  "cmdblabel.category_required": "分类不能为空",
  "cmdblabel.valueMode_required": "值模式不能为空",
  "cmdblabel.category.invalid": "分类值无效（base/network/business/custom）",
  "cmdblabel.valueMode.invalid": "值模式无效（free/enum/dict）",
  "cmdblabel.status.invalid": "状态值无效（enabled/disabled）",
  "cmdblabel.options_required_for_enum": "枚举模式下选项不能为空",
  "cmdblabel.dictCode_required_for_dict": "字典模式下字典编码不能为空",
  "cmdblabel.export_failed": "导出失败",
  "cmdblabel.import_failed": "导入失败"
}
```

### 英文 (`frontend/src/modules/business/cmdb/locales/en-US.json`)

```json
{
  "cmdblabel.key_required": "Key is required",
  "cmdblabel.name_required": "Name is required",
  "cmdblabel.category_required": "Category is required",
  "cmdblabel.valueMode_required": "Value mode is required",
  "cmdblabel.category.invalid": "Invalid category (base/network/business/custom)",
  "cmdblabel.valueMode.invalid": "Invalid value mode (free/enum/dict)",
  "cmdblabel.status.invalid": "Invalid status (enabled/disabled)",
  "cmdblabel.options_required_for_enum": "Options are required for enum mode",
  "cmdblabel.dictCode_required_for_dict": "Dict code is required for dict mode",
  "cmdblabel.export_failed": "Export failed",
  "cmdblabel.import_failed": "Import failed"
}
```

## 测试用例

### 1. 导出测试
- 空数据导出
- 正常数据导出
- 带筛选条件导出（按 category, status, keyword）
- 大量数据导出（接近 10,000 行限制）

### 2. 模板下载测试
- 模板格式正确
- 包含示例数据

### 3. 导入测试
- **空文件**: 返回错误
- **缺少必填表头**: 返回表头错误
- **缺少必填字段**: 返回字段错误
- **枚举值无效**: category/valueMode/status 错误
- **CSV 内重复 key**: 返回重复错误
- **业务逻辑错误**: enum 模式缺少 options, dict 模式缺少 dictCode
- **正常创建**: created 计数正确
- **正常更新**: 基于 key 匹配，updated 计数正确
- **混合创建和更新**: 统计正确

## 验收标准

- [ ] 后端 3 个接口正常工作（/export, /import-template, /import）
- [ ] 前端 UI 正确集成（导出按钮 + 导入按钮）
- [ ] 权限控制正确（export/import 权限）
- [ ] 导出功能正常（CSV 格式正确，含 BOM）
- [ ] 导入功能正常（创建/更新逻辑正确）
- [ ] 错误处理完善（行号、字段、错误信息清晰）
- [ ] 国际化文案完整（中英文）
- [ ] 审计日志记录正确

---

**准备就绪，可以委托 Codex 实现。**
