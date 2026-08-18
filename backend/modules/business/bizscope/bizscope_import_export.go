package bizscope

import (
	"fmt"
	"strings"

	"pantheon-base/pkg/common"
	"pantheon-base/pkg/database"
	"pantheon-base/pkg/impexp"

	"gorm.io/gorm"
)

var maxBizScopeExportRows = 10000

var bizscopeImportEnvironments = map[string]struct{}{
	"dev":  {},
	"test": {},
	"prod": {},
}

var bizscopeImportStatuses = map[string]struct{}{
	"active":   {},
	"inactive": {},
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
func (s *Service) Export(query BizScopeListQuery, dataScope *common.DataScopeReq) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, common.NewBadRequest("database.not_initialized")
	}

	db := s.db.Model(&BizScope{}).Scopes(database.WithDataScope(dataScope))
	if code := strings.TrimSpace(query.Code); code != "" {
		like := "%" + common.EscapeLikePattern(code) + "%"
		db = db.Where("code LIKE ?", like)
	}
	if name := strings.TrimSpace(query.Name); name != "" {
		like := "%" + common.EscapeLikePattern(name) + "%"
		db = db.Where("name LIKE ?", like)
	}
	if query.Environment != "" {
		db = db.Where("environment = ?", query.Environment)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.DeptID > 0 {
		db = db.Where("dept_id = ?", query.DeptID)
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
		Filename: "business-scope-export.csv",
		Headers:  bizscopeImportExportHeaders(),
		Rows:     rows,
	}, nil
}

// BuildImportTemplate 构建业务域导入模板
func (s *Service) BuildImportTemplate() *impexp.CSVFile {
	return &impexp.CSVFile{
		Filename: "business-scope-import-template.csv",
		Headers:  bizscopeImportExportHeaders(),
		Rows: [][]string{
			{"#说明：保留第一行表头；code/name/environment 必填；environment 取值 dev/test/prod；status 取值 active/inactive，留空默认 active。", "", "", "", "", ""},
			{"volcano-prod", "火山平台-生产", "ops-team", "prod", "active", "核心业务生产环境"},
			{"volcano-test", "火山平台-测试", "ops-team", "test", "active", "测试环境"},
		},
	}
}

type bizscopeImportRow struct {
	Code        string
	Name        string
	Owner       string
	Environment string
	Status      string
	Remark      string
	Existing    *BizScope
}

// Import 导入业务域
func (s *Service) Import(records [][]string, createdBy string) (*impexp.ImportResult, error) {
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
	for _, header := range []string{"code", "name", "environment"} {
		if _, ok := headerIndex[header]; !ok {
			impexp.AppendImportError(result, 0, header, "import.header.missing")
		}
	}
	if result.Failed > 0 {
		return result, nil
	}

	existingByCode, err := s.loadBizScopesForImport()
	if err != nil {
		return nil, err
	}

	rows := make([]bizscopeImportRow, 0, len(records)-1)
	seenCode := make(map[string]int, len(records)-1)
	for rowIndex := 1; rowIndex < len(records); rowIndex++ {
		record := records[rowIndex]
		if impexp.IsCSVRecordEmpty(record) {
			continue
		}
		rowNumber := rowIndex + 1
		row := parseBizscopeImportRow(record, headerIndex, rowNumber, result)
		if firstRow, ok := seenCode[row.Code]; ok && row.Code != "" {
			impexp.AppendImportError(result, rowNumber, "code", fmt.Sprintf("import.duplicate.row.%d", firstRow))
		} else if row.Code != "" {
			seenCode[row.Code] = rowNumber
		}
		row.Existing = existingByCode[row.Code]
		rows = append(rows, row)
	}

	if result.Failed > 0 {
		return result, nil
	}
	if err := s.applyBizscopeImportRows(result, rows, createdBy); err != nil {
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
		existingByCode[scopes[i].Code] = &scopes[i]
	}
	return existingByCode, nil
}

func parseBizscopeImportRow(
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

	if row.Code == "" {
		impexp.AppendImportError(result, rowNumber, "code", "business.bizscope.code_required")
	}
	if row.Name == "" {
		impexp.AppendImportError(result, rowNumber, "name", "business.bizscope.name_required")
	}
	if row.Environment == "" {
		impexp.AppendImportError(result, rowNumber, "environment", "business.bizscope.environment_required")
	} else if _, ok := bizscopeImportEnvironments[row.Environment]; !ok {
		impexp.AppendImportError(result, rowNumber, "environment", "business.bizscope.environment_invalid")
	}

	if row.Status != "" {
		if _, ok := bizscopeImportStatuses[row.Status]; !ok {
			impexp.AppendImportError(result, rowNumber, "status", "business.bizscope.status_invalid")
		}
	}

	return row
}

func (s *Service) applyBizscopeImportRows(result *impexp.ImportResult, rows []bizscopeImportRow, createdBy string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for i := range rows {
			if rows[i].Existing != nil {
				// NOTE: biz_business_scope has no created_by/updated_by audit columns
				// (see BizScope model and migration 000012), so the import does not
				// write them here.
				updates := map[string]interface{}{
					"name":        rows[i].Name,
					"owner":       rows[i].Owner,
					"environment": rows[i].Environment,
					"remark":      rows[i].Remark,
				}
				if rows[i].Status != "" {
					updates["status"] = rows[i].Status
				}
				if err := tx.Model(rows[i].Existing).Updates(updates).Error; err != nil {
					return err
				}
				result.Updated++
				continue
			}

			status := rows[i].Status
			if status == "" {
				status = "active"
			}
			scope := BizScope{
				Code:        rows[i].Code,
				Name:        rows[i].Name,
				Owner:       rows[i].Owner,
				Environment: rows[i].Environment,
				Status:      status,
				Remark:      rows[i].Remark,
			}
			if err := tx.Create(&scope).Error; err != nil {
				return err
			}
			result.Created++
		}
		return nil
	})
}
