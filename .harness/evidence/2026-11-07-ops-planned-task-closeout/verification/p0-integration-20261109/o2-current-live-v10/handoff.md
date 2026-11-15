# O2 current-source v10 proposal — UNSEALED, execution disabled

Parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`; L2,
business VM acceptance. Generator scope is ONLY new
`.local/ops-acceptance/continue-o2-live-v10-*` and this new evidence directory.
No product/backend/frontend/Base, old helper/evidence, credentials, remote,
release/lock, delegation/model override or gate-waiver changes occurred.

From the Ops root, these aliases resolve every evidence path below:

```powershell
$ev='.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/o2-current-live-v10'
$owner="$ev/../o2-service-transaction-20261110"
```

## Concrete implementation

New helpers: `continue-o2-live-v10-{contract,prepare,runner,verify,seal,root-execute}.py`,
`continue-o2-live-v10-contract_test.py`, and
`continue-o2-live-v10-{driver,driver_test,transport_test}.go`.
The Go files are exact v9 copies after version-filename replacement. Original
tests and all 45 assertion calls are preserved; changed assertions: **none**.
Python AST checks preserve the matrix, remote command, pin and cleanup bodies.

The 12 cases retain their order, actual default Deploy SSH, independent exact
hostkey verification, one authenticated verifier connection per driver process,
no transport retries/reconnect/replay, result-dependent exact ownership cleanup,
exit1 failures, exit75 deliberate crash, and exact SSH command statuses. Physical
cold crash/new-process probing/recovery and actual unmodified lease waits remain
in the executable matrix. No production timeout/lease changes were introduced.

`contract.py` discovers current Windows named-file tests, linked
deploy/service/capability package-test dependencies and Linux build dependencies
using `go list -buildvcs=false -mod=readonly -deps -test -json`. It pins actual
module/stdlib/transitive source, Go module inputs, native tools and installed
Python helper dependencies. Historical helper/evidence hashes are a separate
protected inventory; there is no 185-count assumption or v8-source equality.

`root-execute.py` uses new independently reviewed authorization receipts rather
than a mutable reviewed flag. Missing authorization rejects before runner/private
resource imports. An exclusive durable attempt receipt consumes the request
before execution and rejects replay, including after a failed start.

## Fresh local evidence

Latest: `$ev/local-verification-1794294352885167700/receipt.json`.
All seven commands exit0: 11 Python regressions; gofmt; native gopls; named-file
vet; 15 Go tests with `-race -count=1`; Windows race build; Linux build.
Python AST diagnostics pass. Windows uses MinGW `D:/msys64/mingw64/bin`, CGO1,
Go1.26.10 and task-local `continue-o2-live-v10-gocache`; no test DSN was needed.
Exact source/helper bytes, go-list streams, logs and transient binaries are
archived. Before/after pins match **for that local snapshot**, not a final freeze.

`process-exits.json` records actual native invalid-identity exit1 and both
UNSEALED check/unauthorized execution exit1. `artifact-hashes.json` holds complete
hashes. The first verification remains FAILED despite seven passing commands:
`attempt_ownership.go` drifted; see `retained-drift-failure.json`.
Original protected pins remain unchanged. Inventory entries are not runtime cases;
actual runtime cases executed: **0**.

SHA256:

- Latest receipt: `48c682a8fb068f833f7f36ad3a61feba34c0f67c8d6e8642fffbb94048989aee`.
- Latest `unsealed-request.json`: `6bcc0a1651aaf6cb2e9e22b9a52d7ceb3f0872eb30c3235313938a7757b62556`.
- Current `source-before.json`: `c26f290a4b310a70ff41aeb39371b898f2e9b89af022dbed3bb3da92afbb82d8`.
- Protected OLD inventory: `ac492f907d94f50e7509b7830d1a76bfa83b4897fcabd11ca0ee26d62375e495`.

## Later freeze/build/seal prerequisites and exact commands

Missing now: `$owner/handoff.md` plus final independent CODE APPROVE and ARCH
CLEAR reports/receipts. Dispatcher must obtain these; Generator issues no approvals.
Receipt contract: `schema=o2-v10-review-v1`, role, verdict, exitCode0,
subjectSHA256, report path and reportSHA256. Report verdict occupies its own line.
Owner receipts additionally assert sourceStable and pin sourceHashes, including
capability.go, service/state.go, deploy/target_selection.go and all owner changes;
subjectSHA256 binds the final owner handoff. Use the actual final reviewed receipt
paths if their names differ from the following proposed names.

```powershell
$code="$owner/final-code-report.receipt.json"
$arch="$owner/final-arch-report.receipt.json"
$run=@(python -B .local/ops-acceptance/continue-o2-live-v10-verify.py --final --owner-code $code --owner-arch $arch)
if($LASTEXITCODE -ne 0){throw 'Final verification failed; preserve evidence'}
$receipt=($run[-1] | ConvertFrom-Json).receipt
python -B .local/ops-acceptance/continue-o2-live-v10-seal.py --verification $receipt --owner-code $code --owner-arch $arch
```

Then independently review `sealed-candidate-request.json`: CODE APPROVE,
ARCH CLEAR and DISPATCHER AUTHORIZE receipts bind that exact request SHA256.
New `authorization.json` uses schema `o2-v10-authorization-v1`, requestSHA256 and
codeReceipt/archReceipt/dispatcherReceipt references, each with path and sha256.
Only Dispatcher performs the following actual-host action after those reviews:

```powershell
python -B .local/ops-acceptance/continue-o2-live-v10-root-execute.py --phase execute --request "$ev/sealed-candidate-request.json" --authorization "$ev/authorization.json"
```

Host remains115.190.251.143; private connect material is loaded only during
authorized root execution. Remaining gaps: final current-source freeze,
independent CODE/ARCH/Dispatcher approval, real 12-case VM/cold restart,
public HTTP/browser/SSE composition and full repository gates.
**This task does not implement public Base identity/audit acceptance. No P0 closeout.**

Gate Outcomes: drift caught a real concurrent edit; old185 substitution, bad
review/receipt/hash, missing authority and replay regressions reject; no waiver.
