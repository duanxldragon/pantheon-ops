# R1 sealed-byte deployment repair — Generator L2 plan

Task: argo-adoption-v2-20261110. Parent: 2026-11-07-ops-planned-task-closeout.
Parent packet: docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md.
CODE input: ../2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/resume-argo-adoption-independent-code-report.md (HIGH R1 only).
ARCH input: sibling independent-arch-report.md (CLEAR; independent CODE remains blocking).
Original seal: ../argo-adoption-20261110/artifact-seal.json, SHA256 08ba8f44013b10f92c533a1f96ed55a7346b0f6e17a968b8c313fdf1ab4d79ec.
Role: Generator. Root owns independent delta CODE/ARCH, authorized test-host operations, full CGO/source-stable race and product acceptance. No self-approval.

## Scope and minimum approach

Business-domain local deployment tooling; no logical or physical platform/system/backend/provider/frontend change. Reuse the original deployment helper unchanged and Python stdlib. Preserve the complete original 31-file inventory, verification receipts (including failures), Linux binary and active lanes.

Only new files: .local/ops-acceptance/argo-adoption-v2-root-run.py, argo-adoption-v2-deploy-test.py, argo-adoption-v2-verify.py; new v2 evidence/seal and current EV argo-adoption-v2-sealed-bytes-handoff.md. No credentials/config, remote operations, delegation/model change, commit, PR or release.

1. Capture bounded regular, single-link, non-symlink/non-reparse seal once. Check SHA256 against explicit externally supplied --seal; parse the same immutable buffer. Capture helper once in the sealed inventory and construct the exact quoted command from the buffer whose SHA256 matched its pin.
2. Verify fd/path identity and shape around capture; refuse changed captured bytes/unsafe paths before transport import/connect. After completed capture, path changes cannot alter the captured approved command; do not claim namespace atomicity. Keep binary on one checked fd for preflight and bounded streaming; unchanged helper checks remote digest before native dry-run/single Update. Keep exact test host, private transport, output/deadline bounds and no retries.
3. Deterministic memory replacement probes for both R1 windows and valid captured-byte path, unsafe captures, fd/path drift, bounds, exact binary stream, wrong peer and uncertainty. Only fake command-recording transport, never credentials or command execution. Small disposable files provide real fd identity.
4. Fresh offline tests, unchanged helper guards, Python 3.9 AST/in-memory compile, exact diff/whitespace checks and regression sensitivity mutants. Preserve before/after all original evidence and source hashes, plus new source stability. New request seal binds unchanged original inventory/final3 receipt/binary and all three new Python files.

## Structural scope and evidence

Affected chain: v2 main -> prepare -> captured seal SHA/JSON -> original inventory/final3/v2 receipt pins -> captured helper SHA -> retained binary fd -> execute -> root private transport -> exact approved helper command -> bounded stream -> unchanged remote digest/immutable/native Update.
Risk nodes: checked-vs-consumed bytes, local path/fd identity, bounded binary/output, connect/exec, optimistic single native Update. No new engine/API/shared contract or dependency. CodeGraph MCP tools are unavailable in this session; the task supplies exact files and those files were read directly. No index write or delegated exploration.

Verification receipt and command logs: argo-adoption-v2-<fresh-label>-verification.json/logs. Source delta: matching .diff. New seal: argo-adoption-v2-artifact-seal.json. Review/handoff: current EV argo-adoption-v2-sealed-bytes-handoff.md; root independent CODE/ARCH pending.

Go source is unchanged and is checked against all original sealed hashes. Retain original final3 181 non-group/22-parent passes as existing approved-scope evidence, without relabeling it as a fresh/full race. No whole package, Base rebuild, credentials, server readiness or remote rollback is run by this lane. Full CGO race, real controller/service restoration, session/IAM/audit, 192Mi runtime and O3/D1/P0 remain downstream gaps.

Stop condition: narrow fix locally verified and sealed, detailed exact-command handoff ready for independent root review. Gate Outcomes: checked-byte probes catch seal/helper reopen regressions; fd/path/bounds/peer gates catch unsafe local captures and unauthorized exec; independent deployment acceptance remains open.
