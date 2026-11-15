package contracts

import (
	"errors"
	"reflect"
	"sync"

	"github.com/duanxldragon/pantheon-base/backend/internal/middleware"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// BusinessTenantGroupOptions overrides the sealed business providers per group.
type BusinessTenantGroupOptions struct {
	DB            *gorm.DB
	RDB           *redis.Client
	ModeReader    tenant.ModeReader
	Authorization businessauth.Reader
	DataScope     bool
	PlatformScope bool
}

var businessProviders struct {
	sync.RWMutex
	mode          tenant.ModeReader
	authorization businessauth.Reader
	sealed        bool
}

// ErrBusinessProviders reports missing, duplicated, or unsealed providers.
var ErrBusinessProviders = errors.New("business.tenant.providers_unavailable")

func nilBusinessProvider(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice:
		return v.IsNil()
	}
	return false
}

// RegisterBusinessTenantModeReader registers the single tenant mode reader.
func RegisterBusinessTenantModeReader(reader tenant.ModeReader) error {
	businessProviders.Lock()
	defer businessProviders.Unlock()
	if nilBusinessProvider(reader) || businessProviders.sealed || businessProviders.mode != nil {
		return ErrBusinessProviders
	}
	businessProviders.mode = reader
	return nil
}

// RegisterBusinessAuthorizationReader registers the single authorization reader.
func RegisterBusinessAuthorizationReader(reader businessauth.Reader) error {
	businessProviders.Lock()
	defer businessProviders.Unlock()
	if nilBusinessProvider(reader) || businessProviders.sealed || businessProviders.authorization != nil {
		return ErrBusinessProviders
	}
	businessProviders.authorization = reader
	return nil
}

// RegisteredBusinessTenantModeReader is the startup composition seam for auth.
func RegisteredBusinessTenantModeReader() tenant.ModeReader {
	businessProviders.RLock()
	defer businessProviders.RUnlock()
	return businessProviders.mode
}

// RegisteredBusinessOwnerLifecycleReader exposes the existing auth provider's
// optional maintenance capability during module Init, before global sealing.
func RegisteredBusinessOwnerLifecycleReader() (businessauth.OwnerLifecycleReader, error) {
	businessProviders.RLock()
	defer businessProviders.RUnlock()
	reader, ok := businessProviders.authorization.(businessauth.OwnerLifecycleReader)
	if !ok || nilBusinessProvider(reader) {
		return nil, ErrBusinessProviders
	}
	return reader, nil
}

// SealBusinessTenantProviders freezes provider registration for the process.
func SealBusinessTenantProviders() error {
	businessProviders.Lock()
	defer businessProviders.Unlock()
	if businessProviders.mode == nil || businessProviders.authorization == nil {
		return ErrBusinessProviders
	}
	businessProviders.sealed = true
	return nil
}

// TenantScopedBusinessGroup is additive; all legacy groups retain their chains.
func TenantScopedBusinessGroup(r *gin.RouterGroup, path string, opts BusinessTenantGroupOptions) (gin.IRoutes, error) {
	businessProviders.RLock()
	if opts.ModeReader == nil {
		opts.ModeReader = businessProviders.mode
	}
	if opts.Authorization == nil {
		opts.Authorization = businessProviders.authorization
	}
	businessProviders.RUnlock()
	if r == nil || opts.DB == nil || opts.RDB == nil || nilBusinessProvider(opts.ModeReader) || nilBusinessProvider(opts.Authorization) {
		return nil, ErrBusinessProviders
	}
	return r.Group(path).
		Use(middleware.BusinessTokenAuthMiddleware(opts.RDB)).
		Use(middleware.BusinessTenantContextMiddleware(opts.DB, opts.ModeReader, opts.Authorization, opts.PlatformScope)).
		Use(middleware.BusinessAuthorizationMiddleware(opts.Authorization, opts.ModeReader, opts.DataScope)), nil
}
