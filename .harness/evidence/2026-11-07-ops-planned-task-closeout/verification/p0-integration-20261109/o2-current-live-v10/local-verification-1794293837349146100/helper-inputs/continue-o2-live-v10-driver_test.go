package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	d "github.com/duanxldragon/pantheon-base/backend/modules/business/deploy"
	mysqlconfig "github.com/go-sql-driver/mysql"
)

func TestOriginalAcceptanceFunctionBodiesPreservedExceptVerifierRelease(t *testing.T) {
	// Parse the original and candidate, then compare every pre-existing function
	// except remote. Only the two explicit verifier-release calls may differ.
	set := token.NewFileSet()
	read := func(name string, candidate bool) map[string]string {
		file, err := parser.ParseFile(set, filepath.Join(filepath.Dir(driverSourcePath(t)), name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		functions := make(map[string]string)
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok || fn.Name.Name == "remote" {
				continue
			}
			if candidate && (fn.Name.Name == "run" || fn.Name.Name == "main") {
				ast.Inspect(fn.Body, func(node ast.Node) bool {
					if block, ok := node.(*ast.BlockStmt); ok {
						kept := block.List[:0]
						for _, statement := range block.List {
							if expression, ok := statement.(*ast.ExprStmt); ok {
								if call, ok := expression.X.(*ast.CallExpr); ok {
									if selector, ok := call.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "closeVerifier" {
										continue
									}
								}
							}
							kept = append(kept, statement)
						}
						block.List = kept
					}
					return true
				})
			}
			var output bytes.Buffer
			if err := format.Node(&output, set, fn); err != nil {
				t.Fatal(err)
			}
			text := strings.ReplaceAll(output.String(), "resume-o2-v8", "continue-o2-live-v10")
			var tokens scanner.Scanner
			tokens.Init(set.AddFile("function", -1, len(text)), []byte(text), nil, 0)
			var normalized strings.Builder
			for {
				_, kind, literal := tokens.Scan()
				if kind == token.EOF {
					break
				}
				fmt.Fprintf(&normalized, "%s:%q;", kind, literal)
			}
			functions[fn.Name.Name] = normalized.String()
		}
		return functions
	}
	original, candidate := read("resume-o2-v8-driver.go", false), read("continue-o2-live-v10-driver.go", true)
	for name, body := range original {
		if candidate[name] != body {
			t.Errorf("original acceptance function %s changed beyond verifier release", name)
		}
	}
}

func driverSourcePath(t *testing.T) string {
	t.Helper()
	// Named-file tests retain this absolute source location in their call frames.
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("named-file source location unavailable")
	}
	return file
}

func TestColdProbeTracksCrashedClaimAfterEarlierFailure(t *testing.T) {
	// Local deterministic test data only; no DB, VM, or persisted lease mutation.
	now := time.Unix(1000, 0)
	lease := now.Add(5 * time.Minute)
	rows := []d.DeployTaskAttempt{
		{ID: 1, TaskID: 9, AttemptNo: 1, Status: "failed"},
		{ID: 2, TaskID: 9, AttemptNo: 2, Status: "running", LeaseExpiresAt: &lease},
	}
	c := config{TaskID: 9, AttemptID: 2, AttemptNo: 2, LeaseExpiresUnix: lease.Unix()}
	if !activeLeasePreserved(rows, rows, c, now) {
		t.Fatal("prior failed attempt must not invalidate preservation of the live crashed claim")
	}
	t.Run("new claim is rejected", func(t *testing.T) {
		after := append(append([]d.DeployTaskAttempt{}, rows...), d.DeployTaskAttempt{ID: 3, TaskID: 9, AttemptNo: 3, Status: "running", LeaseExpiresAt: &lease})
		if activeLeasePreserved(rows, after, c, now) {
			t.Fatal("duplicate claim accepted")
		}
	})
	t.Run("rewritten lease is rejected", func(t *testing.T) {
		after := append([]d.DeployTaskAttempt{}, rows...)
		changed := lease.Add(time.Nanosecond)
		after[1].LeaseExpiresAt = &changed
		if activeLeasePreserved(rows, after, c, now) {
			t.Fatal("subsecond lease rewrite accepted")
		}
	})
	t.Run("different same-length claim is rejected", func(t *testing.T) {
		after := append([]d.DeployTaskAttempt{}, rows...)
		after[0].ID = 3
		if activeLeasePreserved(rows, after, c, now) {
			t.Fatal("lost history accepted")
		}
	})
	t.Run("expired claim is rejected", func(t *testing.T) {
		if activeLeasePreserved(rows, rows, c, lease) {
			t.Fatal("expired claim accepted as active")
		}
	})
	t.Run("released claim is rejected", func(t *testing.T) {
		after := append([]d.DeployTaskAttempt{}, rows...)
		after[1].Status, after[1].LeaseExpiresAt = "retryable", nil
		if activeLeasePreserved(rows, after, c, now) {
			t.Fatal("released claim accepted as preserved")
		}
	})
}

func TestTypedDiagnosticsExcludeSensitiveMessages(t *testing.T) {
	const secret = "local-only-sensitive-sentinel"
	errs := []error{fmt.Errorf("%s: %w", secret, errVerifierPinMismatch), context.DeadlineExceeded,
		&mysqlconfig.MySQLError{Number: 1045, SQLState: [5]byte{'2', '8', '0', '0', '0'}, Message: secret}, errors.New(secret)}
	for _, err := range errs {
		raw, marshalErr := json.Marshal(safeDiagnosticError(err))
		if marshalErr != nil || strings.Contains(string(raw), secret) || strings.Contains(key(err), secret) {
			t.Fatal("sensitive message leaked")
		}
	}
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	prior := os.Stdout
	os.Stdout = write
	emitVerifierError("handshake-auth", fmt.Errorf("%s: %w", secret, errVerifierPinMismatch))
	os.Stdout = prior
	_ = write.Close()
	raw, err := io.ReadAll(read)
	_ = read.Close()
	if err != nil || strings.Contains(string(raw), secret) {
		t.Fatal("verifier diagnostic leaked")
	}
	var row map[string]any
	if json.Unmarshal(raw, &row) != nil || row["pinMismatch"] != true || row["stage"] != "handshake-auth" {
		t.Fatal("typed pin mismatch missing")
	}
}

func TestActualVerifierMarkerShellAndUninstallSemantics(t *testing.T) {
	// Execute the same verifier shell locally; this is not VM acceptance.
	dir := t.TempDir()
	identity := "o2-live-local-marker-contract"
	r := &driver{dir: filepath.ToSlash(dir), c: config{Identity: identity}}
	bash := "D:/msys64/usr/bin/bash.exe"
	run := func(script string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, bash, "-se")
		cmd.Stdin = strings.NewReader(script)
		raw, err := cmd.Output()
		return string(raw), err
	}
	if out, err := run(r.markerCommand("done")); err != nil || out != "absent" {
		t.Fatalf("absent marker must be exit 0: %q %v", out, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "done"), []byte(identity), 0600); err != nil {
		t.Fatal(err)
	}
	if out, err := run(r.markerCommand("done")); err != nil || out != "present" {
		t.Fatalf("valid marker: %q %v", out, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "done"), []byte("foreign-owner"), 0600); err != nil {
		t.Fatal(err)
	}
	if out, err := run(r.markerCommand("done")); err == nil || out != "" {
		t.Fatal("wrong marker owner accepted")
	}
	if err := os.WriteFile(filepath.Join(dir, "done"), []byte(identity), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := run("rm -f -- '" + r.dir + "/done'\n"); err != nil {
		t.Fatal(err)
	}
	if out, err := run(r.markerCommand("done")); err != nil || out != "absent" {
		t.Fatal("install->uninstall must remove completion and verifier must exit 0")
	}
}

func TestV8CategoriesDoNotEchoUnknownPayloads(t *testing.T) {
	const sentinel = "business.deploy.task.secretShapedSentinel"
	if got := v8ErrorCategory(errors.New(sentinel)); got != "noncanonical-suppressed" {
		t.Fatal("unknown key-shaped error must be suppressed")
	}
	if got := v8PanicCategory(sentinel); got != "unknown-suppressed" {
		t.Fatal("unknown panic payload must be suppressed")
	}
	if v8ErrorCategory(errors.New("business.deploy.task.sshConnectFailed")) != "business.deploy.task.sshConnectFailed" {
		t.Fatal("connection category missing")
	}
	if v8ErrorCategory(context.DeadlineExceeded) != "context-deadline" || v8PanicCategory("VM marker deadline") != "marker-deadline" {
		t.Fatal("deadline categories missing")
	}
}
