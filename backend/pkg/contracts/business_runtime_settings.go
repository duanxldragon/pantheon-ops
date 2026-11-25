package contracts

import (
	"errors"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// ErrBusinessSettings reports an unavailable or invalid startup-owned capability.
var ErrBusinessSettings = errors.New("business settings unavailable")

// BusinessSettingSeed declares nonpublic, unencrypted module defaults. Register
// during compiled startup before system initialization, never from a request.
type BusinessSettingSeed struct {
	Key       string
	Value     string
	ValueType string
	GroupKey  string
	Remark    string
}

// BusinessSettingDefinition associates a compiled setting seed with its owning module.
type BusinessSettingDefinition struct {
	BusinessSettingSeed
	Module string
}

// BusinessRuntimeSettingsReader reads declared settings within one trusted request.
type BusinessRuntimeSettingsReader interface {
	GetByKey(string) (string, error)
}

// BusinessRuntimeSettingsProvider is implemented and installed by system/config
// only after its required default transaction commits. Gin tenancy must come
// from trusted middleware; the provider owns mode and stored metadata checks.
type BusinessRuntimeSettingsProvider interface {
	BusinessReaderFor(*gin.Context, []BusinessSettingDefinition) (BusinessRuntimeSettingsReader, error)
}

type businessSettingsRegistry struct {
	sync.RWMutex
	modules  map[string][]BusinessSettingDefinition
	sealed   bool
	provider BusinessRuntimeSettingsProvider
}

var businessSettings = businessSettingsRegistry{modules: make(map[string][]BusinessSettingDefinition)}
var businessSettingModuleName = regexp.MustCompile("^[a-z][a-z0-9_]{0,30}$")
var businessSettingKeySuffix = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$`)

// RegisterBusinessSettings registers validated module defaults before startup sealing.
func RegisterBusinessSettings(module string, seeds []BusinessSettingSeed) error {
	return businessSettings.register(module, seeds)
}

func (r *businessSettingsRegistry) register(module string, seeds []BusinessSettingSeed) error {
	namespace := strings.TrimPrefix(module, "business.")
	if module != "business."+namespace || !businessSettingModuleName.MatchString(namespace) || len(seeds) == 0 {
		return ErrBusinessSettings
	}
	switch namespace {
	case "site", "platform", "system", "login", "audit", "upload", "mail", "sms", "i18n", "ui", "org", "iam", "auth", "setting", "dict", "notification", "security":
		return ErrBusinessSettings
	}
	definitions := make([]BusinessSettingDefinition, 0, len(seeds))
	seen := make(map[string]bool, len(seeds))
	for _, seed := range seeds {
		suffix := strings.TrimPrefix(seed.Key, namespace+".")
		if seed.Key != namespace+"."+suffix || !businessSettingKeySuffix.MatchString(suffix) || len(seed.Key) > 128 || seen[seed.Key] || strings.TrimSpace(seed.GroupKey) == "" || len(seed.GroupKey) > 32 || seed.GroupKey != strings.TrimSpace(seed.GroupKey) || strings.TrimSpace(seed.Remark) == "" || len(seed.Remark) > 255 {
			return ErrBusinessSettings
		}
		switch seed.ValueType {
		case "string", "number", "boolean", "json":
		default:
			return ErrBusinessSettings
		}
		seen[seed.Key] = true
		definitions = append(definitions, BusinessSettingDefinition{BusinessSettingSeed: seed, Module: module})
	}
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].Key < definitions[j].Key })
	r.Lock()
	defer r.Unlock()
	if r.sealed || r.modules[module] != nil {
		return ErrBusinessSettings
	}
	r.modules[module] = definitions
	return nil
}

// SealBusinessSettingSeeds freezes compiled declarations before system bootstrap.
func SealBusinessSettingSeeds() []BusinessSettingDefinition { return businessSettings.seal() }
func (r *businessSettingsRegistry) seal() []BusinessSettingDefinition {
	r.Lock()
	defer r.Unlock()
	r.sealed = true
	result := make([]BusinessSettingDefinition, 0)
	for _, definitions := range r.modules {
		result = append(result, definitions...)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result
}

// RegisterBusinessRuntimeSettingsProvider installs the sole provider after declarations are sealed.
func RegisterBusinessRuntimeSettingsProvider(provider BusinessRuntimeSettingsProvider) error {
	return businessSettings.registerProvider(provider)
}

func (r *businessSettingsRegistry) registerProvider(provider BusinessRuntimeSettingsProvider) error {
	r.Lock()
	defer r.Unlock()
	if !r.sealed || nilBusinessProvider(provider) || r.provider != nil {
		return ErrBusinessSettings
	}
	r.provider = provider
	return nil
}

// RegisteredBusinessRuntimeSettingsReader binds a reader to one request. Never
// write this request reader into a shared mutable business service.
func RegisteredBusinessRuntimeSettingsReader(module string, c *gin.Context) (BusinessRuntimeSettingsReader, error) {
	return businessSettings.readerFor(module, c)
}

func (r *businessSettingsRegistry) readerFor(module string, c *gin.Context) (BusinessRuntimeSettingsReader, error) {
	r.RLock()
	definitions := append([]BusinessSettingDefinition(nil), r.modules[module]...)
	provider := r.provider
	ready := r.sealed && !nilBusinessProvider(provider)
	r.RUnlock()
	if !ready || len(definitions) == 0 {
		return nil, ErrBusinessSettings
	}
	return provider.BusinessReaderFor(c, definitions)
}

// BusinessSettingDefault returns a registered compiled default, or an empty string for an unknown key.
func BusinessSettingDefault(key string) string {
	businessSettings.RLock()
	defer businessSettings.RUnlock()
	for _, definitions := range businessSettings.modules {
		for _, definition := range definitions {
			if definition.Key == key {
				return definition.Value
			}
		}
	}
	return ""
}
