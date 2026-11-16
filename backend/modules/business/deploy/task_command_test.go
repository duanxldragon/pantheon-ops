package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/gorm"
)

func commandFixture(t *testing.T, db *gorm.DB, svc *DeployService, status string) *TaskResponse {
	t.Helper()
	scopeID, hostID := seedDeployReliabilityFixture(t, db, "command-"+status, "10.40.9.2")
	task := DeployTask{Name: "command-" + status, BusinessScopeID: scopeID, Status: status, ExecutorType: ExecutorTypeManual, CreatedBy: "1"}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&DeployTaskHost{TaskID: task.ID, HostID: hostID, BusinessScopeID: scopeID, Status: TaskHostStatusFailed, ErrorMessage: "old failure"}).Error; err != nil {
		t.Fatal(err)
	}
	resp, err := svc.GetTask(task.ID, &common.DataScopeReq{IsAdmin: true})
	if err != nil {
		t.Fatal(err)
	}
	return resp
}
func TestTaskCommandStateMatrix(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	states := []string{TaskStatusDraft, TaskStatusPending, TaskStatusRunning, TaskStatusPauseRequested, TaskStatusPaused, TaskStatusCancelRequested, TaskStatusCanceled, TaskStatusFailed, TaskStatusSuccess}
	for _, action := range []string{"pause", "resume", "retry", "cancel"} {
		for _, status := range states {
			t.Run(action+"/"+status, func(t *testing.T) {
				task := commandFixture(t, db, svc, status)
				valid := action == "pause" && status == TaskStatusRunning || action == "resume" && status == TaskStatusPaused || action == "retry" && status == TaskStatusFailed || action == "cancel" && (status == TaskStatusPending || status == TaskStatusRunning || status == TaskStatusPauseRequested || status == TaskStatusPaused || status == TaskStatusCancelRequested)
				resp, err := svc.ControlTask(context.Background(), task.ID, "1", &common.DataScopeReq{IsAdmin: true}, action, TaskCommandRequest{StateVersion: task.StateVersion})
				if valid {
					if err != nil {
						t.Fatal(err)
					}
					if resp.StateVersion == task.StateVersion || resp.StateVersion == "" {
						t.Fatal("version did not advance")
					}
					if _, err = svc.ControlTask(context.Background(), task.ID, "1", nil, action, TaskCommandRequest{StateVersion: task.StateVersion}); !errors.Is(err, ErrTaskCommandConflict) {
						t.Fatalf("stale replay: %v", err)
					}
				} else {
					if err == nil || !deployTaskConflictError(err) {
						t.Fatalf("invalid state: %v", err)
					}
					var stored DeployTask
					db.First(&stored, task.ID)
					if stored.Status != status {
						t.Fatal("invalid command mutated task")
					}
				}
			})
		}
	}
}
func TestTaskCommandConcurrentABAAndLogIsolation(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	task := commandFixture(t, db, svc, TaskStatusPaused)
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.ControlTask(context.Background(), task.ID, "1", nil, "resume", TaskCommandRequest{StateVersion: task.StateVersion})
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrTaskCommandConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("success=%d conflict=%d", successes, conflicts)
	}
	// Restore all task facts, even updated_at, to A; the monotonic event survives.
	if err := db.Model(&DeployTask{}).Where("id = ?", task.ID).UpdateColumns(map[string]any{"status": TaskStatusPaused, "updated_at": task.UpdatedAt, "updated_by": ""}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ControlTask(context.Background(), task.ID, "1", nil, "resume", TaskCommandRequest{StateVersion: task.StateVersion}); !errors.Is(err, ErrTaskCommandConflict) {
		t.Fatalf("ABA accepted: %v", err)
	}
	before, err := svc.GetTask(task.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []taskEventInput{{taskID: task.ID, taskHostID: 1, eventType: "host.started"}, {taskID: task.ID, taskStepID: 1, eventType: "step.started"}} {
		if err := svc.appendTaskEvent(event); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&DeployTaskLogChunk{TaskID: task.ID, Sequence: 1, Content: "new output", Stream: "stdout"}).Error; err != nil {
		t.Fatal(err)
	}
	after, err := svc.GetTask(task.ID, nil)
	if err != nil || before.StateVersion != after.StateVersion {
		t.Fatalf("log/host/step invalidated commands: %v", err)
	}
	list, err := svc.ListTasks(TaskQuery{Page: 1, PageSize: 1}, nil)
	if err != nil || len(list.Items) != 1 || list.Items[0].StateVersion != after.StateVersion {
		t.Fatalf("list version: %v", err)
	}
}
func TestTaskCommandScopeAndContext(t *testing.T) {
	_, svc, task, _ := previewFixture(t, false)
	denied := &common.DataScopeReq{Mode: common.DataScopeModeSelf, UserID: 999}
	if _, err := svc.ControlTask(context.Background(), task.ID, "999", denied, "cancel", TaskCommandRequest{StateVersion: "forged"}); !isDeployTaskForbidden(err) {
		t.Fatalf("version leaked before scope: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.ControlTask(ctx, task.ID, "1", nil, "cancel", TaskCommandRequest{StateVersion: task.StateVersion}); !errors.Is(err, context.Canceled) {
		t.Fatalf("context: %v", err)
	}
	for _, value := range []string{"", "bad", string(make([]byte, 65))} {
		if _, err := svc.ControlTask(context.Background(), task.ID, "1", nil, "cancel", TaskCommandRequest{StateVersion: value}); !errors.Is(err, ErrTaskCommandVersionRequired) {
			t.Fatalf("invalid version: %v", err)
		}
	}
}
func TestTaskCommandFailureRollbackAndPostCommitCancel(t *testing.T) {
	for _, failure := range []string{"event", "host", "after-event", "none"} {
		t.Run(failure, func(t *testing.T) {
			db := setupDeployTestDB(t)
			svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
			task := commandFixture(t, db, svc, TaskStatusRunning)
			if err := db.Model(&DeployTaskHost{}).Where("task_id = ?", task.ID).Update("status", TaskHostStatusRunning).Error; err != nil {
				t.Fatal(err)
			}
			var canceled atomic.Int32
			var observed atomic.Bool
			svc.taskCancels.Store(task.ID, context.CancelFunc(func() {
				canceled.Add(1)
				var stored DeployTask
				if err := db.First(&stored, task.ID).Error; err == nil && stored.Status == TaskStatusCanceled {
					observed.Store(true)
				}
			}))
			injected := errors.New("injected " + failure)
			name := "command-failure-" + failure
			if failure == "event" || failure == "after-event" {
				position := "gorm:create"
				callback := db.Callback().Create().Before(position)
				if failure == "after-event" {
					callback = db.Callback().Create().After(position)
				}
				callback.Register(name, func(tx *gorm.DB) {
					if tx.Statement.Table == (DeployTaskEvent{}).TableName() {
						tx.AddError(injected)
					}
				})
				defer db.Callback().Create().Remove(name)
			}
			if failure == "host" {
				db.Callback().Update().Before("gorm:update").Register(name, func(tx *gorm.DB) {
					if tx.Statement.Table == (DeployTaskHost{}).TableName() {
						tx.AddError(injected)
					}
				})
				defer db.Callback().Update().Remove(name)
			}
			_, err := svc.ControlTask(context.Background(), task.ID, "1", nil, "cancel", TaskCommandRequest{StateVersion: task.StateVersion})
			var stored DeployTask
			db.First(&stored, task.ID)
			var host DeployTaskHost
			db.Where("task_id = ?", task.ID).First(&host)
			var count int64
			db.Model(&DeployTaskEvent{}).Where("task_id = ?", task.ID).Count(&count)
			if failure != "none" {
				if !errors.Is(err, injected) || stored.Status != TaskStatusRunning || host.Status != TaskHostStatusRunning || count != 0 || canceled.Load() != 0 {
					t.Fatalf("failed transaction escaped: %v %s %s events%d canceled%d", err, stored.Status, host.Status, count, canceled.Load())
				}
			} else {
				if err != nil || count != 1 || canceled.Load() != 1 || !observed.Load() {
					t.Fatalf("cancel before commit or omitted: %v", err)
				}
			}
		})
	}
}
func TestTaskCommandRetryHostFailureRollsBack(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	task := commandFixture(t, db, svc, TaskStatusFailed)
	injected := errors.New("host reset unavailable")
	db.Callback().Update().Before("gorm:update").Register("fail-reset", func(tx *gorm.DB) {
		if tx.Statement.Table == (DeployTaskHost{}).TableName() {
			tx.AddError(injected)
		}
	})
	defer db.Callback().Update().Remove("fail-reset")
	if _, err := svc.ControlTask(context.Background(), task.ID, "1", nil, "retry", TaskCommandRequest{StateVersion: task.StateVersion}); !errors.Is(err, injected) {
		t.Fatalf("swallowed reset: %v", err)
	}
	var stored DeployTask
	db.First(&stored, task.ID)
	var count int64
	db.Model(&DeployTaskEvent{}).Where("task_id = ?", task.ID).Count(&count)
	if stored.Status != TaskStatusFailed || count != 0 {
		t.Fatal("failed retry committed")
	}
}

func TestTaskCommandHandlersRequireViewedVersion(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewDeployHandler(svc)
	router.POST("/tasks/:id/pause", handler.PauseTask)
	router.POST("/tasks/:id/resume", handler.ResumeTask)
	router.POST("/tasks/:id/retry", handler.RetryTask)
	router.POST("/tasks/:id/cancel", handler.CancelTask)
	for _, action := range []string{"pause", "resume", "retry", "cancel"} {
		t.Run(action, func(t *testing.T) {
			state := TaskStatusRunning
			if action == "resume" {
				state = TaskStatusPaused
			}
			if action == "retry" {
				state = TaskStatusFailed
			}
			task := commandFixture(t, db, svc, state)
			for _, item := range []struct {
				body string
				code int
			}{{"", common.CodeParamInvalid}, {`{}`, common.CodeParamInvalid}, {`{"stateVersion":"bad"}`, common.CodeParamInvalid}, {`{"stateVersion":"` + strings.Repeat("0", 64) + `"}`, 409}, {`{"stateVersion":"` + task.StateVersion + `"}`, 200}, {`{"stateVersion":"` + task.StateVersion + `"}`, 409}} {
				recorder := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/tasks/%d/%s", task.ID, action), strings.NewReader(item.body))
				req.Header.Set("Content-Type", "application/json")
				req.ContentLength = -1 // also accepts legitimate chunked JSON
				router.ServeHTTP(recorder, req)
				var body struct{ Code int }
				if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Code != item.code {
					t.Fatalf("request %q got %d/%s want%d", item.body, recorder.Code, recorder.Body.String(), item.code)
				}
			}
		})
	}
}

func TestTaskCommandManualPauseResumeAndRetryDoNotDispatchSSH(t *testing.T) {
	db, svc, task, _ := previewFixture(t, false)
	svc.asyncExecution = true
	started, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	paused, err := svc.ControlTask(context.Background(), task.ID, "1", nil, "pause", TaskCommandRequest{StateVersion: started.StateVersion})
	if err != nil || paused.Status != TaskStatusPaused {
		t.Fatalf("manual pause stranded: %v", err)
	}
	resumed, err := svc.ControlTask(context.Background(), task.ID, "1", nil, "resume", TaskCommandRequest{StateVersion: paused.StateVersion})
	if err != nil || resumed.Status != TaskStatusRunning {
		t.Fatalf("manual resume: %v", err)
	}
	if _, err := svc.MarkHostResult(resumed.Hosts[0].ID, MarkHostResultRequest{Status: TaskHostStatusFailed, ErrorMessage: "manual failure"}, "1", nil); err != nil {
		t.Fatal(err)
	}
	failed, err := svc.GetTask(task.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	retried, err := svc.ControlTask(context.Background(), task.ID, "1", nil, "retry", TaskCommandRequest{StateVersion: failed.StateVersion})
	if err != nil || retried.Status != TaskStatusRunning {
		t.Fatalf("manual retry: %v", err)
	}
	// Reporting a manual result immediately must not race a queued SSH worker.
	if _, err := svc.MarkHostResult(retried.Hosts[0].ID, MarkHostResultRequest{Status: TaskHostStatusSuccess}, "1", nil); err != nil {
		t.Fatal(err)
	}
	var attempts int64
	db.Model(&DeployTaskAttempt{}).Where("task_id = ?", task.ID).Count(&attempts)
	if attempts != 0 {
		t.Fatal("manual task dispatched remote attempts")
	}
}
