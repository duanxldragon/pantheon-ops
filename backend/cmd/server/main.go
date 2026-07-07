package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"pantheon-ops/backend/internal/middleware"
	"pantheon-ops/backend/modules/auth"
	"pantheon-ops/backend/modules/business"
	"pantheon-ops/backend/modules/lowcode"
	"pantheon-ops/backend/modules/platform"
	"pantheon-ops/backend/modules/system"
	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/database"
	"pantheon-ops/backend/pkg/logging"
	"pantheon-ops/backend/pkg/telemetry"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"
)

func main() {
	// 0. 初始化结构化日志
	env := os.Getenv("PANTHEON_ENV")
	if env == "" {
		env = "development"
	}

	if err := logging.InitLogger(env); err != nil {
		slog.Error("Failed to initialize logger", "error", err)
		os.Exit(1)
	}
	defer logging.Sync()

	logging.Info("Starting Pantheon Base",
		zap.String("version", "0.8.3"),
		zap.String("environment", env),
	)

	// 0b. 初始化 OpenTelemetry 追踪
	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otlpEndpoint != "" {
		_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tp, err := telemetry.InitTracer("pantheon-base", otlpEndpoint)
		if err != nil {
			logging.Error("Failed to initialize tracer", zap.Error(err))
		} else {
			defer func() {
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer shutdownCancel()
				if err := tp.Shutdown(shutdownCtx); err != nil {
					logging.Error("Error shutting down tracer", zap.Error(err))
				}
			}()
			logging.Info("OpenTelemetry tracer initialized", zap.String("endpoint", otlpEndpoint))
		}
	}

	// 1. 初始化核心基础能力
	common.InitLocationService()
	if err := common.InitSecurityConfig(); err != nil {
		logging.Error("Security configuration invalid", zap.Error(err))
		os.Exit(1)
	}

	// 1. 初始化数据库
	dsn := os.Getenv("PANTHEON_DSN")
	if dsn == "" {
		logging.Fatal("PANTHEON_DSN is required")
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
	r.Use(middleware.CSPMiddleware()) // Content-Security-Policy
	r.Use(middleware.BodySizeLimit(middleware.DefaultMaxBodyBytes))
	r.Use(middleware.CORSMiddleware())
	r.Use(otelgin.Middleware("pantheon-base")) // OpenTelemetry 追踪
	r.Use(middleware.PrometheusMiddleware())   // Prometheus 指标采集
	r.Use(middleware.RequestContextMiddleware(), middleware.OperationLogMiddleware(database.DB))
	r.Use(middleware.CSRFMiddleware())

	// 3. 注册 Prometheus metrics 端点。生产环境默认要求显式 token 或公开开关。
	if shouldExposeMetrics(env) {
		r.GET("/metrics", metricsAccessMiddleware(), gin.WrapH(promhttp.Handler()))
	} else {
		logging.Warn("Prometheus metrics endpoint disabled; set PANTHEON_METRICS_BEARER_TOKEN or PANTHEON_METRICS_PUBLIC=true to expose it")
	}

	// 4. 注册底座模块
	api := r.Group("/api/v1")
	platform.RegisterPlatformRoutes(api, database.DB)
	lowcode.InitLowcodeModule(api, database.DB)
	system.InitSystemModule(api, database.DB)
	auth.InitAuthModule(api, database.DB)
	business.InitBusinessModules(api, database.DB)

	// 5. 启动服务器
	port := os.Getenv("PANTHEON_PORT")
	if port == "" {
		port = "8080"
	}
	slog.Info("starting server", "port", port)
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("failed to run server", "error", err)
		os.Exit(1)
	}
}

func shouldExposeMetrics(env string) bool {
	if envFlag("PANTHEON_METRICS_ENABLED") == "false" {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(env), "production") {
		return true
	}
	if strings.TrimSpace(os.Getenv("PANTHEON_METRICS_BEARER_TOKEN")) != "" {
		return true
	}
	return envFlag("PANTHEON_METRICS_PUBLIC") == "true"
}

func metricsAccessMiddleware() gin.HandlerFunc {
	expectedToken := strings.TrimSpace(os.Getenv("PANTHEON_METRICS_BEARER_TOKEN"))
	return func(c *gin.Context) {
		if expectedToken == "" {
			c.Next()
			return
		}
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if header != "Bearer "+expectedToken {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}

func envFlag(name string) string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return "true"
	case "0", "false", "no", "off":
		return "false"
	default:
		return ""
	}
}
