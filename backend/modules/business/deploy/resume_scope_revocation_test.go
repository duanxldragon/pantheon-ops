package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/gorm"
)

type admissionScopeReader struct {
	bizcap.BizScopeReader
	read func(context.Context, uint64, *common.DataScopeReq) (bizcap.BizScopeRef, error)
}

func (r admissionScopeReader) GetActive(ctx context.Context, id uint64, scope *common.DataScopeReq) (bizcap.BizScopeRef, error) {
	return r.read(ctx, id, scope)
}

func executionAdmissionFacts(t *testing.T, db *gorm.DB, id uint64) string {
	t.Helper()
	var task DeployTask
	var hosts []DeployTaskHost
	var events []DeployTaskEvent
	var attempts []DeployTaskAttempt
	for _, read := range []*gorm.DB{db.First(&task, id), db.Where("task_id = ?", id).Order("id").Find(&hosts), db.Where("task_id = ?", id).Order("id").Find(&events), db.Where("task_id = ?", id).Order("id").Find(&attempts)} {
		if read.Error != nil {
			t.Fatal(read.Error)
		}
	}
	raw, err := json.Marshal([]any{task, hosts, events, attempts})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestResumeScopeRevocationAdmission(t *testing.T) {
	for _, action := range []string{"resume", "retry"} {
		for _, executor := range []string{ExecutorTypeManual, ExecutorTypeSSH} {
			for _, direct := range []bool{false, true} {
				t.Run(action+"/"+executor+"/direct="+boolString(direct), func(t *testing.T) {
					db, svc, task, _ := previewFixture(t, false)
					started, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil)
					if err != nil {
						t.Fatal(err)
					}
					status := TaskStatusPaused
					if action == "retry" {
						status = TaskStatusFailed
					}
					if err := db.Model(&DeployTask{}).Where("id = ?", task.ID).Updates(map[string]any{"status": status, "executor_type": executor}).Error; err != nil {
						t.Fatal(err)
					}
					if err := db.Model(&DeployTaskHost{}).Where("id = ?", started.Hosts[0].ID).Updates(map[string]any{"status": TaskHostStatusFailed, "error_message": "retained failure"}).Error; err != nil {
						t.Fatal(err)
					}
					view, err := svc.GetTask(task.ID, nil)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := svc.bizScopeReader.GetActive(context.Background(), task.BusinessScopeID, nil); err != nil {
						t.Fatal(err)
					}
					if err := db.Model(&deployTestBizScope{}).Where("id = ?", task.BusinessScopeID).Update("status", "inactive").Error; err != nil {
						t.Fatal(err)
					}
					if _, err := svc.bizScopeReader.GetActive(context.Background(), task.BusinessScopeID, nil); err == nil {
						t.Fatal("owner accepted inactive source")
					}
					before := executionAdmissionFacts(t, db, task.ID)
					// An admitted SSH command would queue the real worker. Refusal
					// must happen before any task/event/attempt change or dispatch.
					svc.SetAsyncExecution(executor == ExecutorTypeSSH)
					if direct {
						if action == "resume" {
							_, err = svc.ResumeTask(task.ID, "1", nil)
						} else {
							_, err = svc.RetryTask(task.ID, "1", nil)
						}
					} else {
						_, err = svc.ControlTask(context.Background(), task.ID, "1", nil, action, TaskCommandRequest{StateVersion: view.StateVersion})
					}
					if err == nil {
						t.Fatal("inactive source admitted execution")
					}
					if before != executionAdmissionFacts(t, db, task.ID) {
						t.Fatal("refused admission changed task/hosts/events/attempts")
					}
					if !strings.Contains(err.Error(), "scope") {
						t.Fatalf("unexpected refusal: %v", err)
					}
					if _, running := svc.taskWorkers.Load(task.ID); running {
						t.Fatal("refused command dispatched a worker")
					}
				})
			}
		}
	}
}

func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func TestResumeScopeRevocationBetweenClaimAndExecute(t *testing.T) {
	db, svc, task, _ := previewFixture(t, false)
	started, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	var stored DeployTask
	if err := db.First(&stored, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	stored.ExecutorType = ExecutorTypeSSH
	var host DeployTaskHost
	if err := db.First(&host, started.Hosts[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	attempt := svc.claimAttempt(task.ID, host.ID, "scope-revocation-test")
	if attempt == nil {
		t.Fatal("active claim failed")
	}
	if err := db.Model(&deployTestBizScope{}).Where("id = ?", task.BusinessScopeID).Update("status", "inactive").Error; err != nil {
		t.Fatal(err)
	}
	var connects atomic.Int32
	svc.sshRunnerFactory = func(context.Context, cmdbHostSnapshot, StartTaskRequest) (deploySSHRunner, error) {
		connects.Add(1)
		return &fakeDeploySSHRunner{}, nil
	}
	owned := svc.ownedService(tokenForAttempt(*attempt, host))
	err = owned.executeAsyncHostAttempt(context.Background(), stored, host, StartTaskRequest{}, "1")
	if err == nil || connects.Load() != 0 {
		t.Fatalf("revoked claim reached SSH: err=%v connects=%d", err, connects.Load())
	}
	svc.finishAttempt(owned.ownership, err)
	var result DeployTaskHost
	if err := db.First(&result, host.ID).Error; err != nil {
		t.Fatal(err)
	}
	if result.Status != TaskHostStatusRunning {
		t.Fatal("failed ownership read must not convert into a host failure write")
	}
}

func TestResumeScopeRevocationFailClosed(t *testing.T) {
	db, svc, task, _ := previewFixture(t, false)
	started, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	paused, err := svc.ControlTask(context.Background(), task.ID, "1", nil, "pause", TaskCommandRequest{StateVersion: started.StateVersion})
	if err != nil {
		t.Fatal(err)
	}
	reader := svc.bizScopeReader
	injected := errors.New("source owner unavailable")
	for _, name := range []string{"nil-reader", "owner-error", "wrong-owner", "inactive-projection", "canceled-owner", "source-gone"} {
		t.Run(name, func(t *testing.T) {
			svc.bizScopeReader = reader
			if name == "nil-reader" {
				svc.bizScopeReader = nil
			}
			if name == "source-gone" {
				if err := db.Delete(&deployTestBizScope{}, task.BusinessScopeID).Error; err != nil {
					t.Fatal(err)
				}
			}
			if name != "nil-reader" && name != "source-gone" {
				svc.bizScopeReader = admissionScopeReader{BizScopeReader: reader, read: func(ctx context.Context, id uint64, scope *common.DataScopeReq) (bizcap.BizScopeRef, error) {
					switch name {
					case "owner-error":
						return bizcap.BizScopeRef{}, injected
					case "canceled-owner":
						return bizcap.BizScopeRef{}, context.Canceled
					case "wrong-owner":
						return bizcap.BizScopeRef{ID: id + 1, Status: "active"}, nil
					default:
						return bizcap.BizScopeRef{ID: id, Status: "inactive"}, nil
					}
				}}
			}
			before := executionAdmissionFacts(t, db, task.ID)
			_, err := svc.ControlTask(context.Background(), task.ID, "1", nil, "resume", TaskCommandRequest{StateVersion: paused.StateVersion})
			if err == nil || before != executionAdmissionFacts(t, db, task.ID) {
				t.Fatalf("failed-open %s: %v", name, err)
			}
			if name == "owner-error" && !errors.Is(err, injected) {
				t.Fatal("owner cause lost")
			}
			if name != "nil-reader" && (!deployTaskValidationError(err) || err.Error() != "business.deploy.task.scopeInvalid") {
				t.Fatalf("noncanonical scope rejection: %v", err)
			}
			if name == "canceled-owner" && !errors.Is(err, context.Canceled) {
				t.Fatal("owner cancellation lost")
			}
		})
	}
	svc.bizScopeReader = reader
	if _, err := svc.ControlTask(context.Background(), task.ID, "1", nil, "cancel", TaskCommandRequest{StateVersion: paused.StateVersion}); err != nil {
		t.Fatalf("gone source blocked cleanup: %v", err)
	}
	if err := svc.validateActiveExecutionScope(context.Background(), nil, nil); err == nil {
		t.Fatal("nil source accepted")
	}
	if err := svc.validateActiveExecutionScope(context.Background(), &DeployTask{TargetType: TargetTypeHost}, nil); err == nil {
		t.Fatal("missing declared host source accepted")
	}
}

func TestResumeScopeRevocationCurrentOwnerAndVisibility(t *testing.T) {
	db, svc, task, hostID := previewFixture(t, false)
	started, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	paused, err := svc.ControlTask(context.Background(), task.ID, "1", nil, "pause", TaskCommandRequest{StateVersion: started.StateVersion})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"owner-outside-data-scope", "host-outside-data-scope", "host-owner-changed", "host-gone"} {
		t.Run(name, func(t *testing.T) {
			if err := db.Model(&deployTestBizScope{}).Where("id = ?", task.BusinessScopeID).Update("dept_id", 1).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&deployTestHost{}).Where("id = ?", hostID).Updates(map[string]any{"dept_id": 1, "business_scope_id": task.BusinessScopeID}).Error; err != nil {
				t.Fatal(err)
			}
			scope := &common.DataScopeReq{Mode: common.DataScopeModeCustom, DeptIDs: []uint64{1}}
			switch name {
			case "owner-outside-data-scope":
				if err := db.Model(&deployTestBizScope{}).Where("id = ?", task.BusinessScopeID).Update("dept_id", 2).Error; err != nil {
					t.Fatal(err)
				}
			case "host-outside-data-scope":
				if err := db.Model(&deployTestHost{}).Where("id = ?", hostID).Update("dept_id", 2).Error; err != nil {
					t.Fatal(err)
				}
			case "host-owner-changed":
				if err := db.Model(&deployTestHost{}).Where("id = ?", hostID).Update("business_scope_id", task.BusinessScopeID+1).Error; err != nil {
					t.Fatal(err)
				}
			case "host-gone":
				if err := db.Delete(&deployTestHost{}, hostID).Error; err != nil {
					t.Fatal(err)
				}
			}
			before := executionAdmissionFacts(t, db, task.ID)
			if _, err := svc.ControlTask(context.Background(), task.ID, "1", scope, "resume", TaskCommandRequest{StateVersion: paused.StateVersion}); err == nil {
				t.Fatal("current ownership/visibility bypassed")
			}
			if before != executionAdmissionFacts(t, db, task.ID) {
				t.Fatal("scope refusal mutated task")
			}
		})
	}
}

func TestResumeScopeRevocationStartRefusedWithoutMutation(t *testing.T) {
	db, svc, task, _ := previewFixture(t, false)
	if err := db.Model(&deployTestBizScope{}).Where("id = ?", task.BusinessScopeID).Update("status", "inactive").Error; err != nil {
		t.Fatal(err)
	}
	before := executionAdmissionFacts(t, db, task.ID)
	if _, err := svc.StartTaskContext(context.Background(), task.ID, StartTaskRequest{}, "1", nil); err == nil {
		t.Fatal("inactive start accepted")
	}
	if before != executionAdmissionFacts(t, db, task.ID) {
		t.Fatal("refused start mutated facts")
	}
}

func TestResumeScopeRevocationAfterSSHConnectSendsNoCommand(t *testing.T) {
	db, svc, task, _ := previewFixture(t, false)
	started, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	var stored DeployTask
	if err := db.First(&stored, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	stored.ExecutorType = ExecutorTypeSSH
	received := make(chan []string, 1)
	target, req := startDeploySSHTransport(t, serveDeploySSHScript(received, 0))
	target.ID = started.Hosts[0].HostID
	var connects atomic.Int32
	svc.sshRunnerFactory = func(ctx context.Context, host cmdbHostSnapshot, request StartTaskRequest) (deploySSHRunner, error) {
		runner, err := newDeploySSHRunner(ctx, host, request)
		if err != nil {
			return nil, err
		}
		connects.Add(1)
		if err := db.Model(&deployTestBizScope{}).Where("id = ?", task.BusinessScopeID).Update("status", "inactive").Error; err != nil {
			_ = runner.Close()
			return nil, err
		}
		return runner, nil
	}
	var taskHost DeployTaskHost
	if err := db.First(&taskHost, started.Hosts[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	owned, ownedCtx := ownedClaim(t, svc, stored, taskHost, "scope-command")
	ctx, cancel := context.WithTimeout(ownedCtx, 5*time.Second)
	defer cancel()
	if err := owned.executeOrchestratedSSHHost(ctx, stored, target, started.Hosts[0], req, "1"); err == nil {
		t.Fatal("revoked command accepted")
	}
	if owned.ownership.check() == nil {
		t.Fatal("revocation did not latch cancellation")
	}
	if connects.Load() != 1 {
		t.Fatal("actual loopback SSH connection not established")
	}
	select {
	case <-received:
		t.Fatal("revoked source sent an SSH command")
	default:
	}
	var host DeployTaskHost
	if err := db.First(&host, started.Hosts[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if host.Status != TaskHostStatusRunning || host.ErrorMessage != "" {
		t.Fatalf("revoked command wrote a failure after losing ownership: %s", host.Status)
	}
}

// Admission cancellation is request-bound; cleanup remains available after the
// source is inactive and never depends on the active execution guard.
func TestResumeScopeRevocationCleanupAndCanceledRequest(t *testing.T) {
	db, svc, task, _ := previewFixture(t, false)
	started, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&deployTestBizScope{}).Where("id = ?", task.BusinessScopeID).Update("status", "inactive").Error; err != nil {
		t.Fatal(err)
	}
	paused, err := svc.ControlTask(context.Background(), task.ID, "1", nil, "pause", TaskCommandRequest{StateVersion: started.StateVersion})
	if err != nil {
		t.Fatalf("inactive source blocked pause: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	cancel()
	before := executionAdmissionFacts(t, db, task.ID)
	if _, err := svc.ControlTask(ctx, task.ID, "1", &common.DataScopeReq{IsAdmin: true}, "resume", TaskCommandRequest{StateVersion: paused.StateVersion}); err == nil {
		t.Fatal("canceled admission accepted")
	}
	if before != executionAdmissionFacts(t, db, task.ID) {
		t.Fatal("canceled request mutated state")
	}
	if _, err := svc.ControlTask(context.Background(), task.ID, "1", nil, "cancel", TaskCommandRequest{StateVersion: paused.StateVersion}); err != nil {
		t.Fatalf("inactive source blocked cleanup: %v", err)
	}
}
