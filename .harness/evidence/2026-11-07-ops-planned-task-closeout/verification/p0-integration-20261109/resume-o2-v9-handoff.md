# O2 v9 verifier reuse candidate handoff

Task: `2026-11-07-ops-planned-task-closeout`. Tier: L2. Parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md:197`. Lane: scoped O2 acceptance Generator/debugger, local tooling only. Landing boundary: business/deploy acceptance tooling in ignored `.local/ops-acceptance/resume-o2-v9-*`; no platform/system-domain or product implementation change. Review owner: root. This handoff is distinct from the earlier failed CLI report/receipt.

## Finished local outcome

The existing v9 files were continued, compiled, tested and source-sealed. The verifier now lazily authenticates at most one independent, exactly pinned `ssh.Client` per Go driver process, then opens a new bounded session for each verifier command. First handshake failure is cached as failure; no reconnect, retry or command replay exists. Closing or cancelling the verifier terminates its connection. Public DeployService and the product default SSH provider remain the compiled v8 implementation.

The final verification receipt is `resume-o2-v9-candidate-verification-1794274678887620900.json`: 19 checks passed and all five commands exited 0. It records every Go/Python input hash and confirms they stayed unchanged throughout verification. The sealed candidate/root check also exited 0. All 185 actual product dependency files match v8, all 102 prior v7/v8 files in the preservation inventory match, and all 11 files in the continuation failure-preservation inventory match. Existing failed receipts and the copied `resume-o2-v9-generator-first-failed-events.jsonl` remain intact.

Only local tests, local fixture SSH, local compilation, dependency inventory reads and new evidence writes occurred here. No real credential loading, remote execution, remote configuration, product/SQL/lease/ledger/lock change, delegation, model override, persistent toolchain configuration, commit or release occurred. Fixture passwords/host keys are generated or static local test data, not acceptance credentials. Local fixture results are not true-VM acceptance or root approval.

Evidence filenames in this document resolve under `.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/`. Tool filenames resolve under `.local/ops-acceptance/`.

## Grounded reason for this candidate

The complete unchanged v8 matrix `0e2ebc53350e` has 12 cases: 10 passed, 136 positive passes, 26 negative passes, zero failed assertion records, two retained driver failures and matrix exit 1. The two failures are `revoke-rotation` and `revoke-host`, in verifier authentication before pin verification. Cold real-lease restart passed. Its 728.016-second matrix duration and real lease wait were not shortened or replaced for v9.

Root's read-only `resume-server-followup-20261110T011422960483.json` records actual sshd journal categories from the bounded shared matrix window: MaxStartups 42, resets 515, authentication failures 399 and success 110. The journal used the actual sshd SYSLOG_IDENTIFIER. Raw journals remain remotely; raw records were not retained in local evidence. These shared-window counts do not attribute every failure to a specific acceptance client. No product defect is proved by that evidence.

The concrete pressure reduced here is v8's repeated verifier TCP/SSH authentication for every marker command/poll. v9 changes that connection lifetime only. Product SSH connections, credentials, pin values, controls, case order and lease behavior are not substituted. Another first-handshake failure on the VM must still fail and be retained; this candidate cannot guarantee a green remote matrix.

## Exact implementation and preserved contracts

`resume-o2-v9-candidate.diff` contains the exact unified v8-to-v9 driver, original-test and runner diff. Its SHA256 is `694f389fd9eb83bbfaab026c3223ca169c6136217dacb7afdfe42b1a37d80346`.

- `resume-o2-v9-driver.go:131`: `remote` uses the same verifier address and 18-second bound, then dispatches through the process-owned client. `verifier` caches both a successful connection and the first handshake error. `connectVerifier` retains independent fingerprint equality checking and authentication with the existing x/crypto/ssh APIs. `pinCallbacks` is a numeric handshake diagnostic; the reuse test observes exactly one callback for three commands.
- `resume-o2-v9-driver.go:230`: release closes the connection and cancels an in-progress first handshake. The command mutex serializes socket deadline ownership; the separate lifetime mutex permits shutdown while network work runs. The cancellation callback is stopped or joined before reuse so a completed command's old context cannot close the next command's connection. Timeout is terminal for the connection, with no redial.
- `resume-o2-v9-driver.go:269`: every command receives a fresh session/channel and an at-most-18-second deadline. Session open and command I/O are bounded; caller cancellation closes the verifier transport. SSH command exit status is retained. A refused session produces a failure on the existing transport, never a fresh connection or replay.
- `resume-o2-v9-driver.go:623`: after the existing crash-ready event is emitted, verifier release occurs explicitly before the deliberate `os.Exit(75)`. `main` releases on normal/failure exit before its existing result events and `os.Exit(1)` path. Original crash/result event logic stays intact.
- `resume-o2-v9-runner.py:33`: the Windows/Linux dependency inventories use the installed exact Go 1.26.10 executable with process-local matching GOROOT, GOTOOLCHAIN=local and MinGW PATH. The build entry reuses the v9 verification wrapper instead of leaving a duplicate stale build path. Versioned tool/evidence paths are v9.
- `resume-o2-v9-root-execute.py:10`: execution remains guarded by `ROOT_REVIEWED = False`. `--phase check` hashes the product dependencies, candidate files and evidence, and recalculates the actual two-platform dependency inventory without credentials. `--phase execute` rejects before request/resource loading while unreviewed. Execution after review requires a separately sealed `root-reviewed` request.
- `resume-o2-v9-seal.py`: reuses the v8 compiled dependency inventory and requires exact equality for all 185 files; checks fresh verification input/log hashes; creates immutable build/source/offline/diff/request evidence; includes runner, driver, tests, verifier/build/seal/prepare tooling, public pin/helper source, root entry and both binaries in the root request. Reviewed resealing permits exactly the root entry's False-to-True guard change; any other candidate mutation requires new verification.

All 45 original literal `r.check` call sites are preserved verbatim. The new Go source regression parses every original acceptance function except `remote` and compares its tokens with v9, removing only the two explicit verifier-release calls in `run`/`main` and normalizing versioned source filenames. All those bodies match. Original marker/ownership, public controls, data scope, history/projection, fresh child process, lease recovery and error-category logic are preserved.

Python AST comparisons confirm `root_execute_after_review`, cleanup, root remote command and root pin logic are identical after version normalization. This also preserves all cases, per-case isolation, actual cold child/probe/recovery sequencing, crash prerequisite failure-event retention, cleanup and the real five-minute lease wait. The fresh child still runs the executable in a distinct OS process; it owns its own driver instance and independently pinned connection whenever it uses verifier commands. The additional two-driver fixture verifies connection ownership separation; it does not claim a fresh-VM child test.

Case order remains: success, failure-retry, pause-resume, cancel, concurrent, revoke-credential, revoke-rotation, revoke-source, revoke-host, start-negative, hostkey-negative, cold-restart. Ordinary case execution budget is 130 seconds; cold execution 380 seconds with 40 seconds reserved within 420; actual unmodified lease wait bound is 305 seconds; remote command bound is 18 seconds. Exact-owned marker cleanup and retained isolated databases stay unchanged.

## Fresh commands, tests and exits

Local invocation: `python .local/ops-acceptance/resume-o2-v9-build.py`, final attempt `1794274678887620900`, exit 0. Its commands run from `backend/` with `-mod=readonly`. The exact executable is `D:/cache/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.10.windows-amd64/bin/go.exe`. Windows process environment: PATH begins `D:/msys64/mingw64/bin`, CGO_ENABLED=1, GOOS=windows, GOARCH=amd64, GOTOOLCHAIN=local, matching GOROOT, GOMAXPROCS=2 and the v9-only GOCACHE. Linux cross-build changes GOOS=linux and CGO_ENABLED=0. No persistent configuration is written.

Let `DRIVER`, `BASE_TEST`, `TRANSPORT_TEST` denote absolute paths to `resume-o2-v9-driver.go`, `resume-o2-v9-driver_test.go`, `resume-o2-v9-transport_test.go` respectively. These abbreviations are only for readability here; the JSON receipt records the exact arguments.

| Command                                                                   | Exit | Seconds | Fresh log                                       |
| ------------------------------------------------------------------------- | ---: | ------: | ----------------------------------------------- |
| `python resume-o2-v9-root-entry_test.py`                                  |    0 |   0.172 | `resume-o2-v9-verify-1794274678887620900-0.txt` |
| `go vet -mod=readonly DRIVER BASE_TEST TRANSPORT_TEST`                    |    0 |   3.360 | `resume-o2-v9-verify-1794274678887620900-1.txt` |
| `go test -mod=readonly -race -count=1 -v DRIVER BASE_TEST TRANSPORT_TEST` |    0 |   8.140 | `resume-o2-v9-verify-1794274678887620900-2.txt` |
| `go build -mod=readonly -race -o resume-o2-v9-driver.exe DRIVER`          |    0 |   3.735 | `resume-o2-v9-verify-1794274678887620900-3.txt` |
| `go build -mod=readonly -o resume-o2-v9-driver-linux DRIVER` (Linux)      |    0 |   3.265 | `resume-o2-v9-verify-1794274678887620900-4.txt` |

Go reports PASS for 15 top-level tests and their subtests, with no race detector findings. The original four v8 tests remain. Named verifier tests cover:

- `TestVerifierReusesOneAuthenticatedTransportAndExactPinOnce`: three distinct commands/sessions, one accepted TCP connection, one successful authentication and exactly one matching pin callback. Expired contexts from already-completed commands do not poison reuse.
- `TestVerifierChangedPinRejectedWithoutAuthenticationOrRetry`: incorrect pin rejected before authentication, then the same cached failure returned without a second dial/session.
- `TestVerifierFirstHandshakeEOFIsFinal`: first handshake error remains the same failure; no second dial.
- `TestVerifierSessionRefusalNeverReconnects`: explicit channel refusal retained for two separate caller commands, one authenticated transport, zero executed commands.
- `TestVerifierCommandExit17PreservedWithoutReplay`: exact SSH exit 17 retained and a later distinct command uses the same transport; no replay.
- `TestVerifierTimeoutBoundsHandshakeChannelAndCommand`: stalled handshake, channel opening and command execution all return bounded deadline failures; later commands cannot reconnect.
- `TestVerifierCancellationAndShutdownInterruptActiveCommand`: cancellation preserves context.Canceled, shutdown interrupts the command, repeated release is safe and later commands cannot reconnect.
- `TestVerifierShutdownBeforeUseNeverDials` and `TestVerifierShutdownInterruptsFirstHandshake`: closed lifetime prevents an initial dial; shutdown during a stalled first handshake promptly terminates it.
- `TestVerifierIndependentDriverClientsDoNotShareTransport`: two driver instances independently authenticate, and closing one does not affect the other's connection.
- `TestOriginalAcceptanceFunctionBodiesPreservedExceptVerifierRelease`: original acceptance function bodies stay unchanged apart from the allowed release calls and source filename normalization.

The Python root-entry tests all pass: unreviewed execution rejects before validation/resource imports; local checking never invokes the matrix; a generator candidate cannot be promoted by self-verification; a hash mismatch rejects before runner import.

`resume-o2-v9-candidate-offline-final.json` records 17 offline checks, including exact 45 assertions, all 12 cases/order, unchanged root remote/pin/cleanup/matrix ASTs, Python syntax, guard rejection before resource imports, and the in-memory cold crash-prerequisite failure regression. That failure remains exit 1/blocked with original safe initial events retained. Crash-ready exit 75 and its release-before-exit source branch are preserved; actual crash/lease acceptance belongs to root's real matrix.

`resume-o2-v9-process-exits.json` additionally runs the sealed Windows binary with an invalid empty identity: exit 1, failure/result events preserved, no verifier connection or credentials loaded. It runs the unreviewed root executable: exit 1 from the review guard, before request/resource loading. The local fixture command proves exact SSH exit 17 separately.

Sealing command: `python .local/ops-acceptance/resume-o2-v9-seal.py --verification .harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/resume-o2-v9-candidate-verification-1794274678887620900.json`, exit 0. The invoked candidate `--phase check` exits 0 with sourceCount=185, cases=12, guardEnabled=true, remoteExecuted=false and credentialsLoaded=false. Log: `resume-o2-v9-root-candidate-check.txt`.

The three earlier failed v9 verification receipts/logs are retained. Their latest compiler output established inherited GOROOT `D:/Go` (Go 1.26.5) conflicting with the invoked Go 1.26.10 executable. The fix is limited to matching GOROOT in child-process environments. An initial successful verification after that fix is also retained as `resume-o2-v9-candidate-verification.json`; it predates the final lifecycle/source-regression changes and is not the final candidate evidence.

## Source and artifact hashes

The local and evidence copies of `resume-o2-v9-root-phase-request.json` are byte-identical. SHA256: `46aba1c0f2e5c451898e9b0efb54e32206f6e5219b268b27b581bbed037d64e2`. They contain the complete 185-file product hash inventory and the full evidence/log hash map.

Both v9 product inventory JSONs (`build-ready-source-before` and `build-ready-source-after`) have SHA256 `73d338a0a3bf52781d21aa02bfc6d40205cca4b1fde7a357d480853ee63e8c80`; their parsed maps exactly equal v8. Build receipt SHA256: `9ba376368fb8b2838c5e4d0a1e2e6fc4a55a4d45f339e67c3e1e65a1307acf1f`. Final verification receipt SHA256: `e533e5427624357cf0385d395344f842b61aca83980c145a00fa9e466873d34b`.

| Candidate artifact                      | SHA256                                                             |
| --------------------------------------- | ------------------------------------------------------------------ |
| `resume-o2-v9-driver.go`                | `2a1270e2deccb1436a252653a4d73eac863b8d22d9847a6a25c453a1c8182b92` |
| `resume-o2-v9-driver_test.go`           | `5606aaf722584bc7439504860f6d3cf802af81cef16e87e3604b73e3901af2c8` |
| `resume-o2-v9-transport_test.go`        | `c05a5e8dcfaee1a4d4fb58c853ddac089832efb562bbebdfd0e98eedd744f5fe` |
| `resume-o2-v9-runner.py`                | `bb8bb8b799ff7580c7a1d123a28b879fa6c359e7fe3802dc1e00aecaf380c95a` |
| `resume-o2-v9-verify.py`                | `fc348ac5bba18743be65b1614c2d513db8dfc9ecb08e7a284dc10290ae9755de` |
| `resume-o2-v9-build.py`                 | `e1994ba89f55044b0f3efdbf593c5e8366adb4bc03a8fcc5e931683893794d01` |
| `resume-o2-v9-seal.py`                  | `76a71e7360368b02acce2ebbe4edbe70ae0a41bfca8ab7ed29e325ebe0f9b6b5` |
| `resume-o2-v9-prepare.py`               | `2201f2f2aca06931ef3a8e8ac1ac856907f8d1eef961883445cd2dae10a1252b` |
| `resume-o2-v9-root-execute.py`          | `bab26199a2292be44fa9087f09d27688e9dc4d3a2a48001934476cfe2deaccbf` |
| `resume-o2-v9-root-entry_test.py`       | `ac965322a927b37e5b49b8c294605165a60f27e97924d81856ca18deaa3141fc` |
| `resume-o2-v9-driver.exe`               | `817db179039da303fcc657ad36c18e5594830af3600517618603d16dd87ffc4b` |
| `resume-o2-v9-driver-linux`             | `27adb04f2b43d3c32e94b362ab58ea8854a3e1766b15e934b1af2c05eb584278` |
| Existing unchanged `remote.py`          | `37acccd12ba3b30e38182f11101821f110355ed660c4cf559d91cc1c638e1b5e` |
| Existing unchanged public `known_hosts` | `45a3379d7e5d75b57e4d47c37ea424b611ddc56ee6d2a70fdc89c05c1ae8213e` |

## Root-owned executable action and guard

The candidate is concrete and locally checked; remote acceptance is still root-owned under the user's existing test-server authorization. This lane did not enable ROOT_REVIEWED or invoke the matrix. Review this handoff, exact diff, full request and fresh local receipts first.

The currently safe executable check, from the ops root, is:

```powershell
python .local/ops-acceptance/resume-o2-v9-root-execute.py --phase check --request .local/ops-acceptance/resume-o2-v9-root-phase-request.json
```

For execution after root review:

1. Root changes only `ROOT_REVIEWED = False` to `ROOT_REVIEWED = True` in `resume-o2-v9-root-execute.py:10`, preserving UTF-8/LF. Do not edit runner/product/assertions/credentials or execute against the old entry hash.
2. Root runs `python .local/ops-acceptance/resume-o2-v9-seal.py --reviewed`. This rechecks dependencies, verified sources, binaries and evidence and creates NEW immutable local/evidence `resume-o2-v9-root-reviewed-phase-request.json` copies binding the reviewed entry. It rejects any candidate change beyond that one guard. The unchanged runner/build identity remains valid because the review entry is sealed in the root request separately from the runner's build artifacts.
3. Root runs the reviewed check and the exact full-matrix entry below through the existing authorized root execution mechanism:

```powershell
python .local/ops-acceptance/resume-o2-v9-root-execute.py --phase check --request .local/ops-acceptance/resume-o2-v9-root-reviewed-phase-request.json
python .local/ops-acceptance/resume-o2-v9-root-execute.py --phase execute --request .local/ops-acceptance/resume-o2-v9-root-reviewed-phase-request.json
```

The root entry offers no case-subset switch. It validates all source/artifact/evidence hashes and the complete case order, then calls the unchanged `root_execute_after_review(CASES)` body. That body uses the existing authenticated root helper, privately injects credentials/DSN through stdin, creates fresh isolated per-case schemas and exact-owned marker directories, retains evidence/failures, keeps the real cold lease wait and performs the original exact-owned cleanup. Product sources/lease records are not edited by this candidate. Preserve failure exits and receipts; no reconnect/retry/replay to obtain a pass.

If root changes any verified runner/driver/test/tooling source beyond the guard, the reviewed seal rejects it. Build/verify the revised candidate and issue a new immutable request instead of executing it under this request's old hashes.

## Remaining acceptance limits and stop condition

Local candidate implementation, named-file vet/race, two-platform builds, original-contract regression, source/artifact sealing, root guard/check and this detailed handoff are complete. Root still owns the actual v9 12-case VM matrix and subsequent review; there is no v9 matrix receipt or independent O2/P0 approval here. MaxStartups journal counts do not prove the cause of every client's failure. Full IAM middleware/session/role/data-grant derivation and browser/SSE network acceptance remain outside this leaf.

Gate Outcomes: final source/function comparison preserved the original acceptance | named transport tests proved one pinned authenticated connection and separate bounded sessions | EOF/pin/channel/timeout/cancel tests retained failures without reconnect or replay | exact process exit1 and crash-ready exit75 contract preserved | 185 product dependencies, 102 prior v7/v8 artifacts and 11 continuation artifacts sealed unchanged | root-reviewed true-VM acceptance remains pending.
