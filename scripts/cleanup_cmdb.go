package main

import (
	"fmt"
	"os"

	"pantheon-ops/backend/modules/business"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	dsn := os.Getenv("PANTHEON_DSN")
	if dsn == "" {
		dsn = "root:DHCCroot@2025@tcp(127.0.0.1:3306)/pantheon_ops?charset=utf8mb4&parseTime=True&loc=Local"
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Println("Failed to connect database:", err)
		return
	}

	fmt.Println("Cleaning up retired CMDB module metadata while preserving biz_cmdb_* tables...")
	if err := business.CleanupRetiredBusinessModules(db); err != nil {
		fmt.Println("Cleanup failed:", err)
		return
	}
	fmt.Println("Successfully cleaned up retired CMDB module metadata.")
}
