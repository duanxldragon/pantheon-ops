package bizscope

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	bizcap "pantheon-base/modules/business/capability"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/database"
	"pantheon-base/pkg/testmysql"

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
	svc := newBizScopeServiceWithCMDB(t, db)

	scope := BizScope{Code: "his-dev", Name: "HIS 开发", Environment: "dev", Status: "active", DeptID: 10}
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

func TestBizScopeListRespectsDataScope(t *testing.T) {
	db := setupBizScopeTestDB(t)
	svc := NewService(db)
	if err := db.Create(&[]BizScope{
		{Code: "dept-10", Name: "Dept 10", Environment: "dev", Status: "active", DeptID: 10},
		{Code: "dept-20", Name: "Dept 20", Environment: "dev", Status: "active", DeptID: 20},
	}).Error; err != nil {
		t.Fatalf("seed scopes: %v", err)
	}

	page, err := svc.List(&BizScopeListQuery{}, &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 10})
	if err != nil {
		t.Fatalf("list scoped scopes: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].DeptID != 10 {
		t.Fatalf("expected only dept 10 scope, got %+v", page)
	}
}

func TestBizScopeCreateUsesCurrentDepartment(t *testing.T) {
	db := setupBizScopeTestDB(t)
	svc := NewService(db)
	item, err := svc.Create(&CreateBizScopeRequest{Code: "dept-create", Name: "Dept Create", Environment: "dev", Status: "active"}, &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 10})
	if err != nil {
		t.Fatalf("create scoped scope: %v", err)
	}
	if item.DeptID != 10 {
		t.Fatalf("expected department 10, got %d", item.DeptID)
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
	svc := newBizScopeServiceWithCMDB(t, db)

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

// testCMDBHostReader is a hermetic CMDB host reader backed by the test-local
// bizScopeTestHost fixture table. It mirrors the dept data-scope filtering the
// real CMDB capability applies, so scoped Get/Delete flows can be exercised
// without pulling the cmdb package into this test binary.
type testCMDBHostReader struct {
	db *gorm.DB
}

func (r testCMDBHostReader) GetByIDs(ctx context.Context, req bizcap.HostIDsQuery) (bizcap.HostPage, error) {
	ids := common.NormalizeUint64IDs(req.HostIDs)
	if len(ids) == 0 {
		return bizcap.HostPage{Items: []bizcap.HostRef{}}, nil
	}
	var rows []bizScopeTestHost
	if err := r.db.WithContext(ctx).Model(&bizScopeTestHost{}).
		Scopes(database.WithDataScope(req.DataScope)).
		Where("id IN ?", ids).
		Order("id ASC").
		Find(&rows).Error; err != nil {
		return bizcap.HostPage{}, err
	}
	return bizcap.HostPage{Items: mapTestHostRefs(rows), Total: int64(len(rows))}, nil
}

func (r testCMDBHostReader) ListByBusinessScope(ctx context.Context, req bizcap.HostScopeQuery) (bizcap.HostPage, error) {
	query := r.db.WithContext(ctx).Model(&bizScopeTestHost{}).
		Scopes(database.WithDataScope(req.DataScope)).
		Where("business_scope_id = ?", req.BusinessScopeID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return bizcap.HostPage{}, err
	}
	var rows []bizScopeTestHost
	if err := query.Order("id DESC").Find(&rows).Error; err != nil {
		return bizcap.HostPage{}, err
	}
	return bizcap.HostPage{Items: mapTestHostRefs(rows), Total: total}, nil
}

func (r testCMDBHostReader) ListAvailable(ctx context.Context, req bizcap.AvailableHostQuery) (bizcap.HostPage, error) {
	query := r.db.WithContext(ctx).Model(&bizScopeTestHost{}).
		Scopes(database.WithDataScope(req.DataScope)).
		Where("(business_scope_id = 0 OR business_scope_id IS NULL)")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return bizcap.HostPage{}, err
	}
	var rows []bizScopeTestHost
	if err := query.Order("id DESC").Find(&rows).Error; err != nil {
		return bizcap.HostPage{}, err
	}
	return bizcap.HostPage{Items: mapTestHostRefs(rows), Total: total}, nil
}

func (r testCMDBHostReader) HasBusinessScopeReferences(ctx context.Context, businessScopeID uint64) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&bizScopeTestHost{}).
		Where("business_scope_id = ?", businessScopeID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

type testCMDBOwnershipCommand struct{}

func (testCMDBOwnershipCommand) Bind(context.Context, bizcap.BindOwnershipRequest) error {
	return nil
}

func (testCMDBOwnershipCommand) Unbind(context.Context, bizcap.UnbindOwnershipRequest) error {
	return nil
}

func (testCMDBOwnershipCommand) WithBusinessScopeOwnershipLock(_ context.Context, _ uint64, action func() error) error {
	return action()
}

func mapTestHostRefs(rows []bizScopeTestHost) []bizcap.HostRef {
	items := make([]bizcap.HostRef, 0, len(rows))
	for _, row := range rows {
		items = append(items, bizcap.HostRef{
			ID:                row.ID,
			Hostname:          row.Hostname,
			IP:                row.IP,
			OS:                row.OS,
			Status:            row.Status,
			BusinessScopeID:   row.BusinessScopeID,
			BusinessScopeCode: row.BusinessScopeCode,
			BusinessScopeName: row.BusinessScopeName,
			DeptID:            row.DeptID,
		})
	}
	return items
}

func newBizScopeServiceWithCMDB(t *testing.T, db *gorm.DB) *Service {
	t.Helper()
	return NewService(db, ServiceDependencies{
		HostReader:       testCMDBHostReader{db: db},
		OwnershipCommand: testCMDBOwnershipCommand{},
	})
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
