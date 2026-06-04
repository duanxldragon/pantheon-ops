package database

import (
	"testing"

	"gorm.io/gorm"
	"pantheon-platform/backend/pkg/common"
	"pantheon-platform/backend/pkg/testmysql"
)

type dataScopeTestRow struct {
	ID     uint64 `gorm:"primaryKey"`
	DeptID uint64
}

func (dataScopeTestRow) TableName() string {
	return "data_scope_test_rows"
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
