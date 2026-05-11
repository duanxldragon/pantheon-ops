package iam

type UserProfileResp struct {
	ID          uint64                      `json:"id"`
	Username    string                      `json:"username"`
	Nickname    string                      `json:"nickname"`
	Avatar      string                      `json:"avatar"`
	Email       string                      `json:"email"`
	Phone       string                      `json:"phone"`
	Preferences *UserPlatformPreferenceResp `json:"preferences,omitempty"`
	DeptID      uint64                      `json:"deptId"`
	PostID      uint64                      `json:"postId"`
	Status      int                         `json:"status"`
	Roles       []string                    `json:"roles"`
	Perms       []string                    `json:"perms"`
	ProfileExt  map[string]interface{}      `json:"profileExt,omitempty"`
	CreatedAt   string                      `json:"createdAt"`
}

// UserListResp 列表页返回 DTO
type UserListResp struct {
	ID        uint64   `json:"id"`
	Username  string   `json:"username"`
	Nickname  string   `json:"nickname"`
	Email     string   `json:"email"`
	Phone     string   `json:"phone"`
	DeptID    uint64   `json:"deptId"`
	DeptName  string   `json:"deptName"`
	PostID    uint64   `json:"postId"`
	PostName  string   `json:"postName"`
	Status    int      `json:"status"`
	CreatedAt string   `json:"createdAt"`
	RoleIDs   []uint64 `json:"roleIds"`
	RoleKeys  []string `json:"roleKeys"`
}

type UserListPageResp struct {
	Items    []UserListResp `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

type UserDetailResp struct {
	ID         uint64                 `json:"id"`
	Username   string                 `json:"username"`
	Nickname   string                 `json:"nickname"`
	Avatar     string                 `json:"avatar"`
	Email      string                 `json:"email"`
	Phone      string                 `json:"phone"`
	DeptID     uint64                 `json:"deptId"`
	DeptName   string                 `json:"deptName"`
	PostID     uint64                 `json:"postId"`
	PostName   string                 `json:"postName"`
	Status     int                    `json:"status"`
	CreatedAt  string                 `json:"createdAt"`
	UpdatedAt  string                 `json:"updatedAt"`
	RoleIDs    []uint64               `json:"roleIds"`
	RoleKeys   []string               `json:"roleKeys"`
	ProfileExt map[string]interface{} `json:"profileExt,omitempty"`
}

type UserListQuery struct {
	Username  string `form:"username" json:"username"`
	Nickname  string `form:"nickname" json:"nickname"`
	DeptID    uint64 `form:"deptId" json:"deptId"`
	PostID    uint64 `form:"postId" json:"postId"`
	Status    *int   `form:"status" json:"status"`
	Page      int    `form:"page" json:"page"`
	PageSize  int    `form:"pageSize" json:"pageSize"`
	SortField string `form:"sortField" json:"sortField"`
	SortOrder string `form:"sortOrder" json:"sortOrder"`
}

type UserCreateReq struct {
	Username   string                 `json:"username" binding:"required"`
	Password   string                 `json:"password" binding:"required"`
	Nickname   string                 `json:"nickname"`
	Avatar     string                 `json:"avatar"`
	Email      string                 `json:"email"`
	Phone      string                 `json:"phone"`
	DeptID     uint64                 `json:"deptId"`
	PostID     uint64                 `json:"postId"`
	Status     int                    `json:"status"`
	RoleIDs    []uint64               `json:"roleIds"`
	ProfileExt map[string]interface{} `json:"profileExt"`
}

type UserUpdateReq struct {
	Nickname   string                 `json:"nickname"`
	Avatar     string                 `json:"avatar"`
	Email      string                 `json:"email"`
	Phone      string                 `json:"phone"`
	DeptID     uint64                 `json:"deptId"`
	PostID     uint64                 `json:"postId"`
	Status     int                    `json:"status"`
	RoleIDs    []uint64               `json:"roleIds"`
	ProfileExt map[string]interface{} `json:"profileExt"`
}

type UserResetPasswordReq struct {
	NewPassword string `json:"newPassword" binding:"required"`
}

type UserBatchStatusReq struct {
	UserIDs []uint64 `json:"userIds"`
	Status  int      `json:"status"`
}

type UserProfileUpdateReq struct {
	Nickname   string                 `json:"nickname"`
	Avatar     string                 `json:"avatar"`
	Email      string                 `json:"email"`
	Phone      string                 `json:"phone"`
	ProfileExt map[string]interface{} `json:"profileExt"`
}
