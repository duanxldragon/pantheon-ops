package system

import (
	"testing"

	"pantheon-platform/backend/internal/middleware"
)

func TestDetectOperationLogSourceDomain(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want string
	}{
		{name: "config setting", url: "/api/v1/system/setting/group/ui", want: operationLogSourceDomainConfig},
		{name: "config upload", url: "/api/v1/system/upload", want: operationLogSourceDomainConfig},
		{name: "config i18n", url: "/api/v1/system/i18n/lifecycle/archive", want: operationLogSourceDomainConfig},
		{name: "audit", url: "/api/v1/system/operation-log/list", want: operationLogSourceDomainAudit},
		{name: "auth", url: "/api/v1/system/session/list", want: operationLogSourceDomainAuth},
		{name: "iam", url: "/api/v1/system/user/list", want: operationLogSourceDomainIAM},
		{name: "org", url: "/api/v1/system/dept/tree", want: operationLogSourceDomainOrg},
		{name: "platform", url: "/dashboard/summary", want: operationLogSourceDomainPlatform},
		{name: "other", url: "/api/v1/system/dict/type/list", want: operationLogSourceDomainOther},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectOperationLogSourceDomain(tc.url); got != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, got)
			}
		})
	}
}

func TestDetectOperationLogSourcePage(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want string
	}{
		{name: "setting", url: "/api/v1/system/setting/group/ui", want: operationLogSourcePageSetting},
		{name: "upload", url: "/api/v1/system/upload", want: operationLogSourcePageUpload},
		{name: "i18n", url: "/api/v1/system/i18n/lifecycle/archive", want: operationLogSourcePageI18n},
		{name: "operation log", url: "/api/v1/system/operation-log/list", want: operationLogSourcePageOperationLog},
		{name: "login log", url: "/api/v1/system/login-log/list", want: operationLogSourcePageLoginLog},
		{name: "session", url: "/api/v1/system/session/list", want: operationLogSourcePageSession},
		{name: "user", url: "/api/v1/system/user/list", want: operationLogSourcePageUser},
		{name: "role", url: "/api/v1/system/role/list", want: operationLogSourcePageRole},
		{name: "menu", url: "/api/v1/system/menu/tree", want: operationLogSourcePageMenu},
		{name: "permission", url: "/api/v1/system/permission/list", want: operationLogSourcePagePermission},
		{name: "dept", url: "/api/v1/system/dept/tree", want: operationLogSourcePageDept},
		{name: "post", url: "/api/v1/system/post/list", want: operationLogSourcePagePost},
		{name: "dashboard", url: "/dashboard/summary", want: operationLogSourcePageDashboard},
		{name: "other", url: "/api/v1/system/dict/type/list", want: operationLogSourcePageOther},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectOperationLogSourcePage(tc.url); got != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, got)
			}
		})
	}
}

func TestDetectOperationLogFailureCategory(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		errorMsg   string
		jsonResult string
		want       string
	}{
		{name: "validation", status: 2, errorMsg: "upload.file.type_not_allowed", jsonResult: `{"code":400,"message":"upload.file.type_not_allowed"}`, want: operationLogFailureValidation},
		{name: "auth", status: 2, errorMsg: "auth.token.invalid", jsonResult: `{"code":401,"message":"auth.token.invalid"}`, want: operationLogFailureAuth},
		{name: "permission", status: 2, errorMsg: "permission.denied", jsonResult: `{"code":403,"message":"permission.denied"}`, want: operationLogFailurePermission},
		{name: "server", status: 2, errorMsg: "database.not_initialized", jsonResult: `{"code":500,"message":"database.not_initialized"}`, want: operationLogFailureServer},
		{name: "business fallback", status: 2, errorMsg: "custom.rule.rejected", jsonResult: `{"code":409,"message":"custom.rule.rejected"}`, want: operationLogFailureBusiness},
		{name: "success empty", status: 1, errorMsg: "", jsonResult: `{"code":200,"message":"success"}`, want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectOperationLogFailureCategory(tc.status, tc.errorMsg, tc.jsonResult); got != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, got)
			}
		})
	}
}

func TestFilterOperationLogs(t *testing.T) {
	rows := []middleware.SystemLogOper{
		{OperURL: "/api/v1/system/upload", Status: 2, ErrorMsg: "upload.file.type_not_allowed", JsonResult: `{"code":400,"message":"upload.file.type_not_allowed"}`},
		{OperURL: "/api/v1/system/user", Status: 2, ErrorMsg: "permission.denied", JsonResult: `{"code":403,"message":"permission.denied"}`},
		{OperURL: "/dashboard/summary", Status: 1, ErrorMsg: "", JsonResult: `{"code":200,"message":"success"}`},
	}

	filtered := filterOperationLogs(rows, &OperationLogQuery{
		SourceDomain:    operationLogSourceDomainConfig,
		SourcePage:      operationLogSourcePageUpload,
		FailureCategory: operationLogFailureValidation,
	})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 row, got %d", len(filtered))
	}
	if filtered[0].OperURL != "/api/v1/system/upload" {
		t.Fatalf("unexpected filtered row: %s", filtered[0].OperURL)
	}
}
