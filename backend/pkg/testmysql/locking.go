package testmysql

import (
	"errors"
	"sync"
	"testing"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AssertRowLocked fails the test unless another transaction holds a lock on
// table.id. The probe runs SELECT ... FOR UPDATE NOWAIT on its own connection
// and expects MySQL 3572 (ER_LOCK_NOWAIT), so a missing lock fails fast instead
// of waiting for the lock timeout.
func AssertRowLocked(t *testing.T, db *gorm.DB, table string, id uint64) {
	t.Helper()
	err := db.Transaction(func(tx *gorm.DB) error {
		var lockedID uint64
		return tx.Table(table).
			Select("id").
			Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).
			Where("id = ?", id).
			Scan(&lockedID).Error
	})
	var mysqlErr *mysqlDriver.MySQLError
	if !errors.As(err, &mysqlErr) || mysqlErr.Number != 3572 {
		t.Fatalf("expected %s/%d to be locked (MySQL 3572), got %v", table, id, err)
	}
}

// AssertSerialisedWriters proves that two writers of the same parent row cannot
// interleave. The first runs inside a transaction and blocks until released;
// while it is open the row must stay locked and the second writer must not
// complete; after release the second writer proceeds to completion. That lock
// is what makes a delete-then-insert replace atomic per parent row.
func AssertSerialisedWriters(
	t *testing.T,
	db *gorm.DB,
	table string,
	id uint64,
	first, second func(tx *gorm.DB) error,
) {
	t.Helper()

	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseFirst) }) }
	var wg sync.WaitGroup
	t.Cleanup(func() {
		release()
		wg.Wait()
	})

	firstDone := make(chan error, 1)
	secondDone := make(chan error, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		firstDone <- db.Transaction(func(tx *gorm.DB) error {
			if err := first(tx); err != nil {
				return err
			}
			close(firstStarted)
			<-releaseFirst
			return nil
		})
	}()
	select {
	case <-firstStarted:
	case err := <-firstDone:
		t.Fatalf("first writer failed before holding %s/%d: %v", table, id, err)
	}

	AssertRowLocked(t, db, table, id)

	wg.Add(1)
	go func() {
		defer wg.Done()
		secondDone <- db.Transaction(second)
	}()
	select {
	case err := <-secondDone:
		t.Fatalf("second writer completed while the first still held %s/%d (err=%v)", table, id, err)
	case <-time.After(200 * time.Millisecond):
	}

	release()
	if err := <-firstDone; err != nil {
		t.Fatalf("first writer failed: %v", err)
	}
	if err := <-secondDone; err != nil {
		t.Fatalf("second writer failed: %v", err)
	}
}
