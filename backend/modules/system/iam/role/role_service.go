package iam

import (
	"fmt"
	"pantheon-base/pkg/common"
	"strings"
	"time"

	"pantheon-base/pkg/database"
	"pantheon-base/pkg/rbacbind"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RoleService struct {
	db *gorm.DB
}

type roleListAssociations struct {
	menus       map[uint64][]uint64
	permissions map[uint64][]string
	dataScopes  map[uint64]string
}

const deletedRoleKeyPrefix = "__deleted_role_"

const (
	errDatabaseNotInitialized = "database.not_initialized"
	condIDIn                  = "id IN ?"
)

// roleDataScopePolicy mirrors PermissionRoleDataScopePolicy to avoid cross-directory import.
// Table name: system_role_data_scope
type roleDataScopePolicy struct {
	ID      uint64 `gorm:"primaryKey;autoIncrement"`
	RoleKey string `gorm:"size:64;not null;uniqueIndex"`
	Mode    string `gorm:"size:32;not null;default:'all'"`
	DeptIDs string `gorm:"type:text"`
}

func (roleDataScopePolicy) TableName() string {
	return "system_role_data_scope"
}

func NewRoleService(db *gorm.DB) *RoleService {
	return &RoleService{db: db}
}

func (s *RoleService) Migrate() error {
	if s.db == nil {
		return common.NewBadRequest(errDatabaseNotInitialized)
	}
	if err := s.db.AutoMigrate(&SystemRole{}, &SystemRolePermission{}, &SystemRoleMenu{}, &roleDataScopePolicy{}); err != nil {
		return err
	}
	return s.Bootstrap()
}

func (s *RoleService) Bootstrap() error {
	if s.db == nil {
		return common.NewBadRequest(errDatabaseNotInitialized)
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
		return nil, common.NewBadRequest(errDatabaseNotInitialized)
	}

	var roles []SystemRole
	page, pageSize := normalizeRolePageQuery(query)
	db := buildRoleListQuery(s.db, query)

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

	associations, err := s.loadRoleListAssociations(roles)
	if err != nil {
		return nil, err
	}

	return &RoleListPageResp{
		Items:    buildRoleListItems(roles, associations),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func buildRoleListQuery(db *gorm.DB, query *RoleListQuery) *gorm.DB {
	db = db.Model(&SystemRole{})
	if query == nil {
		return db
	}
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		pattern := fmt.Sprintf("%%%s%%", common.EscapeLikePattern(keyword))
		db = db.Where("role_name LIKE ? OR role_key LIKE ?", pattern, pattern)
	}
	if roleName := strings.TrimSpace(query.RoleName); roleName != "" {
		db = db.Where("role_name LIKE ?", fmt.Sprintf("%%%s%%", common.EscapeLikePattern(roleName)))
	}
	if roleKey := strings.TrimSpace(query.RoleKey); roleKey != "" {
		db = db.Where("role_key LIKE ?", fmt.Sprintf("%%%s%%", common.EscapeLikePattern(roleKey)))
	}
	if query.Status != nil && common.IsEnabledStatus(*query.Status) {
		db = db.Where("status = ?", *query.Status)
	}
	return db
}

func (s *RoleService) loadRoleListAssociations(roles []SystemRole) (roleListAssociations, error) {
	roleIDs := make([]uint64, 0, len(roles))
	for _, item := range roles {
		roleIDs = append(roleIDs, item.ID)
	}
	menus, err := s.loadRoleMenus(roleIDs)
	if err != nil {
		return roleListAssociations{}, err
	}
	permissions, err := s.loadRolePermissions(roleIDs)
	if err != nil {
		return roleListAssociations{}, err
	}
	return roleListAssociations{
		menus:       menus,
		permissions: permissions,
		dataScopes:  s.loadRoleDataScopes(roleIDs),
	}, nil
}

func buildRoleListItems(roles []SystemRole, associations roleListAssociations) []RoleListResp {
	items := make([]RoleListResp, 0, len(roles))
	for _, item := range roles {
		items = append(items, RoleListResp{
			ID:             item.ID,
			RoleName:       item.RoleName,
			RoleKey:        item.RoleKey,
			Sort:           item.Sort,
			Status:         item.Status,
			CreatedAt:      item.CreatedAt.Format(time.RFC3339),
			MenuIDs:        associations.menus[item.ID],
			PermissionKeys: associations.permissions[item.ID],
			DataScope:      associations.dataScopes[item.ID],
		})
	}
	return items
}

func (s *RoleService) ListRoleMembers(roleID uint64, query *RoleMemberQuery) (*RoleMemberPageResp, error) {
	if s.db == nil {
		return nil, common.NewBadRequest(errDatabaseNotInitialized)
	}
	if _, err := s.getRole(roleID); err != nil {
		return nil, err
	}
	return s.listUsersByRoleMembership(roleID, query, true)
}

func (s *RoleService) ListAssignableUsers(roleID uint64, query *RoleMemberQuery) (*RoleMemberPageResp, error) {
	if s.db == nil {
		return nil, common.NewBadRequest(errDatabaseNotInitialized)
	}
	if _, err := s.getRole(roleID); err != nil {
		return nil, err
	}
	return s.listUsersByRoleMembership(roleID, query, false)
}

func (s *RoleService) AddRoleMembers(roleID uint64, userIDs []uint64) (int, error) {
	if s.db == nil {
		return 0, common.NewBadRequest(errDatabaseNotInitialized)
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

	userIDsToAdd := make([]uint64, 0, len(normalizedUserIDs))
	for _, userID := range normalizedUserIDs {
		if _, ok := existing[userID]; ok {
			continue
		}
		userIDsToAdd = append(userIDsToAdd, userID)
	}
	if len(userIDsToAdd) == 0 {
		return 0, nil
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		return rbacbind.InsertUserRoles(tx, userIDsToAdd, roleID)
	}); err != nil {
		return 0, err
	}

	return len(userIDsToAdd), nil
}

func (s *RoleService) RemoveRoleMembers(roleID uint64, userIDs []uint64) (int, error) {
	if s.db == nil {
		return 0, common.NewBadRequest(errDatabaseNotInitialized)
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
	if role.RoleKey == common.AdminRoleKey {
		for _, userID := range normalizedUserIDs {
			if userID == common.BuiltinAdminUserID {
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
		return nil, common.NewBadRequest(errDatabaseNotInitialized)
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
	dataScope := normalizeRoleDataScope(req.DataScope)

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&role).Error; err != nil {
			return err
		}
		if err := s.replaceRoleMenus(tx, role.ID, menuIDs); err != nil {
			return err
		}
		if err := s.replaceRolePermissions(tx, role.ID, permissionKeys); err != nil {
			return err
		}
		if err := s.upsertRoleDataScopePolicy(tx, role.RoleKey, dataScope); err != nil {
			return err
		}
		return nil
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
		DataScope:      dataScope,
	}, nil
}

// UpdateRole 更新角色。
func (s *RoleService) UpdateRole(roleID uint64, req *RoleUpdateReq) (*RoleListResp, error) {
	if s.db == nil {
		return nil, common.NewBadRequest(errDatabaseNotInitialized)
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
	dataScope := normalizeRoleDataScope(req.DataScope)

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		return s.updateRoleInTransaction(tx, &role, menuIDs, permissionKeys, dataScope)
	}); err != nil {
		return nil, err
	}

	return buildRoleListResp(role, menuIDs, permissionKeys, dataScope), nil
}

func (s *RoleService) updateRoleInTransaction(tx *gorm.DB, role *SystemRole, menuIDs []uint64, permissionKeys []string, dataScope string) error {
	if err := tx.Save(role).Error; err != nil {
		return err
	}
	if err := s.replaceRoleMenus(tx, role.ID, menuIDs); err != nil {
		return err
	}
	if err := s.replaceRolePermissions(tx, role.ID, permissionKeys); err != nil {
		return err
	}
	return s.upsertRoleDataScopePolicy(tx, role.RoleKey, dataScope)
}

func buildRoleListResp(role SystemRole, menuIDs []uint64, permissionKeys []string, dataScope string) *RoleListResp {
	return &RoleListResp{
		ID:             role.ID,
		RoleName:       role.RoleName,
		RoleKey:        role.RoleKey,
		Sort:           role.Sort,
		Status:         role.Status,
		CreatedAt:      role.CreatedAt.Format(time.RFC3339),
		MenuIDs:        menuIDs,
		PermissionKeys: permissionKeys,
		DataScope:      dataScope,
	}
}

// DeleteRole 删除角色。
func (s *RoleService) DeleteRole(roleID uint64) error {
	if s.db == nil {
		return common.NewBadRequest(errDatabaseNotInitialized)
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		return s.deleteRoleInTransaction(tx, roleID)
	}); err != nil {
		return err
	}

	return reloadRolePolicies()
}

func (s *RoleService) deleteRoleInTransaction(tx *gorm.DB, roleID uint64) error {
	role, err := loadDeletableRole(tx, roleID)
	if err != nil {
		return err
	}
	if err := deleteRoleBindings(tx, role.ID, role.RoleKey); err != nil {
		return err
	}
	if err := deleteRoleDataScopes(tx, role.RoleKey); err != nil {
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
}

func loadDeletableRole(tx *gorm.DB, roleID uint64) (SystemRole, error) {
	// 前置检查放在事务内并对角色行加锁，避免成员绑定检查与删除之间的并发窗口。
	var role SystemRole
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&role, roleID).Error; err != nil {
		return role, err
	}
	if role.ID == common.BuiltinAdminRoleID || role.RoleKey == common.AdminRoleKey {
		return role, common.NewConflict("role.delete.error.protected")
	}
	var userCount int64
	if err := tx.Table("system_user_role").Where("role_id = ?", roleID).Count(&userCount).Error; err != nil {
		return role, err
	}
	if userCount > 0 {
		return role, common.NewInternal("role.delete.error.has_users")
	}
	return role, nil
}

func deleteRoleBindings(tx *gorm.DB, roleID uint64, roleKey string) error {
	if err := tx.Exec("DELETE FROM system_role_menu WHERE role_id = ?", roleID).Error; err != nil {
		return err
	}
	if err := tx.Exec("DELETE FROM system_role_permission WHERE role_id = ?", roleID).Error; err != nil {
		return err
	}
	return tx.Model(&database.CasbinRule{}).
		Where("ptype = ? AND v0 = ?", "p", roleKey).
		Delete(&database.CasbinRule{}).Error
}

func deleteRoleDataScopes(tx *gorm.DB, roleKey string) error {
	// Data-scope tables use role_key; retaining rows would let a recreated role inherit stale policy.
	if err := tx.Where(condRoleKeyEquals, roleKey).Delete(&roleDataScopePolicy{}).Error; err != nil {
		return err
	}
	if !tx.Migrator().HasTable("permission_role_data_scope_policy") {
		return nil
	}
	return tx.Table("permission_role_data_scope_policy").
		Where(condRoleKeyEquals, roleKey).Delete(nil).Error
}

func (s *RoleService) BatchUpdateRoleStatus(roleIDs []uint64, status int) (int, error) {
	if s.db == nil {
		return 0, common.NewBadRequest(errDatabaseNotInitialized)
	}
	normalizedIDs := normalizeUint64IDs(roleIDs)
	if len(normalizedIDs) == 0 {
		return 0, common.NewBadRequest("role.batch.empty")
	}
	if !common.IsEnabledStatus(status) {
		return 0, common.NewBadRequest(errParamInvalid)
	}

	var roles []SystemRole
	if err := s.db.Where(condIDIn, normalizedIDs).Find(&roles).Error; err != nil {
		return 0, err
	}
	if len(roles) != len(normalizedIDs) {
		return 0, common.NewNotFound("role.batch.not_found")
	}
	if status == common.StatusDisabled {
		for _, role := range roles {
			if role.ID == common.BuiltinAdminRoleID || role.RoleKey == common.AdminRoleKey {
				return 0, common.NewConflict("role.update.error.protected")
			}
		}
	}

	if err := s.db.Model(&SystemRole{}).
		Where(condIDIn, normalizedIDs).
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

// upsertRoleDataScopePolicy creates or updates a data scope policy for a role.
func (s *RoleService) upsertRoleDataScopePolicy(tx *gorm.DB, roleKey string, mode string) error {
	mode = normalizeRoleDataScope(mode)
	if !isValidRoleDataScopeMode(mode) {
		return common.NewBadRequest("permission.data_scope.mode_invalid")
	}

	policy := roleDataScopePolicy{
		RoleKey: roleKey,
		Mode:    mode,
		DeptIDs: "",
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "role_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"mode", "dept_ids"}),
	}).Create(&policy).Error
}

// loadRoleDataScopes loads data scope policies for a list of role IDs.
func (s *RoleService) loadRoleDataScopes(roleIDs []uint64) map[uint64]string {
	result := make(map[uint64]string, len(roleIDs))
	for _, id := range roleIDs {
		result[id] = common.DataScopeModeAll
	}
	if len(roleIDs) == 0 {
		return result
	}
	if !s.db.Migrator().HasTable(&roleDataScopePolicy{}) {
		return result
	}

	var roleRows []struct {
		ID      uint64 `gorm:"column:id"`
		RoleKey string `gorm:"column:role_key"`
	}
	if err := s.db.Table("system_role").
		Select("id, role_key").
		Where(condIDIn, roleIDs).
		Scan(&roleRows).Error; err != nil {
		return result
	}

	roleKeys := make([]string, 0, len(roleRows))
	roleKeyToID := make(map[string]uint64, len(roleRows))
	for _, row := range roleRows {
		roleKeys = append(roleKeys, row.RoleKey)
		roleKeyToID[row.RoleKey] = row.ID
	}

	if len(roleKeys) == 0 {
		return result
	}

	var policies []roleDataScopePolicy
	if err := s.db.Where("role_key IN ?", roleKeys).Find(&policies).Error; err != nil {
		return result
	}
	for _, policy := range policies {
		if roleID, ok := roleKeyToID[policy.RoleKey]; ok {
			result[roleID] = policy.Mode
		}
	}
	return result
}

// normalizeRoleDataScope normalizes and validates a data scope mode string.
func normalizeRoleDataScope(dataScope string) string {
	mode := strings.TrimSpace(strings.ToLower(dataScope))
	switch mode {
	case common.DataScopeModeSelf, common.DataScopeModeDept, common.DataScopeModeDeptAndChildren, common.DataScopeModeCustom:
		return mode
	default:
		return common.DataScopeModeAll
	}
}

func isValidRoleDataScopeMode(mode string) bool {
	switch mode {
	case common.DataScopeModeAll, common.DataScopeModeSelf, common.DataScopeModeDept, common.DataScopeModeDeptAndChildren, common.DataScopeModeCustom:
		return true
	default:
		return false
	}
}
