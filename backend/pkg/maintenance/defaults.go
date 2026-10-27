package maintenance

import (
	"errors"
	"time"
)

// DefaultInterval is the interval used for the periodic runner and for tasks
// registered without an explicit interval.
const DefaultInterval = 15 * time.Minute

// errTaskPanic marks a maintenance task that panicked instead of returning.
var errTaskPanic = errors.New("maintenance.task.panic")
