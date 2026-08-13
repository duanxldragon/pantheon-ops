package database

import (
	"testing"

	"gorm.io/gorm"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/testmysql"
)

type dataScopeTestRow struct {
	ID     uint64 `gorm:"primaryKey"`
	DeptID uint64
}

func (dataScopeTestRow) TableName() string {
	return "data_scope_test_rows"
}

type dataScopeCreatedByRow struct {
	ID        uint64 `gorm:"primaryKey"`
	CreatedBy uint64
}

func (dataScopeCreatedByRow) TableName() string {
	return "data_scope_created_by_rows"
}

type dataScopeCreateByRow struct {
	ID       uint64 `gorm:"primaryKey"`
	CreateBy uint64 `gorm:"column:create_by"`
}

func (dataScopeCreateByRow) TableName() string {
	return "data_scope_create_by_rows"
}

func TestWithDataScopeDeptModeWithoutDeptReturnsEmpty(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&dataScopeTestRow{}); err != nil {
		t.Fatalf("migrate data scope rows: %v", err)
	}
	if err := db.Create(&dataScopeTestRow{ID: 1, DeptID: 10}).Error; err != nil {
		t.Fatalf("seed row: %v", err)
	}

	var rows []dataScopeTestRow
	err := db.Model(&dataScopeTestRow{}).
		Scopes(WithDataScope(&common.DataScopeReq{Mode: common.DataScopeModeDept})).
		Find(&rows).Error
	if err != nil {
		t.Fatalf("query scoped rows: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected empty result without dept id, got %+v", rows)
	}
}

func TestWithDataScopeDefaultModeKeepsExistingBehavior(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&dataScopeTestRow{}); err != nil {
		t.Fatalf("migrate data scope rows: %v", err)
	}
	if err := db.Create(&dataScopeTestRow{ID: 1, DeptID: 0}).Error; err != nil {
		t.Fatalf("seed row: %v", err)
	}

	var rows []dataScopeTestRow
	err := db.Session(&gorm.Session{}).
		Model(&dataScopeTestRow{}).
		Scopes(WithDataScope(&common.DataScopeReq{})).
		Find(&rows).Error
	if err != nil {
		t.Fatalf("query default scoped rows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected default scope to keep current behavior, got %+v", rows)
	}
}

func TestWithDataScopeDeptAndChildrenUsesExpandedDeptIDs(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&dataScopeTestRow{}); err != nil {
		t.Fatalf("migrate data scope rows: %v", err)
	}
	if err := db.Create(&[]dataScopeTestRow{
		{ID: 1, DeptID: 10},
		{ID: 2, DeptID: 11},
		{ID: 3, DeptID: 20},
	}).Error; err != nil {
		t.Fatalf("seed rows: %v", err)
	}

	var rows []dataScopeTestRow
	err := db.Model(&dataScopeTestRow{}).
		Scopes(WithDataScope(&common.DataScopeReq{
			Mode:    common.DataScopeModeDeptAndChildren,
			DeptID:  10,
			DeptIDs: []uint64{10, 11},
		})).
		Order("id asc").
		Find(&rows).Error
	if err != nil {
		t.Fatalf("query scoped rows: %v", err)
	}
	if len(rows) != 2 || rows[0].ID != 1 || rows[1].ID != 2 {
		t.Fatalf("expected dept and child rows, got %+v", rows)
	}
}

func TestWithDataScopeSelfUsesCreatedByColumn(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&dataScopeCreatedByRow{}); err != nil {
		t.Fatalf("migrate created_by rows: %v", err)
	}
	if err := db.Create(&[]dataScopeCreatedByRow{
		{ID: 1, CreatedBy: 7},
		{ID: 2, CreatedBy: 42},
	}).Error; err != nil {
		t.Fatalf("seed created_by rows: %v", err)
	}

	var rows []dataScopeCreatedByRow
	err := db.Model(&dataScopeCreatedByRow{}).
		Scopes(WithDataScope(&common.DataScopeReq{
			Mode:   common.DataScopeModeSelf,
			UserID: 42,
		})).
		Find(&rows).Error
	if err != nil {
		t.Fatalf("query created_by scoped rows: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != 2 {
		t.Fatalf("expected created_by row 2, got %+v", rows)
	}
}

func TestWithDataScopeSelfUsesCreateByColumn(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&dataScopeCreateByRow{}); err != nil {
		t.Fatalf("migrate create_by rows: %v", err)
	}
	if err := db.Create(&[]dataScopeCreateByRow{
		{ID: 1, CreateBy: 7},
		{ID: 2, CreateBy: 42},
	}).Error; err != nil {
		t.Fatalf("seed create_by rows: %v", err)
	}

	var rows []dataScopeCreateByRow
	err := db.Model(&dataScopeCreateByRow{}).
		Scopes(WithDataScope(&common.DataScopeReq{
			Mode:   common.DataScopeModeSelf,
			UserID: 42,
		})).
		Find(&rows).Error
	if err != nil {
		t.Fatalf("query create_by scoped rows: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != 2 {
		t.Fatalf("expected create_by row 2, got %+v", rows)
	}
}

func TestWithDataScopeSelfFallsBackToIDWhenOwnerColumnIsAbsent(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&dataScopeTestRow{}); err != nil {
		t.Fatalf("migrate data scope rows: %v", err)
	}
	if err := db.Create(&[]dataScopeTestRow{
		{ID: 1, DeptID: 10},
		{ID: 2, DeptID: 20},
	}).Error; err != nil {
		t.Fatalf("seed data scope rows: %v", err)
	}

	var rows []dataScopeTestRow
	err := db.Model(&dataScopeTestRow{}).
		Scopes(WithDataScope(&common.DataScopeReq{
			Mode:   common.DataScopeModeSelf,
			UserID: 2,
		})).
		Find(&rows).Error
	if err != nil {
		t.Fatalf("query id fallback scoped rows: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != 2 {
		t.Fatalf("expected id fallback row 2, got %+v", rows)
	}
}
