package observability

import (
	"time"

	"gorm.io/gorm"
)

// queryStatusSuccess is the status a Prometheus-compatible datasource returns
// for a successful query.
const queryStatusSuccess = "success"

// PrometheusDatasource stores an observability backend configuration. The table
// name is retained for migration compatibility with the original Prometheus-only model.
type PrometheusDatasource struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	DeptID      uint64         `gorm:"column:dept_id;index" json:"deptId"`
	Name        string         `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Type        string         `gorm:"size:32;not null;default:'prometheus'" json:"type"`
	URL         string         `gorm:"size:512;not null" json:"url"`
	AccessMode  string         `gorm:"size:32;not null;default:'proxy'" json:"accessMode"` // proxy, direct
	Username    string         `gorm:"size:128" json:"username,omitempty"`
	Password    string         `gorm:"size:512" json:"-"`         // encrypted
	BearerToken string         `gorm:"size:1024" json:"-"`        // encrypted
	Timeout     int            `gorm:"default:30" json:"timeout"` // seconds
	IsDefault   bool           `gorm:"default:false" json:"isDefault"`
	Status      string         `gorm:"size:32;not null;default:'unknown'" json:"status"` // healthy, unhealthy, unknown
	LastChecked *time.Time     `json:"lastChecked,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	CreatedBy   string         `gorm:"size:64" json:"createdBy"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the obs_prometheus_datasource table name.
func (PrometheusDatasource) TableName() string {
	return "obs_prometheus_datasource"
}

// DatasourceDTO for API requests
type DatasourceDTO struct {
	Name        string `json:"name" binding:"required"`
	Type        string `json:"type" binding:"required,oneof=prometheus victoriametrics loki tempo"`
	URL         string `json:"url" binding:"required,url"`
	AccessMode  string `json:"accessMode" binding:"omitempty,oneof=proxy direct"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	BearerToken string `json:"bearerToken"`
	Timeout     int    `json:"timeout"`
	IsDefault   bool   `json:"isDefault"`
}

// DatasourceResponse is the Prometheus datasource shape returned to clients.
type DatasourceResponse struct {
	ID          uint64     `json:"id"`
	DeptID      uint64     `json:"deptId"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	URL         string     `json:"url"`
	AccessMode  string     `json:"accessMode"`
	Username    string     `json:"username,omitempty"`
	Timeout     int        `json:"timeout"`
	IsDefault   bool       `json:"isDefault"`
	Status      string     `json:"status"`
	LastChecked *time.Time `json:"lastChecked,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	CreatedBy   string     `json:"createdBy"`
}

// DatasourceListResponse is the list of configured Prometheus datasources.
type DatasourceListResponse struct {
	Items []DatasourceResponse `json:"items"`
	Total int64                `json:"total"`
}
