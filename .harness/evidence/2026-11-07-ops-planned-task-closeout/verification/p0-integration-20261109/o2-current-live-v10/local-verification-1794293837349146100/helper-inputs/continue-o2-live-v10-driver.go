// Acceptance-only driver. All execution uses the current public DeployService
// and its default SSH provider. No transport substitution or production DB.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	bs "github.com/duanxldragon/pantheon-base/backend/modules/business/bizscope"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	cg "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/group"
	ch "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/host"
	d "github.com/duanxldragon/pantheon-base/backend/modules/business/deploy"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/gin-gonic/gin"
	mysqlconfig "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/ssh"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type config struct {
	DSN, Key, User, AuthMode, Secret, Fingerprint, Identity, Case, Mode string
	TaskID, AttemptID                                                   uint64
	AttemptNo                                                           int
	LeaseExpiresUnix                                                    int64
}
type driver struct {
	c                          config
	db                         *gorm.DB
	svc                        *d.DeployService
	scope                      *common.DataScopeReq
	scopeID, hostID, credID    uint64
	positive, negative, failed int
	dir                        string
	diagnosticPhase            string
	diagnosticTaskID           uint64
	verifierConnections        int
	verifierCommandMu          sync.Mutex
	verifierMu                 sync.Mutex
	verifierClient             *ssh.Client
	verifierRaw                net.Conn
	verifierAttempted          bool
	verifierClosed             bool
	verifierError              error
	verifierCancel             context.CancelFunc
}

var errVerifierPinMismatch = errors.New("independent verifier pin mismatch")
var errVerifierClosed = errors.New("independent verifier closed")

func emit(v any) { _ = json.NewEncoder(os.Stdout).Encode(v) }
func safeDiagnosticError(e error) map[string]any {
	row := map[string]any{"errorType": fmt.Sprintf("%T", e)}
	var dbErr *mysqlconfig.MySQLError
	if errors.As(e, &dbErr) {
		row["mysqlNumber"] = dbErr.Number
		row["sqlState"] = string(dbErr.SQLState[:])
	}
	row["deadlineExceeded"] = errors.Is(e, context.DeadlineExceeded)
	row["contextCanceled"] = errors.Is(e, context.Canceled)
	frames := []map[string]any{}
	pcs := make([]uintptr, 10)
	count := runtime.Callers(2, pcs)
	iterator := runtime.CallersFrames(pcs[:count])
	for {
		frame, more := iterator.Next()
		if filepath.Base(frame.File) == "continue-o2-live-v10-driver.go" {
			frames = append(frames, map[string]any{"file": filepath.Base(frame.File), "line": frame.Line})
		}
		if !more {
			break
		}
	}
	row["driverFrames"] = frames
	return row
}
func must(e error) {
	if e != nil {
		row := safeDiagnosticError(e)
		row["phase"] = "operation-error"
		row["errorKey"] = key(e)
		emit(row)
		panic("operation failed; sensitive details suppressed")
	}
}
func (r *driver) check(name string, negative, ok bool) {
	kind := "positive"
	if negative {
		kind = "negative"
	}
	if ok {
		if negative {
			r.negative++
		} else {
			r.positive++
		}
	} else {
		r.failed++
	}
	emit(map[string]any{"check": name, "kind": kind, "passed": ok, "case": r.c.Case})
}
func key(e error) string {
	if e == nil {
		return ""
	}
	if regexp.MustCompile(`^(business|cmdbhost)\.[A-Za-z0-9_.]+$`).MatchString(e.Error()) {
		return e.Error()
	}
	var ex *ssh.ExitError
	if errors.As(e, &ex) {
		return fmt.Sprintf("ssh.exit.%d", ex.ExitStatus())
	}
	return "suppressed"
}
func (r *driver) remote(script string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
	defer cancel()
	return r.remoteContext(ctx, "115.190.251.143:22", script)
}

// One independently pinned authenticated connection per process. Commands are
// serialized because the socket deadline covers channel open as well as Run.
// A failed first handshake is cached; no command reconnects or replays.
func (r *driver) verifier(ctx context.Context, address string) (*ssh.Client, net.Conn, error) {
	r.verifierMu.Lock()
	if r.verifierClosed {
		r.verifierMu.Unlock()
		return nil, nil, errVerifierClosed
	}
	if r.verifierAttempted {
		defer r.verifierMu.Unlock()
		return r.verifierClient, r.verifierRaw, r.verifierError
	}
	r.verifierAttempted = true
	r.verifierConnections++
	ctx, cancel := context.WithCancel(ctx)
	r.verifierCancel = cancel
	r.verifierMu.Unlock()
	defer cancel()
	emit(map[string]any{"phase": "v9-verifier-connection", "ordinal": r.verifierConnections, "operationPhase": r.diagnosticPhase})
	client, raw, err := r.connectVerifier(ctx, address)
	r.verifierMu.Lock()
	defer r.verifierMu.Unlock()
	r.verifierCancel = nil
	if r.verifierClosed {
		if client != nil {
			_ = client.Close()
		}
		return nil, nil, errVerifierClosed
	}
	r.verifierClient, r.verifierRaw, r.verifierError = client, raw, err
	return client, raw, err
}

func (r *driver) connectVerifier(ctx context.Context, address string) (*ssh.Client, net.Conn, error) {
	var auth ssh.AuthMethod
	if r.c.AuthMode == "password" {
		auth = ssh.Password(r.c.Secret)
	} else {
		signer, e := ssh.ParsePrivateKey([]byte(r.c.Secret))
		if e != nil {
			emitVerifierError("private-key-parse", e)
			return nil, nil, e
		}
		auth = ssh.PublicKeys(signer)
	}
	pinChecked, pinMatched := false, false
	pinCallbacks := 0
	cfg := &ssh.ClientConfig{User: r.c.User, Auth: []ssh.AuthMethod{auth}, HostKeyCallback: func(_ string, _ net.Addr, k ssh.PublicKey) error {
		pinCallbacks++
		pinChecked = true
		pinMatched = ssh.FingerprintSHA256(k) == r.c.Fingerprint
		if !pinMatched {
			return errVerifierPinMismatch
		}
		return nil
	}}
	raw, e := (&net.Dialer{Timeout: 8 * time.Second}).DialContext(ctx, "tcp", address)
	if e != nil {
		emitVerifierError("tcp-dial", e)
		return nil, nil, e
	}
	owned := false
	defer func() {
		if !owned {
			_ = raw.Close()
		}
	}()
	deadline, _ := ctx.Deadline()
	if e = raw.SetDeadline(deadline); e != nil {
		return nil, nil, e
	}
	stop := watchVerifierCancellation(ctx, raw)
	defer stop()
	connection, channels, requests, e := ssh.NewClientConn(raw, address, cfg)
	stop()
	emit(map[string]any{"phase": "independent-ssh-handshake", "pinChecked": pinChecked, "pinMatched": pinMatched, "pinCallbacks": pinCallbacks, "deadlineExceeded": errors.Is(verifierContextError(ctx), context.DeadlineExceeded)})
	if contextError := verifierContextError(ctx); contextError != nil {
		e = contextError
	}
	if e != nil {
		emitVerifierError("handshake-auth", e)
		return nil, nil, e
	}
	client := ssh.NewClient(connection, channels, requests)
	if e = raw.SetDeadline(time.Time{}); e != nil {
		_ = client.Close()
		return nil, nil, e
	}
	owned = true
	return client, raw, nil
}

func (r *driver) closeVerifier() {
	r.verifierMu.Lock()
	r.verifierClosed = true
	client, cancel := r.verifierClient, r.verifierCancel
	r.verifierMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if client != nil {
		_ = client.Close()
	}
}

// Wait for an already-running callback before reusing the transport. A prior
// command's cancellation must never close a later command's session.
func watchVerifierCancellation(ctx context.Context, raw net.Conn) func() {
	done := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { _ = raw.Close(); close(done) })
	var once sync.Once
	return func() {
		once.Do(func() {
			if !stop() {
				<-done
			}
		})
	}
}

func verifierContextError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// The socket can report its deadline before the context timer is scheduled.
	if deadline, ok := ctx.Deadline(); ok && !time.Now().Before(deadline) {
		return context.DeadlineExceeded
	}
	return nil
}

func (r *driver) remoteContext(ctx context.Context, address, script string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 18*time.Second)
	defer cancel()
	r.verifierCommandMu.Lock()
	defer r.verifierCommandMu.Unlock()
	if e := ctx.Err(); e != nil {
		return "", e
	}
	client, raw, e := r.verifier(ctx, address)
	if e != nil {
		return "", e
	}
	deadline, _ := ctx.Deadline()
	if e = raw.SetDeadline(deadline); e != nil {
		return "", e
	}
	defer raw.SetDeadline(time.Time{})
	// Closing the transport bounds stalled channel opens and session I/O too.
	// Cancellation is terminal for this connection, never a retry trigger.
	stop := watchVerifierCancellation(ctx, raw)
	defer stop()
	session, e := client.NewSession()
	if e != nil {
		if contextError := verifierContextError(ctx); contextError != nil {
			e = contextError
		}
		emitVerifierError("channel", e)
		return "", e
	}
	defer session.Close()
	session.Stdin = strings.NewReader(script)
	var out bytes.Buffer
	session.Stdout = &out
	e = session.Run("/bin/bash -se")
	stop()
	if contextError := verifierContextError(ctx); contextError != nil {
		e = contextError
	}
	event := map[string]any{"phase": "independent-ssh-command", "passed": e == nil, "deadlineExceeded": errors.Is(e, context.DeadlineExceeded)}
	var exit *ssh.ExitError
	if errors.As(e, &exit) {
		event["exitStatus"] = exit.ExitStatus()
	} else if e == nil {
		event["exitStatus"] = 0
	}
	emit(event)
	if e != nil {
		emitVerifierError("command", e)
	}
	return out.String(), e
}
func (r *driver) markerCommand(name string) string {
	if name != "started" && name != "done" && name != "gate" {
		panic("marker outside exact owned allowlist")
	}
	path := r.dir + "/" + name
	return "if test -f '" + path + "'; then test ! -L '" + path + "'; test \"$(cat '" + path + "')\" = '" + r.c.Identity + "'; printf present; else printf absent; fi\n"
}
func (r *driver) exists(name string) bool {
	r.observePhase("marker-"+name, r.diagnosticTaskID)
	out, e := r.remote(r.markerCommand(name))
	must(e)
	if out != "present" && out != "absent" {
		panic("independent VM verifier returned invalid marker state")
	}
	emit(map[string]any{"phase": "independent-marker", "marker": name, "present": out == "present", "ownershipChecked": true})
	return out == "present"
}
func (r *driver) waitFile(name string) {
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if r.exists(name) {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	panic("VM marker deadline")
}
func (r *driver) get(id uint64) *d.TaskResponse {
	t, e := r.svc.GetTask(id, r.scope)
	if e != nil {
		row := safeDiagnosticError(e)
		row["phase"] = "public-get-task-error"
		emit(row)
	}
	must(e)
	return t
}
func (r *driver) wait(id uint64, status string) *d.TaskResponse {
	deadline := time.Now().Add(35 * time.Second)
	for time.Now().Before(deadline) {
		t := r.get(id)
		if t.Status == status {
			return t
		}
		time.Sleep(120 * time.Millisecond)
	}
	emit(map[string]any{"phase": "status-deadline", "taskId": id, "expected": status, "actual": r.get(id).Status})
	panic("task state deadline")
}
func (r *driver) settled(id uint64) {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var n int64
		must(r.db.Model(&d.DeployTaskAttempt{}).Where("task_id = ? AND status = ?", id, "running").Count(&n).Error)
		if n == 0 {
			time.Sleep(200 * time.Millisecond)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	panic("worker attempt deadline")
}
func (r *driver) task(script string) *d.TaskResponse {
	p, e := r.svc.CreatePackage(d.CreatePackageRequest{Name: r.c.Identity, Version: "1.0.0", InstallCommand: "ulimit -v 65536\n" + script, UninstallCommand: "test \"$(cat '" + r.dir + "/.owner')\" = '" + r.c.Identity + "'\nrm -f -- '" + r.dir + "/done'\nprintf 'o2-inverse o2-stdout\\n'\nprintf 'o2-stderr\\n' >&2\n", Status: d.PackageStatusEnabled}, "o2-live")
	must(e)
	t, e := r.svc.CreateTask(d.CreateTaskRequest{Name: r.c.Identity, PackageID: p.ID, BusinessScopeID: r.scopeID, TargetType: d.TargetTypeHost, TargetIDs: []uint64{r.hostID}, ExecutorType: d.ExecutorTypeSSH}, "o2-live", r.scope)
	must(e)
	r.diagnosticTaskID = t.ID
	emit(map[string]any{"phase": "task-created", "taskId": t.ID, "packageId": p.ID, "credentialId": r.credID})
	return t
}
func (r *driver) start(t *d.TaskResponse, async bool) (*d.TaskResponse, error) {
	r.observePhase("public-reviewed-start", t.ID)
	r.svc.SetAsyncExecution(async)
	req := d.StartTaskRequest{CredentialRefID: r.credID, HostFingerprint: r.c.Fingerprint, TimeoutSeconds: 18, IdempotencyKey: r.c.Identity, RequirePreview: true}
	p, e := r.svc.PreviewTaskContext(context.Background(), t.ID, req, "o2-live", r.scope)
	if e != nil {
		row := safeDiagnosticError(e)
		row["phase"] = "public-preview-error"
		emit(row)
		return nil, e
	}
	req.IntentVersion = p.IntentVersion
	response, e := r.svc.StartTaskContext(context.Background(), t.ID, req, "o2-live", r.scope)
	if e != nil {
		row := safeDiagnosticError(e)
		row["phase"] = "public-start-error"
		emit(row)
	}
	emit(map[string]any{"phase": "v8-product-start-category", "errorCategory": v8ErrorCategory(e), "taskId": t.ID})
	return response, e
}
func (r *driver) control(id uint64, action, version string, scope *common.DataScopeReq) (*d.TaskResponse, error) {
	r.observePhase("public-control-"+action, id)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	t, e := r.svc.ControlTask(ctx, id, "o2-live", scope, action, d.TaskCommandRequest{StateVersion: version})
	emit(map[string]any{"phase": "control", "taskId": id, "action": action, "errorKey": key(e)})
	return t, e
}
func (r *driver) currentControl(id uint64, action string) (*d.TaskResponse, error) {
	return r.control(id, action, r.get(id).StateVersion, r.scope)
}
func (r *driver) projection(id uint64, kind, query string, scope *common.DataScopeReq) (int, []map[string]any) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/tasks/"+fmt.Sprint(id)+"/"+kind+query, nil)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(id)}}
	c.Set(common.DataScopeContextKey, scope)
	h := d.NewDeployHandler(r.svc)
	if kind == "logs" {
		h.ListTaskLogs(c)
	} else {
		h.ListTaskEvents(c)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	must(json.Unmarshal(recorder.Body.Bytes(), &body))
	types := map[string]string{}
	if len(body.Data.Items) > 0 {
		for _, field := range []string{"sequence", "taskHostId", "taskStepId", "attemptNo"} {
			if value, ok := body.Data.Items[0][field]; ok {
				types[field] = fmt.Sprintf("%T", value)
			} else {
				types[field] = "missing"
			}
		}
	}
	emit(map[string]any{"phase": "public-projection-structure", "kind": kind, "httpStatus": recorder.Code, "responseCode": body.Code, "fieldTypes": types})
	return body.Code, body.Data.Items
}
func (r *driver) history(id uint64) {
	t := r.get(id)
	r.check("public current checkpoint and recorded safe history", false, len(t.Steps) > 0 && len(t.StepHistory) > 0 && !t.StepHistoryLimited)
	code, logs := r.projection(id, "logs", "", r.scope)
	stdout, stderr := false, false
	last := float64(0)
	good := code == common.CodeSuccess && len(logs) > 0
	for _, v := range logs {
		seq, _ := v["sequence"].(float64)
		content, _ := v["content"].(string)
		stream, _ := v["stream"].(string)
		stdout = stdout || (stream == "stdout" && strings.Contains(content, "o2-stdout"))
		stderr = stderr || (stream == "stderr" && strings.Contains(content, "o2-stderr"))
		good = good && seq > last && v["taskHostId"] != float64(0) && v["taskStepId"] != float64(0) && v["attemptNo"] != float64(0) && !strings.Contains(content, r.c.Secret)
		last = seq
	}
	r.check("public log sequence ownership attempts and secret exclusion", false, good)
	r.check("both actual VM streams persisted", false, stdout && stderr)
	tailCode, tail := r.projection(id, "logs", fmt.Sprintf("?after=%.0f", last), r.scope)
	r.check("log cursor excludes already delivered sequence", false, tailCode == common.CodeSuccess && last > 0 && len(tail) == 0)
	code, events := r.projection(id, "events", "", r.scope)
	last = 0
	good = code == common.CodeSuccess && len(events) > 0
	types := []string{}
	for _, v := range events {
		seq, _ := v["sequence"].(float64)
		good = good && seq > last
		last = seq
		typ, _ := v["eventType"].(string)
		types = append(types, typ)
	}
	r.check("public event sequence ordered across committed transitions", false, good)
	tailCode, tail = r.projection(id, "events", fmt.Sprintf("?after=%.0f", last), r.scope)
	r.check("event reconnect cursor no replay at latest sequence", false, tailCode == common.CodeSuccess && last > 0 && len(tail) == 0)
	var attempts []d.DeployTaskAttempt
	must(r.db.Where("task_id = ?", id).Order("attempt_no").Find(&attempts).Error)
	safe := []map[string]any{}
	for _, a := range attempts {
		safe = append(safe, map[string]any{"attemptNo": a.AttemptNo, "status": a.Status, "leaseReleased": a.LeaseExpiresAt == nil})
	}
	emit(map[string]any{"phase": "persisted-state", "taskId": id, "status": t.Status, "finished": t.FinishedAt != nil, "steps": t.Steps, "history": t.StepHistory, "logCount": len(logs), "eventTypes": types, "attempts": safe})
}
func (r *driver) child(id uint64, mode string) {
	r.observePhase("fresh-child-"+mode, id)
	executable, e := os.Executable()
	must(e)
	cfg := r.c
	cfg.Mode = mode
	cfg.TaskID = id
	raw, e := json.Marshal(cfg)
	must(e)
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable)
	cmd.Stdin = bytes.NewReader(append(raw, '\n'))
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = nil
	e = cmd.Run()
	// Child output contains only allowlisted evidence. Never emit raw errors.
	scanner := bufio.NewScanner(&output)
	for scanner.Scan() {
		var event map[string]any
		if json.Unmarshal(scanner.Bytes(), &event) == nil {
			event["origin"] = "fresh-process"
			emit(event)
		}
	}
	must(scanner.Err())
	r.check("new OS process "+mode+" uses committed task state", false, e == nil)
}
func (r *driver) negatives(t *d.TaskResponse) {
	_, e := r.control(t.ID, "pause", "", r.scope)
	r.check("missing state version denied", true, errors.Is(e, d.ErrTaskCommandVersionRequired))
	denied := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 999, UserID: 999}
	_, e = r.control(t.ID, "cancel", "", denied)
	r.check("scope denial precedes malformed command", true, key(e) == "business.deploy.task.forbidden")
	code, _ := r.projection(t.ID, "logs", "?after=invalid", denied)
	r.check("public logs deny scope before invalid cursor", true, code == common.CodeForbidden)
	code, _ = r.projection(t.ID, "events", "?after=invalid", denied)
	r.check("public events deny scope before invalid cursor", true, code == common.CodeForbidden)
	for _, action := range []string{"pause", "resume", "retry", "cancel"} {
		_, e = r.currentControl(t.ID, action)
		r.check("terminal success rejects "+action, true, e != nil)
	}
}
func (r *driver) initialize() {
	must(r.svc.Migrate())
	b := bs.NewService(r.db)
	must(b.Migrate())
	must(ch.NewHostService(r.db, b).Migrate())
	must(r.db.AutoMigrate(&cg.Group{}))
	scope, e := b.Create(&bs.CreateBizScopeRequest{Code: r.c.Identity, Name: r.c.Identity, Environment: "acceptance", Status: "active", DeptID: 10}, r.scope)
	must(e)
	r.scopeID = scope.ID
	host, e := ch.NewHostService(r.db, b).Create(ch.CreateHostRequest{Hostname: r.c.Identity, IP: "115.190.251.143", SSHPort: 22, OS: "linux", BusinessScopeID: r.scopeID, DeptID: 10}, "o2-live")
	must(e)
	r.hostID = host.ID
	cred, e := r.svc.CreateCredential(d.CreateDeployCredentialRequest{Name: r.c.Identity, Username: r.c.User, AuthMode: r.c.AuthMode, Secret: r.c.Secret}, "o2-live")
	must(e)
	r.credID = cred.ID
}
func (r *driver) loop() string {
	return "printf '" + r.c.Identity + "' > '" + r.dir + "/started'\nprintf 'o2-stdout\\n'\nprintf 'o2-stderr\\n' >&2\nfor i in {1..80}; do\n if test -f '" + r.dir + "/gate'; then printf '" + r.c.Identity + "' > '" + r.dir + "/done'; exit 0; fi\n printf 'o2-progress\\n'; sleep 0.1\ndone\nexit 19\n"
}
func (r *driver) run() {
	if r.c.Mode == "inspect" {
		r.history(r.c.TaskID)
		return
	}
	if r.c.Mode == "resume" {
		r.svc.SetAsyncExecution(true)
		_, e := r.currentControl(r.c.TaskID, "resume")
		must(e)
		r.wait(r.c.TaskID, d.TaskStatusSuccess)
		r.settled(r.c.TaskID)
		r.history(r.c.TaskID)
		return
	}
	if r.c.Mode == "queue-probe" || r.c.Mode == "recover" {
		r.svc.SetAsyncExecution(true)
		var before []d.DeployTaskAttempt
		must(r.db.Where("task_id = ?", r.c.TaskID).Find(&before).Error)
		count, e := r.svc.ReconcileDeployQueue("o2-live-restarted")
		must(e)
		r.check("fresh process discovers exactly one isolated persisted queue task", false, count == 1)
		if r.c.Mode == "queue-probe" {
			time.Sleep(time.Second)
			var attempts []d.DeployTaskAttempt
			must(r.db.Where("task_id = ?", r.c.TaskID).Find(&attempts).Error)
			preserved := activeLeasePreserved(before, attempts, r.c, time.Now())
			r.check("fresh queue startup preserves the exact actual unexpired abandoned attempt lease without new claims", false, preserved)
			safe := []map[string]any{}
			for _, attempt := range attempts {
				safe = append(safe, map[string]any{"id": attempt.ID, "attemptNo": attempt.AttemptNo, "status": attempt.Status, "leasePresent": attempt.LeaseExpiresAt != nil, "isCrashedAttempt": attempt.ID == r.c.AttemptID})
			}
			emit(map[string]any{"phase": "active-lease-observation", "attempts": safe, "expectedAttemptId": r.c.AttemptID, "expectedAttemptNo": r.c.AttemptNo, "attemptCountUnchanged": len(before) == len(attempts), "leaseMutation": false})
			r.check("startup during active lease never duplicates VM completion", false, !r.exists("done"))
		} else {
			r.wait(r.c.TaskID, d.TaskStatusSuccess)
			r.settled(r.c.TaskID)
			r.history(r.c.TaskID)
			r.check("lease-expired actual process restart completes real VM execution", false, r.exists("done"))
			var attempts []d.DeployTaskAttempt
			must(r.db.Where("task_id = ?", r.c.TaskID).Find(&attempts).Error)
			expired, recovered := false, false
			for _, a := range attempts {
				expired = expired || (a.ID == r.c.AttemptID && a.Status == "retryable" && a.LeaseExpiresAt == nil)
				recovered = recovered || (a.AttemptNo > r.c.AttemptNo && a.Status == "success" && a.LeaseExpiresAt == nil)
			}
			r.check("recovery retains the actual expired claim and a later successful released claim", false, expired && recovered)
		}
		return
	}
	r.initialize()
	if r.c.Mode == "crash-start" {
		t := r.task(r.loop())
		_, e := r.start(t, true)
		must(e)
		r.waitFile("started")
		var attempt d.DeployTaskAttempt
		must(r.db.Where("task_id = ? AND status = ?", t.ID, "running").First(&attempt).Error)
		if attempt.LeaseExpiresAt == nil || !attempt.LeaseExpiresAt.After(time.Now()) {
			panic("no actual active lease for crash")
		}
		emit(map[string]any{"phase": "actual-process-crash-ready", "taskId": t.ID, "attemptId": attempt.ID, "attemptNo": attempt.AttemptNo, "leaseExpiresUnix": attempt.LeaseExpiresAt.Unix(), "taskStatus": r.get(t.ID).Status})
		// Deliberate abrupt process exit preserves the real durable lease.
		// Earlier failed attempts are retained; the probe follows this exact claim.
		r.closeVerifier()
		os.Exit(75)
	}
	switch r.c.Case {
	case "success":
		t := r.task("printf '" + r.c.Identity + "' > '" + r.dir + "/done'\nprintf 'o2-stdout\\n'\nprintf 'o2-stderr\\n' >&2\n")
		changed := "printf 'changed-definition\\n'\n"
		_, e := r.svc.UpdatePackage(t.PackageID, d.UpdatePackageRequest{InstallCommand: &changed}, "o2-live")
		r.check("referenced package definition cannot replace frozen intent", true, key(e) == "business.deploy.package.immutable")
		result, e := r.start(t, false)
		must(e)
		r.check("reviewed real SSH start success", false, result.Status == d.TaskStatusSuccess)
		r.check("independent VM marker verifies success", false, r.exists("done"))
		r.history(t.ID)
		r.child(t.ID, "inspect")
		r.negatives(result)
		inverse, e := r.svc.RollbackTask(context.Background(), t.ID, "o2-live")
		must(e)
		r.check("public frozen rollback uses uninstall action and links the original install", false, inverse.Action == d.TaskActionUninstall && inverse.SourceTaskID != nil && *inverse.SourceTaskID == t.ID)
		result, e = r.start(&d.TaskResponse{ID: inverse.ID}, false)
		must(e)
		r.check("frozen public rollback succeeds", false, result.Status == d.TaskStatusSuccess)
		r.check("independent VM rollback removed marker", false, !r.exists("done"))
		r.child(inverse.ID, "inspect")
	case "failure-retry":
		t := r.task("printf 'o2-stdout\\n'\nprintf 'o2-stderr\\n' >&2\nif ! test -f '" + r.dir + "/gate'; then exit 17; fi\nprintf '" + r.c.Identity + "' > '" + r.dir + "/done'\n")
		_, e := r.start(t, false)
		r.check("actual remote exit 17 makes task failed", true, e != nil && r.get(t.ID).Status == d.TaskStatusFailed)
		r.history(t.ID)
		r.child(t.ID, "inspect")
		inverse, e := r.svc.RollbackTask(context.Background(), t.ID, "o2-live")
		_ = inverse
		r.check("failed partial install inverse obeys host state policy", true, key(e) == "business.deploy.task.targetStatusMismatch")
		_, e = r.remote("printf '" + r.c.Identity + "' > '" + r.dir + "/gate'\n")
		must(e)
		old := r.get(t.ID).StateVersion
		r.svc.SetAsyncExecution(true)
		_, e = r.currentControl(t.ID, "retry")
		must(e)
		r.wait(t.ID, d.TaskStatusSuccess)
		r.settled(t.ID)
		r.check("retry actual VM marker", false, r.exists("done"))
		r.history(t.ID)
		history := r.get(t.ID).StepHistory
		first, second := false, false
		for _, h := range history {
			first = first || (h.AttemptNo == 1 && h.Status == d.StepStatusFailed)
			second = second || (h.AttemptNo >= 2 && h.Status == d.StepStatusSuccess)
		}
		r.check("retry retains failed prior and successful new step history", false, first && second)
		_, e = r.control(t.ID, "retry", old, r.scope)
		r.check("stale retry version conflicts", true, errors.Is(e, d.ErrTaskCommandConflict))
		r.child(t.ID, "inspect")
	case "pause-resume", "cancel", "concurrent", "revoke-credential", "revoke-rotation", "revoke-source", "revoke-host":
		t := r.task(r.loop())
		_, e := r.start(t, true)
		must(e)
		r.waitFile("started")
		viewed := r.get(t.ID)
		if r.c.Case == "concurrent" {
			var wg sync.WaitGroup
			errs := make(chan error, 2)
			for i := 0; i < 2; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, e := r.svc.ControlTask(context.Background(), t.ID, "o2-live", r.scope, "pause", d.TaskCommandRequest{StateVersion: viewed.StateVersion})
					errs <- e
				}()
			}
			wg.Wait()
			close(errs)
			wins, conflicts := 0, 0
			for e := range errs {
				if e == nil {
					wins++
				}
				if errors.Is(e, d.ErrTaskCommandConflict) {
					conflicts++
				}
			}
			r.check("concurrent viewed commands exactly one commit one conflict", false, wins == 1 && conflicts == 1)
		} else {
			action := "pause"
			if r.c.Case == "cancel" {
				action = "cancel"
			}
			_, e = r.currentControl(t.ID, action)
			must(e)
		}
		expected := d.TaskStatusPaused
		if r.c.Case == "cancel" {
			expected = d.TaskStatusCanceled
		}
		result := r.wait(t.ID, expected)
		r.settled(t.ID)
		r.check("real running command acknowledged "+expected, false, result.Status == expected)
		_, e = r.control(t.ID, "cancel", viewed.StateVersion, r.scope)
		r.check("stale running view conflicts after control", true, errors.Is(e, d.ErrTaskCommandConflict))
		time.Sleep(8500 * time.Millisecond)
		r.check("independent VM interrupted attempt never wrote completion", false, !r.exists("done"))
		r.child(t.ID, "inspect")
		switch r.c.Case {
		case "pause-resume":
			_, e = r.remote("printf '" + r.c.Identity + "' > '" + r.dir + "/gate'\n")
			must(e)
			r.child(t.ID, "resume")
			r.check("fresh process resumed actual VM completion", false, r.exists("done"))
		case "revoke-credential", "revoke-rotation":
			req := d.UpdateDeployCredentialRequest{}
			if r.c.Case == "revoke-credential" {
				disabled := "disabled"
				req.Status = &disabled
			} else {
				req.Secret = &r.c.Secret
			}
			_, e = r.svc.UpdateCredential(r.credID, req, "o2-live")
			must(e)
			_, e = r.remote("printf '" + r.c.Identity + "' > '" + r.dir + "/gate'\n")
			must(e)
			r.svc.SetAsyncExecution(true)
			_, e = r.currentControl(t.ID, "resume")
			must(e)
			r.wait(t.ID, d.TaskStatusFailed)
			r.settled(t.ID)
			r.check("public credential revoke or rotation blocks recovered execution", true, !r.exists("done"))
			r.child(t.ID, "inspect")
		case "revoke-source":
			inactive := "inactive"
			updated, updateError := bs.NewService(r.db).Update(r.scopeID, &bs.UpdateBizScopeRequest{Status: &inactive}, r.scope)
			e = updateError
			must(e)
			r.check("public source revocation persists the contractual inactive state", false, updated.Status == inactive)
			_, activeError := bs.NewService(r.db).GetActive(context.Background(), r.scopeID, r.scope)
			r.check("owner capability denies the inactive business scope", true, activeError != nil)
			_, e = r.remote("printf '" + r.c.Identity + "' > '" + r.dir + "/gate'\n")
			must(e)
			r.svc.SetAsyncExecution(true)
			_, e = r.control(t.ID, "resume", result.StateVersion, r.scope)
			if e == nil {
				r.waitTerminal(t.ID)
				r.settled(t.ID)
			}
			r.check("inactive live business scope denies resumed VM side effect", true, e != nil && !r.exists("done"))
			emit(map[string]any{"phase": "source-revocation-observation", "resumeErrorKey": key(e), "taskStatus": r.get(t.ID).Status, "sourceStatus": inactive, "ownerActiveDenied": activeError != nil, "vmCompletionPresent": r.exists("done")})
		case "revoke-host":
			dept := uint64(999)
			_, e = ch.NewHostService(r.db, bs.NewService(r.db)).Update(r.hostID, ch.UpdateHostRequest{DeptID: &dept}, "o2-live", nil)
			must(e)
			_, e = r.svc.GetTask(t.ID, r.scope)
			r.check("current public host ownership change revokes task read", true, key(e) == "business.deploy.task.forbidden")
			_, e = r.control(t.ID, "resume", result.StateVersion, r.scope)
			r.check("current public host ownership change revokes viewed resume", true, key(e) == "business.deploy.task.forbidden")
			code, _ := r.projection(t.ID, "logs", "", r.scope)
			r.check("current public host ownership change revokes log projection", true, code == common.CodeForbidden)
		}
	case "hostkey-negative":
		t := r.task("printf 'o2-stdout\\n'\nprintf 'o2-stderr\\n' >&2\nprintf '" + r.c.Identity + "' > '" + r.dir + "/done'\n")
		req := d.StartTaskRequest{CredentialRefID: r.credID, HostFingerprint: "SHA256:incorrect-o2-acceptance-pin", TimeoutSeconds: 5, RequirePreview: true}
		preview, e := r.svc.PreviewTaskContext(context.Background(), t.ID, req, "o2-live", r.scope)
		must(e)
		req.IntentVersion = preview.IntentVersion
		r.observePhase("wrong-pin-product-start", t.ID)
		_, e = r.svc.StartTaskContext(context.Background(), t.ID, req, "o2-live", r.scope)
		emit(map[string]any{"phase": "v8-product-start-category", "taskId": t.ID, "errorCategory": v8ErrorCategory(e)})
		r.check("actual VM host-key mismatch denied by product SSH", true, key(e) == "business.deploy.task.sshHostKeyMismatch")
		r.check("untrusted host never executes marker script", true, !r.exists("done") && r.get(t.ID).Status == d.TaskStatusFailed)
	case "start-negative":
		t := r.task("printf 'o2-stdout\\n'\nprintf 'o2-stderr\\n' >&2\nprintf '" + r.c.Identity + "' > '" + r.dir + "/done'\n")
		req := d.StartTaskRequest{CredentialRefID: r.credID, HostFingerprint: r.c.Fingerprint, TimeoutSeconds: 5, RequirePreview: true}
		_, e := r.svc.StartTaskContext(context.Background(), t.ID, req, "o2-live", r.scope)
		r.check("public reviewed start requires preview", true, errors.Is(e, d.ErrTaskPreviewRequired))
		p, e := r.svc.PreviewTaskContext(context.Background(), t.ID, req, "o2-live", r.scope)
		must(e)
		disabled := "disabled"
		_, e = r.svc.UpdateCredential(r.credID, d.UpdateDeployCredentialRequest{Status: &disabled}, "o2-live")
		must(e)
		req.IntentVersion = p.IntentVersion
		_, e = r.svc.StartTaskContext(context.Background(), t.ID, req, "o2-live", r.scope)
		r.check("credential disabled after preview denies actual start", true, key(e) == "business.deploy.credential.inactive")
		r.check("denied start leaves task unstarted and actual VM untouched", true, r.get(t.ID).Status == t.Status && !r.exists("done"))
	default:
		panic("unknown acceptance case")
	}
}
func main() {
	r := &driver{}
	defer func() {
		r.closeVerifier()
		if failure := recover(); failure != nil {
			emit(map[string]any{"phase": "v8-failure-location", "operationPhase": r.diagnosticPhase, "taskId": r.diagnosticTaskID, "panicCategory": v8PanicCategory(failure), "verifierConnections": r.verifierConnections})
			r.failed++
			emit(map[string]any{"phase": "driver-failure", "error": "operation or deadline; secret details suppressed"})
		}
		emit(map[string]any{"phase": "driver-result", "positivePassed": r.positive, "negativePassed": r.negative, "failed": r.failed, "case": r.c.Case})
		if r.failed > 0 {
			os.Exit(1)
		}
	}()
	gin.SetMode(gin.ReleaseMode)
	must(json.NewDecoder(bufio.NewReader(os.Stdin)).Decode(&r.c))
	if !regexp.MustCompile(`^o2-live-[a-z0-9-]{8,70}$`).MatchString(r.c.Identity) {
		panic("invalid owned identity")
	}
	parsed, e := mysqlDriverConfig(r.c.DSN)
	must(e)
	if parsed != "ops_p0_o2_live_"+strings.ReplaceAll(strings.TrimPrefix(r.c.Identity, "o2-live-"), "-", "_") {
		panic("DB does not match this exact invocation identity")
	}
	must(os.Setenv("PANTHEON_DEPLOY_CREDENTIAL_KEY", r.c.Key))
	r.dir = "/opt/pantheon-p0-20261109/" + r.c.Identity
	r.db, e = gorm.Open(mysql.Open(r.c.DSN), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	must(e)
	sql, e := r.db.DB()
	must(e)
	defer sql.Close()
	sql.SetMaxOpenConns(8)
	sql.SetMaxIdleConns(3)
	r.scope = &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 10, UserID: 10}
	r.svc = d.NewDeployService(r.db, cmdb.NewDeployCMDBCapability(r.db))
	r.run()
}

// Refuse non-owned DBs before invoking any service migration or operation.
func mysqlDriverConfig(dsn string) (string, error) {
	cfg, err := mysqlconfig.ParseDSN(dsn)
	if err != nil || cfg.Net != "tcp" || cfg.Addr != "127.0.0.1:33316" {
		return "", errors.New("DB outside owned local acceptance instance")
	}
	db := cfg.DBName
	if !regexp.MustCompile(`^ops_p0_o2_live_[a-z0-9_]{8,90}$`).MatchString(db) {
		return "", errors.New("DB outside owned O2 identity")
	}
	return db, nil
}

// Only typed structural diagnostics leave the verifier; no addresses, credentials,
// remote stderr, error messages or provider values are included.
func emitVerifierError(stage string, err error) {
	row := map[string]any{"phase": "independent-ssh-error", "stage": stage, "errorType": fmt.Sprintf("%T", err)}
	var network *net.OpError
	if errors.As(err, &network) {
		row["netOperation"] = network.Op
		row["timeout"] = network.Timeout()
	}
	var exit *ssh.ExitError
	if errors.As(err, &exit) {
		row["exitStatus"] = exit.ExitStatus()
	}
	var channel *ssh.OpenChannelError
	if errors.As(err, &channel) {
		row["channelReason"] = int(channel.Reason)
	}
	chain := []string{}
	for e := err; e != nil && len(chain) < 8; e = errors.Unwrap(e) {
		chain = append(chain, fmt.Sprintf("%T", e))
	}
	row["errorChainTypes"] = chain
	row["pinMismatch"] = errors.Is(err, errVerifierPinMismatch)
	row["deadlineExceeded"] = errors.Is(err, context.DeadlineExceeded)
	row["authenticationFailed"] = strings.Contains(err.Error(), "unable to authenticate")
	row["unexpectedEOF"] = strings.Contains(err.Error(), "EOF")
	emit(row)
}

// Startup must preserve the actual live claim, including prior failed attempts.
func activeLeasePreserved(before, after []d.DeployTaskAttempt, c config, now time.Time) bool {
	if len(before) != len(after) || c.AttemptID == 0 || c.AttemptNo <= 0 {
		return false
	}
	expectedFound, active := false, 0
	prior := make(map[uint64]d.DeployTaskAttempt, len(before))
	for _, a := range before {
		prior[a.ID] = a
	}
	for _, a := range after {
		previous, exists := prior[a.ID]
		if !exists || previous.TaskID != a.TaskID || previous.AttemptNo != a.AttemptNo || previous.Status != a.Status {
			return false
		}
		if a.Status == "running" && a.LeaseExpiresAt != nil && a.LeaseExpiresAt.After(now) {
			active++
		}
		if a.ID == c.AttemptID {
			expectedFound = a.TaskID == c.TaskID && a.AttemptNo == c.AttemptNo && a.Status == "running" && a.LeaseExpiresAt != nil && a.LeaseExpiresAt.After(now) && a.LeaseExpiresAt.Unix() == c.LeaseExpiresUnix && previous.LeaseExpiresAt != nil && previous.LeaseExpiresAt.Equal(*a.LeaseExpiresAt)
		}
	}
	return expectedFound && active == 1
}
func (r *driver) waitTerminal(id uint64) {
	deadline := time.Now().Add(35 * time.Second)
	for time.Now().Before(deadline) {
		status := r.get(id).Status
		if status == d.TaskStatusSuccess || status == d.TaskStatusFailed {
			return
		}
		time.Sleep(120 * time.Millisecond)
	}
	emit(map[string]any{"phase": "status-deadline", "taskId": id, "expected": "terminal success or failure", "actual": r.get(id).Status})
	panic("task terminal deadline")
}

// Static categories only. Unknown provider errors and panic payloads never leave memory.
func v8ErrorCategory(err error) string {
	if err == nil {
		return "none"
	}
	if errors.Is(err, context.Canceled) {
		return "context-canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "context-deadline"
	}
	switch err.Error() {
	case "business.deploy.task.sshConnectFailed", "business.deploy.task.sshHostKeyMismatch",
		"business.deploy.task.sshAuthFailed", "business.deploy.task.scopeInvalid",
		"business.deploy.task.forbidden", "business.deploy.task.commandConflict":
		return err.Error()
	}
	return "noncanonical-suppressed"
}
func v8PanicCategory(value any) string {
	text, ok := value.(string)
	if !ok {
		return "unknown-suppressed"
	}
	switch text {
	case "operation failed; sensitive details suppressed":
		return "operation-error"
	case "VM marker deadline":
		return "marker-deadline"
	case "task state deadline", "task terminal deadline":
		return "task-state-deadline"
	case "worker attempt deadline":
		return "worker-attempt-deadline"
	case "no actual active lease for crash":
		return "crash-active-lease-missing"
	case "invalid owned identity", "DB does not match this exact invocation identity":
		return "ownership-refused"
	}
	return "unknown-suppressed"
}
func (r *driver) observePhase(phase string, taskID uint64) {
	r.diagnosticPhase, r.diagnosticTaskID = phase, taskID
	emit(map[string]any{"phase": "v8-operation-phase", "operationPhase": phase, "taskId": taskID})
}
