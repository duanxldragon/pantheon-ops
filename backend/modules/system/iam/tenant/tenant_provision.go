package tenant

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common/security"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/rbacbind"
	tenantctx "github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Tenant-local role keys. They live in the tenant's own namespace
// (system_role.tenant_id = <tenant>), so every tenant may own its own copy
// without colliding with the platform's roles (contract §4/§7).
//
// The canonical values are declared in pkg/tenant so the middleware can
// recognise a tenant administrator without importing this module.
const (
	// TenantRoleAdminKey is the tenant administrator: the full console, with
	// every query automatically confined to the tenant (see ScopeTenantReads).
	TenantRoleAdminKey = tenantctx.TenantRoleAdminKey
	// TenantRoleMemberKey is a read-only tenant member.
	TenantRoleMemberKey = tenantctx.TenantRoleMemberKey
)

const (
	// sqlSelectMenuIdentity is the menu projection both the provisioning and the
	// repair paths classify on (module + path). One literal keeps the two
	// queries from drifting apart when a column is added.
	sqlSelectMenuIdentity = "id, module, path"
	// condTenantScopedRow restricts a repair statement to tenant-owned rows. The
	// platform namespace (tenant_id = 0) is never touched by the repair.
	condTenantScopedRow = "tenant_id <> ?"
	// condTenantIDMatches is the tenant_id equality predicate shared by the
	// provisioning queries and their retry paths.
	condTenantIDMatches = "tenant_id = ?"
)

// tenantRoleDefinition describes one provisioned tenant role.
type tenantRoleDefinition struct {
	Key     string
	Name    string
	Sort    int
	Methods []string // HTTP methods granted on /api/v1/*
}

// tenantRoleDefinitions is the role set every tenant is provisioned with.
//
// tenant_admin deliberately receives the same /api/v1/* breadth as the platform
// `admin` role: the chosen product model is "the full console, automatically
// confined to the caller's tenant". Confinement is enforced in two places —
// tenant-scoped reads (ScopeTenantReads) and a platform-capability gate on the
// tenant-console write routes — not by narrowing the Casbin surface, because a
// path allowlist here would silently rot as routes are added.
var tenantRoleDefinitions = []tenantRoleDefinition{
	{
		Key:     TenantRoleAdminKey,
		Name:    "租户管理员",
		Sort:    1,
		Methods: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
	},
	{
		Key:     TenantRoleMemberKey,
		Name:    "租户成员",
		Sort:    2,
		Methods: []string{"GET"},
	},
}

// ErrTenantAdminUsernameTaken is returned when the requested tenant admin
// username already exists in the target tenant.
var ErrTenantAdminUsernameTaken = errors.New("tenant admin username already exists in this tenant")

// ProvisionInput carries everything needed to make a freshly created tenant
// usable. AdminUsername may be empty: the tenant roles and settings are still
// provisioned, but no login is created (the caller can add members later).
type ProvisionInput struct {
	TenantID      uint64
	AdminUsername string
	AdminPassword string
	AdminNickname string
}

// ProvisionResult reports what provisioning actually created.
type ProvisionResult struct {
	AdminUserID     uint64
	RoleIDs         map[string]uint64
	RootDeptID      uint64
	PolicyCount     int
	SettingsCopied  int64
	DictTypesCopied int64
	DictItemsCopied int64
}

// ProvisionTenant turns a bare `tenants` row into a usable tenant.
//
// Before this existed, CreateTenant inserted one row and nothing else —
// InitializeTenantDefaults was a documented no-op — so a new tenant had no
// administrator, no roles and no settings, and nobody could log into it. This
// runs the whole set in one transaction:
//
//  1. tenant-scoped roles (tenant_admin, tenant_member)
//  2. the tenant's administrator user, owned by the tenant (system_user.tenant_id)
//  3. the admin's role binding, in the tenant's namespace
//  4. the owner membership that the login issuance gate requires
//  5. Casbin policies for role:<key>@tenant:<id>
//  6. a copy of the platform settings so the tenant renders with sane defaults
//  7. a copy of the platform default dictionaries (same idempotent pattern)
//
// Everything is idempotent: re-running for an existing tenant re-creates only
// what is missing, so it doubles as a repair entry point.
func (s *Service) ProvisionTenant(in ProvisionInput) (*ProvisionResult, error) {
	if s.db == nil {
		return nil, errors.New("tenant provision: database not initialized")
	}
	if in.TenantID == 0 {
		return nil, errors.New("tenant provision: tenant id is required")
	}
	if _, err := s.GetTenantByID(in.TenantID); err != nil {
		return nil, err
	}

	result := &ProvisionResult{RoleIDs: make(map[string]uint64)}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Provisioning is a chain of read-then-insert steps (roles, root dept,
		// administrator, bindings). Locking the tenant row makes two concurrent
		// callers for the same tenant queue instead of racing: the second one
		// reads what the first committed instead of inserting duplicates.
		if err := lockTenantRow(tx, in.TenantID); err != nil {
			return err
		}

		roleIDs, err := provisionTenantRoles(tx, in.TenantID)
		if err != nil {
			return err
		}
		result.RoleIDs = roleIDs

		if err := provisionTenantRoleMenus(tx, in.TenantID, roleIDs); err != nil {
			return err
		}

		if err := provisionTenantRoleBindings(tx, in.TenantID, roleIDs); err != nil {
			return err
		}
		// Re-provisioning is the repair entry point: clear grants that a pre-fix
		// copy left behind before returning.
		if err := repairTenantPlatformGrants(tx, in.TenantID); err != nil {
			return err
		}

		rootDeptID, err := provisionTenantRootDept(tx, in.TenantID)
		if err != nil {
			return err
		}
		result.RootDeptID = rootDeptID

		if strings.TrimSpace(in.AdminUsername) == "" {
			return nil
		}

		userID, err := provisionTenantAdminUser(tx, in)
		if err != nil {
			return err
		}
		result.AdminUserID = userID

		if err := provisionTenantAdminBinding(tx, in.TenantID, userID, roleIDs); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Casbin writes go through the shared synced enforcer, which has its own
	// adapter and therefore cannot join the SQL transaction above. They are
	// idempotent (AddPolicy is a no-op when the rule exists), so a partial
	// failure is repaired by re-running provisioning.
	result.PolicyCount = grantTenantRolePolicies(s.db, in.TenantID)

	// The copies are read-then-insert as well: the tenant-side existence check
	// decides what to insert, `system_setting` and `system_dict_type` would turn
	// a lost race into a duplicate-key 500, and `system_dict_item` has no unique
	// key at all, so a lost race would silently double the items. Taking the
	// same tenant row lock in a second short transaction extends the
	// serialisation above to the copy step without making provisioning and the
	// copies one long transaction.
	var (
		settings  int64
		dictTypes int64
		dictItems int64
	)
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := lockTenantRow(tx, in.TenantID); err != nil {
			return err
		}
		var err error
		if settings, err = s.copyPlatformSettingsToTenant(tx, in.TenantID); err != nil {
			return err
		}
		dictTypes, dictItems, err = s.copyPlatformDictsToTenant(tx, in.TenantID)
		return err
	})
	if err != nil {
		return nil, err
	}
	result.SettingsCopied = settings
	result.DictTypesCopied = dictTypes
	result.DictItemsCopied = dictItems

	return result, nil
}

// lockTenantRow takes the tenant's row lock for the provisioning transaction.
// Its only job is to serialise same-tenant provisioning; see the call site.
func lockTenantRow(tx *gorm.DB, tenantID uint64) error {
	var existing struct{ ID uint64 }
	if err := tx.Model(&Tenant{}).
		Select("id").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(condIDEquals, tenantID).
		Take(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTenantNotFound
		}
		return fmt.Errorf("lock tenant row: %w", err)
	}
	return nil
}

// provisionTenantRoles ensures the tenant owns its role set and returns the
// role_key -> role_id map.
//
// The unique key (tenant_id, role_key) is the arbiter, not a read-then-insert.
// Two callers provisioning the same tenant at once both used to miss the row
// and both insert; the loser then failed the whole provisioning transaction
// with a duplicate-entry error. OnConflict DoNothing makes the insert
// convergent, and the read-back is a locking read (SELECT ... FOR UPDATE) so it
// observes the winner's row: under REPEATABLE READ a plain SELECT keeps the
// transaction's older snapshot and would still come back empty after the
// loser's insert was suppressed.
func provisionTenantRoles(tx *gorm.DB, tenantID uint64) (map[string]uint64, error) {
	roleIDs := make(map[string]uint64, len(tenantRoleDefinitions))
	for _, def := range tenantRoleDefinitions {
		row := struct {
			TenantID uint64
			RoleName string
			RoleKey  string
			Sort     int
			Status   int
		}{TenantID: tenantID, RoleName: def.Name, RoleKey: def.Key, Sort: def.Sort, Status: 1}
		if err := tx.Table("system_role").
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&row).Error; err != nil {
			return nil, fmt.Errorf("create tenant role %s: %w", def.Key, err)
		}
		var stored struct{ ID uint64 }
		if err := tx.Table("system_role").
			Select("id").
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(condTenantIDMatches+" AND role_key = ?", tenantID, def.Key).
			Limit(1).
			Scan(&stored).Error; err != nil {
			return nil, fmt.Errorf("read back tenant role %s: %w", def.Key, err)
		}
		if stored.ID == 0 {
			return nil, fmt.Errorf("read back tenant role %s: row missing after upsert", def.Key)
		}
		roleIDs[def.Key] = stored.ID
	}
	return roleIDs, nil
}

// tenantPlatformMenuModule names the platform-only menu module for fixtures.
// Production classification goes through pkg/tenant.IsPlatformOnlyMenu.
const tenantPlatformMenuModule = tenantctx.PlatformOnlyMenuModuleTenant

// condRoleIDEquals is the predicate both role-binding tables are read and
// written by, so the placeholder style stays identical across the four queries.
const condRoleIDEquals = "role_id = ?"

// provisionTenantRoleMenus grants the tenant administrator the tenant console
// navigation.
//
// Role-menu grants are tenant-scoped (system_role_menu.tenant_id), so a freshly
// provisioned tenant starts with an empty sidebar unless the grants are seeded
// here: the navigation resolver joins system_role_menu by role id, and a new
// tenant's role ids have no rows at all. The symptom is silent — the tenant
// administrator logs in, the shell renders, and every page is unreachable
// because the menu tree is empty. The tenant smoke suite caught it on main
// immediately after role-menu grants became tenant-scoped.
//
// Platform-only menus (module system.tenant) are deliberately excluded: a tenant
// administrator must not see platform tenancy management, which is the same
// boundary the console capability signal enforces.
func provisionTenantRoleMenus(tx *gorm.DB, tenantID uint64, roleIDs map[string]uint64) error {
	if tenantID == tenantctx.PlatformGlobalTenantID {
		// The platform console keeps its own menu model (tenant 0 grants). A
		// tenant-scoped grant set here would be invisible to platform subjects
		// and would only add noise.
		return nil
	}
	adminRoleID := roleIDs[TenantRoleAdminKey]
	if adminRoleID == 0 {
		return errors.New("tenant provision: tenant_admin role missing")
	}
	type menuRow struct {
		ID     uint64 `gorm:"column:id"`
		Module string `gorm:"column:module"`
		Path   string `gorm:"column:path"`
	}
	var menus []menuRow
	if err := tx.Table("system_menu").
		Select(sqlSelectMenuIdentity).
		Where("is_visible = ? AND type <> ?", 1, "F").
		Order("sort asc, id asc").
		Scan(&menus).Error; err != nil {
		return fmt.Errorf("list tenant navigation menus: %w", err)
	}
	for _, menu := range menus {
		if tenantctx.IsPlatformOnlyMenu(menu.Module, menu.Path) {
			continue
		}
		if err := rbacbind.EnsureRoleMenu(tx, tenantID, adminRoleID, menu.ID); err != nil {
			return fmt.Errorf("grant tenant admin menu %d: %w", menu.ID, err)
		}
	}
	return nil
}

// tenantRootDeptName is the display name of the root department every tenant
// owns. It mirrors the platform tree's root name (dept.defaultRootDeptName) so
// a tenant console reads the same label at the top of its department tree.
const tenantRootDeptName = "Pantheon Base"

// provisionTenantRootDept ensures the tenant owns exactly one root department
// and returns its id (0 when tenantID is the platform global tenant).
//
// A tenant's department tree needs a root to hang off. DeptService.CreateDept
// rejects a top-level create (`parent_id = 0` -> `dept.parent.required`) by
// design — "departments must be placed under the organisation root or an
// existing department" — so without a provisioned root a freshly created tenant
// can never create its first department, and therefore never its first post
// either (PostService.ensurePostDeptID requires a non-root tenant department).
// Before this existed `system_dept` held rows for tenant 0 only, so every tenant
// console rendered an empty department tree that could not be populated from the
// UI: GET /system/dept/tree answered `[]` and the create form always failed.
//
// Idempotent: an existing root is reused, so re-running provisioning repairs a
// tenant that lost its root instead of duplicating it.
func provisionTenantRootDept(tx *gorm.DB, tenantID uint64) (uint64, error) {
	if tenantID == 0 {
		return 0, nil
	}

	var existing struct{ ID uint64 }
	if err := tx.Table("system_dept").
		Select("id").
		Where(condTenantIDMatches+" AND is_root = ?", tenantID, common.StatusFlagYes).
		Order("id asc").
		Limit(1).
		Scan(&existing).Error; err != nil {
		return 0, fmt.Errorf("look up tenant root dept: %w", err)
	}
	if existing.ID != 0 {
		return existing.ID, nil
	}

	row := struct {
		TenantID  uint64
		ParentID  uint64
		Ancestors string
		IsRoot    int
		DeptName  string
		Sort      int
		Status    int
	}{
		TenantID: tenantID,
		ParentID: 0,
		IsRoot:   common.StatusFlagYes,
		DeptName: tenantRootDeptName,
		Status:   common.StatusEnabled,
	}
	if err := tx.Table("system_dept").Create(&row).Error; err != nil {
		return 0, fmt.Errorf("create tenant root dept: %w", err)
	}

	var created struct{ ID uint64 }
	if err := tx.Table("system_dept").
		Select("id").
		Where(condTenantIDMatches+" AND is_root = ?", tenantID, common.StatusFlagYes).
		Order("id asc").
		Limit(1).
		Scan(&created).Error; err != nil {
		return 0, fmt.Errorf("read back tenant root dept: %w", err)
	}
	return created.ID, nil
}

// provisionTenantAdminUser creates the tenant's administrator. The username is
// unique per tenant, so each tenant may own its own `admin`.
func provisionTenantAdminUser(tx *gorm.DB, in ProvisionInput) (uint64, error) {
	username := strings.TrimSpace(in.AdminUsername)

	hash, err := bcrypt.GenerateFromPassword([]byte(in.AdminPassword), security.GetBcryptCost())
	if err != nil {
		return 0, fmt.Errorf("hash tenant admin password: %w", err)
	}
	nickname := strings.TrimSpace(in.AdminNickname)
	if nickname == "" {
		nickname = username
	}
	row := struct {
		TenantID uint64
		Username string
		Password string
		Nickname string
		Status   int
	}{TenantID: in.TenantID, Username: username, Password: string(hash), Nickname: nickname, Status: 1}
	// Same convergent shape as provisionTenantRoles: the unique key
	// (tenant_id, username) decides, so re-provisioning (and any caller that
	// slipped past the tenant row lock) reuses the row instead of failing.
	if err := tx.Table("system_user").
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&row).Error; err != nil {
		return 0, fmt.Errorf("create tenant admin: %w", err)
	}

	var created struct{ ID uint64 }
	if err := tx.Table("system_user").
		Select("id").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(condTenantIDMatches+" AND username = ?", in.TenantID, username).
		Limit(1).
		Scan(&created).Error; err != nil {
		return 0, fmt.Errorf("read back tenant admin: %w", err)
	}
	if created.ID == 0 {
		return 0, errors.New("read back tenant admin: row missing after upsert")
	}
	return created.ID, nil
}

// provisionTenantRoleBindings copies the platform `admin` role's permission and
// menu bindings onto the tenant's own roles.
//
// The product model for a tenant administrator is "the full console,
// automatically confined to the caller's tenant" — confinement is enforced by
// tenant.WithTenantScope and by the Casbin domain, NOT by shrinking the console
// surface. Until this step existed, provisioning created the roles and granted
// the Casbin policies but left the roles carrying ZERO permission keys, so
// GET /system/menu/tree answered with an empty tree and the tenant
// administrator signed into a console with no sidebar at all: the API surface
// worked, the console rendered nothing. The navigation tree is driven by
// permission keys (see MenuService.getScopedNavigationMenuTree), so the binding
// is what makes the console exist for a tenant subject.
//
// `tenant_member` receives the same menu visibility but only the read-only
// permission keys, matching its read-only Casbin surface (GET on /api/v1/*).
//
// Idempotent: existing bindings are skipped, so re-running provisioning is
// still a valid repair path.
func provisionTenantRoleBindings(tx *gorm.DB, tenantID uint64, roleIDs map[string]uint64) error {
	if tenantID == 0 || len(roleIDs) == 0 {
		return nil
	}
	platformRoleID, err := platformAdminRoleID(tx)
	if err != nil {
		return err
	}
	if platformRoleID == 0 {
		// No platform seed in this database (fresh schema, canary, unit-test
		// fixture). There is nothing to derive the console surface from, and
		// failing the tenant creation over it would be worse than leaving the
		// role without bindings.
		return nil
	}
	permissionKeys, menuIDs, err := readPlatformAdminBindings(tx, platformRoleID)
	if err != nil {
		return err
	}
	if len(permissionKeys) == 0 && len(menuIDs) == 0 {
		return nil
	}

	bindings := []struct {
		roleKey      string
		readOnlyOnly bool
	}{{roleKey: TenantRoleAdminKey}, {roleKey: TenantRoleMemberKey, readOnlyOnly: true}}

	for _, binding := range bindings {
		roleID := roleIDs[binding.roleKey]
		if roleID == 0 {
			continue
		}
		if err := grantTenantRolePermissions(tx, tenantID, roleID, binding.roleKey, permissionKeys, binding.readOnlyOnly); err != nil {
			return err
		}
		if err := grantTenantRoleMenus(tx, tenantID, roleID, binding.roleKey, menuIDs); err != nil {
			return err
		}
	}
	return nil
}

// platformAdminRoleID resolves the platform `admin` role, whose bindings are the
// source of the tenant console surface. A zero result means this database has no
// platform seed.
func platformAdminRoleID(tx *gorm.DB) (uint64, error) {
	var platformRoleID uint64
	if err := tx.Table("system_role").
		Select("id").
		Where("tenant_id = 0 AND role_key = ? AND status = ? AND deleted_at IS NULL", "admin", 1).
		Limit(1).
		Scan(&platformRoleID).Error; err != nil {
		return 0, fmt.Errorf("look up platform admin role: %w", err)
	}
	return platformRoleID, nil
}

// readPlatformAdminBindings reads the platform admin role's permission keys and
// menu ids, the two surfaces a tenant role is derived from. Platform-only
// bindings are dropped: the console capability gate refuses those routes for
// every tenant subject, so copying them would only produce menu entries whose
// pages answer 403 tenant.platformOnly (and stale permission keys that make the
// frontend expose actions the server rejects).
func readPlatformAdminBindings(tx *gorm.DB, platformRoleID uint64) ([]string, []uint64, error) {
	var permissionKeys []string
	if err := tx.Table("system_role_permission").
		Select("permission_key").
		Where(condRoleIDEquals, platformRoleID).
		Order("permission_key asc").
		Pluck("permission_key", &permissionKeys).Error; err != nil {
		return nil, nil, fmt.Errorf("read platform admin permissions: %w", err)
	}
	permissionKeys = tenantctx.FilterPlatformOnlyPermissionKeys(permissionKeys)

	var menuIDs []uint64
	if err := tx.Table("system_role_menu").
		Select("menu_id").
		Where(condRoleIDEquals, platformRoleID).
		Order("menu_id asc").
		Pluck("menu_id", &menuIDs).Error; err != nil {
		return nil, nil, fmt.Errorf("read platform admin menus: %w", err)
	}
	if len(menuIDs) == 0 {
		return permissionKeys, nil, nil
	}
	type menuRow struct {
		ID     uint64 `gorm:"column:id"`
		Module string `gorm:"column:module"`
		Path   string `gorm:"column:path"`
	}
	var menus []menuRow
	if err := tx.Table("system_menu").
		Select(sqlSelectMenuIdentity).
		Where("id IN ?", menuIDs).
		Scan(&menus).Error; err != nil {
		return nil, nil, fmt.Errorf("read platform admin menu rows: %w", err)
	}
	allowed := make(map[uint64]struct{}, len(menus))
	for _, menu := range menus {
		if tenantctx.IsPlatformOnlyMenu(menu.Module, menu.Path) {
			continue
		}
		allowed[menu.ID] = struct{}{}
	}
	filtered := make([]uint64, 0, len(menuIDs))
	for _, menuID := range menuIDs {
		if _, ok := allowed[menuID]; ok {
			filtered = append(filtered, menuID)
		}
	}
	return permissionKeys, filtered, nil
}

// RepairTenantPlatformGrants removes platform-only menu and permission grants
// from every tenant-owned role, and prunes write-capable keys from the built-in
// read-only member role.
//
// The grant copy used to mirror the platform administrator verbatim, so tenants
// provisioned before this fix carry `system.tenant` / `system:i18n` / menu-write
// bindings, and their member role carries write keys (security-event:clear,
// session:delete, ...). Re-provisioning only adds missing rows, so without this
// repair the old rows would outlive the fix and keep rendering menus and
// actions whose routes answer 403 tenant.platformOnly / permission.denied.
// Runs at startup, is idempotent and never touches the platform namespace
// (tenant_id = 0).
func (s *Service) RepairTenantPlatformGrants() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	return repairTenantPlatformGrants(s.db, 0)
}

// repairTenantPlatformGrants removes the platform-only grants inside one
// tenant's namespace, or every tenant namespace when tenantID is 0.
func repairTenantPlatformGrants(tx *gorm.DB, tenantID uint64) error {
	if tx == nil || tx.Migrator() == nil {
		return nil
	}
	if !tx.Migrator().HasTable("system_menu") ||
		!tx.Migrator().HasTable("system_role_menu") ||
		!tx.Migrator().HasTable("system_role_permission") {
		return nil
	}

	platformMenuIDs, err := collectPlatformOnlyMenuIDs(tx)
	if err != nil {
		return err
	}
	if err := deleteTenantMenuGrants(tx, tenantID, platformMenuIDs); err != nil {
		return err
	}

	platformKeys, err := collectTenantPlatformPermissionKeys(tx)
	if err != nil {
		return err
	}
	if err := deleteTenantPermissionGrants(tx, tenantID, platformKeys); err != nil {
		return err
	}
	return pruneTenantMemberWriteGrants(tx, tenantID)
}

// pruneTenantMemberWriteGrants keeps the built-in member role read-only.
//
// Provisioning has filtered write-capable keys for a while, but tenants
// provisioned before that filter still carry them (for example
// security-event:clear), so the console renders actions the member's GET-only
// Casbin surface refuses and the governance board reports a permanent gap.
// The repair is convergent: it drops exactly the keys the read-only contract
// does not allow, and leaves list/detail/view/tree/export untouched.
func pruneTenantMemberWriteGrants(tx *gorm.DB, tenantID uint64) error {
	if !tx.Migrator().HasTable("system_role") ||
		!tx.Migrator().HasTable("system_role_permission") {
		return nil
	}
	type memberRole struct {
		TenantID uint64
		RoleID   uint64
	}
	var roles []memberRole
	roleQuery := tx.Table("system_role").
		Select("tenant_id, id AS role_id").
		Where(condTenantScopedRow, tenantctx.PlatformGlobalTenantID).
		Where("role_key = ? AND deleted_at IS NULL", TenantRoleMemberKey)
	if tenantID != 0 {
		roleQuery = roleQuery.Where(condTenantIDMatches, tenantID)
	}
	if err := roleQuery.Scan(&roles).Error; err != nil {
		return fmt.Errorf("repair tenant member grants: list roles: %w", err)
	}
	for _, role := range roles {
		var keys []string
		if err := tx.Table("system_role_permission").
			Where("tenant_id = ? AND role_id = ?", role.TenantID, role.RoleID).
			Pluck("permission_key", &keys).Error; err != nil {
			return fmt.Errorf("repair tenant member grants: list permission keys: %w", err)
		}
		writeKeys := make([]string, 0, len(keys))
		for _, key := range keys {
			if !isReadOnlyPermissionKey(key) {
				writeKeys = append(writeKeys, key)
			}
		}
		if len(writeKeys) == 0 {
			continue
		}
		if err := tx.Table("system_role_permission").
			Where("tenant_id = ? AND role_id = ? AND permission_key IN ?", role.TenantID, role.RoleID, writeKeys).
			Delete(nil).Error; err != nil {
			return fmt.Errorf("repair tenant member grants: delete write keys: %w", err)
		}
	}
	return nil
}

// collectPlatformOnlyMenuIDs resolves every menu row that belongs to a
// platform-only surface, including the button rows that hang under such a page
// (they carry no path of their own, so they are matched by parent).
func collectPlatformOnlyMenuIDs(tx *gorm.DB) ([]uint64, error) {
	type menuRow struct {
		ID     uint64 `gorm:"column:id"`
		Module string `gorm:"column:module"`
		Path   string `gorm:"column:path"`
	}
	var menus []menuRow
	if err := tx.Table("system_menu").Select(sqlSelectMenuIdentity).Scan(&menus).Error; err != nil {
		return nil, fmt.Errorf("repair tenant platform grants: list menus: %w", err)
	}
	platformMenuIDs := make([]uint64, 0)
	for _, menu := range menus {
		if tenantctx.IsPlatformOnlyMenu(menu.Module, menu.Path) {
			platformMenuIDs = append(platformMenuIDs, menu.ID)
		}
	}
	if len(platformMenuIDs) == 0 {
		return platformMenuIDs, nil
	}
	var childIDs []uint64
	if err := tx.Table("system_menu").
		Select("id").
		Where("parent_id IN ?", platformMenuIDs).
		Pluck("id", &childIDs).Error; err != nil {
		return nil, fmt.Errorf("repair tenant platform grants: list platform button rows: %w", err)
	}
	return append(platformMenuIDs, childIDs...), nil
}

// deleteTenantMenuGrants removes the menu grants that point at a platform-only
// surface, inside one tenant namespace or every tenant namespace (tenantID 0).
func deleteTenantMenuGrants(tx *gorm.DB, tenantID uint64, menuIDs []uint64) error {
	if len(menuIDs) == 0 {
		return nil
	}
	menuGrants := tx.Table("system_role_menu").
		Where(condTenantScopedRow, tenantctx.PlatformGlobalTenantID).
		Where("menu_id IN ?", menuIDs)
	if tenantID != 0 {
		menuGrants = menuGrants.Where(condTenantIDMatches, tenantID)
	}
	if err := menuGrants.Delete(nil).Error; err != nil {
		return fmt.Errorf("repair tenant platform grants: delete menu grants: %w", err)
	}
	return nil
}

// collectTenantPlatformPermissionKeys lists the permission keys stored under a
// tenant namespace that only a platform-capable subject may hold.
func collectTenantPlatformPermissionKeys(tx *gorm.DB) ([]string, error) {
	var tenantKeys []string
	if err := tx.Table("system_role_permission").
		Distinct("permission_key").
		Where(condTenantScopedRow, tenantctx.PlatformGlobalTenantID).
		Pluck("permission_key", &tenantKeys).Error; err != nil {
		return nil, fmt.Errorf("repair tenant platform grants: list permission keys: %w", err)
	}
	platformKeys := make([]string, 0, len(tenantKeys))
	for _, key := range tenantKeys {
		if tenantctx.IsPlatformOnlyPermissionKey(key) {
			platformKeys = append(platformKeys, key)
		}
	}
	return platformKeys, nil
}

// deleteTenantPermissionGrants removes the platform-only permission grants from
// one tenant namespace or every tenant namespace (tenantID 0).
func deleteTenantPermissionGrants(tx *gorm.DB, tenantID uint64, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	permissionGrants := tx.Table("system_role_permission").
		Where(condTenantScopedRow, tenantctx.PlatformGlobalTenantID).
		Where("permission_key IN ?", keys)
	if tenantID != 0 {
		permissionGrants = permissionGrants.Where(condTenantIDMatches, tenantID)
	}
	if err := permissionGrants.Delete(nil).Error; err != nil {
		return fmt.Errorf("repair tenant platform grants: delete permission grants: %w", err)
	}
	return nil
}

// grantTenantRolePermissions copies the missing permission keys onto one tenant
// role. readOnlyOnly keeps a read-only role on the GET-only subset, so the
// console never advertises an action the server would refuse.
//
// The read above is only a fast path. The unique key
// (tenant_id, role_id, permission_key) is the authority: two concurrent repair
// runs that both miss the same key must converge on one row, not fail the loser
// with a duplicate-entry 500.
func grantTenantRolePermissions(
	tx *gorm.DB,
	tenantID, roleID uint64,
	roleKey string,
	permissionKeys []string,
	readOnlyOnly bool,
) error {
	existing := make(map[string]struct{})
	var currentKeys []string
	if err := tx.Table("system_role_permission").
		Select("permission_key").
		Where(condRoleIDEquals, roleID).
		Pluck("permission_key", &currentKeys).Error; err != nil {
		return fmt.Errorf("read existing permissions for %s: %w", roleKey, err)
	}
	for _, key := range currentKeys {
		existing[key] = struct{}{}
	}
	type permissionGrant struct {
		TenantID      uint64
		RoleID        uint64
		PermissionKey string
	}
	grants := make([]permissionGrant, 0, len(permissionKeys))
	for _, key := range permissionKeys {
		if readOnlyOnly && !isReadOnlyPermissionKey(key) {
			continue
		}
		if _, taken := existing[key]; taken {
			continue
		}
		grants = append(grants, permissionGrant{TenantID: tenantID, RoleID: roleID, PermissionKey: key})
	}
	if len(grants) == 0 {
		return nil
	}
	if err := tx.Table("system_role_permission").
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(grants, 100).Error; err != nil {
		return fmt.Errorf("grant permissions to %s: %w", roleKey, err)
	}
	return nil
}

// grantTenantRoleMenus copies the missing menu grants onto one tenant role, so
// the tenant console renders the same navigation as the platform one. The
// composite primary key is the arbiter, for the same concurrency reason as
// grantTenantRolePermissions.
func grantTenantRoleMenus(tx *gorm.DB, tenantID, roleID uint64, roleKey string, menuIDs []uint64) error {
	existing := make(map[uint64]struct{})
	var currentMenus []uint64
	if err := tx.Table("system_role_menu").
		Select("menu_id").
		Where(condRoleIDEquals, roleID).
		Pluck("menu_id", &currentMenus).Error; err != nil {
		return fmt.Errorf("read existing menus for %s: %w", roleKey, err)
	}
	for _, id := range currentMenus {
		existing[id] = struct{}{}
	}
	type menuGrant struct {
		RoleID   uint64
		MenuID   uint64
		TenantID uint64
	}
	grants := make([]menuGrant, 0, len(menuIDs))
	for _, id := range menuIDs {
		if _, taken := existing[id]; taken {
			continue
		}
		grants = append(grants, menuGrant{RoleID: roleID, MenuID: id, TenantID: tenantID})
	}
	if len(grants) == 0 {
		return nil
	}
	if err := tx.Table("system_role_menu").
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(grants, 100).Error; err != nil {
		return fmt.Errorf("grant menus to %s: %w", roleKey, err)
	}
	return nil
}

// isReadOnlyPermissionKey reports whether a permission key only reads. A tenant
// member's Casbin surface is GET on /api/v1/*, so handing it the write-capable
// keys would render action buttons that the server then refuses — the console
// would advertise capabilities it does not have.
func isReadOnlyPermissionKey(key string) bool {
	switch {
	case strings.HasSuffix(key, ":list"),
		strings.HasSuffix(key, ":detail"),
		strings.HasSuffix(key, ":view"),
		strings.HasSuffix(key, ":tree"),
		strings.HasSuffix(key, ":export"):
		return true
	default:
		return false
	}
}

// provisionTenantAdminBinding binds the admin user to the tenant admin role and
// records the owner membership the login issuance gate checks.
func provisionTenantAdminBinding(tx *gorm.DB, tenantID, userID uint64, roleIDs map[string]uint64) error {
	adminRoleID := roleIDs[TenantRoleAdminKey]
	if adminRoleID == 0 {
		return errors.New("tenant provision: tenant_admin role missing")
	}

	var bound int64
	if err := tx.Table("system_user_role").
		Where(condTenantIDMatches+" AND user_id = ? AND role_id = ?", tenantID, userID, adminRoleID).
		Count(&bound).Error; err != nil {
		return fmt.Errorf("check tenant admin role binding: %w", err)
	}
	if bound == 0 {
		row := struct {
			TenantID uint64
			UserID   uint64
			RoleID   uint64
		}{TenantID: tenantID, UserID: userID, RoleID: adminRoleID}
		if err := tx.Table("system_user_role").
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&row).Error; err != nil {
			return fmt.Errorf("bind tenant admin role: %w", err)
		}
	}

	var member int64
	if err := tx.Table("tenant_memberships").
		Where(condTenantIDMatches+" AND user_id = ?", tenantID, userID).
		Count(&member).Error; err != nil {
		return fmt.Errorf("check tenant owner membership: %w", err)
	}
	if member == 0 {
		row := struct {
			TenantID uint64
			UserID   uint64
			Role     string
			Status   string
		}{TenantID: tenantID, UserID: userID, Role: "owner", Status: "active"}
		if err := tx.Table("tenant_memberships").
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&row).Error; err != nil {
			return fmt.Errorf("create tenant owner membership: %w", err)
		}
	}
	return nil
}

// grantTenantRolePolicies writes the Casbin policies for the tenant's roles as
// the domain subjects role:<key>@tenant:<id> (contract §4). The middleware
// already consults exactly these subjects for a tenant-scoped request, so this
// is the step that turns a membership role into real permissions.
func grantTenantRolePolicies(db *gorm.DB, tenantID uint64) int {
	if database.Enforcer == nil {
		return 0
	}
	added := 0
	for _, def := range tenantRoleDefinitions {
		subject := tenantctx.RoleSubject(def.Key, tenantID)
		for _, method := range def.Methods {
			ok, err := database.Enforcer.AddPolicy(subject, "/api/v1/*", method)
			if err == nil && ok {
				added++
			}
		}
	}
	return added + grantTenantMemberExportPolicies(db, tenantID)
}

// grantTenantMemberExportPolicies grants the exact POST routes behind the
// `:export` permission keys the member role holds. A member's wildcard grant is
// GET-only, but every export endpoint is POST, so without these rules the
// console renders an export action that the middleware then refuses. Only keys
// the member actually holds are granted, so the repair stays additive and never
// widens the role beyond its permission set.
func grantTenantMemberExportPolicies(db *gorm.DB, tenantID uint64) int {
	if db == nil {
		return 0
	}
	var roleID uint64
	if err := db.Table("system_role").
		Where(condTenantIDMatches+" AND role_key = ? AND deleted_at IS NULL", tenantID, TenantRoleMemberKey).
		Limit(1).
		Pluck("id", &roleID).Error; err != nil || roleID == 0 {
		return 0
	}
	var keys []string
	if err := db.Table("system_role_permission").
		Where(condTenantIDMatches+" AND role_id = ?", tenantID, roleID).
		Pluck("permission_key", &keys).Error; err != nil {
		return 0
	}
	held := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		held[strings.TrimSpace(key)] = struct{}{}
	}
	subject := tenantctx.RoleSubject(TenantRoleMemberKey, tenantID)
	added := 0
	for _, route := range tenantctx.ReadOnlyExportRoutes {
		if _, ok := held[route.PermissionKey]; !ok {
			continue
		}
		if ok, err := database.Enforcer.AddPolicy(subject, route.Path, "POST"); err == nil && ok {
			added++
		}
	}
	return added
}

// platformScopedSettingKeys are deployment-wide flags that must NOT be copied
// into a tenant. They describe the deployment itself, not tenant behaviour:
//   - platform.tenant_mode: one mode per deployment, one-way switch
//     (contract §6/§7). A per-tenant copy would let a tenant's row shadow the
//     platform flag for every reader that is not tenant-scoped.
//   - platform.app_mode: the deployment's application mode (enterprise /
//     consumer / hybrid), read platform-wide by pkg/capability.
//
// Everything else (site.*, security.*, upload.*, ui.*, ...) is legitimately
// tenant-overridable and is copied so the tenant renders with sane defaults.
var platformScopedSettingKeys = []string{
	"platform.tenant_mode",
	"platform.app_mode",
}

// platformSettingRow is the column set a platform setting is copied through.
// One struct serves both halves of the copy, so a column cannot be read but not
// written (or the reverse) without the compiler noticing.
//
// setting_value and remark are nullable in the schema and stay pointers: the
// copy must reproduce a NULL as NULL rather than turn it into an empty string,
// which would be indistinguishable from a deliberately blanked setting.
type platformSettingRow struct {
	TenantID     uint64    `gorm:"column:tenant_id"`
	SettingKey   string    `gorm:"column:setting_key"`
	SettingValue *string   `gorm:"column:setting_value"`
	GroupKey     string    `gorm:"column:group_key"`
	Module       string    `gorm:"column:module"`
	LabelKey     string    `gorm:"column:label_key"`
	Description  string    `gorm:"column:description"`
	ValueType    string    `gorm:"column:value_type"`
	Sort         int       `gorm:"column:sort"`
	IsPublic     int64     `gorm:"column:is_public"`
	IsEncrypted  int64     `gorm:"column:is_encrypted"`
	Remark       *string   `gorm:"column:remark"`
	IsReadonly   int64     `gorm:"column:is_readonly"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

// copyPlatformSettingsToTenant seeds a new tenant with the deployment-wide
// settings so a tenant-scoped read returns real values instead of an empty set
// (contract §3.3: settings are tenant-scoped). Existing tenant rows win — the
// copy only fills keys the tenant does not have yet.
//
// Deployment-wide flags are excluded (platformScopedSettingKeys): they are
// platform-scoped by contract and a tenant copy would make any reader that
// runs without a resolved tenant context non-deterministic.
//
// Every NOT-NULL column without a default must be copied explicitly:
// `group_key` is one of them, and omitting it fails the whole insert with
// MySQL 1364 ("Field 'group_key' doesn't have a default value"). Copying the
// full descriptive column set also keeps the tenant's settings renderable —
// a row with only a key and a value would lose its group, module and type.
//
// The copy is a read followed by an insert, not `INSERT ... SELECT` over the
// same table. MySQL takes shared next-key locks on the rows an INSERT ... SELECT
// reads, and when source and target are the same table those locks collide with
// the insert-intention locks of a concurrent copy: two tenants created at the
// same moment deadlock with
//
//	Error 1213 (40001): Deadlock found when trying to get lock
//
// and the operator sees a 500 on tenant creation. A plain SELECT is a
// non-locking consistent read, so two concurrent copies no longer contend.
func (s *Service) copyPlatformSettingsToTenant(db *gorm.DB, tenantID uint64) (int64, error) {
	if tenantID == 0 {
		return 0, nil
	}

	var source []platformSettingRow
	if err := db.Table("system_setting").
		Select("setting_key, setting_value, group_key, module, label_key, description, "+
			"value_type, sort, is_public, is_encrypted, remark, is_readonly").
		Where(condTenantIDMatches, platformGlobalTenantID).
		Where("setting_key NOT IN ?", platformScopedSettingKeys).
		Scan(&source).Error; err != nil {
		return 0, fmt.Errorf("read platform settings: %w", err)
	}
	if len(source) == 0 {
		return 0, nil
	}

	owned, err := s.tenantSettingKeys(db, tenantID)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	copies := make([]platformSettingRow, 0, len(source))
	for _, row := range source {
		if _, taken := owned[row.SettingKey]; taken {
			continue
		}
		row.TenantID = tenantID
		row.CreatedAt = now
		row.UpdatedAt = now
		copies = append(copies, row)
	}
	if len(copies) == 0 {
		return 0, nil
	}
	if err := db.Table("system_setting").Create(&copies).Error; err != nil {
		return 0, fmt.Errorf("copy platform settings to tenant: %w", err)
	}
	return int64(len(copies)), nil
}

// tenantSettingKeys lists the setting keys a tenant already owns. Re-provisioning
// is a repair entry point, so the copy has to skip them instead of failing on
// the (tenant_id, setting_key) unique key.
func (s *Service) tenantSettingKeys(db *gorm.DB, tenantID uint64) (map[string]struct{}, error) {
	var keys []string
	if err := db.Table("system_setting").
		Select("setting_key").
		Where(condTenantIDMatches, tenantID).
		Pluck("setting_key", &keys).Error; err != nil {
		return nil, fmt.Errorf("read tenant setting keys: %w", err)
	}
	owned := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		owned[key] = struct{}{}
	}
	return owned, nil
}

// platformDictTypeRow and platformDictItemRow are the copied column sets for the
// two dictionary tables, for the same reason platformSettingRow exists.
type platformDictTypeRow struct {
	TenantID  uint64    `gorm:"column:tenant_id"`
	DictCode  string    `gorm:"column:dict_code"`
	DictName  string    `gorm:"column:dict_name"`
	Module    string    `gorm:"column:module"`
	Status    int       `gorm:"column:status"`
	Remark    *string   `gorm:"column:remark"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

type platformDictItemRow struct {
	TenantID     uint64    `gorm:"column:tenant_id"`
	DictCode     string    `gorm:"column:dict_code"`
	ItemLabelKey string    `gorm:"column:item_label_key"`
	ItemValue    string    `gorm:"column:item_value"`
	ItemColor    *string   `gorm:"column:item_color"`
	Sort         int       `gorm:"column:sort"`
	Status       int       `gorm:"column:status"`
	Remark       *string   `gorm:"column:remark"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

// copyPlatformDictsToTenant seeds the tenant with the platform's default
// dictionaries (system_dict_type + system_dict_item) so tenant-scoped pages
// resolve their labels out of the box instead of showing empty select options.
//
// Same idempotency rules as copyPlatformSettingsToTenant, and the same
// read-then-insert shape for the same deadlock reason:
//   - existing tenant-owned keys win — a tenant's private dict (or its own
//     customization of a copied one) is never overwritten, which is also how
//     tenant-private key conflicts are excluded;
//   - the tenant-side existence check deliberately does NOT filter deleted_at:
//     soft deleted rows still occupy the (tenant_id, dict_code[, item_value])
//     unique keys, so re-inserting them would fail as a duplicate — skipping
//     them keeps the copy retry-safe;
//   - deleted platform rows are not copied (deleted_at IS NULL);
//   - every NOT-NULL-without-default column is listed explicitly
//     (dict_name / item_label_key / item_value), matching the MySQL 1364 rule
//     documented on the settings copy.
func (s *Service) copyPlatformDictsToTenant(db *gorm.DB, tenantID uint64) (int64, int64, error) {
	if tenantID == 0 {
		return 0, 0, nil
	}
	typesCopied, err := s.copyPlatformDictTypes(db, tenantID)
	if err != nil {
		return 0, 0, err
	}
	itemsCopied, err := s.copyPlatformDictItems(db, tenantID)
	if err != nil {
		return typesCopied, 0, err
	}
	return typesCopied, itemsCopied, nil
}

func (s *Service) copyPlatformDictTypes(db *gorm.DB, tenantID uint64) (int64, error) {
	var source []platformDictTypeRow
	if err := db.Table("system_dict_type").
		Select("dict_code, dict_name, module, status, remark").
		Where(condTenantIDMatches, platformGlobalTenantID).
		Where("deleted_at IS NULL").
		Scan(&source).Error; err != nil {
		return 0, fmt.Errorf("read platform dict types: %w", err)
	}
	if len(source) == 0 {
		return 0, nil
	}

	var ownedCodes []string
	if err := db.Table("system_dict_type").
		Select("dict_code").
		Where(condTenantIDMatches, tenantID).
		Pluck("dict_code", &ownedCodes).Error; err != nil {
		return 0, fmt.Errorf("read tenant dict types: %w", err)
	}
	owned := make(map[string]struct{}, len(ownedCodes))
	for _, code := range ownedCodes {
		owned[code] = struct{}{}
	}

	now := time.Now()
	copies := make([]platformDictTypeRow, 0, len(source))
	for _, row := range source {
		if _, taken := owned[row.DictCode]; taken {
			continue
		}
		row.TenantID = tenantID
		row.CreatedAt = now
		row.UpdatedAt = now
		copies = append(copies, row)
	}
	if len(copies) == 0 {
		return 0, nil
	}
	if err := db.Table("system_dict_type").Create(&copies).Error; err != nil {
		return 0, fmt.Errorf("copy platform dict types to tenant: %w", err)
	}
	return int64(len(copies)), nil
}

func (s *Service) copyPlatformDictItems(db *gorm.DB, tenantID uint64) (int64, error) {
	var source []platformDictItemRow
	if err := db.Table("system_dict_item").
		Select("dict_code, item_label_key, item_value, item_color, sort, status, remark").
		Where(condTenantIDMatches, platformGlobalTenantID).
		Where("deleted_at IS NULL").
		Scan(&source).Error; err != nil {
		return 0, fmt.Errorf("read platform dict items: %w", err)
	}
	if len(source) == 0 {
		return 0, nil
	}

	// The tenant-side key is (dict_code, item_value); the separator is a byte a
	// dictionary value cannot contain, so the pair cannot alias another pair.
	const pairSeparator = "\x00"
	var ownedPairs []struct {
		DictCode  string `gorm:"column:dict_code"`
		ItemValue string `gorm:"column:item_value"`
	}
	if err := db.Table("system_dict_item").
		Select("dict_code, item_value").
		Where(condTenantIDMatches, tenantID).
		Scan(&ownedPairs).Error; err != nil {
		return 0, fmt.Errorf("read tenant dict items: %w", err)
	}
	owned := make(map[string]struct{}, len(ownedPairs))
	for _, pair := range ownedPairs {
		owned[pair.DictCode+pairSeparator+pair.ItemValue] = struct{}{}
	}

	now := time.Now()
	copies := make([]platformDictItemRow, 0, len(source))
	for _, row := range source {
		if _, taken := owned[row.DictCode+pairSeparator+row.ItemValue]; taken {
			continue
		}
		row.TenantID = tenantID
		row.CreatedAt = now
		row.UpdatedAt = now
		copies = append(copies, row)
	}
	if len(copies) == 0 {
		return 0, nil
	}
	if err := db.Table("system_dict_item").Create(&copies).Error; err != nil {
		return 0, fmt.Errorf("copy platform dict items to tenant: %w", err)
	}
	return int64(len(copies)), nil
}
