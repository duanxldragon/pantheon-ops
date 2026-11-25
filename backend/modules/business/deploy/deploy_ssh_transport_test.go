package deploy

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	"golang.org/x/crypto/ssh"
)

// This fixture speaks real SSH on loopback and never invokes an OS command.
// Every key is ephemeral; the runner must use the fixture's pinned host key.
func startDeploySSHTransport(t *testing.T, serve func(*ssh.ServerConn, <-chan ssh.NewChannel)) (cmdbHostSnapshot, StartTaskRequest) {
	t.Helper()
	_, hostPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	hostSigner, err := ssh.NewSignerFromKey(hostPrivate)
	if err != nil {
		t.Fatal(err)
	}
	_, userPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	userSigner, err := ssh.NewSignerFromKey(userPrivate)
	if err != nil {
		t.Fatal(err)
	}
	privateBlock, err := ssh.MarshalPrivateKey(userPrivate, "loopback test only")
	if err != nil {
		t.Fatal(err)
	}
	config := &ssh.ServerConfig{
		PasswordCallback: func(meta ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if meta.User() == "fixture" && string(password) == "ephemeral-test-password" {
				return nil, nil
			}
			return nil, errors.New("fixture rejected password")
		},
		PublicKeyCallback: func(meta ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if meta.User() == "fixture" && string(key.Marshal()) == string(userSigner.PublicKey().Marshal()) {
				return nil, nil
			}
			return nil, errors.New("fixture rejected public key")
		},
	}
	config.AddHostKey(hostSigner)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var transport net.Conn
	stopped := false
	done := make(chan struct{})
	go func() {
		defer close(done)
		raw, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer raw.Close()
		mu.Lock()
		transport = raw
		closing := stopped
		mu.Unlock()
		if closing {
			return
		}
		conn, channels, requests, handshakeErr := ssh.NewServerConn(raw, config)
		if handshakeErr != nil {
			return
		}
		defer conn.Close()
		go ssh.DiscardRequests(requests)
		serve(conn, channels)
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		mu.Lock()
		stopped = true
		if transport != nil {
			_ = transport.Close()
		}
		mu.Unlock()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("loopback SSH fixture failed to stop")
		}
	})
	port := listener.Addr().(*net.TCPAddr).Port
	return cmdbHostSnapshot{IP: "127.0.0.1", SSHPort: port}, StartTaskRequest{
		SSHUser: "fixture", SSHPassword: "ephemeral-test-password",
		SSHPrivateKey:   string(pem.EncodeToMemory(privateBlock)),
		HostFingerprint: ssh.FingerprintSHA256(hostSigner.PublicKey()),
	}
}

type deploySSHTransportResult struct {
	stdout, stderr string
	err            error
}

func awaitDeploySSHTransport(t *testing.T, runner deploySSHRunner, done <-chan deploySSHTransportResult) deploySSHTransportResult {
	t.Helper()
	select {
	case result := <-done:
		return result
	case <-time.After(time.Second):
		t.Error("SSH cancellation did not finish within one second")
		_ = runner.Close()
		select {
		case result := <-done:
			return result
		case <-time.After(3 * time.Second):
			t.Fatal("SSH runner failed to stop after connection close")
		}
	}
	return deploySSHTransportResult{}
}

func serveDeploySSHScript(received chan<- []string, exitCode uint32) func(*ssh.ServerConn, <-chan ssh.NewChannel) {
	return func(conn *ssh.ServerConn, channels <-chan ssh.NewChannel) {
		newChannel, ok := <-channels
		if !ok {
			return
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			return
		}
		defer channel.Close()
		for request := range requests {
			if request.Type != "exec" {
				_ = request.Reply(false, nil)
				continue
			}
			var command struct{ Value string }
			if err := ssh.Unmarshal(request.Payload, &command); err != nil {
				return
			}
			_ = request.Reply(true, nil)
			script, err := io.ReadAll(channel)
			if err != nil {
				return
			}
			received <- []string{command.Value, string(script)}
			_, _ = io.WriteString(channel, "fixture stdout\n")
			_, _ = io.WriteString(channel.Stderr(), "fixture stderr\n")
			_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{exitCode}))
			_ = channel.Close()
			_ = conn.Wait() // The SSH server keeps the host connection alive.
			return
		}
	}
}

func TestDeploySSHTransportScriptAndExitStatus(t *testing.T) {
	for _, authMode := range []string{"password", "private_key"} {
		for _, exitCode := range []uint32{0, 17} {
			t.Run(fmt.Sprintf("%s/exit%d", authMode, exitCode), func(t *testing.T) {
				received := make(chan []string, 1)
				host, req := startDeploySSHTransport(t, serveDeploySSHScript(received, exitCode))
				req.AuthMode = authMode
				runner, err := newDeploySSHRunner(context.Background(), host, req)
				if err != nil {
					t.Fatal(err)
				}
				defer runner.Close()
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				stdout, stderr, err := runDeployScript(ctx, runner, "printf 'protocol fixture'\n")
				if stdout != "fixture stdout\n" || stderr != "fixture stderr\n" {
					t.Fatalf("output = %q / %q", stdout, stderr)
				}
				if exitCode == 0 && err != nil {
					t.Fatal(err)
				}
				if exitCode != 0 {
					var exitErr *ssh.ExitError
					if !errors.As(err, &exitErr) || exitErr.ExitStatus() != int(exitCode) {
						t.Fatalf("exit = %v", err)
					}
				}
				select {
				case got := <-received:
					if got[0] != "/bin/bash -se" || got[1] != "printf 'protocol fixture'\n" {
						t.Fatalf("command/stdin = %#v", got)
					}
				default:
					t.Fatal("script was not received")
				}
			})
		}
	}
}

func TestDeploySSHTransportRejectsChangedHostKey(t *testing.T) {
	host, req := startDeploySSHTransport(t, func(_ *ssh.ServerConn, _ <-chan ssh.NewChannel) {
		t.Error("untrusted host reached authenticated handler")
	})
	req.HostFingerprint = "SHA256:incorrect-fixture-key"
	runner, err := newDeploySSHRunner(context.Background(), host, req)
	if runner != nil {
		_ = runner.Close()
		t.Fatal("untrusted host returned a runner")
	}
	if err == nil || err.Error() != errDeployTaskSSHHostKeyMismatch {
		t.Fatalf("host-key error = %v", err)
	}
}

func TestDeploySSHTransportCancelDuringSessionOpen(t *testing.T) {
	opening := make(chan struct{})
	host, req := startDeploySSHTransport(t, func(conn *ssh.ServerConn, channels <-chan ssh.NewChannel) {
		if _, ok := <-channels; !ok {
			return
		}
		close(opening) // Deliberately withhold the channel-open response.
		_ = conn.Wait()
	})
	runner, err := newDeploySSHRunner(context.Background(), host, req)
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan deploySSHTransportResult, 1)
	go func() {
		stdout, stderr, runErr := runner.RunScript(ctx, "fixture")
		done <- deploySSHTransportResult{stdout, stderr, runErr}
	}()
	select {
	case <-opening:
	case <-time.After(3 * time.Second):
		t.Fatal("session was not requested")
	}
	cancel()
	result := awaitDeploySSHTransport(t, runner, done)
	if !errors.Is(result.err, context.Canceled) {
		t.Fatalf("cancel error = %v", result.err)
	}
}

func TestDeploySSHTransportCancelDuringOutput(t *testing.T) {
	writing := make(chan struct{})
	host, req := startDeploySSHTransport(t, func(_ *ssh.ServerConn, channels <-chan ssh.NewChannel) {
		newChannel, ok := <-channels
		if !ok {
			return
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			return
		}
		defer channel.Close()
		for request := range requests {
			if request.Type != "exec" {
				_ = request.Reply(false, nil)
				continue
			}
			_ = request.Reply(true, nil)
			if _, err := io.ReadAll(channel); err != nil {
				return
			}
			_, _ = io.WriteString(channel, "fixture stdout\n")
			_, _ = io.WriteString(channel.Stderr(), "fixture stderr\n")
			close(writing)
			chunk := strings.Repeat("bounded fixture output\n", 256)
			for {
				if _, err := io.WriteString(channel, chunk); err != nil {
					return
				}
				if _, err := io.WriteString(channel.Stderr(), chunk); err != nil {
					return
				}
			}
		}
	})
	runner, err := newDeploySSHRunner(context.Background(), host, req)
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan deploySSHTransportResult, 1)
	go func() {
		stdout, stderr, runErr := runner.RunScript(ctx, "fixture")
		done <- deploySSHTransportResult{stdout, stderr, runErr}
	}()
	select {
	case <-writing:
	case <-time.After(3 * time.Second):
		t.Fatal("output did not start")
	}
	cancel()
	result := awaitDeploySSHTransport(t, runner, done)
	if !errors.Is(result.err, context.Canceled) {
		t.Fatalf("cancel error = %v", result.err)
	}
}

func TestDeploySSHTransportPreCanceledContextDoesNotOpenSession(t *testing.T) {
	opened := make(chan struct{}, 1)
	host, req := startDeploySSHTransport(t, func(_ *ssh.ServerConn, channels <-chan ssh.NewChannel) {
		for newChannel := range channels {
			opened <- struct{}{}
			_ = newChannel.Reject(ssh.Prohibited, "pre-canceled fixture")
		}
	})
	runner, err := newDeploySSHRunner(context.Background(), host, req)
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = runner.RunScript(ctx, "fixture")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("cancel error = %v", err)
	}
	select {
	case <-opened:
		t.Error("pre-canceled context opened a remote session")
	default:
	}
}

// This case covers the production credential/provider/step/log persistence path,
// using an isolated real MySQL database and the actual loopback SSH client.
func TestDeploySSHTransportBusinessExecution(t *testing.T) {
	for _, exitCode := range []uint32{0, 17} {
		t.Run(fmt.Sprintf("exit%d", exitCode), func(t *testing.T) {
			db := setupDeployTestDB(t)
			svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
			t.Setenv(deployCredentialKeyEnv, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
			received := make(chan []string, 1)
			target, req := startDeploySSHTransport(t, serveDeploySSHScript(received, exitCode))
			credential, err := svc.CreateCredential(CreateDeployCredentialRequest{
				Name: "loopback-key", Username: req.SSHUser, AuthMode: "private_key", Secret: req.SSHPrivateKey,
			}, "1", nil)
			if err != nil {
				t.Fatal(err)
			}
			scopeID, hostID := seedDeployReliabilityFixture(t, db, "loopback-ssh", target.IP)
			if err := db.Model(&deployTestHost{}).Where("id = ?", hostID).Update("ssh_port", target.SSHPort).Error; err != nil {
				t.Fatal(err)
			}
			pkg, err := svc.CreatePackage(CreatePackageRequest{
				Name: "loopback-component", Version: "1.0.0", InstallCommand: "echo frozen-fixture", Status: PackageStatusEnabled,
			}, "1")
			if err != nil {
				t.Fatal(err)
			}
			task, err := svc.CreateTask(CreateTaskRequest{
				Name: "loopback-deploy", PackageID: pkg.ID, BusinessScopeID: scopeID,
				TargetType: TargetTypeHost, TargetIDs: []uint64{hostID}, ExecutorType: ExecutorTypeSSH,
			}, "1", nil)
			if err != nil {
				t.Fatal(err)
			}
			// A mutable definition cannot replace the task's frozen executable input.
			if err := db.Model(&DeployPackage{}).Where("id = ?", pkg.ID).Update("install_command", "echo changed-fixture").Error; err != nil {
				t.Fatal(err)
			}
			result, err := svc.StartTask(task.ID, StartTaskRequest{
				CredentialRefID: credential.ID, HostFingerprint: req.HostFingerprint, TimeoutSeconds: 5,
			}, "1", nil)
			if exitCode == 0 && err != nil {
				t.Fatal(err)
			}
			if exitCode != 0 {
				if err == nil || !strings.Contains(err.Error(), "status 17") {
					t.Fatalf("remote failure = %v", err)
				}
				result, err = svc.GetTask(task.ID, nil)
				if err != nil {
					t.Fatal(err)
				}
			}
			expectedTask, expectedHost, expectedStep := TaskStatusSuccess, TaskHostStatusSuccess, StepStatusSuccess
			if exitCode != 0 {
				expectedTask, expectedHost, expectedStep = TaskStatusFailed, TaskHostStatusFailed, StepStatusFailed
			}
			if result.Status != expectedTask || len(result.Hosts) != 1 || result.Hosts[0].Status != expectedHost {
				t.Fatalf("execution outcome = %s / %#v", result.Status, result.Hosts)
			}
			if len(result.Steps) != 1 || result.Steps[0].Status != expectedStep || result.Steps[0].AttemptNo != 1 {
				t.Fatalf("step outcome = %#v", result.Steps)
			}
			select {
			case got := <-received:
				if got[0] != "/bin/bash -se" || !strings.Contains(got[1], "echo frozen-fixture") || strings.Contains(got[1], "echo changed-fixture") {
					t.Fatal("frozen script/exec input was not preserved")
				}
			default:
				t.Fatal("SSH fixture did not receive executable input")
			}
			var logs []DeployTaskLogChunk
			if err := db.Where("task_id = ?", task.ID).Order("sequence").Find(&logs).Error; err != nil {
				t.Fatal(err)
			}
			sawStdout, sawStderr := false, false
			for _, log := range logs {
				if log.TaskHostID != result.Hosts[0].ID || log.TaskStepID != result.Steps[0].ID || log.AttemptNo != 1 {
					t.Fatalf("log attribution = %#v", log)
				}
				sawStdout = sawStdout || (log.Stream == "stdout" && strings.Contains(log.Content, "fixture stdout"))
				sawStderr = sawStderr || (log.Stream == "stderr" && strings.Contains(log.Content, "fixture stderr"))
				if strings.Contains(log.Content, "PRIVATE KEY") {
					t.Fatal("credential appeared in logs")
				}
			}
			if !sawStdout || !sawStderr {
				t.Fatal("both SSH output streams must be persisted")
			}
			var storedCredential DeployCredentialRef
			if err := db.First(&storedCredential, credential.ID).Error; err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(storedCredential.SecretEncrypted, "v1:") || strings.Contains(storedCredential.SecretEncrypted, "PRIVATE KEY") {
				t.Fatal("credential was not encrypted at rest")
			}
			var storedHost deployTestHost
			if err := db.First(&storedHost, hostID).Error; err != nil {
				t.Fatal(err)
			}
			installed := strings.Contains(string(storedHost.InstalledComponents), "loopback-component")
			if installed != (exitCode == 0) {
				t.Fatal("component writeback did not match the remote exit status")
			}
		})
	}
}

func TestDeploySSHTransportCancelDuringHandshake(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		// The peer accepts TCP but deliberately never supplies an SSH banner.
		runner, err := newDeploySSHRunner(ctx, cmdbHostSnapshot{IP: "127.0.0.1", SSHPort: listener.Addr().(*net.TCPAddr).Port}, StartTaskRequest{SSHUser: "fixture", SSHPassword: "fixture", HostFingerprint: "SHA256:fixture"})
		if runner != nil {
			_ = runner.Close()
		}
		done <- err
	}()
	var peer net.Conn
	select {
	case peer = <-accepted:
	case <-time.After(3 * time.Second):
		t.Fatal("TCP was not accepted")
	}
	defer peer.Close()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("handshake cancel = %v", err)
		}
	case <-time.After(time.Second):
		t.Error("SSH handshake ignored host cancellation")
		_ = peer.Close()
		<-done
	}
}

func TestDeploySSHTransportCanceledConnectionPreservesCommand(t *testing.T) {
	for _, command := range []string{TaskStatusPauseRequested, TaskStatusCancelRequested} {
		t.Run(command, func(t *testing.T) {
			db := setupDeployTestDB(t)
			svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
			scopeID, hostID := seedDeployReliabilityFixture(t, db, "interrupted-connect", "127.0.0.1")
			pkg, err := svc.CreatePackage(CreatePackageRequest{Name: "interrupted-fixture", Version: "1.0.0", InstallCommand: "echo fixture", Status: PackageStatusEnabled}, "1")
			if err != nil {
				t.Fatal(err)
			}
			task, err := svc.CreateTask(CreateTaskRequest{Name: "interrupted-connect", PackageID: pkg.ID, BusinessScopeID: scopeID, TargetType: TargetTypeHost, TargetIDs: []uint64{hostID}, ExecutorType: ExecutorTypeSSH}, "1", nil)
			if err != nil {
				t.Fatal(err)
			}
			taskHost := DeployTaskHost{TaskID: task.ID, HostID: hostID, Hostname: "interrupted-connect", HostIP: "127.0.0.1", SSHPort: 22, OS: "linux", BusinessScopeID: scopeID, Status: TaskHostStatusRunning}
			if err := db.Create(&taskHost).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&DeployTask{}).Where("id = ?", task.ID).Update("status", TaskStatusRunning).Error; err != nil {
				t.Fatal(err)
			}
			var ownedTask DeployTask
			if err := db.First(&ownedTask, task.ID).Error; err != nil {
				t.Fatal(err)
			}
			owned, _ := ownedClaim(t, svc, ownedTask, taskHost, "interrupted-worker")
			if err := db.Model(&DeployTask{}).Where("id = ?", task.ID).Update("status", command).Error; err != nil {
				t.Fatal(err)
			}
			var currentTask DeployTask
			if err := db.First(&currentTask, task.ID).Error; err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			// The real connection factory returns the host cancellation before any
			// network/credential work; the provider must preserve the command outcome.
			if err := owned.executeOrchestratedSSHHost(ctx, currentTask, cmdbHostSnapshot{ID: hostID, IP: "127.0.0.1", SSHPort: 22}, taskHostToResponse(&taskHost), StartTaskRequest{}, "1"); err != nil {
				t.Fatal(err)
			}
			if err := owned.completeOwnedAttempt(command, nil, "1", false); err != nil {
				t.Fatal(err)
			}
			if err := db.First(&currentTask, task.ID).Error; err != nil {
				t.Fatal(err)
			}
			expected := TaskStatusCanceled
			if command == TaskStatusPauseRequested {
				expected = TaskStatusPaused
			}
			if currentTask.Status != expected {
				t.Fatalf("interrupted task = %s, want %s", currentTask.Status, expected)
			}
			if err := db.First(&taskHost, taskHost.ID).Error; err != nil {
				t.Fatal(err)
			}
			expectedHost := TaskHostStatusSkipped
			if command == TaskStatusPauseRequested {
				expectedHost = TaskHostStatusRunning
			}
			if taskHost.Status != expectedHost || taskHost.ErrorMessage != "" {
				t.Fatalf("command interruption was recorded as a host failure: %#v", taskHost)
			}
			var failedEvents int64
			if err := db.Model(&DeployTaskEvent{}).Where("task_id = ? AND event_type = ?", task.ID, "host.failed").Count(&failedEvents).Error; err != nil {
				t.Fatal(err)
			}
			if failedEvents != 0 {
				t.Fatal("command interruption emitted host.failed")
			}
		})
	}
}
