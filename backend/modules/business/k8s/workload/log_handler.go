package workload

import (
	"bufio"
	"context"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	logPongWait   = 60 * time.Second
	logPingPeriod = (logPongWait * 9) / 10
	logTailLines  = 200
)

// logWriteWait bounds a stalled consumer: the provider scanner blocks on this
// write, so without a deadline a slow client would pin the stream forever.
// A variable lets the regression shorten the deadline instead of sleeping 10s.
var logWriteWait = 10 * time.Second

// logWriter serializes every frame on a connection. gorilla/websocket panics
// when two goroutines write concurrently, and the ping ticker and the log
// stream both write here.
type logWriter struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

func (w *logWriter) write(messageType int, data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	_ = w.conn.SetWriteDeadline(time.Now().Add(logWriteWait))
	return w.conn.WriteMessage(messageType, data)
}

var logUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// A browser always sends Origin for WebSocket handshakes, so the allowlist
	// is the CSRF boundary that keeps a foreign page from opening a stream with
	// the victim's cookie session. Non-browser clients send no Origin and
	// cannot be driven by a foreign page.
	CheckOrigin: websocketOriginAllowed,
}

// websocketOriginAllowed accepts an absent Origin (non-browser clients) and an
// Origin listed in PANTHEON_ALLOWED_ORIGINS. Without a configured allowlist it
// falls back to same-origin so a deployed console keeps working.
func websocketOriginAllowed(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	if allowed := parseAllowedOrigins(os.Getenv("PANTHEON_ALLOWED_ORIGINS")); len(allowed) > 0 {
		normalized := strings.TrimSuffix(strings.ToLower(origin), "/")
		return slices.Contains(allowed, normalized)
	}
	parsed, err := url.Parse(origin)
	return err == nil && parsed.Host != "" && strings.EqualFold(parsed.Host, r.Host)
}

func parseAllowedOrigins(raw string) []string {
	var allowed []string
	for entry := range strings.SplitSeq(raw, ",") {
		entry = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(entry)), "/")
		if entry != "" {
			allowed = append(allowed, entry)
		}
	}
	return allowed
}

// PodLogs streams a pod's logs over WebSocket. It uses the client-go pod log
// API with Follow=true to tail new lines until the connection closes.
//
// Route: GET /clusters/:id/pods/:namespace/:podName/logs
func (h *WorkloadHandler) PodLogs(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	namespace := c.Param("namespace")
	podName := c.Param("podName")
	if namespace == "" || podName == "" {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	container := c.Query("container")

	// Frame origin, ownership and provider work in that order so a rejected
	// handshake never opens a namespace or provider stream.
	if !websocketOriginAllowed(c.Request) {
		common.Fail(c, common.CodeForbidden, "k8s.workload.logs_origin_denied")
		return
	}
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	if err := h.svc.RequireLogAccess(ctx, clusterID, namespace, common.GetDataScope(c)); err != nil {
		common.FailWithError(c, common.CodeForbidden, err, "k8s.workload.logs_forbidden")
		return
	}
	clientset, err := h.svc.clusterSvc.GetClientset(clusterID, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.workload.pods_failed")
		return
	}

	conn, err := logUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	_ = conn.SetReadDeadline(time.Now().Add(logPongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(logPongWait))
		return nil
	})

	writer := &logWriter{conn: conn}

	// Ping ticker to keep the connection alive.
	pingDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(logPingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := writer.write(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-pingDone:
				return
			}
		}
	}()
	defer close(pingDone)

	// Drain inbound messages to detect close. A closed client must cancel the
	// provider stream: Follow=true otherwise blocks forever and leaks the
	// handler goroutine plus the API-server connection.
	go func() {
		defer cancel()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	streamPodLogs(ctx, writer, clientset, namespace, podName, container)
}

func streamPodLogs(ctx context.Context, writer *logWriter, clientset kubernetes.Interface, namespace, podName, container string) {
	tail := int64(logTailLines)
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: container,
		Follow:    true,
		TailLines: &tail,
	})
	stream, err := req.Stream(ctx)
	if err != nil {
		_ = writer.write(websocket.TextMessage, []byte("failed to open pod log stream: "+err.Error()))
		return
	}
	defer func() {
		_ = stream.Close()
	}()

	scanner := bufio.NewScanner(stream)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if err := writer.write(websocket.TextMessage, scanner.Bytes()); err != nil {
			return
		}
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		_ = writer.write(websocket.TextMessage, []byte("pod log stream interrupted: "+err.Error()))
	}
}
