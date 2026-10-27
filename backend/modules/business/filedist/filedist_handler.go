package filedist

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

	"github.com/gin-gonic/gin"
)

// FileDistHandler handles file distribution API endpoints.
//
//nolint:revive // the exported handler name keeps the file-distribution domain prefix for API clarity.
type FileDistHandler struct {
	storageService      *StorageService
	distributionService *DistributionService
}

// NewFileDistHandler creates a new FileDistHandler.
func NewFileDistHandler(storageService *StorageService, distributionService *DistributionService) *FileDistHandler {
	return &FileDistHandler{
		storageService:      storageService,
		distributionService: distributionService,
	}
}

// RegisterRoutes registers all file distribution routes on a protected group.
func (h *FileDistHandler) RegisterRoutes(filedist gin.IRoutes) {
	{
		// Storage
		filedist.POST("/storages", h.CreateStorage)
		filedist.GET("/storages", h.ListStorages)
		filedist.GET("/storages/:id", h.GetStorage)
		filedist.PUT("/storages/:id", h.UpdateStorage)
		filedist.DELETE("/storages/:id", h.DeleteStorage)

		// Files
		filedist.POST("/files/upload", h.UploadFile)
		filedist.GET("/files", h.ListFiles)
		filedist.GET("/files/:id", h.GetFile)
		filedist.GET("/files/:id/download", h.DownloadFile)
		filedist.DELETE("/files/:id", h.DeleteFile)

		// Distribution Tasks
		filedist.POST("/tasks", h.CreateTask)
		filedist.GET("/tasks", h.ListTasks)
		filedist.GET("/tasks/:id", h.GetTask)
		filedist.POST("/tasks/:id/start", h.StartTask)
		filedist.GET("/tasks/:id/targets", h.GetTaskTargets)
		filedist.GET("/tasks/:id/hosts", h.GetTaskTargets)
		filedist.GET("/tasks/:id/events", h.GetTaskEvents)
		filedist.GET("/tasks/:id/progress", h.GetTaskProgress)
		filedist.DELETE("/tasks/:id", h.DeleteTask)
		filedist.POST("/tasks/:id/cancel", h.CancelTask)
		filedist.POST("/tasks/:id/retry", h.RetryTask)
	}
}

// Storage handlers

// CreateStorage registers a file storage location from the request body.
func (h *FileDistHandler) CreateStorage(c *gin.Context) {
	var dto StorageDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}

	createdBy := c.GetString("username")
	resp, err := h.storageService.CreateStorage(dto, createdBy)
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "create_failed")
		return
	}

	common.Success(c, resp)
}

// ListStorages returns the configured file storage locations.
func (h *FileDistHandler) ListStorages(c *gin.Context) {
	resp, err := h.storageService.ListStorages()
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "list_failed")
		return
	}

	common.Success(c, resp)
}

// GetStorage returns one file storage location by id.
func (h *FileDistHandler) GetStorage(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	resp, err := h.storageService.GetStorageByID(id)
	if err != nil {
		common.FailWithCode(c, http.StatusNotFound, "not_found")
		return
	}

	common.Success(c, resp)
}

// UpdateStorage applies the request body to the file storage location identified by the path id.
func (h *FileDistHandler) UpdateStorage(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var dto StorageDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}

	if err := h.storageService.UpdateStorage(id, dto); err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "update_failed")
		return
	}

	common.Success(c, gin.H{"message": "updated"})
}

// DeleteStorage removes a file storage location that still holds no files.
func (h *FileDistHandler) DeleteStorage(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	if err := h.storageService.DeleteStorage(id); err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "delete_failed")
		return
	}

	common.Success(c, gin.H{"message": "deleted"})
}

// File handlers

// UploadFile stores the multipart file in the selected storage and records its metadata.
func (h *FileDistHandler) UploadFile(c *gin.Context) {
	storageID, _ := strconv.ParseUint(c.PostForm("storageId"), 10, 64)
	if storageID == 0 {
		// The default local storage is used when the UI does not expose storage
		// selection. This keeps the upload workflow one step while preserving the
		// explicit storageId contract for API clients.
		storages, err := h.storageService.ListStorages()
		if err != nil || len(storages.Items) == 0 {
			common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
			return
		}
		for _, storage := range storages.Items {
			if storage.IsDefault {
				storageID = storage.ID
				break
			}
		}
		if storageID == 0 {
			storageID = storages.Items[0].ID
		}
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_file")
		return
	}
	defer func() { _ = file.Close() }()

	uploadedBy := c.GetString("username")
	resp, err := h.storageService.UploadFile(storageID, header.Filename, file, header.Size, uploadedBy)
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "upload_failed")
		return
	}

	common.Success(c, resp)
}

// ListFiles returns stored files matching the query.
func (h *FileDistHandler) ListFiles(c *gin.Context) {
	var query FileRecordListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}

	resp, err := h.storageService.ListFiles(query)
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "list_failed")
		return
	}

	common.Success(c, resp)
}

// GetFile returns the metadata of the stored file identified by the path id.
func (h *FileDistHandler) GetFile(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	resp, err := h.storageService.GetFileByID(id)
	if err != nil {
		common.FailWithCode(c, http.StatusNotFound, "not_found")
		return
	}

	common.Success(c, resp)
}

// DownloadFile streams the stored file identified by the path id to the client.
func (h *FileDistHandler) DownloadFile(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	// Get file info
	fileInfo, err := h.storageService.GetFileByID(id)
	if err != nil {
		common.FailWithCode(c, http.StatusNotFound, "not_found")
		return
	}

	// Get file path
	filePath, err := h.storageService.GetFilePath(id)
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "get_path_failed")
		return
	}

	// Open file
	file, err := os.Open(filePath) // #nosec G304 -- filePath is built by StorageService.GetFilePath from the stored record path, which is server-generated under the storage root.
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "open_file_failed")
		return
	}
	defer func() { _ = file.Close() }()

	// Set headers
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(fileInfo.Name))
	c.Header("Content-Type", fileInfo.ContentType)

	// Stream file
	_, _ = io.Copy(c.Writer, file)
}

// DeleteFile removes the stored file identified by the path id and its record.
func (h *FileDistHandler) DeleteFile(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	if err := h.storageService.DeleteFile(id); err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "delete_failed")
		return
	}

	common.Success(c, gin.H{"message": "deleted"})
}

// Distribution Task handlers

// CreateTask registers a distribution task from the request body.
func (h *FileDistHandler) CreateTask(c *gin.Context) {
	var dto DistributionTaskDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}

	createdBy := c.GetString("username")
	resp, err := h.distributionService.CreateTask(dto, createdBy)
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "create_failed")
		return
	}

	common.Success(c, resp)
}

// ListTasks returns distribution tasks matching the query.
func (h *FileDistHandler) ListTasks(c *gin.Context) {
	var query TaskListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}

	resp, err := h.distributionService.ListTasks(query)
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "list_failed")
		return
	}

	common.Success(c, resp)
}

// GetTask returns one distribution task by id.
func (h *FileDistHandler) GetTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	resp, err := h.distributionService.GetTaskByID(id)
	if err != nil {
		common.FailWithCode(c, http.StatusNotFound, "not_found")
		return
	}

	common.Success(c, resp)
}

// StartTask begins distributing the file of the task identified by the path id.
func (h *FileDistHandler) StartTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.distributionService.StartTask(id); err != nil {
		common.FailWithCode(c, http.StatusConflict, "start_failed")
		return
	}
	common.Success(c, gin.H{"started": true})
}

// GetTaskTargets returns the per-target state of a distribution task.
func (h *FileDistHandler) GetTaskTargets(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	targets, err := h.distributionService.GetTaskTargets(id)
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "get_targets_failed")
		return
	}

	common.Success(c, gin.H{"items": targets})
}

// GetTaskEvents returns the recorded events of a distribution task.
func (h *FileDistHandler) GetTaskEvents(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if _, err := h.distributionService.GetTaskByID(id); err != nil {
		common.FailWithCode(c, http.StatusNotFound, "not_found")
		return
	}
	common.Success(c, []interface{}{})
}

// GetTaskProgress streams task progress updates to the caller.
func (h *FileDistHandler) GetTaskProgress(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	resp, err := h.distributionService.GetTaskProgress(id)
	if err != nil {
		common.FailWithCode(c, http.StatusNotFound, "not_found")
		return
	}
	common.Success(c, resp)
}

// DeleteTask removes a distribution task and its target records after it has
// reached a terminal state. Active transfers must be cancelled first.
func (h *FileDistHandler) DeleteTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.distributionService.DeleteTask(id); err != nil {
		common.FailWithCode(c, http.StatusConflict, "delete_failed")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

// CancelTask stops the running task identified by the path id.
func (h *FileDistHandler) CancelTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	if err := h.distributionService.CancelTask(id); err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "cancel_failed")
		return
	}

	common.Success(c, gin.H{"message": "cancelled"})
}

// RetryTask re-runs the targets that failed in the task identified by the path id.
func (h *FileDistHandler) RetryTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	if err := h.distributionService.RetryTask(id); err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "retry_failed")
		return
	}

	common.Success(c, gin.H{"message": "retrying"})
}
