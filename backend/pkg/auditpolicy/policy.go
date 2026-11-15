// Package auditpolicy is the leaf, startup-only metadata audit strategy.
package auditpolicy

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// Route identifies one declared method and route template pair.
type Route struct{ Method, Template string }

// Outcome is the closed set of safe audit result categories.
type Outcome uint8

// Outcome categories recorded for declared routes.
const (
	OutcomeUnknown Outcome = iota
	OutcomeSuccess
	OutcomeDenied
	OutcomeInvalid
	OutcomeUpstream
	OutcomeStorage
	OutcomeConflict
)

// SafeMetadata is the fixed, bounded metadata projection kept per request.
type SafeMetadata struct {
	ActorID        uint64  `json:"actorId"`
	TenantID       uint64  `json:"tenantId"`
	DatasourceID   uint64  `json:"datasourceId"`
	HistoryID      uint64  `json:"historyId"`
	ViewID         uint64  `json:"viewId"`
	Outcome        Outcome `json:"outcome"`
	Count          uint32  `json:"count"`
	CatalogVersion uint16  `json:"catalogVersion"`
}

// ErrRegistration reports a rejected or sealed route declaration.
var ErrRegistration = errors.New("audit.policy.invalid_registration")
var registry struct {
	sync.RWMutex
	routes map[Route]struct{}
	sealed bool
}

const metadataKey = "pantheon_audit_safe_metadata"
const outcomeKey = "pantheon_audit_typed_outcome"

// Static segments and parameter names are code-owned; query/path input cannot
// become a declaration. Wildcards are deliberately unsupported.
var templatePattern = regexp.MustCompile(`^/(?:[A-Za-z0-9_-]+|:[A-Za-z][A-Za-z0-9_]*)(?:/(?:[A-Za-z0-9_-]+|:[A-Za-z][A-Za-z0-9_]*))*$`)

// RegisterMetadataOnly declares routes auditable with safe metadata only.
func RegisterMetadataOnly(routes []Route) error {
	registry.Lock()
	defer registry.Unlock()
	if registry.sealed || len(routes) == 0 {
		return ErrRegistration
	}
	staged := make(map[Route]struct{}, len(routes))
	for _, route := range routes {
		switch route.Method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		default:
			return ErrRegistration
		}
		if !templatePattern.MatchString(route.Template) || len(route.Template) > 255 {
			return ErrRegistration
		}
		if _, ok := registry.routes[route]; ok {
			return ErrRegistration
		}
		if _, ok := staged[route]; ok {
			return ErrRegistration
		}
		staged[route] = struct{}{}
	}
	if registry.routes == nil {
		registry.routes = make(map[Route]struct{})
	}
	for route := range staged {
		registry.routes[route] = struct{}{}
	}
	return nil
}

// Seal freezes the route registry for the rest of the process.
func Seal() error { registry.Lock(); defer registry.Unlock(); registry.sealed = true; return nil }

// ValidateRoutes requires every declaration to correspond to an actual Gin route.
func ValidateRoutes(routes gin.RoutesInfo) error {
	registry.RLock()
	defer registry.RUnlock()
	actual := make(map[Route]struct{}, len(routes))
	for _, route := range routes {
		actual[Route{Method: route.Method, Template: route.Path}] = struct{}{}
	}
	for route := range registry.routes {
		if _, ok := actual[route]; !ok {
			return ErrRegistration
		}
	}
	return nil
}

// Match uses FullPath, already selected by Gin before global handlers execute,
// even when a later middleware aborts. No request body/path/header is retained.
func Match(c *gin.Context) (Route, bool) {
	if c == nil || c.Request == nil {
		return Route{}, false
	}
	route := Route{Method: c.Request.Method, Template: c.FullPath()}
	registry.RLock()
	defer registry.RUnlock()
	_, ok := registry.routes[route]
	return route, ok
}

// SetMetadata stores bounded metadata only for a declared route.
func SetMetadata(c *gin.Context, metadata SafeMetadata) {
	if _, ok := Match(c); !ok {
		return
	}
	if metadata.Outcome > OutcomeConflict || metadata.Count > 10000 {
		metadata = SafeMetadata{}
	}
	c.Set(metadataKey, metadata)
}

// SetOutcome is used by response helpers without inspecting response bytes.
func SetOutcome(c *gin.Context, outcome Outcome) {
	if _, ok := Match(c); !ok || outcome > OutcomeConflict {
		return
	}
	c.Set(outcomeKey, outcome)
}

// Metadata returns the safe metadata recorded for the current request.
func Metadata(c *gin.Context) SafeMetadata {
	metadata := SafeMetadata{}
	if value, ok := c.Get(metadataKey); ok {
		if typed, ok := value.(SafeMetadata); ok {
			metadata = typed
		}
	}
	// Explicit safe categories survive a generic response helper failure.
	if value, ok := c.Get(outcomeKey); ok {
		if typed, ok := value.(Outcome); ok && (metadata.Outcome == OutcomeUnknown || typed == OutcomeSuccess || metadata.Outcome == OutcomeSuccess) {
			metadata.Outcome = typed
		}
	}
	if c.Writer.Status() >= 400 || c.IsAborted() {
		if metadata.Outcome == OutcomeSuccess {
			metadata.Outcome = OutcomeDenied
		}
	}
	return metadata
}

// Encode serializes safe metadata for the audit sink.
func Encode(metadata SafeMetadata) string { value, _ := json.Marshal(metadata); return string(value) }

// OutcomeName is the stable name for an outcome category.
func OutcomeName(outcome Outcome) string {
	names := [...]string{"unknown", "success", "denied", "invalid", "upstream", "storage", "conflict"}
	if int(outcome) >= len(names) {
		return "unknown"
	}
	return names[outcome]
}

// ActionName is the stable audit action name for a declared route.
func ActionName(route Route) string { return strings.ToLower(route.Method) + " " + route.Template }
