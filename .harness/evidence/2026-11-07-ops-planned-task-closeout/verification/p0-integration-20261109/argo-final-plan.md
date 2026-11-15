# Argo final follow-up plan

Parent task: `2026-11-07-ops-planned-task-closeout`; P0 integration `p0-integration-20261109`. Tier L2. Landing side: Ops business domain only, logically separated from Base system authorization and platform audit. Independent review inputs: `argo-product-independent-code.md` REQUEST CHANGES (6) and `argo-product-independent-arch.md` BLOCK. Root owns renewed independent review and overall O3/D1/P0 decision.

Scope: source/tests in `backend/modules/business/cicd/argocd/**`, new ignored `.local/ops-acceptance/argo-final-*` drivers and new `argo-final-*` evidence. Preserve all existing edits, original live failures, O2 source-stable case artifacts and diagnosis drivers. No UI/SQL/schema/manifest/ledger/lock/shared-source changes; no remote execution or credential configuration reads.

Minimum approach: reuse existing DTO nonempty update semantics, existing MySQL row locks, existing history metadata and public `common` errors. No dependencies, queue or system Service imports.

1. Add hostile MySQL tests before source repair: current/proposed PUT authorization and unchanged DB/provider zero; group/kind collisions; polling/config drift; list query/authority/cancel faults; ticket-correlated owned result recovery and concurrency/no-repeat.
2. `argocd_service.go`, `argocd_handler*.go`: reuse one proposed-update mapping, authorize both complete effective configurations again under transaction locks, classify collection faults rather than treating them as denial/notfound.
3. `argocd_authorization.go`, `argocd_preview.go`: explicit execution identity/config/credential/version fingerprint, shared group/kind allowlist, retain source availability checks and UID/RV/spec conditional patch.
4. `argocd_action.go`, `argocd_preview.go`: correlate exact ticket in private history metadata and recover same authorized actor's existing result before expiry/provider reads; authorize current and originally reviewed source/target, never dispatch from recovery. Keep terminal nonce/Unknown semantics.
5. Fresh scoped cgo MySQL race, vet/build, format/diff, hashes; concrete frontend/machine/session/audit/migration handoff and own staged provider driver for coordinating root.

Runtime limits: original real product preview failed at `argocd_product_live_test.go:106` (`app_sync_failed`), fixture setup 8 passed, cleanup 0; do not infer root cause until new trace exists. Scoped race previously skipped live. Base public live-authority/session port and before-global-denial audit policy are independent P0 prerequisites. Default missing authority stays failclosed. No fabricated actor or automatic review bypass. No rendered UI evidence: this lane cannot edit UI.

Verification: `.local/ops-acceptance/argo-final-verify.py --phase red|green|verify`, fresh exclusive log labels, `PANTHEON_TEST_DSN_REQUIRED=1`, isolated test DBs at existing fixture MySQL, explicit MinGW and cgo, source hash stability. Root retains full-tree race/overlay/rebuild ownership; this lane must not rebuild/mutate shared files.
