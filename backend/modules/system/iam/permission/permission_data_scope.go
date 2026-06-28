package iam

import (
	"sort"
	"strconv"
	"strings"

	"pantheon-ops/backend/pkg/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type permissionDataScopeRoleRow struct {
	ID       uint64 `gorm:"column:id"`
	RoleName string `gorm:"column:role_name"`
	RoleKey  string `gorm:"column:role_key"`
	Status   int    `gorm:"column:status"`
	Sort     int    `gorm:"column:sort"`
}

func (s *PermissionService) ListDataScopePolicies(query *PermissionDataScopeQuery) (*PermissionDataScopePolicyListResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	var roles []permissionDataScopeRoleRow
	db := s.db.Table("system_role").Where("deleted_at IS NULL")
	if query != nil {
		if strings.TrimSpace(query.RoleKey) != "" {
			db = db.Where("role_key LIKE ?", "%"+common.EscapeLikePattern(strings.TrimSpace(query.RoleKey))+"%")
		}
		if query.Status != nil && common.IsEnabledStatus(*query.Status) {
			db = db.Where("status = ?", *query.Status)
		}
	}
	if err := db.
		Order(clause.OrderByColumn{Column: clause.Column{Name: "sort"}, Desc: false}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}, Desc: false}).
		Find(&roles).Error; err != nil {
		return nil, err
	}

	roleKeys := make([]string, 0, len(roles))
	for _, role := range roles {
		roleKeys = append(roleKeys, role.RoleKey)
	}
	policies, err := s.loadRoleDataScopePolicies(roleKeys)
	if err != nil {
		return nil, err
	}

	items := make([]PermissionDataScopePolicyResp, 0, len(roles))
	for _, role := range roles {
		policy, ok := policies[role.RoleKey]
		mode := common.DataScopeModeAll
		deptIDs := []uint64{}
		id := uint64(0)
		if ok {
			id = policy.ID
			mode = normalizeDataScopeMode(policy.Mode)
			deptIDs = parsePermissionDataScopeDeptIDs(policy.DeptIDs)
		}
		items = append(items, PermissionDataScopePolicyResp{
			ID:           id,
			RoleName:     role.RoleName,
			RoleKey:      role.RoleKey,
			Status:       role.Status,
			Mode:         mode,
			DeptIDs:      deptIDs,
			PolicyExists: ok,
		})
	}

	return &PermissionDataScopePolicyListResp{Items: items, Total: len(items)}, nil
}

func (s *PermissionService) UpdateDataScopePolicy(roleKey string, req *PermissionDataScopePolicyUpdateReq) (*PermissionDataScopePolicyResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	if req == nil {
		return nil, common.NewBadRequest("param.invalid")
	}
	roleKey = strings.TrimSpace(roleKey)
	if roleKey == "" {
		return nil, common.NewBadRequest("param.invalid")
	}
	if err := s.ensureRoleKeyExists(roleKey); err != nil {
		return nil, err
	}
	mode := normalizeDataScopeMode(req.Mode)
	if !isValidDataScopeMode(mode) {
		return nil, common.NewBadRequest("permission.data_scope.mode_invalid")
	}
	deptIDs := normalizePermissionDataScopeDeptIDs(req.DeptIDs)
	if mode == common.DataScopeModeCustom && len(deptIDs) == 0 {
		return nil, common.NewBadRequest("permission.data_scope.dept_required")
	}
	if mode != common.DataScopeModeCustom {
		deptIDs = []uint64{}
	}

	policy := PermissionRoleDataScopePolicy{
		RoleKey: roleKey,
		Mode:    mode,
		DeptIDs: joinPermissionDataScopeDeptIDs(deptIDs),
	}
	if err := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "role_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"mode", "dept_ids"}),
	}).Create(&policy).Error; err != nil {
		return nil, err
	}

	list, err := s.ListDataScopePolicies(&PermissionDataScopeQuery{RoleKey: roleKey})
	if err != nil {
		return nil, err
	}
	for _, item := range list.Items {
		if item.RoleKey == roleKey {
			return &item, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *PermissionService) loadRoleDataScopePolicies(roleKeys []string) (map[string]PermissionRoleDataScopePolicy, error) {
	result := make(map[string]PermissionRoleDataScopePolicy, len(roleKeys))
	if len(roleKeys) == 0 {
		return result, nil
	}
	if !s.db.Migrator().HasTable(&PermissionRoleDataScopePolicy{}) {
		return result, nil
	}
	var policies []PermissionRoleDataScopePolicy
	if err := s.db.Where("role_key IN ?", roleKeys).Find(&policies).Error; err != nil {
		return nil, err
	}
	for _, policy := range policies {
		result[policy.RoleKey] = policy
	}
	return result, nil
}

func normalizeDataScopeMode(mode string) string {
	return strings.TrimSpace(strings.ToLower(mode))
}

func isValidDataScopeMode(mode string) bool {
	switch mode {
	case common.DataScopeModeAll, common.DataScopeModeSelf, common.DataScopeModeDept, common.DataScopeModeDeptAndChildren, common.DataScopeModeCustom:
		return true
	default:
		return false
	}
}

func parsePermissionDataScopeDeptIDs(raw string) []uint64 {
	parts := strings.Split(raw, ",")
	result := make([]uint64, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.ParseUint(strings.TrimSpace(part), 10, 64)
		if err != nil || value == 0 {
			continue
		}
		result = append(result, value)
	}
	return normalizePermissionDataScopeDeptIDs(result)
}

func normalizePermissionDataScopeDeptIDs(ids []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(ids))
	result := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func joinPermissionDataScopeDeptIDs(ids []uint64) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, strconv.FormatUint(id, 10))
	}
	return strings.Join(parts, ",")
}
