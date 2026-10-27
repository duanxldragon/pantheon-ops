//nolint:revive // Permission service intentionally exposes a broad administrative facade.
package permission

import (
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/impexp"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const permissionPtypeClause = "ptype = ?"

// permissionRoleKeyLikeClause matches the casbin subject (role key) column.
// Extracted as a constant because it is repeated across policy query builders
// (SonarCloud go:S1192).
const permissionRoleKeyLikeClause = "v0 LIKE ?"

// permissionTenantClause scopes a query to a single tenant. Repeated across the
// service and workbench query builders (SonarCloud go:S1192).
const permissionTenantClause = "tenant_id = ?"

const (
	errPermissionPolicyExists  = "permission.policy.exists"
	errPermissionRoleInvalid   = "permission.role.invalid"
	errPermissionTenantInvalid = "permission.tenant.invalid"
)

const (
	protectedManagementPolicyPrefixPermission = "/api/v1/system/permission"
	protectedManagementPolicyPrefixRole       = "/api/v1/system/role"
)

const (
	workbenchCoverageComplete = "complete"
	workbenchCoveragePageGap  = "page-gap"
	workbenchCoverageAPIGap   = "api-gap"
)

const (
	workbenchIntegrityUnknown = "unknown"
	workbenchIntegrityClean   = "clean"
)

type PermissionService struct {
	db *gorm.DB
	// tenantCtx is the per-request tenant context (contract §2.3/§3.3). Set from
	// the Gin context by the handler layer via WithTenantContext; nil keeps the
	// historical global behaviour for platform-internal callers, compat binds
	// the platform-global scope. Without it the data-scope console reads and
	// writes every tenant's policy rows, which is how a tenant administrator
	// could list and overwrite another tenant's role data scope (F7/F8/F9).
	tenantCtx *tenant.Context
}

// WithTenantContext returns a shallow view of the service bound to a request
// tenant context. Only a nil context returns the receiver unchanged.
func (s *PermissionService) WithTenantContext(ctx *tenant.Context) *PermissionService {
	if ctx == nil {
		return s
	}
	bound := *s
	bound.tenantCtx = ctx
	return &bound
}

// tenantOwnerID is the tenant a policy row belongs to (write path: from the
// resolved request context, never from the request body — contract §3.3).
// Compat/platform => 0, which is the platform/global scope.
func (s *PermissionService) tenantOwnerID() uint64 {
	if s.tenantCtx == nil || !s.tenantCtx.IsMulti() {
		return 0
	}
	return s.tenantCtx.TenantID
}

// isPlatformScope reports whether the caller operates on the platform/global
// population (tenant 0) rather than inside a tenant. Platform scope is the only
// scope allowed to read or write rows that are not pinned to a tenant.
func (s *PermissionService) isPlatformScope() bool {
	return s.tenantOwnerID() == tenant.PlatformGlobalTenantID
}

// policySubjectForWrite resolves the stored Casbin subject for a policy write
// from the caller's resolved scope. A tenant subject may only write inside its
// own domain: a body `tenantId` that disagrees is rejected, never honoured
// (contract §2.3.1 rule 2 — the tenant of a write comes from the context, not
// from the request body). Platform scope keeps the explicit-target behaviour
// because the platform console legitimately manages tenant namespaces.
func (s *PermissionService) policySubjectForWrite(roleKey string, requestedTenantID uint64) (string, error) {
	targetTenant := requestedTenantID
	if !s.isPlatformScope() {
		if targetTenant != tenant.PlatformGlobalTenantID && targetTenant != s.tenantOwnerID() {
			return "", common.NewForbidden("tenant.forbidden")
		}
		targetTenant = s.tenantOwnerID()
	}
	return s.resolvePolicySubject(roleKey, targetTenant)
}

// policyInTenantScope reports whether a stored policy row belongs to the
// caller's tenant domain. A tenant caller must not be able to update or delete
// a policy by id just because the id exists (acceptance F8 class).
func (s *PermissionService) policyInTenantScope(policy *database.CasbinRule) bool {
	if s.isPlatformScope() {
		return true
	}
	suffix := fmt.Sprintf("@tenant:%d", s.tenantOwnerID())
	subject := strings.TrimSpace(policy.V0)
	return strings.HasPrefix(subject, "role:") && strings.HasSuffix(subject, suffix)
}

// applyPolicyTenantScope confines policy reads to the caller's tenant domain.
// Global policy rows are platform-global state even when they carry a role key
// that also exists in a tenant, so a tenant subject never sees them.
func (s *PermissionService) applyPolicyTenantScope(db *gorm.DB) *gorm.DB {
	if s.isPlatformScope() {
		return db
	}
	return db.Where(permissionRoleKeyLikeClause, "%@tenant:"+strconv.FormatUint(s.tenantOwnerID(), 10))
}

type workbenchRemediationRecord struct {
	roleKey      string
	issueType    string
	issueKey     string
	beforeState  string
	afterState   string
	action       string
	createdCount int
	skippedCount int
}

func NewPermissionService(db *gorm.DB) *PermissionService {
	return &PermissionService{db: db}
}

func (s *PermissionService) Migrate() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	if err := s.db.AutoMigrate(&PermissionWorkbenchRemediationEvent{}); err != nil {
		return err
	}
	if err := s.db.AutoMigrate(&PermissionRoleDataScopePolicy{}); err != nil {
		return err
	}
	return s.Bootstrap()
}

func (s *PermissionService) Bootstrap() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	if !s.db.Migrator().HasTable("system_role") || !s.db.Migrator().HasTable("casbin_rule") {
		return nil
	}
	// Purge only policies whose subject no longer names a live role. A tenant
	// role's policies live under role:<key>@tenant:<id> (contract §4), so the
	// bare-key comparison alone treated EVERY tenant policy as an orphan and
	// deleted it on each boot — the tenant module then re-granted only the
	// default wildcard, silently dropping every policy a tenant had configured.
	if err := s.db.Exec(`
DELETE FROM casbin_rule
WHERE `+permissionPtypeClause+`
  AND NOT EXISTS (
    SELECT 1
    FROM system_role
    WHERE system_role.deleted_at IS NULL
      AND (
        system_role.role_key = casbin_rule.v0
        OR (system_role.tenant_id <> 0
            AND casbin_rule.v0 = CONCAT('role:', system_role.role_key, '@tenant:', system_role.tenant_id))
      )
  )
`, "p").Error; err != nil {
		return err
	}
	return reloadPermissionPolicies()
}

func (s *PermissionService) ListPolicies(query *PermissionPolicyQuery) (*PermissionPolicyPageResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	var policies []database.CasbinRule
	db := s.applyPolicyTenantScope(s.db.Model(&database.CasbinRule{}).Where(permissionPtypeClause, "p"))
	page, pageSize := normalizePermissionPageQuery(query)
	if query != nil {
		if strings.TrimSpace(query.Keyword) != "" {
			keyword := "%" + common.EscapeLikePattern(strings.TrimSpace(query.Keyword)) + "%"
			db = db.Where("v0 LIKE ? OR v1 LIKE ?", keyword, keyword)
		}
		if strings.TrimSpace(query.RoleKey) != "" {
			db = db.Where(permissionRoleKeyLikeClause, "%"+strings.TrimSpace(query.RoleKey)+"%")
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

	sortColumn, sortDesc := normalizePermissionPolicySort(query)
	orderedDB := db.Order(clause.OrderByColumn{Column: clause.Column{Name: sortColumn}, Desc: sortDesc})
	if sortColumn != "id" {
		orderedDB = orderedDB.Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}, Desc: true})
	}
	if err := orderedDB.
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

func (s *PermissionService) CreatePolicy(operatorRoleKeys []string, req *PermissionPolicyCreateReq) (*PermissionPolicyResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	roleKey := strings.TrimSpace(req.RoleKey)
	subject, err := s.policySubjectForWrite(roleKey, req.TenantId)
	if err != nil {
		return nil, err
	}
	path, method, err := s.validatePolicyPayload(0, operatorRoleKeys, roleKey, subject, req.Path, req.Method)
	if err != nil {
		return nil, err
	}

	policy := database.CasbinRule{
		PType: "p",
		V0:    subject,
		V1:    path,
		V2:    method,
	}
	if err := s.db.Create(&policy).Error; err != nil {
		return nil, common.NewConflict(errPermissionPolicyExists)
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

func (s *PermissionService) UpdatePolicy(operatorRoleKeys []string, policyID uint64, req *PermissionPolicyUpdateReq) (*PermissionPolicyResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	var policy database.CasbinRule
	if err := s.db.First(&policy, policyID).Error; err != nil {
		return nil, err
	}
	if !s.policyInTenantScope(&policy) {
		return nil, common.NewNotFound("permission.policy.not_found")
	}
	roleKey := strings.TrimSpace(req.RoleKey)
	subject, err := s.policySubjectForWrite(roleKey, req.TenantId)
	if err != nil {
		return nil, err
	}
	path, method, err := s.validatePolicyPayload(policyID, operatorRoleKeys, roleKey, subject, req.Path, req.Method)
	if err != nil {
		return nil, err
	}

	policy.PType = "p"
	policy.V0 = subject
	policy.V1 = path
	policy.V2 = method
	policy.V3 = ""
	policy.V4 = ""
	policy.V5 = ""
	if err := s.db.Save(&policy).Error; err != nil {
		return nil, common.NewConflict(errPermissionPolicyExists)
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

func (s *PermissionService) DeletePolicy(operatorRoleKeys []string, policyID uint64) error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	var policy database.CasbinRule
	if err := s.db.First(&policy, policyID).Error; err != nil {
		return err
	}
	if !s.policyInTenantScope(&policy) {
		return common.NewNotFound("permission.policy.not_found")
	}
	if err := ensurePolicyWriteAllowed(operatorRoleKeys, policy.V1); err != nil {
		return err
	}
	if err := s.db.Delete(&database.CasbinRule{}, policyID).Error; err != nil {
		return err
	}
	return reloadPermissionPolicies()
}

func (s *PermissionService) ExportPolicies(query *PermissionPolicyQuery) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
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
		return nil, common.ErrDatabaseNotInitialized
	}

	workbench, err := s.GetWorkbench(query)
	if err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(workbench.Roles))
	for _, role := range workbench.Roles {
		coverage := workbenchCoverageComplete
		switch {
		case role.HasPageGap && role.HasAPIGap:
			coverage = workbenchCoveragePageGap + "," + workbenchCoverageAPIGap
		case role.HasPageGap:
			coverage = workbenchCoveragePageGap
		case role.HasAPIGap:
			coverage = workbenchCoverageAPIGap
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
		return nil, common.ErrDatabaseNotInitialized
	}
	roleKey := strings.TrimSpace(req.RoleKey)
	if roleKey == "" {
		return nil, common.NewBadRequest("param.invalid")
	}
	if err := s.ensureRoleKeyExists(roleKey); err != nil {
		return nil, err
	}
	// The补齐 writes must land in the caller's namespace. Before this resolved
	// from the bare role key, a tenant administrator's remediation created
	// GLOBAL policies that every tenant's same-named role inherited.
	subject, err := s.policySubjectForWrite(roleKey, tenant.PlatformGlobalTenantID)
	if err != nil {
		return nil, err
	}

	// Fetch only the missing API policies for the target role,
	// avoiding the full workbench computation overhead.
	role, err := s.getRoleMissingAPIPolicies(roleKey)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, common.NewBadRequest(errPermissionRoleInvalid)
	}
	resp := &PermissionWorkbenchRemediateResp{
		RoleKey:         roleKey,
		SkippedCount:    role.RequiredAPIPolicyCount - role.MissingAPIPolicyCount,
		CreatedPolicies: []PermissionWorkbenchAPIPolicyResp{},
	}
	if len(role.MissingAPIPolicies) == 0 {
		_ = s.recordWorkbenchRemediation(workbenchRemediationRecord{
			roleKey:      roleKey,
			issueType:    workbenchCoverageAPIGap,
			beforeState:  "complete",
			afterState:   "complete",
			action:       "noop",
			skippedCount: resp.SkippedCount,
		})
		return resp, nil
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		return createMissingWorkbenchPolicies(tx, subject, role.MissingAPIPolicies, resp)
	}); err != nil {
		return nil, err
	}

	resp.CreatedCount = len(resp.CreatedPolicies)
	if resp.CreatedCount > 0 {
		if err := reloadPermissionPolicies(); err != nil {
			return nil, err
		}
	}
	if err := s.recordWorkbenchRemediation(workbenchRemediationRecord{
		roleKey:      roleKey,
		issueType:    workbenchCoverageAPIGap,
		issueKey:     joinWorkbenchPolicyKeys(role.MissingAPIPolicies),
		beforeState:  workbenchCoverageAPIGap,
		afterState:   "complete",
		action:       "remediated",
		createdCount: resp.CreatedCount,
		skippedCount: resp.SkippedCount,
	}); err != nil {
		return nil, err
	}
	return resp, nil
}

func createMissingWorkbenchPolicies(tx *gorm.DB, subject string, missing []PermissionWorkbenchAPIPolicyResp, resp *PermissionWorkbenchRemediateResp) error {
	for _, item := range missing {
		policy := database.CasbinRule{
			PType: "p",
			V0:    subject,
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
}

func (s *PermissionService) ListWorkbenchRemediationEvents(query *PermissionWorkbenchRemediationQuery) ([]PermissionWorkbenchRemediationResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
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
	if !s.isPlatformScope() {
		db = db.Where(permissionTenantClause, s.tenantOwnerID())
	}
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

func (s *PermissionService) recordWorkbenchRemediation(record workbenchRemediationRecord) error {
	if s.db == nil || !s.db.Migrator().HasTable(&PermissionWorkbenchRemediationEvent{}) {
		return nil
	}
	return s.db.Create(&PermissionWorkbenchRemediationEvent{
		TenantID:     s.tenantOwnerID(),
		RoleKey:      strings.TrimSpace(record.roleKey),
		IssueType:    strings.TrimSpace(record.issueType),
		IssueKey:     strings.TrimSpace(record.issueKey),
		BeforeState:  strings.TrimSpace(record.beforeState),
		AfterState:   strings.TrimSpace(record.afterState),
		Action:       strings.TrimSpace(record.action),
		CreatedCount: record.createdCount,
		SkippedCount: record.skippedCount,
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

func (s *PermissionService) ImportPolicies(operatorRoleKeys []string, records [][]string) (*impexp.ImportResult, error) {
	result := &impexp.ImportResult{
		Applied: false,
		Errors:  []impexp.ImportError{},
	}
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	if len(records) == 0 {
		impexp.AppendImportError(result, 0, "file", "import.file.empty")
		return result, nil
	}

	// Step 1: Validate header
	headerIndex, err := validateImportHeader(records, result)
	if err != nil {
		return result, nil
	}

	// Step 2: Validate and deduplicate rows
	rows, err := validateImportRows(records, headerIndex, result)
	if err != nil {
		return result, nil
	}

	// Step 3: Reject protected management policies for non-admin operators.
	if err := validateImportPolicyWriteScope(operatorRoleKeys, rows); err != nil {
		return nil, err
	}

	// Step 4: Validate role keys exist in the target namespace
	if err := s.validateImportRoleKeys(rows, result); err != nil {
		return result, nil
	}

	// Step 5: Resolve the stored subject for each role key from the request
	// scope (tenant scope writes role:<key>@tenant:<id>; the request body has
	// no say in it) and load existing policies for dedup.
	subjects, err := s.importPolicySubjects(rows)
	if err != nil {
		return nil, err
	}
	existingByKey, err := loadExistingPolicyMap(s.db, subjects, s.isPlatformScope(), s.tenantOwnerID())
	if err != nil {
		return nil, err
	}

	// Step 6: Write new policies in transaction
	if err := writeImportPolicies(s.db, rows, subjects, existingByKey, result); err != nil {
		return nil, err
	}

	// Step 7: Reload enforcer
	if result.Created > 0 {
		if err := reloadPermissionPolicies(); err != nil {
			return nil, err
		}
	}

	result.Applied = true
	return result, nil
}

// policyImportRow represents a validated import row.
type policyImportRow struct {
	RowNumber int
	RoleKey   string
	Path      string
	Method    string
}

// validateImportHeader checks that required headers are present.
func validateImportHeader(records [][]string, result *impexp.ImportResult) (map[string]int, error) {
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
		return nil, common.NewBadRequest("import.header.missing")
	}
	return headerIndex, nil
}

// validateImportRows parses, validates, and deduplicates CSV rows.
func validateImportRows(records [][]string, headerIndex map[string]int, result *impexp.ImportResult) ([]policyImportRow, error) {
	rows := make([]policyImportRow, 0, len(records)-1)
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
			impexp.AppendImportError(result, rowNumber, "roleKey", errPermissionRoleInvalid)
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
		rows = append(rows, policyImportRow{
			RowNumber: rowNumber,
			RoleKey:   roleKey,
			Path:      path,
			Method:    method,
		})
	}
	if result.Failed > 0 {
		return nil, common.NewBadRequest("import.row.invalid")
	}
	return rows, nil
}

// importPolicySubjects resolves, once per imported role key, the subject the
// row will be stored under. Tenant scope forces the caller's domain, so an
// imported row can never become a global (platform) policy.
func (s *PermissionService) importPolicySubjects(rows []policyImportRow) (map[string]string, error) {
	subjects := make(map[string]string, len(rows))
	for _, row := range rows {
		if _, ok := subjects[row.RoleKey]; ok {
			continue
		}
		subject, err := s.policySubjectForWrite(row.RoleKey, tenant.PlatformGlobalTenantID)
		if err != nil {
			return nil, err
		}
		subjects[row.RoleKey] = subject
	}
	return subjects, nil
}

// validateImportRoleKeys checks that all role keys in the import exist in the
// namespace the rows will be written to: the caller's tenant in tenant scope,
// the platform-global namespace otherwise. Checking by bare key across every
// tenant let a tenant point a policy at somebody else's role (acceptance F8).
func (s *PermissionService) validateImportRoleKeys(rows []policyImportRow, result *impexp.ImportResult) error {
	roleKeys := make([]string, 0, len(rows))
	for _, row := range rows {
		roleKeys = append(roleKeys, row.RoleKey)
	}
	db := s.db.Table("system_role").Where("role_key IN ? AND deleted_at IS NULL", roleKeys)
	if !s.isPlatformScope() {
		db = db.Where(permissionTenantClause, s.tenantOwnerID())
	} else {
		db = db.Where(permissionTenantClause, tenant.PlatformGlobalTenantID)
	}
	var existingRoleCount int64
	if err := db.Count(&existingRoleCount).Error; err != nil {
		return err
	}
	normalizedRoleKeys := impexp.SplitPipeValues(strings.Join(roleKeys, "|"))
	if existingRoleCount != int64(len(normalizedRoleKeys)) {
		for _, row := range rows {
			var count int64
			rowDB := s.db.Table("system_role").Where("role_key = ? AND deleted_at IS NULL", row.RoleKey)
			if !s.isPlatformScope() {
				rowDB = rowDB.Where(permissionTenantClause, s.tenantOwnerID())
			} else {
				rowDB = rowDB.Where(permissionTenantClause, tenant.PlatformGlobalTenantID)
			}
			if err := rowDB.Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				impexp.AppendImportError(result, row.RowNumber, "roleKey", errPermissionRoleInvalid)
			}
		}
		return common.NewBadRequest("import.role.invalid")
	}
	return nil
}

// loadExistingPolicyMap loads the existing policies that the import could
// collide with: the subject set being written, never the whole installation.
func loadExistingPolicyMap(db *gorm.DB, subjects map[string]string, platformScope bool, tenantID uint64) (map[string]database.CasbinRule, error) {
	subjectList := make([]string, 0, len(subjects))
	for _, subject := range subjects {
		subjectList = append(subjectList, subject)
	}
	if len(subjectList) == 0 {
		return map[string]database.CasbinRule{}, nil
	}
	query := db.Model(&database.CasbinRule{}).Where(permissionPtypeClause, "p")
	if platformScope {
		query = query.Where("v0 IN ?", subjectList)
	} else {
		query = query.Where(permissionRoleKeyLikeClause, "%@tenant:"+strconv.FormatUint(tenantID, 10))
	}
	var policies []database.CasbinRule
	if err := query.Find(&policies).Error; err != nil {
		return nil, err
	}
	existingByKey := make(map[string]database.CasbinRule, len(policies))
	for _, policy := range policies {
		existingByKey[fmt.Sprintf("%s|%s|%s", policy.V0, policy.V1, policy.V2)] = policy
	}
	return existingByKey, nil
}

// writeImportPolicies writes new (non-duplicate) policies in a transaction.
func writeImportPolicies(db *gorm.DB, rows []policyImportRow, subjects map[string]string, existingByKey map[string]database.CasbinRule, result *impexp.ImportResult) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			subject := subjects[row.RoleKey]
			compositeKey := fmt.Sprintf("%s|%s|%s", subject, row.Path, row.Method)
			if _, ok := existingByKey[compositeKey]; ok {
				result.Updated++
				continue
			}
			policy := database.CasbinRule{
				PType: "p",
				V0:    subject,
				V1:    row.Path,
				V2:    row.Method,
			}
			if err := tx.Create(&policy).Error; err != nil {
				return err
			}
			result.Created++
		}
		return nil
	})
}

func (s *PermissionService) listPoliciesForExport(query *PermissionPolicyQuery) ([]database.CasbinRule, error) {
	var policies []database.CasbinRule
	db := s.applyPolicyTenantScope(s.db.Model(&database.CasbinRule{}).Where(permissionPtypeClause, "p"))
	if query != nil {
		if strings.TrimSpace(query.RoleKey) != "" {
			db = db.Where(permissionRoleKeyLikeClause, "%"+common.EscapeLikePattern(strings.TrimSpace(query.RoleKey))+"%")
		}
		if strings.TrimSpace(query.Path) != "" {
			db = db.Where("v1 LIKE ?", "%"+common.EscapeLikePattern(strings.TrimSpace(query.Path))+"%")
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

func (s *PermissionService) validatePolicyPayload(policyID uint64, operatorRoleKeys []string, roleKey string, subject string, path string, method string) (string, string, error) {
	roleKey = strings.TrimSpace(roleKey)
	path = strings.TrimSpace(path)
	method = normalizePolicyMethod(method)
	if roleKey == "" || path == "" || method == "" {
		return "", "", common.NewBadRequest("param.invalid")
	}
	if err := ensurePolicyWriteAllowed(operatorRoleKeys, path); err != nil {
		return "", "", err
	}
	if err := s.ensureRoleKeyExists(roleKey); err != nil {
		return "", "", err
	}
	if err := s.ensurePolicyUnique(policyID, subject, path, method); err != nil {
		return "", "", err
	}
	return path, method, nil
}

// resolvePolicySubject maps (roleKey, tenantId) to the stored Casbin subject:
// global policies keep the plain role key; tenant-scoped policies store the
// contract §4 form `role:<key>@tenant:<id>`. The tenant must exist and be
// active (a policy for a suspended/archived tenant would silently grant
// nothing but still pollute the namespace), and the tenant row itself is
// owned by system/org — this is a read-only existence check (contract §7).
func (s *PermissionService) resolvePolicySubject(roleKey string, tenantID uint64) (string, error) {
	if tenantID == 0 {
		return tenant.GlobalRoleSubject(roleKey), nil
	}
	var row tenant.Tenant
	if err := s.db.Select("id", "status").Where("id = ?", tenantID).First(&row).Error; err != nil {
		return "", common.NewBadRequest(errPermissionTenantInvalid)
	}
	if row.Status != tenant.TenantStatusActive {
		return "", common.NewBadRequest(errPermissionTenantInvalid)
	}
	return tenant.RoleSubject(roleKey, tenantID), nil
}

// ensureRoleKeyExists verifies the role key exists in the caller's tenant.
//
// It used to accept a role_key that existed in ANY tenant, which let a tenant
// administrator write policies (including data-scope policies) for a role that
// belongs to somebody else — acceptance F8. Platform scope is exempt because
// the platform console manages global roles, whose rows live at tenant 0.
func (s *PermissionService) ensureRoleKeyExists(roleKey string) error {
	var count int64
	db := s.db.Table("system_role").Where("role_key = ? AND deleted_at IS NULL", roleKey)
	if !s.isPlatformScope() {
		db = db.Where(permissionTenantClause, s.tenantOwnerID())
	}
	if err := db.Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return common.NewBadRequest(errPermissionRoleInvalid)
	}
	return nil
}

func (s *PermissionService) ensurePolicyUnique(policyID uint64, subject string, path string, method string) error {
	var count int64
	db := s.db.Model(&database.CasbinRule{}).Where(permissionPtypeClause+" AND v0 = ? AND v1 = ? AND v2 = ?", "p", subject, path, method)
	if policyID > 0 {
		db = db.Where("id <> ?", policyID)
	}
	if err := db.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return common.NewConflict(errPermissionPolicyExists)
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

// normalizePermissionPolicySort maps a client-supplied sort field to a
// whitelisted Casbin rule column (v0=roleKey, v1=path, v2=method), guarding
// against ORDER BY injection. Unknown fields fall back to id desc.
func normalizePermissionPolicySort(query *PermissionPolicyQuery) (string, bool) {
	if query == nil {
		return "id", true
	}

	sortWhitelist := map[string]string{
		"id":      "id",
		"roleKey": "v0",
		"path":    "v1",
		"method":  "v2",
	}

	column, ok := sortWhitelist[strings.TrimSpace(query.SortField)]
	if !ok {
		return "id", true
	}

	return column, strings.ToLower(strings.TrimSpace(query.SortOrder)) == "desc"
}

func isProtectedManagementPolicy(path string) bool {
	normalized := strings.TrimSpace(path)
	return strings.HasPrefix(normalized, protectedManagementPolicyPrefixPermission) ||
		strings.HasPrefix(normalized, protectedManagementPolicyPrefixRole)
}

func canWritePolicy(operatorRoleKeys []string, path string) bool {
	for _, roleKey := range operatorRoleKeys {
		if strings.TrimSpace(roleKey) == "admin" {
			return true
		}
	}
	return !isProtectedManagementPolicy(path)
}

func ensurePolicyWriteAllowed(operatorRoleKeys []string, path string) error {
	if canWritePolicy(operatorRoleKeys, path) {
		return nil
	}
	return common.NewForbidden("permission.escalation.forbidden")
}

func validateImportPolicyWriteScope(operatorRoleKeys []string, rows []policyImportRow) error {
	for _, row := range rows {
		if err := ensurePolicyWriteAllowed(operatorRoleKeys, row.Path); err != nil {
			return err
		}
	}
	return nil
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
	if err := database.Enforcer.LoadPolicy(); err != nil {
		slog.Error("casbin policy reload failed after DB write", "component", "system/iam/permission", "error", err)
		return err
	}
	database.NotifyCasbinWatcher()
	return nil
}
