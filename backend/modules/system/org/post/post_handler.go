package post

import (
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/impexp"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
)

type PostHandler struct {
	service *PostService
}

func NewPostHandler(s *PostService) *PostHandler {
	return &PostHandler{service: s}
}

// svc returns the request-scoped service view bound to the resolved tenant
// context (contract §3.3). A nil context returns the shared service; compat
// binds the platform-global scope (tenant_id=0). Admin operations must use this
// instead of h.service directly, otherwise they read and write platform-global
// posts.
func (h *PostHandler) svc(c *gin.Context) *PostService {
	return h.service.WithTenantContext(tenant.FromGin(c))
}

func (h *PostHandler) GetPostList(c *gin.Context) {
	var query PostListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}

	list, err := h.svc(c).ListPosts(&query)
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
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}

	post, err := h.svc(c).CreatePost(&req)
	if err != nil {
		common.FailWithError(c, common.CodeForError(err), err, common.MessageRequestFailed)
		return
	}
	common.Success(c, post)
}

func (h *PostHandler) UpdatePost(c *gin.Context) {
	common.SetAuditMetadata(c, "post.update.title", common.BusinessUpdate)
	var req PostUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}

	postID, ok := common.ParseUintParam(c, "id")
	if !ok {
		return
	}

	post, err := h.svc(c).UpdatePost(postID, &req)
	if err != nil {
		common.FailWithError(c, common.CodeForError(err), err, common.MessageRequestFailed)
		return
	}
	common.Success(c, post)
}

func (h *PostHandler) BatchUpdatePostStatus(c *gin.Context) {
	common.SetAuditMetadata(c, "post.batch_status.title", common.BusinessUpdate)

	var req PostBatchStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}

	updatedCount, err := h.svc(c).BatchUpdatePostStatus(req.PostIDs, req.Status)
	if err != nil {
		common.FailWithError(c, common.CodeForError(err), err, common.MessageRequestFailed)
		return
	}
	common.Success(c, gin.H{"updatedCount": updatedCount})
}

func (h *PostHandler) DeletePost(c *gin.Context) {
	common.SetAuditMetadata(c, "post.delete.title", common.BusinessDelete)
	postID, ok := common.ParseUintParam(c, "id")
	if !ok {
		return
	}

	if err := h.svc(c).DeletePost(postID); err != nil {
		common.FailWithError(c, common.CodeForError(err), err, common.MessageRequestFailed)
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *PostHandler) BatchDeletePosts(c *gin.Context) {
	common.SetAuditMetadata(c, "post.batch_delete.title", common.BusinessDelete)

	var req common.BatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}
	resp := common.BatchDelete(req.IDs, h.svc(c).DeletePost)
	common.Success(c, resp)
}

func (h *PostHandler) ExportPosts(c *gin.Context) {
	common.SetAuditMetadata(c, "post.export.title", common.BusinessExport)

	var query PostListQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}

	file, err := h.svc(c).ExportPosts(&query)
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

	records, ok := impexp.ReadImportCSV(c)
	if !ok {
		return
	}

	result, err := h.svc(c).ImportPosts(records)
	if err != nil {
		common.Fail(c, common.CodeError, "post.import.error")
		return
	}
	common.Success(c, result)
}
