package deploy

import (
	"context"
	_ "embed"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"strings"
	"sync"
	"testing"
)

func TestLogSequenceIsUniqueWithinEachTask(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, nil)
	first := DeployTask{Name: "first", Status: TaskStatusDraft}
	second := DeployTask{Name: "second", Status: TaskStatusDraft}
	if err := db.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	svc.appendLogChunk(first.ID, 0, 0, 1, "stdout", "first-task-output")
	svc.appendLogChunk(second.ID, 0, 0, 1, "stdout", "second-task-output")
	var rows []DeployTaskLogChunk
	if err := db.Order("task_id").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].TaskID != first.ID || rows[1].TaskID != second.ID || rows[0].Sequence != 1 || rows[1].Sequence != 1 {
		t.Fatalf("cross-task output was lost: %+v", rows)
	}
	duplicate := DeployTaskLogChunk{TaskID: first.ID, Sequence: 1, Stream: "stdout", Content: "duplicate"}
	if err := db.Create(&duplicate).Error; err == nil {
		t.Fatal("duplicate within one task was accepted")
	}
}

//go:embed migrations/000036_deploy_log_sequence_scope.up.sql
var logSequenceUp string

//go:embed migrations/000036_deploy_log_sequence_scope.down.sql
var logSequenceDown string

func TestLogSequenceMigrationPreservesHistory(t *testing.T) {
	db := testmysql.Open(t)
	raw, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	conn, err := raw.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	apply := func(script string) error {
		var lines []string
		for _, line := range strings.Split(script, "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "--") {
				lines = append(lines, line)
			}
		}
		for _, statement := range strings.Split(strings.Join(lines, "\n"), ";") {
			if strings.TrimSpace(statement) != "" {
				if _, err := conn.ExecContext(context.Background(), statement); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := apply(logSequenceUp); err != nil {
		t.Fatal("empty database migration", err)
	}
	if _, err := conn.ExecContext(context.Background(), "CREATE TABLE biz_deploy_task_log_chunk (id BIGINT PRIMARY KEY, task_id BIGINT NOT NULL, sequence BIGINT NOT NULL, content TEXT, UNIQUE KEY uk_deploy_task_log_chunk_seq(sequence))"); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(context.Background(), "INSERT INTO biz_deploy_task_log_chunk VALUES (1,1,1,'retained')"); err != nil {
		t.Fatal(err)
	}
	if err := apply(logSequenceUp); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(context.Background(), "INSERT INTO biz_deploy_task_log_chunk VALUES (2,2,1,'second task')"); err != nil {
		t.Fatal("cross-task output remains blocked", err)
	}
	for _, script := range []string{logSequenceUp, logSequenceDown, logSequenceUp} {
		if err := apply(script); err != nil {
			t.Fatal("migration replay/down", err)
		}
	}
	var count int
	if err := conn.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM biz_deploy_task_log_chunk WHERE (id=1 AND content='retained') OR (id=2 AND content='second task')").Scan(&count); err != nil || count != 2 {
		t.Fatal("migration lost historical output", err, count)
	}
	if _, err := conn.ExecContext(context.Background(), "INSERT INTO biz_deploy_task_log_chunk VALUES (3,2,1,'duplicate')"); err == nil {
		t.Fatal("within-task duplicate was accepted after retained down")
	}
	if _, err := conn.ExecContext(context.Background(), "ALTER TABLE biz_deploy_task_log_chunk DROP INDEX uk_deploy_task_log_chunk_seq"); err != nil {
		t.Fatal(err)
	}
	if err := apply(logSequenceUp); err != nil {
		t.Fatal("missing index not restored", err)
	}
	if _, err := conn.ExecContext(context.Background(), "ALTER TABLE biz_deploy_task_log_chunk DROP INDEX uk_deploy_task_log_chunk_seq, ADD UNIQUE INDEX uk_deploy_task_log_chunk_seq(id)"); err != nil {
		t.Fatal(err)
	}
	if err := apply(logSequenceUp); err == nil {
		t.Fatal("unexpected legacy index shape was silently replaced")
	}
}

func TestConcurrentTaskLogSequencesStayScoped(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, nil)
	tasks := []DeployTask{{Name: "one", Status: TaskStatusDraft}, {Name: "two", Status: TaskStatusDraft}}
	if err := db.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	var writes sync.WaitGroup
	for _, task := range tasks {
		for i := 0; i < 8; i++ {
			writes.Add(1)
			go func(id uint64) {
				defer writes.Done()
				svc.appendLogChunk(id, 0, 0, 1, "stdout", "bounded-concurrent-output")
			}(task.ID)
		}
	}
	writes.Wait()
	for _, task := range tasks {
		var logs []DeployTaskLogChunk
		if err := db.Where("task_id = ?", task.ID).Order("sequence").Find(&logs).Error; err != nil {
			t.Fatal(err)
		}
		if len(logs) != 8 {
			t.Fatalf("task %d lost concurrent chunks: %d", task.ID, len(logs))
		}
		for i, log := range logs {
			if log.Sequence != uint64(i+1) {
				t.Fatal("nonconsecutive task-local sequence")
			}
		}
	}
}
