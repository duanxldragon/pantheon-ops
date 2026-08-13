package org

import (
	"strconv"

	"pantheon-base/pkg/common"
	"pantheon-base/pkg/impexp"

	"github.com/gin-gonic/gin"
)

const errRequestFailed = "request.failed"
const errParamInvalid = "param.invalid"

type PostHandler struct {
	service *PostService
}

func NewPostHandler(s *PostService) *PostHandler {
	return &PostHandler{service: s}
}

func (h *PostHandler) GetPostList(c *gin.Context) {
	var query PostListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
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
	common.SetAuditMetadata(c, "post.create.title", common.BusinessInsert)
	var req PostCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	post, err := h.service.CreatePost(&req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, post)
}

func (h *PostHandler) UpdatePost(c *gin.Context) {
	common.SetAuditMetadata(c, "post.update.title", common.BusinessUpdate)
	var req PostUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	post, err := h.service.UpdatePost(postID, &req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, post)
}

func (h *PostHandler) BatchUpdatePostStatus(c *gin.Context) {
	common.SetAuditMetadata(c, "post.batch_status.title", common.BusinessUpdate)

	var req PostBatchStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	updatedCount, err := h.service.BatchUpdatePostStatus(req.PostIDs, req.Status)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{"updatedCount": updatedCount})
}

func (h *PostHandler) DeletePost(c *gin.Context) {
	common.SetAuditMetadata(c, "post.delete.title", common.BusinessDelete)
	postID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	if err := h.service.DeletePost(postID); err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *PostHandler) BatchDeletePosts(c *gin.Context) {
	common.SetAuditMetadata(c, "post.batch_delete.title", common.BusinessDelete)

	var req common.BatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	resp := common.BatchDelete(req.IDs, h.service.DeletePost)
	common.Success(c, resp)
}

func (h *PostHandler) ExportPosts(c *gin.Context) {
	common.SetAuditMetadata(c, "post.export.title", common.BusinessExport)

	var query PostListQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
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
	common.SetAuditMetadata(c, "post.import.title", common.BusinessImport)

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
