package platform

import (
	"context"
	"net/http"
	"time"

	"pantheon-platform/backend/pkg/common"
	"pantheon-platform/backend/pkg/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type healthDependency struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type healthResp struct {
	Status       string                      `json:"status"`
	Service      string                      `json:"service"`
	Timestamp    string                      `json:"timestamp"`
	RequestID    string                      `json:"requestId,omitempty"`
	Dependencies map[string]healthDependency `json:"dependencies"`
}

func RegisterHealthRoutes(r *gin.RouterGroup, db *gorm.DB) {
	r.GET("/health", func(c *gin.Context) {
		resp := healthResp{
			Status:    "ok",
			Service:   "pantheon-platform",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: common.GetRequestID(c),
			Dependencies: map[string]healthDependency{
				"database": {Status: "ok"},
				"redis":    {Status: "disabled"},
			},
		}

		if db == nil {
			resp.Status = "degraded"
			resp.Dependencies["database"] = healthDependency{Status: "down", Message: "database.not_initialized"}
		} else if sqlDB, err := db.DB(); err != nil {
			resp.Status = "degraded"
			resp.Dependencies["database"] = healthDependency{Status: "down", Message: err.Error()}
		} else if err := sqlDB.PingContext(c.Request.Context()); err != nil {
			resp.Status = "degraded"
			resp.Dependencies["database"] = healthDependency{Status: "down", Message: err.Error()}
		}

		if database.RDB != nil {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()
			if err := database.RDB.Ping(ctx).Err(); err != nil {
				resp.Status = "degraded"
				resp.Dependencies["redis"] = healthDependency{Status: "down", Message: err.Error()}
			} else {
				resp.Dependencies["redis"] = healthDependency{Status: "ok"}
			}
		}

		statusCode := http.StatusOK
		if resp.Status != "ok" {
			statusCode = http.StatusServiceUnavailable
		}
		common.SuccessWithStatus(c, statusCode, resp)
	})
}
