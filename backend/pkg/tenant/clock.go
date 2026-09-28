package tenant

import "time"

// timeNowNano is a package-level shim so tests can control TTL expiry.
var timeNowNano = func() int64 { return time.Now().UnixNano() }

// realTimeNowNano is the production clock, preserved for test cleanup.
var realTimeNowNano = timeNowNano
