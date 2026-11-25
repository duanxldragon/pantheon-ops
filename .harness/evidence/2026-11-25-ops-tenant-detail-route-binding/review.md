# Review: 2026-11-25-ops-tenant-detail-route-binding

Independent CODE/ARCH review: **not performed**. Recorded gap, not an approval.

## What a reviewer should check

1. The detail keys in `cmdb/permission_routes.go` and `deploy/permission_routes.go` against the seeds and the frontend permission gates.
2. That the rewritten tests exercise Base's `permissionroutes.Covers` contract rather than a private reimplementation.
3. That `business:deploy:task:detail` is the right key for `/events` (an SSE stream) and `/rollback` (read-only info).
4. The residual: authorization is bound, but tenant row isolation on detail reads is a separate property not proven by this task.
