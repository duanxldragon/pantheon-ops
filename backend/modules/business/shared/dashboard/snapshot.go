// Package dashboard defines the bounded metadata shared by business-owned home widgets.
package dashboard

import (
	"strconv"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
)

// MaxItems caps the number of rows a single snapshot may expose.
const MaxItems = 8

// QueryTimeout bounds widget metadata queries.
const QueryTimeout = 3 * time.Second

// Row deliberately excludes credentials, endpoints, query text and task snapshots.
type Row struct {
	ID              uint64     `json:"id"`
	Name            string     `json:"name"`
	Status          string     `json:"status"`
	BusinessScopeID uint64     `json:"businessScopeId,omitempty"`
	Kind            string     `json:"kind,omitempty"`
	ObservedAt      *time.Time `json:"observedAt,omitempty"`
}

// Snapshot is the bounded metadata set exposed to a dashboard widget.
type Snapshot struct {
	Items      []Row      `json:"items"`
	Partial    bool       `json:"partial"`
	SampledAt  time.Time  `json:"sampledAt"`
	RangeStart *time.Time `json:"rangeStart,omitempty"`
	RangeEnd   *time.Time `json:"rangeEnd,omitempty"`
}

// Limit parses a request limit string, falling back to MaxItems on error.
func Limit(raw string) int {
	n, err := strconv.Atoi(raw)
	if err != nil {
		return MaxItems
	}
	return BoundLimit(n)
}

// BoundLimit clamps n to the allowed [1, MaxItems] range.
func BoundLimit(n int) int {
	if n < 1 || n > MaxItems {
		return MaxItems
	}
	return n
}

// ScopeAllowed reports whether the given data scope is permitted for widgets.
func ScopeAllowed(scope *common.DataScopeReq) bool {
	if scope == nil {
		return false
	}
	if scope.IsAdmin {
		return true
	}
	switch strings.TrimSpace(scope.Mode) {
	case common.DataScopeModeAll, common.DataScopeModeSelf, common.DataScopeModeDept, common.DataScopeModeDeptAndChildren, common.DataScopeModeCustom:
		return true
	default:
		return false
	}
}

// NewSnapshot builds a bounded snapshot from the given rows.
func NewSnapshot(rows []Row, limit int, partial bool) *Snapshot {
	limit = BoundLimit(limit)
	if len(rows) > limit {
		rows, partial = rows[:limit], true
	}
	if rows == nil {
		rows = []Row{}
	}
	return &Snapshot{Items: rows, Partial: partial, SampledAt: time.Now().UTC()}
}
