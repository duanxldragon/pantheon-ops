package label

import (
	"errors"
	"testing"

	cmdbgroup "pantheon-ops/backend/modules/business/cmdb/group"
	cmdbhost "pantheon-ops/backend/modules/business/cmdb/host"
	"pantheon-ops/backend/pkg/testmysql"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&LabelSchema{}, &cmdbhost.Host{}, &cmdbgroup.Group{}); err != nil {
		t.Fatalf("migrate cmdb label schema: %v", err)
	}
	return db
}

func TestCreateLabelSchema(t *testing.T) {
	db := setupTestDB(t)
	svc := NewLabelService(db)

	resp, err := svc.Create(CreateLabelSchemaRequest{
		Key:         "env",
		Name:        "环境",
		ValueMode:   "enum",
		DictCode:    "cmdb_env",
		Required:    true,
		Description: "部署环境",
	})
	if err != nil {
		t.Fatalf("create label schema: %v", err)
	}
	if resp.Key != "env" || resp.ValueMode != "enum" || resp.DictCode != "cmdb_env" || !resp.Required {
		t.Fatalf("unexpected label schema response: %+v", resp)
	}
}

func TestLabelSchemaOptions(t *testing.T) {
	db := setupTestDB(t)
	svc := NewLabelService(db)

	resp, err := svc.Create(CreateLabelSchemaRequest{
		Key:       "region",
		Name:      "区域",
		ValueMode: "enum",
		Options:   []string{"西安开发环境", "西安测试环境", "西安开发环境", " "},
	})
	if err != nil {
		t.Fatalf("create label schema: %v", err)
	}
	if len(resp.Options) != 2 || resp.Options[0] != "西安开发环境" || resp.Options[1] != "西安测试环境" {
		t.Fatalf("unexpected created options: %+v", resp.Options)
	}

	nextOptions := []string{"Prometheus", "Node Exporter"}
	updated, err := svc.Update(resp.ID, UpdateLabelSchemaRequest{Options: &nextOptions})
	if err != nil {
		t.Fatalf("update label schema options: %v", err)
	}
	if len(updated.Options) != 2 || updated.Options[0] != "Prometheus" || updated.Options[1] != "Node Exporter" {
		t.Fatalf("unexpected updated options: %+v", updated.Options)
	}
}

func TestCreateLabelSchemaRejectsDuplicateKey(t *testing.T) {
	db := setupTestDB(t)
	svc := NewLabelService(db)

	req := CreateLabelSchemaRequest{Key: "biz", Name: "业务系统", ValueMode: "free"}
	if _, err := svc.Create(req); err != nil {
		t.Fatalf("create label schema: %v", err)
	}
	_, err := svc.Create(req)
	if !errors.Is(err, ErrLabelSchemaKeyExists) {
		t.Fatalf("expected ErrLabelSchemaKeyExists, got %v", err)
	}
}

func TestDeleteLabelSchemaRejectsReferencedHostLabel(t *testing.T) {
	db := setupTestDB(t)
	svc := NewLabelService(db)
	created, err := svc.Create(CreateLabelSchemaRequest{Key: "env", Name: "环境", ValueMode: "free"})
	if err != nil {
		t.Fatalf("create label schema: %v", err)
	}
	if err := db.Create(&cmdbhost.Host{
		Hostname:    "host-1",
		IP:          "10.0.0.1",
		OS:          "linux",
		LabelValues: datatypes.JSON([]byte(`[{"key":"env","val":"prod"}]`)),
		Status:      "pending",
	}).Error; err != nil {
		t.Fatalf("create host: %v", err)
	}

	err = svc.Delete(created.ID)
	if !errors.Is(err, ErrLabelSchemaInUse) {
		t.Fatalf("expected ErrLabelSchemaInUse, got %v", err)
	}
}

func TestDeleteLabelSchemaRejectsReferencedGroupCondition(t *testing.T) {
	db := setupTestDB(t)
	svc := NewLabelService(db)
	created, err := svc.Create(CreateLabelSchemaRequest{Key: "biz", Name: "业务系统", ValueMode: "free"})
	if err != nil {
		t.Fatalf("create label schema: %v", err)
	}
	if err := db.Create(&cmdbgroup.Group{
		Name:       "订单系统",
		Conditions: datatypes.JSON([]byte(`{"operator":"AND","rules":[{"key":"biz","op":"eq","val":"order"}]}`)),
	}).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}

	err = svc.Delete(created.ID)
	if !errors.Is(err, ErrLabelSchemaInUse) {
		t.Fatalf("expected ErrLabelSchemaInUse, got %v", err)
	}
}
