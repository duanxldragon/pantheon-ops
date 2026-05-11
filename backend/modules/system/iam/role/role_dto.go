package iam

// RoleListResp 角色列表 DTO
type RoleListResp struct {
	ID             uint64   `json:"id"`
	RoleName       string   `json:"roleName"`
	RoleKey        string   `json:"roleKey"`
	Sort           int      `json:"sort"`
	Status         int      `json:"status"`
	CreatedAt      string   `json:"createdAt"`
	MenuIDs        []uint64 `json:"menuIds"`
	PermissionKeys []string `json:"permissionKeys"`
}

type RoleListPageResp struct {
	Items    []RoleListResp `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

type RoleListQuery struct {
	RoleName  string `form:"roleName" json:"roleName"`
	RoleKey   string `form:"roleKey" json:"roleKey"`
	Status    *int   `form:"status" json:"status"`
	Page      int    `form:"page" json:"page"`
	PageSize  int    `form:"pageSize" json:"pageSize"`
	SortField string `form:"sortField" json:"sortField"`
	SortOrder string `form:"sortOrder" json:"sortOrder"`
}

type RoleCreateReq struct {
	RoleName       string   `json:"roleName" binding:"required"`
	RoleKey        string   `json:"roleKey" binding:"required"`
	Sort           int      `json:"sort"`
	Status         int      `json:"status"`
	MenuIDs        []uint64 `json:"menuIds"`
	PermissionKeys []string `json:"permissionKeys"`
}

type RoleUpdateReq struct {
	RoleName       string   `json:"roleName" binding:"required"`
	RoleKey        string   `json:"roleKey" binding:"required"`
	Sort           int      `json:"sort"`
	Status         int      `json:"status"`
	MenuIDs        []uint64 `json:"menuIds"`
	PermissionKeys []string `json:"permissionKeys"`
}

type RoleBatchStatusReq struct {
	RoleIDs []uint64 `json:"roleIds" binding:"required"`
	Status  int      `json:"status"`
}
