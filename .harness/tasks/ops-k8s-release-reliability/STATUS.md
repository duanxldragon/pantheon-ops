# Status: K8s Release Reliability

- Status: `complete`
- Updated: `2026-08-18`
- Executor: Codex `/root`
- Baseline commit: `ee213508a32ebdf27685128e0c719e9036272de5`

## Dirty Worktree Preflight

- Observed before implementation with `git status --short`.
- Collision set: existing untracked `backend/modules/business/k8s/**` and
  `frontend/src/modules/business/k8s/**`; modified shared frontend i18n files;
  untracked task/evidence artifacts; and pre-existing backend dependency,
  BizScope, CMDB, Deploy, overlay, migration, and frontend business-module
  changes.
- Ownership rule: preserve all existing worktree content. This task will extend
  only the K8s-owned files and task artifacts after revalidating their current
  contents; no reset, restore, revert, or overwrite of unrelated work.

## Resume Preflight

- 2026-08-17: Codex `/root` resumed this L2 child after reading the parent
  packet, queue, manifest, parent status, this task packet, handoff, manifest,
  and status in the required order.
- Baseline remains `ee213508a32ebdf27685128e0c719e9036272de5`.
- The dirty-file collision set remains the full existing `git status --short`
  set recorded above. In particular, all K8s implementation and evidence files
  are already untracked prior work and must be reviewed in place rather than
  replaced or reverted.

## Completed

- Durable intent, rollout-aware success, reconciliation, and Cluster reference
  task contract created.
- Baseline revalidated: `go test -count=1 ./modules/business/k8s/...` and
  `frontend/npm run type-check` both pass on the inherited worktree.
- Release Create now preserves the caller idempotency key; same-key/different
  snapshot conflicts are rejected.
- Rollout failure and timeout now return their terminal error after durable
  status persistence instead of returning nil.
- Conditional observation/target/final updates require exactly one row; same
  terminal outcome is idempotent.
- Reconcile cannot terminally close an applying record until target generation is
  durably recorded.
- Unsafe named-container fallback was removed; unknown container names fail.
- MySQL-backed intent/idempotency/rollback/failure/timeout/reconcile/delete
  scenarios pass with the local compose MySQL and isolated test databases.
- MinGW CGO race test for the Release package passes.
- K8s design source and required evidence artifacts are present:
  `.harness/evidence/ops-k8s-release-reliability/` and
  `docs/designs/BUSINESS_K8S_MODULE_DESIGN*.md`.

## Decisions

- Kubernetes API remains runtime truth; CMDB does not copy Workload objects.
- VM Deploy and K8s Release share identity/audit semantics, not executor code.
- Canonical Release states are `pending`, `applying`, `succeeded`, `failed`,
  and `timed_out`. An `applying` row whose terminal update was not persisted is
  an explicitly recoverable reconciliation record, never a success claim.
- Create and rollback accept caller idempotency keys. A repeated key with the
  same immutable request snapshot returns the existing Release; a different
  snapshot is rejected. Missing keys are assigned once for backward-compatible
  single submissions and therefore do not provide retry deduplication.
- Release intent is committed while the target Cluster row is locked. Cluster
  deletion takes the same lock and checks active Release references before
  soft-delete, closing the create/delete race within this database.
- Rollout policy is bounded and injectable: Deployment, StatefulSet, and
  DaemonSet only; production defaults come from narrowly scoped K8s Release
  environment configuration, while fake-client tests supply short bounds.
- Legacy AutoMigrate table conversion and production ALTER migration generation
  remain explicitly deferred by maintainer instruction. The fresh-install
  business schema is kept aligned; evidence will state the upgrade gap.

## Evidence

- `.harness/evidence/ops-k8s-release-reliability/commands.json`
- `.harness/evidence/ops-k8s-release-reliability/summary.md`
- `.harness/evidence/ops-k8s-release-reliability/review.md`
- `.harness/evidence/ops-k8s-release-reliability/release-desktop.png`
- `.harness/evidence/ops-k8s-release-reliability/release-mobile.png`
- `.harness/evidence/ops-k8s-release-reliability/release-create-mobile.png`

## Next Atomic Action

Carry the frozen K8s Release contract into the parent quality gates. The
remaining explicit gap is production migration of legacy AutoMigrate tables,
which the maintainer deferred to the later model-generated migration pass.

## Blockers

None recorded; external cluster operations remain prohibited without approval.
