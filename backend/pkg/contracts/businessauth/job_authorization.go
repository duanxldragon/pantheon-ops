package businessauth

import "context"

// JobInput identifies a durable job through its trusted persisted owner and
// registered action. It is not an HTTP identity or a serialized access grant.
// Callers obtain ActorID and ResourceTenantID from their own trusted job
// record and derive Action from a registered server-side route.
type JobInput struct {
	ActorID          uint64 `json:"-"`
	ResourceTenantID uint64 `json:"-"`
	Action           Action `json:"-"`
}

// JobAuthorizationReader refreshes the owner's current status, permissions and
// data scope before dispatch or execution. Jobs survive logout; no session,
// token, cached roles or creation-time data scope participates in this grant.
// Compat grants require a platform owner and resource tenant 0. Multi grants
// require an enabled owner belonging to an active resource tenant, an active
// membership there, and grants from that tenant's roles. DataScope comes
// from the granting roles in the resource tenant. Businesses revalidate
// their current source and targets against the persisted tenant and DataScope.
type JobAuthorizationReader interface {
	AuthorizeJob(context.Context, JobInput) (Authorization, error)
}
