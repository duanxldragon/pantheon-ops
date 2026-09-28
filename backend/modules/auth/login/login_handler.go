//nolint:revive // HTTP handler surface is intentionally exported for the auth module.
package login

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	authsessiondomain "github.com/duanxldragon/pantheon-base/backend/modules/auth/session"
	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	commonhttp "github.com/duanxldragon/pantheon-base/backend/pkg/common/http"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/impexp"
	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"
	"github.com/duanxldragon/pantheon-base/backend/pkg/platformprefs"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AuthHandler struct {
	service                  *Runtime
	invalidateTokenAuthCache func(string)
}

const csrfGenerateErrorKey = "csrf.generate.error"
const msgParamInvalid = "param.invalid"
const headerUserAgent = "User-Agent"

func NewAuthHandler(s *Runtime, invalidators ...func(string)) *AuthHandler {
	h := &AuthHandler{service: s}
	if len(invalidators) > 0 {
		h.invalidateTokenAuthCache = invalidators[0]
	}
	return h
}

func failOnCSRFCookieError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	common.FailWithError(c, common.CodeError, err, csrfGenerateErrorKey)
	return true
}

func buildAuthSessionResponse(tokenPair *authtoken.Pair, userInfo *UserInfoResp) AuthTokenResp {
	return AuthTokenResp{
		Token:            tokenPair.AccessToken,
		AccessToken:      tokenPair.AccessToken,
		RefreshToken:     tokenPair.RefreshToken,
		TokenType:        tokenPair.TokenType,
		AccessExpiresAt:  tokenPair.AccessExpiresAt.Format("2006-01-02 15:04:05"),
		RefreshExpiresAt: tokenPair.RefreshExpiresAt.Format("2006-01-02 15:04:05"),
		SessionID:        tokenPair.SessionID,
		User:             userInfo,
	}
}

func writeLoginSuccessResponse(c *gin.Context, tokenPair *authtoken.Pair, userInfo *UserInfoResp) bool {
	commonhttp.SetAccessTokenCookie(c.Writer, tokenPair.AccessToken)
	commonhttp.SetRefreshTokenCookie(c.Writer, tokenPair.RefreshToken)
	_, csrfErr := commonhttp.SetCSRFCookie(c.Writer)
	if failOnCSRFCookieError(c, csrfErr) {
		return false
	}

	common.Success(c, buildAuthSessionResponse(tokenPair, userInfo))
	return true
}

func writeMFASuccessResponse(c *gin.Context, resp *AuthTokenResp) bool {
	if resp.Token != "" {
		commonhttp.SetAccessTokenCookie(c.Writer, resp.Token)
	}
	if resp.RefreshToken != "" {
		commonhttp.SetRefreshTokenCookie(c.Writer, resp.RefreshToken)
	}
	_, csrfErr := commonhttp.SetCSRFCookie(c.Writer)
	if failOnCSRFCookieError(c, csrfErr) {
		return false
	}

	common.Success(c, resp)
	return true
}

func writeRefreshSuccessResponse(c *gin.Context, tokenPair *authtoken.Pair) bool {
	commonhttp.SetAccessTokenCookie(c.Writer, tokenPair.AccessToken)
	commonhttp.SetRefreshTokenCookie(c.Writer, tokenPair.RefreshToken)
	_, csrfErr := commonhttp.SetCSRFCookie(c.Writer)
	if failOnCSRFCookieError(c, csrfErr) {
		return false
	}

	common.Success(c, buildAuthSessionResponse(tokenPair, nil))
	return true
}

func (h *AuthHandler) LoginHandler(c *gin.Context) {
	common.SetAuditMetadata(c, "auth.login.audit.title", common.BusinessOther)

	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	service := h.service
	if req.TenantId > 0 {
		service = h.service.WithTenantContext(&tenant.Context{TenantID: req.TenantId, Mode: tenant.ModeMulti, ResolvedBy: "login-choice"})
	}

	ip := c.ClientIP()
	userAgent := c.GetHeader(headerUserAgent)
	clientInfo := authsessiondomain.ParseClientInfo(userAgent)

	sourceKey := buildLoginSourceKey(ip)
	currentUser, err := service.LoginWithSource(&req, sourceKey)
	if err != nil {
		messageKey := common.ResolveErrorMessageKey(err, "auth.login.error")
		service.RecordLoginLog(common.GetRequestID(c), strings.TrimSpace(req.Username), ip, clientInfo.Browser, clientInfo.OS, 0, messageKey)
		common.Fail(c, common.CodeUnauthorized, messageKey)
		return
	}

	tenantClaim, tenantErr := service.resolveLoginTenantClaim(currentUser.ID, req.TenantId)
	if tenantErr != nil {
		if errors.Is(tenantErr, tenant.ErrTenantForbidden) && req.TenantId == 0 {
			if candidates := service.ListLoginTenantCandidates(c.Request.Context(), currentUser.ID); len(candidates) > 1 {
				common.Success(c, gin.H{
					"tenantSelectionRequired": true,
					"tenantCandidates":        candidates,
				})
				return
			}
		}
		messageKey := common.ResolveErrorMessageKey(tenantErr, "auth.login.error")
		service.RecordLoginLog(common.GetRequestID(c), currentUser.Username, ip, clientInfo.Browser, clientInfo.OS, 0, messageKey)
		common.Fail(c, common.CodeForbidden, messageKey)
		return
	}

	if service.getAuthRuntimePolicy().MFAEnabled {
		challenge, err := service.CreateMFAChallengeForTenant(currentUser, tenantClaim)
		if err != nil {
			messageKey := common.ResolveErrorMessageKey(err, "auth.mfa.challenge.error")
			service.RecordLoginLog(common.GetRequestID(c), currentUser.Username, ip, clientInfo.Browser, clientInfo.OS, 0, messageKey)
			common.Fail(c, common.CodeError, messageKey)
			return
		}
		common.Success(c, challenge)
		return
	}

	roles, err := service.GetUserRoles(currentUser.ID)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.role.list.error")
		return
	}

	tokenPair, err := service.CreateSessionForTenantWithContext(c.Request.Context(), currentUser.ID, roles, ip, userAgent, req.TenantId)
	if err != nil {
		// Tenant gates surface their own i18n keys (contract §5); all other
		// failures keep the generic session-creation message.
		messageKey := common.ResolveErrorMessageKey(err, "auth.session.create.error")
		if tenant.IsTenantGateError(err) {
			messageKey = common.ErrMessage(err)
		}
		service.RecordLoginLog(common.GetRequestID(c), currentUser.Username, ip, clientInfo.Browser, clientInfo.OS, 0, messageKey)
		common.Fail(c, common.CodeForbidden, messageKey)
		return
	}

	userInfo, err := service.GetCurrentUserInfo(currentUser.ID)
	if err != nil {
		messageKey := common.ResolveErrorMessageKey(err, "auth.current_user.error")
		service.RecordLoginLog(common.GetRequestID(c), currentUser.Username, ip, clientInfo.Browser, clientInfo.OS, 0, messageKey)
		common.Fail(c, common.CodeError, messageKey)
		return
	}

	service.RecordLoginLog(common.GetRequestID(c), currentUser.Username, ip, clientInfo.Browser, clientInfo.OS, 1, "auth.loginSuccess")
	if !writeLoginSuccessResponse(c, tokenPair, userInfo) {
		return
	}
}

func (h *AuthHandler) VerifyMFAHandler(c *gin.Context) {
	common.SetAuditMetadata(c, "auth.mfa.verify.title", common.BusinessOther)

	var req MFAVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	service := h.service
	if req.TenantId > 0 {
		service = h.service.WithTenantContext(&tenant.Context{TenantID: req.TenantId, Mode: tenant.ModeMulti, ResolvedBy: "mfa-choice"})
	}

	ip := c.ClientIP()
	userAgent := c.GetHeader(headerUserAgent)
	clientInfo := authsessiondomain.ParseClientInfo(userAgent)
	resp, err := service.VerifyMFAChallengeWithContext(c.Request.Context(), &req, ip, userAgent)
	if err != nil {
		// Tenant gates (explicit choice rejected at MFA-final session issuance)
		// surface their contract §5 keys; everything else keeps MFA messages.
		messageKey := common.ResolveErrorMessageKey(err, "auth.mfa.verify.error")
		if tenant.IsTenantGateError(err) {
			messageKey = common.ErrMessage(err)
		}
		service.RecordLoginLog(common.GetRequestID(c), "", ip, clientInfo.Browser, clientInfo.OS, 0, messageKey)
		common.Fail(c, common.CodeForbidden, messageKey)
		return
	}

	username := ""
	if resp.User != nil {
		username = resp.User.Username
	}
	service.RecordLoginLog(common.GetRequestID(c), username, ip, clientInfo.Browser, clientInfo.OS, 1, "auth.loginSuccess")

	if !writeMFASuccessResponse(c, resp) {
		return
	}
}

// GetLoginTenantCandidates lists the tenants the authenticated subject may
// explicitly log into (multi-membership picker, contract §3.1). Empty under
// compat or on error — the UI keeps the picker hidden (fail closed).
func (h *AuthHandler) GetLoginTenantCandidates(c *gin.Context) {
	candidates := h.service.ListLoginTenantCandidates(c.Request.Context(), common.GetUserID(c))
	common.Success(c, gin.H{"items": candidates})
}

func (h *AuthHandler) RefreshTokenHandler(c *gin.Context) {
	common.SetAuditMetadata(c, "auth.session.refresh.title", common.BusinessOther)

	refreshToken := ""
	if cookie, err := c.Cookie(commonhttp.CookieRefreshToken); err == nil && cookie != "" {
		refreshToken = cookie
	}
	if refreshToken == "" {
		var req RefreshTokenReq
		if err := c.ShouldBindJSON(&req); err != nil {
			common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
			return
		}
		refreshToken = req.RefreshToken
	}
	if refreshToken == "" {
		common.Fail(c, common.CodeUnauthorized, "token.invalid")
		return
	}

	refreshClaims, err := parseRefreshTokenWithContext(c.Request.Context(), refreshToken)
	if err != nil {
		common.FailWithError(c, common.CodeUnauthorized, err, "token.invalid")
		return
	}

	tokenPair, err := h.service.RefreshSessionWithContext(c.Request.Context(), refreshClaims.SessionID, refreshClaims.UserID, c.ClientIP(), c.GetHeader(headerUserAgent))
	if err != nil {
		common.FailWithError(c, common.CodeUnauthorized, err, "auth.session.refresh.error")
		return
	}

	// 刷新令牌轮换：新 pair 签发成功后立即失效旧 refresh token，防止旧 token 在剩余 TTL 内被重放。
	if err := authtoken.DeleteRefresh(c.Request.Context(), database.RDB, refreshToken); err != nil {
		logging.Warn("revoke rotated refresh token failed",
			zap.String("session_id", refreshClaims.SessionID), zap.Error(err))
	}

	if !writeRefreshSuccessResponse(c, tokenPair) {
		return
	}
}
func (h *AuthHandler) GetCurrentUserInfo(c *gin.Context) {
	resp, err := h.service.GetCurrentUserInfo(common.GetUserID(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.current_user.error")
		return
	}
	common.Success(c, resp)
}

func (h *AuthHandler) UpdateCurrentUserPreferences(c *gin.Context) {
	common.SetAuditMetadata(c, "auth.preferences.update.title", common.BusinessUpdate)

	var req UserPlatformPreferenceUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}

	result, err := h.service.UpdateCurrentUserPreferences(common.GetUserID(c), &req)
	if err != nil {
		messageKey := common.ResolveErrorMessageKey(err, "auth.preference.update.error")
		common.SetAuditStatus(c, 2)
		common.SetAuditErrorMsg(c, messageKey)
		common.Fail(c, common.CodeError, messageKey)
		return
	}
	common.SetAuditParam(c, buildPreferenceAuditPayload(result.Previous, result.Current))
	common.SetAuditResult(c, buildPreferenceAuditResult(result.User, result.Previous, result.Current))
	common.Success(c, result.User)
}

func (h *AuthHandler) UpdatePassword(c *gin.Context) {
	common.SetAuditMetadata(c, "auth.password.update.title", common.BusinessUpdate)

	var req PasswordUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if err := h.service.UpdatePassword(common.GetUserID(c), c.GetString("sessionId"), &req); err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.password.update.error")
		return
	}
	common.Success(c, gin.H{"passwordUpdated": true})
}
func (h *AuthHandler) GetLoginLogList(c *gin.Context) {
	var query LoginLogQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.service.WithTenantContext(tenant.FromGin(c)).ListLoginLogs(&query)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.login_log.list.error")
		return
	}
	common.Success(c, resp)
}

func (h *AuthHandler) GetSecurityEventList(c *gin.Context) {
	var query SecurityEventQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.service.WithTenantContext(tenant.FromGin(c)).ListSecurityEvents(&query)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.security_event.list.error")
		return
	}
	common.Success(c, resp)
}

func (h *AuthHandler) AcknowledgeSecurityEvent(c *gin.Context) {
	common.SetAuditMetadata(c, "auth.security_event.acknowledge.title", common.BusinessUpdate)

	eventID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}

	var req SecurityEventAcknowledgeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}

	if err := h.service.WithTenantContext(tenant.FromGin(c)).AcknowledgeSecurityEvent(
		eventID,
		common.GetUserID(c),
		c.GetString("username"),
		req.AcknowledgementNote,
	); err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.security_event.acknowledge.error")
		return
	}
	common.Success(c, gin.H{"acknowledged": true})
}

func (h *AuthHandler) BatchAcknowledgeSecurityEvents(c *gin.Context) {
	common.SetAuditMetadata(c, "auth.security_event.batch_acknowledge.title", common.BusinessUpdate)

	var req SecurityEventBatchAcknowledgeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}

	acknowledgedCount, err := h.service.WithTenantContext(tenant.FromGin(c)).BatchAcknowledgeSecurityEvents(
		req.IDs,
		common.GetUserID(c),
		c.GetString("username"),
		req.AcknowledgementNote,
	)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.security_event.acknowledge.error")
		return
	}
	common.Success(c, SecurityEventBatchAcknowledgeResp{AcknowledgedCount: acknowledgedCount})
}

func (h *AuthHandler) CleanupSecurityEvents(c *gin.Context) {
	common.SetAuditMetadata(c, "auth.security_event.cleanup.title", common.BusinessClean)

	var req SecurityEventCleanupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}

	clearedCount, err := h.service.WithTenantContext(tenant.FromGin(c)).CleanupSecurityEvents(req.RetentionDays, req.StartedAt, req.EndedAt)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.security_event.cleanup.error")
		return
	}
	common.Success(c, SecurityEventCleanupResp{ClearedCount: clearedCount})
}

func (h *AuthHandler) ExportLoginLogs(c *gin.Context) {
	common.SetAuditMetadata(c, "audit.login_log.export.title", common.BusinessExport)

	var query LoginLogQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	file, err := h.service.WithTenantContext(tenant.FromGin(c)).ExportLoginLogs(&query)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.login_log.export.error")
		return
	}
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.login_log.export.error")
	}
}

func buildPreferenceAuditPayload(previous, current *platformprefs.PlatformPreference) string {
	return marshalAuthAuditPayload(gin.H{
		"scope":  "platform.shell.preferences",
		"before": previous,
		"after":  current,
	})
}

func buildPreferenceAuditResult(resp *UserInfoResp, previous *platformprefs.PlatformPreference, current *platformprefs.PlatformPreference) string {
	return marshalAuthAuditPayload(gin.H{
		"userId":      resp.ID,
		"username":    resp.Username,
		"preferences": current,
		"changed":     !preferencesEqual(previous, current),
	})
}

func preferencesEqual(previous, current *platformprefs.PlatformPreference) bool {
	if previous == nil && current == nil {
		return true
	}
	if previous == nil || current == nil {
		return false
	}
	return previous.Theme == current.Theme &&
		previous.Language == current.Language &&
		previous.LayoutMode == current.LayoutMode &&
		previous.DensityMode == current.DensityMode
}

func marshalAuthAuditPayload(payload any) string {
	data, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func (h *AuthHandler) CleanupLoginLogs(c *gin.Context) {
	common.SetAuditMetadata(c, "audit.login_log.cleanup.title", common.BusinessClean)

	var req LoginLogCleanupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}

	clearedCount, err := h.service.CleanupLoginLogs(req.RetentionDays, req.StartedAt, req.EndedAt)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.login_log.cleanup.error")
		return
	}
	common.Success(c, LoginLogCleanupResp{ClearedCount: clearedCount})
}

func (h *AuthHandler) CleanupHistoricSessions(c *gin.Context) {
	common.SetAuditMetadata(c, "auth.session.cleanup.title", common.BusinessClean)

	var req SessionCleanupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}

	clearedCount, err := h.service.CleanupHistoricSessions(req.RetentionDays, req.StartedAt, req.EndedAt)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.session.cleanup.error")
		return
	}
	common.Success(c, SessionCleanupResp{ClearedCount: clearedCount})
}

func (h *AuthHandler) BatchRevokeSessions(c *gin.Context) {
	common.SetAuditMetadata(c, "auth.session.revoke.title", common.BusinessForce)

	var req SessionBatchRevokeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}

	revokedCount, err := h.service.BatchRevokeSessions(c.GetString("sessionId"), req.SessionIDs)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.session.revoke.error")
		return
	}
	common.Success(c, gin.H{"revokedCount": revokedCount})
}

func (h *AuthHandler) BatchDeleteLoginLogs(c *gin.Context) {
	common.SetAuditMetadata(c, "audit.login_log.batch_delete.title", common.BusinessDelete)

	var req LoginLogBatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}

	deletedCount, err := h.service.WithTenantContext(tenant.FromGin(c)).BatchDeleteLoginLogs(req.IDs)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.login_log.batch_delete.error")
		return
	}
	common.Success(c, gin.H{"deletedCount": deletedCount})
}
func (h *AuthHandler) GetSessionList(c *gin.Context) {
	var query AdminSessionQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.service.ListAllSessions(&query)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.session.list.error")
		return
	}
	common.Success(c, resp)
}
func (h *AuthHandler) RevokeAnySession(c *gin.Context) {
	common.SetAuditMetadata(c, "auth.session.revoke.title", common.BusinessForce)

	if err := h.service.RevokeAnySession(c.GetString("sessionId"), strings.TrimSpace(c.Param("id"))); err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.session.revoke.error")
		return
	}
	common.Success(c, gin.H{"revoked": true})
}
func (h *AuthHandler) GetSecurityOverview(c *gin.Context) {
	resp, err := h.service.GetSecurityOverview(common.GetUserID(c), c.GetString("username"), c.GetString("sessionId"))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.security.overview.error")
		return
	}
	common.Success(c, resp)
}

func (h *AuthHandler) VerifyOperationPassword(c *gin.Context) {
	common.SetAuditMetadata(c, "auth.operation.verify.title", common.BusinessOther)

	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	token, err := h.service.VerifyPasswordForOperationWithContext(c.Request.Context(), common.GetUserID(c), c.GetString("sessionId"), req.Password)
	if err != nil {
		common.FailWithError(c, common.CodeUnauthorized, err, "auth.operation.verify.error")
		return
	}
	common.Success(c, gin.H{"operationToken": token})
}

func (h *AuthHandler) TouchActivity(c *gin.Context) {
	common.SetAuditMetadata(c, "auth.session.touch.title", common.BusinessUpdate)

	if err := h.service.TouchSessionActivity(c.GetString("sessionId"), common.GetUserID(c), c.ClientIP(), c.GetHeader(headerUserAgent)); err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.session.touch.error")
		return
	}
	common.Success(c, gin.H{"touched": true})
}

func (h *AuthHandler) LogoutHandler(c *gin.Context) {
	common.SetAuditMetadata(c, "auth.logout.title", common.BusinessForce)

	if err := h.service.RevokeSession(c.GetString("sessionId")); err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.logout.error")
		return
	}
	// Revoke the current opaque access token as well as the DB session. Without
	// this step a token cached by TokenAuthMiddleware could remain usable after
	// logout until its normal TTL elapsed.
	if accessToken := commonhttp.ExtractAccessToken(c.Request); accessToken != "" {
		if h.invalidateTokenAuthCache != nil {
			h.invalidateTokenAuthCache(accessToken)
		}
		if err := authtoken.DeleteSession(c.Request.Context(), database.RDB, accessToken); err != nil {
			logging.Warn("delete access token after logout failed",
				zap.String("session_id", c.GetString("sessionId")),
				zap.Error(err))
		}
	}
	commonhttp.ClearTokenCookies(c.Writer)
	common.Success(c, gin.H{"loggedOut": true})
}
func (h *AuthHandler) GetSessions(c *gin.Context) {
	resp, err := h.service.ListSessions(common.GetUserID(c), c.GetString("sessionId"))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.session.current_list.error")
		return
	}
	common.Success(c, resp)
}
func (h *AuthHandler) RevokeSession(c *gin.Context) {
	common.SetAuditMetadata(c, "auth.session.revoke_self.title", common.BusinessForce)

	if err := h.service.RevokeSession(strings.TrimSpace(c.Param("id"))); err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.session.revoke_self.error")
		return
	}
	common.Success(c, gin.H{"revoked": true})
}
func (h *AuthHandler) GetOwnLoginLogs(c *gin.Context) {
	var query LoginLogQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.service.WithTenantContext(tenant.FromGin(c)).ListOwnLoginLogs(c.GetString("username"), &query)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.login_log.current_user.error")
		return
	}
	common.Success(c, resp)
}

const loginSourceKeyUnknownIP = "ip:unknown"

func buildLoginSourceKey(ip string) string {
	trimmed := strings.TrimSpace(ip)
	if trimmed == "" {
		return loginSourceKeyUnknownIP
	}
	return "ip:" + trimmed
}

func parseRefreshTokenWithContext(ctx context.Context, refreshToken string) (*TokenRefreshClaims, error) {
	if refreshToken == "" {
		return nil, errors.New("token.invalid")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	userID, sessionID, err := authtoken.ValidateRefresh(ctx, database.RDB, refreshToken)
	if err != nil {
		return nil, err
	}

	return &TokenRefreshClaims{
		UserID:    userID,
		SessionID: sessionID,
		ID:        refreshToken,
	}, nil
}
