package org

import (
	"errors"
	"fmt"
	"strings"

	"pantheon-ops/backend/pkg/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// dept_tree.go - Tree building and query functions

// GetDeptTree returns department tree with optional filtering
func (s *DeptService) GetDeptTree(query *DeptListQuery) ([]*DeptTreeResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	var depts []SystemDept
	sortColumn, sortDesc := normalizeDeptSort(query)
	if err := s.db.Model(&SystemDept{}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: sortColumn}, Desc: sortDesc}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}, Desc: false}).
		Find(&depts).Error; err != nil {
		return nil, err
	}

	postCountByDept, err := s.loadDeptPostCounts()
	if err != nil {
		return nil, err
	}
	depts = filterDeptTreeNodes(depts, query, postCountByDept)
	return buildDeptTree(depts, 0, postCountByDept), nil
}

// GetOverview returns department overview statistics
func (s *DeptService) GetOverview() (*DeptOverviewResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	var depts []SystemDept
	if err := s.db.Order("id asc").Find(&depts).Error; err != nil {
		return nil, err
	}

	type orgPostRow struct {
		DeptID uint64
		Status int
	}
	posts := make([]orgPostRow, 0)
	if s.db.Migrator().HasTable("system_post") {
		if err := s.db.Table("system_post").Select("dept_id, status").Order("id asc").Find(&posts).Error; err != nil {
			return nil, err
		}
	}

	childCountByDept := make(map[uint64]int, len(depts))
	for _, dept := range depts {
		childCountByDept[dept.ParentID]++
	}
	postCountByDept := make(map[uint64]int, len(posts))
	enabledPostCountByDept := make(map[uint64]int, len(posts))
	for _, post := range posts {
		postCountByDept[post.DeptID]++
		if post.Status == common.StatusEnabled {
			enabledPostCountByDept[post.DeptID]++
		}
	}

	resp := &DeptOverviewResp{}
	var rootID uint64
	for _, dept := range depts {
		resp.TotalDeptCount++
		if dept.Status == common.StatusEnabled {
			resp.EnabledDeptCount++
		} else {
			resp.DisabledDeptCount++
		}
		if dept.IsRoot == common.StatusFlagYes {
			resp.RootDeptCount++
			rootID = dept.ID
			continue
		}
		if strings.TrimSpace(dept.Leader) == "" {
			resp.LeaderlessDeptCount++
		}
		if postCountByDept[dept.ID] == 0 {
			resp.NoPostDeptCount++
		}
		if childCountByDept[dept.ID] == 0 && postCountByDept[dept.ID] == 0 {
			resp.EmptyDeptCount++
		}
	}

	for _, post := range posts {
		resp.TotalPostCount++
		if post.Status == common.StatusEnabled {
			resp.EnabledPostCount++
		}
	}

	if rootID > 0 {
		resp.DirectChildDeptCount = childCountByDept[rootID]
	}
	resp.HealthIssueCount = resp.LeaderlessDeptCount + resp.NoPostDeptCount + resp.DisabledDeptCount
	return resp, nil
}

// ListLeaderCandidates returns candidate users for department leadership
func (s *DeptService) ListLeaderCandidates(deptID uint64) ([]DeptLeaderCandidateResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	if deptID == 0 {
		return nil, common.NewNotFound("dept.not_found")
	}

	var dept SystemDept
	if err := s.db.First(&dept, deptID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.NewNotFound("dept.not_found")
		}
		return nil, err
	}
	if dept.IsRoot == common.StatusFlagYes {
		return []DeptLeaderCandidateResp{}, nil
	}

	type leaderCandidateRow struct {
		UserID   uint64 `gorm:"column:user_id"`
		Username string `gorm:"column:username"`
		Nickname string `gorm:"column:nickname"`
		DeptID   uint64 `gorm:"column:dept_id"`
		DeptName string `gorm:"column:dept_name"`
		PostID   uint64 `gorm:"column:post_id"`
		PostName string `gorm:"column:post_name"`
	}

	rows := make([]leaderCandidateRow, 0)
	if err := s.db.Table("system_user AS u").
		Select("u.id AS user_id, u.username, u.nickname, u.dept_id, d.dept_name, u.post_id, p.post_name").
		Joins("JOIN system_dept AS d ON d.id = u.dept_id").
		Joins("JOIN system_post AS p ON p.id = u.post_id AND p.dept_id = u.dept_id").
		Where("u.deleted_at IS NULL AND u.status = ? AND u.dept_id = ? AND u.post_id > 0", common.StatusEnabled, deptID).
		Order("u.nickname asc").
		Order("u.username asc").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]DeptLeaderCandidateResp, 0, len(rows))
	for _, row := range rows {
		displayName := strings.TrimSpace(row.Nickname)
		if displayName == "" {
			displayName = row.Username
		}
		result = append(result, DeptLeaderCandidateResp{
			UserID:      row.UserID,
			Username:    row.Username,
			Nickname:    row.Nickname,
			DisplayName: displayName,
			DeptID:      row.DeptID,
			DeptName:    row.DeptName,
			PostID:      row.PostID,
			PostName:    row.PostName,
		})
	}
	return result, nil
}

// loadDeptPostCounts loads post counts per department
func (s *DeptService) loadDeptPostCounts() (map[uint64]int, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	if !s.db.Migrator().HasTable("system_post") {
		return map[uint64]int{}, nil
	}

	type deptPostCountRow struct {
		DeptID uint64
		Count  int
	}

	rows := make([]deptPostCountRow, 0)
	if err := s.db.Table("system_post").
		Select("dept_id, COUNT(1) AS count").
		Group("dept_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[uint64]int, len(rows))
	for _, row := range rows {
		result[row.DeptID] = row.Count
	}
	return result, nil
}

// normalizeDeptSort normalizes sort field and direction
func normalizeDeptSort(query *DeptListQuery) (string, bool) {
	if query == nil {
		return "sort", false
	}

	sortWhitelist := map[string]string{
		"id":        "id",
		"deptName":  "dept_name",
		"dept_name": "dept_name",
		"sort":      "sort",
		"leader":    "leader",
		"status":    "status",
	}
	column, ok := sortWhitelist[strings.TrimSpace(query.SortField)]
	if !ok {
		column = "sort"
	}
	return column, strings.ToLower(strings.TrimSpace(query.SortOrder)) == "desc"
}

// filterDeptTreeNodes filters departments based on query
func filterDeptTreeNodes(depts []SystemDept, query *DeptListQuery, postCountByDept map[uint64]int) []SystemDept {
	if query == nil {
		return depts
	}

	nameFilter := strings.TrimSpace(query.DeptName)
	governanceFilter := strings.TrimSpace(query.Governance)
	statusFilterEnabled := query.Status != nil && common.IsEnabledStatus(*query.Status)
	if nameFilter == "" && !statusFilterEnabled && governanceFilter == "" {
		return depts
	}

	byID := make(map[uint64]SystemDept, len(depts))
	childCountByDept := make(map[uint64]int, len(depts))
	included := make(map[uint64]struct{}, len(depts))
	for _, dept := range depts {
		byID[dept.ID] = dept
		childCountByDept[dept.ParentID]++
	}
	for _, dept := range depts {
		if matchesDeptQuery(dept, nameFilter, query.Status, governanceFilter, childCountByDept, postCountByDept) {
			included[dept.ID] = struct{}{}
			for _, ancestorID := range splitAncestors(dept.Ancestors) {
				if _, ok := byID[ancestorID]; ok {
					included[ancestorID] = struct{}{}
				}
			}
		}
	}

	filtered := make([]SystemDept, 0, len(included))
	for _, dept := range depts {
		if _, ok := included[dept.ID]; ok {
			filtered = append(filtered, dept)
		}
	}
	return filtered
}

// matchesDeptQuery checks if department matches query filters
func matchesDeptQuery(
	dept SystemDept,
	nameFilter string,
	status *int,
	governance string,
	childCountByDept map[uint64]int,
	postCountByDept map[uint64]int,
) bool {
	if nameFilter != "" && !strings.Contains(strings.ToLower(dept.DeptName), strings.ToLower(nameFilter)) {
		return false
	}
	if status != nil && common.IsEnabledStatus(*status) && dept.Status != *status {
		return false
	}
	switch strings.ToLower(governance) {
	case "", "all":
	case "leaderless":
		return dept.IsRoot != common.StatusFlagYes && strings.TrimSpace(dept.Leader) == ""
	case "no-post":
		return dept.IsRoot != common.StatusFlagYes && postCountByDept[dept.ID] == 0
	case "empty":
		return dept.IsRoot != common.StatusFlagYes && childCountByDept[dept.ID] == 0 && postCountByDept[dept.ID] == 0
	default:
		return false
	}
	return true
}

// buildDeptTree recursively builds department tree
func buildDeptTree(depts []SystemDept, parentID uint64, postCountByDept map[uint64]int) []*DeptTreeResp {
	childCountByDept := make(map[uint64]int, len(depts))
	for _, dept := range depts {
		childCountByDept[dept.ParentID]++
	}
	if postCountByDept == nil {
		postCountByDept = make(map[uint64]int)
	}
	tree := make([]*DeptTreeResp, 0)
	for _, dept := range depts {
		if dept.ParentID != parentID {
			continue
		}
		node := toDeptTreeResp(dept, childCountByDept[dept.ID], postCountByDept[dept.ID])
		node.Children = buildDeptTree(depts, dept.ID, postCountByDept)
		tree = append(tree, node)
	}
	return tree
}

// toDeptTreeResp converts SystemDept to DeptTreeResp
func toDeptTreeResp(dept SystemDept, childDeptCount int, postCount int) *DeptTreeResp {
	isLeaderless := dept.IsRoot != common.StatusFlagYes && strings.TrimSpace(dept.Leader) == ""
	isNoPost := dept.IsRoot != common.StatusFlagYes && postCount == 0
	isEmpty := dept.IsRoot != common.StatusFlagYes && childDeptCount == 0 && postCount == 0
	return &DeptTreeResp{
		ID:              dept.ID,
		ParentID:        dept.ParentID,
		Ancestors:       dept.Ancestors,
		IsRoot:          dept.IsRoot == common.StatusFlagYes,
		DeptName:        dept.DeptName,
		Sort:            dept.Sort,
		LeaderUserID:    dept.LeaderUserID,
		Leader:          dept.Leader,
		Phone:           dept.Phone,
		Email:           dept.Email,
		Status:          dept.Status,
		ChildDeptCount:  childDeptCount,
		PostCount:       postCount,
		IsLeaderless:    isLeaderless,
		IsNoPost:        isNoPost,
		IsEmpty:         isEmpty,
	}
}

// splitAncestors splits ancestors string to IDs
func splitAncestors(value string) []uint64 {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]uint64, 0, len(parts))
	for _, part := range parts {
		var id uint64
		_, _ = fmt.Sscanf(strings.TrimSpace(part), "%d", &id)
		if id > 0 {
			result = append(result, id)
		}
	}
	return result
}
