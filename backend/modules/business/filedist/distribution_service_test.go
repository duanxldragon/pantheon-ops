package filedist

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/secrets"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

func TestCreateTaskPreservesExplicitZeroRetries(t *testing.T) {
	t.Setenv(secrets.BusinessSecretKeyEnv, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&FileStorage{}, &FileRecord{}, &DistributionTask{}, &DistributionTarget{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	storage := FileStorage{Name: "test", Type: "local", BasePath: t.TempDir(), Status: "active"}
	if err := db.Create(&storage).Error; err != nil {
		t.Fatalf("create storage: %v", err)
	}
	file := FileRecord{StorageID: storage.ID, Name: "payload.txt", Path: "payload.txt", Size: 1, MD5: "x", UploadedAt: time.Now()}
	if err := db.Create(&file).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}

	storageService := NewStorageService(db)
	engine := NewTransferEngine(db, storageService, 1)
	service := NewDistributionService(db, storageService, engine)
	scheduledAt := time.Now().Add(time.Hour)
	dto := DistributionTaskDTO{
		Name:        "no-retry",
		FileID:      file.ID,
		ScheduledAt: &scheduledAt,
		Targets: []TargetConfig{{
			Host: "127.0.0.1", Port: 22, TargetPath: "/tmp/payload.txt", Username: "ops",
			HostFingerprint: "SHA256:test", MaxRetries: 0,
			Password: "fixture-password-never-in-snapshot", PrivateKey: "fixture-key-never-in-snapshot",
		}},
	}
	created, err := service.CreateTask(dto, "tester")
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	var target DistributionTarget
	if err := db.Where("task_id = ?", created.ID).First(&target).Error; err != nil {
		t.Fatalf("load target: %v", err)
	}
	if target.MaxRetries != 0 {
		t.Fatalf("expected zero retries, got %d", target.MaxRetries)
	}
	var task DistributionTask
	if err := db.First(&task, created.ID).Error; err != nil {
		t.Fatalf("load snapshot: %v", err)
	}
	if strings.Contains(string(task.Targets), dto.Targets[0].Password) || strings.Contains(string(task.Targets), dto.Targets[0].PrivateKey) {
		t.Fatal("task snapshot contains plaintext credentials")
	}
	var snapshot []map[string]interface{}
	if err := json.Unmarshal(task.Targets, &snapshot); err != nil || len(snapshot) != 1 {
		t.Fatalf("invalid target snapshot: %v", err)
	}
	for _, key := range []string{"password", "privateKey"} {
		if _, ok := snapshot[0][key]; ok {
			t.Fatalf("credential field persisted in snapshot: %s", key)
		}
	}
	if snapshot[0]["host"] != dto.Targets[0].Host || snapshot[0]["targetPath"] != dto.Targets[0].TargetPath {
		t.Fatal("nonsecret target facts changed")
	}
	for _, pair := range [][2]string{{target.Password, dto.Targets[0].Password}, {target.PrivateKey, dto.Targets[0].PrivateKey}} {
		plain, err := secrets.Decrypt(pair[0])
		if err != nil || plain != pair[1] || pair[0] == pair[1] {
			t.Fatal("encrypted execution credential was lost or stored in plaintext")
		}
	}
	if dto.Targets[0].Password != "fixture-password-never-in-snapshot" || dto.Targets[0].PrivateKey != "fixture-key-never-in-snapshot" {
		t.Fatal("caller-owned request was mutated")
	}
}

func TestTransferErrorWithoutRetriesFinishesTarget(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&DistributionTarget{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	target := DistributionTarget{TaskID: 1, Host: "127.0.0.1", TargetPath: "/tmp/payload.txt", Status: "transferring", MaxRetries: 0}
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("create target: %v", err)
	}

	engine := NewTransferEngine(db, nil, 1)
	engine.handleTransferError(&TransferTask{ID: target.ID, MaxRetries: 0}, "connect_failed")

	var updated DistributionTarget
	if err := db.First(&updated, target.ID).Error; err != nil {
		t.Fatalf("load target: %v", err)
	}
	if updated.Status != "failed" || updated.RetryCount != 0 || updated.FinishedAt == nil {
		t.Fatalf("unexpected terminal target: %+v", updated)
	}
}
