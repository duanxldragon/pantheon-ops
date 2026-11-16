package observability

import (
	"bytes"
	"context"
	"errors"
	"log"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestWorkspaceLifecycleActualPublicFailureDoesNotLogPrivateSQL(t *testing.T) {
	db := lifecycleMySQL(t)
	var captured bytes.Buffer
	db = db.Session(&gorm.Session{Logger: logger.New(log.New(&captured, "", 0), logger.Config{LogLevel: logger.Warn, Colorful: false})})
	_, reader := actualOwnerLifecycle(t, db)
	// This owned synthetic table failure exercises the actual public reader's
	// SQL error path. Neither real credentials nor production data are involved.
	if err := db.Exec("RENAME TABLE system_user TO fixture_retained_system_user").Error; err != nil {
		t.Fatal(err)
	}
	captured.Reset()
	facts, err := reader.ReadOwnerLifecycle(context.Background(), 11, 9987)
	if !errors.Is(err, businessauth.ErrOwnerLifecycleUnavailable) || facts != (businessauth.OwnerLifecycle{}) {
		t.Fatal("actual public failure was not unknown")
	}
	if captured.Len() != 0 {
		t.Fatalf("public lifecycle SQL failure produced %d bytes of private SQL/owner diagnostics", captured.Len())
	}
}
