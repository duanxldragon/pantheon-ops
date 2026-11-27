package deploy

import (
	"context"
	_ "embed"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

//go:embed migrations/000042_deploy_credential_scope.up.sql
var credentialScopeUp string

const deployCredentialScopeTestKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// seedCredentialScopes creates two active business scopes owned by different
// departments, plus a host inside the first one.
func seedCredentialScopes(t *testing.T, svc *DeployService, db *gorm.DB) (scopeA, scopeB, hostID uint64) {
	t.Helper()
	a := deployTestBizScope{Code: "cred-scope-a", Name: "Credential Scope A", Status: "active", DeptID: 20}
	b := deployTestBizScope{Code: "cred-scope-b", Name: "Credential Scope B", Status: "active", DeptID: 10}
	for _, scope := range []*deployTestBizScope{&a, &b} {
		if err := db.Create(scope).Error; err != nil {
			t.Fatalf("seed credential scope: %v", err)
		}
	}
	host := deployTestHost{
		Hostname: "credential-scope-host", IP: "10.40.0.91", SSHPort: 22, OS: "linux", Status: "assigned",
		BusinessScopeID: a.ID, BusinessScopeName: a.Name, DeptID: 20,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed credential host: %v", err)
	}
	return a.ID, b.ID, host.ID
}

// SEC-004: a restricted role only sees credentials owned by a business scope
// inside its data scope; another department's credential is neither visible nor
// editable by ID.
func TestDeployCredentialListRespectsBusinessScope(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	t.Setenv(deployCredentialKeyEnv, deployCredentialScopeTestKey)
	scopeA, scopeB, _ := seedCredentialScopes(t, svc, db)

	deptA := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 20}
	deptB := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 10}
	credA, err := svc.CreateCredential(CreateDeployCredentialRequest{Name: "cred-a", Username: "root", AuthMode: deployCredentialAuthModePassword, Secret: "secret-a", BusinessScopeID: scopeA}, "1", deptA)
	if err != nil {
		t.Fatalf("create scoped credential A: %v", err)
	}
	credB, err := svc.CreateCredential(CreateDeployCredentialRequest{Name: "cred-b", Username: "root", AuthMode: deployCredentialAuthModePassword, Secret: "secret-b", BusinessScopeID: scopeB}, "1", deptB)
	if err != nil {
		t.Fatalf("create scoped credential B: %v", err)
	}

	listA, err := svc.ListCredentials(deptA)
	if err != nil {
		t.Fatalf("list credentials for dept A: %v", err)
	}
	if len(listA) != 1 || listA[0].ID != credA.ID || listA[0].BusinessScopeID != scopeA {
		t.Fatalf("dept A must only see its own credential, got %+v", listA)
	}
	for _, row := range listA {
		if strings.Contains(row.Name, "cred-b") || row.Username != "root" {
			t.Fatalf("unexpected metadata in scoped list: %+v", row)
		}
	}

	listAll, err := svc.ListCredentials(nil)
	if err != nil {
		t.Fatalf("list credentials as admin: %v", err)
	}
	if len(listAll) != 2 {
		t.Fatalf("admin should see both credentials, got %d", len(listAll))
	}

	if _, err := svc.UpdateCredential(credB.ID, UpdateDeployCredentialRequest{Name: ptrString("hijacked")}, "1", deptA); err == nil || !strings.Contains(err.Error(), "credential.not_found") {
		t.Fatalf("cross-scope update must be denied as not found, got %v", err)
	}
	if err := svc.DeleteCredential(credB.ID, deptA); err == nil || !strings.Contains(err.Error(), "credential.not_found") {
		t.Fatalf("cross-scope delete must be denied as not found, got %v", err)
	}
}

// SEC-004: starting a task with another scope's credential by ID is refused;
// the same-scope credential still starts normally.
func TestDeployCredentialStartRequiresMatchingBusinessScope(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	t.Setenv(deployCredentialKeyEnv, deployCredentialScopeTestKey)
	scopeA, scopeB, hostID := seedCredentialScopes(t, svc, db)
	deptA := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 20}
	deptB := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 10}

	credA, err := svc.CreateCredential(CreateDeployCredentialRequest{Name: "start-a", Username: "root", AuthMode: deployCredentialAuthModePassword, Secret: "secret-a", BusinessScopeID: scopeA}, "1", deptA)
	if err != nil {
		t.Fatalf("create credential A: %v", err)
	}
	credB, err := svc.CreateCredential(CreateDeployCredentialRequest{Name: "start-b", Username: "root", AuthMode: deployCredentialAuthModePassword, Secret: "secret-b", BusinessScopeID: scopeB}, "1", deptB)
	if err != nil {
		t.Fatalf("create credential B: %v", err)
	}

	pkg, err := svc.CreatePackage(CreatePackageRequest{Name: "credential-pkg", Version: "1.0.0", InstallCommand: "echo ok", Status: "enabled"}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	task, err := svc.CreateTask(CreateTaskRequest{
		Name: "credential-scope-task", PackageID: pkg.ID, BusinessScopeID: scopeA,
		TargetType: TargetTypeHost, TargetIDs: []uint64{hostID}, ExecutorType: ExecutorTypeSSH,
	}, "1", deptA)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	if _, err := svc.StartTask(task.ID, StartTaskRequest{CredentialRefID: credB.ID}, "1", deptA); err == nil || !strings.Contains(err.Error(), "credential.out_of_scope") {
		t.Fatalf("foreign-scope credential must be refused, got %v", err)
	}

	svc.sshRunnerFactory = func(context.Context, cmdbHostSnapshot, StartTaskRequest) (deploySSHRunner, error) {
		return &fakeDeploySSHRunner{stdout: "done"}, nil
	}
	if _, err := svc.StartTask(task.ID, StartTaskRequest{CredentialRefID: credA.ID, HostFingerprint: "SHA256:test"}, "1", deptA); err != nil {
		t.Fatalf("same-scope credential should start: %v", err)
	}

	// The started task keeps the immutable reference, so deletion is refused.
	if err := svc.DeleteCredential(credA.ID, deptA); err == nil || !strings.Contains(err.Error(), "credential.in_use") {
		t.Fatalf("in-use credential must not be deleted, got %v", err)
	}
}

func TestDeployCredentialScopeTransferBumpsVersion(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	t.Setenv(deployCredentialKeyEnv, deployCredentialScopeTestKey)
	scopeA, scopeB, _ := seedCredentialScopes(t, svc, db)
	credential, err := svc.CreateCredential(CreateDeployCredentialRequest{
		Name: "scope-version", Username: "root", AuthMode: deployCredentialAuthModePassword,
		Secret: "secret", BusinessScopeID: scopeA,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}
	updated, err := svc.UpdateCredential(credential.ID, UpdateDeployCredentialRequest{
		BusinessScopeID: &scopeB,
	}, "1", nil)
	if err != nil {
		t.Fatalf("move credential: %v", err)
	}
	if updated.Version != credential.Version+1 {
		t.Fatalf("scope transfer version = %d, want %d", updated.Version, credential.Version+1)
	}
}

func TestCredentialExecutionMetadataChangesBumpVersion(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	t.Setenv(deployCredentialKeyEnv, deployCredentialScopeTestKey)
	scopeA, _, _ := seedCredentialScopes(t, svc, db)
	credential, err := svc.CreateCredential(CreateDeployCredentialRequest{
		Name: "metadata-version", Username: "root", AuthMode: deployCredentialAuthModePassword,
		Secret: "secret", BusinessScopeID: scopeA,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}

	updates := []struct {
		name string
		req  UpdateDeployCredentialRequest
	}{
		{name: "username", req: UpdateDeployCredentialRequest{Username: ptrString("deploy")}},
		{name: "auth mode", req: UpdateDeployCredentialRequest{AuthMode: ptrString(deployCredentialAuthModePrivateKey)}},
		{name: "status", req: UpdateDeployCredentialRequest{Status: ptrString("disabled")}},
	}
	for _, update := range updates {
		t.Run(update.name, func(t *testing.T) {
			updated, err := svc.UpdateCredential(credential.ID, update.req, "1", nil)
			if err != nil {
				t.Fatalf("update credential: %v", err)
			}
			if updated.Version != credential.Version+1 {
				t.Fatalf("credential version = %d, want %d", updated.Version, credential.Version+1)
			}
			credential = updated
		})
	}

	unchanged, err := svc.UpdateCredential(credential.ID, UpdateDeployCredentialRequest{
		Username: ptrString(credential.Username),
		AuthMode: ptrString(credential.AuthMode),
		Status:   ptrString(credential.Status),
	}, "1", nil)
	if err != nil {
		t.Fatalf("update unchanged metadata: %v", err)
	}
	if unchanged.Version != credential.Version {
		t.Fatalf("no-op metadata update changed version from %d to %d", credential.Version, unchanged.Version)
	}
}

func TestUpdateCredentialRechecksDataScopeUnderLock(t *testing.T) {
	db := setupDeployTestDB(t)
	if db.Dialector.Name() != "mysql" {
		t.Skip("credential update locking requires MySQL")
	}
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	t.Setenv(deployCredentialKeyEnv, deployCredentialScopeTestKey)
	scopeA, scopeB, _ := seedCredentialScopes(t, svc, db)
	deptA := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 20}
	credential, err := svc.CreateCredential(CreateDeployCredentialRequest{
		Name: "update-scope-race", Username: "root", AuthMode: deployCredentialAuthModePassword,
		Secret: "original-secret", BusinessScopeID: scopeA,
	}, "1", deptA)
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}
	var before DeployCredentialRef
	if err := db.First(&before, credential.ID).Error; err != nil {
		t.Fatalf("load credential before ownership change: %v", err)
	}

	transferTx := db.Begin()
	if transferTx.Error != nil {
		t.Fatalf("begin ownership transfer: %v", transferTx.Error)
	}
	var locked DeployCredentialRef
	if err := transferTx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, credential.ID).Error; err != nil {
		transferTx.Rollback()
		t.Fatalf("lock credential for ownership transfer: %v", err)
	}

	const callbackName = "test:update-credential-query-started"
	queryStarted := make(chan struct{}, 1)
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "biz_deploy_credential_ref" {
			select {
			case queryStarted <- struct{}{}:
			default:
			}
		}
	}); err != nil {
		transferTx.Rollback()
		t.Fatalf("register query-start synchronization: %v", err)
	}
	updated := make(chan error, 1)
	go func() {
		_, err := svc.UpdateCredential(credential.ID, UpdateDeployCredentialRequest{
			Secret: ptrString("unauthorized-rotation"),
		}, "1", deptA)
		updated <- err
	}()
	select {
	case <-queryStarted:
	case <-time.After(5 * time.Second):
		transferTx.Rollback()
		t.Fatal("restricted update did not reach its credential query")
	}
	select {
	case err := <-updated:
		transferTx.Rollback()
		t.Fatalf("update must wait for the in-flight ownership transfer, returned early: %v", err)
	case <-time.After(500 * time.Millisecond):
	}

	if err := transferTx.Model(&DeployCredentialRef{}).Where("id = ?", credential.ID).
		Updates(map[string]any{"business_scope_id": scopeB, "version": gorm.Expr("version + 1")}).Error; err != nil {
		transferTx.Rollback()
		t.Fatalf("move credential during restricted update: %v", err)
	}
	if err := transferTx.Commit().Error; err != nil {
		t.Fatalf("commit ownership transfer: %v", err)
	}
	select {
	case err := <-updated:
		if err == nil || !strings.Contains(err.Error(), "credential.not_found") {
			t.Fatalf("restricted update after ownership transfer must be denied, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("restricted update did not finish after ownership transfer")
	}

	var after DeployCredentialRef
	if err := db.First(&after, credential.ID).Error; err != nil {
		t.Fatalf("reload credential after ownership transfer: %v", err)
	}
	if after.BusinessScopeID != scopeB || after.Version != before.Version+1 || after.SecretEncrypted != before.SecretEncrypted {
		t.Fatalf("restricted update changed transferred credential: before=%+v after=%+v", before, after)
	}
}

func TestDeployCredentialStartClaimRechecksScopeUnderLock(t *testing.T) {
	db := setupDeployTestDB(t)
	if db.Dialector.Name() != "mysql" {
		t.Skip("credential claim locking requires MySQL")
	}
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	t.Setenv(deployCredentialKeyEnv, deployCredentialScopeTestKey)
	scopeA, scopeB, hostID := seedCredentialScopes(t, svc, db)
	deptA := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 20}
	credential, err := svc.CreateCredential(CreateDeployCredentialRequest{
		Name: "claim-scope-race", Username: "root", AuthMode: deployCredentialAuthModePassword,
		Secret: "secret", BusinessScopeID: scopeA,
	}, "1", deptA)
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name: "claim-scope-package", Version: "1.0.0", InstallCommand: "echo ok", Status: PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	task, err := svc.CreateTask(CreateTaskRequest{
		Name: "claim-scope-task", PackageID: pkg.ID, BusinessScopeID: scopeA,
		TargetType: TargetTypeHost, TargetIDs: []uint64{hostID}, ExecutorType: ExecutorTypeSSH,
	}, "1", deptA)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	persistedTask, err := svc.loadVisibleTask(task.ID, deptA)
	if err != nil {
		t.Fatalf("load task: %v", err)
	}
	req, version, err := svc.prepareStartCredential(
		StartTaskRequest{CredentialRefID: credential.ID, HostFingerprint: "SHA256:claim-race"},
		*persistedTask, false,
	)
	if err != nil {
		t.Fatalf("resolve credential before ownership change: %v", err)
	}
	hosts, err := svc.prepareStartTargets(persistedTask, req, deptA)
	if err != nil {
		t.Fatalf("prepare targets: %v", err)
	}
	claim := &startTaskClaim{
		service: svc, dataScope: deptA, task: persistedTask, req: req, hosts: hosts, actor: "1",
		now: time.Now(), timeout: time.Minute, leaseOwner: deployHostLeaseOwner(persistedTask.ID),
		credentialVersion: version,
	}

	if err := db.Model(&DeployCredentialRef{}).Where("id = ?", credential.ID).
		Update("business_scope_id", scopeB).Error; err != nil {
		t.Fatalf("move credential after preflight resolution: %v", err)
	}
	if err := svc.deploymentTransaction(claim.claim); !errors.Is(err, ErrTaskPreviewConflict) {
		t.Fatalf("claim should reject credential ownership drift under row lock, got %v", err)
	}
	var current DeployTask
	if err := db.First(&current, task.ID).Error; err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if current.Status != TaskStatusDraft {
		t.Fatalf("rejected claim mutated task status to %q", current.Status)
	}
	var hostRows int64
	if err := db.Model(&DeployTaskHost{}).Where("task_id = ?", task.ID).Count(&hostRows).Error; err != nil {
		t.Fatalf("count task hosts: %v", err)
	}
	if hostRows != 0 {
		t.Fatalf("rejected claim persisted %d task host rows", hostRows)
	}

	if err := db.Model(&DeployCredentialRef{}).Where("id = ?", credential.ID).
		Update("business_scope_id", scopeA).Error; err != nil {
		t.Fatalf("restore credential scope: %v", err)
	}
	deleteReq, deleteVersion, err := svc.prepareStartCredential(
		StartTaskRequest{CredentialRefID: credential.ID, HostFingerprint: "SHA256:deleted-race"},
		*persistedTask, false,
	)
	if err != nil {
		t.Fatalf("resolve credential before deletion: %v", err)
	}
	deleteClaim := &startTaskClaim{
		service: svc, dataScope: deptA, task: persistedTask, req: deleteReq, hosts: hosts, actor: "1",
		now: time.Now(), timeout: time.Minute, leaseOwner: deployHostLeaseOwner(persistedTask.ID),
		credentialVersion: deleteVersion,
	}
	if err := svc.DeleteCredential(credential.ID, nil); err != nil {
		t.Fatalf("delete unreferenced credential: %v", err)
	}
	if err := svc.deploymentTransaction(deleteClaim.claim); !errors.Is(err, ErrTaskPreviewConflict) {
		t.Fatalf("claim should return a preview conflict after credential deletion, got %v", err)
	}
}

func TestDeleteCredentialSerializesWithCredentialClaim(t *testing.T) {
	db := setupDeployTestDB(t)
	if db.Dialector.Name() != "mysql" {
		t.Skip("credential claim locking requires MySQL")
	}
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	t.Setenv(deployCredentialKeyEnv, deployCredentialScopeTestKey)
	scopeA, _, hostID := seedCredentialScopes(t, svc, db)
	credential, err := svc.CreateCredential(CreateDeployCredentialRequest{
		Name: "delete-claim-race", Username: "root", AuthMode: deployCredentialAuthModePassword,
		Secret: "secret", BusinessScopeID: scopeA,
	}, "1", &common.DataScopeReq{IsAdmin: true})
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name: "delete-claim-package", Version: "1.0.0", InstallCommand: "echo ok", Status: PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	task, err := svc.CreateTask(CreateTaskRequest{
		Name: "delete-claim-task", PackageID: pkg.ID, BusinessScopeID: scopeA,
		TargetType: TargetTypeHost, TargetIDs: []uint64{hostID}, ExecutorType: ExecutorTypeSSH,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	claimTx := db.Begin()
	if claimTx.Error != nil {
		t.Fatalf("begin claim transaction: %v", claimTx.Error)
	}
	var locked DeployCredentialRef
	if err := claimTx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, credential.ID).Error; err != nil {
		claimTx.Rollback()
		t.Fatalf("lock credential for claim: %v", err)
	}

	started := make(chan struct{})
	queryStarted := make(chan struct{}, 1)
	if err := db.Callback().Query().Before("gorm:query").Register("test:credential-delete-query-started", func(*gorm.DB) {
		select {
		case queryStarted <- struct{}{}:
		default:
		}
	}); err != nil {
		claimTx.Rollback()
		t.Fatalf("register query-start synchronization: %v", err)
	}
	deleted := make(chan error, 1)
	go func() {
		close(started)
		deleted <- svc.DeleteCredential(credential.ID, nil)
	}()
	<-started
	select {
	case <-queryStarted:
	case <-time.After(5 * time.Second):
		claimTx.Rollback()
		t.Fatal("delete did not reach its credential query")
	}
	select {
	case err := <-deleted:
		claimTx.Rollback()
		t.Fatalf("delete must wait for the in-flight credential claim, returned early: %v", err)
	case <-time.After(500 * time.Millisecond):
	}

	if err := claimTx.Model(&DeployTask{}).Where("id = ?", task.ID).
		Update("credential_ref_id", credential.ID).Error; err != nil {
		claimTx.Rollback()
		t.Fatalf("persist task credential reference: %v", err)
	}
	if err := claimTx.Commit().Error; err != nil {
		t.Fatalf("commit task credential reference: %v", err)
	}
	if err := <-deleted; err == nil || !strings.Contains(err.Error(), "credential.in_use") {
		t.Fatalf("delete must observe the committed task reference, got %v", err)
	}
}

// A global (scope 0) credential stays admin-owned: a restricted role cannot
// create one, and a restricted-role start cannot use it on a scoped task.
func TestDeployCredentialGlobalScopeIsAdminOnly(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	t.Setenv(deployCredentialKeyEnv, deployCredentialScopeTestKey)
	scopeA, _, _ := seedCredentialScopes(t, svc, db)
	deptA := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 20}

	if _, err := svc.CreateCredential(CreateDeployCredentialRequest{Name: "global-attempt", Username: "root", AuthMode: deployCredentialAuthModePassword, Secret: "s"}, "1", deptA); err == nil || !strings.Contains(err.Error(), "credential.business_scope_required") {
		t.Fatalf("scoped role must not create a global credential, got %v", err)
	}

	global, err := svc.CreateCredential(CreateDeployCredentialRequest{Name: "global-cred", Username: "root", AuthMode: deployCredentialAuthModePassword, Secret: "s"}, "1", nil)
	if err != nil {
		t.Fatalf("admin global credential: %v", err)
	}
	task := DeployTask{ID: 7, BusinessScopeID: scopeA, CredentialRefID: global.ID}
	if _, _, err := svc.resolveStartCredential(StartTaskRequest{CredentialRefID: global.ID}, task, false); err == nil || !strings.Contains(err.Error(), "credential.out_of_scope") {
		t.Fatalf("scoped start must not use a global credential, got %v", err)
	}
	// Admin/system callers and the queue-recovery path carry allowGlobal=true.
	if _, _, err := svc.resolveStartCredential(StartTaskRequest{CredentialRefID: global.ID}, task, true); err != nil {
		t.Fatalf("admin start should use a global credential: %v", err)
	}
	legacyTask := DeployTask{ID: 8, BusinessScopeID: 0, CredentialRefID: global.ID}
	if _, _, err := svc.resolveStartCredential(StartTaskRequest{CredentialRefID: global.ID}, legacyTask, false); err == nil || !strings.Contains(err.Error(), "credential.out_of_scope") {
		t.Fatalf("scoped role must not use a global credential on a legacy task, got %v", err)
	}
	if _, _, err := svc.resolveStartCredential(StartTaskRequest{CredentialRefID: global.ID}, legacyTask, true); err != nil {
		t.Fatalf("admin start should use a global credential on a legacy task: %v", err)
	}

	// Cross-scope create is refused even when the requested scope exists.
	if _, err := svc.CreateCredential(CreateDeployCredentialRequest{Name: "foreign", Username: "root", AuthMode: deployCredentialAuthModePassword, Secret: "s", BusinessScopeID: scopeA}, "1", &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 99}); err == nil || !strings.Contains(err.Error(), "credential.business_scope_forbidden") {
		t.Fatalf("cross-department create must be refused, got %v", err)
	}
}

func ptrString(value string) *string { return &value }

// The T05 migration must add business_scope_id on the legacy-upgrade path and
// stay a no-op when replayed, matching golang-migrate's statement-by-statement
// execution.
func TestDeployCredentialScopeMigrationIsIdempotent(t *testing.T) {
	db := testmysql.Open(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `CREATE TABLE biz_deploy_credential_ref (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
		name VARCHAR(128) NOT NULL,
		PRIMARY KEY (id)
	) ENGINE=InnoDB`); err != nil {
		t.Fatal(err)
	}
	applySQLScript(t, conn, credentialScopeUp, "T05 migration")
	applySQLScript(t, conn, credentialScopeUp, "T05 migration rerun")

	var count int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_credential_ref' AND COLUMN_NAME='business_scope_id'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("business_scope_id column count = %d, want 1", count)
	}
	var indexes int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_credential_ref' AND INDEX_NAME='idx_deploy_credential_business_scope'`).Scan(&indexes); err != nil {
		t.Fatal(err)
	}
	if indexes == 0 {
		t.Fatal("business scope index was not created")
	}
}
