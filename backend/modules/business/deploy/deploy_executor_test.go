package deploy

import (
	"context"
	"testing"
	"time"
)

type contextDeployRunner struct{}

func (contextDeployRunner) RunScript(ctx context.Context, _ string) (string, string, error) {
	<-ctx.Done()
	return "", "", ctx.Err()
}

func (contextDeployRunner) Close() error { return nil }

func TestCredentialReferenceRequestRejectsInlineSecrets(t *testing.T) {
	if err := validateCredentialReferenceRequest(StartTaskRequest{CredentialRefID: 7, SSHPassword: "inline"}); err == nil {
		t.Fatal("expected inline credential material to be rejected")
	}
	if err := validateCredentialReferenceRequest(StartTaskRequest{CredentialRefID: 7}); err != nil {
		t.Fatalf("expected reference-only request to pass: %v", err)
	}
}

func TestRunDeployScriptPropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := runDeployScript(ctx, contextDeployRunner{}, "true")
	if err == nil || err != context.Canceled {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestNormalizeDeployExecutionTimeoutBounds(t *testing.T) {
	if got := normalizeDeployExecutionTimeout(0); got != defaultDeployExecutionTimeout {
		t.Fatalf("default timeout = %s", got)
	}
	if got := normalizeDeployExecutionTimeout(int((maxDeployExecutionTimeout / time.Second) + 1)); got != maxDeployExecutionTimeout {
		t.Fatalf("bounded timeout = %s", got)
	}
}
