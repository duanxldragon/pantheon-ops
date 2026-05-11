package config

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/impexp"
	uploadpkg "pantheon-ops/backend/pkg/upload"
)

type SettingHandler struct {
	service       *SettingService
	uploadService *uploadpkg.Service
}

func NewSettingHandler(service *SettingService, uploadService *uploadpkg.Service) *SettingHandler {
	return &SettingHandler{service: service, uploadService: uploadService}
}

func (h *SettingHandler) GetSettingList(c *gin.Context) {
	var query SettingListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	items, err := h.service.List(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "setting.list.error")
		return
	}
	common.Success(c, items)
}

func (h *SettingHandler) GetSettingOverview(c *gin.Context) {
	overview, err := h.service.GetOverview()
	if err != nil {
		common.Fail(c, common.CodeError, "setting.overview.error")
		return
	}
	common.Success(c, overview)
}

func (h *SettingHandler) GetSettingGroup(c *gin.Context) {
	group, err := h.service.GetGroup(c.Param("groupKey"))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, group)
}

func (h *SettingHandler) UpdateSettingGroup(c *gin.Context) {
	var req SettingGroupUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	groupKey := c.Param("groupKey")
	c.Set("operationLog.title", settingAuditTitle)
	c.Set("operationLog.businessType", settingAuditBusinessType)
	successPayload := ""
	if payload, err := h.service.BuildAuditPayload(groupKey, &req, false); err == nil && payload != "" {
		c.Set("operationLog.param", payload)
	}
	if payload, err := h.service.BuildAuditPayload(groupKey, &req, true); err == nil {
		successPayload = payload
	}

	group, err := h.service.UpdateGroup(groupKey, &req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
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
	resp, err := h.service.GetPublicSettings()
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
	var req SettingCacheRefreshReq
	if err := c.ShouldBindJSON(&req); err != nil && c.Request.ContentLength > 0 {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	resp, err := h.service.RefreshSettingCache(req.GroupKeys)
	if err != nil {
		common.Fail(c, common.CodeError, "setting.cache.refresh.error")
		return
	}
	common.Success(c, resp)
}

func (h *SettingHandler) GetSettingAuditList(c *gin.Context) {
	var query SettingAuditQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	page, err := h.service.ListAudit(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "setting.audit.list.error")
		return
	}
	common.Success(c, page)
}

func (h *SettingHandler) ExportSettingAudit(c *gin.Context) {
	var query SettingAuditQuery
	if err := c.ShouldBindJSON(&query); err != nil && c.Request.ContentLength > 0 {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	file, err := h.service.ExportAudit(&query)
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
	common.SetAuditMetadata(c, "上传文件", common.BusinessImport)
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

	stored, err := h.uploadService.Store(fileHeader, c.DefaultQuery("scope", "general"), requestBaseURL(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, stored)
}

func (h *SettingHandler) ServeUploadedFile(c *gin.Context) {
	if h.uploadService == nil {
		common.Fail(c, common.CodeError, "upload.config.unavailable")
		return
	}

	objectKey := strings.TrimLeft(c.Param("filepath"), "/")
	if objectKey == "" {
		common.Fail(c, common.CodeParamInvalid, "upload.file.not_found")
		return
	}

	filePath, err := h.uploadService.ResolveLocalPath(objectKey)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "upload.file.not_found")
		return
	}
	if _, statErr := os.Stat(filePath); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			common.Fail(c, common.CodeError, "upload.file.not_found")
			return
		}
		common.Fail(c, common.CodeError, "upload.file.read.error")
		return
	}

	if contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(filePath))); contentType != "" {
		c.Header("Content-Type", contentType)
	}
	c.File(filePath)
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
