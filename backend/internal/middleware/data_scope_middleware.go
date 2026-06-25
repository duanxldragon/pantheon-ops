package middleware

import (
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// --- DataScope caches ---
var (
	// User dept cache: userID -> deptID
	userDeptCacheMu sync.RWMutex
	userDeptCache   = make(map[string]userDeptEntry)
	userDeptTTL     = 5 * time.Minute

	// Role policy cache: roleKey hash -> policies
	rolePolicyCacheMu sync.RWMutex
	rolePolicyCache   = make(map[string]rolePolicyEntry)
	rolePolicyTTL     = 5 * time.Minute

	// Table existence cache (avoid DDL metadata queries per request)
	tableExistCacheMu sync.RWMutex
	tableExistCache   = make(map[string]bool)
)

type userDeptEntry struct {
	deptID   uint64
	cachedAt time.Time
}

type rolePolicyEntry struct {
	policies []SystemRoleDataScope
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
		return nil
	}
	if cachedHasTable(db, &SystemRoleDataScope{}) {
		return nil
	}
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
	return nil
}

func DataScopeMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := common.GetUserID(c)
		roleKeys := readRoleKeysFromContext(c)
		scope := &common.DataScopeReq{
			UserID:   userID,
			RoleKeys: roleKeys,
			Mode:     common.DataScopeModeAll,
			IsAdmin:  hasAdminRole(roleKeys),
		}

		if db != nil && userID > 0 {
			scope.DeptID = loadCurrentUserDeptID(db, userID)
		}
		if db != nil && !scope.IsAdmin {
			applyRoleDataScopePolicy(db, scope)
		}

		c.Set(common.DataScopeContextKey, scope)
		c.Next()
	}
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

func applyRoleDataScopePolicy(db *gorm.DB, scope *common.DataScopeReq) {
	if scope == nil || len(scope.RoleKeys) == 0 || !cachedHasTable(db, &SystemRoleDataScope{}) {
		return
	}

	policies, ok := loadRoleDataScopePolicies(db, scope.RoleKeys)
	if !ok {
		return
	}

	if len(policies) == 0 {
		return
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
}

func loadRoleDataScopePolicies(db *gorm.DB, roleKeys []string) ([]SystemRoleDataScope, bool) {
	cacheKey := buildRolePolicyCacheKey(db, roleKeys)
	rolePolicyCacheMu.RLock()
	if entry, ok := rolePolicyCache[cacheKey]; ok && time.Since(entry.cachedAt) < rolePolicyTTL {
		rolePolicyCacheMu.RUnlock()
		return entry.policies, true
	}
	rolePolicyCacheMu.RUnlock()

	var policies []SystemRoleDataScope
	if err := db.Where("role_key IN ?", roleKeys).Find(&policies).Error; err != nil {
		slog.Warn("data scope: failed to load role policies", "roles", strings.Join(roleKeys, ","), "error", err)
		return nil, false
	}

	rolePolicyCacheMu.Lock()
	rolePolicyCache[cacheKey] = rolePolicyEntry{policies: policies, cachedAt: time.Now()}
	if len(rolePolicyCache) > 1000 {
		now := time.Now()
		for k, v := range rolePolicyCache {
			if now.Sub(v.cachedAt) > rolePolicyTTL {
				delete(rolePolicyCache, k)
			}
		}
	}
	rolePolicyCacheMu.Unlock()

	return policies, true
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

func buildRolePolicyCacheKey(db *gorm.DB, roleKeys []string) string {
	return buildDatabaseCacheNamespace(db) + ":roles:" + strings.Join(roleKeys, ",")
}

func buildTableExistCacheKey(db *gorm.DB, tableName string) string {
	return buildDatabaseCacheNamespace(db) + ":table:" + tableName
}

func buildDatabaseCacheNamespace(db *gorm.DB) string {
	if db == nil {
		return "nil"
	}
	sqlDB, err := db.DB()
	if err == nil && sqlDB != nil {
		return fmt.Sprintf("%p", sqlDB)
	}
	return fmt.Sprintf("%p", db)
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
	customDeptIDs := 0

	for _, policy := range policies {
		switch strings.TrimSpace(policy.Mode) {
		case "", common.DataScopeModeAll:
			return common.DataScopeModeAll
		case common.DataScopeModeCustom:
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
	default:
		return common.DataScopeModeAll
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
	for _, roleKey := range roleKeys {
		if strings.TrimSpace(roleKey) == "admin" {
			return true
		}
	}
	return false
}
