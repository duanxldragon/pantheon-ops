package org

import (
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/testmysql"
)

func setupPostTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := testmysql.Open(t)
	if err := db.AutoMigrate(&SystemPost{}); err != nil {
		t.Fatalf("migrate post: %v", err)
	}
	if err := db.Exec("CREATE TABLE IF NOT EXISTS system_user (id INTEGER PRIMARY KEY AUTO_INCREMENT, post_id INTEGER, deleted_at DATETIME)").Error; err != nil {
		t.Fatalf("create user fixture: %v", err)
	}
	if err := db.Exec("CREATE TABLE IF NOT EXISTS system_dept (id INTEGER PRIMARY KEY, parent_id INTEGER, is_root INTEGER, dept_name TEXT)").Error; err != nil {
		t.Fatalf("create dept fixture: %v", err)
	}
	if err := db.Exec("INSERT INTO system_dept (id, parent_id, is_root, dept_name) VALUES (1, 0, 1, 'Pantheon Base'), (10, 1, 0, '研发中心')").Error; err != nil {
		t.Fatalf("seed dept fixture: %v", err)
	}
	return db
}

type postFixtureUser struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement"`
	PostID    uint64         `gorm:"column:post_id"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (postFixtureUser) TableName() string {
	return "system_user"
}

func migratePostFixtureUsers(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&postFixtureUser{}); err != nil {
		t.Fatalf("migrate user fixture: %v", err)
	}
}

func TestPostService_DeleteReleasesPostCode(t *testing.T) {
	db := setupPostTestDB(t)
	service := NewPostService(db)

	created, err := service.CreatePost(&PostCreateReq{
		PostCode: "developer",
		PostName: "Developer",
		DeptID:   10,
		Status:   1,
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := service.DeletePost(created.ID); err != nil {
		t.Fatalf("delete post: %v", err)
	}

	var deleted SystemPost
	if err := db.Unscoped().First(&deleted, created.ID).Error; err != nil {
		t.Fatalf("load deleted post: %v", err)
	}
	if !strings.HasPrefix(deleted.PostCode, deletedPostCodePrefix) {
		t.Fatalf("expected archived post code, got %s", deleted.PostCode)
	}

	recreated, err := service.CreatePost(&PostCreateReq{
		PostCode: "developer",
		PostName: "Developer",
		DeptID:   10,
		Status:   1,
	})
	if err != nil {
		t.Fatalf("recreate post with same code: %v", err)
	}
	if recreated.PostCode != "developer" {
		t.Fatalf("expected developer, got %s", recreated.PostCode)
	}
}

func TestPostService_DeleteIgnoresSoftDeletedUsers(t *testing.T) {
	db := setupPostTestDB(t)
	service := NewPostService(db)

	created, err := service.CreatePost(&PostCreateReq{
		PostCode: "ops",
		PostName: "Operations",
		DeptID:   10,
		Status:   1,
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := db.Exec("INSERT INTO system_user (id, post_id, deleted_at) VALUES (1, ?, ?)", created.ID, time.Now()).Error; err != nil {
		t.Fatalf("seed soft deleted user: %v", err)
	}

	if err := service.DeletePost(created.ID); err != nil {
		t.Fatalf("delete post with soft deleted user: %v", err)
	}
}

func TestPostService_MigrateReleasesLegacyDeletedPostCode(t *testing.T) {
	db := setupPostTestDB(t)
	service := NewPostService(db)

	legacy := SystemPost{
		DeptID:   10,
		PostCode: "legacy_post",
		PostName: "Legacy Post",
		Status:   1,
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("seed legacy post: %v", err)
	}
	if err := db.Model(&legacy).Update("deleted_at", time.Now()).Error; err != nil {
		t.Fatalf("soft delete legacy post: %v", err)
	}

	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate post: %v", err)
	}
	var repaired SystemPost
	if err := db.Unscoped().First(&repaired, legacy.ID).Error; err != nil {
		t.Fatalf("load repaired post: %v", err)
	}
	if !strings.HasPrefix(repaired.PostCode, deletedPostCodePrefix) {
		t.Fatalf("expected archived legacy post code, got %s", repaired.PostCode)
	}

	if _, err := service.CreatePost(&PostCreateReq{DeptID: 10, PostCode: "legacy_post", PostName: "Legacy Post", Status: 1}); err != nil {
		t.Fatalf("expected legacy post code to be reusable: %v", err)
	}
}

func TestPostService_BatchUpdatePostStatus(t *testing.T) {
	db := setupPostTestDB(t)
	service := NewPostService(db)

	created, err := service.CreatePost(&PostCreateReq{
		PostCode: "batch_post",
		PostName: "Batch Post",
		DeptID:   10,
		Status:   1,
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}

	updated, err := service.BatchUpdatePostStatus([]uint64{created.ID, created.ID}, 2)
	if err != nil {
		t.Fatalf("batch disable post: %v", err)
	}
	if updated != 1 {
		t.Fatalf("expected 1 updated post, got %d", updated)
	}
	var disabled SystemPost
	if err := db.First(&disabled, created.ID).Error; err != nil {
		t.Fatalf("load disabled post: %v", err)
	}
	if disabled.Status != 2 {
		t.Fatalf("expected post status 2, got %d", disabled.Status)
	}

	if _, err := service.BatchUpdatePostStatus([]uint64{999}, 1); err == nil || common.ErrMessage(err) != "post.batch.not_found" {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestPostService_UpdateRejectsDisableWhenUsersAssigned(t *testing.T) {
	db := setupPostTestDB(t)
	service := NewPostService(db)

	created, err := service.CreatePost(&PostCreateReq{
		PostCode: "frontend",
		PostName: "Frontend Engineer",
		DeptID:   10,
		Status:   1,
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := db.Exec("INSERT INTO system_user (id, post_id, deleted_at) VALUES (1, ?, NULL)", created.ID).Error; err != nil {
		t.Fatalf("seed active user: %v", err)
	}

	_, err = service.UpdatePost(created.ID, &PostUpdateReq{
		DeptID:   10,
		PostCode: created.PostCode,
		PostName: created.PostName,
		Sort:     created.Sort,
		Status:   2,
		Remark:   created.Remark,
	})
	if err == nil || common.ErrMessage(err) != "post.status.error.has_users" {
		t.Fatalf("expected disable blocked by active users, got %v", err)
	}
}

func TestPostService_BatchDisableRejectsPostsAssignedToUsers(t *testing.T) {
	db := setupPostTestDB(t)
	service := NewPostService(db)

	created, err := service.CreatePost(&PostCreateReq{
		PostCode: "backend",
		PostName: "Backend Engineer",
		DeptID:   10,
		Status:   1,
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := db.Exec("INSERT INTO system_user (id, post_id, deleted_at) VALUES (1, ?, NULL)", created.ID).Error; err != nil {
		t.Fatalf("seed active user: %v", err)
	}

	_, err = service.BatchUpdatePostStatus([]uint64{created.ID}, 2)
	if err == nil || common.ErrMessage(err) != "post.status.error.has_users" {
		t.Fatalf("expected batch disable blocked by active users, got %v", err)
	}
}

func TestPostService_ImportTemplateAndExport(t *testing.T) {
	db := setupPostTestDB(t)
	service := NewPostService(db)

	template := service.BuildPostImportTemplate()
	if len(template.Rows) == 0 || !strings.HasPrefix(template.Rows[0][0], "#") {
		t.Fatalf("expected template to include ignored instruction rows, got %+v", template.Rows)
	}
	templateResult, err := service.ImportPosts(append([][]string{template.Headers}, template.Rows...))
	if err != nil {
		t.Fatalf("import template comments: %v", err)
	}
	if !templateResult.Applied || templateResult.Created != 0 || templateResult.Failed != 0 {
		t.Fatalf("expected template comments to be ignored, got %+v", templateResult)
	}

	result, err := service.ImportPosts([][]string{
		template.Headers,
		{"Pantheon Base/研发中心", "developer", "研发工程师", "10", "1", "负责研发交付"},
	})
	if err != nil {
		t.Fatalf("import post: %v", err)
	}
	if !result.Applied || result.Created != 1 || result.Failed != 0 {
		t.Fatalf("unexpected import result: %+v", result)
	}

	exported, err := service.ExportPosts(&PostListQuery{PostCode: "developer"})
	if err != nil {
		t.Fatalf("export post: %v", err)
	}
	if len(exported.Rows) != 1 || exported.Rows[0][1] != "developer" || exported.Rows[0][2] != "研发工程师" {
		t.Fatalf("unexpected export rows: %+v", exported.Rows)
	}
	if exported.Headers[len(exported.Headers)-1] != "governanceActionsLabel" {
		t.Fatalf("expected governance export headers, got %+v", exported.Headers)
	}
}

func TestPostService_ExportPostsIncludesGovernanceColumns(t *testing.T) {
	db := setupPostTestDB(t)
	service := NewPostService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate post service: %v", err)
	}

	migratePostFixtureUsers(t, db)

	activePost := SystemPost{
		DeptID:   10,
		PostCode: "developer",
		PostName: "研发工程师",
		Status:   1,
	}
	if err := db.Create(&activePost).Error; err != nil {
		t.Fatalf("seed active post: %v", err)
	}
	disabledPost := SystemPost{
		DeptID:   10,
		PostCode: "assistant",
		PostName: "助理",
		Status:   2,
	}
	if err := db.Create(&disabledPost).Error; err != nil {
		t.Fatalf("seed disabled post: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_user (post_id, deleted_at) VALUES (?, NULL)`, activePost.ID).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	exported, err := service.ExportPosts(&PostListQuery{})
	if err != nil {
		t.Fatalf("export posts: %v", err)
	}
	if len(exported.Rows) != 2 {
		t.Fatalf("expected 2 exported rows, got %+v", exported.Rows)
	}

	var activeRow []string
	var disabledRow []string
	for _, row := range exported.Rows {
		switch row[1] {
		case "developer":
			activeRow = row
		case "assistant":
			disabledRow = row
		}
	}
	if len(activeRow) == 0 || len(disabledRow) == 0 {
		t.Fatalf("expected active and disabled rows, got %+v", exported.Rows)
	}
	if activeRow[6] != "1" || activeRow[7] != "post" || activeRow[8] != "in-use" || activeRow[9] != "1" || activeRow[10] != "users" || activeRow[11] != "reassign-users" || activeRow[12] != "Post" || activeRow[13] != "Assigned Members" || activeRow[14] != "Users" || activeRow[15] != "Reassign Users" {
		t.Fatalf("unexpected active governance export row: %+v", activeRow)
	}
	if disabledRow[6] != "0" || disabledRow[7] != "post" || disabledRow[8] != "disabled" || disabledRow[9] != "0" || disabledRow[10] != "none" || disabledRow[11] != "delete-or-keep-disabled" || disabledRow[12] != "Post" || disabledRow[13] != "Disabled" || disabledRow[14] != "No Blocker" || disabledRow[15] != "Delete or Keep Disabled" {
		t.Fatalf("unexpected disabled governance export row: %+v", disabledRow)
	}
}

func TestPostService_ListPostsIncludesGovernanceFields(t *testing.T) {
	db := setupPostTestDB(t)
	service := NewPostService(db)

	activePost, err := service.CreatePost(&PostCreateReq{
		DeptID:   10,
		PostCode: "developer",
		PostName: "研发工程师",
		Status:   1,
	})
	if err != nil {
		t.Fatalf("create active post: %v", err)
	}
	if _, err := service.CreatePost(&PostCreateReq{
		DeptID:   10,
		PostCode: "assistant",
		PostName: "助理",
		Status:   2,
	}); err != nil {
		t.Fatalf("create disabled post: %v", err)
	}
	if err := db.Exec("INSERT INTO system_user (id, post_id, deleted_at) VALUES (1, ?, NULL)", activePost.ID).Error; err != nil {
		t.Fatalf("seed active user: %v", err)
	}

	resp, err := service.ListPosts(&PostListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list posts: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 posts, got %+v", resp.Items)
	}

	var inUse *PostListResp
	var disabled *PostListResp
	for index := range resp.Items {
		row := &resp.Items[index]
		switch row.PostCode {
		case "developer":
			inUse = row
		case "assistant":
			disabled = row
		}
	}
	if inUse == nil || disabled == nil {
		t.Fatalf("expected developer and assistant rows, got %+v", resp.Items)
	}
	if inUse.AssignedUserCount != 1 || len(inUse.GovernanceTags) != 1 || inUse.GovernanceTags[0] != "in-use" {
		t.Fatalf("unexpected in-use governance row: %+v", inUse)
	}
	if len(disabled.GovernanceTags) != 1 || disabled.GovernanceTags[0] != "disabled" {
		t.Fatalf("unexpected disabled governance row: %+v", disabled)
	}
}
