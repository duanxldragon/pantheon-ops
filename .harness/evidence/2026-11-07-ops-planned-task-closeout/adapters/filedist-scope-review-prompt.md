Independent read-only source review. No tools. Report proved P1/P2 defects or APPROVE for supplied scope only, maximum 250 words. Formal O1 FileDist gap: tables have username-only ownership and address-only targets, no numeric owner/dept/stable CMDB ID. This bounded containment deliberately denies all restricted-scope roles until reviewed ownership/selection integration. Existing group chain is TokenAuth -> Casbin -> IAM DataScope -> this business gate. DataScope context is only server-set. Missing/unknown/empty scope is denied; trusted IsAdmin or explicit trim-space all admitted. No schema/API/worker behavior changes. Dedicated groups /filedist and /business/filedist; no public handler registration elsewhere. Preserve shared Base ownership. Review gate/placement, bypasses, resource I/O before denial, compatibility and tests; omitted IAM internals/storage/worker code are not approved.

FILE backend/modules/business/filedist/filedist_handler.go
package filedist

import (
"io"
"net/http"
"os"
"path/filepath"
"strconv"
"strings"

    "github.com/duanxldragon/pantheon-base/backend/pkg/common"

    "github.com/gin-gonic/gin"

)

// FileDistHandler handles file distribution API endpoints.
//
//nolint:revive // the exported handler name keeps the file-distribution domain prefix for API clarity.
type FileDistHandler struct {
storageService *StorageService
distributionService *DistributionService
}

// NewFileDistHandler creates a new FileDistHandler.
func NewFileDistHandler(storageService *StorageService, distributionService *DistributionService) *FileDistHandler {
return &FileDistHandler{
storageService: storageService,
distributionService: distributionService,
}
}

// RegisterRoutes registers all file distribution routes on a protected group.
func (h *FileDistHandler) RegisterRoutes(filedist gin.IRoutes) {
filedist.Use(requireFileDistUnrestrictedScope)
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

// FileDist has username-only ownership and address-only targets. Until a
// reviewed ownership/target contract exists, restricted scopes cannot safely
// access this module. Only trust the IAM-resolved context, never request input.
// The supplied group already enforces authentication, Casbin and DataScope.
func requireFileDistUnrestrictedScope(c *gin.Context) {
scope := common.GetDataScope(c)
if scope == nil || (!scope.IsAdmin && strings.TrimSpace(scope.Mode) != common.DataScopeModeAll) {
common.FailWithCode(c, http.StatusForbidden, "permission.denied")
c.Abort()
return
}
c.Next()
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

FILE backend/modules/business/filedist/data_scope_test.go
package filedist

import (
"encoding/json"
"io"
"net/http"
"net/http/httptest"
"strings"
"testing"

    "github.com/duanxldragon/pantheon-base/backend/pkg/common"
    "github.com/gin-gonic/gin"

)

func TestFileDistRoutesRejectUnsupportedDataScope(t *testing.T) {
gin.SetMode(gin.TestMode)
cases := []struct {
name string
scope any
}{
{"missing", nil}, {"wrong-type", "all"},
{"typed-nil", (*common.DataScopeReq)(nil)},
{"empty", &common.DataScopeReq{}},
{"whitespace", &common.DataScopeReq{Mode: " "}},
{"self-numeric-username", &common.DataScopeReq{UserID: 1, Mode: common.DataScopeModeSelf}},
{"dept", &common.DataScopeReq{DeptID: 1, Mode: common.DataScopeModeDept}},
{"children", &common.DataScopeReq{DeptID: 1, DeptIDs: []uint64{1, 2}, Mode: common.DataScopeModeDeptAndChildren}},
{"custom", &common.DataScopeReq{DeptIDs: []uint64{1, 2}, Mode: common.DataScopeModeCustom}},
{"unknown", &common.DataScopeReq{Mode: "tenant"}},
}
for _, tc := range cases {
t.Run(tc.name, func(t *testing.T) {
router := gin.New()
router.Use(gin.RecoveryWithWriter(io.Discard))
router.Use(func(c *gin.Context) {
if tc.scope != nil {
c.Set(common.DataScopeContextKey, tc.scope)
}
c.Set("username", "1")
c.Next()
})
handler := NewFileDistHandler(nil, nil)
handler.RegisterRoutes(router.Group("/filedist"))
handler.RegisterRoutes(router.Group("/business/filedist"))
// Enumerate the registered surface so newly added aliases/actions are
// also protected. Nil services prove denial precedes any handler I/O.
for _, route := range router.Routes() {
t.Run(route.Method+route.Path, func(t *testing.T) {
path := strings.ReplaceAll(route.Path, ":id", "1")
request := httptest.NewRequest(route.Method, path+"?scope=all&isAdmin=true", strings.NewReader(`{"scope":"all","isAdmin":true}`))
request.Header.Set("Content-Type", "application/json")
request.Header.Set("X-Data-Scope", "all")
request.Header.Set("X-Admin", "true")
recorder := httptest.NewRecorder()
router.ServeHTTP(recorder, request)
var response struct {
Code int `json:"code"`
Message string `json:"message"`
}
if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
t.Fatalf("expected denial envelope, HTTP %d: %s", recorder.Code, recorder.Body.String())
}
if response.Code != http.StatusForbidden || response.Message != "permission.denied" {
t.Fatalf("unsupported scope reached handler: HTTP %d: %s", recorder.Code, recorder.Body.String())
}
})
}
})
}
}

func TestFileDistExplicitUnrestrictedScope(t *testing.T) {
gin.SetMode(gin.TestMode)
for _, scope := range []*common.DataScopeReq{
{UserID: 1, Mode: common.DataScopeModeAll},
{UserID: 1, Mode: " all "},
{UserID: 1, IsAdmin: true, Mode: common.DataScopeModeSelf},
{UserID: 1, IsAdmin: true},
} {
router := gin.New()
calls := 0
router.GET("/allowed", func(c *gin.Context) {
c.Set(common.DataScopeContextKey, scope)
c.Next()
}, requireFileDistUnrestrictedScope, func(c *gin.Context) {
calls++
c.Status(http.StatusNoContent)
})
recorder := httptest.NewRecorder()
router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/allowed", nil))
if recorder.Code != http.StatusNoContent || calls != 1 {
t.Fatalf("trusted all/admin scope blocked: %+v HTTP %d calls %d", scope, recorder.Code, calls)
}
}
}

func TestFileDistScopeGateDoesNotAffectSiblingRoutes(t *testing.T) {
router := gin.New()
NewFileDistHandler(nil, nil).RegisterRoutes(router.Group("/filedist"))
router.GET("/other", func(c *gin.Context) { c.Status(http.StatusNoContent) })
recorder := httptest.NewRecorder()
router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/other", nil))
if recorder.Code != http.StatusNoContent {
t.Fatalf("sibling route blocked: %d", recorder.Code)
}
}

FILE backend/modules/business/filedist/module.go
package filedist

import (
"context"
"strings"

    "github.com/duanxldragon/pantheon-base/backend/pkg/contracts"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

)

// FileDistModule wires the storage and file distribution services into the
// business router. The module owns the transfer workers used by distribution
// tasks and keeps the public registration surface small for the overlay.
//
//nolint:revive // the exported module name keeps the file-distribution domain prefix for API clarity.
type FileDistModule struct {
db *gorm.DB
storageService *StorageService
distributionService *DistributionService
transferEngine *TransferEngine
handler *FileDistHandler
}

// InitFileDistModule creates the file distribution module rooted at uploadDir.
// The current storage service persists files under the configured local path;
// uploadDir is retained as the default base path for the first local storage.
func InitFileDistModule(r *gin.RouterGroup, db *gorm.DB, uploadDir string) *FileDistModule {
_ = r
if strings.TrimSpace(uploadDir) == "" {
uploadDir = "./uploads"
}

    storageService := NewStorageServiceWithBasePath(db, uploadDir)
    transferEngine := NewTransferEngine(db, storageService, 5)
    distributionService := NewDistributionService(db, storageService, transferEngine)

    module := &FileDistModule{
    	db:                  db,
    	storageService:      storageService,
    	distributionService: distributionService,
    	transferEngine:      transferEngine,
    }
    module.handler = NewFileDistHandler(storageService, distributionService)

    // Start workers once during server assembly so tasks created through the
    // HTTP API are processed without requiring a second lifecycle hook.
    transferEngine.Start(context.Background())

    return module

}

// Migrate creates the storage, uploaded-file, and distribution-task tables.
func (m *FileDistModule) Migrate(_ *gorm.DB) error {
if err := m.storageService.Migrate(); err != nil {
return err
}
if err := m.distributionService.Migrate(); err != nil {
return err
}
if err := m.storageService.EnsureDefaultStorage("system"); err != nil {
return err
}
return seedFileDistMenusAndI18n(m.db)
}

// RegisterRoutes registers all file distribution endpoints.
func (m *FileDistModule) RegisterRoutes(r *gin.RouterGroup) {
m.handler.RegisterRoutes(contracts.DataScopedGroup(r, "/filedist", m.db))
m.handler.RegisterRoutes(contracts.DataScopedGroup(r, "/business/filedist", m.db))
}

// Stop terminates transfer workers. It is safe to call more than once.
func (m *FileDistModule) Stop() {
if m != nil && m.transferEngine != nil {
m.transferEngine.Stop()
}
}
