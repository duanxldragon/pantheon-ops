package oidc

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// InitiateOIDCLogin handles the OIDC login initiation
func InitiateOIDCLogin(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authURL, err := service.InitiateLogin(c.Request.Context(), c.ClientIP())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to initiate OIDC login",
			})
			return
		}

		// Redirect to OIDC provider
		c.Redirect(http.StatusFound, authURL)
	}
}

// HandleOIDCCallback handles the OIDC provider callback
func HandleOIDCCallback(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Query("code")
		state := c.Query("state")

		if code == "" || state == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Missing code or state parameter",
			})
			return
		}

		// Handle callback
		result, err := service.HandleCallback(c.Request.Context(), code, state, c.ClientIP())
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Authentication failed",
				"details": err.Error(),
			})
			return
		}

		// 会话创建待接入既有 session service; 当前先返回用户信息。
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"user": gin.H{
				"id":       result.User.ID,
				"username": result.User.Username,
				"email":    result.User.Email,
				"nickname": result.User.Nickname,
			},
			"is_new": result.IsNew,
			"message": func() string {
				if result.IsNew {
					return "Account created successfully"
				}
				return "Login successful"
			}(),
		})

		// 会话服务接入后补充: 设置会话 Cookie 并跳转 dashboard。
	}
}

// GetOIDCConfig returns OIDC configuration info for frontend
func GetOIDCConfig(config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.Enabled {
			c.JSON(http.StatusOK, gin.H{
				"enabled": false,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"enabled":  true,
			"loginURL": "/api/v1/auth/oidc/login",
			"provider": config.ProviderURL,
			"scopes":   config.Scopes,
		})
	}
}
