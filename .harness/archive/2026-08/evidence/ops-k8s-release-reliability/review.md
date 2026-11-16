# Findings-First Self-Review

Task: `ops-k8s-release-reliability`.

## Findings

1. **Create originally discarded the caller idempotency key.**
   The first MySQL run exposed duplicate Releases for the same request. Resolved
   by passing `CreateReleaseRequest.IdempotencyKey` into the durable intent
   writer and adding a same-key/different-snapshot conflict test.

2. **Rollout failure and timeout were persisted but returned as success.**
   The first MySQL run exposed nil errors for failed and timed-out rollouts.
   Resolved by returning the terminal rollout error after the durable status
   transition succeeds.

3. **Reconcile could close an applying record before the target generation was
   persisted.**
   Resolved by keeping un-targeted records recoverably applying and by checking
   all conditional update `RowsAffected` values. Same-terminal closure is now
   idempotent.

4. **The old local image helper silently changed the first container when a
   named container was missing.**
   Resolved by removing the unsafe helper; all mutations use the strict
   not-found behavior.

## Resolution And Evidence

- Findings 1–4 are fixed in
  `backend/modules/business/k8s/release/release_service.go` and covered by
  MySQL-backed tests in `release_service_test.go`.
- Cluster lock/reference behavior is covered by rollback/delete tests.
- Frontend type-check, production build, contract checks, and rendered
  desktop/mobile screenshots are present in this evidence directory.
- No external cluster credentials or mutation were used.

## Remaining Gap

Production upgrade migration for existing Release tables is explicitly deferred
by maintainer instruction to the later model-generated migration pass. The
child does not claim that gap is closed.

## Machine Readable

```json
{
  "taskId": "ops-k8s-release-reliability",
  "status": "verification",
  "findings": 4,
  "resolved": 4,
  "externalClusterMutated": false,
  "mysqlBackedTests": true,
  "raceDetector": "mingw-cgo-pass",
  "residualRisk": [
    "production legacy-table migration deferred by maintainer",
    "live Kubernetes API/controller compatibility not exercised"
  ]
}
```
