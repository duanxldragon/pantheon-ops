# Code Review Summary

**APPROVE — current O2 CODE implementation only.** Independent review under L2 parent `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`.

**Files reviewed:** all 10 final changed files plus focused ownership/RC/renewal/watchdog/fencing, command and Service query support. **Issues:** 0; CRITICAL 0, HIGH 0, MEDIUM 0, LOW 0.

## Spec compliance and root-cause guard

Stage 1 passed before final quality approval. The two original HIGH findings are repaired in current bytes:

- Service owner completion uses the actual caller SQL transaction through `capability.go:148`, `target_selection.go:167` and `service/state.go:31`. Validation, scope/hierarchy reads, state/revision/result writes and uninstall/retire remain owner-controlled. `SkipDefaultTransaction` prevents independent write transactions. Owned completion (`attempt_ownership.go:888`) and manual completion (`deploy_service.go:2221`) both consume the binding. Independent real-owner fault runs passed outer commit failure, post-mutation refusal/deadline, uninstall second-command and post-retire rollback; assertions include all nine existing Deploy/CMDB tables and ServiceInstance bytes. Normal owned/manual install and uninstall commits passed.
- First-claim cancellation now enters queue recovery (`deploy_service.go:1610`) and serialized task-first settlement (`attempt_ownership.go:650`, `:676`). Live attempts defer settlement. Host/task/event/lease changes roll back together on error and replay without duplicates. Additional independently authored overlay cases passed no-lease/no-reservation behavior, preservation of another task's physical lease, and rollback after physical deletion failure.

Unknown/nonbinding and nil-returning injected owners refuse before their applier is invoked; successful no-owner/no-link compatibility matches the supplied scope. This repairs the primary contract and keeps failures visible. No masking workaround was identified.

The inspected protocol retains cumulative three chargeable attempts/free pauses, database clock and READ COMMITTED transactions, worker and physical ownership, dual renewal, independent watchdog, old callback fencing, same-version commands and unknown recovery. No new direct system Service/Repository dependency or platform/Base change occurs in this final delta. This is a business-to-business logical extension.

## Independent current-source verification

- Required MySQL on port 33316, `PANTHEON_TEST_DSN_REQUIRED=1`, `CGO_ENABLED=1`, explicit `D:/msys64/mingw64/bin`, isolated review cache and automatic per-test fixtures.
- Full scoped capability/Service/Deploy race: exit 0, **177 top-level passes, 0 failures, 2 explicit live-VM skips**. All 179 final listed tests accounted for. Both original independent HIGH regressions passed; actual-tree tests are gofmt-equivalent to the preserved original artifact.
- New independent cancellation overlay: exit 0, all three boundary subcases passed. No product test file was changed.
- Native `gopls v0.23.0 check` on every final changed file: exit 0, no compile/type errors. Existing `deploy_service.go:4612` formatting suggestion remains. Scoped `git diff --check` passed. MCP LSP/AST and local ast-grep were unavailable; actual native diagnostics and focused searches/manual security/fallback inspection were used.

All **60/60 source bindings and 10/10 final delta hashes** match current bytes at the end of independent verification. Final manifest SHA256: `0f2eafdb206b05703f64e6ee6538f2e3c23591cad78bb4ae4c354dc0b9859093`.

Critical current hashes:

| Source                        | SHA256                                                             |
| ----------------------------- | ------------------------------------------------------------------ |
| `service/state.go`            | `f78c9dd45fbea516fce61a598f60f9cf77a8045359761e75b5c14ce742bd41f0` |
| `deploy/attempt_ownership.go` | `843ae292a60679be8e75e5b437a36bd19718c8fd5c33b842a22081ae1ee788f9` |
| `deploy/deploy_service.go`    | `3c52f840a49a7f2dd989fe549ab2e0ff1e7e46d20a906e60bdd779cf9b187b9a` |

[Verification facts](independent-verification-facts.json) pin all final-source, supplied RED/GREEN and native-gopls evidence hashes, original regression hash, commands and results. Old engine/VM evidence is historical, not current proof.

## Stop condition

Scoped CODE review is complete. ARCH retains its independent verdict. Public HTTP, current Base identity/IAM/audit, VM/cold restart, official consumption, full Ops/frontend/rebuild gates and all P0 remain open. No product source, original evidence, remote, credential, generator, seal, release or lock changes were made; independent artifacts are confined to this new review directory.
