package iam

// MenuTreeResp 菜单树返回 DTO
type MenuTreeResp struct {
	ID         uint64          `json:"id"`
	ParentID   uint64          `json:"parentId"`
	TitleKey   string          `json:"titleKey"`
	Path       string          `json:"path"`
	Component  string          `json:"component"`
	PagePerm   string          `json:"pagePerm"`
	Perms      string          `json:"perms"`
	Type       string          `json:"type"`
	Icon       string          `json:"icon"`
	RouteName  string          `json:"routeName"`
	Module     string          `json:"module"`
	Sort       int             `json:"sort"`
	IsVisible  int             `json:"isVisible"`
	IsCache    int             `json:"isCache"`
	IsExternal int             `json:"isExternal"`
	ActiveMenu string          `json:"activeMenu"`
	HideInNav  int             `json:"hideInNav"`
	Children   []*MenuTreeResp `json:"children,omitempty"` // 子菜单
}

type MenuCreateReq struct {
	ParentID   uint64 `json:"parentId"`
	TitleKey   string `json:"titleKey" binding:"required"`
	Path       string `json:"path"`
	Component  string `json:"component"`
	PagePerm   string `json:"pagePerm"`
	Perms      string `json:"perms"`
	Type       string `json:"type"`
	Icon       string `json:"icon"`
	RouteName  string `json:"routeName"`
	Module     string `json:"module"`
	Sort       int    `json:"sort"`
	IsVisible  int    `json:"isVisible"`
	IsCache    int    `json:"isCache"`
	IsExternal int    `json:"isExternal"`
	ActiveMenu string `json:"activeMenu"`
	HideInNav  int    `json:"hideInNav"`
}

type MenuUpdateReq struct {
	ParentID   uint64 `json:"parentId"`
	TitleKey   string `json:"titleKey" binding:"required"`
	Path       string `json:"path"`
	Component  string `json:"component"`
	PagePerm   string `json:"pagePerm"`
	Perms      string `json:"perms"`
	Type       string `json:"type"`
	Icon       string `json:"icon"`
	RouteName  string `json:"routeName"`
	Module     string `json:"module"`
	Sort       int    `json:"sort"`
	IsVisible  int    `json:"isVisible"`
	IsCache    int    `json:"isCache"`
	IsExternal int    `json:"isExternal"`
	ActiveMenu string `json:"activeMenu"`
	HideInNav  int    `json:"hideInNav"`
}

type MenuListQuery struct {
	TitleKey  string `form:"titleKey"`
	Path      string `form:"path"`
	IsVisible *int   `form:"isVisible"`
	SortField string `form:"sortField"`
	SortOrder string `form:"sortOrder"`
	Scope     string `form:"scope"`
}
