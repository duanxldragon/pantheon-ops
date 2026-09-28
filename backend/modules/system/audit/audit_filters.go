package system

import (
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/internal/middleware"
)

const (
	operationLogSourceDomainPlatform = "platform"
	operationLogSourceDomainAuth     = "auth"
	operationLogSourceDomainIAM      = "iam"
	operationLogSourceDomainOrg      = "org"
	operationLogSourceDomainConfig   = "config"
	operationLogSourceDomainAudit    = "audit"
	operationLogSourceDomainOther    = "other"

	operationLogSourcePageDashboard    = "dashboard"
	operationLogSourcePageSetting      = "setting"
	operationLogSourcePageUpload       = "upload"
	operationLogSourcePageI18n         = "i18n"
	operationLogSourcePageOperationLog = "operationLog"
	operationLogSourcePageLoginLog     = "loginLog"
	operationLogSourcePageSession      = "session"
	operationLogSourcePageUser         = "user"
	operationLogSourcePageRole         = "role"
	operationLogSourcePageMenu         = "menu"
	operationLogSourcePagePermission   = "permission"
	operationLogSourcePageDept         = "dept"
	operationLogSourcePagePost         = "post"
	operationLogSourcePageOther        = "other"

	operationLogFailureValidation = "validation"
	operationLogFailureAuth       = "auth"
	operationLogFailurePermission = "permission"
	operationLogFailureServer     = "server"
	operationLogFailureBusiness   = "business"
)

func detectOperationLogSourceDomain(operURL string) string {
	return middleware.DetectOperationLogSourceDomain(operURL)
}

func detectOperationLogSourcePage(operURL string) string {
	return middleware.DetectOperationLogSourcePage(operURL)
}

func detectOperationLogFailureCategory(status int, errorMsg string, jsonResult string) string {
	return middleware.DetectOperationLogFailureCategory(status, errorMsg, jsonResult)
}

func filterOperationLogs(rows []middleware.SystemLogOper, query *OperationLogQuery) []middleware.SystemLogOper {
	if query == nil {
		return rows
	}
	filter := operationLogFilter{
		sourceDomain:    strings.TrimSpace(query.SourceDomain),
		sourcePage:      strings.TrimSpace(query.SourcePage),
		failureCategory: strings.TrimSpace(query.FailureCategory),
	}
	if filter.empty() {
		return rows
	}
	filtered := make([]middleware.SystemLogOper, 0, len(rows))
	for _, row := range rows {
		if filter.matches(row) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

type operationLogFilter struct {
	sourceDomain    string
	sourcePage      string
	failureCategory string
}

func (f operationLogFilter) empty() bool {
	return f.sourceDomain == "" && f.sourcePage == "" && f.failureCategory == ""
}

func (f operationLogFilter) matches(row middleware.SystemLogOper) bool {
	return matchesOperationLogFilterValue(f.sourceDomain, row.SourceDomain, func() string {
		return detectOperationLogSourceDomain(row.OperURL)
	}) &&
		matchesOperationLogFilterValue(f.sourcePage, row.SourcePage, func() string {
			return detectOperationLogSourcePage(row.OperURL)
		}) &&
		matchesOperationLogFilterValue(f.failureCategory, row.FailureCategory, func() string {
			return detectOperationLogFailureCategory(row.Status, row.ErrorMsg, row.JsonResult)
		})
}

func matchesOperationLogFilterValue(expected, stored string, detect func() string) bool {
	if expected == "" {
		return true
	}
	actual := strings.TrimSpace(stored)
	if actual == "" {
		actual = detect()
	}
	return actual == expected
}
