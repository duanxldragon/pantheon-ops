package oidc

import (
	"net/http"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
)

// resolveLoginTenant returns the tenancy the OIDC login flow runs under.
// /auth/* routes resolve tenancy from the login gate, not the business
// middleware, so no tenant context is guaranteed here: fall back to the
// platform-global scope (tenant 0), which is the compat-mode semantic and
// can never widen a tenant's user population.
func resolveLoginTenant(c *gin.Context) uint64 {
	if tc := tenant.FromGin(c); tc != nil {
		return tc.TenantID
	}
	return tenant.PlatformGlobalTenantID
}

// InitiateOIDCLogin handles the OIDC login initiation
func InitiateOIDCLogin(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authURL, err := service.InitiateLogin(c.Request.Context(), c.ClientIP())
		if err != nil {
			// Raw (not common.Fail): app-owned browser-facing login entry.
			// The only protocol element here is the 302 redirect below
			// (RFC 6749 §4.1.1); no IdP or SPA parses this JSON.
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
			// Raw: redirect_uri is reached by the user's browser. IdP-side
			// protocol errors arrive as redirect query params (RFC 6749
			// §4.1.2.1); nothing parses this JSON body.
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Missing code or state parameter",
			})
			return
		}

		// Handle callback
		result, err := service.HandleCallback(c.Request.Context(), code, state, c.ClientIP(), resolveLoginTenant(c))
		if err != nil {
			// Raw: same browser-facing contract as the param check above.
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Authentication failed",
				"details": err.Error(),
			})
			return
		}

		// Raw: interim business payload. Session creation against the existing
		// session service is still pending, so moving this body onto the common
		// envelope belongs to the session-integration change that also updates
		// the consumer.
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
			// Raw: frontend-discovery config, not an OIDC protocol response;
			// any envelope move must land together with its SPA consumer.
			c.JSON(http.StatusOK, gin.H{
				"enabled": false,
			})
			return
		}

		// Raw: same frontend-discovery contract as the disabled branch.
		c.JSON(http.StatusOK, gin.H{
			"enabled":  true,
			"loginURL": "/api/v1/auth/oidc/login",
			"provider": config.ProviderURL,
			"scopes":   config.Scopes,
		})
	}
}
