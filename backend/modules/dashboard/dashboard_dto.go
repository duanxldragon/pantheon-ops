package dashboard

type SummaryResp struct {
	TotalUsers                int64                     `json:"totalUsers"`
	EnabledUsers              int64                     `json:"enabledUsers"`
	TotalRoles                int64                     `json:"totalRoles"`
	TotalDepts                int64                     `json:"totalDepts"`
	TotalPosts                int64                     `json:"totalPosts"`
	TotalDictTypes            int64                     `json:"totalDictTypes"`
	TotalSettings             int64                     `json:"totalSettings"`
	TotalI18nEntries          int64                     `json:"totalI18nEntries"`
	ActiveModuleCount         int64                     `json:"activeModuleCount"`
	VisibleMenuCount          int64                     `json:"visibleMenuCount"`
	ActiveSessionCount        int64                     `json:"activeSessionCount"`
	LoginSuccessCount         int64                     `json:"loginSuccessCount"`
	LoginFailureCount         int64                     `json:"loginFailureCount"`
	TotalSecurityEventCount   int64                     `json:"totalSecurityEventCount"`
	PendingSecurityEventCount int64                     `json:"pendingSecurityEventCount"`
	TodayOperationCount       int64                     `json:"todayOperationCount"`
	LastSuccessfulLoginAt     string                    `json:"lastSuccessfulLoginAt"`
	PeriodDays                int                       `json:"periodDays"`
	RecentLogins              []RecentLoginActivityResp `json:"recentLogins"`
	OrgGovernanceTaskCount    int                       `json:"orgGovernanceTaskCount"`
	OrgGovernanceTasks        []DashboardTodoResp       `json:"orgGovernanceTasks"`
}

type RecentLoginActivityResp struct {
	ID        uint64 `json:"id"`
	Username  string `json:"username"`
	Ipaddr    string `json:"ipaddr"`
	Browser   string `json:"browser"`
	OS        string `json:"os"`
	Status    int    `json:"status"`
	Msg       string `json:"msg"`
	LoginTime string `json:"loginTime"`
}

type DashboardTodoResp struct {
	TaskKey          string `json:"taskKey"`
	Domain           string `json:"domain"`
	ScopeLabel       string `json:"scopeLabel"`
	IssueLabel       string `json:"issueLabel"`
	ActionLabel      string `json:"actionLabel"`
	ResourceLabel    string `json:"resourceLabel"`
	RelatedUserCount int    `json:"relatedUserCount"`
	RoutePath        string `json:"routePath"`
	RouteStateDeptID uint64 `json:"routeStateDeptId"`
}
