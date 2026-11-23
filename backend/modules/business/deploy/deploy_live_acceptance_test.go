package deploy

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
)

func liveDeploySSHRequest(t *testing.T) (cmdbHostSnapshot, StartTaskRequest) {
	t.Helper()
	host := os.Getenv("PANTHEON_ACCEPTANCE_HOST")
	if host == "" {
		t.Skip("live VM acceptance not requested")
	}
	if host != "115.190.251.143" {
		t.Fatal("VM outside recorded acceptance scope")
	}
	req := StartTaskRequest{SSHUser: os.Getenv("PANTHEON_ACCEPTANCE_SSH_USER"), AuthMode: os.Getenv("PANTHEON_ACCEPTANCE_SSH_AUTH_MODE"), HostFingerprint: os.Getenv("PANTHEON_ACCEPTANCE_SSH_FINGERPRINT"), TimeoutSeconds: 10}
	secret := os.Getenv("PANTHEON_ACCEPTANCE_SSH_SECRET")
	if req.SSHUser == "" || secret == "" || req.HostFingerprint == "" {
		t.Fatal("required live SSH credential reference unavailable")
	}
	if req.AuthMode == "password" {
		req.SSHPassword = secret
	} else {
		req.SSHPrivateKey = secret
	}
	return cmdbHostSnapshot{IP: host, SSHPort: 22, OS: "linux"}, req
}

func TestDeployLiveVMExecution(t *testing.T) {
	target, req := liveDeploySSHRequest(t)
	for _, fail := range []bool{false, true} {
		name := "success"
		if fail {
			name = "failure"
		}
		t.Run(name, func(t *testing.T) {
			db := setupDeployTestDB(t)
			svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
			t.Setenv(deployCredentialKeyEnv, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
			secret := req.SSHPassword
			if secret == "" {
				secret = req.SSHPrivateKey
			}
			credential, err := svc.CreateCredential(CreateDeployCredentialRequest{Name: "p0-vm", Username: req.SSHUser, AuthMode: req.AuthMode, Secret: secret}, "1")
			if err != nil {
				t.Fatal(err)
			}
			scopeID, hostID := seedDeployReliabilityFixture(t, db, "p0-test-vm", target.IP)
			markerRunner, err := newDeploySSHRunner(context.Background(), target, req)
			if err != nil {
				t.Fatal("allocate owned marker connection failed")
			}
			markerCtx, markerCancel := context.WithTimeout(context.Background(), 10*time.Second)
			markerRoot, _, markerErr := markerRunner.RunScript(markerCtx, "mktemp -d /tmp/omx-ops-live-XXXXXXXXXXXX")
			markerCancel()
			markerRunner.Close()
			markerRoot = strings.TrimSpace(markerRoot)
			if markerErr != nil || !regexp.MustCompile("^/tmp/omx-ops-live-[A-Za-z0-9]{12}$").MatchString(markerRoot) {
				t.Fatal("exclusive remote marker directory allocation failed")
			}
			marker := markerRoot + "/" + name + ".txt"
			t.Cleanup(func() {
				runner, err := newDeploySSHRunner(context.Background(), target, req)
				if err != nil {
					t.Error("cleanup SSH connection failed")
					return
				}
				defer runner.Close()
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if _, _, err := runner.RunScript(ctx, "rm -f -- "+shellQuoteSingle(marker)+"\nrmdir -- "+shellQuoteSingle(markerRoot)); err != nil {
					t.Error("owned deployment marker cleanup failed")
				}
			})
			script := "printf 'p0-frozen-marker' > " + marker + "\nprintf 'p0-live-stdout\\n'\nprintf 'p0-live-stderr\\n' >&2\n"
			if fail {
				script += "exit 17\n"
			}
			pkg, err := svc.CreatePackage(CreatePackageRequest{Name: "p0-vm-component", Version: "1.0.0", InstallCommand: script, UninstallCommand: "rm -f -- " + marker + "\nprintf 'p0-live-uninstall\\n'", Status: PackageStatusEnabled}, "1")
			if err != nil {
				t.Fatal(err)
			}
			task, err := svc.CreateTask(CreateTaskRequest{Name: "p0-vm-task", PackageID: pkg.ID, BusinessScopeID: scopeID, TargetType: TargetTypeHost, TargetIDs: []uint64{hostID}, ExecutorType: ExecutorTypeSSH}, "1", nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&DeployPackage{}).Where("id = ?", pkg.ID).Updates(map[string]interface{}{"install_command": "echo changed-forward", "uninstall_command": "echo changed-inverse"}).Error; err != nil {
				t.Fatal(err)
			}
			start := StartTaskRequest{CredentialRefID: credential.ID, HostFingerprint: req.HostFingerprint, TimeoutSeconds: 10}
			result, runErr := svc.StartTask(task.ID, start, "1", nil)
			if fail {
				if runErr == nil || !strings.Contains(runErr.Error(), "status 17") {
					t.Fatalf("expected remote exit17, got %v", runErr)
				}
				result, err = svc.GetTask(task.ID, nil)
				if err != nil {
					t.Fatal(err)
				}
			} else if runErr != nil {
				t.Fatal(runErr)
			}
			expectedTask, expectedStep := TaskStatusSuccess, StepStatusSuccess
			if fail {
				expectedTask, expectedStep = TaskStatusFailed, StepStatusFailed
			}
			if result.Status != expectedTask || len(result.Steps) != 1 || result.Steps[0].Status != expectedStep || len(result.StepHistory) != 1 {
				t.Fatal("real VM result/checkpoint attribution incorrect")
			}
			var logs []DeployTaskLogChunk
			if err := db.Where("task_id = ?", task.ID).Find(&logs).Error; err != nil {
				t.Fatal(err)
			}
			stdout, stderr := false, false
			for _, log := range logs {
				if log.AttemptNo != 1 || log.TaskStepID != result.Steps[0].ID {
					t.Fatal("real VM log attribution incorrect")
				}
				stdout = stdout || strings.Contains(log.Content, "p0-live-stdout")
				stderr = stderr || strings.Contains(log.Content, "p0-live-stderr")
				if strings.Contains(log.Content, secret) {
					t.Fatal("credential leaked into log")
				}
			}
			if !stdout || !stderr {
				t.Fatal("real VM streams not persisted")
			}
			runner, err := newDeploySSHRunner(context.Background(), target, req)
			if err != nil {
				t.Fatal(err)
			}
			defer runner.Close()
			out, _, err := runner.RunScript(context.Background(), "cat "+marker)
			if err != nil || out != "p0-frozen-marker" {
				t.Fatal("frozen executable input did not run on VM")
			}
			// Reinstantiate the service to prove committed metadata survives a process boundary.
			restarted := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
			persisted, err := restarted.GetTask(task.ID, nil)
			if err != nil || persisted.Status != expectedTask || len(persisted.StepHistory) != 1 {
				t.Fatal("persisted VM execution state unavailable after service restart")
			}
			// A recorded package inverse is used; editing the mutable package cannot replace it.
			rollback, err := restarted.RollbackTask(context.Background(), task.ID, "1")
			if fail {
				// Failed installation does not prove that the host is online.
				// The current inverse policy must deny it until state is reconciled.
				if err == nil || err.Error() != "business.deploy.task.targetStatusMismatch" {
					t.Fatal("failed install inverse bypassed host state validation")
				}
				t.Log("failed partial installation inverse denied by current host state policy; owned marker cleaned separately")
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			rolled, err := restarted.StartTask(rollback.ID, start, "1", nil)
			if err != nil || rolled.Status != TaskStatusSuccess {
				t.Fatalf("real package inverse failed: %v", err)
			}
			var inverseLogs []DeployTaskLogChunk
			if err := db.Where("task_id = ?", rollback.ID).Find(&inverseLogs).Error; err != nil {
				t.Fatal(err)
			}
			foundInverse := false
			for _, line := range inverseLogs {
				foundInverse = foundInverse || strings.Contains(line.Content, "p0-live-uninstall")
			}
			if !foundInverse {
				t.Fatal("second task lost real inverse output")
			}
			if _, _, err := runner.RunScript(context.Background(), "test ! -f "+marker); err != nil {
				t.Fatal("frozen inverse did not remove owned marker")
			}
			denied := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 777, UserID: 777}
			if _, err := restarted.GetTask(task.ID, denied); err == nil {
				t.Fatal("unauthorized actor could recover execution state")
			}
			t.Logf("actual VM exit/streams/frozen input/package inverse/history/scope/cleanup verified at %s", time.Now().UTC().Format(time.RFC3339))
		})
	}
}

func TestDeployLiveVMCancellation(t *testing.T) {
	target, req := liveDeploySSHRequest(t)
	runner, err := newDeploySSHRunner(context.Background(), target, req)
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	started := time.Now()
	out, _, err := runner.RunScript(ctx, "printf 'p0-before-cancel\\n'\nsleep 10\nprintf 'p0-after-cancel\\n'")
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > 3*time.Second {
		t.Fatalf("actual VM command ignored cancellation: %v", err)
	}
	if strings.Contains(out, "p0-after-cancel") {
		t.Fatal("canceled VM output appeared successful")
	}
}
