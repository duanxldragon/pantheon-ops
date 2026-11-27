# Pantheon Base v0.19.1

The auth-owned durable-job reader supports multi-tenant execution. It resolves
the persisted owner and checks the current tenant, membership, roles, permission,
policy and data scope before granting work. Compat continues to authorize
platform jobs in tenant 0.

The Go contract retains its existing interface and requires no database
migration. Consumers rebuild from the official Foundation release and refresh
the grant at dispatch and execution using the trusted job record. Before
provider I/O they revalidate the current source and stable targets against the
persisted tenant and the returned scope.

## Delivery evidence

See the Base [task packet](https://github.com/duanxldragon/pantheon-base/blob/main/docs/harness/tasks/2026-11-27-tenant-business-job-authorization.task.md) and
[candidate evidence](https://github.com/duanxldragon/pantheon-base/blob/main/.harness/evidence/2026-11-27-tenant-business-job-authorization/summary.md).
