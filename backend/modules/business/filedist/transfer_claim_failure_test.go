package filedist

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// This driver exercises the real GORM transaction path without a MySQL server.
// It models both possible persistent outcomes of a failed COMMIT response.
type transferClaimFailureState struct {
	mu                sync.Mutex
	stage             string
	targetStatus      string
	targetError       string
	injected          bool
	cancelOnFailure   bool
	terminalFailures  int
	terminalAttempts  int
	terminalNotice    chan struct{}
	discoveryFailures int
}

type transferClaimFailureConnector struct{ state *transferClaimFailureState }

func (c transferClaimFailureConnector) Connect(context.Context) (driver.Conn, error) {
	return &transferClaimFailureConn{state: c.state}, nil
}

func (c transferClaimFailureConnector) Driver() driver.Driver { return transferClaimFailureDriver{} }

type transferClaimFailureDriver struct{}

func (transferClaimFailureDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("fixture requires its connector")
}

type transferClaimFailureConn struct {
	state *transferClaimFailureState
	tx    *transferClaimFailureTx
}

func (c *transferClaimFailureConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepared statement")
}

func (c *transferClaimFailureConn) Close() error { return nil }

func (c *transferClaimFailureConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *transferClaimFailureConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	c.tx = &transferClaimFailureTx{conn: c}
	return c.tx, nil
}

func (c *transferClaimFailureConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	if !strings.Contains(query, "filedist_task") {
		return nil, errors.New("unexpected query outside parent claim")
	}
	if strings.HasPrefix(query, "SELECT `id` FROM") {
		if c.state.discoveryFailures > 0 {
			c.state.discoveryFailures--
			return nil, errors.New("injected private discovery SQL diagnostic")
		}
		return &transferClaimFailureRows{delivered: true}, nil
	}
	if c.state.stage == "parent-read" && !c.state.injected {
		return nil, c.failClaim()
	}
	return &transferClaimFailureRows{}, nil
}

func (c *transferClaimFailureConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	if !strings.Contains(query, "filedist_target") || !strings.HasPrefix(query, "UPDATE") {
		return nil, errors.New("unexpected statement outside target update")
	}
	var status string
	for _, arg := range args {
		if value, ok := arg.Value.(string); ok && (value == statusTransferring || value == statusFailed) {
			status = value
			break
		}
	}
	if status == statusTransferring && c.state.stage == "target-update" && !c.state.injected {
		return nil, c.failClaim()
	}
	if status == statusFailed {
		c.state.terminalAttempts++
		if c.state.terminalAttempts == 1 && c.state.terminalNotice != nil {
			close(c.state.terminalNotice)
		}
		if c.state.terminalFailures > 0 {
			c.state.terminalFailures--
			return nil, errors.New("injected terminal-write failure")
		}
	}
	if c.state.targetStatus == statusCancelled || c.state.targetStatus == statusCompleted || c.state.targetStatus == statusFailed {
		return driver.RowsAffected(0), nil
	}
	if c.tx == nil {
		return nil, errors.New("target update requires transaction")
	}
	c.tx.status = status
	if status == statusFailed && len(args) > 0 {
		c.tx.message, _ = args[0].Value.(string)
	}
	return driver.RowsAffected(1), nil
}

func (c *transferClaimFailureConn) failClaim() error {
	c.state.injected = true
	if c.state.cancelOnFailure {
		c.state.targetStatus = statusCancelled
	}
	return errors.New("injected private SQL diagnostic")
}

type transferClaimFailureTx struct {
	conn    *transferClaimFailureConn
	status  string
	message string
}

func (tx *transferClaimFailureTx) Commit() error {
	tx.conn.state.mu.Lock()
	defer tx.conn.state.mu.Unlock()
	defer func() { tx.conn.tx = nil }()
	if strings.HasPrefix(tx.conn.state.stage, "commit-") && tx.status == statusTransferring && !tx.conn.state.injected {
		if tx.conn.state.stage == "commit-applied" {
			tx.conn.state.targetStatus = tx.status
		}
		return tx.conn.failClaim()
	}
	if tx.status != "" {
		tx.conn.state.targetStatus = tx.status
		tx.conn.state.targetError = tx.message
	}
	return nil
}

func (tx *transferClaimFailureTx) Rollback() error {
	tx.conn.tx = nil
	return nil
}

type transferClaimFailureRows struct{ delivered bool }

func (*transferClaimFailureRows) Columns() []string { return []string{"id", "status"} }
func (*transferClaimFailureRows) Close() error      { return nil }
func (r *transferClaimFailureRows) Next(values []driver.Value) error {
	if r.delivered {
		return io.EOF
	}
	r.delivered = true
	values[0], values[1] = int64(10), statusRunning
	return nil
}

func TestDistributionClaimTransactionFailureSettlesConsumedWork(t *testing.T) {
	for _, stage := range []string{"parent-read", "target-update", "commit-applied", "commit-rolled-back"} {
		t.Run(stage, func(t *testing.T) {
			for _, initialStatus := range []string{statusPending, statusRetrying} {
				t.Run(initialStatus, func(t *testing.T) {
					state := &transferClaimFailureState{stage: stage, targetStatus: initialStatus}
					runTransferClaimFailureFixture(t, state)
					if state.targetStatus != statusFailed || state.terminalAttempts == 0 {
						t.Fatalf("consumed target stranded after SQL failure: status=%s terminal_attempts=%d", state.targetStatus, state.terminalAttempts)
					}
					if state.targetError != "business.filedist.task.transferFailed" {
						t.Fatalf("failure did not use canonical sanitized key: %q", state.targetError)
					}
				})
			}
		})
	}
}

func TestDistributionClaimFailurePreservesTerminalCancellation(t *testing.T) {
	state := &transferClaimFailureState{stage: "parent-read", targetStatus: statusPending, cancelOnFailure: true}
	runTransferClaimFailureFixture(t, state)
	if state.targetStatus != statusCancelled || state.targetError != "" {
		t.Fatal("claim-failure settlement resurrected cancelled target")
	}
}

func TestDistributionClaimFailureRetriesTerminalPersistence(t *testing.T) {
	state := &transferClaimFailureState{stage: "parent-read", targetStatus: statusPending, terminalFailures: 1}
	runTransferClaimFailureFixture(t, state)
	if state.targetStatus != statusFailed || state.terminalAttempts != 2 {
		t.Fatalf("failed settlement was silently dropped: status=%s attempts=%d", state.targetStatus, state.terminalAttempts)
	}
}

func runTransferClaimFailureFixture(t *testing.T, state *transferClaimFailureState) {
	t.Helper()
	engine := newTransferClaimFailureEngine(t, state)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	engine.processTask(ctx, &TransferTask{ID: 1, TaskID: 10, FileID: 20})
	if !state.injected {
		t.Fatal("fixture did not inject the requested transaction error")
	}
}

func newTransferClaimFailureEngine(t *testing.T, state *transferClaimFailureState) *TransferEngine {
	t.Helper()
	pool := sql.OpenDB(transferClaimFailureConnector{state: state})
	t.Cleanup(func() { _ = pool.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: pool, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	engine := NewTransferEngine(db, NewStorageService(db), 1)
	t.Cleanup(engine.Stop)
	engine.upload = func(context.Context, SFTPConfig, string, string) error {
		t.Fatal("claim failure reached remote IO")
		return nil
	}
	return engine
}

func TestDistributionClaimFailureShutdownStopsPersistence(t *testing.T) {
	state := &transferClaimFailureState{stage: "parent-read", targetStatus: statusPending, terminalFailures: 100, terminalNotice: make(chan struct{})}
	engine := newTransferClaimFailureEngine(t, state)
	done := make(chan struct{})
	go func() {
		defer close(done)
		engine.processTask(context.Background(), &TransferTask{ID: 1, TaskID: 10, FileID: 20})
	}()
	select {
	case <-state.terminalNotice:
	case <-time.After(time.Second):
		t.Fatal("failure persistence was never attempted")
	}
	engine.Stop()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not interrupt terminal-write retry wait")
	}
	if state.terminalAttempts != 1 {
		t.Fatal("stopped lifecycle performed another persistence retry")
	}
}

func TestDistributionClaimStaleTerminalDoesNotUpload(t *testing.T) {
	for _, terminal := range []string{statusCancelled, statusCompleted, statusFailed} {
		t.Run(terminal, func(t *testing.T) {
			state := &transferClaimFailureState{targetStatus: terminal}
			engine := newTransferClaimFailureEngine(t, state)
			engine.processTask(context.Background(), &TransferTask{ID: 1, TaskID: 10, FileID: 20})
			if state.targetStatus != terminal || state.terminalAttempts != 0 {
				t.Fatal("stale queue item changed a terminal target")
			}
		})
	}
}

type distributionLogCapture struct{ records chan slog.Record }

func (*distributionLogCapture) Enabled(context.Context, slog.Level) bool { return true }
func (h *distributionLogCapture) Handle(_ context.Context, record slog.Record) error {
	h.records <- record.Clone()
	return nil
}
func (h *distributionLogCapture) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *distributionLogCapture) WithGroup(string) slog.Handler      { return h }

func TestDistributionDispatcherReportsDiscoveryFailureAndRecovery(t *testing.T) {
	state := &transferClaimFailureState{targetStatus: statusPending, discoveryFailures: 1}
	engine := newTransferClaimFailureEngine(t, state)
	service := NewDistributionService(engine.db, nil, engine)
	logs := &distributionLogCapture{records: make(chan slog.Record, 4)}
	previous := slog.Default()
	slog.SetDefault(slog.New(logs))
	t.Cleanup(func() { slog.SetDefault(previous) })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go service.runDispatcher(ctx, done)
	defer func() {
		cancel()
		<-done
	}()
	for _, wanted := range []slog.Level{slog.LevelError, slog.LevelInfo} {
		select {
		case record := <-logs.records:
			if record.Level != wanted {
				t.Fatalf("missing failure/recovery transition: level=%v", record.Level)
			}
			var subsystem, errorType string
			record.Attrs(func(attr slog.Attr) bool {
				if strings.Contains(attr.Value.String(), "private") || strings.Contains(attr.Value.String(), "diagnostic") {
					t.Fatal("discovery log leaked raw SQL error")
				}
				if attr.Key == "subsystem" {
					subsystem = attr.Value.String()
				}
				if attr.Key == "error_type" {
					errorType = attr.Value.String()
				}
				return true
			})
			if subsystem != "business.filedist.scheduler" || (wanted == slog.LevelError && errorType == "") {
				t.Fatal("discovery log lacked subsystem or sanitized error class")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("discovery failure/recovery was silent")
		}
	}
}
