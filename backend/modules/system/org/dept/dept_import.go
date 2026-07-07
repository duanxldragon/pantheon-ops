package org

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/impexp"

	"gorm.io/gorm"
)

// dept_import.go - Import functions for dept module

// ImportDepts imports departments from CSV records
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
	if result.Failed > 0 {
		return result, nil
	}

	type importRow struct {
		RowNumber      int
		ParentDeptPath string
		DeptName       string
		Sort           int
		Leader         string
		Phone          string
		Email          string
		Status         int
	}

	rows := make([]importRow, 0, len(records)-1)
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
		rows = append(rows, importRow{
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

	if result.Failed > 0 {
		return result, nil
	}

	_, pathToID, err := impexp.BuildDeptPathMaps(s.db)
	if err != nil {
		return nil, err
	}
	rollbackValidation := common.NewInternal("dept.import.validation_failed")
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			parentID := pathToID[row.ParentDeptPath]
			if parentID == 0 {
				impexp.AppendImportError(result, row.RowNumber, "parentDeptPath", "dept.parent.not_found")
				return rollbackValidation
			}

			var dept SystemDept
			err := tx.Where("parent_id = ? AND dept_name = ?", parentID, row.DeptName).First(&dept).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			if errors.Is(err, gorm.ErrRecordNotFound) {
				ancestors, buildErr := s.buildAncestorsWithDB(tx, parentID)
				if buildErr != nil {
					return buildErr
				}
				dept = SystemDept{
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
			} else {
				if dept.IsRoot == common.StatusFlagYes {
					impexp.AppendImportError(result, row.RowNumber, "deptName", "dept.root.update_forbidden")
					return rollbackValidation
				}
				dept.Sort = row.Sort
				dept.Leader = row.Leader
				dept.Phone = row.Phone
				dept.Email = row.Email
				dept.Status = normalizeSystemStatus(row.Status)
				if err := tx.Save(&dept).Error; err != nil {
					return err
				}
				result.Updated++
			}
			pathToID[row.ParentDeptPath+"/"+row.DeptName] = dept.ID
		}
		return nil
	}); err != nil {
		if errors.Is(err, rollbackValidation) {
			return result, nil
		}
		return nil, err
	}

	result.Applied = true
	return result, nil
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
