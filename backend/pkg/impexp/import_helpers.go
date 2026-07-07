package impexp

import (
	"pantheon-ops/backend/pkg/common"
	"sort"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

type ImportError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ImportResult struct {
	Applied bool          `json:"applied"`
	Created int           `json:"created"`
	Updated int           `json:"updated"`
	Failed  int           `json:"failed"`
	Errors  []ImportError `json:"errors"`
}

func AppendImportError(result *ImportResult, row int, field string, message string) {
	result.Failed++
	result.Errors = append(result.Errors, ImportError{
		Row:     row,
		Field:   field,
		Message: message,
	})
}

func JoinStringSlice(values []string, separator string) string {
	filtered := make([]string, 0, len(values))
	for _, item := range values {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	sort.Strings(filtered)
	return strings.Join(filtered, separator)
}

func SplitPipeValues(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	parts := strings.Split(raw, "|")
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	sort.Strings(result)
	return result
}

func ParseEnabledStatus(raw string) int {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "2", "disabled", "disable", "false", "inactive":
		return 2
	default:
		return 1
	}
}

func BuildDeptPathMaps(db *gorm.DB) (map[uint64]string, map[string]uint64, error) {
	if db == nil {
		return nil, nil, common.NewBadRequest("database.not_initialized")
	}

	type deptRow struct {
		ID       uint64 `gorm:"column:id"`
		ParentID uint64 `gorm:"column:parent_id"`
		DeptName string `gorm:"column:dept_name"`
	}

	var rows []deptRow
	if err := db.Table("system_dept").
		Select("id, parent_id, dept_name").
		Order("id asc").
		Scan(&rows).Error; err != nil {
		return nil, nil, err
	}

	byID := make(map[uint64]deptRow, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}

	pathByID := make(map[uint64]string, len(rows))
	visiting := make(map[uint64]bool, len(rows))
	var resolvePath func(id uint64) (string, error)
	resolvePath = func(id uint64) (string, error) {
		if id == 0 {
			return "", nil
		}
		if path, ok := pathByID[id]; ok {
			return path, nil
		}
		row, ok := byID[id]
		if !ok {
			return "", common.NewNotFound("dept.not_found")
		}
		if visiting[id] {
			return "", common.NewBadRequest("dept.path.circular")
		}
		visiting[id] = true
		defer func() {
			delete(visiting, id)
		}()

		parentPath, err := resolvePath(row.ParentID)
		if err != nil {
			return "", err
		}
		path := strings.TrimSpace(row.DeptName)
		if parentPath != "" {
			path = parentPath + "/" + path
		}
		pathByID[id] = path
		return path, nil
	}

	pathToID := make(map[string]uint64, len(rows))
	for _, row := range rows {
		path, err := resolvePath(row.ID)
		if err != nil {
			return nil, nil, err
		}
		pathToID[path] = row.ID
	}

	return pathByID, pathToID, nil
}

func ReadCSVField(record []string, headerIndex map[string]int, key string) string {
	index, ok := headerIndex[key]
	if !ok || index < 0 || index >= len(record) {
		return ""
	}
	return record[index]
}

func IsCSVRecordEmpty(record []string) bool {
	for _, item := range record {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		return strings.HasPrefix(trimmed, "#")
	}
	return true
}

func IsCSVRecordBlank(record []string) bool {
	for _, item := range record {
		if strings.TrimSpace(item) != "" {
			return false
		}
	}
	return true
}

func ParseCSVInt(value string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	return strconv.Atoi(strings.TrimSpace(value))
}
