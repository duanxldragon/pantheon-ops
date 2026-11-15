package contracts

import (
	"context"
	"errors"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
)

type businessOwnerReaderStub struct{ businessReaderStub }

func (*businessOwnerReaderStub) ReadOwnerLifecycle(context.Context, uint64, uint64) (businessauth.OwnerLifecycle, error) {
	return businessauth.OwnerLifecycle{TenantExists: true, ActorExists: true}, nil
}

func TestBusinessOwnerLifecycleGetterUsesExistingProviderBeforeSeal(t *testing.T) {
	businessProviders.Lock()
	oldMode, oldAuthorization, oldSealed := businessProviders.mode, businessProviders.authorization, businessProviders.sealed
	businessProviders.mode, businessProviders.authorization, businessProviders.sealed = nil, nil, false
	businessProviders.Unlock()
	t.Cleanup(func() {
		businessProviders.Lock()
		businessProviders.mode, businessProviders.authorization, businessProviders.sealed = oldMode, oldAuthorization, oldSealed
		businessProviders.Unlock()
	})
	assertUnavailable := func() {
		t.Helper()
		got, err := RegisteredBusinessOwnerLifecycleReader()
		if got != nil || !errors.Is(err, ErrBusinessProviders) {
			t.Fatal("missing lifecycle capability was accepted")
		}
	}
	assertUnavailable()
	var typedNil *businessOwnerReaderStub
	if RegisterBusinessAuthorizationReader(typedNil) == nil {
		t.Fatal("typed nil registration accepted")
	}
	// The getter must also defend a typed nil already stored in the registry.
	businessProviders.Lock()
	businessProviders.authorization = typedNil
	businessProviders.Unlock()
	assertUnavailable()
	businessProviders.Lock()
	businessProviders.authorization = nil
	businessProviders.Unlock()
	provider := &businessOwnerReaderStub{}
	if err := RegisterBusinessAuthorizationReader(provider); err != nil {
		t.Fatal(err)
	}
	got, err := RegisteredBusinessOwnerLifecycleReader()
	if err != nil || got != provider {
		t.Fatal("pre-seal composition lost lifecycle provider")
	}
	if err := RegisterBusinessTenantModeReader(&businessModeStub{}); err != nil {
		t.Fatal(err)
	}
	if err := SealBusinessTenantProviders(); err != nil {
		t.Fatal(err)
	}
	if RegisterBusinessAuthorizationReader(&businessOwnerReaderStub{}) == nil {
		t.Fatal("sealed provider was replaced")
	}
	got, err = RegisteredBusinessOwnerLifecycleReader()
	if err != nil || got != provider {
		t.Fatal("sealing changed the provider")
	}
	// Old Readers can still register and seal, while capability-dependent
	// consumers receive an error rather than treating them as owner deletion.
	businessProviders.Lock()
	businessProviders.mode, businessProviders.authorization, businessProviders.sealed = nil, nil, false
	businessProviders.Unlock()
	if err := RegisterBusinessAuthorizationReader(&businessReaderStub{}); err != nil {
		t.Fatal(err)
	}
	if err := RegisterBusinessTenantModeReader(&businessModeStub{}); err != nil {
		t.Fatal(err)
	}
	if err := SealBusinessTenantProviders(); err != nil {
		t.Fatal(err)
	}
	assertUnavailable()
}
