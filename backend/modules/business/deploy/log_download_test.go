package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/gin-gonic/gin"
)

type downloadMode struct{}

func (downloadMode) ReadMode(context.Context) (string, error) { return tenant.ModeCompat, nil }

type downloadReader struct {
	allow map[string]bool
	calls int
	scope func(int) *common.DataScopeReq
}

func (*downloadReader) RefreshPrincipal(context.Context, businessauth.Identity) (businessauth.Principal, error) {
	return businessauth.Principal{}, nil
}
func (r *downloadReader) Authorize(_ context.Context, input businessauth.Input) (businessauth.Authorization, error) {
	r.calls++
	if !r.allow[input.Action.Permission] {
		return businessauth.Authorization{}, businessauth.ErrDenied
	}
	scope := &common.DataScopeReq{Mode: "all"}
	if r.scope != nil {
		scope = r.scope(r.calls)
	}
	return businessauth.Authorization{DataScope: scope}, nil
}
func downloadRouter(reader *downloadReader, handler *DeployHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		businessauth.BindGin(c, reader, downloadMode{}, businessauth.Input{Resolution: tenant.Context{Mode: tenant.ModeCompat}, DataScope: true, Action: businessauth.Action{Method: "GET", Path: c.Request.URL.Path, RouteTemplate: deployLogDownloadTemplate}})
	})
	router.GET("/tasks/:id/log-download", handler.DownloadTaskLogs)
	return router
}

func TestDownloadRequiresBothLivePermissions(t *testing.T) {
	for _, key := range []string{"business:deploy:task:export", "business:deploy:task:detail"} {
		t.Run(key, func(t *testing.T) {
			reader := &downloadReader{allow: map[string]bool{"business:deploy:task:export": true, "business:deploy:task:detail": true}}
			reader.allow[key] = false
			recorder := httptest.NewRecorder()
			downloadRouter(reader, NewDeployHandler(&DeployService{})).ServeHTTP(recorder, httptest.NewRequest("GET", "/tasks/1/log-download", nil))
			if recorder.Code != 403 || strings.Contains(recorder.Body.String(), "items") {
				t.Fatalf("missing %s was not denied: %d %s", key, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestDownloadRedactionOverlapsUTF8AndChunkLimit(t *testing.T) {
	chunks := []DeployTaskLogChunk{{Content: "abcdef abc {{DEPLOY_SECRET:abc}}"}, {Content: strings.Repeat("世", 6000)}}
	items, limited, err := redactTaskLogItems(chunks, map[string]string{"long": "abcdef", "short": "abc"})
	if err != nil || !limited || strings.Contains(items[0].Content, "def") || strings.Contains(items[0].Content, "abcdef") || !utf8.ValidString(items[1].Content) || len(items[1].Content) > deployLogChunkBytes {
		t.Fatalf("unsafe redaction/clipping: %v %t %+v", err, limited, items)
	}
	if !strings.Contains(items[0].Content, secretSentinelPrefix+"long}}") {
		t.Fatalf("longest secret was not replaced atomically: %s", items[0].Content)
	}
}

func TestDownloadFinalScopeAndEncodedByteLimit(t *testing.T) {
	db, svc, task, hosts, steps := mixedReadFixture(t)
	for _, host := range hosts {
		if err := db.Model(&deployTestHost{}).Where("id = ?", host.HostID).Update("dept_id", 71).Error; err != nil {
			t.Fatal(err)
		}
	}
	var logs []DeployTaskLogChunk
	for i := 0; i < 150; i++ {
		host := i % 2
		logs = append(logs, DeployTaskLogChunk{TaskID: task.ID, TaskHostID: hosts[host].ID, TaskStepID: steps[host].ID, Sequence: uint64(i + 1), Content: strings.Repeat("<", 16000)})
	}
	if err := db.CreateInBatches(&logs, 50).Error; err != nil {
		t.Fatal(err)
	}
	reader := &downloadReader{allow: map[string]bool{"business:deploy:task:export": true, "business:deploy:task:detail": true}, scope: func(call int) *common.DataScopeReq {
		if call == 3 {
			if err := db.Model(&deployTestHost{}).Where("id = ?", hosts[1].HostID).Update("dept_id", 72).Error; err != nil {
				t.Fatal(err)
			}
		}
		return &common.DataScopeReq{Mode: "dept", DeptID: 71}
	}}
	recorder := httptest.NewRecorder()
	downloadRouter(reader, NewDeployHandler(svc)).ServeHTTP(recorder, httptest.NewRequest("GET", fmt.Sprintf("/tasks/%d/log-download", task.ID), nil))
	var payload struct {
		Code int
		Data struct {
			Items     []TaskLogItem
			Truncated bool
		}
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != 200 || !payload.Data.Truncated || len(payload.Data.Items) == 0 || recorder.Body.Len() > deployLogDownloadBytes || recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("download limits: code=%d truncated=%t rows=%d bytes=%d", payload.Code, payload.Data.Truncated, len(payload.Data.Items), recorder.Body.Len())
	}
	for _, item := range payload.Data.Items {
		if item.TaskHostID != hosts[0].ID {
			t.Fatalf("scope shrink retained forbidden host %d", item.TaskHostID)
		}
	}
	if strings.Contains(fmt.Sprint(recorder.Header()), strings.Repeat("<", 100)) {
		t.Fatal("headers contain output")
	}
}

func TestDownloadFailsClosedForCorruptLegacySecret(t *testing.T) {
	db, svc, task, _, _ := mixedReadFixture(t)
	secret := DeployTaskSecret{TaskID: task.ID, SecretKey: "private", SecretEncrypted: "corrupt-ciphertext"}
	if err := db.Create(&secret).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.safeTaskLogItems(task.ID, []DeployTaskLogChunk{{Content: "private legacy output"}}); err == nil {
		t.Fatal("incomplete secret dictionary must fail closed")
	}
}
