package cmdb

import (
	"context"
	"fmt"
	cmdbgroup "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/group"
	cmdbhost "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/host"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"strconv"
	"testing"
)

func TestTargetSelectionVersionsExclusionsAndUnavailable(t *testing.T) {
	db := setupCapabilityTestDB(t)
	if err := db.AutoMigrate(&cmdbgroup.Group{}); err != nil {
		t.Fatal(err)
	}
	rows := []cmdbhost.Host{{Hostname: "one", IP: "10.8.1.1", OS: "linux", Status: "assigned", BusinessScopeID: 11}, {Hostname: "two", IP: "10.8.1.2", OS: "linux", Status: "assigned", BusinessScopeID: 11}, {Hostname: "outside", IP: "10.8.1.3", OS: "linux", Status: "assigned", BusinessScopeID: 12}, {Hostname: "retired", IP: "10.8.1.4", OS: "linux", Status: "retired", BusinessScopeID: 11}}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	c := NewDeployCMDBCapability(db, activeBizScopeReader{})
	req := TargetSelection{BusinessScopeID: 11, HostIDs: []uint64{rows[0].ID, rows[0].ID, rows[2].ID}, Manual: []string{"10.8.1.1", "10.8.1.2", "10.8.1.4", "not-an-address"}, ExcludedIDs: []uint64{rows[1].ID}}
	result, err := c.ResolveTargetSelection(context.Background(), req, &common.DataScopeReq{IsAdmin: true}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Hosts) != 1 || result.Hosts[0].ID != rows[0].ID || result.DuplicateCount != 2 || result.UnavailableCount != 2 || result.IneligibleCount != 1 || result.ExcludedCount != 1 {
		t.Fatalf("unexpected resolution: %+v", result)
	}
	again, err := c.ResolveTargetSelection(context.Background(), req, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.ScopeVersion != again.ScopeVersion {
		t.Fatal("unstable scope version")
	}
	if err := db.Model(&rows[0]).Update("ip", "10.8.2.1").Error; err != nil {
		t.Fatal(err)
	}
	changed, err := c.ResolveTargetSelection(context.Background(), req, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.ScopeVersion == changed.ScopeVersion {
		t.Fatal("metadata change bypassed version validation")
	}
}
func TestTargetSelectionContextBudgetAndGroupChanges(t *testing.T) {
	db := setupCapabilityTestDB(t)
	if err := db.AutoMigrate(&cmdbgroup.Group{}); err != nil {
		t.Fatal(err)
	}
	host := cmdbhost.Host{Hostname: "selected", IP: "10.8.3.1", OS: "linux", Status: "assigned", BusinessScopeID: 11, LabelValues: []byte(`[{"key":"env","val":"test"}]`)}
	if err := db.Create(&host).Error; err != nil {
		t.Fatal(err)
	}
	group := cmdbgroup.Group{Name: "test", Conditions: []byte(`{"operator":"AND","rules":[{"key":"env","op":"eq","val":"test"}]}`)}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	c := NewDeployCMDBCapability(db, activeBizScopeReader{})
	req := TargetSelection{BusinessScopeID: 11, GroupIDs: []uint64{group.ID}}
	before, err := c.ResolveTargetSelection(context.Background(), req, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Hosts) != 1 {
		t.Fatalf("group unresolved: %+v", before)
	}
	if err := db.Model(&group).Update("conditions", []byte(`{"operator":"AND","rules":[{"key":"env","op":"eq","val":"prod"}]}`)).Error; err != nil {
		t.Fatal(err)
	}
	after, err := c.ResolveTargetSelection(context.Background(), req, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Hosts) != 0 || before.ScopeVersion == after.ScopeVersion {
		t.Fatal("group change did not invalidate scope")
	}
	req.BusinessScopeID = 12
	other, err := c.ResolveTargetSelection(context.Background(), req, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(other.Hosts) != 0 {
		t.Fatal("context switch retained ghost targets")
	}
	req.Manual = make([]string, TargetSelectionBudget+1)
	if _, err := c.ResolveTargetSelection(context.Background(), req, nil, false); err != ErrTargetSelectionInvalid {
		t.Fatalf("budget bypass: %v", err)
	}
}

func TestTargetSelectionUsesOnlyReferencedGroupAncestors(t *testing.T) {
	db := setupCapabilityTestDB(t)
	if err := db.AutoMigrate(&cmdbgroup.Group{}); err != nil {
		t.Fatal(err)
	}
	host := cmdbhost.Host{Hostname: "scoped", IP: "10.8.5.1", OS: "linux", Status: "assigned", BusinessScopeID: 11, LabelValues: []byte(`[{"key":"env","val":"test"}]`)}
	if err := db.Create(&host).Error; err != nil {
		t.Fatal(err)
	}
	parent := cmdbgroup.Group{Name: "parent", Conditions: []byte(`{"operator":"AND","rules":[{"key":"env","op":"eq","val":"test"}]}`)}
	unrelated := cmdbgroup.Group{Name: "unrelated", Conditions: parent.Conditions}
	for _, group := range []*cmdbgroup.Group{&parent, &unrelated} {
		if err := db.Create(group).Error; err != nil {
			t.Fatal(err)
		}
	}
	child := cmdbgroup.Group{Name: "child", ParentID: parent.ID, Conditions: parent.Conditions}
	if err := db.Create(&child).Error; err != nil {
		t.Fatal(err)
	}
	c := NewDeployCMDBCapability(db, activeBizScopeReader{})
	req := TargetSelection{BusinessScopeID: 11, GroupIDs: []uint64{child.ID}}
	before, err := c.ResolveTargetSelection(context.Background(), req, nil, false)
	if err != nil || len(before.Hosts) != 1 {
		t.Fatalf("ancestor selection: %+v %v", before, err)
	}
	if err := db.Model(&unrelated).Update("name", "edited elsewhere").Error; err != nil {
		t.Fatal(err)
	}
	after, err := c.ResolveTargetSelection(context.Background(), req, nil, false)
	if err != nil || after.ScopeVersion != before.ScopeVersion {
		t.Fatalf("unrelated catalog edit invalidated selection: %v", err)
	}
	if err := db.Model(&parent).Update("conditions", []byte(`{"operator":"AND","rules":[{"key":"env","op":"eq","val":"prod"}]}`)).Error; err != nil {
		t.Fatal(err)
	}
	changed, err := c.ResolveTargetSelection(context.Background(), req, nil, false)
	if err != nil || len(changed.Hosts) != 0 || changed.ScopeVersion == before.ScopeVersion {
		t.Fatalf("ancestor change not enforced: %v", err)
	}
	if err := db.Model(&parent).Update("parent_id", child.ID).Error; err != nil {
		t.Fatal(err)
	}
	cycle, err := c.ResolveTargetSelection(context.Background(), req, nil, false)
	if err != nil || cycle.UnavailableCount != 1 || len(cycle.Hosts) != 0 {
		t.Fatalf("cycle broadened targets: %+v %v", cycle, err)
	}
}
func TestTargetSelectionCanonicalIPRejectsAmbiguousAddress(t *testing.T) {
	db := setupCapabilityTestDB(t)
	rows := []cmdbhost.Host{{Hostname: "first", IP: "10.8.6.1", OS: "linux", Status: "assigned", BusinessScopeID: 11}, {Hostname: "mapped", IP: "::ffff:10.8.6.1", OS: "linux", Status: "assigned", BusinessScopeID: 11}}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	c := NewDeployCMDBCapability(db, activeBizScopeReader{})
	req := TargetSelection{BusinessScopeID: 11, Manual: []string{"10.8.6.1"}}
	ambiguous, err := c.ResolveTargetSelection(context.Background(), req, nil, false)
	if err != nil || ambiguous.UnavailableCount != 1 || len(ambiguous.Hosts) != 0 {
		t.Fatalf("ambiguous IP selected arbitrary host: %+v %v", ambiguous, err)
	}
	if err := db.Delete(&rows[0]).Error; err != nil {
		t.Fatal(err)
	}
	mapped, err := c.ResolveTargetSelection(context.Background(), req, nil, false)
	if err != nil || len(mapped.Hosts) != 1 || mapped.Hosts[0].ID != rows[1].ID {
		t.Fatalf("mapped address failed: %+v %v", mapped, err)
	}
}

func TestTargetSelectionSmallIDSetInLargeScope(t *testing.T) {
	db := setupCapabilityTestDB(t)
	rows := make([]cmdbhost.Host, TargetSelectionBudget+1)
	for i := range rows {
		rows[i] = cmdbhost.Host{Hostname: fmt.Sprintf("large-%d", i), IP: fmt.Sprintf("10.90.%d.%d", i/250, i%250+1), OS: "linux", Status: "assigned", BusinessScopeID: 11}
	}
	if err := db.CreateInBatches(&rows, 500).Error; err != nil {
		t.Fatal(err)
	}
	outside := cmdbhost.Host{Hostname: "other-scope", IP: "10.91.1.1", OS: "linux", Status: "assigned", BusinessScopeID: 12}
	if err := db.Create(&outside).Error; err != nil {
		t.Fatal(err)
	}
	c := NewDeployCMDBCapability(db, activeBizScopeReader{})
	req := TargetSelection{BusinessScopeID: 11, HostIDs: []uint64{rows[0].ID, outside.ID}, Manual: []string{strconv.FormatUint(rows[len(rows)-1].ID, 10)}}
	before, err := c.ResolveTargetSelection(context.Background(), req, nil, false)
	if err != nil || len(before.Hosts) != 2 || before.UnavailableCount != 1 {
		t.Fatalf("small selection failed in large scope: %+v %v", before, err)
	}
	if err := db.Model(&rows[1]).Update("hostname", "unrelated-change").Error; err != nil {
		t.Fatal(err)
	}
	after, err := c.ResolveTargetSelection(context.Background(), req, nil, false)
	if err != nil || before.ScopeVersion != after.ScopeVersion {
		t.Fatalf("unrelated row invalidated explicit selection: %v", err)
	}
	if err := db.Model(&rows[0]).Update("hostname", "selected-change").Error; err != nil {
		t.Fatal(err)
	}
	changed, err := c.ResolveTargetSelection(context.Background(), req, nil, false)
	if err != nil || before.ScopeVersion == changed.ScopeVersion {
		t.Fatalf("selected metadata change bypassed version: %v", err)
	}
	if _, err := c.ResolveTargetSelection(context.Background(), TargetSelection{BusinessScopeID: 11, IncludeScope: true}, nil, false); err != ErrTargetSelectionInvalid {
		t.Fatalf("whole-scope safety budget bypassed: %v", err)
	}
	if _, err := c.ResolveTargetSelection(context.Background(), TargetSelection{BusinessScopeID: 11, Manual: []string{rows[0].IP}}, nil, false); err != ErrTargetSelectionInvalid {
		t.Fatalf("canonical-IP scan safety budget bypassed: %v", err)
	}
}
