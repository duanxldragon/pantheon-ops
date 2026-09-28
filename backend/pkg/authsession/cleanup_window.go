package authsession

import (
	"errors"
	"strings"
	"time"
)

// CleanupWindow is an optional RFC3339 [start, end] range used by audit-log
// cleanup endpoints (login-log, security events).
type CleanupWindow struct {
	StartedAt time.Time
	EndedAt   time.Time
}

// ParseCleanupWindow parses an optional "startedAt/endedAt" query pair.
// An absent pair yields (nil, nil); every malformed or inverted shape yields
// the caller-supplied invalidErr so each endpoint keeps its own error code.
func ParseCleanupWindow(startedAt, endedAt, invalidErr string) (*CleanupWindow, error) {
	startedAt = strings.TrimSpace(startedAt)
	endedAt = strings.TrimSpace(endedAt)
	if startedAt == "" && endedAt == "" {
		return nil, nil
	}
	if startedAt == "" || endedAt == "" {
		return nil, errors.New(invalidErr)
	}
	start, err := time.Parse(time.RFC3339, startedAt)
	if err != nil {
		return nil, errors.New(invalidErr)
	}
	end, err := time.Parse(time.RFC3339, endedAt)
	if err != nil {
		return nil, errors.New(invalidErr)
	}
	if end.Before(start) {
		return nil, errors.New(invalidErr)
	}
	return &CleanupWindow{StartedAt: start, EndedAt: end}, nil
}
