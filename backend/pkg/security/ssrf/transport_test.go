package ssrf

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDialCheckedPinsValidatedIP(t *testing.T) {
	v := DefaultValidator()
	lookups := 0
	var dialed string
	lookup := func(context.Context, string) ([]net.IPAddr, error) {
		lookups++
		ip := "93.184.216.34"
		if lookups > 1 {
			ip = "127.0.0.1"
		}
		return []net.IPAddr{{IP: net.ParseIP(ip)}}, nil
	}
	dial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialed = addr
		host, _, _ := net.SplitHostPort(addr)
		// Model the dialer's second resolution, which was the original bypass.
		if net.ParseIP(host) == nil {
			if _, err := lookup(ctx, host); err != nil {
				return nil, err
			}
		}
		a, b := net.Pipe()
		t.Cleanup(func() { a.Close(); b.Close() })
		return a, nil
	}
	conn, err := v.dialChecked(context.Background(), "tcp", "rebinding.test:443", lookup, dial)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
	if lookups != 1 || dialed != "93.184.216.34:443" {
		t.Fatalf("DNS lookups=%d dialed=%s; must dial only the validated IP", lookups, dialed)
	}
}

func TestDialCheckedRejectsUnsafeAnswersBeforeDial(t *testing.T) {
	for _, tc := range []struct {
		name string
		ips  []net.IPAddr
		err  error
	}{
		{name: "empty"},
		{name: "DNS failure", err: errors.New("resolver failed")},
		{name: "loopback", ips: []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}},
		{name: "mixed", ips: []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}, {IP: net.ParseIP("10.0.0.1")}}},
		{name: "IPv6 private", ips: []net.IPAddr{{IP: net.ParseIP("::1")}}},
		{name: "mapped private", ips: []net.IPAddr{{IP: net.ParseIP("::ffff:127.0.0.1")}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			_, err := DefaultValidator().dialChecked(context.Background(), "tcp", "fixture.test:80",
				func(context.Context, string) ([]net.IPAddr, error) { return tc.ips, tc.err },
				func(context.Context, string, string) (net.Conn, error) { called = true; return nil, nil })
			if err == nil || called {
				t.Fatalf("error=%v dial called=%v", err, called)
			}
		})
	}
}

func TestDialCheckedFallbackAndIPFamilies(t *testing.T) {
	for _, network := range []string{"tcp", "tcp4", "tcp6"} {
		t.Run(network, func(t *testing.T) {
			var attempts []string
			_, err := DefaultValidator().dialChecked(context.Background(), network, "fixture.test:443",
				func(context.Context, string) ([]net.IPAddr, error) {
					return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}, {IP: net.ParseIP("2001:4860:4860::8888")}}, nil
				},
				func(_ context.Context, _ string, addr string) (net.Conn, error) {
					attempts = append(attempts, addr)
					return nil, errors.New("fixture refused")
				})
			expected := []string{"8.8.8.8:443", "[2001:4860:4860::8888]:443"}
			if network == "tcp4" {
				expected = expected[:1]
			}
			if network == "tcp6" {
				expected = expected[1:]
			}
			if err == nil || strings.Join(attempts, ",") != strings.Join(expected, ",") {
				t.Fatalf("attempts=%v error=%v", attempts, err)
			}
		})
	}
}

func TestDialCheckedHonorsContextAndTimeout(t *testing.T) {
	t.Run("canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		called := false
		_, err := DefaultValidator().dialChecked(ctx, "tcp", "fixture.test:443",
			func(ctx context.Context, _ string) ([]net.IPAddr, error) { return nil, ctx.Err() },
			func(context.Context, string, string) (net.Conn, error) { called = true; return nil, nil })
		if !errors.Is(err, context.Canceled) || called {
			t.Fatalf("error=%v dial=%v", err, called)
		}
	})
	t.Run("timeout includes lookup", func(t *testing.T) {
		v := DefaultValidator()
		v.Timeout = 10 * time.Millisecond
		_, err := v.dialChecked(context.Background(), "tcp", "fixture.test:443",
			func(ctx context.Context, _ string) ([]net.IPAddr, error) {
				if _, ok := ctx.Deadline(); !ok {
					return nil, errors.New("missing shared lookup deadline")
				}
				<-ctx.Done()
				return nil, ctx.Err()
			}, func(context.Context, string, string) (net.Conn, error) { t.Fatal("must not dial"); return nil, nil })
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error=%v", err)
		}
	})
}

func TestDialCheckedLiteralIPsAndProtectionFlags(t *testing.T) {
	for _, host := range []string{"8.8.8.8", "2001:4860:4860::8888", "127.0.0.1", "fe80::1%lo0"} {
		t.Run(host, func(t *testing.T) {
			called := false
			_, err := DefaultValidator().dialChecked(context.Background(), "tcp", net.JoinHostPort(host, "443"),
				func(context.Context, string) ([]net.IPAddr, error) {
					t.Fatal("literal IP must not resolve")
					return nil, nil
				},
				func(_ context.Context, _ string, addr string) (net.Conn, error) {
					called = true
					return nil, errors.New("fixture refused")
				})
			private := host == "127.0.0.1" || strings.HasPrefix(host, "fe80")
			if err == nil || called == private {
				t.Fatalf("error=%v dial=%v", err, called)
			}
		})
	}
	v := DefaultValidator()
	v.BlockPrivateIPs = false
	v.DNSRebindProtection = false
	var dialed string
	_, _ = v.dialChecked(context.Background(), "tcp", "internal.test:80",
		func(context.Context, string) ([]net.IPAddr, error) {
			t.Fatal("both flags off must preserve ordinary dial")
			return nil, nil
		},
		func(_ context.Context, _ string, addr string) (net.Conn, error) {
			dialed = addr
			return nil, errors.New("fixture refused")
		})
	if dialed != "internal.test:80" {
		t.Fatal(dialed)
	}
	v.DNSRebindProtection = true
	called := false
	_, err := v.dialChecked(context.Background(), "tcp", "127.0.0.1:80", nil,
		func(context.Context, string, string) (net.Conn, error) { called = true; return nil, nil })
	if err == nil || called {
		t.Fatal("rebinding protection still blocks private IPs")
	}
}

func TestPinnedDialPreservesHTTPHostAndTLSSNI(t *testing.T) {
	type target struct{ host, sni string }
	observed := make(chan target, 1)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed <- target{r.Host, r.TLS.ServerName}
		w.WriteHeader(204)
	}))
	server.StartTLS()
	defer server.Close()
	v := DefaultValidator()
	client := v.CreateHTTPClient(time.Second)
	transport := client.Transport.(*http.Transport)
	// TLS verification is disabled only for the self-signed, local fixture.
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return v.dialChecked(ctx, network, addr,
			func(context.Context, string) ([]net.IPAddr, error) {
				return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}, nil
			},
			func(ctx context.Context, network, addr string) (net.Conn, error) {
				if !strings.HasPrefix(addr, "8.8.8.8:") {
					t.Errorf("unpinned address: %s", addr)
				}
				return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
			})
	}
	defer transport.CloseIdleConnections()
	resp, err := client.Get("https://fixture.test/pinned")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	got := <-observed
	if got.host != "fixture.test" || got.sni != "fixture.test" {
		t.Fatalf("Host=%s SNI=%s", got.host, got.sni)
	}
	for _, redirectLimit := range []int{0, 2} {
		v.MaxRedirects = redirectLimit
		check := v.CreateHTTPClient(time.Second).CheckRedirect
		req, _ := http.NewRequest("GET", "http://127.0.0.1", nil)
		if err := check(req, []*http.Request{req}); err == nil {
			t.Fatal("private redirect must be rejected")
		}
	}
}

func TestDialCheckedReservesBudgetForFallback(t *testing.T) {
	for _, parentDeadline := range []bool{false, true} {
		t.Run(fmt.Sprintf("parent_deadline_%t", parentDeadline), func(t *testing.T) {
			checkDialFallback(t, parentDeadline)
		})
	}
}

func checkDialFallback(t *testing.T, parentDeadline bool) {
	t.Helper()
	v := DefaultValidator()
	v.Timeout = 400 * time.Millisecond
	ctx := context.Background()
	if parentDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 300*time.Millisecond)
		defer cancel()
		v.Timeout = time.Second
	}
	attempts := 0
	conn, err := v.dialChecked(ctx, "tcp4", "fallback.test:443",
		func(context.Context, string) ([]net.IPAddr, error) {
			return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}, {IP: net.ParseIP("2001:4860:4860::8888")}, {IP: net.ParseIP("1.1.1.1")}}, nil
		},
		func(attempt context.Context, _ string, addr string) (net.Conn, error) {
			attempts++
			if addr == "8.8.8.8:443" {
				<-attempt.Done()
				return nil, attempt.Err()
			}
			if addr != "1.1.1.1:443" {
				t.Fatalf("unexpected fallback address: %s", addr)
			}
			a, b := net.Pipe()
			t.Cleanup(func() { a.Close(); b.Close() })
			return a, nil
		})
	if err != nil || conn == nil || attempts != 2 {
		t.Fatalf("must reach second verified IPv4 before overall deadline: attempts=%d error=%v", attempts, err)
	}
	if ctx.Err() != nil {
		t.Fatalf("fallback exceeded parent budget: %v", ctx.Err())
	}
}
