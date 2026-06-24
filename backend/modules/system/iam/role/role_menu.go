package iam

import (
	"errors"

	"gorm.io/gorm"
)

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
