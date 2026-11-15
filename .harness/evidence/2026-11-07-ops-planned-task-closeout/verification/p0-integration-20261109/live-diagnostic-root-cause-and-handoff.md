# Live diagnostic handoff

Parent task: `2026-11-07-ops-planned-task-closeout`. Tier: L2 runtime diagnosis. Lane: ignored acceptance drivers and new `live-diagnostic-*` evidence only. Product sources, original drivers/binaries, original failures and frozen source snapshots were preserved. No DB, SSH, provider, credential or deployment phase was executed in this lane.

## Bug Report: confirmed O2 harness defect

**Symptom:** correctly formed public log/event projections fail the driver's ordering checks; fresh-process inspection then exits 1. Some denial checks can pass on successful responses.

**Root cause:** `.local/ops-acceptance/o2-live-driver.go:235` and `:251` require envelope code `0`. The inherited public response defines success as `200` in `backend/pkg/common/http/response.go:21` and emits it at `:31`. Therefore both combined ordering predicates fail before their sequence/ownership/attempt conditions can establish success. The same wrong convention in driver lines `306`, `308`, `528` treats **every** nonzero code, including successful `200`, as denial.

**Minimal reproduction:** execute `.local/ops-acceptance/live-diagnostic-o2-contract.exe`. Its source at `.local/ops-acceptance/live-diagnostic-o2-contract.go:20` calls the actual `common.Success` with actual public `TaskLogItem` DTOs containing increasing sequences and nonzero ownership/attempt fields. It performs the original predicate and the same predicate with `common.CodeSuccess`. No database or network is involved.

**Observed result:** HTTP status/envelope code `200`; original ordering predicate false; corrected ordering predicate true; original denial predicate accepts success; exact forbidden predicate rejects success. JSON-decoded `sequence`, `taskHostId`, `taskStepId`, `attemptNo` are `float64`. This rejects the proposed string-sequence explanation for this reproduction. Public DTO definitions are at `backend/modules/business/deploy/deploy_handler.go:700` and `:713`.

**Minimal fix:** acceptance-driver comparisons use `common.CodeSuccess`; denial checks require `common.CodeForbidden`. No platform or product response change is justified. Separate staged `.local/ops-acceptance/live-diagnostic-o2-v2.go:290` and `:306` implement that driver correction. Existing order, ownership, attempt and secret-exclusion predicates remain. Cursor assertions at `:303` and `:317` additionally require successful responses and a positive final sequence, preventing empty error payloads from proving no replay. The v2 prints public projection structural types, never content, and sanitized error types/MySQL numbers/call lines.

**Verification and practical limit:** local reproduction, all three diagnostic Go vets and all three cgo race builds passed. Actual retained log/event ordering still needs the root-run v2; the offline reproduction proves the harness flaw, not the live data's order. Old scope/cursor checks remain preserved and must not be promoted to valid denial/no-replay evidence.

**Similar issues checked:** the wrong response-code convention appears at all five driver locations above. Literal checks of `velero-live-negative-driver.go` and `live-k8s-argo-service-driver.go` did not find this pattern. Fresh-process reports share the same `history` routine (`o2-live-driver.go:229`), so their sequence failures and exit statuses do not independently demonstrate lost committed state. The old pause/resume evidence actually records successful attempt 2 and actual completion despite those failed predicates.

## Other O2 failures: evidence versus inference

- `cancel`: the preserved case records acknowledged canceled status and stale-command conflict, then an operation error **before** the independent completion-marker assertion. The only fallible operation in that interval is `r.exists("done")` at `o2-live-driver.go:478`, whose independent SSH read calls `must(e)` at `:134`. This is a failed independent observation; the suppressed transport reason remains unknown. It is not evidence that product cancellation failed.
- `start-negative`: preserved evidence passes both preview-required and disabled-credential start-denial checks, then fails before the final unstarted/VM-untouched postcondition at `o2-live-driver.go:552`. That expression performs public `GetTask` and independent marker inspection. The old report does not distinguish which failed.
- `failure-retry`: retry control is accepted, then suppression occurs before the retry-marker assertion at `o2-live-driver.go:419`. Public state polling, attempt settlement SQL and independent SSH inspection occur in that interval. The report does not select one; root must capture a safe error class/call line.
- `revoke-credential`, `revoke-source`, `revoke-host`: observed failures occur after task creation and before any pause/control event. They do not reach the revoke APIs. An enum or authorization finding cannot be inferred from them.

**Credential compatibility is established from unchanged source and the start-negative execution:** `UpdateCredential` accepts `active` and `disabled` (`backend/modules/business/deploy/deploy_credential.go:99`). Create defaults to active (`:68`); resolve rejects non-active (`:185`). The public revoke API is `UpdateCredential(..., Status: disabled)`, deletion is `DeleteCredential` (`:123`), and valid secret rotation increments the stored version (`:110`). Do not replace credentials with invalid placeholders. The credential source hash matches the original frozen O2 snapshot.

**Minimal persistent read reproduction for root:** `.local/ops-acceptance/live-diagnostic-o2-inspect.exe` accepts private stdin `{Identity, DSN, TaskID:1}` for the **exact corresponding retained** `ops_p0_o2_live_593d9f0f9442_<case>` database at the existing local MySQL instance. It enforces a read-only SQL session and calls the same public `GetTask(1, DeptID=10, UserID=10)` at source line `67`; no migration, task control, credential mutation, credential decryption or SSH occurs. It emits status/counts or safe error type/MySQL number. Start with `start-negative` and `failure-retry`; failures may be transient, in which case the v2 call-site evidence is required rather than declaring a persistent product defect.

## Argo: exact product failure retained; root cause pending

`argo-product-live-go.txt` shows `TestArgoProductLive` failing at its **first** preview (`backend/modules/business/cicd/argocd/argocd_product_live_test.go:106`, helper `:62`) with `cicd.argocd.app_sync_failed`. Compilation reached the running test. The separate result confirms fixture/auth setup checks passed 8, cleanup exit 0, product exit 1, and stable source hashes. This does not establish successful product sync, diff, rollback or reconciler behavior; old provider-only28 remains separate.

The public key covers several different failure sites, so it cannot uniquely identify the provider cause. The source path is:

1. `argocd_preview.go:123` calls `readApplication`; `argocd_client.go:230` requires HTTP 200 and bounded valid JSON; `:259` requires nonempty UID, resourceVersion and spec.
2. `argocd_preview.go:180` reads manifests; `:183` requires a resolved 40/64-hex revision.
3. `:192` reads managed resources and `:199` performs strict manifest/live diff.

**Namespace absence alone is not a supported root cause:** the frozen source already passes `DestinationNS` into `applicationDiff` at `:199` and fills missing namespaces for supported namespaced kinds at `:250`. Fixture Service manifests omit namespace, but this code explicitly handles that condition. Spec/project/destination drift returns a conflict; unsupported resource/namespace scope returns forbidden. Preserve these checks and the provider permission scope. There is no evidence yet requiring broader permissions or relaxed spec/response validation.

**Concrete staged reproduction for root:** `live-diagnostic-argo-v2-reviewed-phase-request.json` names the new local/remote scripts and their exact hashes. Execute its command only after the original O2 frozen run. It runs the unchanged `TestArgoProductLive`, uses separate exact owned fixtures and new evidence names, preserves original fixture authority and permissions, and records only request method, safe path template, HTTP status and allowlisted application/decoded manifest/live field names/types in `providerTrace`. Provider values, arbitrary labels/annotation/selector keys, queries and credentials stay private. Source and local/remote hashes must match before SSH. Provider decoding errors cannot become a synthetic successful empty object. Public TLS/SSRF remains unchanged; the relay remains explicit and test-only.

Interpret the trace sequentially: first failing/non-200 read identifies the protocol/access stage; all-200 reads with missing/incompatible application or revision fields identify a response-contract condition; reaching both manifests and managed-resources reads narrows the next reproduction to strict diff validation. No product fix is recommended until the actual failed condition is reproduced with evidence.

## Commands, evidence and next steps

Local evidence: `live-diagnostic-20261109T082326Z-verification.json` and its referenced logs; final Python/schema/hash/ignore/format review: `live-diagnostic-final-review.json`. The final review contains current hashes after Go formatting and the final Argo trace additions; the initial staging manifest intentionally retains its earlier hashes. Original drivers remain byte-identical; full product source before/after hashes are equal. No rendered UI evidence is relevant because this lane changes no UI. CodeGraph was not callable, so exact user-specified source files were read; this is recorded as a retrieval limitation.

After the original frozen run, root can:

1. Execute the read-only O2 inspector on retained failed cases using private stdin; preserve its new output. Do not inject SSH credentials or encryption keys into this probe.
2. Build `python .local/ops-acceptance/live-diagnostic-o2-v2-runner.py --phase build`, then run a narrow `--phase execute --case success` before expanding. The build creates separate driver binaries and fresh diagnostic build receipts. Existing diagnostic evidence is exclusively created; a repeated build needs a new revision name.
3. Execute the exact command in `live-diagnostic-argo-v2-reviewed-phase-request.json`, retaining `live-diagnostic-argo-v2-go.txt` and `live-diagnostic-argo-v2-result.json` as new failures/successes. Known SSH errno13 uses the coordinating root's existing approval path; this lane neither attempts nor bypasses it.

Stop condition: confirmed harness root cause and minimal correction are reproduced; separate compiled/staged diagnostics are reviewable. Actual Argo cause and suppressed O2 operations remain pending root execution. Product-source fix ownership is deferred. No overall O2/P0 approval, full IAM/runtime proof or full Go race result is claimed here.

Gate Outcomes: offline product-envelope reproduction caught wrong success/denial code assumptions | original O2 ordering gate false-positive on valid ordered DTOs | original O2 denial predicates admit successful responses
