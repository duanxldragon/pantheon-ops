# D1: External Integration Acceptance

**Priority:** P0  
**Estimate:** 2-4 engineer-days once endpoints and scoped credentials exist  
**State:** blocked on environment and maintainer gate

## In

- Prometheus or VictoriaMetrics query and timeout paths.
- Loki log query and failure paths.
- Tempo trace query and failure paths.
- Kubernetes Service CRUD with resourceVersion/conflict handling.
- ArgoCD health/sync/rollback paths.
- Velero backup/restore/drill paths.
- Permission, DataScope, audit, cleanup, and idempotency evidence.

## Out

- Production mutation.
- Storing credentials, kubeconfig, or raw external responses in the repository.
- Replacing real acceptance with mocks or screenshots.

## Exit Evidence

- Endpoint owner and environment inventory.
- Redacted command/API logs with timestamps.
- Positive, negative, timeout, retry, and cleanup results per integration.
- Maintainer review linked from the parent evidence directory.

## Stop Conditions

Stop when credentials are broader than the approved test scope, cleanup is not reversible, or the external system cannot provide a bounded test window.
