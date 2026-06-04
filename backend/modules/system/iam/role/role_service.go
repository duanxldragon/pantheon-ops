package iam

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"pantheon-platform/backend/pkg/database"
	"pantheon-platform/backend/pkg/impexp"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RoleService struct {
	db *gorm.DB
}

const deletedRoleKeyPrefix = "__deleted_role_"

func NewRoleService(db *gorm.DB) *RoleService {
	return &RoleService{db: db}
}

func (s *RoleService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	if err := s.db.AutoMigrate(&SystemRole{}, &SystemRolePermission{}, &SystemRoleMenu{}); err != nil {
		return err
	}
	if err := s.releaseDeletedRoleKeys(); err != nil {
		return err
	}
	if err := s.ensureAdminRoleSeed(); err != nil {
		return err
	}
	if err := s.ensureAdminUserBinding(); err != nil {
		return err
	}
	if !s.db.Migrator().HasTable("system_menu") {
		return nil
	}
	return s.backfillRolePermissions()
}

func (s *RoleService) ensureAdminRoleSeed() error {
	var adminRole SystemRole
	err := s.db.Where("role_key = ?", "admin").First(&adminRole).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		adminRole = SystemRole{
			RoleName: "超级管理员",
			RoleKey:  "admin",
			Sort:     1,
			Status:   1,
		}
		var count int64
		if err := s.db.Unscoped().Model(&SystemRole{}).Where("id = ?", 1).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			adminRole.ID = 1
		}
		return s.db.Create(&adminRole).Error
	case err != nil:
		return err
	default:
		updates := map[string]interface{}{}
		if strings.TrimSpace(adminRole.RoleName) == "" {
			updates["role_name"] = "超级管理员"
		}
		if adminRole.Sort == 0 {
			updates["sort"] = 1
		}
		if adminRole.Status != 1 {
			updates["status"] = 1
		}
		if len(updates) == 0 {
			return nil
		}
		return s.db.Model(&adminRole).Updates(updates).Error
	}
}

func (s *RoleService) ensureAdminUserBinding() error {
	if !s.db.Migrator().HasTable("system_user_role") || !s.db.Migrator().HasTable("system_user") {
		return nil
	}

	var adminRoleID uint64
	if err := s.db.Table("system_role").Select("id").Where("role_key = ?", "admin").Limit(1).Pluck("id", &adminRoleID).Error; err != nil {
		return err
	}
	if adminRoleID == 0 {
		return nil
	}

	var adminUserCount int64
	if err := s.db.Table("system_user").Where("id = ?", 1).Count(&adminUserCount).Error; err != nil {
		return err
	}
	if adminUserCount == 0 {
		return nil
	}

	var bindingCount int64
	if err := s.db.Table("system_user_role").Where("user_id = ? AND role_id = ?", 1, adminRoleID).Count(&bindingCount).Error; err != nil {
		return err
	}
	if bindingCount > 0 {
		return nil
	}
	return s.db.Exec("INSERT INTO system_user_role (user_id, role_id) VALUES (?, ?)", 1, adminRoleID).Error
}

// ListRoles 获取角色分页列表。
func (s *RoleService) ListRoles(query *RoleListQuery) (*RoleListPageResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	var roles []SystemRole
	db := s.db.Model(&SystemRole{})
	page, pageSize := normalizeRolePageQuery(query)
	if query != nil {
		if strings.TrimSpace(query.RoleName) != "" {
			db = db.Where("role_name LIKE ?", fmt.Sprintf("%%%s%%", strings.TrimSpace(query.RoleName)))
		}
		if strings.TrimSpace(query.RoleKey) != "" {
			db = db.Where("role_key LIKE ?", fmt.Sprintf("%%%s%%", strings.TrimSpace(query.RoleKey)))
		}
		if query.Status != nil && (*query.Status == 1 || *query.Status == 2) {
			db = db.Where("status = ?", *query.Status)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	sortColumn, sortDesc := normalizeRoleSort(query)
	if err := db.
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: sortColumn},
			Desc:   sortDesc,
		}).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&roles).Error; err != nil {
		return nil, err
	}

	roleIDs := make([]uint64, 0, len(roles))
	for _, item := range roles {
		roleIDs = append(roleIDs, item.ID)
	}

	roleMenus, err := s.loadRoleMenus(roleIDs)
	if err != nil {
		return nil, err
	}
	rolePermissions, err := s.loadRolePermissions(roleIDs)
	if err != nil {
		return nil, err
	}

	items := make([]RoleListResp, 0, len(roles))
	for _, item := range roles {
		items = append(items, RoleListResp{
			ID:             item.ID,
			RoleName:       item.RoleName,
			RoleKey:        item.RoleKey,
			Sort:           item.Sort,
			Status:         item.Status,
			CreatedAt:      item.CreatedAt.Format(time.RFC3339),
			MenuIDs:        roleMenus[item.ID],
			PermissionKeys: rolePermissions[item.ID],
		})
	}

	return &RoleListPageResp{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *RoleService) ListRoleMembers(roleID uint64, query *RoleMemberQuery) (*RoleMemberPageResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if _, err := s.getRole(roleID); err != nil {
		return nil, err
	}
	return s.listUsersByRoleMembership(roleID, query, true)
}

func (s *RoleService) ListAssignableUsers(roleID uint64, query *RoleMemberQuery) (*RoleMemberPageResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if _, err := s.getRole(roleID); err != nil {
		return nil, err
	}
	return s.listUsersByRoleMembership(roleID, query, false)
}

func (s *RoleService) AddRoleMembers(roleID uint64, userIDs []uint64) (int, error) {
	if s.db == nil {
		return 0, errors.New("database.not_initialized")
	}
	if _, err := s.getRole(roleID); err != nil {
		return 0, err
	}

	normalizedUserIDs := normalizeUint64IDs(userIDs)
	if len(normalizedUserIDs) == 0 {
		return 0, errors.New("user.batch.empty")
	}
	if err := s.ensureUsersExist(normalizedUserIDs); err != nil {
		return 0, err
	}

	var existingRows []struct {
		UserID uint64 `gorm:"column:user_id"`
	}
	if err := s.db.Table("system_user_role").
		Select("user_id").
		Where("role_id = ? AND user_id IN ?", roleID, normalizedUserIDs).
		Scan(&existingRows).Error; err != nil {
		return 0, err
	}

	existing := make(map[uint64]struct{}, len(existingRows))
	for _, row := range existingRows {
		existing[row.UserID] = struct{}{}
	}

	addedCount := 0
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, userID := range normalizedUserIDs {
			if _, ok := existing[userID]; ok {
				continue
			}
			if err := tx.Exec(
				"INSERT INTO system_user_role (user_id, role_id) VALUES (?, ?)",
				userID,
				roleID,
			).Error; err != nil {
				return err
			}
			addedCount++
		}
		return nil
	}); err != nil {
		return 0, err
	}

	return addedCount, nil
}

func (s *RoleService) RemoveRoleMembers(roleID uint64, userIDs []uint64) (int, error) {
	if s.db == nil {
		return 0, errors.New("database.not_initialized")
	}
	role, err := s.getRole(roleID)
	if err != nil {
		return 0, err
	}

	normalizedUserIDs := normalizeUint64IDs(userIDs)
	if len(normalizedUserIDs) == 0 {
		return 0, errors.New("user.batch.empty")
	}
	if err := s.ensureUsersExist(normalizedUserIDs); err != nil {
		return 0, err
	}
	if role.RoleKey == "admin" {
		for _, userID := range normalizedUserIDs {
			if userID == 1 {
				return 0, errors.New("user.update.error.protected")
			}
		}
	}

	result := s.db.Table("system_user_role").
		Where("role_id = ? AND user_id IN ?", roleID, normalizedUserIDs).
		Delete(nil)
	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}

// CreateRole 创建角色。
func (s *RoleService) CreateRole(req *RoleCreateReq) (*RoleListResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if err := s.validateRoleCreate(req); err != nil {
		return nil, err
	}

	role := SystemRole{
		RoleName: strings.TrimSpace(req.RoleName),
		RoleKey:  strings.TrimSpace(req.RoleKey),
		Sort:     req.Sort,
		Status:   normalizeRoleStatus(req.Status),
	}
	menuIDs := normalizeUint64IDs(req.MenuIDs)
	permissionKeys := normalizePermissionKeys(req.PermissionKeys)

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&role).Error; err != nil {
			return err
		}
		if err := s.replaceRoleMenus(tx, role.ID, menuIDs); err != nil {
			return err
		}
		return s.replaceRolePermissions(tx, role.ID, permissionKeys)
	}); err != nil {
		return nil, err
	}

	return &RoleListResp{
		ID:             role.ID,
		RoleName:       role.RoleName,
		RoleKey:        role.RoleKey,
		Sort:           role.Sort,
		Status:         role.Status,
		CreatedAt:      role.CreatedAt.Format(time.RFC3339),
		MenuIDs:        menuIDs,
		PermissionKeys: permissionKeys,
	}, nil
}

// UpdateRole 更新角色。
func (s *RoleService) UpdateRole(roleID uint64, req *RoleUpdateReq) (*RoleListResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	var role SystemRole
	if err := s.db.First(&role, roleID).Error; err != nil {
		return nil, err
	}
	if err := s.validateRoleUpdate(&role, req); err != nil {
		return nil, err
	}

	role.RoleName = strings.TrimSpace(req.RoleName)
	role.RoleKey = strings.TrimSpace(req.RoleKey)
	role.Sort = req.Sort
	role.Status = normalizeRoleStatus(req.Status)
	menuIDs := normalizeUint64IDs(req.MenuIDs)
	permissionKeys := normalizePermissionKeys(req.PermissionKeys)

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&role).Error; err != nil {
			return err
		}
		if err := s.replaceRoleMenus(tx, role.ID, menuIDs); err != nil {
			return err
		}
		return s.replaceRolePermissions(tx, role.ID, permissionKeys)
	}); err != nil {
		return nil, err
	}

	return &RoleListResp{
		ID:             role.ID,
		RoleName:       role.RoleName,
		RoleKey:        role.RoleKey,
		Sort:           role.Sort,
		Status:         role.Status,
		CreatedAt:      role.CreatedAt.Format(time.RFC3339),
		MenuIDs:        menuIDs,
		PermissionKeys: permissionKeys,
	}, nil
}

// DeleteRole 删除角色。
func (s *RoleService) DeleteRole(roleID uint64) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}

	var role SystemRole
	if err := s.db.First(&role, roleID).Error; err != nil {
		return err
	}
	if role.ID == 1 || role.RoleKey == "admin" {
		return errors.New("role.delete.error.protected")
	}

	var userCount int64
	if err := s.db.Table("system_user_role").Where("role_id = ?", roleID).Count(&userCount).Error; err != nil {
		return err
	}
	if userCount > 0 {
		return errors.New("role.delete.error.has_users")
	}

	roleKey := role.RoleKey
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM system_role_menu WHERE role_id = ?", roleID).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM system_role_permission WHERE role_id = ?", roleID).Error; err != nil {
			return err
		}
		if err := tx.Model(&database.CasbinRule{}).
			Where("ptype = ? AND v0 = ?", "p", roleKey).
			Delete(&database.CasbinRule{}).Error; err != nil {
			return err
		}
		deletedRoleKey, err := s.allocateDeletedRoleKey(tx, role.ID)
		if err != nil {
			return err
		}
		if err := tx.Model(&role).Update("role_key", deletedRoleKey).Error; err != nil {
			return err
		}
		return tx.Delete(&role).Error
	}); err != nil {
		return err
	}

	return reloadRolePolicies()
}

func (s *RoleService) ExportRoles(query *RoleListQuery) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
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

func (s *RoleService) BatchUpdateRoleStatus(roleIDs []uint64, status int) (int, error) {
	if s.db == nil {
		return 0, errors.New("database.not_initialized")
	}
	normalizedIDs := normalizeUint64IDs(roleIDs)
	if len(normalizedIDs) == 0 {
		return 0, errors.New("role.batch.empty")
	}
	if status != 1 && status != 2 {
		return 0, errors.New("param.invalid")
	}

	var roles []SystemRole
	if err := s.db.Where("id IN ?", normalizedIDs).Find(&roles).Error; err != nil {
		return 0, err
	}
	if len(roles) != len(normalizedIDs) {
		return 0, errors.New("role.batch.not_found")
	}
	if status == 2 {
		for _, role := range roles {
			if role.ID == 1 || role.RoleKey == "admin" {
				return 0, errors.New("role.update.error.protected")
			}
		}
	}

	if err := s.db.Model(&SystemRole{}).
		Where("id IN ?", normalizedIDs).
		Updates(map[string]any{
			"status":     normalizeRoleStatus(status),
			"updated_at": time.Now(),
		}).Error; err != nil {
		return 0, err
	}

	return len(normalizedIDs), nil
}

func (s *RoleService) listRolesForExport(query *RoleListQuery) ([]SystemRole, error) {
	var roles []SystemRole
	db := s.db.Model(&SystemRole{})
	if query != nil {
		if strings.TrimSpace(query.RoleName) != "" {
			db = db.Where("role_name LIKE ?", fmt.Sprintf("%%%s%%", strings.TrimSpace(query.RoleName)))
		}
		if strings.TrimSpace(query.RoleKey) != "" {
			db = db.Where("role_key LIKE ?", fmt.Sprintf("%%%s%%", strings.TrimSpace(query.RoleKey)))
		}
		if query.Status != nil && (*query.Status == 1 || *query.Status == 2) {
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

func (s *RoleService) loadRoleMenus(roleIDs []uint64) (map[uint64][]uint64, error) {
	result := make(map[uint64][]uint64, len(roleIDs))
	if len(roleIDs) == 0 {
		return result, nil
	}

	type roleMenuPair struct {
		RoleID uint64 `gorm:"column:role_id"`
		MenuID uint64 `gorm:"column:menu_id"`
	}

	var pairs []roleMenuPair
	if err := s.db.Table("system_role_menu").
		Select("system_role_menu.role_id, system_role_menu.menu_id").
		Joins("JOIN system_menu ON system_menu.id = system_role_menu.menu_id").
		Where("system_role_menu.role_id IN ? AND system_menu.type <> ?", roleIDs, "F").
		Order("menu_id asc").
		Scan(&pairs).Error; err != nil {
		return nil, err
	}

	for _, pair := range pairs {
		result[pair.RoleID] = append(result[pair.RoleID], pair.MenuID)
	}
	return result, nil
}

func (s *RoleService) loadRolePermissions(roleIDs []uint64) (map[uint64][]string, error) {
	result := make(map[uint64][]string, len(roleIDs))
	if len(roleIDs) == 0 {
		return result, nil
	}

	type rolePermissionPair struct {
		RoleID        uint64 `gorm:"column:role_id"`
		PermissionKey string `gorm:"column:permission_key"`
	}

	var pairs []rolePermissionPair
	if err := s.db.Table("system_role_permission").
		Select("role_id, permission_key").
		Where("role_id IN ?", roleIDs).
		Order("permission_key asc").
		Scan(&pairs).Error; err != nil {
		return nil, err
	}

	for _, pair := range pairs {
		result[pair.RoleID] = append(result[pair.RoleID], pair.PermissionKey)
	}
	return result, nil
}

func (s *RoleService) getRole(roleID uint64) (*SystemRole, error) {
	var role SystemRole
	if err := s.db.First(&role, roleID).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (s *RoleService) ensureUsersExist(userIDs []uint64) error {
	var count int64
	if err := s.db.Table("system_user").Where("id IN ?", userIDs).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(userIDs)) {
		return errors.New("user.batch.not_found")
	}
	return nil
}

func (s *RoleService) listUsersByRoleMembership(roleID uint64, query *RoleMemberQuery, assigned bool) (*RoleMemberPageResp, error) {
	type roleMemberRow struct {
		ID        uint64    `gorm:"column:id"`
		Username  string    `gorm:"column:username"`
		Nickname  string    `gorm:"column:nickname"`
		DeptID    uint64    `gorm:"column:dept_id"`
		DeptName  string    `gorm:"column:dept_name"`
		PostID    uint64    `gorm:"column:post_id"`
		PostName  string    `gorm:"column:post_name"`
		Status    int       `gorm:"column:status"`
		CreatedAt time.Time `gorm:"column:created_at"`
	}

	page, pageSize := normalizeRoleMemberPageQuery(query)
	db := s.db.Table("system_user").
		Select(strings.Join([]string{
			"system_user.id",
			"system_user.username",
			"system_user.nickname",
			"system_user.dept_id",
			"COALESCE(system_dept.dept_name, '') AS dept_name",
			"system_user.post_id",
			"COALESCE(system_post.post_name, '') AS post_name",
			"system_user.status",
			"system_user.created_at",
		}, ", ")).
		Joins("LEFT JOIN system_dept ON system_dept.id = system_user.dept_id").
		Joins("LEFT JOIN system_post ON system_post.id = system_user.post_id")

	if assigned {
		db = db.Joins(
			"JOIN system_user_role ON system_user_role.user_id = system_user.id AND system_user_role.role_id = ?",
			roleID,
		)
	} else {
		db = db.Where(
			"NOT EXISTS (SELECT 1 FROM system_user_role WHERE system_user_role.user_id = system_user.id AND system_user_role.role_id = ?)",
			roleID,
		)
	}

	if query != nil {
		keyword := strings.TrimSpace(query.Keyword)
		if keyword != "" {
			db = db.Where(
				"(system_user.username LIKE ? OR system_user.nickname LIKE ?)",
				fmt.Sprintf("%%%s%%", keyword),
				fmt.Sprintf("%%%s%%", keyword),
			)
		}
		if query.Status != nil && (*query.Status == 1 || *query.Status == 2) {
			db = db.Where("system_user.status = ?", *query.Status)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	var rows []roleMemberRow
	if err := db.
		Order("system_user.created_at desc, system_user.id desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]RoleMemberResp, 0, len(rows))
	for _, row := range rows {
		items = append(items, RoleMemberResp{
			ID:        row.ID,
			Username:  row.Username,
			Nickname:  row.Nickname,
			DeptID:    row.DeptID,
			DeptName:  row.DeptName,
			PostID:    row.PostID,
			PostName:  row.PostName,
			Status:    row.Status,
			CreatedAt: row.CreatedAt.Format(time.RFC3339),
		})
	}

	return &RoleMemberPageResp{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *RoleService) validateRoleCreate(req *RoleCreateReq) error {
	if strings.TrimSpace(req.RoleName) == "" || strings.TrimSpace(req.RoleKey) == "" {
		return errors.New("param.invalid")
	}
	if err := s.ensureRoleKeyUnique(0, req.RoleKey); err != nil {
		return err
	}
	if err := s.ensureMenuIDsExist(req.MenuIDs); err != nil {
		return err
	}
	return s.ensurePermissionKeysExist(req.PermissionKeys)
}

func (s *RoleService) validateRoleUpdate(role *SystemRole, req *RoleUpdateReq) error {
	if strings.TrimSpace(req.RoleName) == "" || strings.TrimSpace(req.RoleKey) == "" {
		return errors.New("param.invalid")
	}
	if role.RoleKey == "admin" && (strings.TrimSpace(req.RoleKey) != "admin" || req.Status == 2) {
		return errors.New("role.update.error.protected")
	}
	if err := s.ensureRoleKeyUnique(role.ID, req.RoleKey); err != nil {
		return err
	}
	if err := s.ensureMenuIDsExist(req.MenuIDs); err != nil {
		return err
	}
	return s.ensurePermissionKeysExist(req.PermissionKeys)
}

func (s *RoleService) ensureRoleKeyUnique(roleID uint64, roleKey string) error {
	var count int64
	db := s.db.Model(&SystemRole{}).Where("role_key = ?", strings.TrimSpace(roleKey))
	if roleID > 0 {
		db = db.Where("id <> ?", roleID)
	}
	if err := db.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("role.key.exists")
	}
	return nil
}

func (s *RoleService) ensureMenuIDsExist(menuIDs []uint64) error {
	normalized := normalizeUint64IDs(menuIDs)
	if len(normalized) == 0 {
		return nil
	}

	var count int64
	if err := s.db.Table("system_menu").Where("id IN ? AND type <> ?", normalized, "F").Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(normalized)) {
		return errors.New("role.menu.invalid")
	}
	return nil
}

func (s *RoleService) ensurePermissionKeysExist(permissionKeys []string) error {
	normalized := normalizePermissionKeys(permissionKeys)
	if len(normalized) == 0 {
		return nil
	}

	type permissionRow struct {
		PagePerm string `gorm:"column:page_perm"`
		Perms    string `gorm:"column:perms"`
	}

	var rows []permissionRow
	if err := s.db.Table("system_menu").
		Select("page_perm, perms").
		Where("page_perm IN ? OR perms IN ?", normalized, normalized).
		Scan(&rows).Error; err != nil {
		return err
	}

	exists := make(map[string]struct{}, len(normalized))
	for _, row := range rows {
		if strings.TrimSpace(row.PagePerm) != "" {
			exists[strings.TrimSpace(row.PagePerm)] = struct{}{}
		}
		if strings.TrimSpace(row.Perms) != "" {
			exists[strings.TrimSpace(row.Perms)] = struct{}{}
		}
	}
	for _, key := range normalized {
		if _, ok := exists[key]; !ok {
			return errors.New("role.permission.invalid")
		}
	}
	return nil
}

func (s *RoleService) replaceRoleMenus(tx *gorm.DB, roleID uint64, menuIDs []uint64) error {
	if err := tx.Exec("DELETE FROM system_role_menu WHERE role_id = ?", roleID).Error; err != nil {
		return err
	}
	for _, menuID := range menuIDs {
		if err := tx.Exec("INSERT INTO system_role_menu (role_id, menu_id) VALUES (?, ?)", roleID, menuID).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *RoleService) replaceRolePermissions(tx *gorm.DB, roleID uint64, permissionKeys []string) error {
	if err := tx.Exec("DELETE FROM system_role_permission WHERE role_id = ?", roleID).Error; err != nil {
		return err
	}
	for _, permissionKey := range normalizePermissionKeys(permissionKeys) {
		if err := tx.Create(&SystemRolePermission{
			RoleID:        roleID,
			PermissionKey: permissionKey,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func normalizeRoleStatus(status int) int {
	if status == 2 {
		return 2
	}
	return 1
}

func normalizeRolePageQuery(query *RoleListQuery) (int, int) {
	page := 1
	pageSize := 10
	if query == nil {
		return page, pageSize
	}
	if query.Page > 0 {
		page = query.Page
	}
	if query.PageSize > 0 {
		pageSize = query.PageSize
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func normalizeRoleMemberPageQuery(query *RoleMemberQuery) (int, int) {
	page := 1
	pageSize := 10
	if query == nil {
		return page, pageSize
	}
	if query.Page > 0 {
		page = query.Page
	}
	if query.PageSize > 0 {
		pageSize = query.PageSize
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func normalizeRoleSort(query *RoleListQuery) (string, bool) {
	if query == nil {
		return "id", true
	}

	sortWhitelist := map[string]string{
		"id":         "id",
		"roleName":   "role_name",
		"role_name":  "role_name",
		"roleKey":    "role_key",
		"role_key":   "role_key",
		"sort":       "sort",
		"status":     "status",
		"createdAt":  "created_at",
		"created_at": "created_at",
	}

	column, ok := sortWhitelist[strings.TrimSpace(query.SortField)]
	if !ok {
		column = "id"
	}

	return column, strings.ToLower(strings.TrimSpace(query.SortOrder)) == "desc"
}

func normalizeUint64IDs(ids []uint64) []uint64 {
	if len(ids) == 0 {
		return []uint64{}
	}
	result := make([]uint64, 0, len(ids))
	seen := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func normalizePermissionKeys(keys []string) []string {
	result := make([]string, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func (s *RoleService) releaseDeletedRoleKeys() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var deletedRoles []SystemRole
		if err := tx.Unscoped().Where("deleted_at IS NOT NULL").Find(&deletedRoles).Error; err != nil {
			return err
		}
		for _, role := range deletedRoles {
			if strings.HasPrefix(role.RoleKey, deletedRoleKeyPrefix) {
				continue
			}
			deletedRoleKey, err := s.allocateDeletedRoleKey(tx, role.ID)
			if err != nil {
				return err
			}
			if err := tx.Unscoped().Model(&SystemRole{}).Where("id = ?", role.ID).Update("role_key", deletedRoleKey).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *RoleService) allocateDeletedRoleKey(tx *gorm.DB, roleID uint64) (string, error) {
	for attempt := 0; attempt < 5; attempt++ {
		candidate := fmt.Sprintf("%s%d", deletedRoleKeyPrefix, roleID)
		if attempt > 0 {
			candidate = fmt.Sprintf("%s%d_%d", deletedRoleKeyPrefix, roleID, time.Now().UnixNano())
		}

		var count int64
		if err := tx.Unscoped().Model(&SystemRole{}).Where("role_key = ? AND id <> ?", candidate, roleID).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
	}
	return "", errors.New("role.delete.error.archive_key_conflict")
}

func reloadRolePolicies() error {
	if database.Enforcer == nil {
		return nil
	}
	return database.Enforcer.LoadPolicy()
}

func (s *RoleService) backfillRolePermissions() error {
	type rolePermissionSeed struct {
		RoleID        uint64 `gorm:"column:role_id"`
		PermissionKey string `gorm:"column:permission_key"`
	}

	var seeds []rolePermissionSeed
	if err := s.db.Table("system_role_menu").
		Select("system_role_menu.role_id AS role_id, COALESCE(NULLIF(system_menu.page_perm, ''), NULLIF(system_menu.perms, '')) AS permission_key").
		Joins("JOIN system_menu ON system_menu.id = system_role_menu.menu_id").
		Where("(system_menu.page_perm <> '' OR system_menu.perms <> '')").
		Scan(&seeds).Error; err != nil {
		return err
	}

	if len(seeds) == 0 {
		return nil
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, seed := range seeds {
			if strings.TrimSpace(seed.PermissionKey) == "" {
				continue
			}
			if err := tx.Where("role_id = ? AND permission_key = ?", seed.RoleID, seed.PermissionKey).
				FirstOrCreate(&SystemRolePermission{
					RoleID:        seed.RoleID,
					PermissionKey: strings.TrimSpace(seed.PermissionKey),
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
