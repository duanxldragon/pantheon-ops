package org

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"pantheon-base/pkg/common"
	"pantheon-base/pkg/impexp"

	"gorm.io/gorm"
)

// dept_import.go - Import functions for dept module

// deptImportRow represents a single parsed department import row.
type deptImportRow struct {
	RowNumber      int
	ParentDeptPath string
	DeptName       string
	Sort           int
	Leader         string
	Phone          string
	Email          string
	Status         int
}

// deptImportRollbackSentinel signals that the import transaction must roll back
// because a business rule was violated. The caller treats it as a soft failure
// and returns the accumulated result with errors instead of a hard error.
var deptImportRollbackSentinel = common.NewInternal("dept.import.validation_failed")

// ImportDepts imports departments from CSV records.
func (s *DeptService) ImportDepts(records [][]string) (*impexp.ImportResult, error) {
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

	headerIndex, ok := s.buildImportHeaderIndex(records, result)
	if !ok {
		return result, nil
	}

	rows, ok := s.parseImportRows(records, headerIndex, result)
	if !ok {
		return result, nil
	}

	_, pathToID, err := impexp.BuildDeptPathMaps(s.db)
	if err != nil {
		return nil, err
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		return s.upsertImportDeptRows(tx, rows, pathToID, result)
	}); err != nil {
		if errors.Is(err, deptImportRollbackSentinel) {
			return result, nil
		}
		return nil, err
	}

	result.Applied = true
	return result, nil
}

// buildImportHeaderIndex parses the CSV header row into a column index map and
// validates that all required headers are present. It returns the index map and
// reports whether header validation passed (no missing headers).
func (s *DeptService) buildImportHeaderIndex(records [][]string, result *impexp.ImportResult) (map[string]int, bool) {
	headerIndex := make(map[string]int, len(records[0]))
	for index, header := range records[0] {
		headerIndex[strings.TrimSpace(header)] = index
	}
	requiredHeaders := []string{"parentDeptPath", "deptName", "sort", "leader", "phone", "email", "status"}
	for _, header := range requiredHeaders {
		if _, ok := headerIndex[header]; !ok {
			impexp.AppendImportError(result, 0, header, "import.header.missing")
		}
	}
	return headerIndex, result.Failed == 0
}

// parseImportRows parses and validates each data row from the CSV records.
// It returns the parsed rows and reports whether every row passed validation.
func (s *DeptService) parseImportRows(records [][]string, headerIndex map[string]int, result *impexp.ImportResult) ([]deptImportRow, bool) {
	rows := make([]deptImportRow, 0, len(records)-1)
	seenPaths := make(map[string]int, len(records)-1)
	for rowIndex := 1; rowIndex < len(records); rowIndex++ {
		record := records[rowIndex]
		if impexp.IsCSVRecordEmpty(record) {
			continue
		}
		rowNumber := rowIndex + 1
		parentPath := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "parentDeptPath"))
		deptName := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "deptName"))
		sortValue, sortErr := impexp.ParseCSVInt(impexp.ReadCSVField(record, headerIndex, "sort"))
		email := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "email"))
		if parentPath == "" {
			impexp.AppendImportError(result, rowNumber, "parentDeptPath", "dept.parent.required")
		}
		if deptName == "" {
			impexp.AppendImportError(result, rowNumber, "deptName", "dept.name.required")
		}
		if sortErr != nil {
			impexp.AppendImportError(result, rowNumber, "sort", "import.field.invalid_integer")
		}
		if err := validateDeptOptionalEmail(email); err != nil {
			impexp.AppendImportError(result, rowNumber, "email", err.Error())
		}
		fullPath := parentPath + "/" + deptName
		if firstRow, ok := seenPaths[fullPath]; ok {
			impexp.AppendImportError(result, rowNumber, "deptName", fmt.Sprintf("import.duplicate.row.%d", firstRow))
		} else {
			seenPaths[fullPath] = rowNumber
		}
		rows = append(rows, deptImportRow{
			RowNumber:      rowNumber,
			ParentDeptPath: parentPath,
			DeptName:       deptName,
			Sort:           sortValue,
			Leader:         strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "leader")),
			Phone:          strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "phone")),
			Email:          email,
			Status:         impexp.ParseEnabledStatus(impexp.ReadCSVField(record, headerIndex, "status")),
		})
	}
	return rows, result.Failed == 0
}

// upsertImportDeptRows applies the parsed rows inside a transaction by delegating
// each row to upsertSingleImportRow. It propagates the first error encountered
// (including the rollback sentinel for business-rule violations).
func (s *DeptService) upsertImportDeptRows(tx *gorm.DB, rows []deptImportRow, pathToID map[string]uint64, result *impexp.ImportResult) error {
	for _, row := range rows {
		if err := s.upsertSingleImportRow(tx, row, pathToID, result); err != nil {
			return err
		}
	}
	return nil
}

// upsertSingleImportRow looks up an existing department and either creates or
// updates it. It returns the rollback sentinel when a business rule prevents
// the import.
func (s *DeptService) upsertSingleImportRow(tx *gorm.DB, row deptImportRow, pathToID map[string]uint64, result *impexp.ImportResult) error {
	parentID := pathToID[row.ParentDeptPath]
	if parentID == 0 {
		impexp.AppendImportError(result, row.RowNumber, "parentDeptPath", "dept.parent.not_found")
		return deptImportRollbackSentinel
	}

	var dept SystemDept
	err := tx.Where("parent_id = ? AND dept_name = ?", parentID, row.DeptName).First(&dept).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.createImportDept(tx, row, parentID, pathToID, result)
	}
	return s.updateImportDept(tx, &dept, row, pathToID, result)
}

// createImportDept builds the ancestor chain and persists a new department for
// the given row, then records its path in pathToID.
func (s *DeptService) createImportDept(tx *gorm.DB, row deptImportRow, parentID uint64, pathToID map[string]uint64, result *impexp.ImportResult) error {
	ancestors, buildErr := s.buildAncestorsWithDB(tx, parentID)
	if buildErr != nil {
		return buildErr
	}
	dept := SystemDept{
		ParentID:  parentID,
		Ancestors: ancestors,
		IsRoot:    0,
		DeptName:  row.DeptName,
		Sort:      row.Sort,
		Leader:    row.Leader,
		Phone:     row.Phone,
		Email:     row.Email,
		Status:    normalizeSystemStatus(row.Status),
	}
	if err := tx.Create(&dept).Error; err != nil {
		return err
	}
	result.Created++
	pathToID[row.ParentDeptPath+"/"+row.DeptName] = dept.ID
	return nil
}

// updateImportDept refuses to modify a root department and otherwise applies the
// row fields to the existing department, then records its path in pathToID.
func (s *DeptService) updateImportDept(tx *gorm.DB, dept *SystemDept, row deptImportRow, pathToID map[string]uint64, result *impexp.ImportResult) error {
	if dept.IsRoot == common.StatusFlagYes {
		impexp.AppendImportError(result, row.RowNumber, "deptName", "dept.root.update_forbidden")
		return deptImportRollbackSentinel
	}
	dept.Sort = row.Sort
	dept.Leader = row.Leader
	dept.Phone = row.Phone
	dept.Email = row.Email
	dept.Status = normalizeSystemStatus(row.Status)
	if err := tx.Save(dept).Error; err != nil {
		return err
	}
	result.Updated++
	pathToID[row.ParentDeptPath+"/"+row.DeptName] = dept.ID
	return nil
}

// validateDeptOptionalEmail validates optional email field
func validateDeptOptionalEmail(value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	if _, err := mail.ParseAddress(value); err != nil {
		return common.NewBadRequest("dept.email.invalid")
	}
	return nil
}
