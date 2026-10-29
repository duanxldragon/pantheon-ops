//nolint:revive // HTTP handler surface is intentionally exported for the auth module.
package login

import (
	"errors"
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	commonhttp "github.com/duanxldragon/pantheon-base/backend/pkg/common/http"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TenantSwitchReq is the body of POST /auth/tenants/switch.
type TenantSwitchReq struct {
	TenantID uint64 `json:"tenantId" binding:"required"`
}

// tenantSummary is the wire shape of one switchable tenant.
type tenantSummary struct {
	ID       uint64 `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Plan     string `json:"plan"`
	Role     string `json:"role"`
	IsActive bool   `json:"isActive"`
}

// ListSwitchableTenants lists the tenants the authenticated subject may switch
// to: exactly its ACTIVE memberships (contract §2.2), narrowed by W4 (W3
// decision A) to memberships where the subject holds at least one role
// binding — a membership without roles is not a usable context and is no
// longer offered. Under compat, or for a subject with no usable memberships,
// the list is empty and the UI keeps the switcher hidden.
//
// This is the read half of the tenant switcher; it never widens scope, it only
// reports what the membership table already grants. `tenant.ListActiveMemberships`
// itself stays unfiltered — /auth/tenants/current uses it to report the caller's own
// membership role honestly, even when that membership is not loginable.
func (h *AuthHandler) ListSwitchableTenants(c *gin.Context) {
	userID := common.GetUserID(c)
	if userID == 0 {
		common.Fail(c, common.CodeUnauthorized, msgTokenInvalid)
		return
	}

	memberships, err := tenant.ListActiveMemberships(h.service.db, userID)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "tenant.switchable.error")
		return
	}
	current := tenant.FromGin(c)

	ids := make([]uint64, 0, len(memberships))
	for _, m := range memberships {
		if !tenant.HasRoleBindingInTenant(h.service.db, userID, m.TenantID) {
			continue
		}
		ids = append(ids, m.TenantID)
	}
	summaries := h.loadTenantSummaries(ids)

	items := make([]tenantSummary, 0, len(memberships))
	for _, m := range memberships {
		summary, ok := summaries[m.TenantID]
		if !ok {
			continue
		}
		summary.Role = m.Role
		summary.IsActive = current != nil && current.TenantID == m.TenantID
		items = append(items, summary)
	}
	common.Success(c, gin.H{"items": items})
}

// GetCurrentTenant reports the tenant context the request resolved to
// (contract §3.1). It is the authoritative answer to "which tenant am I
// looking at right now", including which trusted source resolved it.
func (h *AuthHandler) GetCurrentTenant(c *gin.Context) {
	ctx := tenant.FromGin(c)
	if ctx == nil {
		common.Success(c, gin.H{"mode": tenant.ModeCompat, "tenantId": 0, "resolvedBy": ""})
		return
	}

	resp := gin.H{
		"mode":       ctx.Mode,
		"tenantId":   ctx.TenantID,
		"resolvedBy": ctx.ResolvedBy,
	}
	if ctx.TenantID > 0 {
		if summary, ok := h.loadTenantSummaries([]uint64{ctx.TenantID})[ctx.TenantID]; ok {
			// Report the caller's own membership role for the resolved tenant so
			// this endpoint agrees with /auth/tenants/switchable (contract §2.2).
			// Both must answer with the same role/isActive for the same tenant;
			// reporting an empty role here would mislead any UI that renders the
			// current tenant's controls from it.
			h.attachCallerMembershipRole(c, &summary, ctx.TenantID)
			resp["tenant"] = summary
		}
	}
	common.Success(c, resp)
}

// attachCallerMembershipRole stamps the caller's own membership role onto the
// summary of the tenant the request resolved to. A caller with no membership in
// that tenant keeps the summary untouched, which is the correct answer for a
// platform operator inspecting a tenant it does not belong to.
func (h *AuthHandler) attachCallerMembershipRole(c *gin.Context, summary *tenantSummary, tenantID uint64) {
	userID := common.GetUserID(c)
	if userID == 0 {
		return
	}
	memberships, err := tenant.ListActiveMemberships(h.service.db, userID)
	if err != nil {
		return
	}
	for _, m := range memberships {
		if m.TenantID == tenantID {
			summary.Role = m.Role
			summary.IsActive = true
			return
		}
	}
}

// SwitchTenant re-issues the caller's session bound to another tenant
// (contract §2.2). The target must be an active tenant the subject is an
// active member of; the session id is preserved and the access/refresh pair is
// re-signed with the new claim, so every subsequent request resolves to the
// new tenant without a re-login.
func (h *AuthHandler) SwitchTenant(c *gin.Context) {
	var req TenantSwitchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if req.TenantID == 0 {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}

	userID := common.GetUserID(c)
	sessionID := c.GetString("sessionId")
	if userID == 0 || strings.TrimSpace(sessionID) == "" {
		common.Fail(c, common.CodeUnauthorized, msgTokenInvalid)
		return
	}

	pair, err := h.service.sessionSvc.SwitchSessionTenant(c.Request.Context(), sessionID, userID, req.TenantID)
	if err != nil {
		common.FailWithError(c, tenantSwitchErrorCode(err), err, tenantSwitchMessageKey(err))
		return
	}

	commonhttp.SetAccessTokenCookie(c.Writer, pair.AccessToken)
	commonhttp.SetRefreshTokenCookie(c.Writer, pair.RefreshToken)
	csrfToken, csrfErr := commonhttp.SetCSRFCookie(c.Writer)
	if failOnCSRFCookieError(c, csrfErr) {
		return
	}

	resp := gin.H{
		"tenantId":  req.TenantID,
		"sessionId": pair.SessionID,
		"csrfToken": csrfToken,
	}
	if summary, ok := h.loadTenantSummaries([]uint64{req.TenantID})[req.TenantID]; ok {
		resp["tenant"] = summary
	}
	common.Success(c, resp)
}

// loadTenantSummaries reads the display fields of the given tenants. A missing
// row is simply absent from the map (the caller decides whether that is fatal).
func (h *AuthHandler) loadTenantSummaries(ids []uint64) map[uint64]tenantSummary {
	result := make(map[uint64]tenantSummary, len(ids))
	if h.service.db == nil || len(ids) == 0 || !h.service.db.Migrator().HasTable("tenants") {
		return result
	}
	var rows []struct {
		ID     uint64 `gorm:"column:id"`
		Code   string `gorm:"column:code"`
		Name   string `gorm:"column:name"`
		Status string `gorm:"column:status"`
		Plan   string `gorm:"column:plan"`
	}
	if err := h.service.db.Table("tenants").
		Select("id, code, name, status, plan").
		Where("id IN ?", ids).
		Scan(&rows).Error; err != nil {
		return result
	}
	for _, row := range rows {
		result[row.ID] = tenantSummary{
			ID: row.ID, Code: row.Code, Name: row.Name, Status: row.Status, Plan: row.Plan,
		}
	}
	return result
}

// tenantSwitchErrorCode maps a switch failure to the envelope code (contract §5).
func tenantSwitchErrorCode(err error) int {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return common.CodeUnauthorized
	case errors.Is(err, tenant.ErrTenantForbidden),
		errors.Is(err, tenant.ErrTenantSuspended),
		errors.Is(err, tenant.ErrTenantArchived):
		return common.CodeForbidden
	default:
		// Same-tenant switches and revoked sessions are client-visible
		// rejections, not server faults; keep their envelope class.
		return common.CodeForError(err)
	}
}

// tenantSwitchMessageKey maps a switch failure to its i18n message key.
func tenantSwitchMessageKey(err error) string {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return msgTokenInvalid
	case errors.Is(err, tenant.ErrTenantSuspended):
		return "tenant.suspended"
	case errors.Is(err, tenant.ErrTenantArchived):
		return "tenant.archived"
	case errors.Is(err, tenant.ErrTenantForbidden):
		return "tenant.forbidden"
	default:
		return "tenant.switch.error"
	}
}
