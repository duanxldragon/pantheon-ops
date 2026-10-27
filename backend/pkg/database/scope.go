package database

import (
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
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
		return applyDataScopeMode(db, req)
	}
}

func applyDataScopeMode(db *gorm.DB, req *common.DataScopeReq) *gorm.DB {
	switch strings.TrimSpace(req.Mode) {
	case "", common.DataScopeModeAll:
		return db
	case common.DataScopeModeDept:
		return applyDeptDataScope(db, req.DeptID)
	case common.DataScopeModeDeptAndChildren:
		return applyDeptAndChildrenDataScope(db, req)
	case common.DataScopeModeCustom:
		return applyCustomDataScope(db, req.DeptIDs)
	case common.DataScopeModeSelf:
		return applyRequestedSelfDataScope(db, req.UserID)
	default:
		return denyDataScope(db)
	}
}

func applyDeptDataScope(db *gorm.DB, deptID uint64) *gorm.DB {
	if deptID == 0 {
		return denyDataScope(db)
	}
	return db.Where("dept_id = ?", deptID)
}

func applyDeptAndChildrenDataScope(db *gorm.DB, req *common.DataScopeReq) *gorm.DB {
	if len(req.DeptIDs) > 0 {
		return db.Where("dept_id IN ?", req.DeptIDs)
	}
	return applyDeptDataScope(db, req.DeptID)
}

func applyCustomDataScope(db *gorm.DB, deptIDs []uint64) *gorm.DB {
	if len(deptIDs) == 0 {
		return denyDataScope(db)
	}
	return db.Where("dept_id IN ?", deptIDs)
}

func applyRequestedSelfDataScope(db *gorm.DB, userID uint64) *gorm.DB {
	if userID == 0 {
		return denyDataScope(db)
	}
	return applySelfDataScope(db, userID)
}

func denyDataScope(db *gorm.DB) *gorm.DB {
	return db.Where("1 = 0")
}

func applySelfDataScope(db *gorm.DB, userID uint64) *gorm.DB {
	columns, ok := statementColumns(db)
	if !ok {
		return denyDataScope(db)
	}

	ownerColumns := make([]string, 0, 2)
	for _, column := range []string{"created_by", "create_by"} {
		if _, exists := columns[column]; exists {
			ownerColumns = append(ownerColumns, column)
		}
	}
	if len(ownerColumns) > 0 {
		conditions := make([]string, 0, len(ownerColumns))
		args := make([]any, 0, len(ownerColumns))
		for _, column := range ownerColumns {
			conditions = append(conditions, column+" = ?")
			args = append(args, userID)
		}
		return db.Where(strings.Join(conditions, " OR "), args...)
	}

	if _, exists := columns["id"]; exists {
		return db.Where("id = ?", userID)
	}
	return denyDataScope(db)
}

func statementColumns(db *gorm.DB) (map[string]struct{}, bool) {
	if db == nil || db.Statement == nil {
		return nil, false
	}
	if db.Statement.Schema == nil && db.Statement.Model != nil {
		_ = db.Statement.Parse(db.Statement.Model)
	}
	if db.Statement.Schema == nil {
		return nil, false
	}

	columns := make(map[string]struct{}, len(db.Statement.Schema.Fields))
	for _, field := range db.Statement.Schema.Fields {
		if field.DBName != "" {
			columns[field.DBName] = struct{}{}
		}
	}
	return columns, true
}
