package argocd

import (
	"strconv"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

	"github.com/gin-gonic/gin"
)

// msgParamInvalid is the i18n key returned for a malformed request. The key is part of the
// frontend message catalogue contract, so it stays a single package constant.
const msgParamInvalid = "common.param_invalid"

// ArgoCDHandler handles ArgoCD HTTP requests.
//
//nolint:revive // the exported handler name keeps the Argo CD domain prefix for API clarity.
type ArgoCDHandler struct {
	service *ArgoCDService
}

// NewArgoCDHandler creates a new handler.
func NewArgoCDHandler(service *ArgoCDService) *ArgoCDHandler {
	return &ArgoCDHandler{service: service}
}

// RegisterRoutes registers all ArgoCD routes.
func (h *ArgoCDHandler) RegisterRoutes(r gin.IRoutes) {
	// Servers
	r.POST("/cicd/argocd/servers", h.AddServer)
	r.GET("/cicd/argocd/servers", h.ListServers)
	r.GET("/cicd/argocd/servers/:id", h.GetServer)
	r.PUT("/cicd/argocd/servers/:id", h.UpdateServer)
	r.DELETE("/cicd/argocd/servers/:id", h.DeleteServer)
	r.POST("/cicd/argocd/servers/:id/test", h.TestServerConnection)

	// Repositories
	r.POST("/cicd/argocd/repositories", h.AddRepository)
	r.GET("/cicd/argocd/repositories", h.ListRepositories)
	r.GET("/cicd/argocd/repositories/:id", h.GetRepository)
	r.PUT("/cicd/argocd/repositories/:id", h.UpdateRepository)
	r.DELETE("/cicd/argocd/repositories/:id", h.DeleteRepository)

	// Applications
	r.POST("/cicd/argocd/applications", h.CreateApplication)
	r.GET("/cicd/argocd/applications", h.ListApplications)
	r.GET("/cicd/argocd/applications/:id", h.GetApplication)
	r.PUT("/cicd/argocd/applications/:id", h.UpdateApplication)
	r.DELETE("/cicd/argocd/applications/:id", h.DeleteApplication)
	r.POST("/cicd/argocd/applications/:id/sync", h.SyncApplication)
	r.POST("/cicd/argocd/applications/:id/refresh", h.RefreshApplication)
	r.POST("/cicd/argocd/applications/:id/rollback", h.RollbackApplication)
	r.GET("/cicd/argocd/applications/:id/history", h.GetSyncHistory)
}

// --- Server ---

// AddServer registers an Argo CD server from the request body.
func (h *ArgoCDHandler) AddServer(c *gin.Context) {
	var req AddServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	createdBy := strconv.FormatUint(common.GetUserID(c), 10)
	common.SetAuditMetadata(c, "添加 ArgoCD 服务端", common.BusinessInsert)

	resp, err := h.service.AddServer(req, createdBy)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.server_add_failed")
		return
	}
	common.Success(c, resp)
}

// ListServers returns the registered Argo CD servers.
func (h *ArgoCDHandler) ListServers(c *gin.Context) {
	resp, err := h.service.ListServers()
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.server_list_failed")
		return
	}
	common.Success(c, resp)
}

// GetServer returns one Argo CD server by id.
func (h *ArgoCDHandler) GetServer(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.service.GetServer(id)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.server_get_failed")
		return
	}
	common.Success(c, resp)
}

// UpdateServer applies the request body to the Argo CD server identified by the path id.
func (h *ArgoCDHandler) UpdateServer(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req UpdateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	common.SetAuditMetadata(c, "更新 ArgoCD 服务端", common.BusinessUpdate)

	if err := h.service.UpdateServer(id, req); err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.server_update_failed")
		return
	}
	common.Success(c, gin.H{"updated": true})
}

// DeleteServer removes the Argo CD server identified by the path id.
func (h *ArgoCDHandler) DeleteServer(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	common.SetAuditMetadata(c, "删除 ArgoCD 服务端", common.BusinessDelete)

	if err := h.service.DeleteServer(id); err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.server_delete_failed")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

// TestServerConnection probes the Argo CD server identified by the path id.
func (h *ArgoCDHandler) TestServerConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	ok, err := h.service.TestServerConnection(id)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.server_test_failed")
		return
	}
	common.Success(c, gin.H{"connected": ok})
}

// --- Repository ---

// AddRepository registers a Git repository with the Argo CD server named by serverId.
func (h *ArgoCDHandler) AddRepository(c *gin.Context) {
	var req AddRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	createdBy := strconv.FormatUint(common.GetUserID(c), 10)
	common.SetAuditMetadata(c, "添加 Git 仓库", common.BusinessInsert)

	resp, err := h.service.AddRepository(req, createdBy)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.repo_add_failed")
		return
	}
	common.Success(c, resp)
}

// ListRepositories returns the repositories of the Argo CD server named by serverId.
func (h *ArgoCDHandler) ListRepositories(c *gin.Context) {
	serverID, _ := strconv.ParseUint(c.Query("serverId"), 10, 64)
	resp, err := h.service.ListRepositories(serverID)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.repo_list_failed")
		return
	}
	common.Success(c, resp)
}

// GetRepository returns one Argo CD repository by id.
func (h *ArgoCDHandler) GetRepository(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.service.GetRepository(id)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.repo_get_failed")
		return
	}
	common.Success(c, resp)
}

// UpdateRepository applies the request body to the Argo CD repository identified by the path id.
func (h *ArgoCDHandler) UpdateRepository(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req AddRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	common.SetAuditMetadata(c, "更新 Git 仓库", common.BusinessUpdate)

	if err := h.service.UpdateRepository(id, req); err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.repo_update_failed")
		return
	}
	common.Success(c, gin.H{"updated": true})
}

// DeleteRepository removes the Argo CD repository identified by the path id.
func (h *ArgoCDHandler) DeleteRepository(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	common.SetAuditMetadata(c, "删除 Git 仓库", common.BusinessDelete)

	if err := h.service.DeleteRepository(id); err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.repo_delete_failed")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

// --- Application ---

// CreateApplication deploys a new Argo CD application from the request body.
func (h *ArgoCDHandler) CreateApplication(c *gin.Context) {
	var req CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	createdBy := strconv.FormatUint(common.GetUserID(c), 10)
	common.SetAuditMetadata(c, "创建 ArgoCD 应用", common.BusinessInsert)

	resp, err := h.service.CreateApplication(req, createdBy)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.app_create_failed")
		return
	}
	common.Success(c, resp)
}

// ListApplications returns the applications of the Argo CD server named by serverId.
func (h *ArgoCDHandler) ListApplications(c *gin.Context) {
	serverID, _ := strconv.ParseUint(c.Query("serverId"), 10, 64)
	resp, err := h.service.ListApplications(serverID)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.app_list_failed")
		return
	}
	common.Success(c, resp)
}

// GetApplication returns one Argo CD application by id.
func (h *ArgoCDHandler) GetApplication(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.service.GetApplication(id)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.app_get_failed")
		return
	}
	common.Success(c, resp)
}

// UpdateApplication applies the request body to the Argo CD application identified by the path id.
func (h *ArgoCDHandler) UpdateApplication(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req UpdateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	common.SetAuditMetadata(c, "更新 ArgoCD 应用", common.BusinessUpdate)

	if err := h.service.UpdateApplication(id, req); err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.app_update_failed")
		return
	}
	common.Success(c, gin.H{"updated": true})
}

// DeleteApplication removes the Argo CD application identified by the path id.
func (h *ArgoCDHandler) DeleteApplication(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	common.SetAuditMetadata(c, "删除 ArgoCD 应用", common.BusinessDelete)

	if err := h.service.DeleteApplication(id); err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.app_delete_failed")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

// SyncApplication syncs the Argo CD application identified by the path id to the requested revision.
func (h *ArgoCDHandler) SyncApplication(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req SyncApplicationRequest
	_ = c.ShouldBindJSON(&req) // optional body

	common.SetAuditMetadata(c, "同步 ArgoCD 应用", common.BusinessUpdate)

	resp, err := h.service.SyncApplication(id, req.Revision)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.app_sync_failed")
		return
	}
	common.Success(c, resp)
}

// RefreshApplication asks Argo CD to re-reconcile the application identified by the path id.
func (h *ArgoCDHandler) RefreshApplication(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	common.SetAuditMetadata(c, "刷新 ArgoCD 应用", common.BusinessUpdate)

	if err := h.service.RefreshApplication(id); err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.app_refresh_failed")
		return
	}
	common.Success(c, gin.H{"refreshed": true})
}

// RollbackApplication re-syncs the application identified by the path id to the revision in the request body.
func (h *ArgoCDHandler) RollbackApplication(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req SyncApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Revision == "" {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	common.SetAuditMetadata(c, "回滚 ArgoCD 应用", common.BusinessUpdate)

	resp, err := h.service.RollbackApplication(id, req.Revision)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.app_rollback_failed")
		return
	}
	common.Success(c, resp)
}

// GetSyncHistory returns the recorded sync operations of the application identified by the path id.
func (h *ArgoCDHandler) GetSyncHistory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.service.GetSyncHistory(id)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cicd.argocd.history_get_failed")
		return
	}
	common.Success(c, resp)
}
