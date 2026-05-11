package common

import (
	"github.com/gin-gonic/gin"
)

// 审计日志常量键 (与 middleware/operation_log_middleware.go 保持一致)
const (
	OperationLogTitleKey        = "operationLog.title"
	OperationLogBusinessTypeKey = "operationLog.businessType"
	OperationLogParamKey        = "operationLog.param"
	OperationLogResultKey       = "operationLog.result"
	OperationLogStatusKey       = "operationLog.status"
	OperationLogErrorMsgKey     = "operationLog.errorMsg"
)

// 业务类型枚举
const (
	BusinessOther  = 0 // 其他
	BusinessInsert = 1 // 新增
	BusinessUpdate = 2 // 修改
	BusinessDelete = 3 // 删除
	BusinessGrant  = 4 // 授权
	BusinessExport = 5 // 导出
	BusinessImport = 6 // 导入
	BusinessForce  = 7 // 强退
	BusinessClean  = 8 // 清空
)

// SetAuditMetadata 在 Context 中注入审计元数据
func SetAuditMetadata(c *gin.Context, title string, businessType int) {
	c.Set(OperationLogTitleKey, title)
	c.Set(OperationLogBusinessTypeKey, businessType)
}

// SetAuditParam 覆盖审计参数 (可选，默认记录请求 Body)
func SetAuditParam(c *gin.Context, param string) {
	c.Set(OperationLogParamKey, param)
}

// SetAuditResult 覆盖审计结果 (可选，默认记录返回 Body)
func SetAuditResult(c *gin.Context, result string) {
	c.Set(OperationLogResultKey, result)
}

// SetAuditStatus 覆盖审计状态 (1=成功, 2=失败)
func SetAuditStatus(c *gin.Context, status int) {
	c.Set(OperationLogStatusKey, status)
}

// SetAuditErrorMsg 覆盖审计错误信息
func SetAuditErrorMsg(c *gin.Context, message string) {
	c.Set(OperationLogErrorMsgKey, message)
}
