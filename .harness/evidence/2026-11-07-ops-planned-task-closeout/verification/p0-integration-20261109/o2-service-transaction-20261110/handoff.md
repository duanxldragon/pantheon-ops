# O2 Service transaction Generator handoff

L2 parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`. Authorized continuation of ownership plan-v2/v3 and final engine handoff. Evidence resides in this directory. Business-to-business **logical extension**; implementation is delivered for independent review, with O2/P0 open.

## Implementation

Owner API: `capability.ServiceInstanceStateTransactionBinder.WithServiceInstanceStateTransaction(*gorm.DB) ServiceInstanceStateApplier`. Service Manager returns a bound command object and retains its original DB. Every command uses the caller's current SQL transaction and command context for state validation, hierarchy/DataScope reads, optimistic revision writes, result reads and uninstall/retire. `SkipDefaultTransaction=true` prevents independent per-write begin/commit. Missing/nontransaction bindings and canceled commands refuse; an ended transaction remains unusable.

`targetTransactionService` binds configured owners for actual transactions, covering owned completion and manual reporting. Unknown/nonbindable and nil-returning injected capabilities refuse **before owner invocation**, with outer Deploy/CMDB rollback. Request wrappers, absent owners and absent ServiceInstance linkage retain compatibility; the DB-free recording fake explicitly supports binding.

The second concrete CODE HIGH is also repaired: queue recovery includes `cancel_requested` tasks before first claim. Existing task-first completion excludes live attempts, settles unclaimed hosts, releases the locked host footprint through `releaseHostLease`, and commits task/event together. Tests cover live-owner deferral, rollback, mixed claimed/unclaimed completion and replay. No retry/timeouts, attempt/worker/physical/DB-clock/RC/watchdog/write fences or cumulative-three/free-pause policy changes.

## Exact changed sources

| Exact path                                                          | Current SHA256                                                     |
| ------------------------------------------------------------------- | ------------------------------------------------------------------ |
| `backend/modules/business/capability/capability.go`                 | `b96cabf381428701b53834907a29f9821def51f740d26144c5f3e4b1f0b82e19` |
| `backend/modules/business/deploy/attempt_ownership.go`              | `843ae292a60679be8e75e5b437a36bd19718c8fd5c33b842a22081ae1ee788f9` |
| `backend/modules/business/deploy/cancel_unclaimed_test.go`          | `5f7dd6b7537a6645b0aed7f7cfd7b7a1e2567c0f9e92448fc3409b757d780e72` |
| `backend/modules/business/deploy/deploy_service_test.go`            | `aafed8ce256f9fc6dd12f924f79437c14f44438a53a0838c4076cc5998f76bb4` |
| `backend/modules/business/deploy/deploy_service.go`                 | `3c52f840a49a7f2dd989fe549ab2e0ff1e7e46d20a906e60bdd779cf9b187b9a` |
| `backend/modules/business/deploy/independent_o2_regression_test.go` | `51a4584e4daee6500df9b33070d1e45e9f28cc133695e435f57bb36c4bb4fe09` |
| `backend/modules/business/deploy/service_transaction_test.go`       | `ec727226646c0532d0d2dc405643a8f32858c7f5cb8671e7a1e3278257b3cac8` |
| `backend/modules/business/deploy/target_selection.go`               | `489cffa041660a6c43c5ff75da5ed68276f929378075ff4df46f5990aefa164b` |
| `backend/modules/business/service/state_transaction_test.go`        | `6d6faa3000b591de71035eca9569432278fd0a48c070e758a5d019e8fac5c357` |
| `backend/modules/business/service/state.go`                         | `f78c9dd45fbea516fce61a598f60f9cf77a8045359761e75b5c14ce742bd41f0` |

Initial [source-before.json](source-before.json) matched all 46/46 old deploy bindings. [source-delta-final.json](source-delta-final.json) gives every old→new hash, including new files. Final before/after manifests cover **60 files** across capability/Service/Deploy and match throughout verification. Historical final47/46 evidence remains old-source evidence. Both original independent overlay regressions are now retained in the actual test tree, with their original artifact hash recorded in [test-facts-final.json](test-facts-final.json).

## Fresh verification

Windows Go 1.26.10; `PATH` explicitly includes `D:\msys64\mingw64\bin`, `CGO_ENABLED=1`; required isolated MySQL33316 (`PANTHEON_TEST_DSN_REQUIRED=1`). Task-owned `GOCACHE=.local/o2-service-transaction-gocache`; offline dependencies.

RED: `go test -mod=readonly -race -count=1 -timeout=10m ./modules/business/deploy -run '^(TestServiceTransaction|TestIndependentO2)' -v` → **exit 1: four top-level failures, one normal-commit pass, zero skips**. Actual-owner rollback/deadline/uninstall windows and both independent HIGH reproductions are recorded in [red-linked-owner-final.txt](red-linked-owner-final.txt). Earlier compile/manual-reservation assertion errors remain historical preflight artifacts, distinguished in [red-preflight-facts.json](red-preflight-facts.json).

GREEN preflight: `go test -mod=readonly -race -count=1 -timeout=10m ./modules/business/capability ./modules/business/service ./modules/business/deploy -run '^(TestServiceTransaction|TestStateTransaction|TestIndependentO2|TestCancellationUnclaimed|TestDeployTaskRoutesServiceInstance)' -v` → exit 0, [log](preflight-complete.txt).

Final commands, backend working directory:

- `go test -mod=readonly -race -count=1 -timeout=25m ./modules/business/capability/... ./modules/business/service/... ./modules/business/deploy/... -v` → exit 0.
- `go vet -mod=readonly ./modules/business/capability/... ./modules/business/service/... ./modules/business/deploy/...` → exit 0.
- `go build -mod=readonly ./modules/business/capability/... ./modules/business/service/... ./modules/business/deploy/...` → exit 0.
- `go test -mod=readonly -list . ./modules/business/capability/... ./modules/business/service/... ./modules/business/deploy/...` → exit 0.
- `gopls check modules/business/capability/capability.go modules/business/deploy/attempt_ownership.go modules/business/deploy/cancel_unclaimed_test.go modules/business/deploy/deploy_service_test.go modules/business/deploy/deploy_service.go modules/business/deploy/independent_o2_regression_test.go modules/business/deploy/service_transaction_test.go modules/business/deploy/target_selection.go modules/business/service/state_transaction_test.go modules/business/service/state.go` → exit 0.
- `git diff --check -- modules/business/capability modules/business/service modules/business/deploy` → exit 0.

[Final receipt](focused-final-receipt.json): **177 top-level passes, zero failures**, 179 listed tests accounted for; zero missing original 159 tests. Explicit skips: `TestDeployLiveVMExecution`, `TestDeployLiveVMCancellation` (live acceptance unconfigured). Capability/executor/orchestrator report no test files. Native diagnostics have no compile/type errors; pre-existing `fmt.Sprintf` suggestion at `deploy_service.go:4612` retained. `gofmt -l` all ten changed files exits 0, empty output.

Real-owner regressions assert all nine Deploy/CMDB tables plus ServiceInstance state/version/retire rollback after owner mutation/refusal/deadline, outer commit failure, second uninstall command and post-retire failure; normal owned/manual commits, binding refusal, uncommitted scope/revision reads, conditional-write conflict, replay and no dangling/independent transaction also pass.

## Cleanup and remaining gates

[testmysql cleanup facts](cleanup-format-facts.json): exact generated fixtures removed by test cleanup; before/after inventories contain the same two untouched historical databases. No manual DB deletion. Candidate1 logs/hashes remain in `candidate1/`; its source is superseded by the final fenced-helper refinement.

Original CODE `REQUEST CHANGES` (two HIGH) and ARCH `BLOCK` (one HIGH) remain pending **independent current-source re-review**; this Generator supplies repairs/evidence and does not approve them. No additional concrete scope-compatible findings remain unrepaired from those reports. Dispatcher retains full Ops CGO race/source/integration/rebuild/frontend gates and current-source VM/cold-restart acceptance after re-review. No P0 closure, waiver, release, remote/credential or Base/shared/frontend/O4/Velero/Argo edits.
