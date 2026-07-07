package org

import (
	"fmt"
	"strings"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/impexp"
)

// dept_governance.go - Governance functions for dept module

// ListGovernanceTasks lists governance tasks for departments and posts
func (s *DeptService) ListGovernanceTasks(query *DeptGovernanceTaskQuery) ([]DeptGovernanceTaskResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	var depts []SystemDept
	if err := s.db.Order("sort asc").Order("id asc").Find(&depts).Error; err != nil {
		return nil, err
	}
	childCountByDept, err := s.loadDeptChildCounts()
	if err != nil {
		return nil, err
	}
	postCountByDept, err := s.loadDeptPostCounts()
	if err != nil {
		return nil, err
	}
	userCountByDept, err := s.loadDeptUserCounts()
	if err != nil {
		return nil, err
	}
	postUserCountByPost, err := s.loadPostUserCounts()
	if err != nil {
		return nil, err
	}
	pathByID, _, err := impexp.BuildDeptPathMaps(s.db)
	if err != nil {
		return nil, err
	}

	items := make([]DeptGovernanceTaskResp, 0)
	for _, dept := range depts {
		if dept.IsRoot == common.StatusFlagYes {
			continue
		}
		deptPath := pathByID[dept.ID]
		governanceTags := buildDeptGovernanceTags(dept, childCountByDept[dept.ID], postCountByDept[dept.ID])
		governanceBlockedBy := buildDeptDeleteBlockers(childCountByDept[dept.ID], postCountByDept[dept.ID], userCountByDept[dept.ID])
		governanceActions := buildDeptGovernanceActions(governanceTags, governanceBlockedBy)
		for index, action := range governanceActions {
			if action == "keep-observing" {
				continue
			}
			task := DeptGovernanceTaskResp{
				TaskKey:               fmt.Sprintf("dept:%d:%s", dept.ID, action),
				GovernanceScope:       "dept",
				GovernanceScopeLabel:  impexp.GovernanceScopeLabel("dept"),
				GovernanceTag:         pickDeptTaskTag(governanceTags, action),
				GovernanceBlockedBy:   pickDeptTaskBlockedBy(governanceBlockedBy, action),
				GovernanceAction:      action,
				GovernanceActionLabel: impexp.GovernanceActionLabels([]string{action}),
				DeptID:                dept.ID,
				DeptName:              dept.DeptName,
				DeptPath:              deptPath,
				RelatedUserCount:      userCountByDept[dept.ID],
				ResourceStatus:        dept.Status,
			}
			if task.GovernanceTag == "" && len(governanceTags) > 0 {
				task.GovernanceTag = governanceTags[min(index, len(governanceTags)-1)]
			}
			if task.GovernanceBlockedBy == "" {
				task.GovernanceBlockedBy = "none"
			}
			task.GovernanceTagLabel = impexp.GovernanceTagLabels([]string{task.GovernanceTag})
			task.GovernanceBlockedByLabel = impexp.GovernanceBlockedByLabels([]string{task.GovernanceBlockedBy})
			if matchGovernanceTaskQuery(task, query) {
				items = append(items, task)
			}
		}
	}

	if s.db.Migrator().HasTable("system_post") {
		type postTaskRow struct {
			ID       uint64
			DeptID   uint64
			PostCode string
			PostName string
			Status   int
		}
		posts := make([]postTaskRow, 0)
		if err := s.db.Table("system_post").Select("id, dept_id, post_code, post_name, status").Order("sort asc").Order("id asc").Scan(&posts).Error; err != nil {
			return nil, err
		}
		for _, post := range posts {
			governanceTags := buildLocalPostGovernanceTags(post.Status, postUserCountByPost[post.ID])
			governanceBlockedBy := buildLocalPostGovernanceBlockers(postUserCountByPost[post.ID])
			governanceActions := buildLocalPostGovernanceActions(post.Status, postUserCountByPost[post.ID])
			for _, action := range governanceActions {
				if action == "keep-observing" {
					continue
				}
				task := DeptGovernanceTaskResp{
					TaskKey:               fmt.Sprintf("post:%d:%s", post.ID, action),
					GovernanceScope:       "post",
					GovernanceScopeLabel:  impexp.GovernanceScopeLabel("post"),
					GovernanceTag:         pickPostTaskTag(governanceTags, action),
					GovernanceBlockedBy:   pickPostTaskBlockedBy(governanceBlockedBy, action),
					GovernanceAction:      action,
					GovernanceActionLabel: impexp.GovernanceActionLabels([]string{action}),
					DeptID:                post.DeptID,
					DeptName:              deptNameByID(depts, post.DeptID),
					DeptPath:              pathByID[post.DeptID],
					PostID:                post.ID,
					PostName:              post.PostName,
					RelatedUserCount:      postUserCountByPost[post.ID],
					ResourceStatus:        post.Status,
				}
				if task.GovernanceTag == "" && len(governanceTags) > 0 {
					task.GovernanceTag = governanceTags[0]
				}
				if task.GovernanceBlockedBy == "" {
					task.GovernanceBlockedBy = "none"
				}
				task.GovernanceTagLabel = impexp.GovernanceTagLabels([]string{task.GovernanceTag})
				task.GovernanceBlockedByLabel = impexp.GovernanceBlockedByLabels([]string{task.GovernanceBlockedBy})
				if matchGovernanceTaskQuery(task, query) {
					items = append(items, task)
				}
			}
		}
	}

	return items, nil
}

// loadDeptChildCounts loads child department counts
func (s *DeptService) loadDeptChildCounts() (map[uint64]int, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	var depts []SystemDept
	if err := s.db.Select("id, parent_id").Find(&depts).Error; err != nil {
		return nil, err
	}

	result := make(map[uint64]int, len(depts))
	for _, dept := range depts {
		result[dept.ParentID]++
	}
	return result, nil
}

// loadDeptUserCounts loads user counts per department
func (s *DeptService) loadDeptUserCounts() (map[uint64]int, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	if !s.db.Migrator().HasTable("system_user") {
		return map[uint64]int{}, nil
	}

	type deptUserCountRow struct {
		DeptID uint64
		Count  int
	}

	rows := make([]deptUserCountRow, 0)
	if err := s.db.Table("system_user").
		Select("dept_id, COUNT(1) AS count").
		Where("deleted_at IS NULL").
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

// loadPostUserCounts loads user counts per post
func (s *DeptService) loadPostUserCounts() (map[uint64]int, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	if !s.db.Migrator().HasTable("system_user") {
		return map[uint64]int{}, nil
	}

	type postUserCountRow struct {
		PostID uint64
		Count  int
	}

	rows := make([]postUserCountRow, 0)
	if err := s.db.Table("system_user").
		Select("post_id, COUNT(1) AS count").
		Where("deleted_at IS NULL AND post_id > 0").
		Group("post_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[uint64]int, len(rows))
	for _, row := range rows {
		result[row.PostID] = row.Count
	}
	return result, nil
}

// buildDeptGovernanceTags builds governance tags for a department
func buildDeptGovernanceTags(dept SystemDept, childDeptCount int, postCount int) []string {
	if dept.IsRoot == common.StatusFlagYes {
		return []string{"root"}
	}

	tags := make([]string, 0, 3)
	if strings.TrimSpace(dept.Leader) == "" {
		tags = append(tags, "leaderless")
	}
	if postCount == 0 {
		tags = append(tags, "no-post")
	}
	if childDeptCount == 0 && postCount == 0 {
		tags = append(tags, "empty")
	}
	if len(tags) == 0 {
		tags = append(tags, "clean")
	}
	return tags
}

// buildDeptDeleteBlockers builds delete blockers for a department
func buildDeptDeleteBlockers(childDeptCount, postCount, userCount int) []string {
	blockers := make([]string, 0, 3)
	if childDeptCount > 0 {
		blockers = append(blockers, "children")
	}
	if postCount > 0 {
		blockers = append(blockers, "posts")
	}
	if userCount > 0 {
		blockers = append(blockers, "users")
	}
	if len(blockers) == 0 {
		return []string{"none"}
	}
	return blockers
}

// buildDeptGovernanceActions builds available governance actions
func buildDeptGovernanceActions(tags, deleteBlockedBy []string) []string {
	actions := make([]string, 0, 6)
	seen := make(map[string]struct{}, 6)
	appendAction := func(value string) {
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		actions = append(actions, value)
	}

	for _, tag := range tags {
		switch tag {
		case "leaderless":
			appendAction("assign-leader")
		case "no-post":
			appendAction("create-post")
		case "empty":
			appendAction("review-merge-or-delete")
		}
	}
	for _, blocker := range deleteBlockedBy {
		switch blocker {
		case "children":
			appendAction("clear-child-depts")
		case "posts":
			appendAction("clear-posts")
		case "users":
			appendAction("clear-users")
		}
	}
	if len(actions) == 0 {
		return []string{"keep-observing"}
	}
	return actions
}

// buildLocalPostGovernanceTags builds governance tags for a post
func buildLocalPostGovernanceTags(status, assignedUserCount int) []string {
	tags := make([]string, 0, 2)
	if assignedUserCount > 0 {
		tags = append(tags, "in-use")
	}
	if normalizeSystemStatus(status) == common.StatusDisabled {
		tags = append(tags, "disabled")
	}
	if len(tags) == 0 {
		return []string{"clean"}
	}
	return tags
}

// buildLocalPostGovernanceBlockers builds governance blockers for a post
func buildLocalPostGovernanceBlockers(assignedUserCount int) []string {
	if assignedUserCount > 0 {
		return []string{"users"}
	}
	return []string{"none"}
}

// buildLocalPostGovernanceActions builds available governance actions for a post
func buildLocalPostGovernanceActions(status, assignedUserCount int) []string {
	if assignedUserCount > 0 {
		if normalizeSystemStatus(status) == common.StatusDisabled {
			return []string{"reassign-users", "review-status"}
		}
		return []string{"reassign-users"}
	}
	if normalizeSystemStatus(status) == common.StatusDisabled {
		return []string{"delete-or-keep-disabled"}
	}
	return []string{"keep-observing"}
}

// pickDeptTaskTag picks governance tag for a department action
func pickDeptTaskTag(tags []string, action string) string {
	switch action {
	case "assign-leader":
		return "leaderless"
	case "create-post":
		return "no-post"
	case "review-merge-or-delete":
		return "empty"
	case "clear-child-depts", "clear-posts", "clear-users":
		return "clean"
	default:
		if len(tags) > 0 {
			return tags[0]
		}
		return ""
	}
}

// pickDeptTaskBlockedBy picks governance blockedBy for a department action
func pickDeptTaskBlockedBy(blockedBy []string, action string) string {
	switch action {
	case "clear-child-depts":
		return "children"
	case "clear-posts":
		return "posts"
	case "clear-users":
		return "users"
	default:
		if len(blockedBy) > 0 {
			return blockedBy[0]
		}
		return "none"
	}
}

// pickPostTaskTag picks governance tag for a post action
func pickPostTaskTag(tags []string, action string) string {
	switch action {
	case "reassign-users":
		return "in-use"
	case "review-status", "delete-or-keep-disabled":
		return "disabled"
	default:
		if len(tags) > 0 {
			return tags[0]
		}
		return ""
	}
}

// pickPostTaskBlockedBy picks governance blockedBy for a post action
func pickPostTaskBlockedBy(blockedBy []string, action string) string {
	switch action {
	case "reassign-users":
		return "users"
	default:
		if len(blockedBy) > 0 {
			return blockedBy[0]
		}
		return "none"
	}
}

// matchGovernanceTaskQuery checks if task matches query filters
func matchGovernanceTaskQuery(task DeptGovernanceTaskResp, query *DeptGovernanceTaskQuery) bool {
	if query == nil {
		return true
	}
	scope := strings.ToLower(strings.TrimSpace(query.Scope))
	if scope != "" && scope != "all" && task.GovernanceScope != scope {
		return false
	}
	governance := strings.ToLower(strings.TrimSpace(query.Governance))
	if governance != "" && task.GovernanceTag != governance {
		return false
	}
	blockedBy := strings.ToLower(strings.TrimSpace(query.BlockedBy))
	if blockedBy != "" && task.GovernanceBlockedBy != blockedBy {
		return false
	}
	action := strings.ToLower(strings.TrimSpace(query.Action))
	if action != "" && task.GovernanceAction != action {
		return false
	}
	keyword := strings.ToLower(strings.TrimSpace(query.Keyword))
	if keyword == "" {
		return true
	}
	return strings.Contains(strings.ToLower(task.DeptName), keyword) ||
		strings.Contains(strings.ToLower(task.DeptPath), keyword) ||
		strings.Contains(strings.ToLower(task.PostName), keyword)
}

// buildGovernanceTaskQuerySummary builds query summary string
func buildGovernanceTaskQuerySummary(query *DeptGovernanceTaskQuery) string {
	if query == nil {
		return "scope=all"
	}
	parts := []string{}
	if strings.TrimSpace(query.Scope) != "" {
		parts = append(parts, "scope="+strings.TrimSpace(query.Scope))
	}
	if strings.TrimSpace(query.Keyword) != "" {
		parts = append(parts, "keyword="+strings.TrimSpace(query.Keyword))
	}
	if strings.TrimSpace(query.Governance) != "" {
		parts = append(parts, "governance="+strings.TrimSpace(query.Governance))
	}
	if strings.TrimSpace(query.BlockedBy) != "" {
		parts = append(parts, "blockedBy="+strings.TrimSpace(query.BlockedBy))
	}
	if strings.TrimSpace(query.Action) != "" {
		parts = append(parts, "action="+strings.TrimSpace(query.Action))
	}
	if len(parts) == 0 {
		return "scope=all"
	}
	return strings.Join(parts, "; ")
}

// deptNameByID finds department name by ID
func deptNameByID(depts []SystemDept, deptID uint64) string {
	for _, dept := range depts {
		if dept.ID == deptID {
			return dept.DeptName
		}
	}
	return ""
}
