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

type SystemRoleDataScope struct {
	ID      uint64 `gorm:"primaryKey;autoIncrement"`
	RoleKey string `gorm:"size:64;not null;uniqueIndex"`
	Mode    string `gorm:"size:32;not null;default:'all'"`
	DeptIDs string `gorm:"type:text"`
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
	role_key VARCHAR(64) NOT NULL,
	mode VARCHAR(32) NOT NULL DEFAULT 'all',
	dept_ids TEXT NULL,
	UNIQUE KEY idx_system_role_data_scope_role_key (role_key)
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
			IsAdmin:  hasAdminRole(roleKeys),
		}

		if db != nil && userID > 0 {
			scope.DeptID = loadCurrentUserDeptID(db, userID)
		}
		// 策略读取失败时 fail-closed：非管理员必须拒绝，绝不能按 all 放行全量数据。
		if db != nil && !scope.IsAdmin {
			if applyRoleDataScopePolicy(db, scope) {
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

// applyRoleDataScopePolicy 加载角色数据范围策略并写入 scope。
// 返回 true 表示策略读取失败（fail-closed），调用方必须拒绝本次请求。
func applyRoleDataScopePolicy(db *gorm.DB, scope *common.DataScopeReq) bool {
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

	policies, ok := loadRoleDataScopePolicies(db, scope.RoleKeys)
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

func loadRoleDataScopePolicies(db *gorm.DB, roleKeys []string) ([]SystemRoleDataScope, bool) {
	var policies []SystemRoleDataScope
	if err := db.Where("role_key IN ?", roleKeys).Find(&policies).Error; err != nil {
		slog.Warn("data scope: failed to load role policies", "roles", strings.Join(roleKeys, ","), "error", err)
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

func hasAdminRole(roleKeys []string) bool {
	const adminRoleKey = "admin"
	for _, roleKey := range roleKeys {
		if strings.TrimSpace(roleKey) == adminRoleKey {
			return true
		}
	}
	return false
}
