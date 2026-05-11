package org

type DeptTreeResp struct {
	ID             uint64          `json:"id"`
	ParentID       uint64          `json:"parentId"`
	Ancestors      string          `json:"ancestors"`
	IsRoot         bool            `json:"isRoot"`
	DeptName       string          `json:"deptName"`
	Sort           int             `json:"sort"`
	LeaderUserID   uint64          `json:"leaderUserId"`
	Leader         string          `json:"leader"`
	Phone          string          `json:"phone"`
	Email          string          `json:"email"`
	Status         int             `json:"status"`
	ChildDeptCount int             `json:"childDeptCount"`
	PostCount      int             `json:"postCount"`
	IsLeaderless   bool            `json:"isLeaderless"`
	IsNoPost       bool            `json:"isNoPost"`
	IsEmpty        bool            `json:"isEmpty"`
	Children       []*DeptTreeResp `json:"children,omitempty"`
}

type DeptListQuery struct {
	DeptName   string `form:"deptName" json:"deptName"`
	Status     *int   `form:"status" json:"status"`
	Governance string `form:"governance" json:"governance"`
	SortField  string `form:"sortField" json:"sortField"`
	SortOrder  string `form:"sortOrder" json:"sortOrder"`
}

type DeptCreateReq struct {
	ParentID     uint64 `json:"parentId"`
	DeptName     string `json:"deptName" binding:"required"`
	Sort         int    `json:"sort"`
	LeaderUserID uint64 `json:"leaderUserId"`
	Leader       string `json:"leader"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	Status       int    `json:"status"`
}

type DeptUpdateReq struct {
	ParentID     uint64 `json:"parentId"`
	DeptName     string `json:"deptName" binding:"required"`
	Sort         int    `json:"sort"`
	LeaderUserID uint64 `json:"leaderUserId"`
	Leader       string `json:"leader"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	Status       int    `json:"status"`
}

type DeptLeaderCandidateResp struct {
	UserID      uint64 `json:"userId"`
	Username    string `json:"username"`
	Nickname    string `json:"nickname"`
	DisplayName string `json:"displayName"`
	DeptID      uint64 `json:"deptId"`
	DeptName    string `json:"deptName"`
	PostID      uint64 `json:"postId"`
	PostName    string `json:"postName"`
}

type DeptBatchStatusReq struct {
	DeptIDs []uint64 `json:"deptIds"`
	Status  int      `json:"status"`
}

type DeptBatchLeaderItem struct {
	DeptID       uint64 `json:"deptId"`
	LeaderUserID uint64 `json:"leaderUserId"`
}

type DeptBatchLeaderReq struct {
	Items []DeptBatchLeaderItem `json:"items"`
}

type DeptOverviewResp struct {
	TotalDeptCount       int `json:"totalDeptCount"`
	EnabledDeptCount     int `json:"enabledDeptCount"`
	DisabledDeptCount    int `json:"disabledDeptCount"`
	RootDeptCount        int `json:"rootDeptCount"`
	DirectChildDeptCount int `json:"directChildDeptCount"`
	TotalPostCount       int `json:"totalPostCount"`
	EnabledPostCount     int `json:"enabledPostCount"`
	LeaderlessDeptCount  int `json:"leaderlessDeptCount"`
	NoPostDeptCount      int `json:"noPostDeptCount"`
	EmptyDeptCount       int `json:"emptyDeptCount"`
	HealthIssueCount     int `json:"healthIssueCount"`
}

type DeptGovernanceTaskQuery struct {
	Keyword    string `form:"keyword" json:"keyword"`
	Scope      string `form:"scope" json:"scope"`
	Governance string `form:"governance" json:"governance"`
	BlockedBy  string `form:"blockedBy" json:"blockedBy"`
	Action     string `form:"action" json:"action"`
}

type DeptGovernanceTaskResp struct {
	TaskKey                  string `json:"taskKey"`
	GovernanceScope          string `json:"governanceScope"`
	GovernanceScopeLabel     string `json:"governanceScopeLabel"`
	GovernanceTag            string `json:"governanceTag"`
	GovernanceTagLabel       string `json:"governanceTagLabel"`
	GovernanceBlockedBy      string `json:"governanceBlockedBy"`
	GovernanceBlockedByLabel string `json:"governanceBlockedByLabel"`
	GovernanceAction         string `json:"governanceAction"`
	GovernanceActionLabel    string `json:"governanceActionLabel"`
	DeptID                   uint64 `json:"deptId"`
	DeptName                 string `json:"deptName"`
	DeptPath                 string `json:"deptPath"`
	PostID                   uint64 `json:"postId"`
	PostName                 string `json:"postName"`
	RelatedUserCount         int    `json:"relatedUserCount"`
	ResourceStatus           int    `json:"resourceStatus"`
}
