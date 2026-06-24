package main

import (
	"log/slog"
	"os"

	"pantheon-ops/backend/internal/middleware"
	"pantheon-ops/backend/modules/auth"
	"pantheon-ops/backend/modules/business"
	"pantheon-ops/backend/modules/dashboard"
	"pantheon-ops/backend/modules/platform"
	"pantheon-ops/backend/modules/system"
	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/database"

	"github.com/gin-gonic/gin"
)

func main() {
	// 0. 初始化核心基础能力
	common.InitLocationService()
	if err := common.InitSecurityConfig(); err != nil {
		slog.Error("security configuration invalid", "error", err)
		os.Exit(1)
	}

	// 1. 初始化数据库
	dsn := os.Getenv("PANTHEON_DSN")
	if dsn == "" {
		slog.Error("PANTHEON_DSN is required")
		os.Exit(1)
	}
	database.InitDB(dsn)

	// 1b. 数据库迁移：默认使用版本化迁移(golang-migrate)，开发模式可启用 AutoMigrate
	if database.ShouldAutoMigrate() {
		slog.Info("PANTHEON_AUTO_MIGRATE=true: using GORM AutoMigrate (dev mode)")
		// AutoMigrate 由各模块的 Migrate() 方法在注册时自动执行
	} else {
		if err := database.RunMigrations(dsn); err != nil {
			slog.Error("database migration failed", "error", err)
			os.Exit(1)
		}
	}

	// 2. 初始化 Redis (默认本地地址)
	if redisAddr := os.Getenv("PANTHEON_REDIS_ADDR"); redisAddr != "" {
		database.InitRedis(redisAddr, os.Getenv("PANTHEON_REDIS_PASSWORD"), 0)
	}

	database.InitCasbin(database.DB)

	// 3. 初始化 Gin
	r := gin.Default()
	r.Use(middleware.SecurityHeadersMiddleware())
	r.Use(middleware.BodySizeLimit(middleware.DefaultMaxBodyBytes))
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.RequestContextMiddleware(), middleware.OperationLogMiddleware(database.DB))
	r.Use(middleware.CSRFMiddleware())

	// 3. 注册底座模块
	api := r.Group("/api/v1")
	platform.RegisterHealthRoutes(api, database.DB)
	dashboard.InitDashboardModule(api, database.DB)
	system.InitSystemModule(api, database.DB)
	auth.InitAuthModule(api, database.DB)
	business.InitBusinessModules(api, database.DB)

	// 4. 启动服务器
	port := os.Getenv("PANTHEON_PORT")
	if port == "" {
		port = "8080"
	}
	slog.Info("starting server", "port", port)
	if err := r.Run(":" + port); err != nil {
		slog.Error("failed to run server", "error", err)
		os.Exit(1)
	}
}
