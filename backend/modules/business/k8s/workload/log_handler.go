package workload

import (
	"bufio"
	"context"
	"errors"
	"net/url"
	"strconv"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"

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
	// The default origin check also protects the cookie-authenticated stream.
}

func parsePodLogOptions(query url.Values) (corev1.PodLogOptions, error) {
	tail := int64(logTailLines)
	options := corev1.PodLogOptions{Container: query.Get("container"), Follow: true, TailLines: &tail}
	for key, target := range map[string]*bool{"follow": &options.Follow, "previous": &options.Previous, "timestamps": &options.Timestamps} {
		if values, present := query[key]; present {
			if len(values) != 1 {
				return options, errors.New(msgParamInvalid)
			}
			value, err := strconv.ParseBool(values[0])
			if err != nil {
				return options, errors.New(msgParamInvalid)
			}
			*target = value
		}
	}
	if values, present := query["sinceSeconds"]; present {
		if len(values) != 1 {
			return options, errors.New(msgParamInvalid)
		}
		value, err := strconv.ParseInt(values[0], 10, 64)
		if err != nil || value < 1 || value > 7*24*60*60 {
			return options, errors.New(msgParamInvalid)
		}
		options.SinceSeconds = &value
	}
	return options, nil
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
	options, err := parsePodLogOptions(c.Request.URL.Query())
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	input, err := businessauth.FromGin(c)
	if err != nil {
		common.FailWithCode(c, 403, "permission.denied")
		return
	}
	action := input.Action
	action.Permission = "business:k8s:workload:logs"
	authorize := func() error {
		if _, err := businessauth.Reauthorize(c, &action); err != nil {
			return businessauth.ErrDenied
		}
		_, err := h.svc.clusterSvc.GetByID(clusterID, common.GetDataScope(c))
		return err
	}
	if err := authorize(); err != nil {
		common.FailWithCode(c, 403, "permission.denied")
		return
	}
	initial, err := h.svc.clusterSvc.GetByID(clusterID, common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, 403, "permission.denied")
		return
	}
	checkSource := func() error {
		if err := authorize(); err != nil {
			return err
		}
		current, err := h.svc.clusterSvc.GetByID(clusterID, common.GetDataScope(c))
		if err != nil || current.CredentialRefID != initial.CredentialRefID || current.CredentialVersion != initial.CredentialVersion || current.CredentialStatus != initial.CredentialStatus || current.APIServer != initial.APIServer {
			return businessauth.ErrDenied
		}
		return nil
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
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	_ = conn.SetReadDeadline(time.Now().Add(logPongWait))
	conn.SetReadLimit(256)
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
				if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(logWriteWait)); err != nil {
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
				cancel()
				return
			}
		}
	}()

	streamPodLogsWithOptions(ctx, conn, clientset, namespace, podName, options, checkSource)
}

func streamPodLogs(ctx context.Context, conn *websocket.Conn, clientset kubernetes.Interface, namespace, podName, container string, checks ...func() error) {
	options, _ := parsePodLogOptions(url.Values{"container": {container}})
	streamPodLogsWithOptions(ctx, conn, clientset, namespace, podName, options, checks...)
}

func streamPodLogsWithOptions(ctx context.Context, conn *websocket.Conn, clientset kubernetes.Interface, namespace, podName string, options corev1.PodLogOptions, checks ...func() error) {
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &options)
	stream, err := req.Stream(ctx)
	if err != nil {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(4400, "source-unavailable"), time.Now().Add(logWriteWait))
		return
	}
	defer func() {
		_ = stream.Close()
	}()

	streamCtx, stop := context.WithCancel(ctx)
	defer stop()
	lines := make(chan []byte, 1)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(stream)
		scanner.Buffer(make([]byte, 0, 8192), 16384)
		for scanner.Scan() {
			line := append([]byte(nil), scanner.Bytes()...)
			select {
			case lines <- line:
			case <-streamCtx.Done():
				return
			}
		}
	}()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	check := func() bool {
		if len(checks) > 0 && checks[0] != nil && checks[0]() != nil {
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(4403, "permission-denied"), time.Now().Add(logWriteWait))
			return false
		}
		return true
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !check() {
				return
			}
		case line, ok := <-lines:
			if !ok || !check() {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(logWriteWait))
			if conn.WriteMessage(websocket.TextMessage, line) != nil {
				return
			}
		}
	}
}
