package observability

import (
	"gorm.io/gorm"
)

// InitModule initializes the observability module
func InitModule(db *gorm.DB) error {
	// Auto migrate tables
	if err := db.AutoMigrate(
		&MetricSource{},
		&AlertRule{},
		&AlertRecord{},
		&NotificationChannel{},
	); err != nil {
		return err
	}

	return nil
}
