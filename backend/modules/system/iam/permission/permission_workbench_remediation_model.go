package permission

import "time"

type PermissionWorkbenchRemediationEvent struct {
	ID uint64 `gorm:"primaryKey;autoIncrement"`
	// TenantID is the tenant whose role was remediated (0 = platform/global).
	// Without it a tenant administrator's remediation history was readable by
	// every tenant that happened to own a role with the same key.
	TenantID     uint64    `gorm:"not null;default:0;index:idx_permission_remediation_tenant_created,priority:1"`
	RoleKey      string    `gorm:"size:64;not null;index:idx_permission_remediation_role_created,priority:1"`
	IssueType    string    `gorm:"size:32;not null;index"`
	IssueKey     string    `gorm:"size:255;not null"`
	BeforeState  string    `gorm:"size:32;not null"`
	AfterState   string    `gorm:"size:32;not null"`
	Action       string    `gorm:"size:32;not null;index"`
	CreatedCount int       `gorm:"default:0"`
	SkippedCount int       `gorm:"default:0"`
	CreatedAt    time.Time `gorm:"index:idx_permission_remediation_role_created,priority:2"`
}

func (PermissionWorkbenchRemediationEvent) TableName() string {
	return "permission_workbench_remediation_event"
}
