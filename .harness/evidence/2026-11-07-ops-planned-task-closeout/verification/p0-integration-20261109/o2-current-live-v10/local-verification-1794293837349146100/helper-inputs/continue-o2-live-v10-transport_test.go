package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// Real local TCP/SSH protocol fixture with ephemeral credentials. It does not
// exercise DeployService, a real VM, database, product transport or real leases.
type verifierFixture struct {
	listener                                    net.Listener
	fingerprint                                 string
	accepted, authenticated, sessions, commands atomic.Int32
	mode                                        string
	entered                                     chan struct{}
	closed                                      chan struct{}
	mu                                          sync.Mutex
	connections                                 []net.Conn
	wg                                          sync.WaitGroup
}

func newVerifierFixture(t *testing.T, mode string) *verifierFixture {
	t.Helper()
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(private)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &verifierFixture{listener: listener, fingerprint: ssh.FingerprintSHA256(signer.PublicKey()),
		mode: mode, entered: make(chan struct{}, 16), closed: make(chan struct{})}
	cfg := &ssh.ServerConfig{PasswordCallback: func(c ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
		if c.User() != "fixture" || string(password) != "ephemeral-local-only" {
			return nil, errors.New("fixture authentication denied")
		}
		f.authenticated.Add(1)
		return nil, nil
	}}
	cfg.AddHostKey(signer)
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		for {
			raw, err := listener.Accept()
			if err != nil {
				return
			}
			f.accepted.Add(1)
			f.mu.Lock()
			f.connections = append(f.connections, raw)
			f.mu.Unlock()
			f.wg.Add(1)
			go func() {
				defer f.wg.Done()
				defer raw.Close()
				if mode == "handshake-eof" {
					return
				}
				if mode == "handshake-stall" {
					f.entered <- struct{}{}
					_, _ = io.Copy(io.Discard, raw)
					return
				}
				conn, channels, requests, err := ssh.NewServerConn(raw, cfg)
				if err != nil {
					return
				}
				defer conn.Close()
				go ssh.DiscardRequests(requests)
				for incoming := range channels {
					f.sessions.Add(1)
					if mode == "reject-session" {
						_ = incoming.Reject(ssh.Prohibited, "fixture refusal")
						continue
					}
					if mode == "stall-open" {
						f.entered <- struct{}{}
						<-f.closed
						return
					}
					channel, reqs, err := incoming.Accept()
					if err != nil {
						return
					}
					for request := range reqs {
						if request.Type != "exec" {
							_ = request.Reply(false, nil)
							continue
						}
						f.commands.Add(1)
						_ = request.Reply(true, nil)
						f.entered <- struct{}{}
						if mode == "stall-run" {
							<-f.closed
							_ = channel.Close()
							return
						}
						script, err := io.ReadAll(channel)
						if err != nil {
							return
						}
						_, _ = channel.Write(script)
						status := uint32(0)
						if string(script) == "exit17" {
							status = 17
						}
						_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{status}))
						_ = channel.Close()
					}
				}
			}()
		}
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		close(f.closed)
		f.mu.Lock()
		for _, conn := range f.connections {
			_ = conn.Close()
		}
		f.mu.Unlock()
		done := make(chan struct{})
		go func() { f.wg.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("fixture did not stop")
		}
	})
	return f
}

func (f *verifierFixture) driver(t *testing.T) *driver {
	r := &driver{c: config{User: "fixture", AuthMode: "password", Secret: "ephemeral-local-only", Fingerprint: f.fingerprint}}
	t.Cleanup(r.closeVerifier)
	return r
}

func (f *verifierFixture) run(ctx context.Context, r *driver, script string) (string, error) {
	return r.remoteContext(ctx, f.listener.Addr().String(), script)
}

func captureVerifierEvents(t *testing.T, run func()) []map[string]any {
	t.Helper()
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	prior := os.Stdout
	os.Stdout = write
	defer func() { os.Stdout = prior; _ = read.Close(); _ = write.Close() }()
	done := make(chan []byte, 1)
	go func() { raw, _ := io.ReadAll(read); done <- raw }()
	run()
	os.Stdout = prior
	_ = write.Close()
	var events []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(<-done)), "\n") {
		var event map[string]any
		if json.Unmarshal([]byte(line), &event) != nil {
			t.Fatal("invalid diagnostic event")
		}
		events = append(events, event)
	}
	return events
}

func TestVerifierReusesOneAuthenticatedTransportAndExactPinOnce(t *testing.T) {
	f := newVerifierFixture(t, "success")
	r := f.driver(t)
	events := captureVerifierEvents(t, func() {
		for _, script := range []string{"first", "second", "third"} {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			out, err := f.run(ctx, r, script)
			cancel()
			if err != nil || out != script {
				t.Fatalf("command output/exit mismatch: %q %v", out, err)
			}
			// The previous context and socket deadline must not poison reuse.
			time.Sleep(110 * time.Millisecond)
		}
	})
	pins := 0
	for _, e := range events {
		if e["phase"] == "independent-ssh-handshake" {
			pins++
			if e["pinChecked"] != true || e["pinMatched"] != true || e["pinCallbacks"] != float64(1) {
				t.Fatal("exact pin not enforced")
			}
		}
	}
	if pins != 1 || r.verifierConnections != 1 || f.accepted.Load() != 1 || f.authenticated.Load() != 1 || f.sessions.Load() != 3 || f.commands.Load() != 3 {
		t.Fatal("commands must use three sessions on one pinned authenticated connection")
	}
}

func TestVerifierChangedPinRejectedWithoutAuthenticationOrRetry(t *testing.T) {
	f := newVerifierFixture(t, "success")
	r := f.driver(t)
	r.c.Fingerprint = "SHA256:changed-pin"
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for i := 0; i < 2; i++ {
		if _, err := f.run(ctx, r, "denied"); !errors.Is(err, errVerifierPinMismatch) {
			t.Fatal("changed pin accepted")
		}
	}
	if r.verifierConnections != 1 || f.accepted.Load() != 1 || f.authenticated.Load() != 0 || f.sessions.Load() != 0 {
		t.Fatal("pin failure retried or authenticated")
	}
}

func TestVerifierFirstHandshakeEOFIsFinal(t *testing.T) {
	f := newVerifierFixture(t, "handshake-eof")
	r := f.driver(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, first := f.run(ctx, r, "first")
	_, second := f.run(ctx, r, "second")
	if first == nil || second != first || r.verifierConnections != 1 || f.accepted.Load() != 1 {
		t.Fatal("first handshake failure must be cached without redial")
	}
}

func TestVerifierSessionRefusalNeverReconnects(t *testing.T) {
	f := newVerifierFixture(t, "reject-session")
	r := f.driver(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for i := 0; i < 2; i++ {
		_, err := f.run(ctx, r, "denied")
		var rejected *ssh.OpenChannelError
		if !errors.As(err, &rejected) {
			t.Fatal("session refusal lost")
		}
	}
	if f.accepted.Load() != 1 || f.authenticated.Load() != 1 || f.sessions.Load() != 2 || f.commands.Load() != 0 {
		t.Fatal("refusal caused redial/replay")
	}
}

func TestVerifierCommandExit17PreservedWithoutReplay(t *testing.T) {
	f := newVerifierFixture(t, "success")
	r := f.driver(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := f.run(ctx, r, "exit17")
	var exit *ssh.ExitError
	if !errors.As(err, &exit) || exit.ExitStatus() != 17 {
		t.Fatal("exact command exit17 lost")
	}
	if out, err := f.run(ctx, r, "next"); err != nil || out != "next" {
		t.Fatal("connection did not survive command refusal")
	}
	if f.accepted.Load() != 1 || f.commands.Load() != 2 {
		t.Fatal("command replayed or transport reconnected")
	}
}

func TestVerifierTimeoutBoundsHandshakeChannelAndCommand(t *testing.T) {
	for _, mode := range []string{"handshake-stall", "stall-open", "stall-run"} {
		t.Run(mode, func(t *testing.T) {
			f := newVerifierFixture(t, mode)
			r := f.driver(t)
			ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
			defer cancel()
			start := time.Now()
			_, err := f.run(ctx, r, "blocked")
			if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > 2*time.Second {
				t.Fatalf("deadline lost or unbounded: %v", err)
			}
			fresh, stop := context.WithTimeout(context.Background(), time.Second)
			defer stop()
			if _, err = f.run(fresh, r, "never-reconnect"); err == nil {
				t.Fatal("timed out transport accepted command")
			}
			if r.verifierConnections != 1 || f.accepted.Load() != 1 {
				t.Fatal("timeout redialed")
			}
		})
	}
}

func TestVerifierCancellationAndShutdownInterruptActiveCommand(t *testing.T) {
	for _, action := range []string{"cancel", "shutdown"} {
		t.Run(action, func(t *testing.T) {
			f := newVerifierFixture(t, "stall-run")
			r := f.driver(t)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() { _, err := f.run(ctx, r, "blocked"); done <- err }()
			select {
			case <-f.entered:
			case <-time.After(time.Second):
				t.Fatal("command not started")
			}
			if action == "cancel" {
				cancel()
			} else {
				r.closeVerifier()
				r.closeVerifier()
			}
			select {
			case err := <-done:
				if err == nil || action == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal("cancel/shutdown error lost")
				}
			case <-time.After(time.Second):
				t.Fatal("cancel/shutdown failed to interrupt command")
			}
			fresh, stop := context.WithTimeout(context.Background(), time.Second)
			defer stop()
			if _, err := f.run(fresh, r, "after-close"); err == nil {
				t.Fatal("closed transport accepted command")
			}
			if f.accepted.Load() != 1 || f.authenticated.Load() != 1 || f.commands.Load() != 1 {
				t.Fatal("canceled command reconnected or replayed")
			}
		})
	}
}

func TestVerifierShutdownBeforeUseNeverDials(t *testing.T) {
	f := newVerifierFixture(t, "success")
	r := f.driver(t)
	r.closeVerifier()
	if _, err := f.run(context.Background(), r, "closed"); !errors.Is(err, errVerifierClosed) {
		t.Fatal("released verifier was reopened")
	}
	if f.accepted.Load() != 0 || r.verifierConnections != 0 {
		t.Fatal("released verifier dialed")
	}
}

func TestVerifierShutdownInterruptsFirstHandshake(t *testing.T) {
	f := newVerifierFixture(t, "handshake-stall")
	r := f.driver(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := f.run(ctx, r, "blocked"); done <- err }()
	select {
	case <-f.entered:
	case <-time.After(time.Second):
		t.Fatal("handshake not started")
	}
	r.closeVerifier()
	select {
	case err := <-done:
		if !errors.Is(err, errVerifierClosed) {
			t.Fatal("shutdown did not preserve verifier closure")
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown blocked on first handshake")
	}
	if _, err := f.run(ctx, r, "never-redial"); !errors.Is(err, errVerifierClosed) || f.accepted.Load() != 1 {
		t.Fatal("closed first handshake reopened")
	}
}

func TestVerifierIndependentDriverClientsDoNotShareTransport(t *testing.T) {
	f := newVerifierFixture(t, "success")
	first, second := f.driver(t), f.driver(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for _, r := range []*driver{first, second} {
		if out, err := f.run(ctx, r, "independent"); err != nil || out != "independent" {
			t.Fatal("independent driver did not authenticate")
		}
	}
	first.closeVerifier()
	if out, err := f.run(ctx, second, "still-owned"); err != nil || out != "still-owned" {
		t.Fatal("closing one verifier affected another")
	}
	if first.verifierClient == second.verifierClient || f.accepted.Load() != 2 || f.authenticated.Load() != 2 || f.commands.Load() != 3 {
		t.Fatal("separate driver clients shared or reconnected a transport")
	}
}
