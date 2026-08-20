package observability

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers observability module routes.
func RegisterRoutes(rg *gin.RouterGroup, handler *Handler) {
	obs := rg.Group("/observability")
	{
		// Metric Sources
		metrics := obs.Group("/metrics")
		{
			sources := metrics.Group("/sources")
			{
				sources.GET("", handler.ListMetricSources)
				sources.POST("", handler.CreateMetricSource)
				sources.GET("/:id", handler.GetMetricSource)
				sources.PUT("/:id", handler.UpdateMetricSource)
				sources.DELETE("/:id", handler.DeleteMetricSource)
			}
		}

		// Alerts
		alerts := obs.Group("/alerts")
		{
			// Alert Rules
			rules := alerts.Group("/rules")
			{
				rules.GET("", handler.ListAlertRules)
				rules.POST("", handler.CreateAlertRule)
				rules.GET("/:id", handler.GetAlertRule)
				rules.PUT("/:id", handler.UpdateAlertRule)
				rules.DELETE("/:id", handler.DeleteAlertRule)
				rules.POST("/validate", handler.ValidatePromQL)
			}

			// Alert Records
			records := alerts.Group("/records")
			{
				records.GET("", handler.ListAlertRecords)
			}

			// Active Alerts
			alerts.GET("/active", handler.GetActiveAlerts)

			// Notification Channels
			channels := alerts.Group("/channels")
			{
				channels.GET("", handler.ListNotificationChannels)
				channels.POST("", handler.CreateNotificationChannel)
				channels.GET("/:id", handler.GetNotificationChannel)
				channels.PUT("/:id", handler.UpdateNotificationChannel)
				channels.DELETE("/:id", handler.DeleteNotificationChannel)
				channels.POST("/:id/test", handler.TestNotificationChannel)
			}
		}
	}
}
