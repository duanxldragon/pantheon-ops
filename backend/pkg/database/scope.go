package database

import (
	"pantheon-platform/backend/pkg/common"
	"strings"

	"gorm.io/gorm"
)

// WithDataScope GORM 数据权限拦截钩子
// 使用方式：db.Scopes(database.WithDataScope(req)).Find(&users)
func WithDataScope(req *common.DataScopeReq) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if req == nil || req.IsAdmin {
			return db
		}

		switch strings.TrimSpace(req.Mode) {
		case "", common.DataScopeModeAll:
			return db
		case common.DataScopeModeDept:
			if req.DeptID == 0 {
				return db.Where("1 = 0")
			}
			return db.Where("dept_id = ?", req.DeptID)
		case common.DataScopeModeDeptAndChildren:
			if len(req.DeptIDs) > 0 {
				return db.Where("dept_id IN ?", req.DeptIDs)
			}
			if req.DeptID == 0 {
				return db.Where("1 = 0")
			}
			return db.Where("dept_id = ?", req.DeptID)
		case common.DataScopeModeCustom:
			if len(req.DeptIDs) == 0 {
				return db.Where("1 = 0")
			}
			return db.Where("dept_id IN ?", req.DeptIDs)
		case common.DataScopeModeSelf:
			if req.UserID == 0 {
				return db.Where("1 = 0")
			}
			return db.Where("created_by = ? OR create_by = ?", req.UserID, req.UserID)
		default:
			return db.Where("1 = 0")
		}
	}
}
