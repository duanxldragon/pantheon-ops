package deploy

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestStepHistoryKeepsPriorAttemptCheckpoint(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	task := DeployTask{Name: "history", Status: TaskStatusRunning, CreatedBy: "1"}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	host := DeployTaskHost{TaskID: task.ID, HostID: 1, Status: TaskHostStatusRunning}
	if err := db.Create(&host).Error; err != nil {
		t.Fatal(err)
	}
	step := DeployTaskStep{TaskID: task.ID, TaskHostID: host.ID, StepKey: "install", Status: StepStatusPending, StateVersion: 1, ConditionJSON: datatypes.JSON(`{"password":"private-condition"}`)}
	if err := db.Create(&step).Error; err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{StepStatusRunning, StepStatusFailed, StepStatusRunning, StepStatusSuccess} {
		if err := svc.updateTaskStepStatus(&step, status, "private-code", "private-error"); err != nil {
			t.Fatal(err)
		}
	}
	response, err := svc.GetTask(task.ID, &common.DataScopeReq{IsAdmin: true})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		StepHistory []TaskStepResponse `json:"stepHistory"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.StepHistory) != 2 {
		t.Fatalf("expected both recorded attempts, current=%+v history=%+v", response.Steps, payload.StepHistory)
	}
	found := map[int]TaskStepResponse{}
	for _, row := range payload.StepHistory {
		found[row.AttemptNo] = row
	}
	if found[1].Status != StepStatusFailed || found[1].StateVersion != 3 || found[2].Status != StepStatusSuccess || found[2].StateVersion != 5 {
		t.Fatalf("history was reassigned after retry: %+v", found)
	}
	for _, row := range payload.StepHistory {
		if row.ID != step.ID || row.TaskHostID != host.ID || row.StartedAt == nil || row.FinishedAt == nil {
			t.Fatalf("wrong checkpoint identity/times: %+v", row)
		}
	}
	var history struct {
		StepHistory json.RawMessage `json:"stepHistory"`
	}
	_ = json.Unmarshal(raw, &history)
	for _, private := range []string{"private-condition", "private-code", "private-error", "errorMessage", "conditionJson"} {
		if strings.Contains(string(history.StepHistory), private) {
			t.Fatalf("private field entered history: %s", private)
		}
	}
}

func historyFixture(t *testing.T) (*DeployService, DeployTask, DeployTaskHost, DeployTaskStep) {
	t.Helper()
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	task := DeployTask{Name: "history-fixture", Status: TaskStatusRunning, CreatedBy: "1"}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	host := DeployTaskHost{TaskID: task.ID, HostID: 1, Status: TaskHostStatusRunning}
	if err := db.Create(&host).Error; err != nil {
		t.Fatal(err)
	}
	step := DeployTaskStep{TaskID: task.ID, TaskHostID: host.ID, StepKey: "install", Status: StepStatusPending, StateVersion: 1}
	if err := db.Create(&step).Error; err != nil {
		t.Fatal(err)
	}
	return svc, task, host, step
}

func TestStepHistoryWriteFailureRollsBackCheckpoint(t *testing.T) {
	svc, _, _, step := historyFixture(t)
	name := "test:history-failure"
	if err := svc.db.Callback().Create().Before("gorm:create").Register(name, func(tx *gorm.DB) {
		if tx.Statement.Table == "biz_deploy_step_history" {
			tx.AddError(errors.New("injected history failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer svc.db.Callback().Create().Remove(name)
	if err := svc.updateTaskStepStatus(&step, StepStatusRunning, "", ""); err == nil {
		t.Fatal("history failure was ignored")
	}
	var persisted DeployTaskStep
	if err := svc.db.First(&persisted, step.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.Status != StepStatusPending || persisted.AttemptNo != 0 || persisted.StateVersion != 1 || persisted.StartedAt != nil || step.Status != StepStatusPending {
		t.Fatalf("partial checkpoint commit: %+v", persisted)
	}
	var count int64
	if err := svc.db.Model(&DeployStepHistory{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("history survived rollback: %d %v", count, err)
	}
}

func TestStepHistorySerializesConcurrentCheckpointWrites(t *testing.T) {
	svc, task, host, step := historyFixture(t)
	const writers = 12
	errorsCh := make(chan error, writers)
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			local := step
			errorsCh <- svc.updateTaskStepStatus(&local, StepStatusRunning, "", "")
		}()
	}
	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	var rows []DeployStepHistory
	if err := svc.db.Order("state_version ASC").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != writers {
		t.Fatalf("lost concurrent observations: %d", len(rows))
	}
	for i, row := range rows {
		if row.AttemptNo != i+1 || row.StateVersion != uint64(i+2) || row.TaskID != task.ID || row.TaskHostID != host.ID || row.StepID != step.ID {
			t.Fatalf("mislabeled concurrent checkpoint: %+v", row)
		}
	}
	var persisted DeployTaskStep
	if err := svc.db.First(&persisted, step.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.AttemptNo != writers || persisted.StateVersion != writers+1 {
		t.Fatalf("changed increment semantics: %+v", persisted)
	}
}

func TestStepHistoryPauseAndLegacyRemainHonest(t *testing.T) {
	svc, task, host, step := historyFixture(t)
	// Legacy state alone never creates earlier history.
	if err := svc.db.Model(&step).Updates(map[string]any{"attempt_no": 5, "status": StepStatusFailed}).Error; err != nil {
		t.Fatal(err)
	}
	rows, limited, err := svc.loadStepHistory(task.ID, []DeployTaskHost{host})
	if err != nil || limited || len(rows) != 0 {
		t.Fatalf("fabricated legacy history: %+v %v", rows, err)
	}
	for _, status := range []string{StepStatusRunning, StepStatusPending, StepStatusRunning} {
		if err := svc.updateTaskStepStatus(&step, status, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	rows, _, err = svc.loadStepHistory(task.ID, []DeployTaskHost{host})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("unexpected history: %+v", rows)
	}
	found := map[int]TaskStepHistoryResponse{}
	for _, row := range rows {
		found[row.AttemptNo] = row
	}
	if found[6].Status != StepStatusPending || found[6].FinishedAt != nil || found[6].CapturedAt.IsZero() || found[7].Status != StepStatusRunning {
		t.Fatalf("pause was reconstructed as terminal state: %+v", found)
	}
	if _, ok := found[5]; ok {
		t.Fatal("unrecorded legacy attempt was backfilled")
	}
}

func TestStepHistoryVisibleHostAndStepOwnership(t *testing.T) {
	svc, task, visible, step := historyFixture(t)
	db := svc.db
	sourceHosts := []deployTestHost{{ID: 1, Hostname: "visible", IP: "10.90.0.1", DeptID: 9}, {ID: 2, Hostname: "hidden", IP: "10.90.0.2", DeptID: 99}}
	if err := db.Create(&sourceHosts).Error; err != nil {
		t.Fatal(err)
	}
	hidden := DeployTaskHost{TaskID: task.ID, HostID: 2}
	if err := db.Create(&hidden).Error; err != nil {
		t.Fatal(err)
	}
	hiddenStep := DeployTaskStep{TaskID: task.ID, TaskHostID: hidden.ID, StepKey: "private-host-step", Status: StepStatusPending, StateVersion: 1}
	if err := db.Create(&hiddenStep).Error; err != nil {
		t.Fatal(err)
	}
	for _, record := range []*DeployTaskStep{&step, &hiddenStep} {
		if err := svc.updateTaskStepStatus(record, StepStatusRunning, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	foreign := DeployTaskStep{TaskID: task.ID + 1, TaskHostID: visible.ID, StepKey: "foreign-step", StateVersion: 1}
	if err := db.Create(&foreign).Error; err != nil {
		t.Fatal(err)
	}
	malformed := DeployStepHistory{TaskID: task.ID, TaskHostID: visible.ID, StepID: foreign.ID, StepKey: "private-foreign-history", AttemptNo: 1, Status: StepStatusFailed, StateVersion: 2, CapturedAt: time.Now()}
	if err := db.Create(&malformed).Error; err != nil {
		t.Fatal(err)
	}
	response, err := svc.GetTask(task.ID, &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 9})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Hosts) != 1 || len(response.Steps) != 1 {
		t.Fatalf("mixed-scope task metadata leaked: hosts=%+v steps=%+v", response.Hosts, response.Steps)
	}
	if len(response.StepHistory) != 1 || response.StepHistory[0].ID != step.ID || response.StepHistory[0].TaskHostID != visible.ID {
		t.Fatalf("history scope leak: %+v", response.StepHistory)
	}
	if _, err := svc.GetTask(task.ID, &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 100}); err == nil {
		t.Fatal("denied task history was readable")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := svc.targetRequestService(canceled).loadStepHistory(task.ID, []DeployTaskHost{visible}); err == nil {
		t.Fatal("canceled history query returned data")
	}
}

func TestStepHistoryProjectionBudget(t *testing.T) {
	svc, task, host, _ := historyFixture(t)
	db := svc.db
	var steps []DeployTaskStep
	for i := 0; i <= stepHistoryResponseLimit; i++ {
		steps = append(steps, DeployTaskStep{TaskID: task.ID, TaskHostID: host.ID, StepKey: fmt.Sprintf("step-%d", i), StepOrder: i + 1, AttemptNo: 1, Status: StepStatusSuccess, StateVersion: 2})
	}
	if err := db.CreateInBatches(&steps, 100).Error; err != nil {
		t.Fatal(err)
	}
	records := make([]DeployStepHistory, 0, len(steps))
	for _, step := range steps {
		records = append(records, DeployStepHistory{TaskID: task.ID, TaskHostID: host.ID, StepID: step.ID, StepKey: step.StepKey, StepOrder: step.StepOrder, AttemptNo: 1, Status: StepStatusSuccess, StateVersion: 2, CapturedAt: time.Now()})
	}
	if err := db.CreateInBatches(&records, 100).Error; err != nil {
		t.Fatal(err)
	}
	rows, limited, err := svc.loadStepHistory(task.ID, []DeployTaskHost{host})
	if err != nil || !limited || len(rows) != stepHistoryResponseLimit {
		t.Fatalf("unbounded history: %d %v %v", len(rows), limited, err)
	}
}

//go:embed migrations/000035_deploy_step_history.up.sql
var stepHistoryUp string

//go:embed migrations/000035_deploy_step_history.down.sql
var stepHistoryDown string

func TestStepHistoryMigrationReplayAndRetainedDown(t *testing.T) {
	db := testmysql.Open(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	apply := func(script string) {
		t.Helper()
		var lines []string
		for _, line := range strings.Split(script, "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "--") {
				lines = append(lines, line)
			}
		}
		for _, stmt := range strings.Split(strings.Join(lines, "\n"), ";") {
			if strings.TrimSpace(stmt) != "" {
				if _, err := conn.ExecContext(context.Background(), stmt); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	if _, err := conn.ExecContext(context.Background(), "CREATE TABLE biz_deploy_task_step (id BIGINT PRIMARY KEY, attempt_no INT)"); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(context.Background(), "INSERT INTO biz_deploy_task_step VALUES (42,8)"); err != nil {
		t.Fatal(err)
	}
	apply(stepHistoryUp)
	var count int
	if err := conn.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM biz_deploy_step_history").Scan(&count); err != nil || count != 0 {
		t.Fatalf("legacy backfill: %d %v", count, err)
	}
	if _, err := conn.ExecContext(context.Background(), "INSERT INTO biz_deploy_step_history (task_id,task_host_id,step_id,step_key,step_order,attempt_no,status,state_version,captured_at) VALUES (1,2,42,'retained',1,9,'failed',2,NOW(3))"); err != nil {
		t.Fatal(err)
	}
	apply(stepHistoryUp)
	apply(stepHistoryDown)
	apply(stepHistoryUp)
	if err := conn.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM biz_deploy_step_history WHERE step_id=42 AND attempt_no=9 AND status='failed'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("history lost on replay/down: %d %v", count, err)
	}
	if _, err := conn.ExecContext(context.Background(), "INSERT INTO biz_deploy_step_history (task_id,task_host_id,step_id,step_key,step_order,attempt_no,status,state_version,captured_at) VALUES (1,2,42,'duplicate',1,9,'success',2,NOW(3))"); err == nil {
		t.Fatal("duplicate checkpoint version was accepted")
	}
}
