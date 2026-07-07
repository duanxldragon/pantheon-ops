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
	Username string `form:"username" json:"username"`
	Status   *int   `form:"status" json:"status"`
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"pageSize" json:"pageSize"`
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
	Items    []LoginLogResp `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

type LoginLogCleanupResp struct {
	ClearedCount int64 `json:"clearedCount"`
}
