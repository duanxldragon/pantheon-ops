package database

import (
	"context"
	"time"
)

// WithCustomTimeout wraps a context with a timeout, falling back to a fresh
// background-derived context when the caller passes nil. It returns a cancel
// function that must be called to release resources.
func WithCustomTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if ctx == nil {
		return context.WithTimeout(context.Background(), timeout)
	}
	return context.WithTimeout(ctx, timeout)
}
