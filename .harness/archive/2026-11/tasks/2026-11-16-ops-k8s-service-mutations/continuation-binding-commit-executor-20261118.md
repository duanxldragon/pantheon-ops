# C04 binding/cluster lock executor handoff

Implementation ready and frozen; coordinated GREEN, compatibility, independent review and final runtime verification pending. Whole C04 remains open. Scope/lock protocol: continuation-binding-commit-plan-20261118.md.

## Meaningful RED

Root's actual disposable-MySQL CGO race receipt is `.harness/evidence/2026-11-17-ops-history-audit/continuation-c04-admission-lock-red-20261118-01.log.receipt.json`, exit 1, with corresponding log. All three create/update/delete subtests fail specifically because an independent transaction acquires the Cluster row while the provider mutation is gated: `cluster owner row not protected ... <nil>`. This is a code regression failure, not compile failure or fixture failure. No GREEN is inferred from it.

## Implementation and verification

Three production files and two tests. Cluster's old WithClusterLock uses an extracted byte-equivalent scoped row-lock query; its old row/advisory ordering is retained. New WithClientsetLockWithContext holds the transaction row fence, constructs the client using a transaction-bound credential read and passes the owner transaction to Namespace. Namespace locks the binding row in that same transaction and validates the action/business scope/environment against the locked registration. Service create/update/delete execute within that callback, preserving all provider identity/version/error checks. Reads, HTTP routes, permissions, i18n and DTOs are unchanged.

Fresh executor checks:

- `gofmt -w` on the five source files; completed.
- `go test -run '^$' -mod=readonly ./modules/business/k8s/service ./modules/business/k8s/namespace ./modules/business/k8s/cluster` from backend: exit 0; compile only, no tests run.
- `gopls check` on the five changed source/test files from backend: exit 0, no diagnostics. An earlier invocation from repo root logged a module-loading error despite exit 0; that result is not accepted as diagnostics evidence. The corrected backend invocation is the accepted result.
- Scoped `git diff --check`: exit 0.
- Unchanged `node scripts/check-business-module-boundaries.mjs`: exit 0.

Root coordinates actual DB/race validation after relevant source freezes. Next filter: `^TestK8sServiceMutation(HoldsAdmissionLocksThroughProvider|Admission.*)$` in `./modules/business/k8s/service`; then existing Service mutation/RBAC tests and Cluster/Namespace compatibility. New tests cover independent MySQL NOWAIT lock ownership through simulated create/update/delete, release after success/provider denial/cancellation, blocked Cluster or Binding deadline and subsequent request recovery, and active credential lookup using a single database connection. These new GREEN tests have not been run by this executor.

## Frozen hashes

```json
{
  "backend/modules/business/k8s/cluster/cluster_service.go": "5ce7ae17ebc5976179a63d4fe76384b53f0051b929beab1bd870b3ffdc510c67",
  "backend/modules/business/k8s/namespace/namespace_service.go": "913fc7cf8ff4b6a87514f7716ba1ef05502ef34a47beb2ae1bf043603b60f73c",
  "backend/modules/business/k8s/service/service.go": "f9c2ecf7c96518809566364f8f3d86290c8b9e81e28d26ec3627bdd37c8abaa5",
  "backend/modules/business/k8s/service/mutation_contract_test.go": "e8ca6a48ca36c7b4369fd01851eba11ee45917680d8dde44bebdb4052486ba03",
  "backend/modules/business/k8s/service/mutation_admission_lock_test.go": "98ee0b97f03562883230db17b830d2ba39df6f38a4ad95d95520b4f19dcbc29f"
}
```

## Limits and tradeoffs

The new cancellable path uses no advisory/session lock, avoiding a deferred RELEASE_LOCK on an already-canceled SQL transaction. MySQL row locks release with transaction completion. The lock order is Cluster then Binding. Holding the Cluster row serializes mutations in different namespaces of the same cluster and delays Cluster metadata/credential writers during the caller's existing bounded operation; this is an explicit concurrency cost of reusing the existing stable identity fence.

The callback serializes local binding/registration changes until the simulated provider call returns. It cannot roll back a Kubernetes-accepted write, guarantee no remote side effect after a transport timeout/cancel, or serialize direct external kubectl. Real cluster/role/provider and final business acceptance remain open. No credentials were read, no live provider called, no schema/new dependencies/checker exemptions/commit/release or production operation was introduced. Root retains final acceptance, source-bound runtime, actual full race and independent review ownership.

## GREEN-01 failure and isolated fixture correction

Root's actual `continuation-c04-admission-lock-green-20261118-01.log` and receipt report exit 1. Owner-lock, provider-error, blocked-owner deadline and single-connection controls passed; create/update/delete under `TestK8sServiceMutationAdmissionLocksReleaseOnCancellation` failed at the original five-second wait for canceled provider exit. The mutation worker had already returned. This attempted GREEN remains a failed run.

The executor added `TestK8sServiceMutationProviderFixtureCancellation`, which uses only a local httptest HTTP server/client and valid Service/DeleteOptions bodies; it does not initialize MySQL or call the service manager. Before changing the fixture gate, all three POST/PUT/DELETE subtests reproduced the same failure: the client returned `context.Canceled`, but the server exit signal did not arrive within five seconds. Actual exit 1 is preserved in `continuation-c04-provider-fixture-cancel-red-20261118-01.log`.

The fixture previously waited at its mutation gate before consuming the request body. Go net/http starts background disconnect reads after request body EOF, so the fixture itself prevented observing client disconnect. The root reviewer independently confirmed this ordering in the installed Go HTTP server source. Only `mutation_contract_test.go` was corrected: each mutation now completes the existing bounded body read/decode, then enters the gate before any object mutation. No production code, original timeout, admission-owner NOWAIT probe, cancellation-cause check or zero-mutation/revision assertion was changed.

Fresh narrow non-race verification from backend:

- `go test -mod=readonly -count=1 -timeout=30s -run '^TestK8sServiceMutationProviderFixtureCancellation$' -v ./modules/business/k8s/service`: actual exit 0; POST/PUT/DELETE pass, test duration 0.01 seconds. Log: `continuation-c04-provider-fixture-cancel-green-20261118-01.log`. This is HTTP fixture evidence only, not DB/race or real provider evidence.
- The same three-package compile-only command: exit 0, no tests run.
- Backend-cwd `gopls check` on the five source/test files: exit 0, no diagnostics.
- Scoped `git diff --check`: exit 0; the corrected test was formatted with gofmt.

New frozen SHA256 for `backend/modules/business/k8s/service/mutation_contract_test.go`: `03a1b3b63fff4f9e2ce7e218baf411fb4b931070bb8b619810b842bc1b1075f7`. The other four hashes above remain unchanged. This supersedes the earlier fixture hash while retaining the failed-run history.

Root was notified before modification and owns the next actual MySQL/CGO race run. Requested first filter: `^TestK8sServiceMutation(ProviderFixtureCancellation|AdmissionLocksReleaseOnCancellation)$`; then `^TestK8sServiceMutation(HoldsAdmissionLocksThroughProvider|Admission.*)$`. Corrected DB GREEN, broader compatibility, independent review and whole-C04 acceptance remain pending at this handoff.
