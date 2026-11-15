package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

func TestExecutionWorkbenchMetadataAndLogPagination(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	task := DeployTask{Name: "workbench", Status: TaskStatusPending, CreatedBy: "1"}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	host := DeployTaskHost{TaskID: task.ID, HostID: 21, Hostname: "visible-host", Status: TaskHostStatusPending}
	if err := db.Create(&host).Error; err != nil {
		t.Fatal(err)
	}
	step := DeployTaskStep{TaskID: task.ID, TaskHostID: host.ID, StepKey: "install", StepOrder: 1, Status: StepStatusFailed, AttemptNo: 2, StateVersion: 7, ConditionJSON: datatypes.JSON(`{"password":"hidden-condition"}`)}
	if err := db.Create(&step).Error; err != nil {
		t.Fatal(err)
	}
	foreign := DeployTaskStep{TaskID: task.ID + 1, TaskHostID: host.ID, StepKey: "other-task", StepOrder: 1, Status: StepStatusSuccess, StateVersion: 1}
	if err := db.Create(&foreign).Error; err != nil {
		t.Fatal(err)
	}
	response, err := svc.GetTask(task.ID, &common.DataScopeReq{IsAdmin: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Steps) != 1 || response.Steps[0].ID != step.ID || response.Steps[0].StateVersion != 7 || response.Steps[0].AttemptNo != 2 {
		t.Fatalf("wrong persisted metadata: %+v", response.Steps)
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "hidden-condition") || strings.Contains(string(encoded), "conditionJson") {
		t.Fatal("private execution condition entered the response")
	}
	var logs []DeployTaskLogChunk
	for i := 1; i <= 501; i++ {
		logs = append(logs, DeployTaskLogChunk{TaskID: task.ID, TaskHostID: host.ID, TaskStepID: step.ID, Sequence: uint64(i), Stream: "stdout", Content: "sample"})
	}
	if err := db.Create(&logs).Error; err != nil {
		t.Fatal(err)
	}
	first, err := svc.loadLogChunks(task.ID, 0, taskLogFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 500 || first[0].Sequence != 1 || first[499].Sequence != 500 {
		t.Fatalf("unbounded or reordered log page: %d", len(first))
	}
	next, err := svc.loadLogChunks(task.ID, first[len(first)-1].Sequence, taskLogFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(next) != 1 || next[0].Sequence != 501 || next[0].TaskHostID != host.ID || next[0].TaskStepID != step.ID {
		t.Fatalf("lost next-page source/cursor: %+v", next)
	}
}

func TestManualTaskCancellationFinishesWithoutWorker(t *testing.T) {
	for _, status := range []string{TaskStatusRunning, TaskStatusCancelRequested} {
		t.Run(status, func(t *testing.T) {
			db := setupDeployTestDB(t)
			svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
			task := DeployTask{Name: "manual-cancel", Status: status, ExecutorType: ExecutorTypeManual, CreatedBy: "1"}
			if err := db.Create(&task).Error; err != nil {
				t.Fatal(err)
			}
			pending := DeployTaskHost{TaskID: task.ID, HostID: 21, Status: TaskHostStatusRunning}
			done := DeployTaskHost{TaskID: task.ID, HostID: 22, Status: TaskHostStatusSuccess}
			if err := db.Create(&pending).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Create(&done).Error; err != nil {
				t.Fatal(err)
			}
			response, err := svc.CancelTask(task.ID, "1", &common.DataScopeReq{IsAdmin: true})
			if err != nil {
				t.Fatal(err)
			}
			if response.Status != TaskStatusCanceled || response.FinishedAt == nil {
				t.Fatalf("manual task never reached terminal state: %+v", response)
			}
			if err := db.First(&pending, pending.ID).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.First(&done, done.ID).Error; err != nil {
				t.Fatal(err)
			}
			if pending.Status != TaskHostStatusSkipped || pending.FinishedAt == nil || done.Status != TaskHostStatusSuccess {
				t.Fatal("cancel did not stop unfinished hosts or changed successful hosts")
			}
			var events []DeployTaskEvent
			if err := db.Where("task_id = ? AND event_type = ?", task.ID, "task.canceled").Find(&events).Error; err != nil {
				t.Fatal(err)
			}
			if len(events) != 1 {
				t.Fatalf("terminal event count: %d", len(events))
			}
			if _, err := svc.CancelTask(task.ID, "1", nil); err == nil || err.Error() != errDeployTaskInvalidCancelState {
				t.Fatalf("second cancel bypassed state guard: %v", err)
			}
		})
	}
}

func TestWorkbenchLogSourceIsolationAndFirstError(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	task := DeployTask{Name: "filtered-workbench", Status: TaskStatusFailed, CreatedBy: "1"}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	hosts := []DeployTaskHost{{TaskID: task.ID, HostID: 1}, {TaskID: task.ID, HostID: 2}}
	if err := db.Create(&hosts).Error; err != nil {
		t.Fatal(err)
	}
	steps := []DeployTaskStep{
		{TaskID: task.ID, TaskHostID: hosts[0].ID, StepKey: "old", AttemptNo: 1},
		{TaskID: task.ID, TaskHostID: hosts[0].ID, StepKey: "current", AttemptNo: 2},
		{TaskID: task.ID, TaskHostID: hosts[1].ID, StepKey: "other-host", AttemptNo: 2},
		{TaskID: task.ID + 1, TaskHostID: hosts[0].ID, StepKey: "foreign-task", AttemptNo: 2},
	}
	if err := db.Create(&steps).Error; err != nil {
		t.Fatal(err)
	}
	logs := []DeployTaskLogChunk{
		{TaskID: task.ID, TaskHostID: hosts[0].ID, TaskStepID: steps[0].ID, AttemptNo: 1, Sequence: 1, Stream: "stderr", Content: "old attempt"},
		{TaskID: task.ID, TaskHostID: hosts[0].ID, TaskStepID: steps[1].ID, AttemptNo: 2, Sequence: 2, Stream: "stderr", Content: "first current error"},
		{TaskID: task.ID, TaskHostID: hosts[0].ID, TaskStepID: steps[3].ID, AttemptNo: 2, Sequence: 3, Stream: "stderr", Content: "foreign step"},
		{TaskID: task.ID, TaskHostID: hosts[0].ID, TaskStepID: steps[2].ID, AttemptNo: 2, Sequence: 4, Stream: "stderr", Content: "wrong host step"},
	}
	for i := 5; i < 1105; i++ {
		logs = append(logs, DeployTaskLogChunk{TaskID: task.ID, TaskHostID: hosts[1].ID, TaskStepID: steps[2].ID, AttemptNo: 2, Sequence: uint64(i), Stream: "stdout", Content: "other host noise"})
	}
	logs = append(logs, DeployTaskLogChunk{TaskID: task.ID, TaskHostID: hosts[0].ID, TaskStepID: steps[1].ID, AttemptNo: 2, Sequence: 1105, Stream: "stderr", Content: "last current error"})
	if err := db.CreateInBatches(&logs, 200).Error; err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	handler := NewDeployHandler(svc)
	call := func(query string, scope *common.DataScopeReq, canceled bool) (int, []TaskLogItem) {
		router := gin.New()
		router.Use(func(c *gin.Context) { c.Set(common.DataScopeContextKey, scope) })
		router.GET("/tasks/:id/logs", handler.ListTaskLogs)
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tasks/%d/logs?%s", task.ID, query), nil)
		if canceled {
			ctx, cancel := context.WithCancel(req.Context())
			cancel()
			req = req.WithContext(ctx)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		var payload struct {
			Code int `json:"code"`
			Data struct {
				Items []TaskLogItem `json:"items"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		return payload.Code, payload.Data.Items
	}
	admin := &common.DataScopeReq{IsAdmin: true}
	source := fmt.Sprintf("hostId=%d&attemptNo=2", hosts[0].ID)
	t.Run("host-attempt", func(t *testing.T) {
		code, items := call(source, admin, false)
		if code != 200 || len(items) != 2 || items[0].Sequence != 2 || items[1].Sequence != 1105 {
			t.Fatalf("mixed or lost source: %d %+v", code, items)
		}
	})
	t.Run("first-error-outside-tail", func(t *testing.T) {
		code, items := call(source+fmt.Sprintf("&stepId=%d&firstError=1", steps[1].ID), admin, false)
		if code != 200 || len(items) != 1 || items[0].Content != "first current error" {
			t.Fatalf("wrong first error: %d %+v", code, items)
		}
	})
	t.Run("sparse-cursor", func(t *testing.T) {
		code, items := call(source+"&after=2", admin, false)
		if code != 200 || len(items) != 1 || items[0].Sequence != 1105 {
			t.Fatalf("wrong cursor: %d %+v", code, items)
		}
	})
	t.Run("foreign-step", func(t *testing.T) {
		code, items := call(source+fmt.Sprintf("&stepId=%d", steps[3].ID), admin, false)
		if code != 200 || len(items) != 0 {
			t.Fatalf("foreign source leaked: %d %+v", code, items)
		}
	})
	t.Run("mismatched-host-step", func(t *testing.T) {
		code, items := call(source+fmt.Sprintf("&stepId=%d", steps[2].ID), admin, false)
		if code != 200 || len(items) != 0 {
			t.Fatalf("wrong source leaked: %d %+v", code, items)
		}
	})
	for _, query := range []string{"hostId=0", "stepId=-1", "attemptNo=0", "attemptNo=2147483648", "firstError=true", "after=bad", "hostId=1&hostId=2", "firstError=1&after=2"} {
		t.Run("invalid-"+query, func(t *testing.T) {
			code, _ := call(query, admin, false)
			if code != common.CodeParamInvalid {
				t.Fatalf("query accepted: %d", code)
			}
		})
	}
	t.Run("visibility-before-validation", func(t *testing.T) {
		code, items := call("hostId=bad", &common.DataScopeReq{Mode: "dept", DeptID: 99}, false)
		if code != 403 || len(items) != 0 {
			t.Fatalf("visibility bypassed: %d %+v", code, items)
		}
	})
	t.Run("request-cancellation", func(t *testing.T) {
		code, items := call(source, admin, true)
		if code == 200 || len(items) != 0 {
			t.Fatalf("canceled read returned logs: %d %+v", code, items)
		}
	})
}

func TestWorkbenchLogAttemptSurvivesCheckpointRetry(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	task := DeployTask{Name: "immutable-log-attempt", Status: TaskStatusRunning, CreatedBy: "1"}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	host := DeployTaskHost{TaskID: task.ID, HostID: 1}
	if err := db.Create(&host).Error; err != nil {
		t.Fatal(err)
	}
	step := DeployTaskStep{TaskID: task.ID, TaskHostID: host.ID, StepKey: "install", AttemptNo: 1}
	if err := db.Create(&step).Error; err != nil {
		t.Fatal(err)
	}
	legacy := DeployTaskLogChunk{TaskID: task.ID, TaskHostID: host.ID, TaskStepID: step.ID, Sequence: 1, Stream: "stderr", Content: "unattributed old output"}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	svc.WriteLog(task.ID, host.ID, "install", "stderr", "first recorded attempt")
	if err := db.Model(&step).Update("attempt_no", 2).Error; err != nil {
		t.Fatal(err)
	}
	svc.WriteLog(task.ID, host.ID, "install", "stderr", "second recorded attempt")
	for attempt, expected := range map[uint64]string{1: "first recorded attempt", 2: "second recorded attempt"} {
		rows, err := svc.loadLogChunks(task.ID, 0, taskLogFilter{hostID: host.ID, stepID: step.ID, attemptNo: attempt, firstError: true})
		if err != nil || len(rows) != 1 || rows[0].Content != expected || uint64(rows[0].AttemptNo) != attempt {
			t.Fatalf("checkpoint retry reassigned logs: attempt %d rows=%+v err=%v", attempt, rows, err)
		}
	}
	all, err := svc.loadLogChunks(task.ID, 0, taskLogFilter{hostID: host.ID})
	if err != nil || len(all) != 3 || all[0].AttemptNo != 0 {
		t.Fatalf("lost legacy output: %+v %v", all, err)
	}
	response, err := svc.GetTask(task.ID, &common.DataScopeReq{IsAdmin: true})
	if err != nil || len(response.LogAttempts) != 2 || response.LogAttempts[0].AttemptNo != 1 || response.LogAttempts[1].AttemptNo != 2 {
		t.Fatalf("wrong persisted attempt metadata: %+v %v", response, err)
	}
}
