// Package common provides shared types, utilities, and sentinel errors.
//
// # Error Conventions
//
// Every service-layer error must use a helper from this package:
//
//	common.NewNotFound("auth.user.not_found")
//	common.NewConflict("dept.name_exists")
//	common.NewForbidden("menu.parent.protected")
//	common.NewBadRequest("host.invalid_ip")
//
// Callers classify with errors.Is:
//
//	errors.Is(err, common.ErrNotFound)   // all not-found variants
//	errors.Is(err, common.ErrForbidden)  // all forbidden variants
//
// Do NOT define duplicate ErrXxx variables in sub-packages.
package common

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// ── Sentinel roots ────────────────────────────────────────────────────

var (
	ErrNotFound               = errors.New("not_found")
	ErrConflict               = errors.New("conflict")
	ErrForbidden              = errors.New("forbidden")
	ErrBadRequest             = errors.New("bad_request")
	ErrUnauthorized           = errors.New("unauthorized")
	ErrInternal               = errors.New("internal")
	ErrDatabaseNotInitialized = errors.New("database.not_initialized")
)

// ── Wrapping helpers ──────────────────────────────────────────────────

// NewNotFound wraps a module-specific i18n key with ErrNotFound.
func NewNotFound(key string) error { return fmt.Errorf("%w: %s", ErrNotFound, key) }

// NewConflict wraps a module-specific i18n key with ErrConflict.
func NewConflict(key string) error { return fmt.Errorf("%w: %s", ErrConflict, key) }

// NewForbidden wraps a module-specific i18n key with ErrForbidden.
func NewForbidden(key string) error { return fmt.Errorf("%w: %s", ErrForbidden, key) }

// NewBadRequest wraps a module-specific i18n key with ErrBadRequest.
func NewBadRequest(key string) error { return fmt.Errorf("%w: %s", ErrBadRequest, key) }

// NewUnauthorized wraps a module-specific i18n key with ErrUnauthorized.
func NewUnauthorized(key string) error { return fmt.Errorf("%w: %s", ErrUnauthorized, key) }

// NewInternal wraps a module-specific i18n key with ErrInternal.
func NewInternal(key string) error { return fmt.Errorf("%w: %s", ErrInternal, key) }

// ── Utilities ─────────────────────────────────────────────────────────

// ErrMessage extracts the i18n key from a wrapped error.
//
//	fmt.Errorf("%w: auth.user.not_found", ErrNotFound) → "auth.user.not_found"
func ErrMessage(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if idx := strings.LastIndex(msg, ": "); idx != -1 {
		return msg[idx+2:]
	}
	return msg
}

// ErrorResponse is the JSON envelope returned by HTTP handlers.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// ── I18n utilities (also used by http package) ───────────────────────

var i18nKeyPattern = regexp.MustCompile(`^[a-z0-9_]+(?:\.[a-z0-9_]+)+$`)

func IsI18nMessageKey(message string) bool {
	return i18nKeyPattern.MatchString(strings.TrimSpace(message))
}

func ResolveErrorMessageKey(err error, fallback string) string {
	if err == nil {
		if strings.TrimSpace(fallback) != "" {
			return fallback
		}
		return "request.failed"
	}
	message := ErrMessage(err)
	if IsI18nMessageKey(message) {
		return message
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "request.failed"
}

// GetUserID extracts the user ID from the Gin context.
func GetUserID(c *gin.Context) uint64 {
	for _, key := range []string{"userID", "userId"} {
		val, ok := c.Get(key)
		if !ok {
			continue
		}
		userID, ok := val.(uint64)
		if ok {
			return userID
		}
	}
	return 0
}
