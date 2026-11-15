# O2 v7 generator handoff

Task: `2026-11-07-ops-planned-task-closeout`; parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`; tier L2, business/deploy. Generator result only; no independent approval or P0 completion.

## Proven local result

The v6 true-VM run `1312dbcf3f70` reproduced an inactive source accepted by resume, followed by a VM completion marker and task success. Its 10 positive/2 negative setup checks, failing assertion, exact-owned cleanup and retained DB remain unchanged. `resume-o2-v7-red-analysis.json` links that evidence. Required-MySQL race red reproduced 8 inactive resume/retry admissions and one claimed host reaching SSH, before the repair.

Current declared business scope is read through the existing public `BizScopeReader.GetActive`, separately from CMDB/DataScope task visibility. Start checks the source before preparation and under the existing transaction seam; resume/retry check before state/event/host changes, with ControlTask's transaction-bound reader. Final executor dispatch and every actual SSH command (including probes and pre/post checks) recheck source and current host ownership. Scope refusal remains the canonical `business.deploy.task.scopeInvalid`, with the underlying cause preserved. Missing explicit reader is not replaced in transaction wrappers. Pause/cancel use their existing cleanup policy.

The frozen execution package/template snapshot is still used. Legacy unscoped group semantics are retained without inferring arbitrary new business scopes. No lease/state/retry/idempotency model, schema/Base/shared/auth/UI/lock/baseline/ledger changes, remote execute, commit/push/publication, or self-approval.

## Fresh verification

- `resume-o2-v7-red.json`: required MySQL, MinGW CGO1 race fails on the defect; original log retained.
- `resume-o2-v7-green1.json`: initial complete Deploy race passed 248 test items; two existing live-VM skips. Vet/build hit default cache Access denied; their failed receipts are retained. This receipt predates the canonical error projection correction.
- `resume-o2-v7-green-final.json`: final complete Deploy race 248 passed items, zero failed, required MySQL, MinGW CGO1, source hashes stable; vet/build exit 0. Only existing live-VM skips: TestDeployLiveVMExecution, TestDeployLiveVMCancellation; no MySQL or unexpected skips.
- `resume-o2-v7-diagnostics-final.json`: gopls reports one pre-existing fmt.Sprintf warning at deploy_service.go:4489, verified against the exact same HEAD source snippet; no new diagnostics. gofmt and scoped git diff checks clean, hashes stable. Original diagnostics receipt preserved.
- `resume-o2-v7-artifact-contract.json`: 9 checks retain all v6 assertions, 12 cases, cold-lease algorithms and exact cleanup; 40 prior v6 files unchanged.
- `live-diagnostic-o2-v7-build-ready-result.json` and `live-diagnostic-o2-v7-local-verification-ready.json`: Windows CGO1 race binary, Linux binary, Go driver tests, offline boundary/cleanup safety all pass; actual compiled 185-file dependency inventory and artifacts sealed. Every external compiled module sum matches go.sum.

The new MySQL tests prove inactive resume/retry refusal leaves task/hosts/events/attempts untouched, zero dispatch, claim-to-execute revocation opens no SSH connection, and actual loopback SSH connection-to-command revocation sends no command. Owner visibility, host visibility/current ownership, missing reader, owner failures, malformed projections, canceled requests and deleted sources fail closed. Cleanup remains possible for inactive/deleted sources. Existing active manual/SSH, resume/retry and frozen-snapshot tests pass.

## Review and root path

Self-review caught noncanonical error text and transaction-wrapper fallback for a missing owner reader; both are fixed and tested. This is a generator self-check, not independent CODE/ARCH approval. The user assigned this leaf generator lane; independent review and full-tree closure remain root-owned.

Root-ready request: `resume-o2-v7-phase-request.json`, executable `.local/ops-acceptance/live-diagnostic-o2-v7-runner.py --phase execute`. Run all 12 unchanged cases after reviewing the sealed request, using the existing authorized working tool. Existing restricted v6 preflight errno13 is retained; no repeated preflight or remote execute in this lane. Exact checked owned-directory cleanup and DB retention are unchanged.

Actual Base user/session/role authority remains a future public capability port; the worker's existing nil/supplied DataScope contract does not reconstruct live IAM grants. No full role/session revoke claim. True VM v7 matrix, root full-tree verification and independent acceptance remain open.

Gate Outcomes: required MySQL race caught inactive source admission and claimed-host SSH dispatch | self-review caught canonical error and nil-reader fallback | offline safety preserved all assertions and exact cleanup | default cache access rejected vet/build and workspace cache recovered them | true VM and independent root acceptance remain open.
