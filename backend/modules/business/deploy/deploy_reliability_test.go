package deploy

import (
	"strings"
	"testing"
	"time"

	"pantheon-base/modules/business/cmdb"

	"gorm.io/gorm"
)

// seedDeployReliabilityFixture seeds an active scope and an assigned host and
// returns their IDs for reliability-focused tests.
func seedDeployReliabilityFixture(t *testing.T, db *gorm.DB, hostname, ip string) (uint64, uint64) {
	t.Helper()
	scope := deployTestBizScope{Code: "reliability-scope", Name: "Reliability Scope", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          hostname,
		IP:                ip,
		SSHPort:           22,
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	return scope.ID, host.ID
}

func TestDeployPackageImmutableAfterTaskReference(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scopeID, hostID := seedDeployReliabilityFixture(t, db, "immutable-host", "10.40.0.201")
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "immutable-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo v1",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	if _, err := svc.CreateTask(CreateTaskRequest{
		Name:            "immutable task",
		PackageID:       pkg.ID,
		BusinessScopeID: scopeID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{hostID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", nil); err != nil {
		t.Fatalf("create task: %v", err)
	}

	_, err = svc.UpdatePackage(pkg.ID, UpdatePackageRequest{InstallCommand: stringPtr("echo v2")}, "1")
	if err == nil || err.Error() != errDeployPackageImmutable {
		t.Fatalf("expected immutable conflict for referenced package, got %v", err)
	}

	updated, err := svc.UpdatePackage(pkg.ID, UpdatePackageRequest{Status: stringPtr(PackageStatusDisabled)}, "1")
	if err != nil {
		t.Fatalf("expected status-only update to succeed, got %v", err)
	}
	if updated.Status != PackageStatusDisabled {
		t.Fatalf("expected disabled status, got %s", updated.Status)
	}
}

func TestDeployTaskExecutionSnapshotFrozenAgainstLiveMutation(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	runner := &fakeDeploySSHRunner{stdout: "ok"}
	svc.sshRunnerFactory = func(host cmdbHostSnapshot, req StartTaskRequest) (deploySSHRunner, error) {
		return runner, nil
	}

	scopeID, hostID := seedDeployReliabilityFixture(t, db, "snapshot-host", "10.40.0.202")
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "snapshot-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo frozen",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "snapshot task",
		PackageID:       pkg.ID,
		BusinessScopeID: scopeID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{hostID},
		ExecutorType:    ExecutorTypeSSH,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := db.Model(&DeployPackage{}).Where("id = ?", pkg.ID).Update("install_command", "echo mutated").Error; err != nil {
		t.Fatalf("mutate live package: %v", err)
	}

	if _, err := svc.StartTask(task.ID, StartTaskRequest{
		SSHUser:         "root",
		SSHPassword:     "secret",
		HostFingerprint: "SHA256:test",
		AuthMode:        "password",
	}, "1", nil); err != nil {
		t.Fatalf("start task: %v", err)
	}
	if !strings.Contains(runner.script, "echo frozen") {
		t.Fatalf("expected frozen install command in script, got %q", runner.script)
	}
	if strings.Contains(runner.script, "echo mutated") {
		t.Fatalf("live mutation leaked into execution script: %q", runner.script)
	}
}

func TestDeployTaskStartIdempotentReplay(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scopeID, hostID := seedDeployReliabilityFixture(t, db, "replay-host", "10.40.0.203")
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "replay-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "replay task",
		PackageID:       pkg.ID,
		BusinessScopeID: scopeID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{hostID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	if _, err := svc.StartTask(task.ID, StartTaskRequest{IdempotencyKey: "k1"}, "1", nil); err != nil {
		t.Fatalf("start task: %v", err)
	}
	replayed, err := svc.StartTask(task.ID, StartTaskRequest{IdempotencyKey: "k1"}, "1", nil)
	if err != nil {
		t.Fatalf("replay start: %v", err)
	}
	if !replayed.StartRequestReused {
		t.Fatalf("expected reused flag on idempotent replay")
	}
	if len(replayed.Hosts) != 1 {
		t.Fatalf("expected exactly one task host after replay, got %d", len(replayed.Hosts))
	}
}

func TestDeployTaskStartDifferentKeyWhileRunningConflict(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scopeID, hostID := seedDeployReliabilityFixture(t, db, "conflict-host", "10.40.0.204")
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "conflict-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "conflict task",
		PackageID:       pkg.ID,
		BusinessScopeID: scopeID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{hostID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	if _, err := svc.StartTask(task.ID, StartTaskRequest{IdempotencyKey: "k1"}, "1", nil); err != nil {
		t.Fatalf("start task: %v", err)
	}
	_, err = svc.StartTask(task.ID, StartTaskRequest{IdempotencyKey: "k2"}, "1", nil)
	if err == nil || err.Error() != errDeployTaskAlreadyRunning {
		t.Fatalf("expected already running conflict, got %v", err)
	}
}

func TestDeployHostLeaseBlocksConcurrentStart(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scopeID, hostID := seedDeployReliabilityFixture(t, db, "lease-host", "10.40.0.205")
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "lease-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	create := func(name string) *TaskResponse {
		task, err := svc.CreateTask(CreateTaskRequest{
			Name:            name,
			PackageID:       pkg.ID,
			BusinessScopeID: scopeID,
			TargetType:      TargetTypeHost,
			TargetIDs:       []uint64{hostID},
			ExecutorType:    ExecutorTypeManual,
		}, "1", nil)
		if err != nil {
			t.Fatalf("create task: %v", err)
		}
		return task
	}

	first := create("lease task one")
	second := create("lease task two")

	if _, err := svc.StartTask(first.ID, StartTaskRequest{}, "1", nil); err != nil {
		t.Fatalf("start first task: %v", err)
	}
	_, err = svc.StartTask(second.ID, StartTaskRequest{}, "1", nil)
	if err == nil || err.Error() != errDeployTaskLeaseConflict {
		t.Fatalf("expected lease conflict for second task, got %v", err)
	}
}

func TestDeployHostLeaseTakeoverAfterExpiry(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scopeID, hostID := seedDeployReliabilityFixture(t, db, "takeover-host", "10.40.0.206")
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "takeover-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	create := func(name string) *TaskResponse {
		task, err := svc.CreateTask(CreateTaskRequest{
			Name:            name,
			PackageID:       pkg.ID,
			BusinessScopeID: scopeID,
			TargetType:      TargetTypeHost,
			TargetIDs:       []uint64{hostID},
			ExecutorType:    ExecutorTypeManual,
		}, "1", nil)
		if err != nil {
			t.Fatalf("create task: %v", err)
		}
		return task
	}

	first := create("takeover task one")
	second := create("takeover task two")

	if _, err := svc.StartTask(first.ID, StartTaskRequest{}, "1", nil); err != nil {
		t.Fatalf("start first task: %v", err)
	}
	if err := db.Model(&DeployHostLease{}).Where("host_id = ?", hostID).Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("force expire lease: %v", err)
	}
	if _, err := svc.StartTask(second.ID, StartTaskRequest{}, "1", nil); err != nil {
		t.Fatalf("expected lease takeover after expiry, got %v", err)
	}
}

func TestDeployTaskHostReportIdempotentAndStaleConflict(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scopeID, hostID := seedDeployReliabilityFixture(t, db, "report-host", "10.40.0.207")
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "report-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "report task",
		PackageID:       pkg.ID,
		BusinessScopeID: scopeID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{hostID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	started, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatalf("start task: %v", err)
	}
	host := started.Hosts[0]

	if _, err := svc.MarkHostResult(host.ID, MarkHostResultRequest{Status: TaskHostStatusSuccess, ReportKey: "r1"}, "1", nil); err != nil {
		t.Fatalf("mark host result: %v", err)
	}
	if _, err := svc.MarkHostResult(host.ID, MarkHostResultRequest{Status: TaskHostStatusSuccess, ReportKey: "r1"}, "1", nil); err != nil {
		t.Fatalf("expected idempotent duplicate report, got %v", err)
	}
	_, err = svc.MarkHostResult(host.ID, MarkHostResultRequest{Status: TaskHostStatusFailed, ReportKey: "r2", ErrorMessage: "conflict"}, "1", nil)
	if err == nil || err.Error() != errDeployTaskHostStaleReport {
		t.Fatalf("expected stale report conflict, got %v", err)
	}
}
