package deploy

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func mixedReadFixture(t *testing.T) (*gorm.DB, *DeployService, DeployTask, []DeployTaskHost, []DeployTaskStep) {
	t.Helper()
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	task := DeployTask{Name: "mixed-read", Status: TaskStatusRunning}
	mustCreate := func(value any) {
		t.Helper()
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	mustCreate(&task)
	cmdbHosts := []deployTestHost{{Hostname: "visible-A", IP: "10.71.0.1", DeptID: 71}, {Hostname: "hidden-B", IP: "10.72.0.1", DeptID: 72}}
	mustCreate(&cmdbHosts)
	hosts := []DeployTaskHost{{TaskID: task.ID, HostID: cmdbHosts[0].ID}, {TaskID: task.ID, HostID: cmdbHosts[1].ID}}
	mustCreate(&hosts)
	steps := []DeployTaskStep{{TaskID: task.ID, TaskHostID: hosts[0].ID, StepKey: "A", AttemptNo: 1}, {TaskID: task.ID, TaskHostID: hosts[1].ID, StepKey: "B", AttemptNo: 1}}
	mustCreate(&steps)
	return db, svc, task, hosts, steps
}

func TestReviewMixedScopeLogsBeforeCapAndFirstError(t *testing.T) {
	db, svc, task, hosts, steps := mixedReadFixture(t)
	var logs []DeployTaskLogChunk
	for i := 1; i <= 501; i++ {
		logs = append(logs, DeployTaskLogChunk{TaskID: task.ID, TaskHostID: hosts[1].ID, TaskStepID: steps[1].ID, AttemptNo: 1, Sequence: uint64(i), Stream: "stderr", Content: "hidden B"})
	}
	for i := 502; i <= 1002; i++ {
		logs = append(logs, DeployTaskLogChunk{TaskID: task.ID, TaskHostID: hosts[0].ID, TaskStepID: steps[0].ID, AttemptNo: 1, Sequence: uint64(i), Stream: "stderr", Content: "visible A"})
	}
	if err := db.CreateInBatches(&logs, 200).Error; err != nil {
		t.Fatal(err)
	}
	scope := &common.DataScopeReq{Mode: "dept", DeptID: 71}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set(common.DataScopeContextKey, scope) })
	router.GET("/tasks/:id/logs", NewDeployHandler(svc).ListTaskLogs)
	call := func(query string) (int, []TaskLogItem) {
		t.Helper()
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", fmt.Sprintf("/tasks/%d/logs?%s", task.ID, query), nil))
		var payload struct {
			Code int
			Data struct{ Items []TaskLogItem }
		}
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		return payload.Code, payload.Data.Items
	}
	code, items := call("")
	if code != 200 || len(items) != 500 || items[0].Sequence != 502 || items[499].Sequence != 1001 {
		t.Fatalf("authorization must precede cap: %d %+v", code, items)
	}
	code, items = call("after=1001")
	if code != 200 || len(items) != 1 || items[0].Sequence != 1002 {
		t.Fatalf("cursor: %d %+v", code, items)
	}
	code, items = call("firstError=1")
	if code != 200 || len(items) != 1 || items[0].Sequence != 502 {
		t.Fatalf("first visible error: %d %+v", code, items)
	}
	for _, query := range []string{fmt.Sprintf("hostId=%d", hosts[1].ID), fmt.Sprintf("stepId=%d", steps[1].ID), fmt.Sprintf("hostId=%d&stepId=%d", hosts[0].ID, steps[1].ID), "hostId=999999", "stepId=999999"} {
		code, items = call(query)
		if (code != 403 && code != 200) || len(items) != 0 {
			t.Fatalf("invisible/foreign selection %s: %d %+v", query, code, items)
		}
	}
}

func TestReviewConnectedStreamScopeShrinkAndRevoke(t *testing.T) {
	db, svc, task, hosts, steps := mixedReadFixture(t)
	// Events with inconsistent host/step associations must also be excluded.
	appendEvents := func() {
		t.Helper()
		for _, e := range []taskEventInput{{taskID: task.ID, taskHostID: hosts[0].ID, taskStepID: steps[0].ID, eventType: "step.A", reason: "visible A"}, {taskID: task.ID, taskHostID: hosts[1].ID, taskStepID: steps[1].ID, eventType: "step.B", reason: "hidden B"}, {taskID: task.ID, taskHostID: hosts[0].ID, taskStepID: steps[1].ID, eventType: "step.mismatch", reason: "hidden B"}, {taskID: task.ID, eventType: "task.changed", reason: `{"hosts":["hidden B"]}`}} {
			if err := svc.appendTaskEvent(e); err != nil {
				t.Fatal(err)
			}
		}
	}
	appendEvents()
	// Initially both hosts are visible; shrink the live CMDB scope after opening.
	if err := db.Model(&deployTestHost{}).Where("id = ?", hosts[1].HostID).Update("dept_id", 71).Error; err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(common.DataScopeContextKey, &common.DataScopeReq{Mode: "dept", DeptID: 71})
	})
	router.GET("/tasks/:id/events/stream", NewDeployHandler(svc).StreamTaskEvents)
	server := httptest.NewServer(router)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/tasks/%d/events/stream", server.URL, task.ID), nil)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	scanner := bufio.NewScanner(response.Body)
	readBatch := func(wanted string) string {
		t.Helper()
		var output strings.Builder
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line + "\n")
			if strings.Contains(line, wanted) {
				return output.String()
			}
		}
		t.Fatalf("missing %s: %s %v", wanted, output.String(), scanner.Err())
		return ""
	}
	readBatch("task.changed")
	if err := db.Model(&deployTestHost{}).Where("id = ?", hosts[1].HostID).Update("dept_id", 72).Error; err != nil {
		t.Fatal(err)
	}
	appendEvents()
	output := readBatch("task.changed")
	if strings.Contains(output, "hidden B") || strings.Contains(output, "step.B") || strings.Contains(output, "step.mismatch") {
		t.Fatalf("partial shrink leaked: %s", output)
	}
	if err := db.Model(&deployTestHost{}).Where("id = ?", hosts[0].HostID).Update("dept_id", 72).Error; err != nil {
		t.Fatal(err)
	}
	appendEvents()
	var denied strings.Builder
	for scanner.Scan() {
		denied.WriteString(scanner.Text() + "\n")
	}
	if scanner.Err() != nil {
		t.Fatalf("stream did not terminate after revoke: %v", scanner.Err())
	}
	if !strings.Contains(denied.String(), "event: forbidden") || strings.Contains(denied.String(), "step.A") || strings.Contains(denied.String(), "hidden B") {
		t.Fatalf("revoke did not deny safely: %s", denied.String())
	}
}

func TestReviewEventReplayFiltersBeforeBound(t *testing.T) {
	db, svc, task, hosts, steps := mixedReadFixture(t)
	var events []DeployTaskEvent
	for i := 1; i <= 501; i++ {
		events = append(events, DeployTaskEvent{TaskID: task.ID, TaskHostID: hosts[1].ID, TaskStepID: steps[1].ID, Sequence: uint64(i), EventType: "step.hidden", Reason: "hidden"})
	}
	for i := 502; i <= 1002; i++ {
		events = append(events, DeployTaskEvent{TaskID: task.ID, TaskHostID: hosts[0].ID, TaskStepID: steps[0].ID, Sequence: uint64(i), EventType: "step.visible"})
	}
	if err := db.CreateInBatches(&events, 200).Error; err != nil {
		t.Fatal(err)
	}
	scope := &common.DataScopeReq{Mode: "dept", DeptID: 71}
	batch, err := svc.loadScopedTaskEvents(task.ID, 0, scope)
	if err != nil || len(batch) != 500 || batch[0].Sequence != 502 || batch[499].Sequence != 1001 {
		t.Fatalf("scope must precede event batch cap: count=%d err=%v", len(batch), err)
	}
	batch, err = svc.loadScopedTaskEvents(task.ID, 1001, scope)
	if err != nil || len(batch) != 1 || batch[0].Sequence != 1002 {
		t.Fatalf("sparse event replay: %+v %v", batch, err)
	}
	batch, err = svc.loadScopedTaskEvents(task.ID, 1002, scope)
	if err != nil || len(batch) != 0 {
		t.Fatalf("replay is not finite: %+v %v", batch, err)
	}
}

func TestReviewRetryExhaustedPreservesAllFacts(t *testing.T) {
	for _, versioned := range []bool{false, true} {
		t.Run(fmt.Sprint(versioned), func(t *testing.T) {
			db := setupDeployTestDB(t)
			svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
			view := commandFixture(t, db, svc, TaskStatusFailed)
			// Budget is an SSH worker contract; this fixture has a valid execution scope.
			if err := db.Model(&DeployTask{}).Where("id = ?", view.ID).Update("executor_type", ExecutorTypeSSH).Error; err != nil {
				t.Fatal(err)
			}
			step := DeployTaskStep{TaskID: view.ID, TaskHostID: view.Hosts[0].ID, StepKey: "failed-checkpoint", AttemptNo: 3, Status: StepStatusFailed, StateVersion: 9}
			if err := db.Create(&step).Error; err != nil {
				t.Fatal(err)
			}
			if err := appendStepCheckpointHistory(db, step); err != nil {
				t.Fatal(err)
			}
			view, err := svc.GetTask(view.ID, nil)
			if err != nil {
				t.Fatal(err)
			}
			hostID := view.Hosts[0].ID
			for i := 1; i <= 3; i++ {
				if err := db.Create(&DeployTaskAttempt{TaskID: view.ID, TaskHostID: hostID, AttemptNo: i, Status: "failed"}).Error; err != nil {
					t.Fatal(err)
				}
			}
			before, _ := json.Marshal(view)
			invoke := func() error {
				if versioned {
					_, err := svc.ControlTask(context.Background(), view.ID, "1", nil, "retry", TaskCommandRequest{StateVersion: view.StateVersion})
					return err
				}
				_, err := svc.RetryTask(view.ID, "1", nil)
				return err
			}
			for i := 0; i < 2; i++ {
				err := invoke()
				if err == nil || err.Error() != "business.deploy.attempt.exhausted" {
					t.Fatalf("exhausted retry admitted: %v", err)
				}
			}
			var wg sync.WaitGroup
			failures := make(chan error, 2)
			for i := 0; i < 2; i++ {
				wg.Add(1)
				go func() { defer wg.Done(); failures <- invoke() }()
			}
			wg.Wait()
			close(failures)
			for err := range failures {
				if err == nil || err.Error() != "business.deploy.attempt.exhausted" {
					t.Fatalf("concurrent exhaustion: %v", err)
				}
			}
			var attempts []DeployTaskAttempt
			if err := db.Where("task_host_id = ?", hostID).Order("attempt_no").Find(&attempts).Error; err != nil {
				t.Fatal(err)
			}
			if len(attempts) != 3 {
				t.Fatal("denial changed attempt history")
			}
			for i, attempt := range attempts {
				if attempt.AttemptNo != i+1 || attempt.Status != "failed" {
					t.Fatal("denial changed attempt facts")
				}
			}
			after, err := svc.GetTask(view.ID, nil)
			if err != nil {
				t.Fatal(err)
			}
			encoded, _ := json.Marshal(after)
			if string(before) != string(encoded) {
				t.Fatal("budget denial changed task/host/version/history")
			}
			var count int64
			db.Model(&DeployTaskEvent{}).Where("task_id = ?", view.ID).Count(&count)
			if count != 0 {
				t.Fatalf("denial appended event: %d", count)
			}
			// A paused claim does not consume one of the three attempts.
			if err := db.Model(&DeployTaskAttempt{}).Where("task_host_id = ? AND attempt_no = 3", hostID).Update("status", TaskStatusPaused).Error; err != nil {
				t.Fatal(err)
			}
			if err := invoke(); err != nil {
				t.Fatalf("paused attempt incorrectly charged: %v", err)
			}
			if attempt := svc.claimAttempt(view.ID, hostID, "review-worker"); attempt == nil || attempt.AttemptNo != 4 {
				t.Fatalf("history reset or no executable attempt: %+v", attempt)
			}
		})
	}
	if !deployTaskConflictError(errors.New("business.deploy.attempt.exhausted")) {
		t.Fatal("exhaustion lacks public conflict mapping")
	}
}
