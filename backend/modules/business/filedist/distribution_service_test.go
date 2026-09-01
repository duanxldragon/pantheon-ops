package filedist

import (
	"testing"
	"time"

	"pantheon-base/pkg/testmysql"
)

func TestCreateTaskPreservesExplicitZeroRetries(t *testing.T) {
	t.Setenv("PANTHEON_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
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
	created, err := service.CreateTask(DistributionTaskDTO{
		Name:        "no-retry",
		FileID:      file.ID,
		ScheduledAt: &scheduledAt,
		Targets: []TargetConfig{{
			Host: "127.0.0.1", Port: 22, TargetPath: "/tmp/payload.txt", Username: "ops",
			HostFingerprint: "SHA256:test", MaxRetries: 0,
		}},
	}, "tester")
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
