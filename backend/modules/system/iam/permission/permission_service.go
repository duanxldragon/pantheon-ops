package iam

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"pantheon-platform/backend/pkg/database"
	"pantheon-platform/backend/pkg/impexp"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PermissionService struct {
	db *gorm.DB
}

func NewPermissionService(db *gorm.DB) *PermissionService {
	return &PermissionService{db: db}
}

func (s *PermissionService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	if err := s.db.AutoMigrate(&PermissionWorkbenchRemediationEvent{}); err != nil {
		return err
	}
	if err := s.db.AutoMigrate(&PermissionRoleDataScopePolicy{}); err != nil {
		return err
	}
	if !s.db.Migrator().HasTable("system_role") || !s.db.Migrator().HasTable("casbin_rule") {
		return nil
	}
	if err := s.db.Exec(`
DELETE FROM casbin_rule
WHERE ptype = ?
  AND NOT EXISTS (
    SELECT 1
    FROM system_role
    WHERE system_role.role_key = casbin_rule.v0
      AND system_role.deleted_at IS NULL
  )
`, "p").Error; err != nil {
		return err
	}
	return reloadPermissionPolicies()
}

func (s *PermissionService) ListPolicies(query *PermissionPolicyQuery) (*PermissionPolicyPageResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	var policies []database.CasbinRule
	db := s.db.Model(&database.CasbinRule{}).Where("ptype = ?", "p")
	page, pageSize := normalizePermissionPageQuery(query)
	if query != nil {
		if strings.TrimSpace(query.RoleKey) != "" {
			db = db.Where("v0 LIKE ?", "%"+strings.TrimSpace(query.RoleKey)+"%")
		}
		if strings.TrimSpace(query.Path) != "" {
			db = db.Where("v1 LIKE ?", "%"+strings.TrimSpace(query.Path)+"%")
		}
		if strings.TrimSpace(query.Method) != "" {
			db = db.Where("v2 = ?", normalizePolicyMethod(query.Method))
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	if err := db.
		Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}, Desc: true}).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&policies).Error; err != nil {
		return nil, err
	}

	items := make([]PermissionPolicyResp, 0, len(policies))
	for _, item := range policies {
		items = append(items, PermissionPolicyResp{
			ID:      item.ID,
			PType:   item.PType,
			RoleKey: item.V0,
			Path:    item.V1,
			Method:  item.V2,
		})
	}

	return &PermissionPolicyPageResp{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *PermissionService) CreatePolicy(req *PermissionPolicyCreateReq) (*PermissionPolicyResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	roleKey, path, method, err := s.validatePolicyPayload(0, req.RoleKey, req.Path, req.Method)
	if err != nil {
		return nil, err
	}

	policy := database.CasbinRule{
		PType: "p",
		V0:    roleKey,
		V1:    path,
		V2:    method,
	}
	if err := s.db.Create(&policy).Error; err != nil {
		return nil, errors.New("permission.policy.exists")
	}
	if err := reloadPermissionPolicies(); err != nil {
		return nil, err
	}

	return &PermissionPolicyResp{
		ID:      policy.ID,
		PType:   policy.PType,
		RoleKey: policy.V0,
		Path:    policy.V1,
		Method:  policy.V2,
	}, nil
}

func (s *PermissionService) UpdatePolicy(policyID uint64, req *PermissionPolicyUpdateReq) (*PermissionPolicyResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	var policy database.CasbinRule
	if err := s.db.First(&policy, policyID).Error; err != nil {
		return nil, err
	}
	roleKey, path, method, err := s.validatePolicyPayload(policyID, req.RoleKey, req.Path, req.Method)
	if err != nil {
		return nil, err
	}

	policy.PType = "p"
	policy.V0 = roleKey
	policy.V1 = path
	policy.V2 = method
	policy.V3 = ""
	policy.V4 = ""
	policy.V5 = ""
	if err := s.db.Save(&policy).Error; err != nil {
		return nil, errors.New("permission.policy.exists")
	}
	if err := reloadPermissionPolicies(); err != nil {
		return nil, err
	}

	return &PermissionPolicyResp{
		ID:      policy.ID,
		PType:   policy.PType,
		RoleKey: policy.V0,
		Path:    policy.V1,
		Method:  policy.V2,
	}, nil
}

func (s *PermissionService) DeletePolicy(policyID uint64) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	if err := s.db.Delete(&database.CasbinRule{}, policyID).Error; err != nil {
		return err
	}
	return reloadPermissionPolicies()
}

func (s *PermissionService) ExportPolicies(query *PermissionPolicyQuery) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	policies, err := s.listPoliciesForExport(query)
	if err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(policies))
	for _, item := range policies {
		rows = append(rows, []string{item.V0, item.V1, item.V2})
	}

	return &impexp.CSVFile{
		Filename: "system-permission-export.csv",
		Headers:  []string{"roleKey", "path", "method"},
		Rows:     rows,
	}, nil
}

func (s *PermissionService) BuildImportTemplate() *impexp.CSVFile {
	return &impexp.CSVFile{
		Filename: "system-permission-import-template.csv",
		Headers:  []string{"roleKey", "path", "method"},
		Rows: [][]string{
			{"#说明：保留第一行表头；roleKey 必须是已存在角色标识；method 支持 GET/POST/PUT/PATCH/DELETE；该模板只导入 Casbin 接口策略，不导入菜单/按钮授权。", "", ""},
			{"#admin", "/api/v1/system/user/list", "GET"},
		},
	}
}

func (s *PermissionService) ExportWorkbench(query *PermissionWorkbenchQuery) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	workbench, err := s.GetWorkbench(query)
	if err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(workbench.Roles))
	for _, role := range workbench.Roles {
		coverage := "complete"
		switch {
		case role.HasPageGap && role.HasAPIGap:
			coverage = "page-gap,api-gap"
		case role.HasPageGap:
			coverage = "page-gap"
		case role.HasAPIGap:
			coverage = "api-gap"
		}
		unknownKeys := make([]string, 0, len(role.UnknownPermissions))
		for _, item := range role.UnknownPermissions {
			if strings.TrimSpace(item.Key) != "" {
				unknownKeys = append(unknownKeys, strings.TrimSpace(item.Key))
			}
		}

		rows = append(rows, []string{
			role.RoleName,
			role.RoleKey,
			strconv.Itoa(role.Status),
			strconv.Itoa(role.MenuCount),
			strconv.Itoa(role.PagePermissionCount),
			strconv.Itoa(role.ActionPermissionCount),
			strconv.Itoa(role.APIPolicyCount),
			strconv.Itoa(role.UnknownPermissionCount),
			boolToCSVValue(role.HasPageGap),
			boolToCSVValue(role.HasAPIGap),
			coverage,
			strings.Join(unknownKeys, "|"),
		})
	}

	return &impexp.CSVFile{
		Filename: "system-permission-workbench-export.csv",
		Headers: []string{
			"roleName",
			"roleKey",
			"status",
			"menuCount",
			"pagePermissionCount",
			"actionPermissionCount",
			"apiPolicyCount",
			"unknownPermissionCount",
			"hasPageGap",
			"hasApiGap",
			"coverage",
			"unknownPermissionKeys",
		},
		Rows: rows,
	}, nil
}

func (s *PermissionService) RemediateWorkbenchPolicies(req *PermissionWorkbenchRemediateReq) (*PermissionWorkbenchRemediateResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	roleKey := strings.TrimSpace(req.RoleKey)
	if roleKey == "" {
		return nil, errors.New("param.invalid")
	}
	if err := s.ensureRoleKeyExists(roleKey); err != nil {
		return nil, err
	}

	workbench, err := s.GetWorkbench(&PermissionWorkbenchQuery{RoleKey: roleKey})
	if err != nil {
		return nil, err
	}
	var role *PermissionWorkbenchRoleResp
	for index := range workbench.Roles {
		if workbench.Roles[index].RoleKey == roleKey {
			role = &workbench.Roles[index]
			break
		}
	}
	if role == nil {
		return nil, errors.New("permission.role.invalid")
	}
	resp := &PermissionWorkbenchRemediateResp{
		RoleKey:         roleKey,
		SkippedCount:    role.RequiredAPIPolicyCount - role.MissingAPIPolicyCount,
		CreatedPolicies: []PermissionWorkbenchAPIPolicyResp{},
	}
	if len(role.MissingAPIPolicies) == 0 {
		_ = s.recordWorkbenchRemediation(roleKey, "api-gap", "", "complete", "complete", "noop", 0, resp.SkippedCount)
		return resp, nil
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range role.MissingAPIPolicies {
			policy := database.CasbinRule{
				PType: "p",
				V0:    roleKey,
				V1:    strings.TrimSpace(item.Path),
				V2:    normalizePolicyMethod(item.Method),
			}
			if err := tx.Create(&policy).Error; err != nil {
				return err
			}
			resp.CreatedPolicies = append(resp.CreatedPolicies, PermissionWorkbenchAPIPolicyResp{
				ID:     policy.ID,
				Path:   policy.V1,
				Method: policy.V2,
			})
		}
		return nil
	}); err != nil {
		return nil, err
	}

	resp.CreatedCount = len(resp.CreatedPolicies)
	if resp.CreatedCount > 0 {
		if err := reloadPermissionPolicies(); err != nil {
			return nil, err
		}
	}
	if err := s.recordWorkbenchRemediation(
		roleKey,
		"api-gap",
		joinWorkbenchPolicyKeys(role.MissingAPIPolicies),
		"api-gap",
		"complete",
		"remediated",
		resp.CreatedCount,
		resp.SkippedCount,
	); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *PermissionService) ListWorkbenchRemediationEvents(query *PermissionWorkbenchRemediationQuery) ([]PermissionWorkbenchRemediationResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	limit := 20
	if query != nil && query.Limit > 0 {
		limit = query.Limit
	}
	if limit > 100 {
		limit = 100
	}

	var events []PermissionWorkbenchRemediationEvent
	db := s.db.Model(&PermissionWorkbenchRemediationEvent{})
	if query != nil && strings.TrimSpace(query.RoleKey) != "" {
		db = db.Where("role_key = ?", strings.TrimSpace(query.RoleKey))
	}
	if err := db.Order("id asc").Limit(limit).Find(&events).Error; err != nil {
		return nil, err
	}

	result := make([]PermissionWorkbenchRemediationResp, 0, len(events))
	for _, item := range events {
		result = append(result, PermissionWorkbenchRemediationResp{
			ID:           item.ID,
			RoleKey:      item.RoleKey,
			IssueType:    item.IssueType,
			IssueKey:     item.IssueKey,
			BeforeState:  item.BeforeState,
			AfterState:   item.AfterState,
			Action:       item.Action,
			CreatedCount: item.CreatedCount,
			SkippedCount: item.SkippedCount,
			CreatedAt:    item.CreatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

func (s *PermissionService) recordWorkbenchRemediation(roleKey string, issueType string, issueKey string, beforeState string, afterState string, action string, createdCount int, skippedCount int) error {
	if s.db == nil || !s.db.Migrator().HasTable(&PermissionWorkbenchRemediationEvent{}) {
		return nil
	}
	return s.db.Create(&PermissionWorkbenchRemediationEvent{
		RoleKey:      strings.TrimSpace(roleKey),
		IssueType:    strings.TrimSpace(issueType),
		IssueKey:     strings.TrimSpace(issueKey),
		BeforeState:  strings.TrimSpace(beforeState),
		AfterState:   strings.TrimSpace(afterState),
		Action:       strings.TrimSpace(action),
		CreatedCount: createdCount,
		SkippedCount: skippedCount,
	}).Error
}

func joinWorkbenchPolicyKeys(policies []PermissionWorkbenchAPIPolicyResp) string {
	keys := make([]string, 0, len(policies))
	for _, item := range policies {
		method := normalizePolicyMethod(item.Method)
		path := strings.TrimSpace(item.Path)
		if method == "" || path == "" {
			continue
		}
		keys = append(keys, method+" "+path)
	}
	sort.Strings(keys)
	return strings.Join(keys, "|")
}

func (s *PermissionService) ImportPolicies(records [][]string) (*impexp.ImportResult, error) {
	result := &impexp.ImportResult{
		Applied: false,
		Errors:  []impexp.ImportError{},
	}
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if len(records) == 0 {
		impexp.AppendImportError(result, 0, "file", "import.file.empty")
		return result, nil
	}

	headerIndex := make(map[string]int, len(records[0]))
	for index, header := range records[0] {
		headerIndex[strings.TrimSpace(header)] = index
	}
	requiredHeaders := []string{"roleKey", "path", "method"}
	for _, header := range requiredHeaders {
		if _, ok := headerIndex[header]; !ok {
			impexp.AppendImportError(result, 0, header, "import.header.missing")
		}
	}
	if result.Failed > 0 {
		return result, nil
	}

	type importRow struct {
		RowNumber int
		RoleKey   string
		Path      string
		Method    string
	}

	rows := make([]importRow, 0, len(records)-1)
	seenKeys := make(map[string]int, len(records)-1)
	for rowIndex := 1; rowIndex < len(records); rowIndex++ {
		record := records[rowIndex]
		if impexp.IsCSVRecordEmpty(record) {
			continue
		}

		rowNumber := rowIndex + 1
		roleKey := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "roleKey"))
		path := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "path"))
		method := normalizePolicyMethod(impexp.ReadCSVField(record, headerIndex, "method"))
		if roleKey == "" {
			impexp.AppendImportError(result, rowNumber, "roleKey", "permission.role.invalid")
		}
		if path == "" {
			impexp.AppendImportError(result, rowNumber, "path", "permission.path.required")
		}
		if method == "" {
			impexp.AppendImportError(result, rowNumber, "method", "permission.method.invalid")
		}

		compositeKey := fmt.Sprintf("%s|%s|%s", roleKey, path, method)
		if firstRow, ok := seenKeys[compositeKey]; ok {
			impexp.AppendImportError(result, rowNumber, "roleKey", fmt.Sprintf("import.duplicate.row.%d", firstRow))
		} else {
			seenKeys[compositeKey] = rowNumber
		}
		rows = append(rows, importRow{
			RowNumber: rowNumber,
			RoleKey:   roleKey,
			Path:      path,
			Method:    method,
		})
	}

	if result.Failed > 0 {
		return result, nil
	}

	roleKeys := make([]string, 0, len(rows))
	for _, row := range rows {
		roleKeys = append(roleKeys, row.RoleKey)
	}
	var existingRoleCount int64
	if err := s.db.Table("system_role").Where("role_key IN ? AND deleted_at IS NULL", roleKeys).Count(&existingRoleCount).Error; err != nil {
		return nil, err
	}
	normalizedRoleKeys := impexp.SplitPipeValues(strings.Join(roleKeys, "|"))
	if existingRoleCount != int64(len(normalizedRoleKeys)) {
		for _, row := range rows {
			if err := s.ensureRoleKeyExists(row.RoleKey); err != nil {
				impexp.AppendImportError(result, row.RowNumber, "roleKey", err.Error())
			}
		}
		return result, nil
	}

	policies, err := s.listPoliciesForExport(nil)
	if err != nil {
		return nil, err
	}
	existingByKey := make(map[string]database.CasbinRule, len(policies))
	for _, policy := range policies {
		existingByKey[fmt.Sprintf("%s|%s|%s", policy.V0, policy.V1, policy.V2)] = policy
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			compositeKey := fmt.Sprintf("%s|%s|%s", row.RoleKey, row.Path, row.Method)
			if _, ok := existingByKey[compositeKey]; ok {
				result.Updated++
				continue
			}
			policy := database.CasbinRule{
				PType: "p",
				V0:    row.RoleKey,
				V1:    row.Path,
				V2:    row.Method,
			}
			if err := tx.Create(&policy).Error; err != nil {
				return err
			}
			result.Created++
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if err := reloadPermissionPolicies(); err != nil {
		return nil, err
	}

	result.Applied = true
	return result, nil
}

func (s *PermissionService) listPoliciesForExport(query *PermissionPolicyQuery) ([]database.CasbinRule, error) {
	var policies []database.CasbinRule
	db := s.db.Model(&database.CasbinRule{}).Where("ptype = ?", "p")
	if query != nil {
		if strings.TrimSpace(query.RoleKey) != "" {
			db = db.Where("v0 LIKE ?", "%"+strings.TrimSpace(query.RoleKey)+"%")
		}
		if strings.TrimSpace(query.Path) != "" {
			db = db.Where("v1 LIKE ?", "%"+strings.TrimSpace(query.Path)+"%")
		}
		if strings.TrimSpace(query.Method) != "" {
			db = db.Where("v2 = ?", normalizePolicyMethod(query.Method))
		}
	}

	if err := db.
		Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}, Desc: false}).
		Find(&policies).Error; err != nil {
		return nil, err
	}
	return policies, nil
}

func boolToCSVValue(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func (s *PermissionService) validatePolicyPayload(policyID uint64, roleKey string, path string, method string) (string, string, string, error) {
	roleKey = strings.TrimSpace(roleKey)
	path = strings.TrimSpace(path)
	method = normalizePolicyMethod(method)
	if roleKey == "" || path == "" || method == "" {
		return "", "", "", errors.New("param.invalid")
	}
	if err := s.ensureRoleKeyExists(roleKey); err != nil {
		return "", "", "", err
	}
	if err := s.ensurePolicyUnique(policyID, roleKey, path, method); err != nil {
		return "", "", "", err
	}
	return roleKey, path, method, nil
}

func (s *PermissionService) ensureRoleKeyExists(roleKey string) error {
	var count int64
	if err := s.db.Table("system_role").Where("role_key = ? AND deleted_at IS NULL", roleKey).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("permission.role.invalid")
	}
	return nil
}

func (s *PermissionService) ensurePolicyUnique(policyID uint64, roleKey string, path string, method string) error {
	var count int64
	db := s.db.Model(&database.CasbinRule{}).Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", roleKey, path, method)
	if policyID > 0 {
		db = db.Where("id <> ?", policyID)
	}
	if err := db.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("permission.policy.exists")
	}
	return nil
}

func normalizePermissionPageQuery(query *PermissionPolicyQuery) (int, int) {
	page := 1
	pageSize := 10
	if query == nil {
		return page, pageSize
	}
	if query.Page > 0 {
		page = query.Page
	}
	if query.PageSize > 0 {
		pageSize = query.PageSize
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func normalizePolicyMethod(method string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		return method
	default:
		return ""
	}
}

func reloadPermissionPolicies() error {
	if database.Enforcer == nil {
		return nil
	}
	return database.Enforcer.LoadPolicy()
}
