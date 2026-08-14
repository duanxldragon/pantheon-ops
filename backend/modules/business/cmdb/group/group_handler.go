package group

import (
	"encoding/json"
	"strconv"

	"pantheon-base/pkg/common"

	"github.com/gin-gonic/gin"
)

const (
	msgParamInvalid = "common.param_invalid"
	groupIDRoute    = "/groups/:id"
)

type GroupHandler struct {
	svc *GroupService
}

func NewGroupHandler(svc *GroupService) *GroupHandler {
	return &GroupHandler{svc: svc}
}

func (h *GroupHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/groups", h.List)
	r.GET(groupIDRoute, h.GetByID)
	r.GET("/groups/:id/members", h.GetMembers)
	r.POST("/groups", h.Create)
	r.PUT(groupIDRoute, h.Update)
	r.DELETE(groupIDRoute, h.Delete)
}

func (h *GroupHandler) List(c *gin.Context) {
	items, err := h.svc.List(common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdbgroup.list_failed")
		return
	}
	common.Success(c, items)
}

func (h *GroupHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.GetByID(id, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdbgroup.not_found")
		return
	}
	common.Success(c, resp)
}

func (h *GroupHandler) GetMembers(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	members, group, err := h.svc.GetMembers(id, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdbgroup.not_found")
		return
	}
	memberMaps := make([]HostResponse, len(members))
	for i, m := range members {
		memberMaps[i] = hostToResponse(&m)
	}
	common.Success(c, GroupMemberListResponse{GroupID: group.ID, GroupName: group.Name, Members: memberMaps})
}

func (h *GroupHandler) Create(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.Create(req, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdbgroup.create_failed")
		return
	}
	common.Success(c, resp)
}

func (h *GroupHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.Update(id, req, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdbgroup.update_failed")
		return
	}
	common.Success(c, resp)
}

func (h *GroupHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if err := h.svc.Delete(id); err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdbgroup.delete_failed")
		return
	}
	common.Success(c, nil)
}

func hostToResponse(h *Host) HostResponse {
	var labels []LabelEntry
	if len(h.LabelValues) > 0 {
		_ = json.Unmarshal(h.LabelValues, &labels)
	}
	if labels == nil {
		labels = []LabelEntry{}
	}
	return HostResponse{
		ID:          h.ID,
		Hostname:    h.Hostname,
		IP:          h.IP,
		Status:      h.Status,
		OS:          h.OS,
		OSVersion:   h.OSVersion,
		CPUCores:    h.CPUCores,
		MemoryGB:    h.MemoryGB,
		DiskGB:      h.DiskGB,
		DeptID:      h.DeptID,
		LabelValues: labels,
	}
}
