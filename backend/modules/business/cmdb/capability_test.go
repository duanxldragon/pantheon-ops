package cmdb

import (
	"context"
	"sync"
	"testing"

	bizcap "pantheon-base/modules/business/capability"
	cmdbhost "pantheon-base/modules/business/cmdb/host"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/testmysql"

	"gorm.io/gorm"
)

const ownershipConflictError = "cmdbhost.ownership_conflict"

type activeBizScopeReader struct{}

func (activeBizScopeReader) GetActive(_ context.Context, id uint64, _ *common.DataScopeReq) (bizcap.BizScopeRef, error) {
	return bizcap.BizScopeRef{ID: id, Code: "scope", Name: "Scope", Status: "active"}, nil
}

func (activeBizScopeReader) ResolveActiveByCodes(context.Context, []string, *common.DataScopeReq) (map[string]bizcap.BizScopeRef, error) {
	return map[string]bizcap.BizScopeRef{}, nil
}

func setupCapabilityTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&cmdbhost.Host{}); err != nil {
		t.Fatalf("migrate hosts: %v", err)
	}
	return db
}

func TestOwnershipBindRejectsAlreadyOwnedHost(t *testing.T) {
	db := setupCapabilityTestDB(t)
	host := cmdbhost.Host{Hostname: "owned", IP: "10.2.0.1", OS: "linux", BusinessScopeID: 11}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	command := NewDeployCMDBCapability(db, activeBizScopeReader{})
	err := command.Bind(context.Background(), bizcap.BindOwnershipRequest{BusinessScopeID: 12, HostIDs: []uint64{host.ID}})
	if err == nil || err.Error() != ownershipConflictError {
		t.Fatalf("expected ownership conflict, got %v", err)
	}
}

func TestOwnershipBindIsSingleWinner(t *testing.T) {
	db := setupCapabilityTestDB(t)
	host := cmdbhost.Host{Hostname: "race", IP: "10.2.0.2", OS: "linux"}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	command := NewDeployCMDBCapability(db, activeBizScopeReader{})
	requests := []bizcap.BindOwnershipRequest{
		{BusinessScopeID: 21, HostIDs: []uint64{host.ID}},
		{BusinessScopeID: 22, HostIDs: []uint64{host.ID}},
	}
	start := make(chan struct{})
	errs := make(chan error, len(requests))
	var group sync.WaitGroup
	for _, request := range requests {
		group.Add(1)
		go func(request bizcap.BindOwnershipRequest) {
			defer group.Done()
			<-start
			errs <- command.Bind(context.Background(), request)
		}(request)
	}
	close(start)
	group.Wait()
	close(errs)

	successes := 0
	for err := range errs {
		if err == nil {
			successes++
			continue
		}
		if err.Error() != "cmdbhost.ownership_conflict" {
			t.Fatalf("expected ownership conflict, got %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly one bind winner, got %d", successes)
	}
}

func TestOwnershipUnbindRejectsWrongOwner(t *testing.T) {
	db := setupCapabilityTestDB(t)
	host := cmdbhost.Host{Hostname: "wrong-owner", IP: "10.2.0.3", OS: "linux", BusinessScopeID: 31}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	command := NewDeployCMDBCapability(db)
	err := command.Unbind(context.Background(), bizcap.UnbindOwnershipRequest{BusinessScopeID: 32, HostID: host.ID})
	if err == nil || err.Error() != "cmdbhost.ownership_conflict" {
		t.Fatalf("expected ownership conflict, got %v", err)
	}
}
