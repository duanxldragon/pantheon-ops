package database

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
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

func gormLogLevel() logger.LogLevel {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("PANTHEON_ENV")), "production") {
		return logger.Warn
	}
	return logger.Info
}

// DBDriver represents the database driver type.
// Currently only MySQL is supported; additional drivers can be added here.
type DBDriver string

const (
	DriverMySQL DBDriver = "mysql"
)

// InitDB initializes the database connection.
// The driver is determined by the DSN prefix or PANTHEON_DB_DRIVER env var.
// Currently only MySQL is supported; this abstraction allows future expansion.
func InitDB(dsn string) {
	driver := DBDriver(strings.ToLower(strings.TrimSpace(os.Getenv("PANTHEON_DB_DRIVER"))))
	if driver == "" {
		driver = DriverMySQL // default
	}

	switch driver {
	case DriverMySQL:
		initMySQL(dsn)
	default:
		slog.Error("unsupported database driver", "driver", driver)
		os.Exit(1)
	}
}

func initMySQL(dsn string) {
	var err error
	normalizedDSN, err := normalizeMySQLDSN(dsn)
	if err != nil {
		slog.Error("PANTHEON_DSN must be a valid MySQL DSN", "error", err)
		os.Exit(1)
	}

	DB, err = gorm.Open(mysql.Open(normalizedDSN), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // 使用单数表名
		},
		Logger: logger.Default.LogMode(gormLogLevel()),
	})

	if err != nil {
		slog.Error("failed to connect database", "error", err)
		os.Exit(1)
	}

	sqlDB, _ := DB.DB()
	sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)          // 最大开放连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大存活时间

	slog.Info("Database connection successful", "driver", "mysql")
}
