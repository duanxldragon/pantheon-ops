package auth

import (
	"encoding/json"
	"strconv"
	"strings"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/impexp"
	"pantheon-ops/backend/pkg/platformprefs"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *AuthService
}

const csrfGenerateErrorKey = "csrf.generate.error"

func NewAuthHandler(s *AuthService) *AuthHandler {
	return &AuthHandler{service: s}
}

func failOnCSRFCookieError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	common.FailWithError(c, common.CodeError, err, csrfGenerateErrorKey)
	return true
}

func buildAuthSessionResponse(tokenPair *common.TokenPair, userInfo *UserInfoResp) AuthTokenResp {
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

func writeLoginSuccessResponse(c *gin.Context, tokenPair *common.TokenPair, userInfo *UserInfoResp) bool {
	common.SetAccessTokenCookie(c.Writer, tokenPair.AccessToken)
	common.SetRefreshTokenCookie(c.Writer, tokenPair.RefreshToken)
	_, csrfErr := common.SetCSRFCookie(c.Writer)
	if failOnCSRFCookieError(c, csrfErr) {
		return false
	}

	common.Success(c, buildAuthSessionResponse(tokenPair, userInfo))
	return true
}

func writeMFASuccessResponse(c *gin.Context, resp *AuthTokenResp) bool {
	if resp.Token != "" {
		common.SetAccessTokenCookie(c.Writer, resp.Token)
	}
	if resp.RefreshToken != "" {
		common.SetRefreshTokenCookie(c.Writer, resp.RefreshToken)
	}
	_, csrfErr := common.SetCSRFCookie(c.Writer)
	if failOnCSRFCookieError(c, csrfErr) {
		return false
	}

	common.Success(c, resp)
	return true
}

func writeRefreshSuccessResponse(c *gin.Context, tokenPair *common.TokenPair) bool {
	common.SetAccessTokenCookie(c.Writer, tokenPair.AccessToken)
	common.SetRefreshTokenCookie(c.Writer, tokenPair.RefreshToken)
	_, csrfErr := common.SetCSRFCookie(c.Writer)
	if failOnCSRFCookieError(c, csrfErr) {
		return false
	}

	common.Success(c, buildAuthSessionResponse(tokenPair, nil))
	return true
}

func (h *AuthHandler) LoginHandler(c *gin.Context) {
	common.SetAuditMetadata(c, "auth.login.title", common.BusinessOther)

	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	clientInfo := parseClientInfo(userAgent)

	sourceKey := buildLoginSourceKey(ip)
	currentUser, err := h.service.LoginWithSource(&req, sourceKey)
	if err != nil {
		messageKey := common.ResolveErrorMessageKey(err, "auth.login.error")
		h.service.RecordLoginLog(common.GetRequestID(c), strings.TrimSpace(req.Username), ip, clientInfo.Browser, clientInfo.OS, 0, messageKey)
		common.Fail(c, common.CodeUnauthorized, messageKey)
		return
	}

	if h.service.getAuthRuntimePolicy().MFAEnabled {
		challenge, err := h.service.CreateMFAChallenge(currentUser)
		if err != nil {
			messageKey := common.ResolveErrorMessageKey(err, "auth.mfa.challenge.error")
			h.service.RecordLoginLog(common.GetRequestID(c), currentUser.Username, ip, clientInfo.Browser, clientInfo.OS, 0, messageKey)
			common.Fail(c, common.CodeError, messageKey)
			return
		}
		common.Success(c, challenge)
		return
	}

	roles, err := h.service.GetUserRoles(currentUser.ID)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.role.list.error")
		return
	}

	tokenPair, err := h.service.CreateSession(currentUser, roles, ip, userAgent)
	if err != nil {
		messageKey := common.ResolveErrorMessageKey(err, "auth.session.create.error")
		h.service.RecordLoginLog(common.GetRequestID(c), currentUser.Username, ip, clientInfo.Browser, clientInfo.OS, 0, messageKey)
		common.Fail(c, common.CodeError, messageKey)
		return
	}

	userInfo, err := h.service.GetCurrentUserInfo(currentUser.ID)
	if err != nil {
		messageKey := common.ResolveErrorMessageKey(err, "auth.current_user.error")
		h.service.RecordLoginLog(common.GetRequestID(c), currentUser.Username, ip, clientInfo.Browser, clientInfo.OS, 0, messageKey)
		common.Fail(c, common.CodeError, messageKey)
		return
	}

	h.service.RecordLoginLog(common.GetRequestID(c), currentUser.Username, ip, clientInfo.Browser, clientInfo.OS, 1, "auth.loginSuccess")
	if !writeLoginSuccessResponse(c, tokenPair, userInfo) {
		return
	}
}

func (h *AuthHandler) VerifyMFAHandler(c *gin.Context) {
	common.SetAuditMetadata(c, "auth.mfa.verify.title", common.BusinessOther)

	var req MFAVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	clientInfo := parseClientInfo(userAgent)
	resp, err := h.service.VerifyMFAChallenge(&req, ip, userAgent)
	if err != nil {
		messageKey := common.ResolveErrorMessageKey(err, "auth.mfa.verify.error")
		h.service.RecordLoginLog(common.GetRequestID(c), "", ip, clientInfo.Browser, clientInfo.OS, 0, messageKey)
		common.Fail(c, common.CodeUnauthorized, messageKey)
		return
	}

	username := ""
	if resp.User != nil {
		username = resp.User.Username
	}
	h.service.RecordLoginLog(common.GetRequestID(c), username, ip, clientInfo.Browser, clientInfo.OS, 1, "auth.loginSuccess")

	if !writeMFASuccessResponse(c, resp) {
		return
	}
}

func (h *AuthHandler) RefreshTokenHandler(c *gin.Context) {
	common.SetAuditMetadata(c, "auth.session.refresh.title", common.BusinessOther)

	refreshToken := ""
	if cookie, err := c.Cookie(common.CookieRefreshToken); err == nil && cookie != "" {
		refreshToken = cookie
	}
	if refreshToken == "" {
		var req RefreshTokenReq
		if err := c.ShouldBindJSON(&req); err != nil {
			common.Fail(c, common.CodeParamInvalid, "param.invalid")
			return
		}
		refreshToken = req.RefreshToken
	}
	if refreshToken == "" {
		common.Fail(c, common.CodeUnauthorized, "token.invalid")
		return
	}

	claims, err := common.ParseToken(refreshToken, common.TokenTypeRefresh)
	if err != nil {
		common.FailWithError(c, common.CodeUnauthorized, err, "token.invalid")
		return
	}

	tokenPair, err := h.service.RefreshSession(claims, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		common.FailWithError(c, common.CodeUnauthorized, err, "auth.session.refresh.error")
		return
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
	common.SetAuditMetadata(c, "更新平台偏好", common.BusinessUpdate)

	var req UserPlatformPreferenceUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
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
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
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
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	resp, err := h.service.ListLoginLogs(&query)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.login_log.list.error")
		return
	}
	common.Success(c, resp)
}

func (h *AuthHandler) GetSecurityEventList(c *gin.Context) {
	var query SecurityEventQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	resp, err := h.service.ListSecurityEvents(&query)
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
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	var req SecurityEventAcknowledgeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	if err := h.service.AcknowledgeSecurityEvent(
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

func (h *AuthHandler) ExportLoginLogs(c *gin.Context) {
	common.SetAuditMetadata(c, "audit.login_log.export.title", common.BusinessExport)

	var query LoginLogQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	file, err := h.service.ExportLoginLogs(&query)
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
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
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
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
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
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
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
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	deletedCount, err := h.service.BatchDeleteLoginLogs(req.IDs)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.login_log.batch_delete.error")
		return
	}
	common.Success(c, gin.H{"deletedCount": deletedCount})
}
func (h *AuthHandler) GetSessionList(c *gin.Context) {
	var query AdminSessionQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
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
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	token, err := h.service.VerifyPasswordForOperation(common.GetUserID(c), c.GetString("sessionId"), req.Password)
	if err != nil {
		common.FailWithError(c, common.CodeUnauthorized, err, "auth.operation.verify.error")
		return
	}
	common.Success(c, gin.H{"operationToken": token})
}

func (h *AuthHandler) TouchActivity(c *gin.Context) {
	common.SetAuditMetadata(c, "auth.session.touch.title", common.BusinessUpdate)

	if err := h.service.TouchSessionActivity(c.GetString("sessionId"), common.GetUserID(c), c.ClientIP(), c.GetHeader("User-Agent")); err != nil {
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
	common.ClearTokenCookies(c.Writer)
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
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	resp, err := h.service.ListOwnLoginLogs(c.GetString("username"), &query)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "auth.login_log.current_user.error")
		return
	}
	common.Success(c, resp)
}

func buildLoginSourceKey(ip string) string {
	trimmed := strings.TrimSpace(ip)
	if trimmed == "" {
		return "ip:unknown"
	}
	return "ip:" + trimmed
}
