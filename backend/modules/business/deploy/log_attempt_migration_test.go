package deploy

import (
	"context"
	_ "embed"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

//go:embed migrations/000034_deploy_log_attempt.up.sql
var logAttemptUp string

//go:embed migrations/000034_deploy_log_attempt.down.sql
var logAttemptDown string

func TestLogAttemptMigrationRetainsLegacyAndReplay(t *testing.T) {
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
	apply := func(script string) {
		t.Helper()
		var statements []string
		for _, line := range strings.Split(script, "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "--") {
				statements = append(statements, line)
			}
		}
		for _, statement := range strings.Split(strings.Join(statements, "\n"), ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, err := conn.ExecContext(ctx, statement); err != nil {
				t.Fatal(err)
			}
		}
	}
	// A system-only database has no business log table.
	apply(logAttemptUp)
	if _, err := conn.ExecContext(ctx, "CREATE TABLE biz_deploy_task_log_chunk (id BIGINT UNSIGNED PRIMARY KEY, content TEXT NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, "INSERT INTO biz_deploy_task_log_chunk (id,content) VALUES (1,'retained legacy output')"); err != nil {
		t.Fatal(err)
	}
	apply(logAttemptUp)
	var content string
	var attempt int
	if err := conn.QueryRowContext(ctx, "SELECT content,attempt_no FROM biz_deploy_task_log_chunk WHERE id=1").Scan(&content, &attempt); err != nil || attempt != 0 || content != "retained legacy output" {
		t.Fatalf("legacy rewrite: %q %d %v", content, attempt, err)
	}
	if _, err := conn.ExecContext(ctx, "INSERT INTO biz_deploy_task_log_chunk (id,content,attempt_no) VALUES (2,'recorded attempt',2)"); err != nil {
		t.Fatal(err)
	}
	apply(logAttemptUp)
	apply(logAttemptDown)
	apply(logAttemptUp)
	if err := conn.QueryRowContext(ctx, "SELECT attempt_no FROM biz_deploy_task_log_chunk WHERE id=2").Scan(&attempt); err != nil || attempt != 2 {
		t.Fatalf("replay/down removed metadata: %d %v", attempt, err)
	}
	var count int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM biz_deploy_task_log_chunk").Scan(&count); err != nil || count != 2 {
		t.Fatalf("retained rows: %d %v", count, err)
	}
}
