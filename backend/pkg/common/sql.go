package common

import "strings"

// EscapeLikePattern escapes %, _, and \ for safe use in SQL LIKE patterns.
// Use this when user input is embedded in a LIKE clause with wildcards.
func EscapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}
