package workload

import (
	"bufio"
	"context"
	"net/http"
	"strconv"
	"time"

	"pantheon-base/pkg/common"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	logWriteWait  = 10 * time.Second
	logPongWait   = 60 * time.Second
	logPingPeriod = (logPongWait * 9) / 10
	logTailLines  = 200
)

var logUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Token is already validated by TokenAuthMiddleware (cookie); allowing
	// cross-origin here is acceptable for the internal ops platform.
	CheckOrigin: func(*http.Request) bool { return true },
}

// PodLogs streams a pod's logs over WebSocket. It uses the client-go pod log
// API with Follow=true to tail new lines until the connection closes.
//
// Route: GET /clusters/:clusterId/pods/:namespace/:podName/logs
func (h *WorkloadHandler) PodLogs(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("clusterId"), 10, 64)
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

	clientset, err := h.svc.clusterSvc.GetClientset(clusterID, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.workload.pods_failed")
		return
	}

	conn, err := logUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(logPongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(logPongWait))
		return nil
	})

	// Ping ticker to keep the connection alive.
	pingDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(logPingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(logWriteWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-pingDone:
				return
			}
		}
	}()
	defer close(pingDone)

	// Ignore inbound messages (client sends none); just drain to detect close.
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	streamPodLogs(conn, clientset, namespace, podName, container)
}

func streamPodLogs(conn *websocket.Conn, clientset kubernetes.Interface, namespace, podName, container string) {
	tail := int64(logTailLines)
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: container,
		Follow:    true,
		TailLines: &tail,
	})
	stream, err := req.Stream(context.Background())
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("failed to open pod log stream: "+err.Error()))
		return
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		_ = conn.SetWriteDeadline(time.Now().Add(logWriteWait))
		if err := conn.WriteMessage(websocket.TextMessage, scanner.Bytes()); err != nil {
			return
		}
	}
}
