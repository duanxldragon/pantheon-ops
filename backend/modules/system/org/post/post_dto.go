package org

type PostListResp struct {
	ID                    uint64   `json:"id"`
	DeptID                uint64   `json:"deptId"`
	DeptName              string   `json:"deptName"`
	PostCode              string   `json:"postCode"`
	PostName              string   `json:"postName"`
	Sort                  int      `json:"sort"`
	Status                int      `json:"status"`
	Remark                string   `json:"remark"`
	AssignedUserCount     int      `json:"assignedUserCount"`
	GovernanceTags        []string `json:"governanceTags"`
	GovernanceTagLabels   []string `json:"governanceTagLabels"`
	GovernanceBlockedBy   []string `json:"governanceBlockedBy"`
	GovernanceBlockedDesc []string `json:"governanceBlockedDesc"`
	GovernanceActions     []string `json:"governanceActions"`
	GovernanceActionLabel []string `json:"governanceActionLabel"`
	CreatedAt             string   `json:"createdAt"`
}

type PostListPageResp struct {
	Items    []PostListResp `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

type PostListQuery struct {
	PostCode  string `form:"postCode" json:"postCode"`
	PostName  string `form:"postName" json:"postName"`
	DeptID    uint64 `form:"deptId" json:"deptId"`
	Status    *int   `form:"status" json:"status"`
	Page      int    `form:"page" json:"page"`
	PageSize  int    `form:"pageSize" json:"pageSize"`
	SortField string `form:"sortField" json:"sortField"`
	SortOrder string `form:"sortOrder" json:"sortOrder"`
}

type PostCreateReq struct {
	DeptID   uint64 `json:"deptId"`
	PostCode string `json:"postCode" binding:"required"`
	PostName string `json:"postName" binding:"required"`
	Sort     int    `json:"sort"`
	Status   int    `json:"status"`
	Remark   string `json:"remark"`
}

type PostUpdateReq struct {
	DeptID   uint64 `json:"deptId"`
	PostCode string `json:"postCode" binding:"required"`
	PostName string `json:"postName" binding:"required"`
	Sort     int    `json:"sort"`
	Status   int    `json:"status"`
	Remark   string `json:"remark"`
}

type PostBatchStatusReq struct {
	PostIDs []uint64 `json:"postIds"`
	Status  int      `json:"status"`
}
