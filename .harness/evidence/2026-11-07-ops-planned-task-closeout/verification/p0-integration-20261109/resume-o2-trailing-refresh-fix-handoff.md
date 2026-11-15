# O2 trailing refresh fix — Generator handoff

Parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`. L2, Ops `business/deploy`; local logical state correction, no physical layer split. Inputs: `resume-o2-review-blockers-fix-handoff.md`, `resume-o2-fix-review-code-report.md`, `resume-o2-fix-review-engine-arch-report.md` and original `resume-o2-engine-ownership-plan.md`.

The pending-refresh MEDIUM now has a minimal source repair and red→green regressions. This is implementation evidence for root's independent review, not CODE/ARCH approval or O2/P0 closure. CODE3 HIGH remains open; engine v2 is proposal only.

## Change and intent

Plan: retain existing hook and RequestError contracts; add generation-local pending/denied flags, strengthen only the exact revocation/detail-read tests, run local frontend checks, and write new evidence. The hook records one pending refresh while a request runs. Settlement consumes it by starting one silent trailing request; another request never overlaps. No event/poll means no further request. Explicit denial increments the existing request version, clears pending and blocks that generation; detail forbidden also blocks it. Disposal clears pending. Current generation identity, disposed state and request version gate trailing work. Ordinary offline errors retain content/stale recovery and the existing polling path.

Owned edits only:

- `frontend/src/modules/business/deploy/task/useDeployTaskSnapshot.ts:16`: pending coalescing, trailing settlement and deny/dispose fencing.
- `frontend/tests/unit/business/operational/deploy-read-revocation.test.tsx:31`: pending plus explicit deny/detail forbidden, navigation/disposal, coalesced multiple refreshes and stable request count after success.
- `frontend/tests/unit/business/operational/deploy-detail-read.test.tsx:71`: connected SSE terminal event arrives after an in-flight request captured running; old reply resolves running, trailing read returns success and closes stream without later SSE or advancing polling timers.

## Fresh verification

Exact commands, observed exit codes and log SHA256 values: [commands.json](resume-o2-trailing-refresh-fix/commands.json).

| Evidence                                                                    | Result                                                                                                                                                                               |
| --------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| [units-red.txt](resume-o2-trailing-refresh-fix/units-red.txt)               | Exact old hook: 4 failed / 13 passed. Terminal test expected request 3, observed 2. Denied-generation reload and coalescing failures also reproduced.                                |
| [units-final.txt](resume-o2-trailing-refresh-fix/units-final.txt)           | Six existing deploy context/cursor/revocation/detail/command/log-lifecycle files: **51/51 PASS**, exit 0.                                                                            |
| [native-ts-final.json](resume-o2-trailing-refresh-fix/native-ts-final.json) | Native TypeScript 6.0.3 program includes hook and both test files: zero diagnostics across program, exit 0.                                                                          |
| [lint-final.txt](resume-o2-trailing-refresh-fix/lint-final.txt)             | ESLint on all three edited files: zero warnings/errors, exit 0.                                                                                                                      |
| [build-final.txt](resume-o2-trailing-refresh-fix/build-final.txt)           | `npm run build`: full prebuild gates, `tsc -b`, Vite pass, exit 0. Existing native-config/dynamic-import warnings retained.                                                          |
| [overlay-final.txt](resume-o2-trailing-refresh-fix/overlay-final.txt)       | Business-overlay contract passes, 56 declared paths. Fresh reconstruction remains root-owned.                                                                                        |
| [integrity-final.json](resume-o2-trailing-refresh-fix/integrity-final.json) | 3/3 changed sources and 216/216 protected source/historical paths match; zero added whitespace violations. Git no-index exit 1 signifies differing files; no whitespace diagnostics. |

First green run had one fixture assertion failure because existing status-flow/metric labels also contain success. The corrected assertion checks the actual Arco status tag; this intermediate log remains new evidence. No backend edits/race run in this lane.

## Exact source and preservation

[before.json](resume-o2-trailing-refresh-fix/before.json) verifies all original 16 hashes before editing. [after.json](resume-o2-trailing-refresh-fix/after.json) contains full old/new SHA256 and diff filenames for all three owned files; exact byte copies are `<filename>.old.txt` / `<filename>.new.txt`, with readable `<filename>.diff` alongside them. Snapshot hook SHA256: old `cb1a487107711798af476edfbe5880b09496e75a8ccb0fc361e3ccc1ba9abbdd`, new `7c9cfa7f05f14f2e3e7424a93fb837600f5dd49cae8bcd1a0bcdad12a31e80d2`.

The remaining 13 reviewed files and 43 pre-existing owned paths are unchanged. Original hash manifests, handoff, reviews/receipts, engine proposal and v9 artifacts remain unchanged; new manifests describe this delta without rewriting history.

## Remaining work and gates

[Engine plan v2](resume-o2-engine-ownership-plan-v2.md) independently addresses both ARCH corrections: dual renewal, complete lock order including current start/preview inversion, existing transaction-bound CMDB capability, DB expiry boundary, bounded DB calls, independent watchdog, all persistence fences and source-bound two-service/cross-task tests. Main report: 589 words. Exact engine input hashes: [engine-source-bindings.json](resume-o2-trailing-refresh-fix/engine-source-bindings.json). Root must obtain independent ARCH-v2 before engine implementation.

Impeccable applied before implementation and completion: deployment admin/workbench intended feel remains calm, dense and operational; existing Arco typography/tokens/layout/text are preserved. JSDOM verifies status, stale and forbidden behavior only. This lane produced no desktop/mobile/light/dark rendered evidence because root owns server/browser QA; no visual polish claim. Root retains current-source VM/SSH/runtime checks, full-source/final gates, rebuild, session/IAM/audit, formal consumption/release and O2/P0 acceptance.

Gate Outcomes: terminal SSE red regression caught dropped refresh | denied-generation red regressions caught renewed reads after forbidden | none false-positive.
