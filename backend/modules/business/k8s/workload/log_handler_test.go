package workload

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestWebsocketOriginAllowed pins the CSRF boundary for the log stream: a
// foreign Origin must be refused before any provider connection, an allowlisted
// Origin is admitted, and no-Origin non-browser clients stay supported.
func TestWebsocketOriginAllowed(t *testing.T) {
	t.Setenv("PANTHEON_ALLOWED_ORIGINS", "https://console.example.com, https://ops.example.com/")
	cases := []struct {
		name   string
		origin string
		host   string
		want   bool
	}{
		{name: "allowlisted", origin: "https://console.example.com", want: true},
		{name: "allowlisted trailing slash", origin: "https://ops.example.com/", want: true},
		{name: "foreign origin", origin: "https://evil.example.com", want: false},
		{name: "null origin", origin: "null", want: false},
		{name: "no origin", origin: "", want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://api.example.com/api/v1/business/k8s/clusters/1/pods/default/p/logs", nil)
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if got := websocketOriginAllowed(r); got != tc.want {
				t.Fatalf("allowed = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWebsocketOriginSameOriginFallback(t *testing.T) {
	t.Setenv("PANTHEON_ALLOWED_ORIGINS", "")
	r := httptest.NewRequest("GET", "http://console.example.com/logs", nil)
	r.Host = "console.example.com"
	r.Header.Set("Origin", "http://console.example.com")
	if !websocketOriginAllowed(r) {
		t.Fatal("same-origin handshake rejected without an allowlist")
	}
	r.Header.Set("Origin", "http://evil.example.com")
	if websocketOriginAllowed(r) {
		t.Fatal("cross-origin handshake admitted without an allowlist")
	}
}

// TestLogWriterStalledConsumerBoundedByDeadline pins the backpressure bound: a
// client that never reads cannot make the log stream buffer without limit. The
// provider scanner writes synchronously, so the write deadline is what stops a
// slow consumer; a 1 MB frame cannot fit the default TCP window without the
// peer reading.
func TestLogWriterStalledConsumerBoundedByDeadline(t *testing.T) {
	previous := logWriteWait
	logWriteWait = 100 * time.Millisecond
	defer func() { logWriteWait = previous }()

	hold := make(chan struct{})
	defer close(hold)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := logUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		// Hold the connection open without reading: the client's send buffer
		// fills, so only the write deadline can release the writer.
		<-hold
	}))
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	writer := &logWriter{conn: conn}
	payload := make([]byte, 1<<20)
	start := time.Now()
	// The kernel send buffer grows before it stops accepting frames, so fill
	// it in bounded steps rather than assuming one frame is enough.
	const maxFrames = 256
	frames := 0
	for ; frames < maxFrames; frames++ {
		if err := writer.write(websocket.TextMessage, payload); err != nil {
			break
		}
	}
	if frames == maxFrames {
		t.Fatalf("stalled consumer accepted %d MB without a write error", maxFrames)
	}
	t.Logf("write deadline stopped the stream after %d MB", frames+1)
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("write returned after %v; deadline did not bound the stalled consumer", elapsed)
	}
}

// TestLogWriterSerializesConcurrentFrames pins the single-writer protocol: the
// ping ticker and the log stream write from different goroutines, and
// gorilla/websocket rejects overlapping writes on one connection.
func TestLogWriterSerializesConcurrentFrames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := logUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	writer := &logWriter{conn: conn}
	var wg sync.WaitGroup
	errs := make(chan error, 100)
	for range 100 {
		wg.Go(func() {
			if err := writer.write(websocket.TextMessage, []byte("line")); err != nil {
				errs <- err
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent write rejected: %v", err)
	}
}
