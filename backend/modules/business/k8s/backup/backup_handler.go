package backup

import (
	"strconv"

	"pantheon-base/pkg/common"

	"github.com/gin-gonic/gin"
)

// BackupHandler handles Velero backup HTTP requests.
type BackupHandler struct {
	service *BackupService
}

// NewBackupHandler creates a new handler.
func NewBackupHandler(service *BackupService) *BackupHandler {
	return &BackupHandler{service: service}
}

// RegisterRoutes registers all backup routes.
func (h *BackupHandler) RegisterRoutes(r gin.IRoutes) {
	// Velero status
	r.GET("/backup/velero/status", h.GetVeleroStatus)

	// Storage locations
	r.POST("/backup/locations", h.CreateStorageLocation)
	r.GET("/backup/locations", h.ListStorageLocations)
	r.GET("/backup/locations/:id", h.GetStorageLocation)
	r.PUT("/backup/locations/:id", h.UpdateStorageLocation)
	r.DELETE("/backup/locations/:id", h.DeleteStorageLocation)

	// Backups
	r.POST("/backup/backups", h.CreateBackup)
	r.GET("/backup/backups", h.ListBackups)
	r.GET("/backup/backups/:id", h.GetBackup)
	r.DELETE("/backup/backups/:id", h.DeleteBackup)

	// Schedules
	r.POST("/backup/schedules", h.CreateSchedule)
	r.GET("/backup/schedules", h.ListSchedules)
	r.GET("/backup/schedules/:id", h.GetSchedule)
	r.PUT("/backup/schedules/:id", h.UpdateSchedule)
	r.DELETE("/backup/schedules/:id", h.DeleteSchedule)
	r.POST("/backup/schedules/:id/pause", h.PauseSchedule)
	r.POST("/backup/schedules/:id/resume", h.ResumeSchedule)

	// Restores
	r.POST("/backup/restores", h.CreateRestore)
	r.GET("/backup/restores", h.ListRestores)
	r.GET("/backup/restores/:id", h.GetRestore)
	r.DELETE("/backup/restores/:id", h.DeleteRestore)
}

// --- Velero Status ---

func (h *BackupHandler) GetVeleroStatus(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Query("clusterId"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	dataScope := common.GetDataScope(c)
	resp, err := h.service.GetVeleroStatus(c.Request.Context(), clusterID, dataScope)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.velero_status_failed")
		return
	}
	common.Success(c, resp)
}

// --- Storage Location ---

func (h *BackupHandler) CreateStorageLocation(c *gin.Context) {
	var req CreateStorageLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	createdBy := strconv.FormatUint(common.GetUserID(c), 10)
	common.SetAuditMetadata(c, "创建 Velero 存储位置", common.BusinessInsert)

	resp, err := h.service.CreateStorageLocation(req, createdBy)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.location_create_failed")
		return
	}
	common.Success(c, resp)
}

func (h *BackupHandler) ListStorageLocations(c *gin.Context) {
	resp, err := h.service.ListStorageLocations()
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.location_list_failed")
		return
	}
	common.Success(c, resp)
}

func (h *BackupHandler) GetStorageLocation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.service.GetStorageLocation(id)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.location_get_failed")
		return
	}
	common.Success(c, resp)
}

func (h *BackupHandler) UpdateStorageLocation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	var req CreateStorageLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	common.SetAuditMetadata(c, "更新 Velero 存储位置", common.BusinessUpdate)

	if err := h.service.UpdateStorageLocation(id, req); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.location_update_failed")
		return
	}
	common.Success(c, gin.H{"updated": true})
}

func (h *BackupHandler) DeleteStorageLocation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	common.SetAuditMetadata(c, "删除 Velero 存储位置", common.BusinessDelete)

	if err := h.service.DeleteStorageLocation(id); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.location_delete_failed")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

// --- Backup ---

func (h *BackupHandler) CreateBackup(c *gin.Context) {
	var req CreateBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	createdBy := strconv.FormatUint(common.GetUserID(c), 10)
	dataScope := common.GetDataScope(c)
	common.SetAuditMetadata(c, "创建 Velero 备份", common.BusinessInsert)

	resp, err := h.service.CreateBackup(c.Request.Context(), req, dataScope, createdBy)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.create_failed")
		return
	}
	common.Success(c, resp)
}

func (h *BackupHandler) ListBackups(c *gin.Context) {
	var query BackupListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	dataScope := common.GetDataScope(c)
	resp, err := h.service.ListBackups(query, dataScope)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.list_failed")
		return
	}
	common.Success(c, resp)
}

func (h *BackupHandler) GetBackup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.service.GetBackup(id)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.get_failed")
		return
	}
	common.Success(c, resp)
}

func (h *BackupHandler) DeleteBackup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	common.SetAuditMetadata(c, "删除 Velero 备份", common.BusinessDelete)

	if err := h.service.DeleteBackup(id); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.delete_failed")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

// --- Schedule ---

func (h *BackupHandler) CreateSchedule(c *gin.Context) {
	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	createdBy := strconv.FormatUint(common.GetUserID(c), 10)
	dataScope := common.GetDataScope(c)
	common.SetAuditMetadata(c, "创建定时备份", common.BusinessInsert)

	resp, err := h.service.CreateSchedule(c.Request.Context(), req, dataScope, createdBy)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.schedule_create_failed")
		return
	}
	common.Success(c, resp)
}

func (h *BackupHandler) ListSchedules(c *gin.Context) {
	var query ScheduleListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	dataScope := common.GetDataScope(c)
	resp, err := h.service.ListSchedules(query, dataScope)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.schedule_list_failed")
		return
	}
	common.Success(c, resp)
}

func (h *BackupHandler) GetSchedule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.service.GetSchedule(id)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.schedule_get_failed")
		return
	}
	common.Success(c, resp)
}

func (h *BackupHandler) UpdateSchedule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	common.SetAuditMetadata(c, "更新定时备份", common.BusinessUpdate)

	if err := h.service.UpdateSchedule(id, req); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.schedule_update_failed")
		return
	}
	common.Success(c, gin.H{"updated": true})
}

func (h *BackupHandler) DeleteSchedule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	common.SetAuditMetadata(c, "删除定时备份", common.BusinessDelete)

	if err := h.service.DeleteSchedule(id); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.schedule_delete_failed")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *BackupHandler) PauseSchedule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	common.SetAuditMetadata(c, "暂停定时备份", common.BusinessUpdate)

	if err := h.service.PauseSchedule(id); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.schedule_pause_failed")
		return
	}
	common.Success(c, gin.H{"paused": true})
}

func (h *BackupHandler) ResumeSchedule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	common.SetAuditMetadata(c, "恢复定时备份", common.BusinessUpdate)

	if err := h.service.ResumeSchedule(id); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.schedule_resume_failed")
		return
	}
	common.Success(c, gin.H{"resumed": true})
}

// --- Restore ---

func (h *BackupHandler) CreateRestore(c *gin.Context) {
	var req CreateRestoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	createdBy := strconv.FormatUint(common.GetUserID(c), 10)
	dataScope := common.GetDataScope(c)
	common.SetAuditMetadata(c, "创建 Velero 恢复", common.BusinessInsert)

	resp, err := h.service.CreateRestore(c.Request.Context(), req, dataScope, createdBy)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.restore_create_failed")
		return
	}
	common.Success(c, resp)
}

func (h *BackupHandler) ListRestores(c *gin.Context) {
	var query RestoreListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	dataScope := common.GetDataScope(c)
	resp, err := h.service.ListRestores(query, dataScope)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.restore_list_failed")
		return
	}
	common.Success(c, resp)
}

func (h *BackupHandler) GetRestore(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.service.GetRestore(id)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.restore_get_failed")
		return
	}
	common.Success(c, resp)
}

func (h *BackupHandler) DeleteRestore(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	common.SetAuditMetadata(c, "删除恢复记录", common.BusinessDelete)

	if err := h.service.DeleteRestore(id); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.backup.restore_delete_failed")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}
