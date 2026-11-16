package observability

import (
	"gorm.io/datatypes"
	"time"
)

// QueryHistory is a persisted query workspace history row.
type QueryHistory struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement;index:idx_history_owner,priority:4;index:idx_history_expiry,priority:2" json:"id"`
	TenantID     uint64         `gorm:"not null;index:idx_history_owner,priority:1;index:idx_history_source,priority:1" json:"-"`
	ActorID      uint64         `gorm:"not null;index:idx_history_owner,priority:2" json:"-"`
	DatasourceID uint64         `gorm:"not null;index:idx_history_source,priority:2" json:"datasourceId"`
	SourceKind   string         `gorm:"size:32;not null" json:"sourceKind"`
	Time         datatypes.JSON `gorm:"column:time_intent;type:json;not null" json:"time"`
	Limit        int            `gorm:"column:row_limit;not null" json:"limit"`
	ResultMode   string         `gorm:"size:16;not null" json:"resultMode"`
	Outcome      string         `gorm:"size:16;not null" json:"outcome"`
	DurationMS   uint32         `gorm:"not null" json:"durationMs"`
	Recipe       datatypes.JSON `gorm:"type:json" json:"recipe,omitempty"`
	CreatedAt    time.Time      `gorm:"not null;index:idx_history_owner,priority:3" json:"createdAt"`
	ExpiresAt    time.Time      `gorm:"not null;index:idx_history_expiry,priority:1" json:"-"`
}

// TableName returns the storage table for query history.
func (QueryHistory) TableName() string { return "obs_query_history" }

// SavedView is a persisted, revisioned saved query view.
type SavedView struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement;index:idx_view_owner,priority:4" json:"id"`
	TenantID     uint64         `gorm:"not null;index:idx_view_owner,priority:1;index:idx_view_source,priority:1" json:"-"`
	ActorID      uint64         `gorm:"not null;index:idx_view_owner,priority:2" json:"-"`
	DatasourceID uint64         `gorm:"not null;index:idx_view_source,priority:2" json:"datasourceId"`
	Recipe       datatypes.JSON `gorm:"type:json;not null" json:"recipe"`
	Revision     uint64         `gorm:"not null;default:1" json:"revision"`
	CreatedAt    time.Time      `gorm:"not null" json:"createdAt"`
	UpdatedAt    time.Time      `gorm:"not null;index:idx_view_owner,priority:3" json:"updatedAt"`
}

// TableName returns the storage table for saved views.
func (SavedView) TableName() string { return "obs_saved_view" }

// QueryOwnerBudget is a concurrent-lock guard row. Empty guard rows are
// retained: deleting one permits a concurrent transaction to lock a different
// physical row for the same logical owner.
type QueryOwnerBudget struct {
	TenantID  uint64    `gorm:"primaryKey;autoIncrement:false"`
	ActorID   uint64    `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

// TableName returns the storage table for query owner budget guard rows.
func (QueryOwnerBudget) TableName() string { return "obs_query_owner_budget" }

// SchemaCapability records the capability flag a migrated schema declares.
type SchemaCapability struct {
	Capability string    `gorm:"primaryKey;size:64"`
	Version    uint16    `gorm:"not null"`
	CreatedAt  time.Time `gorm:"not null"`
}

// TableName returns the storage table for schema capability flags.
func (SchemaCapability) TableName() string { return "obs_schema_capability" }
