package common

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Shared handler-layer message keys. Eleven handler files redeclare these
// exact strings as local consts; the S2 migration waves replace those copies
// with these exports. Values are unchanged, so every envelope written today
// stays byte-identical.
const (
	ErrParamInvalid  = "param.invalid"
	ErrRequestFailed = "request.failed"
)

// ParseUintParam parses the named path parameter as a uint64. On a parse
// failure it writes the same 400 envelope the inline blocks produce —
// common.Fail with CodeParamInvalid and the param.invalid key — and reports
// false so the caller returns. A parsed zero is NOT rejected: the handful of
// call sites that treat 0 as invalid (e.g. tenant) keep that check local, so
// the helper preserves the majority shape exactly.
func ParseUintParam(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		Fail(c, CodeParamInvalid, ErrParamInvalid)
		return 0, false
	}
	return id, true
}

// NormalizePagination applies the defaults every inline normalizer shares —
// page falls back to 1, pageSize to 10 — caps pageSize at the module's own
// limit, and derives the list offset. The cap arrives as a parameter because
// modules do not agree on it; unifying it would change behaviour and is out
// of scope.
func NormalizePagination(page, pageSize, maxPageSize int) (int, int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize, (page - 1) * pageSize
}

// NormalizeSort resolves a client sort field against the module's own
// whitelist of input aliases to safe column names, falling back when the
// input is unknown, unpadded-trimmed or blank. Sort direction stays at the
// call site: modules disagree on how anything-but-"asc" maps to descending,
// and that difference is behaviour.
func NormalizeSort(input string, allowed map[string]string, fallback string) string {
	if column, ok := allowed[strings.TrimSpace(input)]; ok {
		return column
	}
	return fallback
}
