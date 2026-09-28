package config

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/impexp"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	uploadpkg "github.com/duanxldragon/pantheon-base/backend/pkg/upload"
	"github.com/gin-gonic/gin"
)

const (
	errParamInvalid       = "param.invalid"
	errRequestFailed      = "request.failed"
	errUploadFileNotFound = "upload.file.not_found"
)

type SettingHandler struct {
	service       *SettingService
	uploadService *uploadpkg.Service
}

// storageDriverLocal is the config value selecting the local disk driver.
const storageDriverLocal = "local"

func NewSettingHandler(service *SettingService, uploadService *uploadpkg.Service) *SettingHandler {
	return &SettingHandler{service: service, uploadService: uploadService}
}

// boundService returns the service view bound to the request tenant context
// (queue-5 settings slice; same canary pattern as DictHandler). Under compat
// the shared service is returned unchanged.
func (h *SettingHandler) boundService(c *gin.Context) *SettingService {
	return h.service.WithTenantContext(tenant.FromGin(c))
}

func (h *SettingHandler) GetSettingList(c *gin.Context) {
	var query SettingListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	items, err := h.boundService(c).List(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "setting.list.error")
		return
	}
	common.Success(c, items)
}

func (h *SettingHandler) GetSettingOverview(c *gin.Context) {
	overview, err := h.boundService(c).GetOverview()
	if err != nil {
		common.Fail(c, common.CodeError, "setting.overview.error")
		return
	}
	common.Success(c, overview)
}

func (h *SettingHandler) GetSettingGroup(c *gin.Context) {
	group, err := h.boundService(c).GetGroup(c.Param("groupKey"))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, group)
}

func (h *SettingHandler) UpdateSettingGroup(c *gin.Context) {
	common.SetAuditMetadata(c, settingAuditTitle, settingAuditBusinessType)

	var req SettingGroupUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	groupKey := c.Param("groupKey")
	successPayload := ""
	if payload, err := h.service.BuildAuditPayload(groupKey, &req, false); err == nil && payload != "" {
		c.Set("operationLog.param", payload)
	}
	if payload, err := h.service.BuildAuditPayload(groupKey, &req, true); err == nil {
		successPayload = payload
	}

	group, err := h.boundService(c).UpdateGroup(groupKey, &req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	if successPayload != "" {
		c.Set("operationLog.param", successPayload)
	}
	if data, err := json.Marshal(gin.H{"updated": true, "groupKey": groupKey}); err == nil {
		c.Set("operationLog.result", string(data))
	}
	common.Success(c, group)
}

func (h *SettingHandler) GetPublicSettings(c *gin.Context) {
	// Public endpoint resolves per-request tenant context: in multi mode the
	// response carries the tenant's overrides on top of global defaults; in
	// compat mode the context is nil/global and behavior is unchanged.
	resp, err := h.boundService(c).GetPublicSettings()
	if err != nil {
		common.Fail(c, common.CodeError, "setting.public.error")
		return
	}
	common.Success(c, resp)
}

const (
	settingAuditTitle        = "setting.group.update"
	settingAuditBusinessType = 1001
)

func (h *SettingHandler) RefreshSettingCache(c *gin.Context) {
	common.SetAuditMetadata(c, "setting.cache.refresh.title", common.BusinessUpdate)

	var req SettingCacheRefreshReq
	if err := c.ShouldBindJSON(&req); err != nil && c.Request.ContentLength > 0 {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	resp, err := h.boundService(c).RefreshSettingCache(req.GroupKeys)
	if err != nil {
		common.Fail(c, common.CodeError, "setting.cache.refresh.error")
		return
	}
	common.Success(c, resp)
}

func (h *SettingHandler) GetSettingAuditList(c *gin.Context) {
	var query SettingAuditQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	page, err := h.boundService(c).ListAudit(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "setting.audit.list.error")
		return
	}
	common.Success(c, page)
}

func (h *SettingHandler) ExportSettingAudit(c *gin.Context) {
	common.SetAuditMetadata(c, "setting.audit.export.title", common.BusinessExport)

	var query SettingAuditQuery
	if err := c.ShouldBindJSON(&query); err != nil && c.Request.ContentLength > 0 {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	file, err := h.boundService(c).ExportAudit(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "setting.audit.export.error")
		return
	}
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "setting.audit.export.error")
		return
	}
}

func (h *SettingHandler) UploadFile(c *gin.Context) {
	common.SetAuditMetadata(c, "upload.file.title", common.BusinessImport)
	if h.uploadService == nil {
		common.Fail(c, common.CodeError, "upload.config.unavailable")
		return
	}

	maxBytes, err := h.uploadService.MaxBytes()
	if err == nil && maxBytes > 0 {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "too large") {
			common.Fail(c, common.CodeParamInvalid, "upload.file.too_large")
			return
		}
		common.Fail(c, common.CodeParamInvalid, "upload.file.required")
		return
	}

	stored, err := h.uploadService.StoreWithContext(c.Request.Context(), fileHeader, h.uploadScope(c), requestBaseURL(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, stored)
}

func (h *SettingHandler) ServeUploadedFile(c *gin.Context) {
	if h.uploadService == nil {
		common.Fail(c, common.CodeError, "upload.config.unavailable")
		return
	}

	cfg, err := h.uploadService.LoadConfig()
	if err != nil {
		common.Fail(c, common.CodeError, errUploadFileNotFound)
		return
	}

	objectKey, err := uploadpkg.NormalizeObjectKey(c.Param("filepath"))
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errUploadFileNotFound)
		return
	}
	// Tenancy is enforced from the resolved context only — never request fields.
	if err := uploadpkg.EnforceTenantObjectScope(tenant.FromGin(c), objectKey); err != nil {
		common.Fail(c, common.CodeError, errUploadFileNotFound)
		return
	}

	// S3 driver: serve through the authorized backend path (download
	// authorization slice). Previously this endpoint failed closed for S3,
	// leaving object-store URLs as the only access path — unauthenticated
	// object reads bypassed the tenant isolation the local path enforces.
	if cfg.StorageDriver == "s3" {
		h.serveS3Object(c, objectKey)
		return
	}
	if cfg.StorageDriver != storageDriverLocal {
		common.Fail(c, common.CodeError, errUploadFileNotFound)
		return
	}

	rootPath, err := filepath.Abs(strings.TrimSpace(cfg.LocalPath))
	if err != nil {
		common.Fail(c, common.CodeError, errUploadFileNotFound)
		return
	}

	extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(objectKey)), ".")
	if contentType := mime.TypeByExtension("." + extension); contentType != "" {
		c.Header("Content-Type", contentType)
	}
	// 纵深防御：禁止 MIME 嗅探；非图片类型强制下载，防止伪装内容被浏览器内联渲染。
	c.Header("X-Content-Type-Options", "nosniff")
	switch extension {
	case "jpg", "jpeg", "png", "gif", "webp":
	default:
		c.Header("Content-Disposition", "attachment")
	}
	if !filepath.IsLocal(objectKey) {
		common.Fail(c, common.CodeParamInvalid, errUploadFileNotFound)
		return
	}
	http.ServeFileFS(c.Writer, c.Request, os.DirFS(rootPath), objectKey)
}

// serveS3Object streams an object from the S3-compatible store through the
// authorized serve endpoint. Headers mirror the local-driver path (nosniff,
// attachment for non-images) so browsers treat both drivers identically.
func (h *SettingHandler) serveS3Object(c *gin.Context, objectKey string) {
	object, size, contentType, err := h.uploadService.OpenS3Object(c.Request.Context(), objectKey)
	if err != nil {
		common.Fail(c, common.CodeError, errUploadFileNotFound)
		return
	}
	defer func() {
		_ = object.Close()
	}()

	extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(objectKey)), ".")
	if contentType != "" {
		c.Header("Content-Type", contentType)
	}
	// 纵深防御：禁止 MIME 嗅探；非图片类型强制下载（与本地驱动一致）。
	c.Header("X-Content-Type-Options", "nosniff")
	switch extension {
	case "jpg", "jpeg", "png", "gif", "webp":
	default:
		c.Header("Content-Disposition", "attachment")
	}
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	_, _ = io.Copy(c.Writer, object)
}

// uploadScope builds the object-key prefix for an upload (queue-5 upload
// slice, contract §3.3/Implementation Notes: "Cache and object keys must
// include canonical tenant identity"). The tenant segment comes from the
// resolved tenant context — never from the request — so a tenant subject
// cannot write into another tenant's namespace and keys cannot collide across
// tenants. Compat keeps the legacy layout (no tenant segment).
func (h *SettingHandler) uploadScope(c *gin.Context) string {
	scope := c.DefaultQuery("scope", "general")
	ctx := tenant.FromGin(c)
	if ctx == nil || !ctx.IsMulti() {
		return scope
	}
	return fmt.Sprintf("t%d/%s", ctx.TenantID, scope)
}

func requestBaseURL(c *gin.Context) string {
	scheme := strings.TrimSpace(c.Request.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	return scheme + "://" + c.Request.Host
}
