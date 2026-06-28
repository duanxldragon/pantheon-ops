package iam

import (
	"fmt"
	"pantheon-ops/backend/pkg/common"
	"strings"
	"time"

	"pantheon-ops/backend/pkg/database"

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
		return common.NewBadRequest("database.not_initialized")
	}
	if err := s.db.AutoMigrate(&SystemRole{}, &SystemRolePermission{}, &SystemRoleMenu{}); err != nil {
		return err
	}
	return s.Bootstrap()
}

func (s *RoleService) Bootstrap() error {
	if s.db == nil {
		return common.NewBadRequest("database.not_initialized")
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

// ListRoles 获取角色分页列表。
func (s *RoleService) ListRoles(query *RoleListQuery) (*RoleListPageResp, error) {
	if s.db == nil {
		return nil, common.NewBadRequest("database.not_initialized")
	}

	var roles []SystemRole
	db := s.db.Model(&SystemRole{})
	page, pageSize := normalizeRolePageQuery(query)
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
		return nil, common.NewBadRequest("database.not_initialized")
	}
	if _, err := s.getRole(roleID); err != nil {
		return nil, err
	}
	return s.listUsersByRoleMembership(roleID, query, true)
}

func (s *RoleService) ListAssignableUsers(roleID uint64, query *RoleMemberQuery) (*RoleMemberPageResp, error) {
	if s.db == nil {
		return nil, common.NewBadRequest("database.not_initialized")
	}
	if _, err := s.getRole(roleID); err != nil {
		return nil, err
	}
	return s.listUsersByRoleMembership(roleID, query, false)
}

func (s *RoleService) AddRoleMembers(roleID uint64, userIDs []uint64) (int, error) {
	if s.db == nil {
		return 0, common.NewBadRequest("database.not_initialized")
	}
	if _, err := s.getRole(roleID); err != nil {
		return 0, err
	}

	normalizedUserIDs := normalizeUint64IDs(userIDs)
	if len(normalizedUserIDs) == 0 {
		return 0, common.NewBadRequest("user.batch.empty")
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
		return 0, common.NewBadRequest("database.not_initialized")
	}
	role, err := s.getRole(roleID)
	if err != nil {
		return 0, err
	}

	normalizedUserIDs := normalizeUint64IDs(userIDs)
	if len(normalizedUserIDs) == 0 {
		return 0, common.NewBadRequest("user.batch.empty")
	}
	if err := s.ensureUsersExist(normalizedUserIDs); err != nil {
		return 0, err
	}
	if role.RoleKey == "admin" {
		for _, userID := range normalizedUserIDs {
			if userID == 1 {
				return 0, common.NewConflict("user.update.error.protected")
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
		return nil, common.NewBadRequest("database.not_initialized")
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
		return nil, common.NewBadRequest("database.not_initialized")
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
		return common.NewBadRequest("database.not_initialized")
	}

	var role SystemRole
	if err := s.db.First(&role, roleID).Error; err != nil {
		return err
	}
	if role.ID == 1 || role.RoleKey == "admin" {
		return common.NewConflict("role.delete.error.protected")
	}

	var userCount int64
	if err := s.db.Table("system_user_role").Where("role_id = ?", roleID).Count(&userCount).Error; err != nil {
		return err
	}
	if userCount > 0 {
		return common.NewInternal("role.delete.error.has_users")
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

func (s *RoleService) BatchUpdateRoleStatus(roleIDs []uint64, status int) (int, error) {
	if s.db == nil {
		return 0, common.NewBadRequest("database.not_initialized")
	}
	normalizedIDs := normalizeUint64IDs(roleIDs)
	if len(normalizedIDs) == 0 {
		return 0, common.NewBadRequest("role.batch.empty")
	}
	if !common.IsEnabledStatus(status) {
		return 0, common.NewBadRequest("param.invalid")
	}

	var roles []SystemRole
	if err := s.db.Where("id IN ?", normalizedIDs).Find(&roles).Error; err != nil {
		return 0, err
	}
	if len(roles) != len(normalizedIDs) {
		return 0, common.NewNotFound("role.batch.not_found")
	}
	if status == common.StatusDisabled {
		for _, role := range roles {
			if role.ID == 1 || role.RoleKey == "admin" {
				return 0, common.NewConflict("role.update.error.protected")
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

func (s *RoleService) getRole(roleID uint64) (*SystemRole, error) {
	var role SystemRole
	if err := s.db.First(&role, roleID).Error; err != nil {
		return nil, err
	}
	return &role, nil
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
				fmt.Sprintf("%%%s%%", common.EscapeLikePattern(keyword)),
				fmt.Sprintf("%%%s%%", common.EscapeLikePattern(keyword)),
			)
		}
		if query.Status != nil && common.IsEnabledStatus(*query.Status) {
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
