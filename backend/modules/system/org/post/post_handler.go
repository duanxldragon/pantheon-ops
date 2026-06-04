package org

import (
	"strconv"

	"pantheon-platform/backend/pkg/common"
	"pantheon-platform/backend/pkg/impexp"

	"github.com/gin-gonic/gin"
)

type PostHandler struct {
	service *PostService
}

func NewPostHandler(s *PostService) *PostHandler {
	return &PostHandler{service: s}
}

func (h *PostHandler) GetPostList(c *gin.Context) {
	var query PostListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	list, err := h.service.ListPosts(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "post.list.error")
		return
	}
	common.Success(c, list)
}

func (h *PostHandler) CreatePost(c *gin.Context) {
	common.SetAuditMetadata(c, "新增岗位", common.BusinessInsert)
	var req PostCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	post, err := h.service.CreatePost(&req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, post)
}

func (h *PostHandler) UpdatePost(c *gin.Context) {
	common.SetAuditMetadata(c, "编辑岗位", common.BusinessUpdate)
	var req PostUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	post, err := h.service.UpdatePost(postID, &req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, post)
}

func (h *PostHandler) BatchUpdatePostStatus(c *gin.Context) {
	common.SetAuditMetadata(c, "批量更新岗位状态", common.BusinessUpdate)

	var req PostBatchStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	updatedCount, err := h.service.BatchUpdatePostStatus(req.PostIDs, req.Status)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, gin.H{"updatedCount": updatedCount})
}

func (h *PostHandler) DeletePost(c *gin.Context) {
	common.SetAuditMetadata(c, "删除岗位", common.BusinessDelete)
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	if err := h.service.DeletePost(postID); err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *PostHandler) BatchDeletePosts(c *gin.Context) {
	common.SetAuditMetadata(c, "批量删除岗位", common.BusinessDelete)

	var req common.BatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	resp := common.BatchDelete(req.IDs, h.service.DeletePost)
	common.Success(c, resp)
}

func (h *PostHandler) ExportPosts(c *gin.Context) {
	common.SetAuditMetadata(c, "导出岗位", common.BusinessExport)

	var query PostListQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	file, err := h.service.ExportPosts(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "post.export.error")
		return
	}
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "post.export.error")
	}
}

func (h *PostHandler) DownloadImportTemplate(c *gin.Context) {
	file := h.service.BuildPostImportTemplate()
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "post.import.template.error")
	}
}

func (h *PostHandler) ImportPosts(c *gin.Context) {
	common.SetAuditMetadata(c, "导入岗位", common.BusinessImport)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "import.file.required")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		common.Fail(c, common.CodeError, "import.file.read.error")
		return
	}
	records, err := impexp.ReadCSV(file)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "import.file.invalid_csv")
		return
	}

	result, err := h.service.ImportPosts(records)
	if err != nil {
		common.Fail(c, common.CodeError, "post.import.error")
		return
	}
	common.Success(c, result)
}
