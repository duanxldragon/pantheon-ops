package contracts

import (
	"github.com/gin-gonic/gin"
	"sync"
	"testing"
)

type runtimeSettingsProviderStub struct {
	calls int
	keys  []BusinessSettingDefinition
}

func (p *runtimeSettingsProviderStub) BusinessReaderFor(_ *gin.Context, definitions []BusinessSettingDefinition) (BusinessRuntimeSettingsReader, error) {
	p.calls++
	p.keys = definitions
	return runtimeSettingsReaderStub{}, nil
}

type runtimeSettingsReaderStub struct{}

func (runtimeSettingsReaderStub) GetByKey(string) (string, error) { return "", nil }
func runtimeSettingsSeeds() []BusinessSettingSeed {
	return []BusinessSettingSeed{{Key: "cmdb.ssh.port", Value: "22", ValueType: "number", GroupKey: "cmdb", Remark: "SSH port"}, {Key: "cmdb.ssh.timeout", Value: "10", ValueType: "number", GroupKey: "cmdb", Remark: "SSH timeout"}}
}
func newRuntimeSettingsRegistry() *businessSettingsRegistry {
	return &businessSettingsRegistry{modules: make(map[string][]BusinessSettingDefinition)}
}

func TestBusinessRuntimeSettingsDeclarationOwnership(t *testing.T) {
	assertReservedNamespacesRejected(t)
	assertInvalidModulesRejected(t)
	assertMalformedKeysRejected(t)
	assertInvalidSeedFieldsRejected(t)
}

func assertReservedNamespacesRejected(t *testing.T) {
	t.Helper()
	for _, namespace := range []string{"site", "platform", "system", "login", "audit", "upload", "mail", "sms", "i18n", "ui", "org", "iam", "auth", "setting", "dict", "notification", "security"} {
		registry := newRuntimeSettingsRegistry()
		seeds := []BusinessSettingSeed{{Key: namespace + ".custom", Value: "1", ValueType: "number", GroupKey: namespace, Remark: "reserved namespace probe"}}
		if registry.register("business."+namespace, seeds) == nil || len(registry.modules) != 0 {
			t.Fatalf("reserved namespace accepted or mutated registry: %s", namespace)
		}
	}
}

func assertInvalidModulesRejected(t *testing.T) {
	t.Helper()
	for _, module := range []string{"cmdb", "business.site", "business.platform", "business.system", "business.cmdb ", "business.a.b"} {
		if newRuntimeSettingsRegistry().register(module, runtimeSettingsSeeds()) == nil {
			t.Fatalf("invalid module accepted: %s", module)
		}
	}
}

func assertMalformedKeysRejected(t *testing.T) {
	t.Helper()
	for _, key := range []string{"login.password", "cmdb", "cmdb.ssh..port", "cmdb.ssh.port ", "cmdbx.ssh.port"} {
		seeds := runtimeSettingsSeeds()
		seeds[0].Key = key
		r := newRuntimeSettingsRegistry()
		if r.register("business.cmdb", seeds) == nil || len(r.modules) != 0 {
			t.Fatalf("foreign or malformed key accepted: %s", key)
		}
	}
	seeds := runtimeSettingsSeeds()
	seeds[1].Key = seeds[0].Key
	r := newRuntimeSettingsRegistry()
	if r.register("business.cmdb", seeds) == nil || len(r.modules) != 0 {
		t.Fatal("duplicate batch partially registered")
	}
}

func assertInvalidSeedFieldsRejected(t *testing.T) {
	t.Helper()
	mutations := map[string]func(*BusinessSettingSeed){
		"type":   func(s *BusinessSettingSeed) { s.ValueType = "secret" },
		"group":  func(s *BusinessSettingSeed) { s.GroupKey = "" },
		"remark": func(s *BusinessSettingSeed) { s.Remark = "" },
	}
	for field, mutate := range mutations {
		seeds := runtimeSettingsSeeds()
		mutate(&seeds[0])
		if newRuntimeSettingsRegistry().register("business.cmdb", seeds) == nil {
			t.Fatalf("invalid %s accepted", field)
		}
	}
}

func TestBusinessRuntimeSettingsFrozenCopiesAndProvider(t *testing.T) {
	r := newRuntimeSettingsRegistry()
	seeds := runtimeSettingsSeeds()
	if err := r.register("business.cmdb", seeds); err != nil {
		t.Fatal(err)
	}
	seeds[0].Value = "changed"
	if r.register("business.cmdb", runtimeSettingsSeeds()) == nil {
		t.Fatal("duplicate module accepted")
	}
	provider := &runtimeSettingsProviderStub{}
	if r.registerProvider(provider) == nil {
		t.Fatal("provider installed before seal")
	}
	snapshot := r.seal()
	if snapshot[0].Value != "22" {
		t.Fatal("caller seed mutation changed registry")
	}
	snapshot[0].Value = "tampered"
	if r.seal()[0].Value != "22" {
		t.Fatal("returned snapshot mutation changed registry")
	}
	if r.register("business.deploy", []BusinessSettingSeed{{Key: "deploy.limit", ValueType: "number", Value: "1", GroupKey: "deploy", Remark: "limit"}}) == nil {
		t.Fatal("late registration accepted")
	}
	if _, err := r.readerFor("business.cmdb", nil); err == nil {
		t.Fatal("reader available before provider")
	}
	var typedNil *runtimeSettingsProviderStub
	if r.registerProvider(typedNil) == nil {
		t.Fatal("typed nil provider accepted")
	}
	if err := r.registerProvider(provider); err != nil {
		t.Fatal(err)
	}
	if r.registerProvider(provider) == nil {
		t.Fatal("provider replacement accepted")
	}
	if _, err := r.readerFor("business.deploy", nil); err == nil || provider.calls != 0 {
		t.Fatal("unknown module reached provider")
	}
	if _, err := r.readerFor("business.cmdb", nil); err != nil {
		t.Fatal(err)
	}
	provider.keys[0].Value = "changed"
	if r.seal()[0].Value != "22" {
		t.Fatal("provider mutated registry")
	}
}

func TestBusinessRuntimeSettingsConcurrentCollision(t *testing.T) {
	r := newRuntimeSettingsRegistry()
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); results <- r.register("business.cmdb", runtimeSettingsSeeds()) }()
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 || len(r.seal()) != 2 {
		t.Fatalf("collision not atomic: successes=%d", success)
	}
}
