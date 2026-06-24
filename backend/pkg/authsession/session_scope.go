package authsession

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

const DefaultSessionIdleMinutes = 30
const DefaultMaxActiveSessionsPerUser = 1
const DefaultSessionRetentionDays = 90

func ApplyActiveScope(db *gorm.DB, alias string, now time.Time, idleMinutes int) *gorm.DB {
	prefix := columnPrefix(alias)
	scoped := db.
		Where(fmt.Sprintf("%srevoked_at IS NULL", prefix)).
		Where(fmt.Sprintf("%srefresh_expires_at > ?", prefix), now)
	if idleMinutes > 0 {
		scoped = scoped.Where(lastSeenExpr(prefix)+" >= ?", now.Add(-time.Duration(idleMinutes)*time.Minute))
	}
	return scoped
}

func CleanupInactiveSessions(db *gorm.DB, now time.Time, idleMinutes int) error {
	if db == nil {
		return nil
	}

	condition := "revoked_at IS NULL AND (refresh_expires_at <= ?"
	args := []any{now}
	if idleMinutes > 0 {
		condition += " OR " + lastSeenExpr("") + " < ?"
		args = append(args, now.Add(-time.Duration(idleMinutes)*time.Minute))
	}
	condition += ")"

	return db.Table("system_user_session").
		Where(condition, args...).
		Update("revoked_at", now).Error
}

func CleanupUserOverflowSessions(db *gorm.DB, userID uint64, now time.Time, idleMinutes int, maxActiveSessions int) error {
	if db == nil || userID == 0 || maxActiveSessions < 0 {
		return nil
	}
	if maxActiveSessions == 0 {
		return db.Table("system_user_session").
			Where("user_id = ? AND revoked_at IS NULL", userID).
			Update("revoked_at", now).Error
	}

	var keepIDs []string
	query := ApplyActiveScope(db.Table("system_user_session"), "", now, idleMinutes).
		Select("session_id").
		Where("user_id = ?", userID).
		Order(lastSeenExpr("") + " desc").
		Order("created_at desc").
		Limit(maxActiveSessions)
	if err := query.Pluck("session_id", &keepIDs).Error; err != nil {
		return err
	}

	overflow := db.Table("system_user_session").
		Where("user_id = ? AND revoked_at IS NULL", userID)
	if len(keepIDs) > 0 {
		overflow = overflow.Where("session_id NOT IN ?", keepIDs)
	}
	return overflow.Update("revoked_at", now).Error
}

func PurgeHistoricSessions(db *gorm.DB, now time.Time, retentionDays int) error {
	if db == nil || retentionDays <= 0 {
		return nil
	}

	cutoff := now.AddDate(0, 0, -retentionDays)
	return db.Table("system_user_session").
		Where("(revoked_at IS NOT NULL AND revoked_at < ?) OR (revoked_at IS NULL AND refresh_expires_at < ?)", cutoff, cutoff).
		Delete(nil).Error
}

func LoadSessionIdleMinutes(db *gorm.DB, fallback int) int {
	if db == nil {
		return fallback
	}

	var raw string
	if err := db.Table("system_setting").
		Select("setting_value").
		Where("setting_key = ?", "login.session_idle_minutes").
		Limit(1).
		Pluck("setting_value", &raw).Error; err != nil {
		return fallback
	}

	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func columnPrefix(alias string) string {
	trimmed := strings.TrimSpace(alias)
	if trimmed == "" {
		return ""
	}
	if strings.HasSuffix(trimmed, ".") {
		return trimmed
	}
	return trimmed + "."
}

func lastSeenExpr(prefix string) string {
	return fmt.Sprintf("COALESCE(%slast_activity_at, %slast_refresh_at, %screated_at)", prefix, prefix, prefix)
}
