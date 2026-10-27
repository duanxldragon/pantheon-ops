package tenant

import (
	"time"
)

// Tenant represents a tenant in the multi-tenant system (TENANT_CONTRACT_V1 §2.1)
type Tenant struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Code      string    `gorm:"size:64;not null;uniqueIndex:uk_tenant_code" json:"code"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	Status    string    `gorm:"size:16;not null;default:active" json:"status"` // active, suspended, deleted
	Plan      string    `gorm:"size:32;not null;default:''" json:"plan"`       // free, basic, professional, enterprise
	Metadata  string    `gorm:"type:text" json:"metadata,omitempty"`           // JSON metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName pins the projection to the `tenants` master table.
func (Tenant) TableName() string {
	return "tenants"
}

// Membership represents user membership in a tenant
type Membership struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID  uint64    `gorm:"not null;index:idx_tenant_membership_tenant_user,priority:1;uniqueIndex:uk_tenant_membership" json:"tenant_id"`
	UserID    uint64    `gorm:"not null;index:idx_tenant_membership_tenant_user,priority:2;uniqueIndex:uk_tenant_membership" json:"user_id"`
	Role      string    `gorm:"size:32;not null" json:"role"` // owner, admin, member
	Status    string    `gorm:"size:16;not null;default:active" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName pins the projection to the `tenant_memberships` table.
func (Membership) TableName() string {
	return "tenant_memberships"
}

// CreateTenantDTO is the input for creating a new tenant.
//
// The Admin* fields drive provisioning: when AdminUsername is supplied the
// tenant is created ready to use — a tenant-local administrator user, the
// tenant role set, the admin's role binding, the owner membership the login
// gate requires, and the Casbin policies for the tenant's roles. Omitting them
// still provisions the roles and settings, but creates no login.
type CreateTenantDTO struct {
	Code string `json:"code" binding:"required,min=2,max=64"`
	Name string `json:"name" binding:"required,min=2,max=128"`
	// Plan carries no `oneof` on purpose: the accepted set is the enabled rows of
	// `tenant_plans` (contract §2.4), which the operator can extend. A static tag
	// here would reject a plan the console has no way to remove, and a new plan
	// would need a code release. CreateTenant validates the code against the
	// store instead.
	Plan     string `json:"plan" binding:"omitempty,max=32"`
	Metadata string `json:"metadata,omitempty"`

	AdminUsername string `json:"admin_username,omitempty" binding:"omitempty,min=2,max=64"`
	// AdminPassword carries no minimum in its binding tag on purpose: the
	// minimum is the deployment's configurable policy
	// (`security.password_min_length`), enforced in CreateTenant. A hardcoded
	// value here would silently disagree with both that policy and the console
	// form — which is exactly how a 6-character password the form accepted came
	// back as a bare `param.invalid`. Only the ceiling is structural.
	AdminPassword string `json:"admin_password,omitempty" binding:"omitempty,max=128"`
	AdminNickname string `json:"admin_nickname,omitempty" binding:"omitempty,max=64"`
}

// UpdateTenantDTO is the input for updating a tenant.
//
// Status accepts the three manageable lifecycle values (contract §2.1).
// `deleted` is deliberately excluded: it is the soft-delete marker written by
// DeleteTenant, not an editable state — allowing it here would let an update
// masquerade as a delete without the secure-action confirmation the delete
// route requires.
type UpdateTenantDTO struct {
	Name     *string `json:"name,omitempty" binding:"omitempty,min=2,max=128"`
	Status   *string `json:"status,omitempty" binding:"omitempty,oneof=active suspended archived"`
	Plan     *string `json:"plan,omitempty" binding:"omitempty,max=32"`
	Metadata *string `json:"metadata,omitempty"`
}

// AddTenantMemberDTO is the input for adding a member to a tenant
type AddTenantMemberDTO struct {
	UserID uint64 `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required,oneof=owner admin member"`
}

// Filter is used for filtering tenants in list queries.
//
// Keyword is a case-insensitive substring match over code and name, which is
// what the console search box sends.
type Filter struct {
	Status  string `form:"status" binding:"omitempty,oneof=active suspended archived deleted"`
	Plan    string `form:"plan" binding:"omitempty,max=32"`
	Keyword string `form:"keyword" binding:"omitempty,max=128"`
	Page    int    `form:"page" binding:"omitempty,min=1"`
	Size    int    `form:"size" binding:"omitempty,min=1,max=100"`
}

// Summary carries per-status tenant counts for the list summary cards.
//
// The counts are facets: they apply every list filter except `status`, so
// `Active + Suspended + Archived` is the number of non-deleted tenants matching
// the current keyword/plan filter. Computing them server-side is what keeps the
// summary independent of the current page.
type Summary struct {
	Total     int64 `json:"total"`
	Active    int64 `json:"active"`
	Suspended int64 `json:"suspended"`
	Archived  int64 `json:"archived"`
	Deleted   int64 `json:"deleted"`
}

// ListResponse wraps paginated tenant list
type ListResponse struct {
	Items   []Tenant `json:"items"`
	Total   int64    `json:"total"`
	Page    int      `json:"page"`
	Size    int      `json:"size"`
	Summary *Summary `json:"summary,omitempty"`
}
