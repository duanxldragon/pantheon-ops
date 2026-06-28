package generator

import "time"

const (
	generatorDatasourceCurrentID = "current"
	generatorDatasourceEnabled   = 1
	generatorDatasourceDisabled  = 0
)

type GeneratorDatasource struct {
	ID                uint64     `gorm:"primaryKey;autoIncrement"`
	Name              string     `gorm:"size:128;not null"`
	Driver            string     `gorm:"size:32;not null;default:mysql"`
	Host              string     `gorm:"size:255;not null"`
	Port              int        `gorm:"not null;default:3306"`
	DatabaseName      string     `gorm:"size:128;not null;column:database_name"`
	Username          string     `gorm:"size:128;not null"`
	PasswordEncrypted string     `gorm:"size:1024;column:password_encrypted"`
	Status            int        `gorm:"not null;default:1"`
	ReadonlyScope     string     `gorm:"size:32;not null;default:metadata_only;column:readonly_scope"`
	Remark            string     `gorm:"size:255"`
	LastCheckedAt     *time.Time `gorm:"column:last_checked_at"`
	LastCheckStatus   string     `gorm:"size:32;column:last_check_status"`
	LastCheckError    string     `gorm:"size:255;column:last_check_error"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (GeneratorDatasource) TableName() string {
	return "system_generator_datasource"
}
