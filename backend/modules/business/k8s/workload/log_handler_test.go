package workload

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestPodLogUpgradeRejectsCrossOrigin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := logUpgrader.Upgrade(w, r, nil)
		if err == nil {
			_ = conn.Close()
		}
	}))
	defer server.Close()
	target := "ws" + strings.TrimPrefix(server.URL, "http")
	for _, scenario := range []struct {
		origin  string
		allowed bool
	}{
		{server.URL, true}, {"https://untrusted.example", false},
	} {
		conn, response, err := websocket.DefaultDialer.Dial(target, http.Header{"Origin": {scenario.origin}})
		if conn != nil {
			_ = conn.Close()
		}
		if scenario.allowed && err != nil {
			t.Fatalf("same-origin logs were denied: %v", err)
		}
		if !scenario.allowed && (err == nil || response == nil || response.StatusCode != http.StatusForbidden) {
			t.Fatalf("cross-origin cookie log stream allowed: response=%v error=%v", response, err)
		}
		if response != nil {
			_ = response.Body.Close()
		}
	}
}
