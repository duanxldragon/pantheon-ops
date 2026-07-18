package bizscope

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/testmysql"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type bizScopeTestHost struct {
	ID                uint64         `gorm:"primaryKey;autoIncrement"`
	Hostname          string         `gorm:"size:128;not null"`
	IP                string         `gorm:"size:45;not null"`
	OS                string         `gorm:"size:32;not null"`
	Status            string         `gorm:"size:32;default:pending"`
	BusinessScopeID   uint64         `gorm:"column:business_scope_id"`
	BusinessScopeCode string         `gorm:"column:business_scope_code"`
	BusinessScopeName string         `gorm:"column:business_scope_name"`
	DeptID            uint64         `gorm:"column:dept_id"`
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

func (bizScopeTestHost) TableName() string { return "biz_cmdb_host" }

func setupBizScopeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)
	svc := NewService(db)
	if err := svc.Migrate(); err != nil {
		t.Fatalf("migrate bizscope tables: %v", err)
	}
	if err := db.AutoMigrate(&bizScopeTestHost{}); err != nil {
		t.Fatalf("migrate host fixtures: %v", err)
	}
	return db
}

func TestBizScopeDetailHostCountRespectsDataScope(t *testing.T) {
	db := setupBizScopeTestDB(t)
	svc := NewService(db)

	scope := BizScope{Code: "his-dev", Name: "HIS 开发", Environment: "dev", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	if err := db.Create(&[]bizScopeTestHost{
		{Hostname: "host-a", IP: "10.0.0.1", OS: "linux", Status: "assigned", BusinessScopeID: scope.ID, BusinessScopeCode: scope.Code, BusinessScopeName: scope.Name, DeptID: 10},
		{Hostname: "host-b", IP: "10.0.0.2", OS: "linux", Status: "assigned", BusinessScopeID: scope.ID, BusinessScopeCode: scope.Code, BusinessScopeName: scope.Name, DeptID: 20},
	}).Error; err != nil {
		t.Fatalf("seed hosts: %v", err)
	}

	detail, err := svc.Get(scope.ID, &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 10})
	if err != nil {
		t.Fatalf("get scoped detail: %v", err)
	}
	if detail.HostCount != 1 {
		t.Fatalf("expected scoped host count 1, got %d", detail.HostCount)
	}
}

func TestBizScopeBindHostsRespectsDataScope(t *testing.T) {
	db := setupBizScopeTestDB(t)
	svc := NewService(db)

	scope := BizScope{Code: "his-dev", Name: "HIS 开发", Environment: "dev", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := bizScopeTestHost{Hostname: "host-a", IP: "10.0.0.1", OS: "linux", Status: "pending", DeptID: 20}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}

	err := svc.BindHosts(scope.ID, []uint64{host.ID}, &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 10})
	if err == nil {
		t.Fatal("expected bind to fail for out-of-scope host")
	}

	var reloaded bizScopeTestHost
	if err := db.First(&reloaded, host.ID).Error; err != nil {
		t.Fatalf("reload host: %v", err)
	}
	if reloaded.BusinessScopeID != 0 {
		t.Fatalf("expected host binding unchanged, got scope %d", reloaded.BusinessScopeID)
	}
}

func TestResolveBizScopeErrorKey(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "code exists", err: errors.New(bizScopeCodeExistsKey), want: bizScopeCodeExistsKey},
		{name: "in use", err: errors.New(bizScopeInUseKey), want: bizScopeInUseKey},
		{name: "not found", err: errors.New(bizScopeNotFoundKey), want: bizScopeNotFoundKey},
		{name: "invalid parameter", err: errors.New("param.invalid"), want: "param.invalid"},
		{name: "database error", err: errors.New("sql: connection refused"), want: "request.failed"},
		{name: "nil error", err: nil, want: "request.failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveBizScopeErrorKey(tt.err); got != tt.want {
				t.Fatalf("resolveBizScopeErrorKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBizScopeCanonicalBusinessErrors(t *testing.T) {
	db := setupBizScopeTestDB(t)
	svc := NewService(db)

	scope := BizScope{Code: "canonical-scope", Name: "Canonical Scope", Environment: "prod", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}

	if _, err := svc.Create(&CreateBizScopeRequest{
		Code:        scope.Code,
		Name:        "Duplicate Scope",
		Environment: "prod",
		Status:      "active",
	}); err == nil || err.Error() != bizScopeCodeExistsKey {
		t.Fatalf("expected %s, got %v", bizScopeCodeExistsKey, err)
	}

	host := bizScopeTestHost{
		Hostname:          "canonical-host",
		IP:                "10.0.0.50",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeCode: scope.Code,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed bound host: %v", err)
	}
	if err := svc.Delete(scope.ID); err == nil || err.Error() != bizScopeInUseKey {
		t.Fatalf("expected %s, got %v", bizScopeInUseKey, err)
	}

	if _, err := svc.Get(scope.ID+999999, nil); err == nil || err.Error() != bizScopeNotFoundKey {
		t.Fatalf("expected %s, got %v", bizScopeNotFoundKey, err)
	}
}
func TestFailBizScopeErrorReturnsCanonicalOrGenericKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "canonical", err: errors.New(bizScopeCodeExistsKey), want: bizScopeCodeExistsKey},
		{name: "generic fallback", err: errors.New("sql: connection refused"), want: "request.failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			failBizScopeError(context, tt.err)

			var response common.Response
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.Code != common.CodeError || response.Message != tt.want {
				t.Fatalf("unexpected response: code=%d message=%q", response.Code, response.Message)
			}
		})
	}
}

func TestSeedI18nReplacesLegacyBizScopeKeys(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.Exec(`CREATE TABLE system_i18n (
		id BIGINT PRIMARY KEY AUTO_INCREMENT,
		module VARCHAR(64),
		locale VARCHAR(16),
		group_name VARCHAR(64),
		` + "`key`" + ` VARCHAR(255),
		value TEXT,
		lifecycle_status VARCHAR(16),
		created_at DATETIME,
		updated_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create i18n table: %v", err)
	}
	for _, key := range legacyBizScopeI18nKeys {
		if err := db.Exec(
			"INSERT INTO system_i18n (module, locale, group_name, `key`, value) VALUES (?, 'zh-CN', 'error', ?, 'legacy')",
			"business.bizscope",
			key,
		).Error; err != nil {
			t.Fatalf("seed legacy key %s: %v", key, err)
		}
	}

	if err := seedI18n(db); err != nil {
		t.Fatalf("seed bizscope i18n: %v", err)
	}

	var legacyCount int64
	if err := db.Table("system_i18n").Where("module = ? AND `key` IN ?", "business.bizscope", legacyBizScopeI18nKeys).Count(&legacyCount).Error; err != nil {
		t.Fatalf("count legacy keys: %v", err)
	}
	if legacyCount != 0 {
		t.Fatalf("expected legacy keys removed, got %d", legacyCount)
	}

	var canonicalCount int64
	if err := db.Table("system_i18n").Where(
		"module = ? AND `key` IN ?",
		"business.bizscope",
		[]string{bizScopeCodeExistsKey, bizScopeInUseKey, bizScopeNotFoundKey},
	).Count(&canonicalCount).Error; err != nil {
		t.Fatalf("count canonical keys: %v", err)
	}
	if canonicalCount != 6 {
		t.Fatalf("expected 6 canonical zh-CN/en-US rows, got %d", canonicalCount)
	}
}
