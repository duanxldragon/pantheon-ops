# O2 v11 source-stable Generator handoff

Parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`; L2.
Only NEW v11 tools/evidence were created. No product/Base/source, credential,
remote, original approval/request/evidence mutations or self-approval occurred.

Request SHA256: **`ffbc6666c896de2b00ec2eb799aa96e41e9b4d3bb0e5a9417d049db361649e51`**.
State: SEALED-CANDIDATE, executionDisabled=true, real cases/attempts=0.

Exact artifacts use `.local/ops-acceptance/continue-o2-live-v11-` plus these suffixes:

| Suffix            | SHA256                                                           |
| ----------------- | ---------------------------------------------------------------- |
| contract.py       | d00052ca5fab99fb73b2f585eff3c295c830eac7eca77fe7b64395ddfbe1fbea |
| prepare.py        | 832a06eb8e8879cbe6e029ae5e7c05ae47ba928af7e296617b1350053db1bd2b |
| driver.go         | cf1f34beb1dfd2daf019cdf3f6203fbb14c7f58b0f51cce8e4a81f52bdfed7c0 |
| driver_test.go    | 312c2f99d16a7110f53cb385969f7e397f1fb58ce08a6957464b38a401b8ef46 |
| transport_test.go | c05a5e8dcfaee1a4d4fb58c853ddac089832efb562bbebdfd0e98eedd744f5fe |
| runner.py         | 890f08ba044d80870ce087cb0ee5a98078602064ee9fab7a9a12bf65e1257f09 |
| verify.py         | d1f7f930c9ca2417f952a052ebb391c8245099ca9843c6fcd105e049d943fb5f |
| seal.py           | c6a6158359a721944b0ecbab1aaf6e63a93a16866ca6259621075fdee3296168 |
| root-execute.py   | 794d18bd6520851808d0dd238bfcb952c9eef1ead002ec3041d5c7ff3821336a |
| contract_test.py  | 124eae8efc1da67d52efbe59fce151840e54bd400ce84530a0d304562533105b |
| driver.exe        | b7caa8f061c93cac95ab56740f8f3c630d67bd7e2fa0a12878f7781a5c2f775a |
| driver-linux      | 4a973b249550159ab4819d617f7d13f6bf05a3c6fc7df359c55821804f872ff9 |
| generate.py       | 18010721eca37889a86c9a0bffc9d1721a0c9d0ed3f4e7d58786f534b1597655 |

`artifact-hashes.json` enumerates 108 exact new artifacts/evidence/logs; SHA256
`d4507e25089764ff909491ee0bdc78b26cb8858d0b4462685f862596f5e0363b`.

Windows named-file AND owner-package discovery now uses actual `-race`, retaining
non-race Windows and Linux selectors. Dynamic selection binds Go/Cgo/C/S/Syso/
embed/import/module inputs plus selected MinGW headers/linker inputs/native tools.
No fixed-count fallback. Before/after **5,522 source/390 tool** inputs match;
all **5,532 archived members** were rehashed and matched locally.

FINAL receipt: `local-verification-1794299401982924300/receipt.json`, SHA256
`39f049a42f95bb6d011e4769cb5dc2d1e283676474b49a52b5b0577861df1d5f`.
Seven commands exit0: 12 Python regressions (original11 retained), gofmt, native
gopls, vet, original15 Go race tests, Windows race binary, Linux binary.
Windows uses required MinGW/CGO1. Verified build metadata: Go1.26.10/amd64,
Windows race=true/CGO1 and Linux non-race/CGO0. Eight Python AST checks pass.
Logs: `.local/ops-acceptance/continue-o2-live-v11-verification-logs-1794299401982924300/command-{0..6}.txt`;
metadata logs share that directory. All logs/caches stay `.local`.

Real RED/GREEN: `race-red-green-1794299531962868300.json`. Actual old discovery
omits race Go/Syso; changed local byte copies leave its inventory equal. Actual
new discovery selects both and `validate_request` rejects both drifts. External
originals are unchanged. `request-bound-gates.json` records sealed check exit0,
unauthorized execution exit1 and rejection of original owner receipts as v11
root authority.

Original12 cases/45 assertions, Go tests and protected runner bodies remain
equivalent modulo filenames. Default SSH/single verifier connection/no replay,
physical lease/cold restart/exact cleanup remain intact. Runtime artifact and
durable attempt paths use v11. Preserved v10 groups: **12 candidate/27 evidence/
188 historical/65 owner**; old inventory **5,246/390**, zero mismatches. Successor
protects232 history pins, including sealed v10 request
`f5bdc55e94a56dbeec56429b20189eeedd4cfc5524a85cdf81ce0bd0f294c2a2`.

Dispatcher must obtain independent v11 CODE APPROVE and ARCH CLEAR reports,
then DISPATCHER AUTHORIZE. Receipt contract remains **o2-v10-review-v1**:
role=code-reviewer/architect/dispatcher, matching standalone verdict line,
exitCode0, subjectSHA256=actual v11 request SHA, report=repository-relative path,
reportSHA256=exact bytes. Original owner reports/receipts retain their unchanged
schema/subject; they grant no new root authority. Authorization JSON uses
o2-v11-authorization-v1, requestSHA256 and codeReceipt/archReceipt/dispatcherReceipt
references, each containing path/sha256, with three distinct receipts.

Exact subsequent Dispatcher commands, after those receipts and authorization:

```powershell
$ev='.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/o2-current-live-v11'
$req="$ev/sealed-candidate-request.json"
python -B .local/ops-acceptance/continue-o2-live-v11-root-execute.py --phase check --request $req --authorization "$ev/authorization.json"
if($LASTEXITCODE -ne 0){throw 'Request/review binding failed'}
python -B .local/ops-acceptance/continue-o2-live-v11-root-execute.py --phase execute --request $req --authorization "$ev/authorization.json"
```

Independent v11 reviews, Dispatcher authorization, real VM/cold-restart execution,
public Base identity/IAM/audit/HTTP/browser/SSE and P0 remain open.
