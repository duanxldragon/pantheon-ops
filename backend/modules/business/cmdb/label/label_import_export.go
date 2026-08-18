package label

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"pantheon-base/pkg/common"
	"pantheon-base/pkg/impexp"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const labelStatusEnabled = "enabled"

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

// Export 导出标签架构定义
func (s *LabelService) Export(query LabelSchemaQuery, _ *common.DataScopeReq) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, common.NewBadRequest("database.not_initialized")
	}

	db := s.db.Model(&LabelSchema{})
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
			strconv.FormatBool(labels[i].Required),
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

// BuildImportTemplate 构建标签导入模板
func (s *LabelService) BuildImportTemplate() *impexp.CSVFile {
	return &impexp.CSVFile{
		Filename: "cmdb-label-import-template.csv",
		Headers:  labelImportExportHeaders(),
		Rows: [][]string{
			{"#说明：保留第一行表头；key/name/category/valueMode 必填；category 取值 base/network/business/custom；valueMode 取值 free/enum/dict；options 多个用 \"; \" 分隔；required 填 true/false；status 取值 enabled/disabled。", "", "", "", "", "", "", "", ""},
			{"env", "环境", "base", "enum", "", "dev; test; staging; prod", "true", "enabled", "运行环境标签"},
			{"region", "区域", "network", "free", "", "", "false", "enabled", "地理区域"},
			{"team", "团队", "business", "dict", "dept_teams", "", "false", "enabled", "所属团队（字典）"},
		},
	}
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

	existingByKey, err := s.loadLabelsForImport()
	if err != nil {
		return nil, err
	}

	rows := make([]labelImportRow, 0, len(records)-1)
	seenKey := make(map[string]int, len(records)-1)
	for rowIndex := 1; rowIndex < len(records); rowIndex++ {
		record := records[rowIndex]
		if impexp.IsCSVRecordEmpty(record) {
			continue
		}
		rowNumber := rowIndex + 1
		row := parseLabelImportRow(record, headerIndex, rowNumber, result)
		if firstRow, ok := seenKey[strings.ToLower(row.Key)]; ok && row.Key != "" {
			impexp.AppendImportError(result, rowNumber, "key", fmt.Sprintf("import.duplicate.row.%d", firstRow))
		} else if row.Key != "" {
			seenKey[strings.ToLower(row.Key)] = rowNumber
		}
		row.Existing = existingByKey[strings.ToLower(row.Key)]
		rows = append(rows, row)
	}

	if result.Failed > 0 {
		return result, nil
	}
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

	optionsRaw := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "options"))
	if optionsRaw != "" {
		for _, item := range strings.FieldsFunc(optionsRaw, func(r rune) bool {
			return r == ';' || r == ','
		}) {
			item = strings.TrimSpace(item)
			if item != "" {
				row.Options = append(row.Options, item)
			}
		}
	}

	requiredRaw := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "required"))
	if requiredRaw != "" {
		row.Required = strings.ToLower(requiredRaw) == "true" || requiredRaw == "1"
	}

	if row.ValueMode == "enum" && len(row.Options) == 0 {
		impexp.AppendImportError(result, rowNumber, "options", "cmdblabel.options_required_for_enum")
	}
	if row.ValueMode == "dict" && row.DictCode == "" {
		impexp.AppendImportError(result, rowNumber, "dictCode", "cmdblabel.dictCode_required_for_dict")
	}

	if row.Status == "" {
		row.Status = labelStatusEnabled
	}

	return row
}

func (s *LabelService) applyLabelImportRows(result *impexp.ImportResult, rows []labelImportRow, createdBy string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for i := range rows {
			var optionsJSON datatypes.JSON
			if len(rows[i].Options) > 0 {
				optionsBytes, err := json.Marshal(rows[i].Options)
				if err != nil {
					return err
				}
				optionsJSON = datatypes.JSON(optionsBytes)
			}

			if rows[i].Existing != nil {
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
				continue
			}

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
		return nil
	})
}
