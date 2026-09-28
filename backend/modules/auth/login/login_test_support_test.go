package login

import (
	"context"
	"testing"

	iamuser "github.com/duanxldragon/pantheon-base/backend/modules/system/iam/user"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/authuser"

	"gorm.io/gorm"
)

// testCredentialRepo returns the production credential adapter so tests
// exercise the same system/iam/user implementation the composition root wires
// into the auth module. Tests may import the system module; production code may
// not — scripts/harness/check-boundaries.mjs exempts *_test.go by construction.
func testCredentialRepo(db *gorm.DB) authuser.Repository {
	return iamuser.NewCredentialRepository(db)
}

// loadAuthUser hands a test the credential projection the runtime actually
// sees, so port-taking internals can be driven with real persisted state
// instead of a hand-built stand-in.
func loadAuthUser(t *testing.T, db *gorm.DB, userID uint64) *authuser.User {
	t.Helper()
	u, err := testCredentialRepo(db).FindByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("load credential projection: %v", err)
	}
	return u
}
