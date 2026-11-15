V6 O2 acceptance harness is locally prepared for root review. The full remote matrix has not run.

Parent task: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`; tier L2, business/deploy acceptance. Only new ignored v6 scripts and new v6 evidence were written. No backend, SQL source, frontend, lock, baseline or ledger edits.

Findings and changes:

- V5 is incomplete: 97 positive and 26 negative assertions passed, but 6 of 12 cases failed, including 2 failed assertions and 5 driver-failure events. All 12 prior exact-owned cleanup receipts prove absence and are preserved. See `resume-o2-v6-v5-failure-analysis.json`.
- The cold crash receipt identifies attempt 2, after a retained failed attempt 1. V6 follows the exact crashed attempt ID/no/actual lease and compares its full persisted expiry before/after startup. It rejects additional claims, missing history and lease rewrites. The five-minute wait uses the persisted crash receipt; no timestamp or lease writes. Startup still must honor the unexpired claim. Recovery must retain the expired original and a later successful released claim.
- BizScope contract uses active/inactive. V6 uses the public scoped update to inactive, verifies owner GetActive denial, and keeps the strict requirement that resumed execution be rejected and the completion marker remain absent. Current ControlTask/ResumeTask authorizes current hosts but omits an active BizScope recheck; this is a suspected business/deploy product gap for a separate source owner. It is not converted into a passing assertion.
- All five V5 independent verifier errors occurred at dial/handshake, not at a remote command exit. V6 separates TCP, handshake/auth/pin, channel, command and deadline outcomes. Only typed fields and fixed booleans are retained; no remote stderr or credential/error text is retained. No retry or assumed transport repair.
- Rollback remains the supported frozen single-package install-to-uninstall inverse; UninstallCommand removes only the owned done marker. The verifier returns absent with exit 0. The action/source link assertion and an actual local bash shell test prove the intended marker expectation, without claiming actual VM rollback success.
- Case status and matrix success now require driver summaries, assertions, zero recorded assertion/driver failures, source identity, all selected cases and exact cleanup. Cold execution has 380s plus a 40s termination/quiescence/cleanup reserve within 420s. Individual SSH operations have an 18s total budget.

Fresh verification:

- `python .local/ops-acceptance/live-diagnostic-o2-v6-runner.py --phase build`: final build-ready receipt passes go vet, Windows CGO/race build using D:/msys64/mingw64/bin, Linux amd64 build and named-file go test -race. Three Go tests cover cold-claim regression (five adverse subcases), typed secret suppression and actual marker shell/uninstall semantics.
- `python .local/ops-acceptance/live-diagnostic-o2-v6-local-verify.py`: 17/17 pass, including malformed identity/DB rejection before DB/SSH, finite command/output handling, exact cleanup rejection checks and artifact/dependency sealing.
- 185 local compiled dependency files match before/after and fresh review; Windows/Linux external module checksums from compiled build info match backend/go.sum. The local backend (devel) module is covered by the file inventory. All 22 saved prior receipt/script hashes match.
- An early offline oversized-output test used a 10ms budget and timed out before its second chunk; the local test budget was corrected. The first failure receipt and all subsequent build/verification receipts remain intact. Final receipts are build-ready and local-verification-ready.
- `python .local/ops-acceptance/live-diagnostic-o2-v6-runner.py --phase preflight`: PermissionError errno13 in the restricted lane. No true VM case or remote mutation occurred. This is an access blocker, not a product failure or matrix success.

Review path: inspect `resume-o2-v6-phase-request.json`, `resume-o2-v6-v5-failure-analysis.json`, `live-diagnostic-o2-v6-build-ready-result.json`, `live-diagnostic-o2-v6-local-verification-ready.json` and `resume-o2-v6-readiness.json`. The request links the live runner and sealed artifacts. Root must review before using an already authorized execution tool with working access. No new permission request, independent approval, O2/P0 closure, or completion approval is supplied.

Runtime gaps: all v6 VM cases, actual lease expiry/recovery, independent SSH failure subtype, inactive-source denial and real VM rollback remain pending. Public-service tests do not rederive full IAM middleware/session/grant authorization or prove browser SSE transport reconnects. Self-review only due the native leaf guard.

Gate Outcomes: cold-lease cardinality false-positive corrected; invalid source-status assumption caught; incomplete case/driver/cleanup evidence cannot pass matrix.
