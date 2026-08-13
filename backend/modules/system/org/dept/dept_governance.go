package org

import (
	"fmt"
	"strings"

	"pantheon-base/pkg/common"
	"pantheon-base/pkg/impexp"
)

const (
	govActionKeepObserving   = "keep-observing"
	govActionClearChildDepts = "clear-child-depts"
	govActionClearPosts      = "clear-posts"
	govActionClearUsers      = "clear-users"
	govActionReassignUsers   = "reassign-users"
	govTagNoPost             = "no-post"
)

// dept_governance.go - Governance functions for dept module

// postGovernanceRow is a lightweight projection used when scanning posts for governance tasks.
type postGovernanceRow struct {
	ID       uint64
	DeptID   uint64
	PostCode string
	PostName string
	Status   int
}

// governanceSnapshot holds the preloaded counts and path maps shared by the task builders.
type governanceSnapshot struct {
	depts               []SystemDept
	childCountByDept    map[uint64]int
	postCountByDept     map[uint64]int
	userCountByDept     map[uint64]int
	postUserCountByPost map[uint64]int
	pathByID            map[uint64]string
}

// ListGovernanceTasks lists governance tasks for departments and posts.
// It orchestrates data loading and delegates the per-resource task building to helpers.
func (s *DeptService) ListGovernanceTasks(query *DeptGovernanceTaskQuery) ([]DeptGovernanceTaskResp, error) {
	snapshot, err := s.loadGovernanceSnapshot()
	if err != nil {
		return nil, err
	}

	items := make([]DeptGovernanceTaskResp, 0)
	s.collectDeptGovernanceTasks(snapshot, query, &items)

	if err := s.collectPostGovernanceTasks(snapshot, query, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// loadGovernanceSnapshot preloads all counts and path maps required for task building.
func (s *DeptService) loadGovernanceSnapshot() (*governanceSnapshot, error) {
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

	return &governanceSnapshot{
		depts:               depts,
		childCountByDept:    childCountByDept,
		postCountByDept:     postCountByDept,
		userCountByDept:     userCountByDept,
		postUserCountByPost: postUserCountByPost,
		pathByID:            pathByID,
	}, nil
}

// collectDeptGovernanceTasks appends governance tasks derived from departments.
func (s *DeptService) collectDeptGovernanceTasks(snapshot *governanceSnapshot, query *DeptGovernanceTaskQuery, items *[]DeptGovernanceTaskResp) {
	for _, dept := range snapshot.depts {
		if dept.IsRoot == common.StatusFlagYes {
			continue
		}
		s.appendDeptGovernanceTasks(dept, snapshot, query, items)
	}
}

// appendDeptGovernanceTasks builds and appends the governance tasks for a single department.
func (s *DeptService) appendDeptGovernanceTasks(dept SystemDept, snapshot *governanceSnapshot, query *DeptGovernanceTaskQuery, items *[]DeptGovernanceTaskResp) {
	deptPath := snapshot.pathByID[dept.ID]
	governanceTags := buildDeptGovernanceTags(dept, snapshot.childCountByDept[dept.ID], snapshot.postCountByDept[dept.ID])
	governanceBlockedBy := buildDeptDeleteBlockers(snapshot.childCountByDept[dept.ID], snapshot.postCountByDept[dept.ID], snapshot.userCountByDept[dept.ID])
	governanceActions := buildDeptGovernanceActions(governanceTags, governanceBlockedBy)

	for index, action := range governanceActions {
		if action == govActionKeepObserving {
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
			RelatedUserCount:      snapshot.userCountByDept[dept.ID],
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
			*items = append(*items, task)
		}
	}
}

// collectPostGovernanceTasks appends governance tasks derived from posts (when the post table exists).
func (s *DeptService) collectPostGovernanceTasks(snapshot *governanceSnapshot, query *DeptGovernanceTaskQuery, items *[]DeptGovernanceTaskResp) error {
	if !s.db.Migrator().HasTable("system_post") {
		return nil
	}

	posts := make([]postGovernanceRow, 0)
	if err := s.db.Table("system_post").
		Select("id, dept_id, post_code, post_name, status").
		Order("sort asc").Order("id asc").
		Scan(&posts).Error; err != nil {
		return err
	}

	for _, post := range posts {
		s.appendPostGovernanceTasks(post, snapshot, query, items)
	}
	return nil
}

// appendPostGovernanceTasks builds and appends the governance tasks for a single post.
func (s *DeptService) appendPostGovernanceTasks(post postGovernanceRow, snapshot *governanceSnapshot, query *DeptGovernanceTaskQuery, items *[]DeptGovernanceTaskResp) {
	governanceTags := buildLocalPostGovernanceTags(post.Status, snapshot.postUserCountByPost[post.ID])
	governanceBlockedBy := buildLocalPostGovernanceBlockers(snapshot.postUserCountByPost[post.ID])
	governanceActions := buildLocalPostGovernanceActions(post.Status, snapshot.postUserCountByPost[post.ID])

	for _, action := range governanceActions {
		if action == govActionKeepObserving {
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
			DeptName:              deptNameByID(snapshot.depts, post.DeptID),
			DeptPath:              snapshot.pathByID[post.DeptID],
			PostID:                post.ID,
			PostName:              post.PostName,
			RelatedUserCount:      snapshot.postUserCountByPost[post.ID],
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
			*items = append(*items, task)
		}
	}
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
		tags = append(tags, govTagNoPost)
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
		case govTagNoPost:
			appendAction("create-post")
		case "empty":
			appendAction("review-merge-or-delete")
		}
	}
	for _, blocker := range deleteBlockedBy {
		switch blocker {
		case "children":
			appendAction(govActionClearChildDepts)
		case "posts":
			appendAction(govActionClearPosts)
		case "users":
			appendAction(govActionClearUsers)
		}
	}
	if len(actions) == 0 {
		return []string{govActionKeepObserving}
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
			return []string{govActionReassignUsers, "review-status"}
		}
		return []string{govActionReassignUsers}
	}
	if normalizeSystemStatus(status) == common.StatusDisabled {
		return []string{"delete-or-keep-disabled"}
	}
	return []string{govActionKeepObserving}
}

// pickDeptTaskTag picks governance tag for a department action
func pickDeptTaskTag(tags []string, action string) string {
	switch action {
	case "assign-leader":
		return "leaderless"
	case "create-post":
		return govTagNoPost
	case "review-merge-or-delete":
		return "empty"
	case govActionClearChildDepts, govActionClearPosts, govActionClearUsers:
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
	case govActionClearChildDepts:
		return "children"
	case govActionClearPosts:
		return "posts"
	case govActionClearUsers:
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
	case govActionReassignUsers:
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
	case govActionReassignUsers:
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
