package database

import (
	"fmt"
	"log"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

func normalizeMySQLDSN(dsn string) (string, error) {
	parsed, err := mysqlDriver.ParseDSN(dsn)
	if err != nil {
		return "", fmt.Errorf("invalid mysql dsn: %w", err)
	}
	if parsed.DBName == "" {
		return "", fmt.Errorf("invalid mysql dsn: database name is required")
	}
	if parsed.Params == nil {
		parsed.Params = map[string]string{}
	}
	if _, ok := parsed.Params["charset"]; !ok {
		parsed.Params["charset"] = "utf8mb4"
	}
	if !parsed.ParseTime {
		parsed.ParseTime = true
	}
	if parsed.Loc == nil {
		parsed.Loc = time.Local
	}
	return parsed.FormatDSN(), nil
}

// InitDB 初始化数据库连接
func InitDB(dsn string) {
	var err error
	normalizedDSN, err := normalizeMySQLDSN(dsn)
	if err != nil {
		log.Fatalf("PANTHEON_DSN must be a valid MySQL DSN: %v", err)
	}

	DB, err = gorm.Open(mysql.Open(normalizedDSN), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // 使用单数表名
		},
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	sqlDB, _ := DB.DB()
	sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)          // 最大开放连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大存活时间

	fmt.Println("Database connection successful")
}
