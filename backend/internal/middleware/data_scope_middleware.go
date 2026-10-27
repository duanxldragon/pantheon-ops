package middleware

import (
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// --- DataScope caches ---
var (
	// User dept cache: userID -> deptID
	userDeptCacheMu sync.RWMutex
	userDeptCache   = make(map[string]userDeptEntry)
	userDeptTTL     = 5 * time.Minute

	// Table existence cache (avoid DDL metadata queries per request)
	tableExistCacheMu sync.RWMutex
	tableExistCache   = make(map[string]bool)
)

type userDeptEntry struct {
	deptID   uint64
	cachedAt time.Time
}

// SystemRoleDataScope mirrors the `system_role_data_scope` table.
//
// TenantID is part of the identity, not decoration: the table's unique key is
// (tenant_id, role_key), because two tenants may legitimately own a role with
// the same key. Without this field the row-level policy read could only be
// expressed as `WHERE role_key IN (...)`, which merged every tenant's policy
// and let one tenant widen or narrow another tenant's visible rows
// (acceptance F10).
type SystemRoleDataScope struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement"`
	TenantID uint64 `gorm:"not null;default:0;uniqueIndex:uk_role_data_scope_tenant_role,priority:1"`
	RoleKey  string `gorm:"size:64;not null;uniqueIndex:uk_role_data_scope_tenant_role,priority:2"`
	Mode     string `gorm:"size:32;not null;default:'all'"`
	DeptIDs  string `gorm:"type:text"`
}

func (SystemRoleDataScope) TableName() string {
	return "system_role_data_scope"
}

func MigrateDataScopePolicy(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if database.ShouldAutoMigrate() {
		if err := db.AutoMigrate(&SystemRoleDataScope{}); err != nil {
			return err
		}
		storeCachedTableExistence(db, (&SystemRoleDataScope{}).TableName(), true)
	} else if !cachedHasTable(db, &SystemRoleDataScope{}) {
		if err := db.Exec(`
CREATE TABLE IF NOT EXISTS system_role_data_scope (
	id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
	tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
	role_key VARCHAR(64) NOT NULL,
	mode VARCHAR(32) NOT NULL DEFAULT 'all',
	dept_ids TEXT NULL,
	UNIQUE KEY uk_role_data_scope_tenant_role (tenant_id, role_key)
		)`).Error; err != nil {
			return err
		}
		storeCachedTableExistence(db, (&SystemRoleDataScope{}).TableName(), true)
	}
	// Existing roles are backfilled by the versioned database migration. Do not
	// repeat that repair on every startup: doing so would resurrect a policy
	// that an operator intentionally deleted. Runtime reads fail closed instead.
	return nil
}

func DataScopeMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := common.GetUserID(c)
		roleKeys := common.GetRoleKeys(c)
		scope := &common.DataScopeReq{
			UserID:   userID,
			RoleKeys: roleKeys,
			Mode:     common.DataScopeModeAll,
			IsAdmin:  hasAdministrativeRole(roleKeys),
		}
		tenantID := resolvePolicyTenantID(c)

		if db != nil && userID > 0 {
			scope.DeptID = loadCurrentUserDeptID(db, userID)
		}
		// 策略读取失败时 fail-closed：非管理员必须拒绝，绝不能按 all 放行全量数据。
		// 例外：租户内角色（tenant_admin / tenant_member）由 tenant scope 无条件收窄，
		// 本策略（平台级、按 dept 的行级机制）不适用于它们，否则租户成员会因没有
		// system_role_data_scope 行而落入 fail-closed 分支，每个请求都 500。
		if db != nil && !scope.IsAdmin && !hasTenantLocalRole(roleKeys) {
			if applyRoleDataScopePolicy(db, scope, tenantID) {
				slog.Warn("data scope: policy lookup failed, denying request", "roles", strings.Join(roleKeys, ","))
				if clientAcceptsJSON(c) {
					common.Fail(c, common.CodeError, "common.serverError")
				} else {
					c.String(http.StatusInternalServerError, "data scope policy unavailable")
				}
				c.Abort()
				return
			}
		}

		c.Set(common.DataScopeContextKey, scope)
		c.Next()
	}
}

// clientAcceptsJSON 判断客户端是否能解析统一 JSON 错误响应；导出等二进制下载走纯文本。
func clientAcceptsJSON(c *gin.Context) bool {
	accept := strings.ToLower(c.GetHeader("Accept"))
	return accept == "" || strings.Contains(accept, "json") || strings.Contains(accept, "*/*") || strings.Contains(accept, "html")
}

func loadCurrentUserDeptID(db *gorm.DB, userID uint64) uint64 {
	cacheKey := buildUserDeptCacheKey(db, userID)

	// Check cache first
	userDeptCacheMu.RLock()
	if entry, ok := userDeptCache[cacheKey]; ok && time.Since(entry.cachedAt) < userDeptTTL {
		userDeptCacheMu.RUnlock()
		return entry.deptID
	}
	userDeptCacheMu.RUnlock()

	var deptID uint64
	if err := db.Table("system_user").Select("dept_id").Where("id = ?", userID).Limit(1).Pluck("dept_id", &deptID).Error; err != nil {
		slog.Warn("data scope: failed to load user dept id", "userID", userID, "error", err)
	}

	// Store in cache
	userDeptCacheMu.Lock()
	userDeptCache[cacheKey] = userDeptEntry{deptID: deptID, cachedAt: time.Now()}
	if len(userDeptCache) > 10000 {
		now := time.Now()
		for k, v := range userDeptCache {
			if now.Sub(v.cachedAt) > userDeptTTL {
				delete(userDeptCache, k)
			}
		}
	}
	userDeptCacheMu.Unlock()

	return deptID
}

// resolvePolicyTenantID returns the tenant whose data-scope policies govern this
// request. Compat / unresolvable contexts fall back to the platform/global
// scope (0), which is also where platform roles' policies live, so the read
// stays deterministic instead of merging every tenant's rows.
func resolvePolicyTenantID(c *gin.Context) uint64 {
	if ctx := tenant.FromGin(c); ctx != nil && ctx.IsMulti() {
		return ctx.TenantID
	}
	return tenant.PlatformGlobalTenantID
}

// applyRoleDataScopePolicy 加载角色数据范围策略并写入 scope。
// 返回 true 表示策略读取失败（fail-closed），调用方必须拒绝本次请求。
func applyRoleDataScopePolicy(db *gorm.DB, scope *common.DataScopeReq, tenantID uint64) bool {
	if scope == nil {
		return true
	}
	if len(scope.RoleKeys) == 0 {
		return false
	}
	for _, roleKey := range scope.RoleKeys {
		if strings.TrimSpace(roleKey) == "" {
			scope.Mode = common.DataScopeModeCustom
			scope.DeptIDs = nil
			return true
		}
	}
	// Query the policy table directly. A missing table or metadata lookup failure
	// must be treated like any other policy read failure, never as mode=all.

	policies, ok := loadRoleDataScopePolicies(db, tenantID, scope.RoleKeys)
	if !ok {
		// 读取失败时按 custom + 空 dept 集合处理：即使调用方忽略返回值，
		// WithDataScope 也会因空集合拒绝所有行，绝不能退回 all。
		scope.Mode = common.DataScopeModeCustom
		scope.DeptIDs = nil
		return true
	}

	if !rolePoliciesCoverAll(scope.RoleKeys, policies) {
		scope.Mode = common.DataScopeModeCustom
		scope.DeptIDs = nil
		return true
	}
	for _, policy := range policies {
		if !isValidDataScopePolicyMode(policy.Mode) {
			scope.Mode = common.DataScopeModeCustom
			scope.DeptIDs = nil
			return true
		}
	}

	scope.Mode = resolveDataScopeMode(policies)
	switch scope.Mode {
	case common.DataScopeModeCustom:
		scope.DeptIDs = resolveCustomDataScopeDeptIDs(policies)
	case common.DataScopeModeDeptAndChildren:
		scope.DeptIDs = loadDeptAndChildrenIDs(db, scope.DeptID)
	case common.DataScopeModeDept, common.DataScopeModeSelf:
		scope.DeptIDs = nil
	case common.DataScopeModeAll:
		scope.DeptIDs = nil
	}
	return false
}

// loadRoleDataScopePolicies reads the policies that govern roleKeys in exactly
// one tenant scope.
//
// The tenant predicate is the whole point: role_key is unique per tenant, not
// globally, so `WHERE role_key IN (...)` returns several tenants' rows and the
// resolver then takes the widest one. Rows written before the tenant dimension
// was enforced live at tenant 0 and are backfilled by migration 000025.
func loadRoleDataScopePolicies(db *gorm.DB, tenantID uint64, roleKeys []string) ([]SystemRoleDataScope, bool) {
	var policies []SystemRoleDataScope
	if err := db.Where("tenant_id = ? AND role_key IN ?", tenantID, roleKeys).Find(&policies).Error; err != nil {
		slog.Warn("data scope: failed to load role policies",
			"tenantID", tenantID, "roles", strings.Join(roleKeys, ","), "error", err)
		return nil, false
	}
	return policies, true
}

func rolePoliciesCoverAll(roleKeys []string, policies []SystemRoleDataScope) bool {
	covered := make(map[string]bool, len(policies))
	for _, policy := range policies {
		covered[policy.RoleKey] = true
	}
	for _, key := range roleKeys {
		key = strings.TrimSpace(key)
		if key == "" || !covered[key] {
			return false
		}
	}
	return true
}

// cachedHasTable caches the result of db.Migrator().HasTable() to avoid
// per-request DDL metadata queries.
func cachedHasTable(db *gorm.DB, model interface{}) bool {
	var tableName string
	switch v := model.(type) {
	case string:
		tableName = v
	case interface{ TableName() string }:
		tableName = v.TableName()
	}
	if tableName == "" {
		return db.Migrator().HasTable(model)
	}
	cacheKey := buildTableExistCacheKey(db, tableName)
	tableExistCacheMu.RLock()
	if exists, ok := tableExistCache[cacheKey]; ok {
		tableExistCacheMu.RUnlock()
		return exists
	}
	tableExistCacheMu.RUnlock()

	exists := db.Migrator().HasTable(model)
	storeCachedTableExistence(db, tableName, exists)
	return exists
}

func storeCachedTableExistence(db *gorm.DB, tableName string, exists bool) {
	cacheKey := buildTableExistCacheKey(db, tableName)
	tableExistCacheMu.Lock()
	tableExistCache[cacheKey] = exists
	tableExistCacheMu.Unlock()
}

func buildUserDeptCacheKey(db *gorm.DB, userID uint64) string {
	return buildDatabaseCacheNamespace(db) + ":user:" + strconv.FormatUint(userID, 10)
}

func buildTableExistCacheKey(db *gorm.DB, tableName string) string {
	return buildDatabaseCacheNamespace(db) + ":table:" + tableName
}

// buildDatabaseCacheNamespace 为每个逻辑数据库生成缓存命名空间。
// 命名空间必须来自连接身份（DSN 的 host + 库名）而不是连接池指针：
// 连接池被关闭并回收后，Go 运行时可能把同一地址分配给新的连接池，
// 使无关连接继承上一个库的缓存结论（历史上曾让 MigrateDataScopePolicy
// 误判策略表已存在而跳过建表）。详见 database.ConnectionCacheNamespace。
func buildDatabaseCacheNamespace(db *gorm.DB) string {
	return database.ConnectionCacheNamespace(db)
}

func loadDeptAndChildrenIDs(db *gorm.DB, deptID uint64) []uint64 {
	if db == nil || deptID == 0 {
		return nil
	}
	if !cachedHasTable(db, "system_dept") {
		return []uint64{deptID}
	}

	var ids []uint64
	deptIDText := strconv.FormatUint(deptID, 10)
	if err := db.Table("system_dept").
		Select("id").
		Where(
			"id = ? OR ancestors = ? OR ancestors LIKE ? OR ancestors LIKE ? OR ancestors LIKE ?",
			deptID,
			deptIDText,
			deptIDText+",%",
			"%,"+deptIDText+",%",
			"%,"+deptIDText,
		).
		Order("id asc").
		Pluck("id", &ids).Error; err != nil {
		slog.Warn("data scope: failed to expand dept children", "deptID", deptID, "error", err)
		return []uint64{deptID}
	}
	if len(ids) == 0 {
		return []uint64{deptID}
	}
	return ids
}

func parseDataScopeDeptIDs(raw string) []uint64 {
	parts := strings.Split(raw, ",")
	result := make([]uint64, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.ParseUint(strings.TrimSpace(part), 10, 64)
		if err != nil || value == 0 {
			continue
		}
		result = append(result, value)
	}
	return result
}

func resolveDataScopeMode(policies []SystemRoleDataScope) string {
	hasSelf := false
	hasDept := false
	hasDeptAndChildren := false
	hasCustom := false
	customDeptIDs := 0

	for _, policy := range policies {
		switch strings.TrimSpace(policy.Mode) {
		case common.DataScopeModeAll:
			return common.DataScopeModeAll
		case common.DataScopeModeCustom:
			hasCustom = true
			customDeptIDs += len(parseDataScopeDeptIDs(policy.DeptIDs))
		case common.DataScopeModeDeptAndChildren:
			hasDeptAndChildren = true
		case common.DataScopeModeDept:
			hasDept = true
		case common.DataScopeModeSelf:
			hasSelf = true
		}
	}

	switch {
	case customDeptIDs > 0:
		return common.DataScopeModeCustom
	case hasDeptAndChildren:
		return common.DataScopeModeDeptAndChildren
	case hasDept:
		return common.DataScopeModeDept
	case hasSelf:
		return common.DataScopeModeSelf
	case hasCustom:
		return common.DataScopeModeCustom
	default:
		return common.DataScopeModeCustom
	}
}

func isValidDataScopePolicyMode(mode string) bool {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case common.DataScopeModeAll,
		common.DataScopeModeSelf,
		common.DataScopeModeDept,
		common.DataScopeModeDeptAndChildren,
		common.DataScopeModeCustom:
		return true
	default:
		return false
	}
}

func resolveCustomDataScopeDeptIDs(policies []SystemRoleDataScope) []uint64 {
	seen := make(map[uint64]struct{})
	result := make([]uint64, 0)
	for _, policy := range policies {
		if strings.TrimSpace(policy.Mode) != common.DataScopeModeCustom {
			continue
		}
		for _, deptID := range parseDataScopeDeptIDs(policy.DeptIDs) {
			if _, ok := seen[deptID]; ok {
				continue
			}
			seen[deptID] = struct{}{}
			result = append(result, deptID)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

// hasAdministrativeRole reports whether the subject is a full administrator of
// its own scope, in which case the row-level data-scope policy does not further
// restrict it.
//
// Two role keys qualify:
//
//   - the platform `admin` key — unrestricted across the platform;
//   - the tenant administrator key (`tenant_admin`) — the product model is
//     "the full console, automatically confined to the caller's tenant", and
//     that confinement comes from the tenant scope, not from this policy. A
//     tenant administrator with no system_role_data_scope row would otherwise
//     fall through to custom + empty dept set and see nothing at all.
//
// The tenant key is checked against the caller's own role list, so a tenant
// administrator can only ever relax its OWN request; confinement to the tenant
// is applied separately and unconditionally by the tenant scope.
func hasAdministrativeRole(roleKeys []string) bool {
	const adminRoleKey = "admin"
	for _, roleKey := range roleKeys {
		trimmed := strings.TrimSpace(roleKey)
		if trimmed == adminRoleKey || trimmed == tenant.TenantRoleAdminKey {
			return true
		}
	}
	return false
}

// hasTenantLocalRole reports whether the subject holds any tenant-local role
// (tenant_admin, tenant_member, ...).
//
// Tenant-local roles derive their row confinement from the tenant scope, which
// is applied unconditionally by the data-access layer, and split read from write
// through Casbin. The platform's dept-based data-scope policy is a
// tenant-agnostic mechanism (contract §7 keeps the role namespaces separate) and
// must therefore not govern them: a tenant member has no
// system_role_data_scope row, so without this exemption it would fall through
// the fail-closed lookup and every request it makes would 500. A tenant admin is
// already covered by hasAdministrativeRole; routing both through this predicate
// keeps a single, list-driven definition of "tenant scoped".
func hasTenantLocalRole(roleKeys []string) bool {
	for _, roleKey := range roleKeys {
		if tenant.IsTenantScopedRoleKey(roleKey) {
			return true
		}
	}
	return false
}
