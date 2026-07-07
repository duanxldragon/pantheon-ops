package iam

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/impexp"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *RoleService) ExportRoles(query *RoleListQuery) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, common.NewBadRequest("database.not_initialized")
	}

	roles, err := s.listRolesForExport(query)
	if err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(roles))
	for _, role := range roles {
		rows = append(rows, []string{
			role.RoleName,
			role.RoleKey,
			fmt.Sprintf("%d", role.Sort),
			fmt.Sprintf("%d", role.Status),
			role.CreatedAt.Format(time.RFC3339),
		})
	}

	return &impexp.CSVFile{
		Filename: "system-role-export.csv",
		Headers:  []string{"roleName", "roleKey", "sort", "status", "createdAt"},
		Rows:     rows,
	}, nil
}

func (s *RoleService) listRolesForExport(query *RoleListQuery) ([]SystemRole, error) {
	var roles []SystemRole
	db := s.db.Model(&SystemRole{})
	if query != nil {
		if strings.TrimSpace(query.RoleName) != "" {
			db = db.Where("role_name LIKE ?", fmt.Sprintf("%%%s%%", common.EscapeLikePattern(strings.TrimSpace(query.RoleName))))
		}
		if strings.TrimSpace(query.RoleKey) != "" {
			db = db.Where("role_key LIKE ?", fmt.Sprintf("%%%s%%", common.EscapeLikePattern(strings.TrimSpace(query.RoleKey))))
		}
		if query.Status != nil && common.IsEnabledStatus(*query.Status) {
			db = db.Where("status = ?", *query.Status)
		}
	}

	sortColumn, sortDesc := normalizeRoleSort(query)
	if err := db.
		Order(clause.OrderByColumn{Column: clause.Column{Name: sortColumn}, Desc: sortDesc}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}, Desc: false}).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (s *RoleService) BuildRoleImportTemplate() *impexp.CSVFile {
	return &impexp.CSVFile{
		Filename: "system-role-import-template.csv",
		Headers:  []string{"roleName", "roleKey", "sort", "status", "menuIds", "permissionKeys"},
		Rows: [][]string{
			{"#说明：保留第一行表头；status 使用 1=启用、2=禁用；menuIds 多个用逗号分隔；permissionKeys 多个用逗号分隔；roleKey 唯一标识，创建后不可修改。", "", "", "", "", ""},
			{"#示例角色", "example_role", "99", "1", "1,2,3", "system:user:list,system:user:create"},
		},
	}
}

func (s *RoleService) ImportRoles(records [][]string) (*impexp.ImportResult, error) {
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
	requiredHeaders := []string{"roleName", "roleKey", "sort", "status"}
	for _, header := range requiredHeaders {
		if _, ok := headerIndex[header]; !ok {
			impexp.AppendImportError(result, 0, header, "import.header.missing")
		}
	}
	if result.Failed > 0 {
		return result, nil
	}

	menuIDSet := make(map[uint64]struct{})
	var allMenus []struct{ ID uint64 }
	if err := s.db.Table("system_menu").Select("id").Scan(&allMenus).Error; err != nil {
		return nil, err
	}
	for _, m := range allMenus {
		menuIDSet[m.ID] = struct{}{}
	}

	permissionKeySet := make(map[string]struct{})
	var allPermissions []struct{ PermissionKey string }
	if err := s.db.Table("system_role_permission").Select("DISTINCT permission_key").Scan(&allPermissions).Error; err != nil {
		return nil, err
	}
	for _, p := range allPermissions {
		if p.PermissionKey != "" {
			permissionKeySet[p.PermissionKey] = struct{}{}
		}
	}

	existingByRoleKey, err := s.loadRolesByRoleKey()
	if err != nil {
		return nil, err
	}

	type importRow struct {
		RoleKey string
		Create  *RoleCreateReq
		Update  *RoleUpdateReq
		Exist   *SystemRole
	}

	rows := make([]importRow, 0, len(records)-1)
	seenRoleKeys := make(map[string]int, len(records)-1)

	for rowIndex := 1; rowIndex < len(records); rowIndex++ {
		record := records[rowIndex]
		if impexp.IsCSVRecordEmpty(record) {
			continue
		}
		rowNumber := rowIndex + 1

		roleName := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "roleName"))
		roleKey := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "roleKey"))
		sortStr := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "sort"))
		statusStr := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "status"))
		menuIdsStr := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "menuIds"))
		permKeysStr := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "permissionKeys"))

		if roleKey == "" {
			impexp.AppendImportError(result, rowNumber, "roleKey", "role.roleKey.required")
		}
		if roleName == "" {
			impexp.AppendImportError(result, rowNumber, "roleName", "role.roleName.required")
		}
		if firstRow, ok := seenRoleKeys[roleKey]; ok && roleKey != "" {
			impexp.AppendImportError(result, rowNumber, "roleKey", fmt.Sprintf("import.duplicate.row.%d", firstRow))
		} else if roleKey != "" {
			seenRoleKeys[roleKey] = rowNumber
		}

		var sort int
		if sortStr != "" {
			if parsed, err := strconv.Atoi(sortStr); err == nil {
				sort = parsed
			} else {
				impexp.AppendImportError(result, rowNumber, "sort", "param.invalid")
			}
		}

		status := common.StatusEnabled
		if statusStr != "" {
			if parsed, err := strconv.Atoi(statusStr); err == nil && common.IsEnabledStatus(parsed) {
				status = parsed
			} else {
				impexp.AppendImportError(result, rowNumber, "status", "param.invalid")
			}
		}

		var menuIDs []uint64
		if menuIdsStr != "" {
			for _, idStr := range strings.Split(menuIdsStr, ",") {
				idStr = strings.TrimSpace(idStr)
				if idStr == "" {
					continue
				}
				id, err := strconv.ParseUint(idStr, 10, 64)
				if err != nil {
					impexp.AppendImportError(result, rowNumber, "menuIds", "param.invalid")
					continue
				}
				if _, ok := menuIDSet[id]; !ok {
					impexp.AppendImportError(result, rowNumber, "menuIds", "role.menu.not_found")
					continue
				}
				menuIDs = append(menuIDs, id)
			}
		}

		var permissionKeys []string
		if permKeysStr != "" {
			for _, key := range strings.Split(permKeysStr, ",") {
				key = strings.TrimSpace(key)
				if key == "" {
					continue
				}
				if _, ok := permissionKeySet[key]; !ok {
					impexp.AppendImportError(result, rowNumber, "permissionKeys", "role.permission.not_found")
					continue
				}
				permissionKeys = append(permissionKeys, key)
			}
		}

		existing := existingByRoleKey[roleKey]
		if existing != nil {
			updateReq := &RoleUpdateReq{
				RoleName:       roleName,
				RoleKey:        roleKey,
				Sort:           sort,
				Status:         status,
				MenuIDs:        menuIDs,
				PermissionKeys: permissionKeys,
			}
			if err := s.validateRoleUpdate(existing, updateReq); err != nil {
				impexp.AppendImportError(result, rowNumber, "roleKey", err.Error())
			}
			rows = append(rows, importRow{
				RoleKey: roleKey,
				Update:  updateReq,
				Exist:   existing,
			})
			continue
		}

		createReq := &RoleCreateReq{
			RoleName:       roleName,
			RoleKey:        roleKey,
			Sort:           sort,
			Status:         status,
			MenuIDs:        menuIDs,
			PermissionKeys: permissionKeys,
		}
		if err := s.validateRoleCreate(createReq); err != nil {
			impexp.AppendImportError(result, rowNumber, "roleKey", err.Error())
		}
		rows = append(rows, importRow{
			RoleKey: roleKey,
			Create:  createReq,
		})
	}

	if result.Failed > 0 {
		return result, nil
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			if row.Update != nil && row.Exist != nil {
				updates := map[string]interface{}{
					"role_name": row.Update.RoleName,
					"sort":      row.Update.Sort,
					"status":    normalizeRoleStatus(row.Update.Status),
				}
				if err := tx.Model(row.Exist).Updates(updates).Error; err != nil {
					return err
				}
				if err := s.replaceRoleMenus(tx, row.Exist.ID, row.Update.MenuIDs); err != nil {
					return err
				}
				if err := s.replaceRolePermissions(tx, row.Exist.ID, row.Update.PermissionKeys); err != nil {
					return err
				}
				result.Updated++
				continue
			}

			role := SystemRole{
				RoleName: row.Create.RoleName,
				RoleKey:  row.Create.RoleKey,
				Sort:     row.Create.Sort,
				Status:   normalizeRoleStatus(row.Create.Status),
			}
			if err := tx.Create(&role).Error; err != nil {
				return err
			}
			if err := s.replaceRoleMenus(tx, role.ID, row.Create.MenuIDs); err != nil {
				return err
			}
			if err := s.replaceRolePermissions(tx, role.ID, row.Create.PermissionKeys); err != nil {
				return err
			}
			result.Created++
		}
		return nil
	}); err != nil {
		return nil, err
	}

	result.Applied = true
	return result, nil
}

func (s *RoleService) loadRolesByRoleKey() (map[string]*SystemRole, error) {
	result := make(map[string]*SystemRole)
	var roles []SystemRole
	if err := s.db.Find(&roles).Error; err != nil {
		return nil, err
	}
	for i := range roles {
		result[roles[i].RoleKey] = &roles[i]
	}
	return result, nil
}
