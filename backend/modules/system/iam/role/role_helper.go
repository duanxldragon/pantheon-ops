package role

import (
	"errors"
	"fmt"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"log/slog"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/rbacbind"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	condRoleKeyEquals = "role_key = ?"
	condIDEquals      = "id = ?"
)

func (s *RoleService) ensureAdminRoleSeed() error {
	var adminRole SystemRole
	err := s.db.Where("tenant_id = ?", tenant.PlatformGlobalTenantID).Where(condRoleKeyEquals, "admin").First(&adminRole).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		adminRole = SystemRole{
			RoleName: "role.admin.name",
			RoleKey:  "admin",
			Sort:     1,
			Status:   common.StatusEnabled,
		}
		var count int64
		if err := s.db.Unscoped().Model(&SystemRole{}).Where(condIDEquals, 1).Count(&count).Error; err != nil {
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
	if err := s.db.Table("system_role").Select("id").
		Where("tenant_id = ?", tenant.PlatformGlobalTenantID).
		Where(condRoleKeyEquals, "admin").
		Limit(1).Pluck("id", &adminRoleID).Error; err != nil {
		return err
	}
	if adminRoleID == 0 {
		return nil
	}

	var adminUserCount int64
	if err := s.db.Table("system_user").Where(condIDEquals, 1).Count(&adminUserCount).Error; err != nil {
		return err
	}
	if adminUserCount == 0 {
		return nil
	}

	// The platform bootstrap binds the built-in admin (user 1) to the platform
	// `admin` role, so the binding lives in the platform-global namespace.
	return rbacbind.EnsureUserRole(s.db, tenant.PlatformGlobalTenantID, 1, adminRoleID)
}

func (s *RoleService) validateRoleCreate(req *RoleCreateReq) error {
	if strings.TrimSpace(req.RoleName) == "" || strings.TrimSpace(req.RoleKey) == "" {
		return common.NewBadRequest(common.MessageParamInvalid)
	}
	if err := s.ensureRoleKeyAllowed(req.RoleKey, 0); err != nil {
		return err
	}
	if err := s.ensureRoleKeyUnique(0, req.RoleKey); err != nil {
		return err
	}
	if err := s.ensureMenuIDsExist(req.MenuIDs); err != nil {
		return err
	}
	if err := s.ensureMenuIDsAssignable(s.TenantOwnerID(), req.MenuIDs); err != nil {
		return err
	}
	if err := s.ensurePermissionKeysExist(req.PermissionKeys); err != nil {
		return err
	}
	return ensurePermissionKeysAssignable(s.TenantOwnerID(), req.PermissionKeys)
}

func (s *RoleService) validateRoleUpdate(role *SystemRole, req *RoleUpdateReq) error {
	if strings.TrimSpace(req.RoleName) == "" || strings.TrimSpace(req.RoleKey) == "" {
		return common.NewBadRequest(common.MessageParamInvalid)
	}
	if role.RoleKey == "admin" && role.TenantID == tenant.PlatformGlobalTenantID &&
		(strings.TrimSpace(req.RoleKey) != "admin" || req.Status == common.StatusDisabled) {
		return common.NewConflict("role.update.error.protected")
	}
	if err := s.ensureRoleKeyAllowed(req.RoleKey, role.ID); err != nil {
		return err
	}
	if err := s.ensureRoleKeyUnique(role.ID, req.RoleKey); err != nil {
		return err
	}
	if err := s.ensureMenuIDsExist(req.MenuIDs); err != nil {
		return err
	}
	if err := s.ensureMenuIDsAssignable(role.TenantID, req.MenuIDs); err != nil {
		return err
	}
	if err := s.ensurePermissionKeysExist(req.PermissionKeys); err != nil {
		return err
	}
	return ensurePermissionKeysAssignable(role.TenantID, req.PermissionKeys)
}

// ensureRoleKeyAllowed enforces the bidirectional role-key namespace mutex
// (contract §7: "两个命名空间互斥").
//
// Forward direction — tenant callers must not take a platform key: role keys
// are unique per tenant, so a tenant could otherwise create a role keyed
// `admin` (or any other platform role key) and hand it to a user. Login then
// puts that bare key into the session, and Casbin consults the GLOBAL subject
// of the same name (contract §4: global first, then domain), so the tenant
// role inherits the platform `admin` policies — a full tenant→platform
// escape. `platform_ops` is reserved even without a role row because the
// platform-capability middleware admits it by name.
//
// Reverse direction — platform callers must not take a key a live tenant
// already owns: the same global-first lookup works the other way, so a new
// platform role keyed like a tenant role would hand that tenant's users the
// platform policies the moment they log in. Soft-deleted roles are excluded
// via the model scope; released keys are already archived under
// deletedRoleKeyPrefix, so legitimate re-use is unaffected.
// reverse mutex must not fire on the role being renamed itself: a platform
// edit of a tenant role that keeps the key would otherwise collide with the
// live tenant row it is updating (excludeRoleID).
func (s *RoleService) ensureRoleKeyAllowed(roleKey string, excludeRoleID uint64) error {
	key := strings.TrimSpace(roleKey)
	if key == "" {
		return nil
	}
	ownerID := s.TenantOwnerID()
	if ownerID != tenant.PlatformGlobalTenantID && key == "platform_ops" {
		return common.NewConflict("role.key.reserved")
	}

	column, value := roleKeyBlockingNamespace(ownerID)
	var count int64
	// Model-based count: gorm pins the statement to system_role and appends
	// `deleted_at IS NULL`, so only live roles in the opposite namespace block.
	q := s.db.Model(&SystemRole{}).
		Where(condRoleKeyEquals, key).
		Where("tenant_id "+column+" ?", value)
	if excludeRoleID > 0 {
		q = q.Where("id <> ?", excludeRoleID)
	}
	if err := q.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return common.NewConflict("role.key.reserved")
	}
	return nil
}

// roleKeyBlockingNamespace is the pure half of the mutex: which side of the
// namespace pair an operator of the given tenant is checked against. A
// platform operator (tenant 0) is blocked by any tenant-owned role
// (tenant_id <> 0); a tenant operator is blocked by platform roles
// (tenant_id = 0).
func roleKeyBlockingNamespace(ownerID uint64) (operator string, value uint64) {
	if ownerID == tenant.PlatformGlobalTenantID {
		return "<>", tenant.PlatformGlobalTenantID
	}
	return "=", tenant.PlatformGlobalTenantID
}

func (s *RoleService) ensureRoleKeyUnique(roleID uint64, roleKey string) error {
	var count int64
	// Role keys are unique PER TENANT in multi mode (contract §4): the scope
	// narrows the lookup to the request tenant, so each tenant can own its own
	// `tenant_admin` without colliding with the platform's roles.
	db := s.Scoped(s.db.Model(&SystemRole{})).Where(condRoleKeyEquals, strings.TrimSpace(roleKey))
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

// ensureMenuIDsAssignable rejects platform-only pages on a tenant-owned role.
// The manage-scoped tree already hides them, but the API is the authority: a
// direct request must not recreate the broken menu entry the tree filter removes.
func (s *RoleService) ensureMenuIDsAssignable(ownerTenantID uint64, menuIDs []uint64) error {
	if ownerTenantID == tenant.PlatformGlobalTenantID {
		return nil
	}
	normalized := normalizeUint64IDs(menuIDs)
	if len(normalized) == 0 {
		return nil
	}
	type menuRow struct {
		ID     uint64 `gorm:"column:id"`
		Module string `gorm:"column:module"`
		Path   string `gorm:"column:path"`
	}
	var rows []menuRow
	if err := s.db.Table("system_menu").
		Select("id, module, path").
		Where("id IN ?", normalized).
		Scan(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if tenant.IsPlatformOnlyMenu(row.Module, row.Path) {
			return common.NewBadRequest("role.menu.invalid")
		}
	}
	return nil
}

// ensurePermissionKeysAssignable rejects platform-only permission keys on a
// tenant-owned role, for the same reason as ensureMenuIDsAssignable.
func ensurePermissionKeysAssignable(ownerTenantID uint64, permissionKeys []string) error {
	if ownerTenantID == tenant.PlatformGlobalTenantID {
		return nil
	}
	for _, key := range normalizePermissionKeys(permissionKeys) {
		if tenant.IsPlatformOnlyPermissionKey(key) {
			return common.NewBadRequest("role.permission.invalid")
		}
	}
	return nil
}

func (s *RoleService) ensureUsersExist(userIDs []uint64) error {
	var count int64
	if err := s.Scoped(s.db.Table("system_user")).Where("id IN ?", userIDs).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(userIDs)) {
		return common.NewNotFound("user.batch.not_found")
	}
	return nil
}

// roleOwningTenantIDTx resolves the tenant that owns a role row, on an existing
// transaction/query handle. `system_role_permission` is keyed by
// (tenant_id, role_id, permission_key), so the grant is stamped from the role
// itself and can never be written into another tenant's namespace. A missing
// role resolves to 0 (the pre-tenant behaviour).
func roleOwningTenantIDTx(tx *gorm.DB, roleID uint64) (uint64, error) {
	var tenantID uint64
	if err := tx.Table("system_role").
		Select("tenant_id").
		Where(condIDEquals, roleID).
		Scan(&tenantID).Error; err != nil {
		return 0, err
	}
	return tenantID, nil
}

// lockRoleOwningTenantIDTx is roleOwningTenantIDTx with a row lock on the role.
//
// replaceRolePermissions deletes the role's grants and re-inserts the requested
// set. Two concurrent replacers that both read "no grants", both delete, and
// both insert produce either a merged set (the second delete missed the first
// insert) or a deadlock on the index gap. Holding the role row makes them queue:
// the loser waits, then replaces the winner's rows, so the surviving set is
// exactly the last caller's.
func lockRoleOwningTenantIDTx(tx *gorm.DB, roleID uint64) (uint64, error) {
	var role struct {
		TenantID uint64 `gorm:"column:tenant_id"`
	}
	if err := tx.Table("system_role").
		Select("tenant_id").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(condIDEquals, roleID).
		Scan(&role).Error; err != nil {
		return 0, err
	}
	return role.TenantID, nil
}

func (s *RoleService) replaceRolePermissions(tx *gorm.DB, roleID uint64, permissionKeys []string) error {
	tenantID, err := lockRoleOwningTenantIDTx(tx, roleID)
	if err != nil {
		return err
	}
	if err := tx.Exec("DELETE FROM system_role_permission WHERE role_id = ?", roleID).Error; err != nil {
		return err
	}
	normalized := normalizePermissionKeys(permissionKeys)
	if len(normalized) == 0 {
		return nil
	}
	permissions := make([]SystemRolePermission, 0, len(normalized))
	for _, permissionKey := range normalized {
		permissions = append(permissions, SystemRolePermission{
			TenantID:      tenantID,
			RoleID:        roleID,
			PermissionKey: permissionKey,
		})
	}
	return tx.CreateInBatches(permissions, 100).Error
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
			if err := tx.Unscoped().Model(&SystemRole{}).Where(condIDEquals, role.ID).Update("role_key", deletedRoleKey).Error; err != nil {
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

// repairStaleTenantPermissionRows rewrites grants that a pre-fix backfill wrote
// into the platform namespace (tenant_id = 0) for a tenant-owned role.
//
// Those rows are invisible to every tenant-scoped read, so the tenant role
// silently loses the permission while a stray grant sits under tenant 0. The
// role may also already own the same key under its real tenant — the unique key
// (tenant_id, role_id, permission_key) allows both — so the duplicate is
// deleted first and only the remaining rows are re-stamped. Soft-deleted roles
// are skipped, matching the seed query: their grants are not in use and
// rewriting them would resurrect data the operator deleted.
func (s *RoleService) repairStaleTenantPermissionRows() error {
	if err := s.db.Exec(`
		DELETE stale
		FROM system_role_permission AS stale
		JOIN system_role AS r
		  ON r.id = stale.role_id
		 AND r.deleted_at IS NULL
		 AND r.tenant_id <> 0
		JOIN system_role_permission AS owned
		  ON owned.role_id = stale.role_id
		 AND owned.permission_key = stale.permission_key
		 AND owned.tenant_id = r.tenant_id
		WHERE stale.tenant_id = 0`).Error; err != nil {
		return fmt.Errorf("repair stale tenant permission rows: delete duplicates: %w", err)
	}
	if err := s.db.Exec(`
		UPDATE system_role_permission AS stale
		JOIN system_role AS r
		  ON r.id = stale.role_id
		 AND r.deleted_at IS NULL
		 AND r.tenant_id <> 0
		SET stale.tenant_id = r.tenant_id
		WHERE stale.tenant_id = 0`).Error; err != nil {
		return fmt.Errorf("repair stale tenant permission rows: stamp tenant: %w", err)
	}
	return nil
}

func (s *RoleService) backfillRolePermissions() error {
	// Repair runs before the seed query and before its empty-seed early return:
	// a database whose stale rows are the only thing to fix must still be fixed.
	if err := s.repairStaleTenantPermissionRows(); err != nil {
		return err
	}

	type rolePermissionSeed struct {
		TenantID      uint64 `gorm:"column:tenant_id"`
		RoleID        uint64 `gorm:"column:role_id"`
		PermissionKey string `gorm:"column:permission_key"`
	}

	var seeds []rolePermissionSeed
	if err := s.db.Table("system_role_menu").
		Select("system_role.tenant_id AS tenant_id, system_role_menu.role_id AS role_id, COALESCE(NULLIF(system_menu.page_perm, ''), NULLIF(system_menu.perms, '')) AS permission_key").
		Joins("JOIN system_menu ON system_menu.id = system_role_menu.menu_id").
		Joins("JOIN system_role ON system_role.id = system_role_menu.role_id AND system_role.deleted_at IS NULL").
		Where("(system_menu.page_perm <> '' OR system_menu.perms <> '')").
		Scan(&seeds).Error; err != nil {
		return err
	}

	if len(seeds) == 0 {
		return nil
	}

	// Several menus can expose the same permission key, so one role can produce
	// the same natural key more than once. The unique index would absorb the
	// duplicates, but deduplicating keeps the batch honest on its own.
	seen := make(map[SystemRolePermission]struct{}, len(seeds))
	permissions := make([]SystemRolePermission, 0, len(seeds))
	for _, seed := range seeds {
		permissionKey := strings.TrimSpace(seed.PermissionKey)
		if permissionKey == "" {
			continue
		}
		grant := SystemRolePermission{
			TenantID:      seed.TenantID,
			RoleID:        seed.RoleID,
			PermissionKey: permissionKey,
		}
		if _, ok := seen[grant]; ok {
			continue
		}
		seen[grant] = struct{}{}
		permissions = append(permissions, grant)
	}
	if len(permissions) == 0 {
		return nil
	}

	return s.db.Table("system_role_permission").
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(permissions, 100).Error
}

func reloadRolePolicies() error {
	if database.Enforcer == nil {
		return nil
	}
	if err := database.Enforcer.LoadPolicy(); err != nil {
		slog.Error("casbin policy reload failed after DB write", "component", "system/iam/role", "error", err)
		return err
	}
	return nil
}

func normalizeRoleStatus(status int) int {
	return common.NormalizeEnabledStatus(status)
}

// normalizeRolePageQuery clamps the role list query through the shared
// pagination helper with the role module's own cap. The nil-extract prelude
// keeps the service's nil-tolerant contract; the shared helper decides the
// defaults. normalizeRoleMemberPageQuery is the same adapter for the member
// query type — generics cannot read struct fields, so the two types cannot
// share one function body.
func normalizeRolePageQuery(query *RoleListQuery) (int, int, int) {
	var page, pageSize int
	if query != nil {
		page, pageSize = query.Page, query.PageSize
	}
	return common.NormalizePagination(page, pageSize, 100)
}

func normalizeRoleMemberPageQuery(query *RoleMemberQuery) (int, int, int) {
	var page, pageSize int
	if query != nil {
		page, pageSize = query.Page, query.PageSize
	}
	return common.NormalizePagination(page, pageSize, 100)
}

var roleSortColumns = map[string]string{
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

func normalizeRoleSort(query *RoleListQuery) (string, bool) {
	if query == nil {
		return "id", true
	}

	return common.NormalizeSort(query.SortField, roleSortColumns, "id"),
		strings.ToLower(strings.TrimSpace(query.SortOrder)) == "desc"
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
