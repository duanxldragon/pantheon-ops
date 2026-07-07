package system

import (
	"errors"
	"fmt"
	"strings"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/impexp"

	"gorm.io/gorm"
)

func (s *I18nService) Export(query *I18nQuery) (*impexp.CSVFile, error) {
	query = normalizeI18nQuery(query)

	db := s.db.Model(&SystemI18n{})
	if query.Module != "" {
		db = db.Where("module = ?", query.Module)
	}
	if query.Group != "" {
		db = db.Where("group_name = ?", query.Group)
	}
	if query.Locale != "" {
		db = db.Where("locale = ?", query.Locale)
	}
	if query.Key != "" {
		db = db.Where("`key` LIKE ?", "%"+common.EscapeLikePattern(query.Key)+"%")
	}

	var rows []SystemI18n
	if err := db.Order("locale ASC").Order("module ASC").Order("`key` ASC").Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([][]string, 0, len(rows))
	for _, row := range rows {
		result = append(result, []string{
			row.Module,
			row.Group,
			row.Key,
			row.Locale,
			row.Value,
			row.Remark,
			row.CreatedAt.Format("2006-01-02 15:04:05"),
			row.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &impexp.CSVFile{
		Filename: "system-i18n-export.csv",
		Headers:  []string{"module", "group", "key", "locale", "value", "remark", "createdAt", "updatedAt"},
		Rows:     result,
	}, nil
}

func (s *I18nService) BuildImportTemplate() *impexp.CSVFile {
	return &impexp.CSVFile{
		Filename: "system-i18n-import-template.csv",
		Headers:  []string{"module", "group", "key", "locale", "value", "remark"},
		Rows: [][]string{
			{"#说明：保留第一行表头；group 为空时默认 messages；module/key/locale/value 必填；已存在记录按 locale + key 更新 value/remark/group；若 module 与现有记录归属不一致，该行会被阻断。", "", "", "", "", ""},
			{"system.config", "messages", "i18n.sample.key", "zh-CN", "示例文案", "sample"},
			{"system.config", "messages", "i18n.sample.key", "en-US", "Sample Text", "sample"},
		},
	}
}

func (s *I18nService) Import(records [][]string) (*impexp.ImportResult, error) {
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
	requiredHeaders := []string{"module", "group", "key", "locale", "value", "remark"}
	for _, header := range requiredHeaders {
		if _, ok := headerIndex[header]; !ok {
			impexp.AppendImportError(result, 0, header, "import.header.missing")
		}
	}
	if result.Failed > 0 {
		return result, nil
	}

	type importRow struct {
		module string
		group  string
		key    string
		locale string
		value  string
		remark string
	}

	type validatedImportRow struct {
		importRow
		rowNumber int
	}

	rows := make([]validatedImportRow, 0, len(records)-1)
	seen := make(map[string]int, len(records)-1)
	for rowIndex := 1; rowIndex < len(records); rowIndex++ {
		record := records[rowIndex]
		if impexp.IsCSVRecordEmpty(record) || impexp.IsCSVRecordBlank(record) {
			continue
		}
		rowNumber := rowIndex + 1
		module := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "module"))
		group := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "group"))
		key := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "key"))
		locale := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "locale"))
		value := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "value"))
		remark := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "remark"))
		if group == "" {
			group = "messages"
		}

		if module == "" {
			impexp.AppendImportError(result, rowNumber, "module", "i18n.module.required")
		}
		if key == "" {
			impexp.AppendImportError(result, rowNumber, "key", "i18n.key.required")
		}
		if locale == "" {
			impexp.AppendImportError(result, rowNumber, "locale", "i18n.locale.required")
		}
		if value == "" {
			impexp.AppendImportError(result, rowNumber, "value", "i18n.value.required")
		}

		duplicateKey := fmt.Sprintf("%s|%s|%s", module, key, locale)
		if firstRow, ok := seen[duplicateKey]; ok {
			impexp.AppendImportError(result, rowNumber, "key", fmt.Sprintf("import.duplicate.row.%d", firstRow))
		} else {
			seen[duplicateKey] = rowNumber
		}

		rows = append(rows, validatedImportRow{
			importRow: importRow{
				module: module,
				group:  group,
				key:    key,
				locale: locale,
				value:  value,
				remark: remark,
			},
			rowNumber: rowNumber,
		})
	}
	if result.Failed > 0 {
		return result, nil
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			var existing SystemI18n
			err := tx.Where("locale = ? AND `key` = ?", row.locale, row.key).First(&existing).Error
			switch {
			case err == nil:
				if strings.TrimSpace(existing.Module) != "" && strings.TrimSpace(existing.Module) != row.module {
					impexp.AppendImportError(result, row.rowNumber, "module", fmt.Sprintf("import.conflict.owner.%s", existing.Module))
					continue
				}
				if err := tx.Model(&existing).Updates(map[string]interface{}{
					"module":     row.module,
					"group_name": row.group,
					"value":      row.value,
					"remark":     row.remark,
				}).Error; err != nil {
					return err
				}
				result.Updated++
			case errors.Is(err, gorm.ErrRecordNotFound):
				if err := tx.Create(&SystemI18n{
					Module: row.module,
					Group:  row.group,
					Key:    row.key,
					Locale: row.locale,
					Value:  row.value,
					Remark: row.remark,
				}).Error; err != nil {
					return err
				}
				result.Created++
			default:
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	result.Applied = result.Created > 0 || result.Updated > 0
	return result, s.ReloadCache()
}
