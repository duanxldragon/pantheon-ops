package iam

import (
	"errors"
	"fmt"
	"pantheon-ops/backend/pkg/common"
	"strings"
	"time"

	"pantheon-ops/backend/pkg/database"

	"gorm.io/gorm"
)

func (s *RoleService) ensureAdminRoleSeed() error {
	var adminRole SystemRole
	err := s.db.Where("role_key = ?", "admin").First(&adminRole).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		adminRole = SystemRole{
			RoleName: "role.admin.name",
			RoleKey:  "admin",
			Sort:     1,
			Status:   common.StatusEnabled,
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
			updates["role_name"] = "role.admin.name"
		}
		if adminRole.Sort == 0 {
			updates["sort"] = 1
		}
		if adminRole.Status != common.StatusEnabled {
			updates["status"] = common.StatusEnabled
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

func (s *RoleService) validateRoleCreate(req *RoleCreateReq) error {
	if strings.TrimSpace(req.RoleName) == "" || strings.TrimSpace(req.RoleKey) == "" {
		return common.NewBadRequest("param.invalid")
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
		return common.NewBadRequest("param.invalid")
	}
	if role.RoleKey == "admin" && (strings.TrimSpace(req.RoleKey) != "admin" || req.Status == common.StatusDisabled) {
		return common.NewConflict("role.update.error.protected")
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
		return common.NewConflict("role.key.exists")
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
			return common.NewBadRequest("role.permission.invalid")
		}
	}
	return nil
}

func (s *RoleService) ensureUsersExist(userIDs []uint64) error {
	var count int64
	if err := s.db.Table("system_user").Where("id IN ?", userIDs).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(userIDs)) {
		return common.NewNotFound("user.batch.not_found")
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
	return "", common.NewConflict("role.delete.error.archive_key_conflict")
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

func reloadRolePolicies() error {
	if database.Enforcer == nil {
		return nil
	}
	return database.Enforcer.LoadPolicy()
}

func normalizeRoleStatus(status int) int {
	return common.NormalizeEnabledStatus(status)
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
