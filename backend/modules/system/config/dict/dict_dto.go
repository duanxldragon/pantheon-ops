package config

type DictTypeResp struct {
	ID                uint64 `json:"id"`
	DictCode          string `json:"dictCode"`
	DictName          string `json:"dictName"`
	Module            string `json:"module"`
	Status            int    `json:"status"`
	Remark            string `json:"remark"`
	ItemCount         int64  `json:"itemCount"`
	ActiveItemCount   int64  `json:"activeItemCount"`
	DisabledItemCount int64  `json:"disabledItemCount"`
	LastItemUpdatedAt string `json:"lastItemUpdatedAt"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

type DictTypeListQuery struct {
	DictCode string `form:"dictCode" json:"dictCode"`
	DictName string `form:"dictName" json:"dictName"`
	Status   *int   `form:"status" json:"status"`
}

type DictTypeCreateReq struct {
	DictCode string `json:"dictCode" binding:"required"`
	DictName string `json:"dictName" binding:"required"`
	Module   string `json:"module"`
	Status   int    `json:"status"`
	Remark   string `json:"remark"`
}

type DictTypeUpdateReq struct {
	DictCode string `json:"dictCode" binding:"required"`
	DictName string `json:"dictName" binding:"required"`
	Module   string `json:"module"`
	Status   int    `json:"status"`
	Remark   string `json:"remark"`
}

type DictItemResp struct {
	ID           uint64 `json:"id"`
	DictCode     string `json:"dictCode"`
	ItemLabelKey string `json:"itemLabelKey"`
	ItemValue    string `json:"itemValue"`
	ItemColor    string `json:"itemColor"`
	Sort         int    `json:"sort"`
	Status       int    `json:"status"`
	Remark       string `json:"remark"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type DictItemPageResp struct {
	Items    []DictItemResp `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

type DictItemListQuery struct {
	DictCode string `form:"dictCode" json:"dictCode"`
	Keyword  string `form:"keyword" json:"keyword"`
	Status   *int   `form:"status" json:"status"`
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"pageSize" json:"pageSize"`
}

type DictItemCreateReq struct {
	DictCode     string `json:"dictCode" binding:"required"`
	ItemLabelKey string `json:"itemLabelKey" binding:"required"`
	ItemValue    string `json:"itemValue" binding:"required"`
	ItemColor    string `json:"itemColor"`
	Sort         int    `json:"sort"`
	Status       int    `json:"status"`
	Remark       string `json:"remark"`
}

type DictItemUpdateReq struct {
	DictCode     string `json:"dictCode" binding:"required"`
	ItemLabelKey string `json:"itemLabelKey" binding:"required"`
	ItemValue    string `json:"itemValue" binding:"required"`
	ItemColor    string `json:"itemColor"`
	Sort         int    `json:"sort"`
	Status       int    `json:"status"`
	Remark       string `json:"remark"`
}

type DictTypeBatchStatusReq struct {
	TypeIDs []uint64 `json:"typeIds"`
	Status  int      `json:"status"`
}

type DictItemBatchStatusReq struct {
	ItemIDs []uint64 `json:"itemIds"`
	Status  int      `json:"status"`
}

type DictItemReorderReq struct {
	Direction string `json:"direction" binding:"required"`
}

type DictUsageReferenceResp struct {
	FilePath   string `json:"filePath"`
	Line       int    `json:"line"`
	Column     int    `json:"column"`
	Snippet    string `json:"snippet"`
	Domain     string `json:"domain"`
	ModuleHint string `json:"moduleHint"`
}

type DictUsageAnalysisResp struct {
	DictCode           string                   `json:"dictCode"`
	ReferenceCount     int                      `json:"referenceCount"`
	ScannedProjectRoot string                   `json:"scannedProjectRoot"`
	References         []DictUsageReferenceResp `json:"references"`
}

type DictOptionResp struct {
	LabelKey string `json:"labelKey"`
	Value    string `json:"value"`
	Color    string `json:"color"`
	Sort     int    `json:"sort"`
}

type DictOptionMapResp map[string][]DictOptionResp

type DictCacheRefreshReq struct {
	Codes []string `json:"codes"`
}

type DictCacheRefreshResp struct {
	RefreshedCodes []string `json:"refreshedCodes"`
	ClearedAll     int      `json:"clearedAll"`
}
