package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"github.com/gin-gonic/gin"
)

func TestDataScopeMiddlewareInjectsRoleDeptScope(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.Exec("CREATE TABLE system_user (id BIGINT PRIMARY KEY, dept_id BIGINT)").Error; err != nil {
		t.Fatalf("create user table: %v", err)
	}
	if err := MigrateDataScopePolicy(db); err != nil {
		t.Fatalf("migrate data scope policy: %v", err)
	}
	if err := db.Exec("INSERT INTO system_user (id, dept_id) VALUES (7, 42)").Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&SystemRoleDataScope{RoleKey: "dept_role", Mode: common.DataScopeModeDept}).Error; err != nil {
		t.Fatalf("seed role data scope: %v", err)
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("userId", uint64(7))
		c.Set("roleKeys", []string{"dept_role"})
		c.Next()
	})
	engine.Use(DataScopeMiddleware(db))
	engine.GET("/scoped", func(c *gin.Context) {
		scope := common.GetDataScope(c)
		if scope == nil {
			t.Fatalf("expected data scope")
		}
		if scope.UserID != 7 || scope.DeptID != 42 || scope.Mode != common.DataScopeModeDept {
			t.Fatalf("unexpected data scope: %+v", scope)
		}
		common.Success(c, gin.H{"mode": scope.Mode})
	})

	req := httptest.NewRequest(http.MethodGet, "/scoped", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected request to pass, got %d", recorder.Code)
	}
}

// TestDataScopeCacheKeysAreScopedPerConnection 守护曾经的主干偶发失败：
// 数据范围缓存过去以连接池指针地址作命名空间，连接池关闭回收后 Go 可能把同一地址
// 分配给新的连接池，使新测试库继承旧库的缓存结论（策略表「已存在」），
// MigrateDataScopePolicy 因此跳过建表，后续 INSERT 报 Error 1146。
func TestDataScopeCacheKeysAreScopedPerConnection(t *testing.T) {
	tableName := (&SystemRoleDataScope{}).TableName()
	first := testmysql.Open(t)
	firstTableKey := buildTableExistCacheKey(first, tableName)
	firstUserKey := buildUserDeptCacheKey(first, 7)

	second := testmysql.Open(t)

	if firstTableKey == buildTableExistCacheKey(second, tableName) {
		t.Fatal("distinct connections must not share a table-existence cache key")
	}
	if firstUserKey == buildUserDeptCacheKey(second, 7) {
		t.Fatal("distinct connections must not share a user-dept cache key")
	}
	// Re-deriving the key for the same connection must not drift, otherwise the
	// cache would miss forever and the early-exit it exists for would be lost.
	if firstTableKey != buildTableExistCacheKey(first, tableName) {
		t.Fatal("cache key must be stable for the same connection")
	}
}

// TestMigrateDataScopePolicyIgnoresForeignCachedVerdict 复现 CI 上的失败序列：
// fail-closed 套件先在旧连接上建了结构损坏的同名策略表并把「表已存在」写进缓存，
// 随后的新连接仍必须自建自己的策略表。
func TestMigrateDataScopePolicyIgnoresForeignCachedVerdict(t *testing.T) {
	poisoned := testmysql.Open(t)
	if err := poisoned.Exec("CREATE TABLE system_role_data_scope (unrelated_col BIGINT)").Error; err != nil {
		t.Fatalf("create broken policy table: %v", err)
	}
	storeCachedTableExistence(poisoned, (&SystemRoleDataScope{}).TableName(), true)

	fresh := testmysql.Open(t)
	if err := MigrateDataScopePolicy(fresh); err != nil {
		t.Fatalf("migrate data scope policy: %v", err)
	}
	if err := fresh.Create(&SystemRoleDataScope{RoleKey: "dept_role", Mode: common.DataScopeModeDept}).Error; err != nil {
		t.Fatalf("fresh connection must create its own policy table: %v", err)
	}
}

func TestDataScopeMiddlewareMergesMultipleRolePoliciesDeterministically(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.Exec("CREATE TABLE system_user (id BIGINT PRIMARY KEY, dept_id BIGINT)").Error; err != nil {
		t.Fatalf("create user table: %v", err)
	}
	if err := MigrateDataScopePolicy(db); err != nil {
		t.Fatalf("migrate data scope policy: %v", err)
	}
	if err := db.Exec("INSERT INTO system_user (id, dept_id) VALUES (7, 42)").Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&SystemRoleDataScope{RoleKey: "dept_role", Mode: common.DataScopeModeDept}).Error; err != nil {
		t.Fatalf("seed dept role: %v", err)
	}
	if err := db.Create(&SystemRoleDataScope{RoleKey: "custom_a", Mode: common.DataScopeModeCustom, DeptIDs: "10"}).Error; err != nil {
		t.Fatalf("seed custom role a: %v", err)
	}
	if err := db.Create(&SystemRoleDataScope{RoleKey: "custom_b", Mode: common.DataScopeModeCustom, DeptIDs: "20"}).Error; err != nil {
		t.Fatalf("seed custom role b: %v", err)
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("userId", uint64(7))
		c.Set("roleKeys", []string{"dept_role", "custom_b", "custom_a"})
		c.Next()
	})
	engine.Use(DataScopeMiddleware(db))
	engine.GET("/scoped", func(c *gin.Context) {
		scope := common.GetDataScope(c)
		if scope == nil {
			t.Fatalf("expected data scope")
		}
		if scope.Mode != common.DataScopeModeCustom {
			t.Fatalf("expected custom scope, got %+v", scope)
		}
		if len(scope.DeptIDs) != 2 || scope.DeptIDs[0] != 10 || scope.DeptIDs[1] != 20 {
			t.Fatalf("expected merged custom dept ids [10 20], got %+v", scope.DeptIDs)
		}
		common.Success(c, gin.H{"mode": scope.Mode})
	})

	req := httptest.NewRequest(http.MethodGet, "/scoped", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected request to pass, got %d", recorder.Code)
	}
}

func TestDataScopeMiddlewareExpandsDeptAndChildrenScope(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.Exec("CREATE TABLE system_user (id BIGINT PRIMARY KEY, dept_id BIGINT)").Error; err != nil {
		t.Fatalf("create user table: %v", err)
	}
	if err := db.Exec("CREATE TABLE system_dept (id BIGINT PRIMARY KEY, parent_id BIGINT, ancestors TEXT, deleted_at DATETIME NULL)").Error; err != nil {
		t.Fatalf("create dept table: %v", err)
	}
	if err := MigrateDataScopePolicy(db); err != nil {
		t.Fatalf("migrate data scope policy: %v", err)
	}
	if err := db.Exec("INSERT INTO system_user (id, dept_id) VALUES (7, 10)").Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Exec("INSERT INTO system_dept (id, parent_id, ancestors, deleted_at) VALUES (10, 1, '1', NULL), (11, 10, '1,10', NULL), (12, 11, '1,10,11', NULL), (20, 1, '1', NULL)").Error; err != nil {
		t.Fatalf("seed depts: %v", err)
	}
	if err := db.Create(&SystemRoleDataScope{RoleKey: "dept_tree", Mode: common.DataScopeModeDeptAndChildren}).Error; err != nil {
		t.Fatalf("seed role data scope: %v", err)
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("userId", uint64(7))
		c.Set("roleKeys", []string{"dept_tree"})
		c.Next()
	})
	engine.Use(DataScopeMiddleware(db))
	engine.GET("/scoped", func(c *gin.Context) {
		scope := common.GetDataScope(c)
		if scope == nil {
			t.Fatalf("expected data scope")
		}
		if scope.Mode != common.DataScopeModeDeptAndChildren {
			t.Fatalf("expected dept_and_children scope, got %+v", scope)
		}
		if len(scope.DeptIDs) != 3 || scope.DeptIDs[0] != 10 || scope.DeptIDs[1] != 11 || scope.DeptIDs[2] != 12 {
			t.Fatalf("expected expanded dept ids [10 11 12], got %+v", scope.DeptIDs)
		}
		common.Success(c, gin.H{"mode": scope.Mode})
	})

	req := httptest.NewRequest(http.MethodGet, "/scoped", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected request to pass, got %d", recorder.Code)
	}
}
