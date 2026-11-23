package deploy

import (
	"context"
	"strings"
	"testing"
)

// TestExecutorPanicIsolatedAndSettledAsFailure pins the worker panic boundary:
// a background attempt runs outside gin's recovery, so a panicking executor
// must be converted into a failed, visible host result instead of taking the
// whole server down.
func TestExecutorPanicIsolatedAndSettledAsFailure(t *testing.T) {
	db, svc, _, task, host := ownershipFixture(t)
	svc.SetExecutor(ownershipExecutorFunc(func(context.Context, DeployExecutionHostRequest) error {
		panic("executor exploded")
	}))
	// A nil settlement error means the panic was converted into a durable
	// failure rather than escaping; only persistence errors are fatal here.
	failed, err := svc.runOwnedHost(context.Background(), task, host, StartTaskRequest{}, "test", "panic-worker", false)
	if err != nil {
		t.Fatalf("settle panic failure: %v", err)
	}
	if !failed {
		t.Fatal("panicking executor reported success")
	}
	var attempt DeployTaskAttempt
	if err := db.Where("task_id = ?", task.ID).Order("id DESC").First(&attempt).Error; err != nil {
		t.Fatal(err)
	}
	if attempt.Status != TaskHostStatusFailed || !strings.Contains(attempt.ErrorMessage, "business.deploy.executor.panic") || !strings.Contains(attempt.ErrorMessage, "executor exploded") {
		t.Fatalf("attempt status=%q message=%q, want failed with the panic preserved", attempt.Status, attempt.ErrorMessage)
	}
}
