package bizscope

import (
	"testing"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/testmysql"

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
