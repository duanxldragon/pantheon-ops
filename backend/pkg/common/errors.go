package common

import "errors"

// Sentinel errors shared across all service packages.
// Always reference these instead of defining local equivalents.
var (
	// ErrDatabaseNotInitialized is returned when a service method is called
	// before the database connection has been established.
	ErrDatabaseNotInitialized = errors.New("database.not_initialized")
)
