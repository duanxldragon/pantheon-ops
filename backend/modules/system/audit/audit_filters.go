package system

import (
	"strings"

	"pantheon-platform/backend/internal/middleware"
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
	sourceDomain := strings.TrimSpace(query.SourceDomain)
	sourcePage := strings.TrimSpace(query.SourcePage)
	failureCategory := strings.TrimSpace(query.FailureCategory)
	if sourceDomain == "" && sourcePage == "" && failureCategory == "" {
		return rows
	}
	filtered := make([]middleware.SystemLogOper, 0, len(rows))
	for _, row := range rows {
		rowSourceDomain := strings.TrimSpace(row.SourceDomain)
		if rowSourceDomain == "" {
			rowSourceDomain = detectOperationLogSourceDomain(row.OperURL)
		}
		if sourceDomain != "" && rowSourceDomain != sourceDomain {
			continue
		}
		rowSourcePage := strings.TrimSpace(row.SourcePage)
		if rowSourcePage == "" {
			rowSourcePage = detectOperationLogSourcePage(row.OperURL)
		}
		if sourcePage != "" && rowSourcePage != sourcePage {
			continue
		}
		rowFailureCategory := strings.TrimSpace(row.FailureCategory)
		if rowFailureCategory == "" {
			rowFailureCategory = detectOperationLogFailureCategory(row.Status, row.ErrorMsg, row.JsonResult)
		}
		if failureCategory != "" && rowFailureCategory != failureCategory {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered
}
