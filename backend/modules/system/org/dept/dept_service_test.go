package org

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"pantheon-ops/backend/pkg/testmysql"
)

func setupDeptTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testmysql.Open(t)
}

func TestDeptService_MigrateCreatesRootAndReparentsTopLevel(t *testing.T) {
	db := setupDeptTestDB(t)
	if err := db.AutoMigrate(&SystemDept{}); err != nil {
		t.Fatalf("migrate dept table: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_dept (id, parent_id, ancestors, is_root, dept_name, status) VALUES (10, 0, '', 0, '研发中心', 1)`).Error; err != nil {
		t.Fatalf("seed dept 10: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_dept (id, parent_id, ancestors, is_root, dept_name, status) VALUES (11, 10, '10', 0, '平台研发部', 1)`).Error; err != nil {
		t.Fatalf("seed dept 11: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_dept (id, parent_id, ancestors, is_root, dept_name, status) VALUES (12, 0, '', 0, '财务部', 1)`).Error; err != nil {
		t.Fatalf("seed dept 12: %v", err)
	}

	service := NewDeptService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate dept service: %v", err)
	}

	var roots []SystemDept
	if err := db.Where("is_root = ?", 1).Find(&roots).Error; err != nil {
		t.Fatalf("load root depts: %v", err)
	}
	if len(roots) != 1 {
		t.Fatalf("expected 1 root dept, got %d", len(roots))
	}

	root := roots[0]
	if root.ParentID != 0 || root.Ancestors != "" {
		t.Fatalf("expected root parent/ancestors to be zero/empty, got parent=%d ancestors=%q", root.ParentID, root.Ancestors)
	}

	var rdDept SystemDept
	if err := db.First(&rdDept, 10).Error; err != nil {
		t.Fatalf("load dept 10: %v", err)
	}
	if rdDept.ParentID != root.ID {
		t.Fatalf("expected dept 10 parent to be root %d, got %d", root.ID, rdDept.ParentID)
	}
	if rdDept.Ancestors != fmt.Sprintf("%d", root.ID) {
		t.Fatalf("expected dept 10 ancestors to be root id, got %q", rdDept.Ancestors)
	}

	var childDept SystemDept
	if err := db.First(&childDept, 11).Error; err != nil {
		t.Fatalf("load dept 11: %v", err)
	}
	expectedAncestors := fmt.Sprintf("%d,%d", root.ID, rdDept.ID)
	if childDept.Ancestors != expectedAncestors {
		t.Fatalf("expected child ancestors %q, got %q", expectedAncestors, childDept.Ancestors)
	}
}

func TestDeptService_DeleteDeptRejectsRoot(t *testing.T) {
	db := setupDeptTestDB(t)
	service := NewDeptService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate dept service: %v", err)
	}

	var root SystemDept
	if err := db.Where("is_root = ?", 1).First(&root).Error; err != nil {
		t.Fatalf("load root dept: %v", err)
	}

	err := service.DeleteDept(root.ID)
	if err == nil || err.Error() != "dept.root.delete_forbidden" {
		t.Fatalf("expected root delete forbidden, got %v", err)
	}
}

func TestDeptService_DeleteIgnoresSoftDeletedUsers(t *testing.T) {
	db := setupDeptTestDB(t)
	service := NewDeptService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate dept service: %v", err)
	}
	if err := db.Exec("CREATE TABLE IF NOT EXISTS system_user (id INTEGER PRIMARY KEY, dept_id INTEGER, deleted_at DATETIME)").Error; err != nil {
		t.Fatalf("create user fixture: %v", err)
	}

	var root SystemDept
	if err := db.Where("is_root = ?", 1).First(&root).Error; err != nil {
		t.Fatalf("load root dept: %v", err)
	}
	child := SystemDept{
		ParentID:  root.ID,
		Ancestors: fmt.Sprintf("%d", root.ID),
		DeptName:  "运维部",
		Status:    1,
	}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("seed child dept: %v", err)
	}
	if err := db.Exec("INSERT INTO system_user (id, dept_id, deleted_at) VALUES (1, ?, ?)", child.ID, time.Now()).Error; err != nil {
		t.Fatalf("seed soft deleted user: %v", err)
	}

	if err := service.DeleteDept(child.ID); err != nil {
		t.Fatalf("delete dept with soft deleted user: %v", err)
	}
}

func TestDeptService_DeleteRejectsDeptWithPosts(t *testing.T) {
	db := setupDeptTestDB(t)
	service := NewDeptService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate dept service: %v", err)
	}
	if err := db.Exec("CREATE TABLE IF NOT EXISTS system_post (id INTEGER PRIMARY KEY, dept_id INTEGER, deleted_at DATETIME)").Error; err != nil {
		t.Fatalf("create post fixture: %v", err)
	}

	var root SystemDept
	if err := db.Where("is_root = ?", 1).First(&root).Error; err != nil {
		t.Fatalf("load root dept: %v", err)
	}
	child := SystemDept{
		ParentID:  root.ID,
		Ancestors: fmt.Sprintf("%d", root.ID),
		DeptName:  "数据部",
		Status:    1,
	}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("seed child dept: %v", err)
	}
	if err := db.Exec("INSERT INTO system_post (id, dept_id, deleted_at) VALUES (1, ?, NULL)", child.ID).Error; err != nil {
		t.Fatalf("seed active post: %v", err)
	}

	err := service.DeleteDept(child.ID)
	if err == nil || err.Error() != "dept.delete.error.has_posts" {
		t.Fatalf("expected dept has posts error, got %v", err)
	}
}

func TestDeptService_BatchUpdateDeptStatus(t *testing.T) {
	db := setupDeptTestDB(t)
	service := NewDeptService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate dept service: %v", err)
	}

	var root SystemDept
	if err := db.Where("is_root = ?", 1).First(&root).Error; err != nil {
		t.Fatalf("load root dept: %v", err)
	}
	child := SystemDept{
		ParentID:  root.ID,
		Ancestors: fmt.Sprintf("%d", root.ID),
		DeptName:  "研发中心",
		Status:    1,
	}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("seed child dept: %v", err)
	}

	updated, err := service.BatchUpdateDeptStatus([]uint64{child.ID, child.ID}, 2)
	if err != nil {
		t.Fatalf("batch disable dept: %v", err)
	}
	if updated != 1 {
		t.Fatalf("expected 1 updated dept, got %d", updated)
	}
	var disabled SystemDept
	if err := db.First(&disabled, child.ID).Error; err != nil {
		t.Fatalf("load disabled dept: %v", err)
	}
	if disabled.Status != 2 {
		t.Fatalf("expected dept status 2, got %d", disabled.Status)
	}

	if _, err := service.BatchUpdateDeptStatus([]uint64{root.ID}, 2); err == nil || err.Error() != "dept.root.status_fixed" {
		t.Fatalf("expected root status fixed error, got %v", err)
	}
}

func TestDeptService_BatchUpdateDeptLeader(t *testing.T) {
	db := setupDeptTestDB(t)
	service := NewDeptService(db)
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS system_post (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		dept_id INTEGER,
		post_code TEXT,
		post_name TEXT,
		status INTEGER,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create post table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS system_user (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT,
		password TEXT,
		nickname TEXT,
		dept_id INTEGER,
		post_id INTEGER,
		status INTEGER,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create user table: %v", err)
	}
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate dept service: %v", err)
	}

	var root SystemDept
	if err := db.Where("is_root = ?", 1).First(&root).Error; err != nil {
		t.Fatalf("load root dept: %v", err)
	}
	child := SystemDept{
		ParentID:  root.ID,
		Ancestors: fmt.Sprintf("%d", root.ID),
		DeptName:  "研发中心",
		Status:    1,
	}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("seed child dept: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_post (dept_id, post_code, post_name, status) VALUES (?, ?, ?, ?)`, child.ID, "rd-manager", "研发经理", 1).Error; err != nil {
		t.Fatalf("seed post: %v", err)
	}
	var postID uint64
	if err := db.Table("system_post").Select("id").Where("post_code = ?", "rd-manager").Limit(1).Pluck("id", &postID).Error; err != nil {
		t.Fatalf("load post id: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_user (username, password, nickname, dept_id, post_id, status) VALUES (?, ?, ?, ?, ?, ?)`, "lisi", "hashed", "李四", child.ID, postID, 1).Error; err != nil {
		t.Fatalf("seed leader user: %v", err)
	}
	var leaderUserID uint64
	if err := db.Table("system_user").Select("id").Where("username = ?", "lisi").Limit(1).Pluck("id", &leaderUserID).Error; err != nil {
		t.Fatalf("load leader user id: %v", err)
	}

	updated, err := service.BatchUpdateDeptLeader([]DeptBatchLeaderItem{{DeptID: child.ID, LeaderUserID: leaderUserID}})
	if err != nil {
		t.Fatalf("batch update dept leader: %v", err)
	}
	if updated != 1 {
		t.Fatalf("expected 1 updated dept, got %d", updated)
	}

	var refreshed SystemDept
	if err := db.First(&refreshed, child.ID).Error; err != nil {
		t.Fatalf("reload child dept: %v", err)
	}
	if refreshed.Leader != "李四" || refreshed.LeaderUserID != leaderUserID {
		t.Fatalf("expected leader updated, got %+v", refreshed)
	}

	if _, err := service.BatchUpdateDeptLeader([]DeptBatchLeaderItem{{DeptID: child.ID}}); err == nil || err.Error() != "dept.leader.required" {
		t.Fatalf("expected leader required error, got %v", err)
	}
}

func TestDeptService_ListLeaderCandidatesAndUpdateWithLeaderUser(t *testing.T) {
	db := setupDeptTestDB(t)
	service := NewDeptService(db)
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS system_post (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		dept_id INTEGER,
		post_code TEXT,
		post_name TEXT,
		status INTEGER,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create post table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS system_user (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT,
		password TEXT,
		nickname TEXT,
		dept_id INTEGER,
		post_id INTEGER,
		status INTEGER,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create user table: %v", err)
	}
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate dept service: %v", err)
	}

	var root SystemDept
	if err := db.Where("is_root = ?", 1).First(&root).Error; err != nil {
		t.Fatalf("load root dept: %v", err)
	}
	dept := SystemDept{
		ParentID:  root.ID,
		Ancestors: fmt.Sprintf("%d", root.ID),
		DeptName:  "研发中心",
		Status:    1,
	}
	if err := db.Create(&dept).Error; err != nil {
		t.Fatalf("seed dept: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_post (dept_id, post_code, post_name, status) VALUES (?, ?, ?, ?)`, dept.ID, "rd-manager", "研发经理", 1).Error; err != nil {
		t.Fatalf("seed post: %v", err)
	}
	var postID uint64
	if err := db.Table("system_post").Select("id").Where("post_code = ?", "rd-manager").Limit(1).Pluck("id", &postID).Error; err != nil {
		t.Fatalf("load post id: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_user (username, password, nickname, dept_id, post_id, status) VALUES (?, ?, ?, ?, ?, ?)`, "zhangsan", "hashed", "张三", dept.ID, postID, 1).Error; err != nil {
		t.Fatalf("seed leader user: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_user (username, password, nickname, dept_id, post_id, status) VALUES (?, ?, ?, ?, ?, ?)`, "lisi", "hashed", "李四", dept.ID, 0, 1).Error; err != nil {
		t.Fatalf("seed no-post user: %v", err)
	}
	var leaderUserID uint64
	if err := db.Table("system_user").Select("id").Where("username = ?", "zhangsan").Limit(1).Pluck("id", &leaderUserID).Error; err != nil {
		t.Fatalf("load leader user id: %v", err)
	}
	var noPostUserID uint64
	if err := db.Table("system_user").Select("id").Where("username = ?", "lisi").Limit(1).Pluck("id", &noPostUserID).Error; err != nil {
		t.Fatalf("load no-post user id: %v", err)
	}

	candidates, err := service.ListLeaderCandidates(dept.ID)
	if err != nil {
		t.Fatalf("list leader candidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0].UserID != leaderUserID || candidates[0].DisplayName != "张三" {
		t.Fatalf("unexpected candidates: %+v", candidates)
	}

	updated, err := service.UpdateDept(dept.ID, &DeptUpdateReq{
		ParentID:     dept.ParentID,
		DeptName:     dept.DeptName,
		Sort:         dept.Sort,
		LeaderUserID: leaderUserID,
		Status:       dept.Status,
	})
	if err != nil {
		t.Fatalf("update dept with leader user: %v", err)
	}
	if updated.LeaderUserID != leaderUserID || updated.Leader != "张三" {
		t.Fatalf("unexpected updated leader fields: %+v", updated)
	}

	if _, err := service.UpdateDept(dept.ID, &DeptUpdateReq{
		ParentID:     dept.ParentID,
		DeptName:     dept.DeptName,
		Sort:         dept.Sort,
		LeaderUserID: noPostUserID,
		Status:       dept.Status,
	}); err == nil || err.Error() != "dept.leader.user_invalid" {
		t.Fatalf("expected invalid leader user error, got %v", err)
	}
}

func TestDeptService_GetDeptTreeIncludesAncestorsForSearch(t *testing.T) {
	db := setupDeptTestDB(t)
	service := NewDeptService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate dept service: %v", err)
	}

	var root SystemDept
	if err := db.Where("is_root = ?", 1).First(&root).Error; err != nil {
		t.Fatalf("load root dept: %v", err)
	}
	if err := db.Create(&SystemDept{
		ParentID:  root.ID,
		Ancestors: fmt.Sprintf("%d", root.ID),
		DeptName:  "研发中心",
		Status:    1,
	}).Error; err != nil {
		t.Fatalf("seed child dept: %v", err)
	}

	tree, err := service.GetDeptTree(&DeptListQuery{DeptName: "研发"})
	if err != nil {
		t.Fatalf("get dept tree: %v", err)
	}
	if len(tree) != 1 {
		t.Fatalf("expected root node in search tree, got %d nodes", len(tree))
	}
	if !tree[0].IsRoot {
		t.Fatalf("expected first node to be root")
	}
	if len(tree[0].Children) != 1 || tree[0].Children[0].DeptName != "研发中心" {
		t.Fatalf("expected matched child under root, got %+v", tree[0].Children)
	}
}

func TestDeptService_GetDeptTreeSupportsGovernanceFilter(t *testing.T) {
	db := setupDeptTestDB(t)
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS system_post (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		dept_id INTEGER,
		post_code TEXT,
		post_name TEXT,
		status INTEGER
	)`).Error; err != nil {
		t.Fatalf("create post table: %v", err)
	}
	service := NewDeptService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate dept service: %v", err)
	}

	var root SystemDept
	if err := db.Where("is_root = ?", 1).First(&root).Error; err != nil {
		t.Fatalf("load root dept: %v", err)
	}
	leaderless := SystemDept{
		ParentID:  root.ID,
		Ancestors: fmt.Sprintf("%d", root.ID),
		DeptName:  "研发中心",
		Status:    1,
	}
	if err := db.Create(&leaderless).Error; err != nil {
		t.Fatalf("seed leaderless dept: %v", err)
	}
	emptyDept := SystemDept{
		ParentID:  root.ID,
		Ancestors: fmt.Sprintf("%d", root.ID),
		DeptName:  "运维中心",
		Leader:    "张三",
		Status:    1,
	}
	if err := db.Create(&emptyDept).Error; err != nil {
		t.Fatalf("seed empty dept: %v", err)
	}
	withPost := SystemDept{
		ParentID:  root.ID,
		Ancestors: fmt.Sprintf("%d", root.ID),
		DeptName:  "产品中心",
		Leader:    "李四",
		Status:    1,
	}
	if err := db.Create(&withPost).Error; err != nil {
		t.Fatalf("seed dept with post: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_post (dept_id, post_code, post_name, status) VALUES (?, ?, ?, ?)`, withPost.ID, "pm", "产品经理", 1).Error; err != nil {
		t.Fatalf("seed post: %v", err)
	}

	tree, err := service.GetDeptTree(&DeptListQuery{Governance: "leaderless"})
	if err != nil {
		t.Fatalf("get leaderless tree: %v", err)
	}
	if len(tree) != 1 || len(tree[0].Children) != 1 || tree[0].Children[0].DeptName != "研发中心" {
		t.Fatalf("unexpected leaderless tree: %+v", tree)
	}

	tree, err = service.GetDeptTree(&DeptListQuery{Governance: "empty"})
	if err != nil {
		t.Fatalf("get empty tree: %v", err)
	}
	if len(tree) != 1 || len(tree[0].Children) != 2 {
		t.Fatalf("unexpected empty tree size: %+v", tree)
	}
	childNames := []string{tree[0].Children[0].DeptName, tree[0].Children[1].DeptName}
	joined := strings.Join(childNames, ",")
	if !strings.Contains(joined, "研发中心") || !strings.Contains(joined, "运维中心") {
		t.Fatalf("expected leaderless and empty depts in empty filter tree, got %+v", childNames)
	}
}

func TestDeptService_ImportTemplateAndExport(t *testing.T) {
	db := setupDeptTestDB(t)
	service := NewDeptService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate dept service: %v", err)
	}

	template := service.BuildDeptImportTemplate()
	if len(template.Rows) == 0 || !strings.HasPrefix(template.Rows[0][0], "#") {
		t.Fatalf("expected template to include ignored instruction rows, got %+v", template.Rows)
	}
	templateResult, err := service.ImportDepts(append([][]string{template.Headers}, template.Rows...))
	if err != nil {
		t.Fatalf("import template comments: %v", err)
	}
	if !templateResult.Applied || templateResult.Created != 0 || templateResult.Failed != 0 {
		t.Fatalf("expected template comments to be ignored, got %+v", templateResult)
	}

	var root SystemDept
	if err := db.Where("is_root = ?", 1).First(&root).Error; err != nil {
		t.Fatalf("load root dept: %v", err)
	}
	result, err := service.ImportDepts([][]string{
		template.Headers,
		{root.DeptName, "研发中心", "10", "张三", "13800138000", "rd@example.com", "1"},
	})
	if err != nil {
		t.Fatalf("import dept: %v", err)
	}
	if !result.Applied || result.Created != 1 || result.Failed != 0 {
		t.Fatalf("unexpected import result: %+v", result)
	}

	exported, err := service.ExportDepts(&DeptListQuery{DeptName: "研发"})
	if err != nil {
		t.Fatalf("export dept: %v", err)
	}
	if len(exported.Rows) != 1 || exported.Rows[0][0] != root.DeptName || exported.Rows[0][1] != "研发中心" {
		t.Fatalf("unexpected export rows: %+v", exported.Rows)
	}
	if exported.Headers[len(exported.Headers)-1] != "governanceActionsLabel" {
		t.Fatalf("expected governance export headers, got %+v", exported.Headers)
	}
}

func TestDeptService_ExportDeptsSupportsGovernanceFilter(t *testing.T) {
	db := setupDeptTestDB(t)
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS system_post (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		dept_id INTEGER,
		post_code TEXT,
		post_name TEXT,
		status INTEGER
	)`).Error; err != nil {
		t.Fatalf("create post table: %v", err)
	}
	service := NewDeptService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate dept service: %v", err)
	}

	var root SystemDept
	if err := db.Where("is_root = ?", 1).First(&root).Error; err != nil {
		t.Fatalf("load root dept: %v", err)
	}
	if err := db.Create(&SystemDept{
		ParentID:  root.ID,
		Ancestors: fmt.Sprintf("%d", root.ID),
		DeptName:  "研发中心",
		Status:    1,
	}).Error; err != nil {
		t.Fatalf("seed leaderless dept: %v", err)
	}
	if err := db.Create(&SystemDept{
		ParentID:  root.ID,
		Ancestors: fmt.Sprintf("%d", root.ID),
		DeptName:  "财务中心",
		Leader:    "王五",
		Status:    1,
	}).Error; err != nil {
		t.Fatalf("seed normal dept: %v", err)
	}

	exported, err := service.ExportDepts(&DeptListQuery{Governance: "leaderless"})
	if err != nil {
		t.Fatalf("export leaderless dept: %v", err)
	}
	if len(exported.Rows) != 1 || exported.Rows[0][1] != "研发中心" {
		t.Fatalf("unexpected leaderless export rows: %+v", exported.Rows)
	}
	if exported.Rows[0][12] != "leaderless|no-post|empty" {
		t.Fatalf("expected governance tags in export, got %+v", exported.Rows[0])
	}
	if exported.Rows[0][11] != "dept" || exported.Rows[0][13] != "3" || exported.Rows[0][14] != "none" || exported.Rows[0][15] != "assign-leader|create-post|review-merge-or-delete" {
		t.Fatalf("unexpected governance action export row: %+v", exported.Rows[0])
	}
	if exported.Rows[0][16] != "Department" || exported.Rows[0][17] != "Leader Missing | No Posts | Empty Department" || exported.Rows[0][18] != "No Blocker" || exported.Rows[0][19] != "Assign Leader | Create Post | Review Merge or Delete" {
		t.Fatalf("unexpected governance label export row: %+v", exported.Rows[0])
	}
}

func TestDeptService_ExportDeptsIncludesGovernanceMetrics(t *testing.T) {
	db := setupDeptTestDB(t)
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS system_post (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		dept_id INTEGER,
		post_code TEXT,
		post_name TEXT,
		status INTEGER
	)`).Error; err != nil {
		t.Fatalf("create post table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS system_user (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		dept_id INTEGER,
		status INTEGER,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create user table: %v", err)
	}
	service := NewDeptService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate dept service: %v", err)
	}

	var root SystemDept
	if err := db.Where("is_root = ?", 1).First(&root).Error; err != nil {
		t.Fatalf("load root dept: %v", err)
	}
	parent := SystemDept{
		ParentID:  root.ID,
		Ancestors: fmt.Sprintf("%d", root.ID),
		DeptName:  "研发中心",
		Leader:    "张三",
		Status:    1,
	}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("seed parent dept: %v", err)
	}
	child := SystemDept{
		ParentID:  parent.ID,
		Ancestors: fmt.Sprintf("%d,%d", root.ID, parent.ID),
		DeptName:  "平台组",
		Leader:    "李四",
		Status:    1,
	}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("seed child dept: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_post (dept_id, post_code, post_name, status) VALUES (?, ?, ?, ?)`, parent.ID, "dev", "研发工程师", 1).Error; err != nil {
		t.Fatalf("seed post: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_user (dept_id, status, deleted_at) VALUES (?, ?, NULL)`, parent.ID, 1).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	exported, err := service.ExportDepts(&DeptListQuery{})
	if err != nil {
		t.Fatalf("export dept: %v", err)
	}
	if len(exported.Rows) != 2 {
		t.Fatalf("expected 2 exported rows, got %+v", exported.Rows)
	}

	var parentRow []string
	var childRow []string
	for _, row := range exported.Rows {
		switch row[1] {
		case "研发中心":
			parentRow = row
		case "平台组":
			childRow = row
		}
	}
	if len(parentRow) == 0 || len(childRow) == 0 {
		t.Fatalf("expected parent and child export rows, got %+v", exported.Rows)
	}
	if parentRow[7] == "" || parentRow[8] != "1" || parentRow[9] != "1" || parentRow[10] != "1" || parentRow[11] != "dept" || parentRow[12] != "clean" || parentRow[13] != "0" || parentRow[14] != "children|posts|users" || parentRow[15] != "clear-child-depts|clear-posts|clear-users" || parentRow[16] != "Department" || parentRow[17] != "Healthy" || parentRow[18] != "Child Departments | Posts | Users" || parentRow[19] != "Clear Child Departments | Clear Posts | Clear Users" {
		t.Fatalf("unexpected parent governance export row: %+v", parentRow)
	}
	if childRow[8] != "0" || childRow[9] != "0" || childRow[10] != "0" || childRow[11] != "dept" || childRow[12] != "no-post|empty" || childRow[13] != "2" || childRow[14] != "none" || childRow[15] != "create-post|review-merge-or-delete" || childRow[16] != "Department" || childRow[17] != "No Posts | Empty Department" || childRow[18] != "No Blocker" || childRow[19] != "Create Post | Review Merge or Delete" {
		t.Fatalf("unexpected child governance export row: %+v", childRow)
	}
}

func TestDeptService_ListGovernanceTasks(t *testing.T) {
	db := setupDeptTestDB(t)
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS system_post (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		dept_id INTEGER,
		post_code TEXT,
		post_name TEXT,
		status INTEGER
	)`).Error; err != nil {
		t.Fatalf("create post table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS system_user (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		dept_id INTEGER,
		post_id INTEGER,
		status INTEGER,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create user table: %v", err)
	}
	service := NewDeptService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate dept service: %v", err)
	}

	var root SystemDept
	if err := db.Where("is_root = ?", 1).First(&root).Error; err != nil {
		t.Fatalf("load root dept: %v", err)
	}
	dept := SystemDept{
		ParentID:  root.ID,
		Ancestors: fmt.Sprintf("%d", root.ID),
		DeptName:  "研发中心",
		Status:    1,
	}
	if err := db.Create(&dept).Error; err != nil {
		t.Fatalf("create dept: %v", err)
	}
	post := struct {
		ID uint64
	}{}
	if err := db.Exec(`INSERT INTO system_post (dept_id, post_code, post_name, status) VALUES (?, ?, ?, ?)`, dept.ID, "dev", "研发工程师", 1).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := db.Raw(`SELECT id FROM system_post WHERE post_code = ?`, "dev").Scan(&post).Error; err != nil {
		t.Fatalf("load post id: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_user (dept_id, post_id, status, deleted_at) VALUES (?, ?, 1, NULL)`, dept.ID, post.ID).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	items, err := service.ListGovernanceTasks(&DeptGovernanceTaskQuery{})
	if err != nil {
		t.Fatalf("list governance tasks: %v", err)
	}
	if len(items) < 2 {
		t.Fatalf("expected multiple governance tasks, got %+v", items)
	}
	foundAssignLeader := false
	foundReassignUsers := false
	for _, item := range items {
		if item.GovernanceAction == "assign-leader" && item.DeptID == dept.ID {
			foundAssignLeader = true
		}
		if item.GovernanceAction == "reassign-users" && item.PostID == post.ID && item.GovernanceScope == "post" {
			foundReassignUsers = true
		}
	}
	if !foundAssignLeader || !foundReassignUsers {
		t.Fatalf("expected leader and post governance tasks, got %+v", items)
	}
}

func TestDeptService_ExportGovernanceTasks(t *testing.T) {
	db := setupDeptTestDB(t)
	service := NewDeptService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate dept service: %v", err)
	}
	var root SystemDept
	if err := db.Where("is_root = ?", 1).First(&root).Error; err != nil {
		t.Fatalf("load root dept: %v", err)
	}
	if err := db.Create(&SystemDept{
		ParentID:  root.ID,
		Ancestors: fmt.Sprintf("%d", root.ID),
		DeptName:  "研发中心",
		Status:    1,
	}).Error; err != nil {
		t.Fatalf("create dept: %v", err)
	}

	exported, err := service.ExportGovernanceTasks(&DeptGovernanceTaskQuery{Governance: "leaderless"})
	if err != nil {
		t.Fatalf("export governance tasks: %v", err)
	}
	if len(exported.Headers) == 0 || exported.Headers[0] != "governanceSnapshotAt" {
		t.Fatalf("unexpected governance task export headers: %+v", exported.Headers)
	}
	if len(exported.Rows) == 0 || exported.Rows[0][5] != "leaderless" || exported.Rows[0][9] != "assign-leader" {
		t.Fatalf("unexpected governance task export rows: %+v", exported.Rows)
	}
}

func TestDeptService_GetOverview(t *testing.T) {
	db := setupDeptTestDB(t)
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS system_post (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		dept_id INTEGER,
		post_code TEXT,
		post_name TEXT,
		status INTEGER
	)`).Error; err != nil {
		t.Fatalf("create post table: %v", err)
	}

	service := NewDeptService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate dept service: %v", err)
	}

	var root SystemDept
	if err := db.Where("is_root = ?", 1).First(&root).Error; err != nil {
		t.Fatalf("load root dept: %v", err)
	}

	rd := SystemDept{
		ParentID:  root.ID,
		Ancestors: fmt.Sprintf("%d", root.ID),
		DeptName:  "研发中心",
		Leader:    "",
		Status:    1,
	}
	if err := db.Create(&rd).Error; err != nil {
		t.Fatalf("seed rd dept: %v", err)
	}
	ops := SystemDept{
		ParentID:  root.ID,
		Ancestors: fmt.Sprintf("%d", root.ID),
		DeptName:  "运维中心",
		Leader:    "张三",
		Status:    2,
	}
	if err := db.Create(&ops).Error; err != nil {
		t.Fatalf("seed ops dept: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_post (dept_id, post_code, post_name, status) VALUES (?, ?, ?, ?)`, rd.ID, "dev", "研发工程师", 1).Error; err != nil {
		t.Fatalf("seed enabled post: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_post (dept_id, post_code, post_name, status) VALUES (?, ?, ?, ?)`, rd.ID, "qa", "测试工程师", 2).Error; err != nil {
		t.Fatalf("seed disabled post: %v", err)
	}

	overview, err := service.GetOverview()
	if err != nil {
		t.Fatalf("get overview: %v", err)
	}
	if overview.TotalDeptCount != 3 {
		t.Fatalf("expected 3 depts, got %+v", overview)
	}
	if overview.EnabledDeptCount != 2 || overview.DisabledDeptCount != 1 {
		t.Fatalf("unexpected dept status counts: %+v", overview)
	}
	if overview.RootDeptCount != 1 || overview.DirectChildDeptCount != 2 {
		t.Fatalf("unexpected root metrics: %+v", overview)
	}
	if overview.TotalPostCount != 2 || overview.EnabledPostCount != 1 {
		t.Fatalf("unexpected post metrics: %+v", overview)
	}
	if overview.LeaderlessDeptCount != 1 {
		t.Fatalf("expected 1 leaderless dept, got %+v", overview)
	}
	if overview.EmptyDeptCount != 1 {
		t.Fatalf("expected 1 empty dept, got %+v", overview)
	}
	if overview.HealthIssueCount != 3 {
		t.Fatalf("expected 3 health issues, got %+v", overview)
	}
}
