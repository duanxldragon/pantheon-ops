package login

// LoginReq 登录请求 DTO
type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RefreshTokenReq 刷新令牌请求 DTO
type RefreshTokenReq struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// LoginLogQuery 登录日志查询
type LoginLogQuery struct {
	Keyword   string `form:"keyword" json:"keyword"`
	Username  string `form:"username" json:"username"`
	Status    *int   `form:"status" json:"status"`
	StartedAt string `form:"startedAt" json:"startedAt"`
	EndedAt   string `form:"endedAt" json:"endedAt"`
	Page      int    `form:"page" json:"page"`
	PageSize  int    `form:"pageSize" json:"pageSize"`
}

type LoginLogCleanupReq struct {
	RetentionDays int    `json:"retentionDays"`
	StartedAt     string `json:"startedAt"`
	EndedAt       string `json:"endedAt"`
}

type LoginLogBatchDeleteReq struct {
	IDs []uint64 `json:"ids"`
}

// LoginLogResp 登录日志 DTO
type LoginLogResp struct {
	ID            uint64 `json:"id"`
	Username      string `json:"username"`
	Ipaddr        string `json:"ipaddr"`
	LoginLocation string `json:"loginLocation"`
	Browser       string `json:"browser"`
	Os            string `json:"os"`
	Status        int    `json:"status"`
	Msg           string `json:"msg"`
	LoginTime     string `json:"loginTime"`
}

type LoginLogPageResp struct {
	Items []LoginLogResp `json:"items"`
	Total int64          `json:"total"`
	// SuccessCount/FailedCount aggregate the whole filtered set (all pages),
	// so the governance bar shows global numbers instead of page-local ones.
	SuccessCount int64 `json:"successCount"`
	FailedCount  int64 `json:"failedCount"`
	Page         int   `json:"page"`
	PageSize     int   `json:"pageSize"`
}

type LoginLogCleanupResp struct {
	ClearedCount int64 `json:"clearedCount"`
}

type SecurityEventCleanupReq struct {
	RetentionDays int    `json:"retentionDays"`
	StartedAt     string `json:"startedAt"`
	EndedAt       string `json:"endedAt"`
}

type SecurityEventCleanupResp struct {
	ClearedCount int64 `json:"clearedCount"`
}

type SecurityEventBatchAcknowledgeReq struct {
	IDs                 []uint64 `json:"ids"`
	AcknowledgementNote string   `json:"acknowledgementNote"`
}

type SecurityEventBatchAcknowledgeResp struct {
	AcknowledgedCount int64 `json:"acknowledgedCount"`
}
