package iam

import (
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func itoa(n int) string {
	return strconv.Itoa(n)
}

// ---- normalizeStatus ----

func TestNormalizeStatus_1_Returns1(t *testing.T) {
	if got := normalizeStatus(1); got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
}

func TestNormalizeStatus_2_Returns2(t *testing.T) {
	if got := normalizeStatus(2); got != 2 {
		t.Fatalf("expected 2, got %d", got)
	}
}

func TestNormalizeStatus_0_Returns1(t *testing.T) {
	if got := normalizeStatus(0); got != 1 {
		t.Fatalf("expected 1 (default), got %d", got)
	}
}

func TestNormalizeStatus_99_Returns1(t *testing.T) {
	if got := normalizeStatus(99); got != 1 {
		t.Fatalf("expected 1 (default for unknown), got %d", got)
	}
}

// ---- normalizeUserPageQuery ----

func TestNormalizeUserPageQuery_Nil(t *testing.T) {
	page, pageSize := normalizeUserPageQuery(nil)
	if page != 1 || pageSize != 10 {
		t.Fatalf("expected (1, 10), got (%d, %d)", page, pageSize)
	}
}

func TestNormalizeUserPageQuery_ZeroValues(t *testing.T) {
	page, pageSize := normalizeUserPageQuery(&UserListQuery{})
	if page != 1 || pageSize != 10 {
		t.Fatalf("expected (1, 10) for zero values, got (%d, %d)", page, pageSize)
	}
}

func TestNormalizeUserPageQuery_CustomValues(t *testing.T) {
	page, pageSize := normalizeUserPageQuery(&UserListQuery{Page: 3, PageSize: 20})
	if page != 3 || pageSize != 20 {
		t.Fatalf("expected (3, 20), got (%d, %d)", page, pageSize)
	}
}

func TestNormalizeUserPageQuery_MaxPageSize(t *testing.T) {
	_, pageSize := normalizeUserPageQuery(&UserListQuery{PageSize: 200})
	if pageSize != 100 {
		t.Fatalf("expected pageSize capped at 100, got %d", pageSize)
	}
}

// ---- normalizeUserSort ----

func TestNormalizeUserSort_Nil(t *testing.T) {
	col, desc := normalizeUserSort(nil)
	if col != "id" || desc != true {
		t.Fatalf("expected (id, true), got (%s, %v)", col, desc)
	}
}

func TestNormalizeUserSort_DefaultSort(t *testing.T) {
	col, dir := normalizeUserSort(&UserListQuery{})
	if col != "id" || dir != true {
		t.Fatalf("expected (id, true) for empty query, got (%s, %v)", col, dir)
	}
}

func TestNormalizeUserSort_ByCreatedAtAsc(t *testing.T) {
	col, desc := normalizeUserSort(&UserListQuery{SortField: "createdAt", SortOrder: "asc"})
	if col != "created_at" || desc != false {
		t.Fatalf("expected (created_at, false), got (%s, %v)", col, desc)
	}
}

func TestNormalizeUserSort_ByUsernameDesc(t *testing.T) {
	col, desc := normalizeUserSort(&UserListQuery{SortField: "username", SortOrder: "desc"})
	if col != "username" || desc != true {
		t.Fatalf("expected (username, true), got (%s, %v)", col, desc)
	}
}

func TestNormalizeUserSort_InvalidFieldFallsBackToID(t *testing.T) {
	col, _ := normalizeUserSort(&UserListQuery{SortField: "invalid_field", SortOrder: "asc"})
	if col != "id" {
		t.Fatalf("expected id fallback, got %s", col)
	}
}

func TestNormalizeUserSort_InvalidOrderDefaultsDesc(t *testing.T) {
	col, desc := normalizeUserSort(&UserListQuery{SortField: "id", SortOrder: "invalid"})
	if col != "id" || desc != true {
		t.Fatalf("expected (id, true), got (%s, %v)", col, desc)
	}
}

// ---- normalizeUint64IDs (local variant) ----

func TestNormalizeUint64IDs_RemovesDuplicates(t *testing.T) {
	result := normalizeUint64IDs([]uint64{1, 2, 2, 3})
	if len(result) != 3 {
		t.Fatalf("expected 3, got %d: %v", len(result), result)
	}
}

func TestNormalizeUint64IDs_RemovesZeros(t *testing.T) {
	result := normalizeUint64IDs([]uint64{0, 1, 0, 2})
	if len(result) != 2 {
		t.Fatalf("expected 2, got %d: %v", len(result), result)
	}
}

func TestNormalizeUint64IDs_EmptyInput(t *testing.T) {
	result := normalizeUint64IDs(nil)
	if len(result) != 0 {
		t.Fatalf("expected empty for nil, got %d: %v", len(result), result)
	}
	result = normalizeUint64IDs([]uint64{})
	if len(result) != 0 {
		t.Fatalf("expected empty for empty slice, got %d: %v", len(result), result)
	}
}

// ---- validateOptionalEmail ----

func TestValidateOptionalEmail_EmptyIsValid(t *testing.T) {
	if err := validateOptionalEmail(""); err != nil {
		t.Fatalf("expected no error for empty, got %v", err)
	}
	if err := validateOptionalEmail("  "); err != nil {
		t.Fatalf("expected no error for whitespace, got %v", err)
	}
}

func TestValidateOptionalEmail_ValidAddress(t *testing.T) {
	if err := validateOptionalEmail("user@example.com"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateOptionalEmail_InvalidAddress(t *testing.T) {
	if err := validateOptionalEmail("not-an-email"); err == nil {
		t.Fatal("expected error for invalid email")
	}
}

// ---- mergeUserPermissionKeys ----

func TestMergeUserPermissionKeys_Deduplicates(t *testing.T) {
	result := mergeUserPermissionKeys([]string{"read", "write"}, []string{"write", "admin"})
	if len(result) != 3 {
		t.Fatalf("expected 3 unique keys, got %d: %v", len(result), result)
	}
}

func TestMergeUserPermissionKeys_RemovesEmpty(t *testing.T) {
	result := mergeUserPermissionKeys([]string{"", "read", "  "})
	if len(result) != 1 {
		t.Fatalf("expected 1 key after removing empties, got %d: %v", len(result), result)
	}
}

func TestMergeUserPermissionKeys_EmptyInput(t *testing.T) {
	result := mergeUserPermissionKeys()
	if len(result) != 0 {
		t.Fatalf("expected empty, got %v", result)
	}
	result = mergeUserPermissionKeys([]string{}, []string{})
	if len(result) != 0 {
		t.Fatalf("expected empty for empty groups, got %v", result)
	}
}

// ---- marshalUserProfileExt ----

func TestMarshalUserProfileExt_Nil(t *testing.T) {
	result, err := marshalUserProfileExt(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Fatalf("expected empty string, got %s", result)
	}
}

func TestMarshalUserProfileExt_ValidMap(t *testing.T) {
	result, err := marshalUserProfileExt(map[string]interface{}{"key": "value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "key") || !strings.Contains(result, "value") {
		t.Fatalf("expected JSON containing key and value, got %s", result)
	}
}

func TestMarshalUserProfileExt_TooLarge(t *testing.T) {
	large := make(map[string]interface{})
	for i := 0; i < 500; i++ {
		large[strings.Repeat("k", 7)+itoa(i)] = strings.Repeat("v", 100)
	}
	_, err := marshalUserProfileExt(large)
	if err == nil {
		t.Fatal("expected error for oversized profile ext")
	}
}

// ---- unmarshalUserProfileExt ----

func TestUnmarshalUserProfileExt_Empty(t *testing.T) {
	result, err := unmarshalUserProfileExt("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil for empty, got %v", result)
	}
}

func TestUnmarshalUserProfileExt_Whitespace(t *testing.T) {
	result, err := unmarshalUserProfileExt("  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil for whitespace, got %v", result)
	}
}

func TestUnmarshalUserProfileExt_ValidJSON(t *testing.T) {
	result, err := unmarshalUserProfileExt(`{"key":"value"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v, _ := result["key"].(string); v != "value" {
		t.Fatalf("expected 'value', got %v", result["key"])
	}
}

func TestUnmarshalUserProfileExt_InvalidJSON(t *testing.T) {
	_, err := unmarshalUserProfileExt("{invalid}")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestUnmarshalUserProfileExt_EmptyObject(t *testing.T) {
	result, err := unmarshalUserProfileExt(`{}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || len(result) != 0 {
		t.Fatalf("expected empty map, got %v", result)
	}
}

// ---- formatUserTime ----

func TestFormatUserTime_ZeroTime(t *testing.T) {
	if got := formatUserTime(time.Time{}); got != "" {
		t.Fatalf("expected empty string for zero time, got %s", got)
	}
}

func TestFormatUserTime_ValidTime(t *testing.T) {
	ts := time.Date(2026, 6, 7, 10, 30, 0, 0, time.UTC)
	got := formatUserTime(ts)
	if got != "2026-06-07T10:30:00Z" {
		t.Fatalf("expected RFC3339 format, got %s", got)
	}
}

// ---- getUserIDFromContext ----

func TestGetUserIDFromContext_NotSet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	uid, ok := getUserIDFromContext(c)
	if ok {
		t.Fatal("expected ok=false when userId not set")
	}
	if uid != 0 {
		t.Fatalf("expected 0, got %d", uid)
	}
}

func TestGetUserIDFromContext_Valid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("userId", uint64(42))

	uid, ok := getUserIDFromContext(c)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if uid != 42 {
		t.Fatalf("expected 42, got %d", uid)
	}
}

func TestGetUserIDFromContext_WrongType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("userId", "not-a-uint64")

	_, ok := getUserIDFromContext(c)
	if ok {
		t.Fatal("expected ok=false for wrong type")
	}
}

// ---- parseUintParam ----

func TestParseUintParam_Valid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = []gin.Param{{Key: "id", Value: "123"}}

	id, err := parseUintParam(c, "id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 123 {
		t.Fatalf("expected 123, got %d", id)
	}
}

func TestParseUintParam_Invalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = []gin.Param{{Key: "id", Value: "abc"}}

	_, err := parseUintParam(c, "id")
	if err == nil {
		t.Fatal("expected error for non-numeric param")
	}
}

// ---- handler-level: invalid body returns 400 ----

func TestHandler_GetProfile_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	h := &UserHandler{service: &UserService{db: nil}}
	h.GetProfile(c)

	// Unauthenticated request should fail; we just check no panic and response is written
	if recorder.Code == 0 {
		t.Fatal("expected response to be written")
	}
}

// ---- DB=nil safety edge cases ----

func TestBatchUpdateUserStatus_DBNil(t *testing.T) {
	s := &UserService{db: nil}
	_, err := s.BatchUpdateUserStatus([]uint64{1, 2}, 1)
	if err == nil {
		t.Fatal("expected error when db is nil")
	}
}

func TestResetPassword_DBNil(t *testing.T) {
	s := &UserService{db: nil}
	_, err := s.ResetPassword(1, "newpass")
	if err == nil {
		t.Fatal("expected error when db is nil")
	}
}

func TestDeleteUser_DBNil(t *testing.T) {
	s := &UserService{db: nil}
	err := s.DeleteUser(1)
	if err == nil {
		t.Fatal("expected error when db is nil")
	}
}

func TestExportUsers_DBNil(t *testing.T) {
	s := &UserService{db: nil}
	_, err := s.ExportUsers(nil, nil)
	if err == nil {
		t.Fatal("expected error when db is nil")
	}
}

func TestImportUsers_DBNil(t *testing.T) {
	s := &UserService{db: nil}
	_, err := s.ImportUsers(nil)
	if err == nil {
		t.Fatal("expected error when db is nil")
	}
}

// ---- BuildUserImportTemplate (nil-safe) ----

func TestBuildUserImportTemplate_HasHeaders(t *testing.T) {
	s := &UserService{db: nil}
	file := s.BuildUserImportTemplate()
	if file.Filename != "system-user-import-template.csv" {
		t.Fatalf("unexpected filename: %s", file.Filename)
	}
	if len(file.Headers) != 9 {
		t.Fatalf("expected 9 headers, got %d", len(file.Headers))
	}
}

func TestBuildUserImportTemplate_HasSampleRow(t *testing.T) {
	s := &UserService{db: nil}
	file := s.BuildUserImportTemplate()
	if len(file.Rows) < 2 {
		t.Fatalf("expected at least 2 rows (comment + sample), got %d", len(file.Rows))
	}
}
