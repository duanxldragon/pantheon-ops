package impexp

// MaxImportBytes returns the configured maximum CSV import size in bytes
// (shared defensive cap aligned with the global body-limit middleware).
func MaxImportBytes() int { return maxImportBytes }

// MaxImportRows returns the configured maximum data-row count per import.
func MaxImportRows() int { return maxImportRows }
