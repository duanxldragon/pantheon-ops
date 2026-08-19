package namespace

import "time"

type NamespaceBinding struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ClusterID       uint64    `gorm:"not null;uniqueIndex:uk_k8s_namespace_binding" json:"clusterId"`
	Namespace       string    `gorm:"size:255;not null;uniqueIndex:uk_k8s_namespace_binding" json:"namespace"`
	BusinessScopeID uint64    `gorm:"not null;index" json:"businessScopeId"`
	Environment     string    `gorm:"size:32;not null" json:"environment"`
	AllowedActions  string    `gorm:"type:text" json:"allowedActions"`
	CreatedBy       string    `gorm:"size:64" json:"createdBy"`
	UpdatedBy       string    `gorm:"size:64" json:"updatedBy"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func (NamespaceBinding) TableName() string { return "biz_k8s_namespace_binding" }
