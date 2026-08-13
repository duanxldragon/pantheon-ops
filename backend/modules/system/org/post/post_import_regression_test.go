package org

import (
	"errors"
	"testing"

	"pantheon-base/pkg/impexp"
)

// 以下回归测试用于锁定 ImportPosts 的事务边界、逐行校验、重复处理以及
// ListPosts 筛选口径。它们在有 PANTHEON_TEST_DSN 的环境下真实执行；
// 无 DSN 时由 testmysql.Open 自动跳过，不计入失败。

// importRecords 以表头 + 数据行构造 ImportPosts 所需记录。
func importRecords(rows ...[]string) [][]string {
	header := []string{"deptPath", "postCode", "postName", "sort", "status", "remark"}
	records := make([][]string, 0, len(rows)+1)
	records = append(records, header)
	records = append(records, rows...)
	return records
}

// testImportPostCode 是回归测试共用的导入岗位编码。
const testImportPostCode = "developer"

// TestPostService_ImportPostsUpdatesExistingPost 锁定：
// 同一 post_code 二次导入应走“更新”路径（Updated++、Created=0），
// 且持久化值反映最新字段；事务语义不被弱化。
func TestPostService_ImportPostsUpdatesExistingPost(t *testing.T) {
	db := setupPostTestDB(t)
	service := NewPostService(db)

	first, err := service.ImportPosts(importRecords([]string{
		"Pantheon Base/研发中心", testImportPostCode, "研发工程师", "10", "1", "负责研发交付",
	}))
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if !first.Applied || first.Created != 1 || first.Updated != 0 || first.Failed != 0 {
		t.Fatalf("unexpected first import result: %+v", first)
	}

	second, err := service.ImportPosts(importRecords([]string{
		"Pantheon Base/研发中心", testImportPostCode, "高级研发工程师", "20", "2", "负责核心研发",
	}))
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if !second.Applied || second.Created != 0 || second.Updated != 1 || second.Failed != 0 {
		t.Fatalf("unexpected second import result: %+v", second)
	}

	var updated SystemPost
	if err := db.Where("post_code = ?", testImportPostCode).First(&updated).Error; err != nil {
		t.Fatalf("load updated post: %v", err)
	}
	if updated.PostName != "高级研发工程师" || updated.Sort != 20 || updated.Status != 2 || updated.Remark != "负责核心研发" {
		t.Fatalf("post not updated as expected: %+v", updated)
	}
}

// TestPostService_ImportPostsRejectsDuplicateRowsInFile 锁定：
// 同一文件内重复 post_code 必须被拒绝（Failed=1），且批量在写入前整体返回、
// 不创建任何岗位（Applied=false）。错误信息需指向首次出现的行号。
func TestPostService_ImportPostsRejectsDuplicateRowsInFile(t *testing.T) {
	db := setupPostTestDB(t)
	service := NewPostService(db)

	result, err := service.ImportPosts(importRecords(
		[]string{"Pantheon Base/研发中心", testImportPostCode, "研发工程师", "10", "1", "r1"},
		[]string{"Pantheon Base/研发中心", testImportPostCode, "研发工程师", "10", "1", "r2"},
	))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if result.Applied {
		t.Fatalf("expected import not applied due to duplicate, got %+v", result)
	}
	if result.Failed != 1 {
		t.Fatalf("expected 1 failed row, got %d (%+v)", result.Failed, result.Errors)
	}
	if len(result.Errors) != 1 || result.Errors[0].Field != "postCode" {
		t.Fatalf("expected postCode duplicate error, got %+v", result.Errors)
	}
	if result.Errors[0].Row != 3 || result.Errors[0].Message != "import.duplicate.row.2" {
		t.Fatalf("expected duplicate row 3 message 'import.duplicate.row.2', got %+v", result.Errors[0])
	}

	var count int64
	if err := db.Model(&SystemPost{}).Where("post_code = ?", testImportPostCode).Count(&count).Error; err != nil {
		t.Fatalf("count posts: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no post created on duplicate batch, got %d", count)
	}
}

// TestPostService_ImportPostsRejectsMissingHeader 锁定：缺必填表头即整体拒绝。
func TestPostService_ImportPostsRejectsMissingHeader(t *testing.T) {
	db := setupPostTestDB(t)
	service := NewPostService(db)

	result, err := service.ImportPosts([][]string{
		{"deptPath", "postName", "sort", "status", "remark"},
		{"Pantheon Base/研发中心", "研发工程师", "10", "1", "r"},
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if result.Applied || result.Failed != 1 {
		t.Fatalf("expected missing-header rejection, got %+v", result)
	}
	if len(result.Errors) != 1 || result.Errors[0].Field != "postCode" || result.Errors[0].Message != "import.header.missing" {
		t.Fatalf("expected postCode import.header.missing, got %+v", result.Errors)
	}
}

// TestPostService_ImportPostsEmptyFile 锁定：空文件返回 import.file.empty 且不写库。
func TestPostService_ImportPostsEmptyFile(t *testing.T) {
	db := setupPostTestDB(t)
	service := NewPostService(db)

	result, err := service.ImportPosts([][]string{})
	if err != nil {
		t.Fatalf("import empty: %v", err)
	}
	if result.Applied || result.Failed != 1 {
		t.Fatalf("expected empty-file rejection, got %+v", result)
	}
	if len(result.Errors) != 1 || result.Errors[0].Message != "import.file.empty" {
		t.Fatalf("expected import.file.empty, got %+v", result.Errors)
	}
}

// TestPostService_ImportPostsRejectsInvalidDept 锁定：不存在的部门路径被拒绝。
func TestPostService_ImportPostsRejectsInvalidDept(t *testing.T) {
	db := setupPostTestDB(t)
	service := NewPostService(db)

	result, err := service.ImportPosts(importRecords([]string{
		"不存在的部门", testImportPostCode, "研发工程师", "10", "1", "r",
	}))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if result.Applied || result.Failed != 1 {
		t.Fatalf("expected invalid-dept rejection, got %+v", result)
	}
	if len(result.Errors) != 1 || result.Errors[0].Field != "deptPath" || result.Errors[0].Message != "post.dept.invalid" {
		t.Fatalf("expected deptPath post.dept.invalid, got %+v", result.Errors)
	}
}

// TestPostService_ImportPostsRejectsRootDept 锁定：根部门（is_root=1）被禁止导入。
func TestPostService_ImportPostsRejectsRootDept(t *testing.T) {
	db := setupPostTestDB(t)
	service := NewPostService(db)

	result, err := service.ImportPosts(importRecords([]string{
		"Pantheon Base", testImportPostCode, "研发工程师", "10", "1", "r",
	}))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if result.Applied || result.Failed != 1 {
		t.Fatalf("expected root-dept rejection, got %+v", result)
	}
	if len(result.Errors) != 1 || result.Errors[0].Field != "deptPath" || result.Errors[0].Message != "post.dept.root_forbidden" {
		t.Fatalf("expected deptPath post.dept.root_forbidden, got %+v", result.Errors)
	}
}

// TestAppendPostImportDeptErrorPropagatesUnexpectedError 锁定：
// 部门校验中的基础设施错误必须返回调用方，不能泄露到逐行 i18n 错误。
func TestAppendPostImportDeptErrorPropagatesUnexpectedError(t *testing.T) {
	wantErr := errors.New("forced post department query failure")
	result := &impexp.ImportResult{}
	err := appendPostImportDeptError(result, 2, wantErr)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected database error %v to propagate, got %v", wantErr, err)
	}
	if result.Failed != 0 || len(result.Errors) != 0 {
		t.Fatalf("expected no row errors for infrastructure failure, got %+v", result)
	}
}

// TestPostService_ListPostsFiltersByPostCode 锁定：applyPostListFilters 的
// post_code LIKE 筛选口径与原 ListPosts 一致（独立筛选，不影响其它条件）。
func TestPostService_ListPostsFiltersByPostCode(t *testing.T) {
	db := setupPostTestDB(t)
	service := NewPostService(db)

	if _, err := service.CreatePost(&PostCreateReq{DeptID: 10, PostCode: testImportPostCode, PostName: "研发工程师", Status: 1}); err != nil {
		t.Fatalf("create developer: %v", err)
	}
	if _, err := service.CreatePost(&PostCreateReq{DeptID: 10, PostCode: "manager", PostName: "经理", Status: 1}); err != nil {
		t.Fatalf("create manager: %v", err)
	}

	resp, err := service.ListPosts(&PostListQuery{PostCode: "dev", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list posts: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected 1 post matching 'dev', got total=%d items=%d", resp.Total, len(resp.Items))
	}
	if len(resp.Items) != 1 || resp.Items[0].PostCode != testImportPostCode {
		t.Fatalf("unexpected list result: %+v", resp.Items)
	}
}
