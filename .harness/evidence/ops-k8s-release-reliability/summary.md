# K8s Release Reliability Evidence

Task: `ops-k8s-release-reliability` (P0, L2). Owner: `business/k8s`.

## Outcome

Kubernetes remains runtime truth and Release is now a durable, immutable
change/audit record:

- Release intent is committed under the Cluster row lock before any Kubernetes
  mutation.
- Create and rollback use immutable JSON snapshots, SHA-256 fingerprints, and
  idempotency keys. Same-key/different-snapshot requests conflict.
- Canonical states are `pending`, `applying`, `succeeded`, `failed`, and
  `timed_out`; timeout remains recoverable through Reconcile.
- Deployment, StatefulSet, and DaemonSet rollout observation is bounded and
  generation/availability aware.
- Apply ambiguity never blindly reapplies. Reconcile observes the existing
  workload and only closes an intent after the accepted target generation is
  durable.
- Conditional writes check `RowsAffected`; same-terminal concurrent closure is
  idempotent.
- Cluster deletion and Release intent creation share a Cluster row lock and
  active Release reference check.
- BizScope access uses the typed provider Reader contract. Kubeconfig and Secret
  values are not exposed in Release DTOs.
- Release permissions, audit keys, API/UI states, translations, and rendered
  desktop/mobile evidence are present.

## Changed Contracts And Files

- `backend/modules/business/k8s/release/` now owns the release state machine,
  snapshots, rollout observer, reconciliation endpoint, and fake-client tests.
- `backend/modules/business/k8s/cluster/` owns the lock/reference boundary and
  BizScope Reader usage.
- `backend/modules/business/k8s/k8s_seed.go` owns K8s menu, permission, i18n,
  and audit seed entries.
- `frontend/src/modules/business/k8s/` exposes stable idempotency keys, rollout
  status, reconcile, rollback, and empty/error/loading states.
- `docs/designs/BUSINESS_K8S_MODULE_DESIGN.md` and its English counterpart are
  now the K8s business-domain design source.

## Validation

- Backend K8s tests: pass.
- MySQL-backed Release tests using an isolated local database per test:
  pass.
- MinGW CGO race test for Release: pass.
- `go vet ./...`: pass.
- `npm run type-check`: pass.
- `npm run build`: pass, including menu/i18n/UI/search/smoke contract gates.
- Rendered evidence:
  `release-desktop.png`, `release-mobile.png`, `release-create-mobile.png`.

## Residual Risk

- The user explicitly deferred production conversion of legacy AutoMigrate
  tables to model-generated migrations. The existing `000012` fresh-schema file
  is therefore not claimed to upgrade every pre-existing `biz_k8s_release` row
  to the newest model fields; that follow-up belongs to the production
  migration pass.
- No external Kubernetes cluster was mutated. Fake/disposable client evidence
  proves the state machine; live-cluster compatibility still depends on the
  target Kubernetes API version and rollout controller behavior.
- Existing Base permission-workbench route catalog does not auto-enumerate
  business overlay routes. Server-side Casbin remains deny-by-default for
  missing policies; automatic business-route cataloging is a Base-owned
  follow-up, not copied into Ops.
