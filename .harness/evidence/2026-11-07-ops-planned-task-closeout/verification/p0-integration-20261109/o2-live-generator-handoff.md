# O2 real VM generator handoff — execution blocked in restricted lane

Task: `2026-11-07-ops-planned-task-closeout`; tier `L2`; business domain `business/deploy`. Parent: [closeout task](../../../../../docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md), [O2 packet](../../../../../.harness/tasks/2026-08-31-ops-operational-workbench-design/packets/O2-deployment-execution-workbench.md). This is generator preparation and evidence, with no independent approval or O2/P0 closure.

Only new ignored `.local/ops-acceptance/o2-live-*` drivers/scripts and `o2-live-*` evidence were written. Existing product edits, old databases, providers, evidence, SQL, lock, baseline, ledger, commits and publication were preserved. No frontend changes or visual acceptance claim.

## Prepared scope and file plan

Reuse current public `DeployService`, actual production SSH provider, current credential reference APIs, public task preview, public state-version controls, public safe task/history projection and current log/event handlers. Reuse the existing authorized credentialref1/known_hosts helper without emitting configuration or secrets. No mocked transport or substitute capability. Use owner-module service methods to initialize fresh isolated test databases and CMDB/business-scope records; no product SQL edits or provider-independent row-deletion substitute for revocation.

1. `o2-live-driver.go`: current product Go acceptance with independent pinned SSH filesystem verification and fresh OS process reads/recovery.
2. `o2-live-runner.py`: build, read-only preflight and sequential execution phases, private stdin injection, fresh per-case database, exact owned remote directory checks and cleanup.
3. `o2-live-local-verify.py`: pre-execution path/database/server/identity denial and secret-sentinel checks against the actual compiled race driver, plus Python AST/gofmt diagnostics.
4. `o2-live-phase-request.json`: sanitized concrete root execution request with absolute script path, arguments, dependencies, source/artifact hashes, ownership and finite budgets.

The public service is run by the Windows CGO race driver against the actual SSH VM; a Linux amd64 artifact is also built from the same recorded source. The Linux artifact has not been run or installed on the VM. Log/event handlers are exercised through in-process Gin requests with current DataScope inputs, not an authenticated HTTP server or mocked SSH server.

## Ready scenarios; all actual VM acceptance remains pending

| Case              | Intended fresh observable evidence                                                                                                                                                                                                    |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| success           | reviewed credentialref start, frozen package definition, actual marker bytes, stdout/stderr, checkpoint/history, cursor sequence, fresh OS process, inverse marker absence, terminal action denials                                   |
| failure-retry     | real exit17, failed prior-attempt history, public retry, actual completion marker, newer attempt/history, stale-version conflict, inverse host-state denial                                                                           |
| pause-resume      | pause an actual running script, acknowledgement and interrupted marker absence, new OS process resume, actual completion and persisted history                                                                                        |
| cancel            | actual running cancellation, terminal status, stale-view conflict, independent absence after finite remote command budget                                                                                                             |
| concurrent        | two commands on one actual running task's viewed stateVersion; one commit, one conflict; real pause and history                                                                                                                       |
| revoke-credential | public credential disable after actual pause; recovered execution denied and independent completion absent                                                                                                                            |
| revoke-rotation   | public credential rotation changes version; recorded task credential version cannot authorize recovered execution                                                                                                                     |
| revoke-source     | public business-scope disable followed by the viewed resume; record actual public decision and VM side effect; a permissive result is a failed assertion, not acceptance                                                              |
| revoke-host       | public host department ownership change; current task/control/log projection denial                                                                                                                                                   |
| start-negative    | preview required; credential disable after preview denies start and leaves VM untouched                                                                                                                                               |
| hostkey-negative  | actual product SSH host-key mismatch; no VM script marker execution                                                                                                                                                                   |
| cold-restart      | actual owned driver exits75 with an active persisted lease; a fresh queue process respects that lease; wait actual five-minute expiry without editing rows; new process reconstructs queue, attempts, history and actual VM execution |

Database identity is exact `ops_p0_o2_live_<random-run>_<case>` at existing local MySQL33316. Every test database is retained. Runtime DSNs, decrypted credentials and encryption keys are injected only through stdin and never stored in evidence or CLI arguments. No old database is dropped.

Remote ownership is an exact new `/opt/pantheon-p0-20261109/o2-live-<random-run>-<case>` directory. Its resolved parent identity, owner token, allowlisted regular files and exact expected marker SHA256 must match before cleanup. Scripts use harmless markers, an eight-second finite loop and a64MiB shell limit; no remote driver deployment, ports, providers, systemd or other namespaces. Cleanup removes only individually checked files and the resulting empty directory; no recursive deletion or remote process kill. Independent SSH errors cannot pass as absent marker evidence.

## Fresh verification and observed blocker

- `python .local/ops-acceptance/o2-live-runner.py --phase build` → exit0. Current `go vet -mod=readonly`, Windows `CGO_ENABLED=1` with `D:/msys64/mingw64/bin` race driver build, and Linux `GOOS=linux GOARCH=amd64 CGO_ENABLED=0` build all exit0. [Exact commands, exits, timing and artifact SHA256](o2-live-build-result.json). [721 source hashes before](o2-live-build-source-before.json) and [after](o2-live-build-source-after.json) match.
- `python .local/ops-acceptance/o2-live-local-verify.py` → exit0, **7/7**: two Python AST checks, clean gofmt, four executed negative pre-initialization boundaries. Every negative driver invocation correctly exits1, creates no task, and excludes the secret sentinel. [Receipt](o2-live-local-verification.json). These are local script safety tests, not VM negatives.
- `python .local/ops-acceptance/o2-live-runner.py --phase preflight` → exit1, **PermissionError / errno13**. [Sanitized receipt](o2-live-preflight-result.json). One read-only attempt only; no SSH retry or execution bypass. No remote directory, marker, database or provider was created by this lane. No cleanup is due from this lane's preflight.
- Fresh actual VM counts from this lane: **positive0 / negative0 / failed0 / attempted0**. Twelve scenarios are prepared. Earlier protocol13 and previous real VM receipts remain historical and are not recounted as this matrix.
- Full repository `go test -race ./...`, frontend/browser checks and independent review were not run in this acceptance-script lane. The Windows race artifact build is not a full product race-test claim. Product source was not changed.

## Root execution handoff

Concrete request: [o2-live-phase-request.json](../../../../../.local/ops-acceptance/o2-live-phase-request.json).

```powershell
python .local/ops-acceptance/o2-live-runner.py --phase execute
```

Root must execute with its existing authorized execution tool approval; this restricted lane cannot request escalation. Current build/source identity is checked before remote mutations. If another lane changes source after this build, run `--phase build` again before executing. Execute cases sequentially, preserve failed case receipts, and review exact state/cleanup evidence. Optional `--case <name>` is a bounded subset and must not be described as the complete matrix.

Cold restart includes a real five-minute lease wait; other invocations have130-second driver budgets. Root should keep progress visible during the wait. No lease timestamps, attempt status rows or baseline are forged to shorten it.

If a product assertion fails, preserve the original failure/VM state/cleanup receipt and route the smallest source fix through root. This lane has reproduced no product defect because actual VM execution is blocked; no speculative product repair is offered as a verified defect.

## Remaining O2 gaps and review path

- All twelve real VM matrix scenarios await approved root execution and review. Actual runtime positive/negative counts must come from per-case receipts, not planned assertion counts.
- A supplied `DataScopeReq` and owner-module scope/host mutation can prove current public service projection and control behavior. They cannot certify authentication middleware, persisted IAM role/data-grant changes, session revocation or fresh actor-policy rederivation. This driver explicitly leaves those source-auth gaps open.
- In-process log/event cursor checks are not browser/network SSE disconnect/reconnect evidence. Multi-host partial failures, long-log cap/runtime rendering, final browser action matrix and complete visual acceptance remain under their original packet criteria.
- Production schema/migration/retention, full current product verification, independent complete-source review and final functional/visual/production signatures remain open. Evidence preparation does not approve O2 or P0.

Gate Outcomes: exact identity/server/database and secret-sentinel checks pass | build source hashes establish721-file artifact identity | restricted preflight catches errno13 execution boundary | actual VM acceptance and independent approval remain open.
