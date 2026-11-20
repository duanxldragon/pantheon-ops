# C04 real RBAC and provider cancellation loop

Status: root confirmed the request/query-boundary/non-nil-RBAC red and released the minimal production repair. `rbac.go` now passes the request context to the existing query; its old namespace helper remains a Background wrapper. Current source is frozen in `continuation-backend-rbac-cancellation-fixed-source.json`, awaiting root green. Existing Service opaque-port green remains an immutable prior checkpoint.

Latest checkpoint: root green `continuation-c04-rbac-cancellation-fixed.log.receipt.json` exited 0 with all seventeen selected top-level tests and 729 backend Go/module hashes stable. Eleven current/executed candidate hashes match in `continuation-backend-rbac-cancellation-verification.json`. See `continuation-backend-rbac-cancellation-summary.md`; independent review for this candidate remains pending, and whole-task acceptance remains open.

Red test source is now fixed in `continuation-backend-rbac-cancellation-red-source.json`. Gofmt, two-file gopls and scoped diff check passed. New narrow regex `^Test(K8sRBACRequestContext|K8sServiceMutationRealRBAC)` selects three RBAC top-level tests and three Service top-level tests. Allowed SQL/wrapper/provider controls and existing provider in-flight cancellation are expected successes even before the context repair; the request/query-entry/non-nil-admission defects need genuine root red evidence. No production RBAC code has changed.

Coordinator red receipt `../2026-11-17-ops-history-audit/continuation-c04-rbac-cancellation-red.log.receipt.json` records exit 1, CGO=1 race with required disposable MySQL. Both RBAC cancellation tests failed (RBAC package 0.325 seconds); real RBAC-backed create/update/delete cancellation failed (Service package 0.926 seconds). Both allowed SQL/provider controls and both provider GET in-flight cancellation subcases passed. Root reports 729 backend Go/module hashes stable during this red. This establishes the RBAC query-context defect and confirms the provider cancellation already worked in the selected read-stage path.

The production repair changes only `rbac.go`: preserve the existing public signature/role/nil checks and duplicate query path, obtain request context when present, retain the old helper as a Background wrapper, and add GORM WithContext. Local gofmt, three-file gopls, scoped diff check and K8s package compile passed. Full candidate regex `^Test(K8sRBACRequestContext|K8sService(Mutation|PermissionRoutes))` selects thirteen mutation, one permission and three RBAC top-level tests. Its green evidence is pending; no prior receipt is reused for the new source.

## Ownership and compatibility

Executor owns only the next changes in `backend/modules/business/k8s/rbac/rbac.go`, new `rbac/request_context_test.go`, and `service/mutation_contract_test.go`. Other lanes own their code; do not revert it. Existing Service production logic stays frozen unless fresh evidence requires a scoped correction.

Preserve `CheckHandlerPermission`'s public signature and current role/nil authorization compatibility. Pass `c.Request.Context()` into a new internal `namespaceAllowsWithContext`; retain the old `namespaceAllows` signature as a Background wrapper. Attach the provided context to the existing GORM query. Query errors continue to deny through the existing boolean result. Do not change LIKE matching, role grants, duplicate fallback semantics, DTOs, shared Base or foundation code in this loop.

## Initial red candidates

- `TestK8sRBACRequestContextCanceledBeforeAdmission`: an existing allowed binding must not authorize after request cancellation; the unchanged old method executes its contextless Count and therefore exposes the defect.
- `TestK8sRBACRequestContextCancellationAtQueryBoundary`: a deterministic GORM query-entry gate observes the request context and cancellation, with a successful real owner-SQL control. This proves context at the query boundary, not already-blocked driver/transport cancellation.
- `TestK8sServiceMutationRealRBACCanceledAdmission`: a non-nil real RBAC service in the existing Service handler must observe the canceled request and deny before any provider request. Keep a successful bound-provider control to rule out deny-all fixtures.

Root runs the real disposable-MySQL/CGO race red before production changes. Record all old-code successes and failures precisely; a passing canceled Manager test is not evidence for the real RBAC boundary.

## Provider in-flight controls

Add `TestK8sServiceMutationRealRBACProviderInFlightCancellation` using deterministic provider GET entered/exited channels and a bounded cleanup release. Start the real RBAC-backed HTTP PUT, wait until the provider GET actually arrives, then cancel the request. Require failed HTTP business outcome, zero PUT/mutation and bounded provider exit. A direct Manager subcase preserves `errors.Is(context.Canceled)`.

Cancellation occurs during the Update read request. It does not promise rollback of a PUT/DELETE already accepted by Kubernetes, and it does not establish shared Cluster/Binding lifetime or revocation atomicity. No sleeps, database subprocesses, background runtimes or production credentials are introduced.

## Verification and stop condition

After the backend window is released: add only the tests, format/check/manifest them, and hand the red regex to root. After root confirms red, make the minimal context fix, run diagnostics/compile/scoped diff checks and freeze a new manifest. Root owns real CGO race and independent review. Completion of this loop requires fresh passing evidence bound to the new source; it does not close live/provider/full-delivery gaps.
