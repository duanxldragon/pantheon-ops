package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"pantheon-base/internal/middleware"
	"pantheon-base/modules/auth"
	"pantheon-base/modules/business"
	"pantheon-base/modules/lowcode"
	"pantheon-base/modules/platform"
	"pantheon-base/modules/system"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/database"
	"pantheon-base/pkg/logging"
	"pantheon-base/pkg/telemetry"
	"pantheon-base/pkg/version"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"
)

func main() {
	env := initEnvironment()
	defer logging.Sync()

	logging.Info("Starting Pantheon Base",
		zap.String("version", version.Version),
		zap.String("environment", env),
	)

	if shutdownTracer := initTelemetry(); shutdownTracer != nil {
		defer shutdownTracer()
	}

	initInfrastructure()
	r := buildRouter(env)
	runServer(r)
}

func initEnvironment() string {
	env := os.Getenv("PANTHEON_ENV")
	if env == "" {
		env = "development"
	}
	if err := logging.InitLogger(env); err != nil {
		slog.Error("Failed to initialize logger", "error", err)
		os.Exit(1)
	}
	return env
}

func initTelemetry() func() {
	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otlpEndpoint == "" {
		return nil
	}
	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tp, err := telemetry.InitTracer("pantheon-base", otlpEndpoint)
	if err != nil {
		logging.Error("Failed to initialize tracer", zap.Error(err))
		return nil
	}
	logging.Info("OpenTelemetry tracer initialized", zap.String("endpoint", otlpEndpoint))
	return func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			logging.Error("Error shutting down tracer", zap.Error(err))
		}
	}
}

func initInfrastructure() {
	common.InitLocationService()
	if err := common.InitSecurityConfig(); err != nil {
		logging.Error("Security configuration invalid", zap.Error(err))
		os.Exit(1)
	}

	dsn := os.Getenv("PANTHEON_DSN")
	if dsn == "" {
		logging.Fatal("PANTHEON_DSN is required")
	}
	database.InitDB(dsn)
	runDatabaseMigrations(dsn)

	if redisAddr := os.Getenv("PANTHEON_REDIS_ADDR"); redisAddr != "" {
		database.InitRedis(redisAddr, os.Getenv("PANTHEON_REDIS_PASSWORD"), 0)
	}
	database.InitCasbin(database.DB)
}

func runDatabaseMigrations(dsn string) {
	if database.ShouldAutoMigrate() {
		slog.Info("PANTHEON_AUTO_MIGRATE=true: using GORM AutoMigrate (dev mode)")
		return
	}
	if err := database.RunMigrations(dsn); err != nil {
		slog.Error("database migration failed", "error", err)
		os.Exit(1)
	}
}

func buildRouter(env string) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.SecurityHeadersMiddleware())
	r.Use(middleware.CSPMiddleware())
	r.Use(middleware.BodySizeLimit(middleware.DefaultMaxBodyBytes))
	r.Use(middleware.CORSMiddleware())
	r.Use(otelgin.Middleware("pantheon-base"))
	r.Use(middleware.PrometheusMiddleware())
	r.Use(middleware.RequestContextMiddleware(), middleware.OperationLogMiddleware(database.DB))
	r.Use(middleware.CSRFMiddleware())
	registerMetricsRoute(r, env)
	registerAPIRoutes(r)
	return r
}

func registerMetricsRoute(r *gin.Engine, env string) {
	if !shouldExposeMetrics(env) {
		logging.Warn("Prometheus metrics endpoint disabled; set PANTHEON_METRICS_BEARER_TOKEN or PANTHEON_METRICS_PUBLIC=true to expose it")
		return
	}
	r.GET("/metrics", metricsAccessMiddleware(), gin.WrapH(promhttp.Handler()))
}

func registerAPIRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	if envFlag("PANTHEON_API_RATE_LIMIT_ENABLED") != envFlagFalse {
		api.Use(middleware.RateLimiter(middleware.RateLimiterConfig{
			MaxRequests: envIntDefault("PANTHEON_API_RATE_LIMIT_MAX", 6000),
			Window:      time.Duration(envIntDefault("PANTHEON_API_RATE_LIMIT_WINDOW_SECONDS", 60)) * time.Second,
			Store:       middleware.NewRedisRateLimitStore(),
		}))
	}
	platform.RegisterPlatformRoutes(api, database.DB)
	lowcode.InitLowcodeModule(api, database.DB)
	system.InitSystemModule(api, database.DB)
	auth.InitAuthModule(api, database.DB)
	business.InitBusinessModules(api, database.DB)
}

func runServer(r *gin.Engine) {
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
	// 优雅停机：SIGINT/SIGTERM 后先停止接收连接并等在途请求完成，
	// 再排空操作日志异步队列，避免部署时丢请求、丢审计日志。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("failed to run server", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received; draining")

	shutdownTimeout := time.Duration(envIntDefault("PANTHEON_SHUTDOWN_TIMEOUT_SECONDS", 15)) * time.Second
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("http server shutdown error", "error", err)
	}
	middleware.ShutdownOperationLog(shutdownCtx)
	slog.Info("server stopped")
}

func envIntDefault(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func shouldExposeMetrics(env string) bool {
	if envFlag("PANTHEON_METRICS_ENABLED") == envFlagFalse {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(env), "production") {
		return true
	}
	if strings.TrimSpace(os.Getenv("PANTHEON_METRICS_BEARER_TOKEN")) != "" {
		return true
	}
	return envFlag("PANTHEON_METRICS_PUBLIC") == envFlagTrue
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

const (
	envFlagTrue  = "true"
	envFlagFalse = "false"
)

func envFlag(name string) string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", envFlagTrue, "yes", "on":
		return envFlagTrue
	case "0", envFlagFalse, "no", "off":
		return envFlagFalse
	default:
		return ""
	}
}
