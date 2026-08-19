package iam

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"pantheon-base/pkg/common"
	"pantheon-base/pkg/impexp"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// maxRoleExportRows 对齐日志导出的行数上限（var 便于测试降低阈值）。
var maxRoleExportRows = 10000

// roleImportRow 是单条导入记录在解析阶段的载体，区分新建（Create）与更新（Update）两条分支。
type roleImportRow struct {
	RoleKey string
	Create  *RoleCreateReq
	Update  *RoleUpdateReq
	Exist   *SystemRole
}

// roleImportContext 承载 ImportRoles 各解析阶段共享的只读上下文，避免单一超大函数反复传递参数。
type roleImportContext struct {
	headerIndex       map[string]int
	menuIDSet         map[uint64]struct{}
	permissionKeySet  map[string]struct{}
	existingByRoleKey map[string]*SystemRole
	seenRoleKeys      map[string]int
}

// ExportRoles 导出角色 CSV（受 maxRoleExportRows 上限与请求上下文取消约束）。
func (s *RoleService) ExportRoles(ctx context.Context, query *RoleListQuery) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, common.NewBadRequest("database.not_initialized")
	}

	roles, err := s.listRolesForExport(ctx, query)
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

func (s *RoleService) listRolesForExport(ctx context.Context, query *RoleListQuery) ([]SystemRole, error) {
	var roles []SystemRole
	db := s.db.WithContext(ctx).Model(&SystemRole{})
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
		Limit(maxRoleExportRows).
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

// ImportRoles 导入角色 CSV。解析、校验、事务、重复处理、权限、错误消息与返回统计契约
// 与历史实现完全一致，仅通过阶段 helper/上下文对象进行等价提炼以降低认知复杂度。
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

	headerIndex := buildImportHeaderIndex(records)
	validateRequiredImportHeaders(result, headerIndex)
	if result.Failed > 0 {
		return result, nil
	}

	menuIDSet, permissionKeySet, err := s.loadImportReferenceSets()
	if err != nil {
		return nil, err
	}

	existingByRoleKey, err := s.loadRolesByRoleKey()
	if err != nil {
		return nil, err
	}

	ctx := &roleImportContext{
		headerIndex:       headerIndex,
		menuIDSet:         menuIDSet,
		permissionKeySet:  permissionKeySet,
		existingByRoleKey: existingByRoleKey,
		seenRoleKeys:      make(map[string]int, len(records)-1),
	}

	rows := make([]roleImportRow, 0, len(records)-1)
	for rowIndex := 1; rowIndex < len(records); rowIndex++ {
		record := records[rowIndex]
		if impexp.IsCSVRecordEmpty(record) {
			continue
		}
		rowNumber := rowIndex + 1
		rows = append(rows, s.parseImportRow(ctx, result, rowNumber, record))
	}

	if result.Failed > 0 {
		return result, nil
	}

	if err := s.applyImportRows(result, rows); err != nil {
		return nil, err
	}

	result.Applied = true
	return result, nil
}

// buildImportHeaderIndex 构建表头名到列索引的映射。
func buildImportHeaderIndex(records [][]string) map[string]int {
	headerIndex := make(map[string]int, len(records[0]))
	for index, header := range records[0] {
		headerIndex[strings.TrimSpace(header)] = index
	}
	return headerIndex
}

// validateRequiredImportHeaders 校验必填表头是否齐全，缺失时追加导入错误。
func validateRequiredImportHeaders(result *impexp.ImportResult, headerIndex map[string]int) {
	requiredHeaders := []string{"roleName", "roleKey", "sort", "status"}
	for _, header := range requiredHeaders {
		if _, ok := headerIndex[header]; !ok {
			impexp.AppendImportError(result, 0, header, "import.header.missing")
		}
	}
}

// loadImportReferenceSets 预加载合法菜单 ID 集合与合法权限 key 集合，用于逐行校验。
func (s *RoleService) loadImportReferenceSets() (map[uint64]struct{}, map[string]struct{}, error) {
	menuIDSet := make(map[uint64]struct{})
	var allMenus []struct{ ID uint64 }
	if err := s.db.Table("system_menu").Select("id").Scan(&allMenus).Error; err != nil {
		return nil, nil, err
	}
	for _, m := range allMenus {
		menuIDSet[m.ID] = struct{}{}
	}

	permissionKeySet := make(map[string]struct{})
	var allPermissions []struct{ PermissionKey string }
	if err := s.db.Table("system_role_permission").Select("DISTINCT permission_key").Scan(&allPermissions).Error; err != nil {
		return nil, nil, err
	}
	for _, p := range allPermissions {
		if p.PermissionKey != "" {
			permissionKeySet[p.PermissionKey] = struct{}{}
		}
	}

	return menuIDSet, permissionKeySet, nil
}

// parseImportRow 解析并校验单行导入记录，构建对应的新建/更新请求。
// 与历史实现一致：无论前序校验是否失败，仍会执行 validateRoleCreate/validateRoleUpdate，
// 所有错误均累计到 result 中，单行最终仅追加一条 importRow。
func (s *RoleService) parseImportRow(ctx *roleImportContext, result *impexp.ImportResult, rowNumber int, record []string) roleImportRow {
	roleName := strings.TrimSpace(impexp.ReadCSVField(record, ctx.headerIndex, "roleName"))
	roleKey := strings.TrimSpace(impexp.ReadCSVField(record, ctx.headerIndex, "roleKey"))
	sortStr := strings.TrimSpace(impexp.ReadCSVField(record, ctx.headerIndex, "sort"))
	statusStr := strings.TrimSpace(impexp.ReadCSVField(record, ctx.headerIndex, "status"))
	menuIdsStr := strings.TrimSpace(impexp.ReadCSVField(record, ctx.headerIndex, "menuIds"))
	permKeysStr := strings.TrimSpace(impexp.ReadCSVField(record, ctx.headerIndex, "permissionKeys"))

	if roleKey == "" {
		impexp.AppendImportError(result, rowNumber, "roleKey", "role.roleKey.required")
	}
	if roleName == "" {
		impexp.AppendImportError(result, rowNumber, "roleName", "role.roleName.required")
	}
	if firstRow, ok := ctx.seenRoleKeys[roleKey]; ok && roleKey != "" {
		impexp.AppendImportError(result, rowNumber, "roleKey", fmt.Sprintf("import.duplicate.row.%d", firstRow))
	} else if roleKey != "" {
		ctx.seenRoleKeys[roleKey] = rowNumber
	}

	sortValue := parseImportSort(rowNumber, sortStr, result)
	statusValue := parseImportStatus(rowNumber, statusStr, result)
	menuIDs := parseImportMenuIDs(rowNumber, menuIdsStr, ctx.menuIDSet, result)
	permissionKeys := parseImportPermissionKeys(rowNumber, permKeysStr, ctx.permissionKeySet, result)

	existing := ctx.existingByRoleKey[roleKey]
	if existing != nil {
		updateReq := &RoleUpdateReq{
			RoleName:       roleName,
			RoleKey:        roleKey,
			Sort:           sortValue,
			Status:         statusValue,
			MenuIDs:        menuIDs,
			PermissionKeys: permissionKeys,
		}
		if err := s.validateRoleUpdate(existing, updateReq); err != nil {
			impexp.AppendImportError(result, rowNumber, "roleKey", err.Error())
		}
		return roleImportRow{RoleKey: roleKey, Update: updateReq, Exist: existing}
	}

	createReq := &RoleCreateReq{
		RoleName:       roleName,
		RoleKey:        roleKey,
		Sort:           sortValue,
		Status:         statusValue,
		MenuIDs:        menuIDs,
		PermissionKeys: permissionKeys,
	}
	if err := s.validateRoleCreate(createReq); err != nil {
		impexp.AppendImportError(result, rowNumber, "roleKey", err.Error())
	}
	return roleImportRow{RoleKey: roleKey, Create: createReq}
}

// parseImportSort 解析 sort 字段；空值或非整数保持默认 0 并追加错误。
func parseImportSort(rowNumber int, sortStr string, result *impexp.ImportResult) int {
	sortValue := 0
	if sortStr == "" {
		return sortValue
	}
	parsed, err := strconv.Atoi(sortStr)
	if err != nil {
		impexp.AppendImportError(result, rowNumber, "sort", errParamInvalid)
		return sortValue
	}
	return parsed
}

// parseImportStatus 解析 status 字段；仅接受合法启用/禁用值，否则保持默认启用并追加错误。
func parseImportStatus(rowNumber int, statusStr string, result *impexp.ImportResult) int {
	statusValue := common.StatusEnabled
	if statusStr == "" {
		return statusValue
	}
	parsed, err := strconv.Atoi(statusStr)
	if err != nil || !common.IsEnabledStatus(parsed) {
		impexp.AppendImportError(result, rowNumber, "status", errParamInvalid)
		return statusValue
	}
	return parsed
}

// parseImportMenuIDs 解析并校验 menuIds 逗号列表，逐条校验数值与菜单存在性。
func parseImportMenuIDs(rowNumber int, menuIdsStr string, menuIDSet map[uint64]struct{}, result *impexp.ImportResult) []uint64 {
	if menuIdsStr == "" {
		return nil
	}
	menuIDs := make([]uint64, 0)
	for _, idStr := range strings.Split(menuIdsStr, ",") {
		idStr = strings.TrimSpace(idStr)
		if idStr == "" {
			continue
		}
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			impexp.AppendImportError(result, rowNumber, "menuIds", errParamInvalid)
			continue
		}
		if _, ok := menuIDSet[id]; !ok {
			impexp.AppendImportError(result, rowNumber, "menuIds", "role.menu.not_found")
			continue
		}
		menuIDs = append(menuIDs, id)
	}
	return menuIDs
}

// parseImportPermissionKeys 解析并校验 permissionKeys 逗号列表，逐条校验权限 key 存在性。
func parseImportPermissionKeys(rowNumber int, permKeysStr string, permissionKeySet map[string]struct{}, result *impexp.ImportResult) []string {
	if permKeysStr == "" {
		return nil
	}
	permissionKeys := make([]string, 0)
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
	return permissionKeys
}

// applyImportRows 在单一事务中应用所有解析后的导入行，统计 created/updated。
func (s *RoleService) applyImportRows(result *impexp.ImportResult, rows []roleImportRow) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			if err := s.applyImportRow(tx, result, row); err != nil {
				return err
			}
		}
		return nil
	})
}

// applyImportRow 在事务内落盘单行导入（更新已存在角色或新建角色）。
func (s *RoleService) applyImportRow(tx *gorm.DB, result *impexp.ImportResult, row roleImportRow) error {
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
		return nil
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
	return nil
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
