package argocd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func requestActor(c *gin.Context) ActionActor {
	return ActionActor{ID: common.GetUserID(c), DataScope: common.GetDataScope(c)}
}

// authorizeRequest guards every HTTP access to a registered configuration.
// Request/audit payloads may contain credentials or manifest literals, so only
// safe identifiers are retained. The authority re-resolves current scope.
func (h *ArgoCDHandler) authorizeRequest(c *gin.Context) {
	common.SetAuditParam(c, "{}")
	common.SetAuditResult(c, "{}")
	actor := requestActor(c)
	if actor.ID == 0 || actor.DataScope == nil || actor.DataScope.UserID != actor.ID || h.service.authorizer == nil {
		common.SetAuditErrorMsg(c, "common.forbidden")
		common.Fail(c, common.CodeForbidden, "common.forbidden")
		c.Abort()
		return
	}
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var input struct {
		ServerID          uint64 `json:"serverId"`
		ClusterID         uint64 `json:"clusterId"`
		SourceRepoURL     string `json:"sourceRepoUrl"`
		RepoURL           string `json:"repoUrl"`
		DestinationServer string `json:"destinationServer"`
		Namespace         string `json:"destinationNamespace"`
	}
	var updateReq UpdateApplicationRequest
	if c.Request.Body != nil && c.Request.Method != http.MethodGet && c.Request.Method != http.MethodDelete {
		data, err := io.ReadAll(io.LimitReader(c.Request.Body, (1<<20)+1))
		_ = c.Request.Body.Close()
		c.Request.Body = io.NopCloser(bytes.NewReader(data))
		if err != nil || len(data) > 1<<20 || (len(bytes.TrimSpace(data)) > 0 && json.Unmarshal(data, &input) != nil) {
			common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
			c.Abort()
			return
		}
		if len(bytes.TrimSpace(data)) > 0 {
			_ = json.Unmarshal(data, &updateReq)
		}
	}
	app := ArgoCDApplication{ServerID: input.ServerID, SourceRepoURL: input.SourceRepoURL, DestinationServer: input.DestinationServer, DestinationNS: input.Namespace}
	server := ArgoCDServer{ID: input.ServerID, ClusterID: input.ClusterID}
	ctx := c.Request.Context()
	path := c.FullPath()
	if strings.Contains(path, "/applications") && id > 0 {
		if err := h.service.db.WithContext(ctx).First(&app, id).Error; err != nil {
			failAction(c, resourceReadError(ctx, err, "cicd.argocd.app_get_failed"), "cicd.argocd.app_get_failed")
			c.Abort()
			return
		}
		server.ID = app.ServerID
	} else if strings.Contains(path, "/repositories") && id > 0 {
		var repository ArgoCDRepository
		if err := h.service.db.WithContext(ctx).First(&repository, id).Error; err != nil {
			failAction(c, resourceReadError(ctx, err, "cicd.argocd.repo_get_failed"), "cicd.argocd.repo_get_failed")
			c.Abort()
			return
		}
		server.ID = repository.ServerID
		app.SourceRepoURL = repository.RepoURL
	} else if strings.Contains(path, "/servers") && id > 0 {
		server.ID = id
	}
	if server.ID == 0 {
		server.ID, _ = strconv.ParseUint(c.Query("serverId"), 10, 64)
	}
	if server.ID > 0 {
		if err := h.service.db.WithContext(ctx).First(&server, server.ID).Error; err != nil {
			failAction(c, resourceReadError(ctx, err, "cicd.argocd.server_get_failed"), "cicd.argocd.server_get_failed")
			c.Abort()
			return
		}
	}
	sources := []string{}
	if app.SourceRepoURL != "" {
		sources = append(sources, app.SourceRepoURL)
	}
	applicationPUT := c.Request.Method == http.MethodPut && path == "/cicd/argocd/applications/:id"
	if !applicationPUT && input.SourceRepoURL != "" && input.SourceRepoURL != app.SourceRepoURL {
		sources = append(sources, input.SourceRepoURL)
	}
	if !applicationPUT && input.RepoURL != "" {
		sources = append(sources, input.RepoURL)
	}
	action := c.Request.Method + " " + path
	if err := h.service.authorizeAction(ctx, actor, action, &app, &server, sources); err != nil {
		failAction(c, err, common.MessageRequestFailed)
		c.Abort()
		return
	}
	if applicationPUT {
		proposed := app
		applyApplicationUpdate(&proposed, updateReq)
		if err := h.service.authorizeAction(ctx, actor, action, &proposed, &server, []string{proposed.SourceRepoURL}); err != nil {
			failAction(c, err, common.MessageRequestFailed)
			c.Abort()
			return
		}
	}
	// Collection authorization does not authorize its individual resources.
	c.Next()
}

func resourceReadError(ctx context.Context, err error, key string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return common.NewNotFound(key)
	}
	logging.Error("Argo CD resource read failed", zap.String("error_type", fmt.Sprintf("%T", err)))
	return common.NewInternal(key)
}

// Only a proven policy denial or vanished record can be filtered. Everything
// else must abort the collection before any partial projection is returned.
func filterableResourceError(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, common.ErrNotFound) || errors.Is(err, common.ErrForbidden)
}

func failCollection(c *gin.Context, err error, key string) {
	logging.Error("Argo CD collection read failed", zap.String("error_type", fmt.Sprintf("%T", err)))
	failAction(c, err, key)
}

func (h *ArgoCDHandler) PreviewApplication(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req ApplicationPreviewRequest
	if c.ShouldBindJSON(&req) != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	common.SetAuditMetadata(c, "预览 ArgoCD 应用变更", common.BusinessOther)
	resp, err := h.service.PreviewApplicationContext(c.Request.Context(), id, req, requestActor(c))
	if err != nil {
		failAction(c, err, "cicd.argocd.app_sync_failed")
		return
	}
	common.Success(c, resp)
}

func failAction(c *gin.Context, err error, fallback string) {
	// Never pass unknown provider/database errors to FailWithError: it logs
	// raw errors. Domain classification is sufficient at this boundary.
	code := common.CodeForError(err)
	message := fallback
	if code == common.CodeForbidden {
		message = "common.forbidden"
	}
	if code == common.CodeParamInvalid {
		message = msgParamInvalid
	}
	common.SetAuditErrorMsg(c, message)
	if errors.Is(err, common.ErrConflict) {
		common.FailWithCode(c, http.StatusConflict, message)
		return
	}
	common.Fail(c, code, message)
}
