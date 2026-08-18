package workload

import (
	"strconv"

	"pantheon-base/pkg/common"

	"github.com/gin-gonic/gin"
)

const msgParamInvalid = "common.param_invalid"

type WorkloadHandler struct {
	svc *WorkloadService
}

func NewWorkloadHandler(svc *WorkloadService) *WorkloadHandler {
	return &WorkloadHandler{svc: svc}
}

func (h *WorkloadHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/workloads", h.List)
	r.GET("/clusters/:clusterId/workloads/:namespace/:kind/:name/pods", h.GetPods)
	r.POST("/clusters/:clusterId/workloads/:namespace/:kind/:name/scale", h.Scale)
	r.POST("/clusters/:clusterId/workloads/:namespace/:kind/:name/restart", h.Restart)
	r.GET("/clusters/:clusterId/pods/:namespace/:podName/logs", h.PodLogs)
}

func (h *WorkloadHandler) List(c *gin.Context) {
	var query WorkloadListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.List(query, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.workload.list_failed")
		return
	}
	common.Success(c, resp)
}

func (h *WorkloadHandler) GetPods(c *gin.Context) {
	clusterID, namespace, kind, name, ok := parseWorkloadParams(c)
	if !ok {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.GetPods(clusterID, namespace, kind, name, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.workload.pods_failed")
		return
	}
	common.Success(c, resp)
}

func (h *WorkloadHandler) Scale(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.workload.audit.scale", common.BusinessUpdate)

	clusterID, namespace, kind, name, ok := parseWorkloadParams(c)
	if !ok {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req ScaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if err := h.svc.Scale(clusterID, namespace, kind, name, req.Replicas, common.GetDataScope(c)); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.workload.scale_failed")
		return
	}
	common.Success(c, nil)
}

func (h *WorkloadHandler) Restart(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.workload.audit.restart", common.BusinessUpdate)

	clusterID, namespace, kind, name, ok := parseWorkloadParams(c)
	if !ok {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if err := h.svc.Restart(clusterID, namespace, kind, name, common.GetDataScope(c)); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.workload.restart_failed")
		return
	}
	common.Success(c, nil)
}

func parseWorkloadParams(c *gin.Context) (uint64, string, string, string, bool) {
	clusterID, err := strconv.ParseUint(c.Param("clusterId"), 10, 64)
	if err != nil {
		return 0, "", "", "", false
	}
	namespace := c.Param("namespace")
	kind := c.Param("kind")
	name := c.Param("name")
	if namespace == "" || kind == "" || name == "" {
		return 0, "", "", "", false
	}
	return clusterID, namespace, kind, name, true
}
