package storage

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

// setupStorageService opens an isolated MySQL test database and migrates the
// storage snapshot tables.
func setupStorageService(t *testing.T) *StorageService {
	t.Helper()
	db := testmysql.Open(t)
	svc := NewStorageService(db, nil, nil)
	if err := svc.Migrate(); err != nil {
		t.Fatalf("migrate storage snapshot tables: %v", err)
	}
	return svc
}

// TestReplaceRowsEmptySnapshotReplacesOwnedRows is a regression test for the
// empty sync path: a cluster with zero PVs is legitimate, and refreshing its
// snapshot must succeed, drop the stale owned rows, and skip the empty INSERT
// while leaving other clusters' snapshots untouched.
func TestReplaceRowsEmptySnapshotReplacesOwnedRows(t *testing.T) {
	svc := setupStorageService(t)
	db := svc.db

	const ownedClusterID uint64 = 100
	const otherClusterID uint64 = 200

	stale := []PV{
		{Name: "stale-pv-1", ClusterID: ownedClusterID},
		{Name: "stale-pv-2", ClusterID: ownedClusterID},
	}
	other := []PV{
		{Name: "other-pv-1", ClusterID: otherClusterID},
	}
	if err := db.Create(&stale).Error; err != nil {
		t.Fatalf("seed owned stale rows: %v", err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("seed other cluster rows: %v", err)
	}

	if err := svc.replaceRows(&PV{}, ownedClusterID, []PV{}); err != nil {
		t.Fatalf("empty snapshot replacement should succeed, got %v", err)
	}

	var ownedLeft int64
	if err := db.Model(&PV{}).Where("cluster_id = ?", ownedClusterID).Count(&ownedLeft).Error; err != nil {
		t.Fatalf("count stale owned rows: %v", err)
	}
	if ownedLeft != 0 {
		t.Fatalf("stale owned snapshot rows should be removed, got %d remaining", ownedLeft)
	}

	var otherRows []PV
	if err := db.Model(&PV{}).Where("cluster_id = ?", otherClusterID).Find(&otherRows).Error; err != nil {
		t.Fatalf("query other cluster rows: %v", err)
	}
	if len(otherRows) != 1 || otherRows[0].Name != "other-pv-1" {
		t.Fatalf("other cluster snapshot should be untouched, got %+v", otherRows)
	}
}

// TestReplaceRowsNonEmptyReplacesSnapshot guards the transactional delete-then-
// insert path so an empty cluster is not the only shape of refresh that works.
func TestReplaceRowsNonEmptyReplacesSnapshot(t *testing.T) {
	svc := setupStorageService(t)
	db := svc.db

	const clusterID uint64 = 300
	if err := db.Create(&[]PV{{Name: "old-pv", ClusterID: clusterID}}).Error; err != nil {
		t.Fatalf("seed stale row: %v", err)
	}

	fresh := []PV{
		{Name: "new-pv-1", ClusterID: clusterID},
		{Name: "new-pv-2", ClusterID: clusterID},
	}
	if err := svc.replaceRows(&PV{}, clusterID, fresh); err != nil {
		t.Fatalf("non-empty snapshot replacement should succeed, got %v", err)
	}

	var rows []PV
	if err := db.Model(&PV{}).Where("cluster_id = ?", clusterID).Order("name").Find(&rows).Error; err != nil {
		t.Fatalf("query replaced rows: %v", err)
	}
	if len(rows) != 2 || rows[0].Name != "new-pv-1" || rows[1].Name != "new-pv-2" {
		t.Fatalf("owned snapshot should be replaced with fresh rows, got %+v", rows)
	}
}