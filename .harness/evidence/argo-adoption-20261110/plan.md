# Argo reviewed-actions Ops adoption — Generator L2 plan

Task ID: argo-adoption-20261110
Parent task: 2026-11-07-ops-planned-task-closeout
Parent packet: docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md
O3: .harness/tasks/2026-08-31-ops-operational-workbench-design/packets/O3-change-preview-and-risk.md
D1: .harness/tasks/2026-09-28-ops-delivery-readiness-gap-closure/packets/D1-external-integration-acceptance.md
Evidence: .harness/evidence/argo-adoption-20261110/
Review: .harness/evidence/argo-adoption-20261110/review.md
Summary: .harness/evidence/argo-adoption-20261110/summary.md
Change reference: none
Role: Generator; root owns independent review, deployment, full stable CGO gates and runtime proof. No P0 approval.

## Scope and success criteria

Business domain only, logical adapter changes within existing physical Argo package. No Base/shared/frontend/lock/ledger/provider-patch/fork edits. Preserve every pre-existing worktree change. No credentials/config reads, remote execution, child agents, commit, push or release.

Reuse the reviewed provider protocol and existing business authorization, encrypted review, reservation, fingerprint, audit/redaction and reconciler. Minimum new code is the narrow native HTTP adapter/protocol validation plus owned deployment helper and focused regressions; no dependency/schema/public DTO changes.

1. Negotiate the exact reviewed-actions/v1 capability before reservation and immediately before dispatch; stock/malformed/incompatible peers and disappearing routes fail closed without native/PATCH/cluster fallback.
2. Submit required native name/namespace/project plus UID/resourceVersion/lowercase SHA1/marker; native rollback uses the private immutable selected history ID including zero. Review/metadata recovery bind the same protocol/history ID. Preserve obsolete reserved result recovery, block obsolete undispatched tickets.
3. Acceptance means Running only after actual correlated Update response; uncertain transport/5xx/missing evidence retains Unknown. Cold recovery returns the same reserved ticket/UID/revision/marker and never resends or terminates name-only.
4. Prepare a root-only reversible host binary mount retaining stock server image/security/RBAC/environment/volumes/probes and 192Mi resource limit. Exact inspected namespace/deployment identity/spec and host directory identities, dedicated owner path, no symlinks/overwrites, bounded hash-verified upload, Linux immutable bit, read-only mount and native UID/RV Update validation. Preserve rollback artifacts and return exact pre/post UID/RV/seals.
5. Fresh focused package CGO race with required password-free private MySQL, vet/build/gopls/gofmt/diff and offline helper tests; final source hash stability plus sealed root command artifacts.

## Structural scope and retrieval

Affected chain: PreviewApplicationContext -> prepareApplicationAction -> native capability + manifests/live diff -> encrypted applicationReview -> submitApplicationAction -> DB reservation/current authority -> reviewed native Sync/Rollback -> correlated response -> durable history -> recoverReviewedAction/applyApplicationStatus.
Risk nodes: provider capability validation, UID/RV and history identity, final permission checks, one dispatch, lost-response reservation, root filesystem staging and Deployment optimistic update.
No new cross-domain dependencies, hub or architecture abstraction. CodeGraph MCP unavailable; existing read-only `codegraph context -p .` queries attempted and `codegraph status` confirms 18 added/17 modified pending changes. Context returned older unrelated symbols and omitted the new untracked Argo action files; read the owned files directly rather than treating the older index as proof. No index mutation.

## Verification and review path

Generator self-review is findings-first, with explicit limitations. Child agents are prohibited; root next performs independent adapter/helper review against artifact-seal.json and provider approval evidence. No rebuild/full-repo gate is run here because it would mutate unrelated assembly/lock/ledger ownership; root owns formal overlay/rebuild and stable full CGO race.

Historical-source/config rollback remains fail-closed when it differs from current source beyond targetRevision: provider v1 has no historical-source rendering API. Supported rollback is still native immutable history-ID rollback; no synthesized source operation. Application UID/RV bind acceptance, not every managed resource through controller terminal. Runtime acceptance, real IAM/session/audit, controller Service restoration, bounded-memory runtime, lost reply/restart and overwritten provider evidence remain root proof obligations.
