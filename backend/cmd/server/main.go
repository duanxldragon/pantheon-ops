package main

import (
	"log"
	"os"

	"pantheon-platform/backend/internal/middleware"
	"pantheon-platform/backend/modules/auth"
	"pantheon-platform/backend/modules/business"
	"pantheon-platform/backend/modules/dashboard"
	"pantheon-platform/backend/modules/platform"
	"pantheon-platform/backend/modules/system"
	"pantheon-platform/backend/pkg/common"
	"pantheon-platform/backend/pkg/database"

	"github.com/gin-gonic/gin"
)

func main() {
	// 0. 初始化核心基础能力
	common.InitLocationService()
	if err := common.InitSecurityConfig(); err != nil {
		log.Fatalf("security configuration invalid: %v", err)
	}

	// 1. 初始化数据库
	dsn := os.Getenv("PANTHEON_DSN")
	if dsn == "" {
		log.Fatalf("PANTHEON_DSN is required")
	}
	database.InitDB(dsn)

	// 2. 初始化 Redis (默认本地地址)
	if redisAddr := os.Getenv("PANTHEON_REDIS_ADDR"); redisAddr != "" {
		database.InitRedis(redisAddr, os.Getenv("PANTHEON_REDIS_PASSWORD"), 0)
	}

	database.InitCasbin(database.DB)

	// 3. 初始化 Gin
	r := gin.Default()
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
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
