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

// CreateTenantDTO is the input for creating a new tenant
type CreateTenantDTO struct {
	Code     string `json:"code" binding:"required,min=2,max=64"`
	Name     string `json:"name" binding:"required,min=2,max=128"`
	Plan     string `json:"plan" binding:"omitempty,oneof=free basic professional enterprise"`
	Metadata string `json:"metadata,omitempty"`
}

// UpdateTenantDTO is the input for updating a tenant
type UpdateTenantDTO struct {
	Name     *string `json:"name,omitempty" binding:"omitempty,min=2,max=128"`
	Status   *string `json:"status,omitempty" binding:"omitempty,oneof=active suspended"`
	Plan     *string `json:"plan,omitempty" binding:"omitempty,oneof=free basic professional enterprise"`
	Metadata *string `json:"metadata,omitempty"`
}

// AddTenantMemberDTO is the input for adding a member to a tenant
type AddTenantMemberDTO struct {
	UserID uint64 `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required,oneof=owner admin member"`
}

// Filter is used for filtering tenants in list queries
type Filter struct {
	Status string `form:"status" binding:"omitempty,oneof=active suspended deleted"`
	Plan   string `form:"plan" binding:"omitempty,oneof=free basic professional enterprise"`
	Page   int    `form:"page" binding:"omitempty,min=1"`
	Size   int    `form:"size" binding:"omitempty,min=1,max=100"`
}

// ListResponse wraps paginated tenant list
type ListResponse struct {
	Items []Tenant `json:"items"`
	Total int64    `json:"total"`
	Page  int      `json:"page"`
	Size  int      `json:"size"`
}
