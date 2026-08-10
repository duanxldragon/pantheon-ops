package iam

import "time"

type PermissionWorkbenchRemediationEvent struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement"`
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
