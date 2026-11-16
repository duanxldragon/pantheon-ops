package filedist

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/secrets"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"gorm.io/gorm"
)

type distributionAuthorityStub struct {
	scope *common.DataScopeReq
	err   error
	calls int
}

func (a *distributionAuthorityStub) AuthorizeFileDistribution(_ context.Context, actorID, _ uint64, _ string) (*common.DataScopeReq, error) {
	a.calls++
	return a.scope, a.err
}

type distributionSelectionStub struct {
	version string
	host    cmdb.DeployHostTarget
	tx      *gorm.DB
}

func (p *distributionSelectionStub) WithTargetTransaction(tx *gorm.DB) cmdb.TargetSelectionCapability {
	p.tx = tx
	return p
}

func (p *distributionSelectionStub) ResolveTargetSelection(context.Context, cmdb.TargetSelection, *common.DataScopeReq, bool) (*cmdb.TargetSelectionResult, error) {
	return &cmdb.TargetSelectionResult{ScopeVersion: p.version, Hosts: []cmdb.SelectionHost{{ID: p.host.ID, IP: p.host.IP, StateVersion: p.host.StateVersion}}}, nil
}

func (p *distributionSelectionStub) ResolveDeployTargets(cmdb.DeployHostResolveRequest) ([]cmdb.DeployHostTarget, error) {
	return []cmdb.DeployHostTarget{p.host}, nil
}

func TestFileOwnerVisibility(t *testing.T) {
	file := FileRecord{OwnerID: 7, DeptID: 3}
	for _, tc := range []struct {
		name    string
		file    FileRecord
		scope   *common.DataScopeReq
		allowed bool
	}{
		{"nil", file, nil, false},
		{"legacy-owner", FileRecord{UploadedBy: "7", DeptID: 3}, &common.DataScopeReq{UserID: 7, IsAdmin: true}, false},
		{"self", file, &common.DataScopeReq{UserID: 7, Mode: common.DataScopeModeSelf}, true},
		{"other-self", file, &common.DataScopeReq{UserID: 8, Mode: common.DataScopeModeSelf}, false},
		{"same-department", file, &common.DataScopeReq{UserID: 8, DeptID: 3, Mode: common.DataScopeModeDept}, true},
		{"other-department", file, &common.DataScopeReq{UserID: 8, DeptID: 4, Mode: common.DataScopeModeDept}, false},
		{"custom", file, &common.DataScopeReq{UserID: 8, DeptIDs: []uint64{3}, Mode: common.DataScopeModeCustom}, true},
		{"unknown-mode", file, &common.DataScopeReq{UserID: 7, Mode: "tenant"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := fileOwnerVisible(tc.file, tc.scope); got != tc.allowed {
				t.Fatalf("visibility=%v want %v", got, tc.allowed)
			}
		})
	}
}

func TestAuthorizedDistributionPersistsCanonicalTargetsAndRechecksAuthority(t *testing.T) {
	t.Setenv(secrets.BusinessSecretKeyEnv, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&FileStorage{}, &FileRecord{}, &DistributionTask{}, &DistributionTarget{}); err != nil {
		t.Fatal(err)
	}
	file := FileRecord{Name: "owned", Path: "payload", UploadedBy: "display-name", OwnerID: 7, DeptID: 3, UploadedAt: time.Now()}
	if err := db.Create(&file).Error; err != nil {
		t.Fatal(err)
	}
	storage := NewStorageService(db)
	engine := NewTransferEngine(db, storage, 1)
	service := NewDistributionService(db, storage, engine)
	authority := &distributionAuthorityStub{scope: &common.DataScopeReq{UserID: 7, DeptID: 3, Mode: common.DataScopeModeSelf}}
	provider := &distributionSelectionStub{version: "version-1", host: cmdb.DeployHostTarget{ID: 11, IP: "127.0.0.1", SSHPort: 2222, StateVersion: 4}}
	service.SetTargetSelectionCapability(provider)
	service.SetFileDistributionAuthorizer(authority)
	scheduled := time.Now().Add(time.Hour)
	dto := DistributionTaskDTO{Name: "stable", FileID: file.ID, ScheduledAt: &scheduled, ScopeVersion: provider.version, Selection: &cmdb.TargetSelection{BusinessScopeID: 2, HostIDs: []uint64{11}}, Targets: []TargetConfig{{HostID: 11, StateVersion: 4, TargetPath: "/tmp/payload", Username: "ssh-user", HostFingerprint: "SHA256:fixture", Password: "fixture-secret"}}}
	created, err := service.CreateAuthorizedTask(context.Background(), dto, 7, "display-name")
	if err != nil {
		t.Fatal(err)
	}
	var task DistributionTask
	var target DistributionTarget
	if err := db.First(&task, created.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("task_id = ?", created.ID).First(&target).Error; err != nil {
		t.Fatal(err)
	}
	if task.OwnerID != 7 || task.DeptID != 3 || target.HostID != 11 || target.Host != provider.host.IP || target.Port != 2222 || target.StateVersion != 4 || provider.tx == nil {
		t.Fatal("trusted identity/endpoint facts were not persisted")
	}
	if strings.Contains(string(task.Targets), "fixture-secret") || dto.Targets[0].Host != "" {
		t.Fatal("snapshot leaked credentials or request input was mutated")
	}
	if err := service.validateTaskAction(task.ID, "execute"); err != nil {
		t.Fatal(err)
	}
	authority.err = errors.New("revoked-private-diagnostic")
	if err := service.StartTask(task.ID); !errors.Is(err, common.ErrForbidden) {
		t.Fatalf("revoked start allowed: %v", err)
	}
	if err := db.First(&task, task.ID).Error; err != nil || task.Status != statusPending || task.StartedAt != nil {
		t.Fatal("denied start dispatched or mutated task")
	}
	if len(engine.taskQueue) != 0 {
		t.Fatal("denied start queued an effect")
	}
	authority.err = nil
	provider.host.IP = "127.0.0.2"
	if err := service.validateTaskAction(task.ID, "execute"); !errors.Is(err, cmdb.ErrTargetSelectionConflict) {
		t.Fatalf("endpoint drift allowed: %v", err)
	}
	provider.host.IP = "127.0.0.1"
	if err := db.Model(&file).Update("owner_id", 8).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.validateTaskAction(task.ID, "execute"); !errors.Is(err, common.ErrForbidden) {
		t.Fatalf("source ownership drift allowed: %v", err)
	}
}

func TestAuthorizedDistributionRejectsForgedAndLegacyFactsWithoutRows(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&FileRecord{}, &DistributionTask{}, &DistributionTarget{}); err != nil {
		t.Fatal(err)
	}
	file := FileRecord{Name: "legacy", UploadedBy: "7", UploadedAt: time.Now()}
	if err := db.Create(&file).Error; err != nil {
		t.Fatal(err)
	}
	s := NewDistributionService(db, NewStorageService(db), nil)
	a := &distributionAuthorityStub{scope: &common.DataScopeReq{UserID: 7, IsAdmin: true}}
	p := &distributionSelectionStub{version: "v1", host: cmdb.DeployHostTarget{ID: 11, IP: "127.0.0.1", StateVersion: 1}}
	s.SetTargetSelectionCapability(p)
	s.SetFileDistributionAuthorizer(a)
	dto := DistributionTaskDTO{Name: "forged", FileID: file.ID, Selection: &cmdb.TargetSelection{BusinessScopeID: 2}, ScopeVersion: "v1", Targets: []TargetConfig{{HostID: 11, StateVersion: 1, Host: "attacker.example"}}}
	if _, err := s.CreateAuthorizedTask(context.Background(), dto, 7, "7"); !errors.Is(err, common.ErrForbidden) {
		t.Fatalf("legacy username treated as owner: %v", err)
	}
	if err := db.Model(&file).Update("owner_id", 7).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateAuthorizedTask(context.Background(), dto, 7, "7"); !errors.Is(err, cmdb.ErrTargetSelectionConflict) {
		t.Fatalf("client endpoint override allowed: %v", err)
	}
	dto.Targets[0].Host = ""
	a.scope.UserID = 8
	if _, err := s.CreateAuthorizedTask(context.Background(), dto, 7, "7"); !errors.Is(err, common.ErrForbidden) {
		t.Fatalf("actor mismatch allowed: %v", err)
	}
	var count int64
	if err := db.Model(&DistributionTask{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("rejection persisted task")
	}
	if err := db.Model(&DistributionTarget{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("rejection persisted target")
	}
}

func TestFileDistOwnershipMigrationPreservesLegacyAndReplays(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&FileRecord{}, &DistributionTask{}, &DistributionTarget{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("ALTER TABLE filedist_file DROP COLUMN owner_id, DROP COLUMN dept_id").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO filedist_file (name, storage_id, path, size, uploaded_by, uploaded_at) VALUES ('legacy', 1, 'legacy', 1, '7', NOW())").Error; err != nil {
		t.Fatal(err)
	}
	rawDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	conn, err := rawDB.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	for _, path := range []string{"migrations/000039_filedist_ownership_selection.up.sql", "migrations/000039_filedist_ownership_selection.up.sql", "migrations/000039_filedist_ownership_selection.down.sql"} {
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, stmt := range strings.Split(string(bytes), ";") {
			if strings.TrimSpace(stmt) == "" {
				continue
			}
			if _, err := conn.ExecContext(context.Background(), stmt); err != nil {
				t.Fatalf("migration %s: %v", path, err)
			}
		}
	}
	var file FileRecord
	if err := db.First(&file).Error; err != nil || file.OwnerID != 0 || file.UploadedBy != "7" {
		t.Fatal("migration granted legacy identity or lost row")
	}
}

func TestQueuedDistributionRevocationPrecedesFileAccess(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&DistributionTarget{}); err != nil {
		t.Fatal(err)
	}
	target := DistributionTarget{TaskID: 1, Host: "127.0.0.1", TargetPath: "/tmp/payload", Status: statusPending}
	if err := db.Create(&target).Error; err != nil {
		t.Fatal(err)
	}
	engine := NewTransferEngine(db, nil, 1)
	engine.processTask(context.Background(), &TransferTask{ID: target.ID, Authorize: func(context.Context) error { return errors.New("secret-diagnostic") }})
	if err := db.First(&target, target.ID).Error; err != nil || target.Status != statusFailed || target.Error != "permission.denied" {
		t.Fatal("revoked attempt reached file I/O or persisted raw error")
	}
}
