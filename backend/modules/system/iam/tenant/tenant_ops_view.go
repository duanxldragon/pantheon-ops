package tenant

import (
	"fmt"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"gorm.io/gorm"
)

// The tenant-ops read model (contract §2.5): a platform-global view of how many
// tenants exist, how much of each plan they occupy, and who the members are.
//
// It is a separate surface from the tenant list on purpose. `/tenant/list` is a
// management grid of rows; this is a rollup whose numbers have to agree with the
// grid, so both must derive from the same predicates — which is why everything
// here reuses the module's shared constants (activeTenantMemberPredicate,
// tenantStatusDeleted, platformGlobalTenantID) instead of restating them.

// OverviewTotals is the installation-wide rollup shown above the tenant table.
type OverviewTotals struct {
	Members int64 `json:"members"`
	Roles   int64 `json:"roles"`
	Depts   int64 `json:"depts"`
}

// OverviewPlanRow is one plan's share of the installation.
//
// `plan` is the raw value stored on the tenant and can be empty or unknown: a
// tenant provisioned before plan definitions were editable carries whatever its
// creator typed. `known` says whether it resolves to a definition, so the page
// can show an unmanaged plan as itself instead of silently filing it under free.
type OverviewPlanRow struct {
	Plan        string `json:"plan"`
	Name        string `json:"name"`
	Known       bool   `json:"known"`
	TenantCount int64  `json:"tenantCount"`
	Quota       Quota  `json:"quota"`
}

// OverviewTenantRow is one tenant with its occupancy.
type OverviewTenantRow struct {
	TenantID  uint64 `json:"tenantId"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Plan      string `json:"plan"`
	PlanName  string `json:"planName"`
	Members   int64  `json:"members"`
	Roles     int64  `json:"roles"`
	Depts     int64  `json:"depts"`
	MaxUsers  int    `json:"maxUsers"`
	MaxRoles  int    `json:"maxRoles"`
	MaxDepts  int    `json:"maxDepts"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// OverviewResp is the payload of GET /system/tenant-ops/overview.
type OverviewResp struct {
	Summary *Summary            `json:"summary"`
	Totals  OverviewTotals      `json:"totals"`
	Plans   []OverviewPlanRow   `json:"plans"`
	Items   []OverviewTenantRow `json:"items"`
	Total   int64               `json:"total"`
	Page    int                 `json:"page"`
	Size    int                 `json:"size"`
}

// opsTenantScope is the WHERE clause every overview read shares: real tenants
// only. The platform row (id 0) is not a tenant, and the soft-delete marker
// stays out unless the caller asks for it explicitly through the status filter —
// the same rule ListTenants applies.
func opsTenantScope(db *gorm.DB, status string) *gorm.DB {
	scoped := db.Where(condIDNotEquals, platformGlobalTenantID)
	if status != "" {
		return scoped.Where(condStatusEquals, status)
	}
	return scoped.Where(condStatusNotEquals, tenantStatusDeleted)
}

// The overview counts the whole installation on every request, so the page it
// pages through is bounded: an unbounded size would turn one console call into a
// table scan of every tenant.
const (
	defaultTenantOverviewSize = 20
	maxTenantOverviewSize     = 100
)

// Overview builds the tenant-ops rollup for one page of tenants.
func (s *Service) Overview(filter Filter) (*OverviewResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Size < 1 {
		filter.Size = defaultTenantOverviewSize
	}
	if filter.Size > maxTenantOverviewSize {
		filter.Size = maxTenantOverviewSize
	}

	resp := &OverviewResp{Page: filter.Page, Size: filter.Size}

	// Status facets and the filtered total come from the same helper the tenant
	// list uses, so the two pages can never disagree about how many tenants exist.
	base := s.db.Model(&Tenant{}).Where(condIDNotEquals, platformGlobalTenantID)
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		pattern := "%" + likeEscape(keyword) + "%"
		base = base.Where("code LIKE ? OR name LIKE ?", pattern, pattern)
	}
	if plan := normalizePlanCode(filter.Plan); plan != "" {
		base = base.Where("plan = ?", plan)
	}
	summary, err := summarizeTenants(base)
	if err != nil {
		return nil, err
	}
	resp.Summary = summary

	scoped := opsTenantScope(base.Session(&gorm.Session{}), filter.Status)
	if err := scoped.Count(&resp.Total).Error; err != nil {
		return nil, fmt.Errorf("count tenants for overview: %w", err)
	}

	var tenants []Tenant
	if err := scoped.Order("id ASC").
		Limit(filter.Size).
		Offset((filter.Page - 1) * filter.Size).
		Find(&tenants).Error; err != nil {
		return nil, fmt.Errorf("list tenants for overview: %w", err)
	}

	tenantIDs := make([]uint64, 0, len(tenants))
	for _, row := range tenants {
		tenantIDs = append(tenantIDs, row.ID)
	}
	members, err := s.countByTenant(opsResourceMembers, tenantIDs)
	if err != nil {
		return nil, err
	}
	roles, err := s.countByTenant(opsResourceRoles, tenantIDs)
	if err != nil {
		return nil, err
	}
	depts, err := s.countByTenant(opsResourceDepts, tenantIDs)
	if err != nil {
		return nil, err
	}

	plans := s.planRows()

	for _, row := range tenants {
		quota := s.plans.QuotaFor(row.Plan)
		planName, known := plans[normalizePlanCode(row.Plan)]
		if !known && normalizePlanCode(row.Plan) != "" {
			planName = ""
		}
		resp.Items = append(resp.Items, OverviewTenantRow{
			TenantID:  row.ID,
			Code:      row.Code,
			Name:      row.Name,
			Status:    row.Status,
			Plan:      row.Plan,
			PlanName:  planName,
			Members:   members[row.ID],
			Roles:     roles[row.ID],
			Depts:     depts[row.ID],
			MaxUsers:  quota.MaxUsers,
			MaxRoles:  quota.MaxRoles,
			MaxDepts:  quota.MaxDepts,
			CreatedAt: row.CreatedAt.Format(time.RFC3339),
			UpdatedAt: row.UpdatedAt.Format(time.RFC3339),
		})
	}

	if resp.Totals, err = s.overviewTotals(); err != nil {
		return nil, err
	}
	if resp.Plans, err = s.planDistribution(); err != nil {
		return nil, err
	}
	return resp, nil
}

// planRows returns the definition names by normalized code. An unreadable store
// yields an empty map rather than an error: the overview reports occupancy, and a
// plan label is decoration on top of the numbers.
func (s *Service) planRows() map[string]string {
	names := map[string]string{}
	if s.plans == nil {
		return names
	}
	list, err := s.plans.List()
	if err != nil {
		return names
	}
	for _, plan := range list {
		names[normalizePlanCode(plan.Code)] = plan.Name
	}
	return names
}

// planDistribution groups tenants by the plan code they carry.
func (s *Service) planDistribution() ([]OverviewPlanRow, error) {
	var rows []struct {
		Plan  string
		Total int64
	}
	if err := s.db.Model(&Tenant{}).
		Where(condIDNotEquals, platformGlobalTenantID).
		Where(condStatusNotEquals, tenantStatusDeleted).
		Select("plan, COUNT(*) AS total").
		Group("plan").
		Order("plan ASC").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("plan distribution: %w", err)
	}

	names := s.planRows()
	out := make([]OverviewPlanRow, 0, len(rows))
	for _, row := range rows {
		code := normalizePlanCode(row.Plan)
		name, known := names[code]
		if !known && code != "" {
			name = ""
		}
		out = append(out, OverviewPlanRow{
			Plan:        row.Plan,
			Name:        name,
			Known:       known,
			TenantCount: row.Total,
			Quota:       s.plans.QuotaFor(row.Plan),
		})
	}
	return out, nil
}

// overviewTotals counts members, roles and departments across every real tenant.
//
// Platform rows are excluded: system_role and system_dept carry tenant_id 0 for
// the platform's own roles and departments, and folding them in would make the
// rollup claim platform administration as some tenant's usage.
func (s *Service) overviewTotals() (OverviewTotals, error) {
	var totals OverviewTotals
	var err error
	if totals.Members, err = s.countScoped(opsResourceMembers); err != nil {
		return totals, err
	}
	if totals.Roles, err = s.countScoped(opsResourceRoles); err != nil {
		return totals, err
	}
	if totals.Depts, err = s.countScoped(opsResourceDepts); err != nil {
		return totals, err
	}
	return totals, nil
}

// opsResource names the three occupancy dimensions the overview reports. Table
// names stay in this switch so no query can receive a caller-supplied identifier.
type opsResource string

const (
	opsResourceMembers opsResource = "members"
	opsResourceRoles   opsResource = "roles"
	opsResourceDepts   opsResource = "depts"
)

// opsTable returns the table, the "counts as occupancy" predicate and its args.
// Members count only active memberships; roles and departments respect soft
// delete.
func opsTable(resource opsResource) (string, string, []any) {
	switch resource {
	case opsResourceRoles:
		return "system_role", "deleted_at IS NULL", nil
	case opsResourceDepts:
		return "system_dept", "deleted_at IS NULL", nil
	default:
		return "tenant_memberships", "status = ?", []any{tenant.MembershipActive}
	}
}

func (s *Service) countScoped(resource opsResource) (int64, error) {
	table, predicate, args := opsTable(resource)
	var count int64
	if err := s.db.Table(table).
		Where(predicate, args...).
		Where("tenant_id <> ?", platformGlobalTenantID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count %s: %w", table, err)
	}
	return count, nil
}

// countByTenant returns one resource's per-tenant counts for the given ids in a
// single grouped query. A tenant with no rows is simply absent from the map,
// which reads as zero.
func (s *Service) countByTenant(resource opsResource, tenantIDs []uint64) (map[uint64]int64, error) {
	counts := make(map[uint64]int64, len(tenantIDs))
	if len(tenantIDs) == 0 {
		return counts, nil
	}
	table, predicate, args := opsTable(resource)
	var rows []struct {
		TenantID uint64
		Total    int64
	}
	if err := s.db.Table(table).
		Select("tenant_id, COUNT(*) AS total").
		Where(predicate, args...).
		Where("tenant_id IN ?", tenantIDs).
		Group("tenant_id").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("count %s per tenant: %w", table, err)
	}
	for _, row := range rows {
		counts[row.TenantID] = row.Total
	}
	return counts, nil
}

// MemberRowFilter narrows the cross-tenant member listing.
type MemberRowFilter struct {
	TenantID uint64 `form:"tenantId" binding:"omitempty,min=1"`
	Status   string `form:"status" binding:"omitempty,oneof=active disabled"`
	Role     string `form:"role" binding:"omitempty,oneof=owner admin member"`
	Keyword  string `form:"keyword" binding:"omitempty,max=128"`
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"pageSize" binding:"omitempty,min=1,max=100"`
}

// MemberRow is one membership as the platform console sees it: the person, the
// tenant they belong to, and that tenant's user ceiling so usage and limit sit on
// the same row.
type MemberRow struct {
	TenantID     uint64 `json:"tenantId"`
	TenantCode   string `json:"tenantCode"`
	TenantName   string `json:"tenantName"`
	TenantStatus string `json:"tenantStatus"`
	Plan         string `json:"plan"`
	PlanName     string `json:"planName"`
	MaxUsers     int    `json:"maxUsers"`
	TenantUsage  int64  `json:"tenantUsage"`
	UserID       uint64 `json:"userId"`
	Username     string `json:"username"`
	Nickname     string `json:"nickname"`
	MemberRole   string `json:"memberRole"`
	MemberStatus string `json:"memberStatus"`
	JoinedAt     string `json:"joinedAt"`
}

// MemberPageResp is the payload of GET /system/tenant-ops/member.
type MemberPageResp struct {
	Items    []MemberRow `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

// memberRow is the joined scan target; the plan fields are resolved in Go from
// the plan store rather than duplicated into SQL.
type memberRow struct {
	TenantID     uint64    `gorm:"column:tenant_id"`
	TenantCode   string    `gorm:"column:tenant_code"`
	TenantName   string    `gorm:"column:tenant_name"`
	TenantStatus string    `gorm:"column:tenant_status"`
	Plan         string    `gorm:"column:plan"`
	UserID       uint64    `gorm:"column:user_id"`
	Username     string    `gorm:"column:username"`
	Nickname     string    `gorm:"column:nickname"`
	MemberRole   string    `gorm:"column:member_role"`
	MemberStatus string    `gorm:"column:member_status"`
	JoinedAt     time.Time `gorm:"column:joined_at"`
}

// ListMembersAcrossPages every active membership of every real tenant.
//
// This is a declared platform-scoped read (contract §3.3): it deliberately spans
// tenants, which is only allowed behind the tenant console's
// PlatformCapabilityMiddleware. Nothing tenant-facing may reuse it.
func (s *Service) ListMembersAcrossPages(filter MemberRowFilter) (*MemberPageResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	page, pageSize := normalizeTenantMemberPagination(filter.Page, filter.PageSize)

	scope := s.db.Table("tenant_memberships").
		Joins("INNER JOIN tenants ON tenants.id = tenant_memberships.tenant_id").
		Joins("LEFT JOIN system_user ON system_user.id = tenant_memberships.user_id").
		Where("tenant_memberships.tenant_id <> ?", platformGlobalTenantID).
		Where("tenants.id <> ?", platformGlobalTenantID).
		Where("tenants.status <> ?", tenantStatusDeleted)

	if filter.TenantID != 0 {
		scope = scope.Where("tenant_memberships.tenant_id = ?", filter.TenantID)
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		scope = scope.Where("tenant_memberships.status = ?", status)
	} else {
		scope = scope.Where("tenant_memberships.status = ?", tenant.MembershipActive)
	}
	if role := strings.TrimSpace(filter.Role); role != "" {
		scope = scope.Where("tenant_memberships.role = ?", role)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		pattern := "%" + likeEscape(keyword) + "%"
		scope = scope.Where(
			"system_user.username LIKE ? OR system_user.nickname LIKE ? OR tenants.code LIKE ? OR tenants.name LIKE ?",
			pattern, pattern, pattern, pattern,
		)
	}

	var total int64
	if err := scope.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count tenant members across tenants: %w", err)
	}

	rows := make([]memberRow, 0)
	if err := scope.Select(`tenant_memberships.tenant_id AS tenant_id,
			tenants.code AS tenant_code,
			tenants.name AS tenant_name,
			tenants.status AS tenant_status,
			tenants.plan AS plan,
			tenant_memberships.user_id AS user_id,
			system_user.username AS username,
			system_user.nickname AS nickname,
			tenant_memberships.role AS member_role,
			tenant_memberships.status AS member_status,
			tenant_memberships.created_at AS joined_at`).
		Order("tenant_memberships.tenant_id ASC, tenant_memberships.id ASC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list tenant members across tenants: %w", err)
	}

	names := s.planRows()
	usage, err := s.countByTenant(opsResourceMembers, distinctTenantIDs(rows))
	if err != nil {
		return nil, err
	}

	items := make([]MemberRow, 0, len(rows))
	for _, row := range rows {
		quota := s.plans.QuotaFor(row.Plan)
		planName, known := names[normalizePlanCode(row.Plan)]
		if !known {
			planName = ""
		}
		items = append(items, MemberRow{
			TenantID:     row.TenantID,
			TenantCode:   row.TenantCode,
			TenantName:   row.TenantName,
			TenantStatus: row.TenantStatus,
			Plan:         row.Plan,
			PlanName:     planName,
			MaxUsers:     quota.MaxUsers,
			TenantUsage:  usage[row.TenantID],
			UserID:       row.UserID,
			Username:     row.Username,
			Nickname:     row.Nickname,
			MemberRole:   row.MemberRole,
			MemberStatus: row.MemberStatus,
			JoinedAt:     row.JoinedAt.Format(time.RFC3339),
		})
	}

	return &MemberPageResp{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func distinctTenantIDs(rows []memberRow) []uint64 {
	seen := make(map[uint64]struct{}, len(rows))
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		if _, ok := seen[row.TenantID]; ok {
			continue
		}
		seen[row.TenantID] = struct{}{}
		ids = append(ids, row.TenantID)
	}
	return ids
}

// ListPlans returns every definition with how many tenants sit on it, which is
// the number that decides whether a plan can be retired.
func (s *Service) ListPlans() ([]PlanResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	list, err := s.plans.List()
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Plan  string
		Total int64
	}
	if err := s.db.Model(&Tenant{}).
		Where(condIDNotEquals, platformGlobalTenantID).
		Where(condStatusNotEquals, tenantStatusDeleted).
		Select("plan, COUNT(*) AS total").
		Group("plan").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("count tenants per plan: %w", err)
	}
	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[normalizePlanCode(row.Plan)] += row.Total
	}

	out := make([]PlanResp, 0, len(list))
	for _, plan := range list {
		out = append(out, PlanResp{
			Plan:        plan,
			TenantCount: counts[normalizePlanCode(plan.Code)],
		})
	}
	return out, nil
}
