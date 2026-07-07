package session

// SessionCleanupReq 管理员清理历史会话请求
type SessionCleanupReq struct {
	RetentionDays int    `json:"retentionDays"`
	StartedAt     string `json:"startedAt"`
	EndedAt       string `json:"endedAt"`
}

// SessionBatchRevokeReq 批量撤销会话请求
type SessionBatchRevokeReq struct {
	SessionIDs []string `json:"sessionIds"`
}

// SessionCleanupResp 管理员清理历史会话响应
type SessionCleanupResp struct {
	ClearedCount int64 `json:"clearedCount"`
}
