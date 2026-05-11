package common

import (
	"github.com/gin-gonic/gin"
)

const (
	// DataScopeContextKey 存储在 Context 中的数据权限标识
	DataScopeContextKey = "pantheon_data_scope"

	DataScopeModeAll             = "all"
	DataScopeModeSelf            = "self"
	DataScopeModeDept            = "dept"
	DataScopeModeDeptAndChildren = "dept_and_children"
	DataScopeModeCustom          = "custom"
)

// DataScopeReq 数据权限请求上下文
type DataScopeReq struct {
	UserID   uint64
	DeptID   uint64
	DeptIDs  []uint64
	RoleKeys []string
	IsAdmin  bool
	Mode     string
	// Resource 资源标识，用于区分不同业务的过滤规则
	Resource string
}

// GetDataScope 从 Gin Context 获取数据权限配置
func GetDataScope(c *gin.Context) *DataScopeReq {
	if val, ok := c.Get(DataScopeContextKey); ok {
		if req, ok := val.(*DataScopeReq); ok {
			return req
		}
	}
	return nil
}
