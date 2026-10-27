package tenant

import (
	"regexp"
	"strings"
	"time"
)

// Plan is a platform-global plan definition (TENANT_CONTRACT_V1 §2.4).
//
// It carries the quota thresholds a tenant's `plan` code resolves to, so the
// console can report usage against a limit the operator actually controls
// instead of a table baked into the binary.
//
// Two properties the rest of the design depends on:
//
//   - The row belongs to no tenant. `tenants` itself is the only other
//     platform-global table in this module, and neither carries a `tenant_id`
//     column: the tenant scope is applied by explicit WithTenantScope calls in
//     the services that need it, so a plan table would be filtered into
//     nothingness if it were scoped by accident.
//   - A limit of -1 means unlimited, 0 is not a legal value. Every consumer
//     (metrics, the overview page, the console) already distinguishes
//     "unlimited" from "exhausted"; allowing 0 would make those two states
//     indistinguishable.
type Plan struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Code         string    `gorm:"size:32;not null;uniqueIndex:uk_tenant_plan_code" json:"code"`
	Name         string    `gorm:"size:64;not null;default:''" json:"name"`
	MaxUsers     int       `gorm:"column:max_users;not null;default:0" json:"maxUsers"`
	MaxRoles     int       `gorm:"column:max_roles;not null;default:0" json:"maxRoles"`
	MaxDepts     int       `gorm:"column:max_depts;not null;default:0" json:"maxDepts"`
	MaxAPIKeys   int       `gorm:"column:max_api_keys;not null;default:0" json:"maxApiKeys"`
	MaxStorageMB int       `gorm:"column:max_storage_mb;not null;default:0" json:"maxStorageMb"`
	Status       string    `gorm:"size:16;not null;default:active" json:"status"` // active, disabled
	Sort         int       `gorm:"not null;default:0" json:"sort"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// TableName pins the projection to the plan definition table.
func (Plan) TableName() string {
	return "tenant_plans"
}

// Plan lifecycle values.
const (
	PlanStatusActive   = "active"
	PlanStatusDisabled = "disabled"
)

// The built-in plan codes are the plans the compiled-in fallback table
// describes. They are named once because three behaviours must agree on the
// same list - SeedDefaults creating the rows, the assignment check still
// accepting a code whose row the operator deleted, and GetDefaultQuota
// supplying its ceilings - and a code missing from one of them reads as an
// unassignable plan.
const (
	planCodeFree         = "free"
	planCodeBasic        = "basic"
	planCodeProfessional = "professional"
	planCodeEnterprise   = "enterprise"
)

// builtInPlanCodes is that list in console order.
var builtInPlanCodes = []string{planCodeFree, planCodeBasic, planCodeProfessional, planCodeEnterprise}

// UnlimitedQuota is the sentinel a quota field takes when a plan places no
// ceiling on that resource.
const UnlimitedQuota = -1

// planCodePattern is the accepted shape of a plan code. Codes travel inside
// `tenants.plan` (VARCHAR(32)) and are typed into the console's plan filter,
// so the constraint is enforced where the code is authored rather than
// trusting the caller.
var planCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,31}$`)

// normalizePlanCode trims the code an operator typed. An empty code is not a
// plan the console can manage; it is the "no plan chosen" value the column
// default already stores, and callers map it to the free tier.
func normalizePlanCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

// planCodeAllowed reports whether a tenant may carry this plan code on create
// or update. Empty is legal (it is what pre-plan tenants store) and resolves to
// the free tier everywhere it is read.
func planCodeAllowed(code string) bool {
	code = normalizePlanCode(code)
	return code == "" || planCodePattern.MatchString(code)
}

// AsQuota projects the row onto the Quota shape every consumer of tenant
// limits already speaks.
func (p Plan) AsQuota() Quota {
	return Quota{
		MaxUsers:     p.MaxUsers,
		MaxRoles:     p.MaxRoles,
		MaxDepts:     p.MaxDepts,
		MaxAPIKeys:   p.MaxAPIKeys,
		MaxStorageMB: p.MaxStorageMB,
	}
}

// IsEnabled reports whether the plan may still be assigned to tenants.
func (p Plan) IsEnabled() bool {
	return normalizePlanStatus(p.Status) == PlanStatusActive
}

func normalizePlanStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case PlanStatusDisabled:
		return PlanStatusDisabled
	default:
		return PlanStatusActive
	}
}

// PlanQuotaReq is the quota block shared by create and update.
//
// Pointers keep "leave this alone" distinguishable from "set it to -1" on the
// update path; the create handler rejects any nil because a plan with an
// undefined ceiling cannot be reported against.
type PlanQuotaReq struct {
	MaxUsers     *int `json:"maxUsers,omitempty"`
	MaxRoles     *int `json:"maxRoles,omitempty"`
	MaxDepts     *int `json:"maxDepts,omitempty"`
	MaxAPIKeys   *int `json:"maxApiKeys,omitempty"`
	MaxStorageMB *int `json:"maxStorageMb,omitempty"`
}

// PlanCreateReq is the body of POST /system/tenant-ops/plan.
type PlanCreateReq struct {
	Code   string `json:"code" binding:"required,min=2,max=32"`
	Name   string `json:"name" binding:"omitempty,max=64"`
	Status string `json:"status" binding:"omitempty,oneof=active disabled"`
	Sort   int    `json:"sort" binding:"omitempty,min=0,max=9999"`
	PlanQuotaReq
}

// PlanUpdateReq is the body of PUT /system/tenant-ops/plan/:code.
type PlanUpdateReq struct {
	Name   *string `json:"name,omitempty" binding:"omitempty,max=64"`
	Status *string `json:"status,omitempty" binding:"omitempty,oneof=active disabled"`
	Sort   *int    `json:"sort,omitempty" binding:"omitempty,min=0,max=9999"`
	PlanQuotaReq
}

// PlanResp is one plan row as the console sees it: the definition plus how many
// tenants currently sit on it.
//
// The tenant count is part of the payload rather than a second endpoint because
// the only decision this page supports is "may I retire this plan", and that
// decision is exactly `tenantCount == 0`.
type PlanResp struct {
	Plan
	TenantCount int64 `json:"tenantCount"`
}
