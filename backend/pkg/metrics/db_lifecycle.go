package metrics

import "context"

// DB metrics collector lifecycle management
var dbMetricsCancel context.CancelFunc

// RegisterDBMetricsCollector stores the cancel function for the DB metrics goroutine.
// Called by database.initMySQL during startup.
func RegisterDBMetricsCollector(cancel context.CancelFunc) {
	dbMetricsCancel = cancel
}

// StopDBMetricsCollector gracefully stops the DB metrics collection goroutine.
// Called by main.go during shutdown sequence.
func StopDBMetricsCollector() {
	if dbMetricsCancel != nil {
		dbMetricsCancel()
		dbMetricsCancel = nil
	}
}
