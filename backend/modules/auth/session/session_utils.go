package session

import (
	"net/netip"
	"strings"
	"time"
	"unicode"
)

// TruncateString returns the string truncated to max length.
func TruncateString(value string, length int) string {
	if len(value) <= length {
		return value
	}
	return value[:length]
}

// NormalizeSessionClientIP returns a normalized client IP string.
func NormalizeSessionClientIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ""
	}
	return addr.String()
}

// NormalizeSessionUserAgent returns a cleaned user agent string.
func NormalizeSessionUserAgent(userAgent string) string {
	userAgent = strings.TrimSpace(userAgent)
	if userAgent == "" {
		return ""
	}
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, userAgent)
	return TruncateString(cleaned, 255)
}

// BuildSessionResp builds a SessionResp from a SystemUserSession.
func BuildSessionResp(item SystemUserSession, currentSessionID string) SessionResp {
	clientInfo := ParseClientInfo(item.UserAgent)
	return SessionResp{
		SessionID:        item.SessionID,
		IsCurrent:        item.SessionID == currentSessionID,
		LastIP:           item.LastIP,
		Browser:          clientInfo.Browser,
		OS:               clientInfo.OS,
		Device:           clientInfo.Device,
		UserAgent:        clientInfo.UserAgent,
		RefreshExpiresAt: item.RefreshExpiresAt.Format(time.RFC3339),
		LastRefreshAt:    FormatNullableTime(item.LastRefreshAt),
		LastActivityAt:   FormatNullableTime(item.LastActivityAt),
		RevokedAt:        FormatNullableTime(item.RevokedAt),
		CreatedAt:        item.CreatedAt.Format(time.RFC3339),
	}
}

// FormatNullableTime formats a time pointer as RFC3339 or returns nil.
func FormatNullableTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(time.RFC3339)
	return &formatted
}
