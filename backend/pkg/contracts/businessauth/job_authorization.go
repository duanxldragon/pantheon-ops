package businessauth

import "context"

// JobInput identifies a durable job through its trusted persisted owner and
// registered action. It is not an HTTP identity or a serialized access grant.
// Callers must never obtain ActorID from an untrusted request body.
type JobInput struct {
	ActorID          uint64 `json:"-"`
	ResourceTenantID uint64 `json:"-"`
	Action           Action `json:"-"`
}

// JobAuthorizationReader refreshes the owner's current status, permissions and
// data scope before dispatch or execution. Jobs survive logout; no session,
// token, cached roles or creation-time data scope participates in this grant.
// The initial contract supports only live compat mode and resource tenant 0.
// Businesses still revalidate the current source and targets against DataScope.
type JobAuthorizationReader interface {
	AuthorizeJob(context.Context, JobInput) (Authorization, error)
}
