package group

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/impexp"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var maxGroupExportRows = 10000

func groupImportExportHeaders() []string {
	return []string{
		"name",
		"parentName",
		"conditions",
		"description",
	}
}

// Export 导出主机分组
func (s *GroupService) Export(dataScope *common.DataScopeReq) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, common.NewBadRequest("database.not_initialized")
	}

	var groups []Group
	if err := s.db.Order("parent_id ASC, id ASC").Limit(maxGroupExportRows).Find(&groups).Error; err != nil {
		return nil, err
	}

	groupsByID := make(map[uint64]*Group, len(groups))
	for i := range groups {
		groupsByID[groups[i].ID] = &groups[i]
	}

	rows := make([][]string, 0, len(groups))
	for i := range groups {
		parentName := ""
		if groups[i].ParentID != 0 {
			if parent, ok := groupsByID[groups[i].ParentID]; ok {
				parentName = parent.Name
			}
		}

		var conditions string
		if len(groups[i].Conditions) > 0 {
			b, _ := json.Marshal(groups[i].Conditions)
			conditions = string(b)
		}

		rows = append(rows, []string{
			groups[i].Name,
			parentName,
			conditions,
			groups[i].Description,
		})
	}

	return &impexp.CSVFile{
		Filename: "cmdb-group-export.csv",
		Headers:  groupImportExportHeaders(),
		Rows:     rows,
	}, nil
}

// BuildImportTemplate 构建主机分组导入模板
func (s *GroupService) BuildImportTemplate() *impexp.CSVFile {
	return &impexp.CSVFile{
		Filename: "cmdb-group-import-template.csv",
		Headers:  groupImportExportHeaders(),
		Rows: [][]string{
			{"#说明：保留第一行表头；name 必填；parentName 为父分组名称；conditions 为JSON格式的筛选条件。", "", "", ""},
			{"所有服务器", "", "[{\"field\":\"status\",\"op\":\"=\",\"value\":\"operational\"}]", "所有状态为运营中的主机"},
			{"Web服务器", "所有服务器", "[{\"field\":\"label.role\",\"op\":\"=\",\"value\":\"web\"}]", "角色标签为web的主机"},
		},
	}
}

type groupImportRow struct {
	RowNumber   int
	Name        string
	ParentName  string
	Conditions  string
	Description string
	Existing    *Group
}

// Import 导入主机分组
func (s *GroupService) Import(records [][]string, createdBy string) (*impexp.ImportResult, error) {
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

	headerIndex := impexp.BuildHeaderIndex(records)
	impexp.RequireHeaders(result, headerIndex, "name")
	if result.Failed > 0 {
		return result, nil
	}

	existingByName, err := s.loadGroupsForImport()
	if err != nil {
		return nil, err
	}

	rows := parseGroupImportRows(records, headerIndex, result, existingByName)

	if result.Failed > 0 {
		return result, nil
	}
	if err := s.applyGroupImportRows(result, rows, existingByName, createdBy); err != nil {
		return nil, err
	}

	result.Applied = true
	return result, nil
}

func (s *GroupService) loadGroupsForImport() (map[string]*Group, error) {
	var groups []Group
	if err := s.db.Find(&groups).Error; err != nil {
		return nil, err
	}
	existingByName := make(map[string]*Group, len(groups))
	for i := range groups {
		existingByName[groups[i].Name] = &groups[i]
	}
	return existingByName, nil
}

func parseGroupImportRow(
	record []string,
	headerIndex map[string]int,
	rowNumber int,
	result *impexp.ImportResult,
) groupImportRow {
	row := groupImportRow{
		Name:        strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "name")),
		ParentName:  strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "parentName")),
		Conditions:  strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "conditions")),
		Description: strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "description")),
	}

	if row.Name == "" {
		impexp.AppendImportError(result, rowNumber, "name", "business.cmdb.group.name_required")
	}

	if row.Conditions != "" && row.Conditions != "[]" {
		var testConditions []map[string]interface{}
		if err := json.Unmarshal([]byte(row.Conditions), &testConditions); err != nil {
			impexp.AppendImportError(result, rowNumber, "conditions", "business.cmdb.group.conditions_invalid_json")
		}
	}

	return row
}

// parseGroupImportRows turns the data records into group rows, flagging names
// that repeat inside the file and attaching the existing group.
func parseGroupImportRows(records [][]string, headerIndex map[string]int, result *impexp.ImportResult, existingByName map[string]*Group) []groupImportRow {
	rows := make([]groupImportRow, 0, len(records)-1)
	seenName := make(map[string]int, len(records)-1)
	impexp.ForEachDataRow(records, func(rowNumber int, record []string) {
		row := parseGroupImportRow(record, headerIndex, rowNumber, result)
		row.RowNumber = rowNumber
		if row.Name != "" {
			if firstRow, ok := seenName[row.Name]; ok {
				impexp.AppendImportError(result, rowNumber, "name", fmt.Sprintf("import.duplicate.row.%d", firstRow))
			} else {
				seenName[row.Name] = rowNumber
			}
		}
		row.Existing = existingByName[row.Name]
		rows = append(rows, row)
	})
	return rows
}

func (s *GroupService) applyGroupImportRows(result *impexp.ImportResult, rows []groupImportRow, existingByName map[string]*Group, createdBy string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for i := range rows {
			row := rows[i]
			parentID, ok := resolveGroupImportParent(result, row, existingByName)
			if !ok {
				continue
			}

			if row.Existing != nil {
				if err := tx.Model(row.Existing).Updates(buildGroupImportUpdates(parentID, row)).Error; err != nil {
					return err
				}
				result.Updated++
				continue
			}

			group := Group{
				Name:        row.Name,
				ParentID:    parentID,
				Conditions:  groupImportConditions(row),
				Description: row.Description,
			}
			if err := tx.Create(&group).Error; err != nil {
				return err
			}
			existingByName[group.Name] = &group
			result.Created++
		}
		return nil
	})
}

func resolveGroupImportParent(result *impexp.ImportResult, row groupImportRow, existingByName map[string]*Group) (uint64, bool) {
	if row.ParentName == "" {
		return 0, true
	}
	parent, ok := existingByName[row.ParentName]
	if !ok {
		impexp.AppendImportError(result, row.RowNumber, "parentName", "cmdbgroup.parent_not_found")
		return 0, false
	}
	return parent.ID, true
}

func groupImportConditions(row groupImportRow) datatypes.JSON {
	if row.Conditions != "" && row.Conditions != "[]" {
		return datatypes.JSON([]byte(row.Conditions))
	}
	return datatypes.JSON([]byte("[]"))
}

func buildGroupImportUpdates(parentID uint64, row groupImportRow) map[string]interface{} {
	return map[string]interface{}{
		"parent_id":   parentID,
		"conditions":  groupImportConditions(row),
		"description": row.Description,
	}
}
