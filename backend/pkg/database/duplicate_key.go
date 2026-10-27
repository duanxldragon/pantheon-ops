package database

import (
	"errors"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// mysqlDuplicateEntry is ER_DUP_ENTRY, raised when a UNIQUE index rejects a
// write.
const mysqlDuplicateEntry = 1062

// IsDuplicateKey reports whether err is a UNIQUE-index violation.
//
// The tenant, role and user create paths all look for a conflicting row before
// inserting. That is the right shape for the common case but it cannot be
// atomic: two requests carrying the same value both pass the pre-check and the
// loser only finds out at the index. Without this predicate the loser is
// reported as a 500 carrying the driver's own message
// ("Error 1062 (23000): Duplicate entry ... for key ..."), which is the wrong
// status for a caller that merely picked a taken name and leaks the index name.
//
// Both shapes are recognised: GORM's translated sentinel when the connection is
// opened with TranslateError, and the driver error otherwise, because the
// repository sets neither consistently and a helper that only knew one of them
// would silently stop working.
func IsDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlDuplicateEntry
}
