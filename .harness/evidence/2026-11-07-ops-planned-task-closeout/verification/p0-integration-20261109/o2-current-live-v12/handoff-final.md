# O2 v12 Generator handoff

Parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`; L2.
NEW v12 tooling is verified and sealed. Request SHA256: **`eb0a1cf9b56afec530deb0eeea7149389b64ef5bfbe7d26b2da9d6705ae24d9d`**.
State: SEALED-CANDIDATE; executionDisabled=true; runtime cases/attempts=0.

Artifacts use `.local/ops-acceptance/continue-o2-live-v12-` plus these suffixes:

| Suffix                      | SHA256                                                           |
| --------------------------- | ---------------------------------------------------------------- |
| contract.py                 | 2a7f7f067f4c9a5a85a085c1c1c679fe6cae7c71b911383cc1ace30a410d6b0e |
| prepare.py                  | 1b8c22c6296624c20c78e20b8bc556da02147739f9d7fa752fb428763ed29d16 |
| driver.go                   | d58c03543f65416ed641fe53949bed70aca59b2a37226502aa18c9b090bd6433 |
| driver_test.go              | e4e541a0756de3854ad16d1022a87a7e65afb0f3708094f5b82cf8841f48a76f |
| transport_test.go           | c05a5e8dcfaee1a4d4fb58c853ddac089832efb562bbebdfd0e98eedd744f5fe |
| runner.py                   | e1322bb41cf2bef7ac9a66950175e9a0e791fd9143364ae19152d6e57371f669 |
| verify.py                   | 6a76e8b16848e2c01531230b0782ab505b3bbaba14822d69a45a7283eec587fc |
| seal.py                     | c53f2d134fe889266ebec6562763c1f5a0e7aae37c43fa7d2ae88c9ecf2e6c67 |
| root-execute.py             | 9eb7cf193bc4a3990244c3a23c46dbf4117100ca3e86a380c01b5b2ba831dd70 |
| contract_test.py            | 277fabb8d7921866d66ff09d60e59eb76990a964ab6bde890437f80102d8e74b |
| native.py                   | dadee9fac55647ddac2f558703c1cb475a1214a3fb49436d7b1ca9a7a5b01ef1 |
| driver.exe                  | df5963eca841c183d9c9d82e69ccbd17554c092082c68070e2f8a61b7af6b6a8 |
| driver-linux                | 8b8f44e852120c653aeaef47c551b2dad510808771fd5c084e2ee1b6a7692a05 |
| generate.py (creation only) | 765d74fc8546a2b5ccaa478d8b22ad9e3444521a8f65a31443ab2fd8577c4d22 |

The 13 candidate hashes include all 11 helpers and two binaries; generate.py is separately recorded.
`artifact-hashes.json` records 491 exact build/evidence/log artifacts; SHA256 `5df95e85df6d463c6639068aa2ac7c1c5a099774250ea41d8e02534bfc0c2182`.
The final handoff/receipt and manifest itself are excluded to avoid circular hashes.

Recursive PE import inspection and native Windows loader file-handle observations bind actual selected compiler,
assembler, linker and objdump origins, environment and ordered search context across 18 tool/cwd probes.
Before/after **5522 source/401 tool** hashes match,
including **16 native pins/10 non-OS MinGW DLLs** with stable file metadata.
OS loader, Windows System DLLs and API sets remain trusted platform inputs; OS/kernel bytes are not pinned.
Foreign/replaced origins, missing/ambiguous loader imports and preceding search shadows reject.
Unused lower-priority PATH candidates are recorded without changing deterministic first-match resolution.

FINAL verification: `local-verification-1794303434945656000/receipt.json`,
SHA256 `c4833ccdbccfd9a657f79bb024b1afda2b78f31286acdcc71fe477ab71fc53dc`. Seven commands exit0: Python regressions, gofmt,
native gopls, vet, Go race tests, Windows race build, Linux build. **15 Python tests (original12+3),
original15 Go tests, zero skips, nine actual Python AST parses** pass.
Build metadata confirms Go1.26.10/amd64, Windows required MinGW/CGO1/race and Linux CGO0/non-race.
Logs/caches/temporary files stay `.local`; exact command arrays and log hashes are in the receipt.
All **5549 archived members** were rehashed: 5,522 source+16 native+11 helpers; zero differences.
Archive proof SHA256: `caa7e436f10af3f9cc5c34cba08b51becc0b9ebb12428764fbca303655d81377`.

Actual RED/GREEN discovery preserves exact v11 request validation: local-copy changes to primary
`cc1.exe → libgmp-10.dll` and transitive `libgcc_s_seh-1.dll → libwinpthread-1.dll`
produce zero v11 DLL hash calls and accepted requests; v12 hashes both twice and rejects both drifts.
External originals remain unchanged. Race Go/Syso regressions also reject both local-copy drifts.
`request-bound-gates.json`: actual sealed check exit0, unauthorized execution exit1, two old owner receipt
subject rejections, zero attempts. A wrapper's duplicate PATH prefix correctly rejected; its failure log is preserved.
The earlier incomplete v10 tempfile test wiring failure is preserved separately and fixed only in NEW v12.

Original12 cases/45 assertions, default SSH/single verifier, no replay, physical lease/cold restart,
process restart, exact cleanup and exit codes remain unchanged. Driver/tests/runner/root entry are equivalent
modulo version filenames. Original product/Base and v11 request `ffbc6666c896de2b00ec2eb799aa96e41e9b4d3bb0e5a9417d049db361649e51` remain immutable;
original groups 12 candidate/88 evidence/232 history/65 owner match; successor protects **358** history pins.
No remote, credentials, owner mutations, delegation, model override, waiver or self-approval.

Dispatcher requires independent NEW CODE APPROVE, ARCH CLEAR and DISPATCHER AUTHORIZE receipts.
Receipt schema stays `o2-v10-review-v1`; role/verdict/exitCode0/reportSHA256 bind this exact request subject.
Original owner receipts grant no new root authority. Authorization schema `o2-v12-authorization-v1`
contains requestSHA256 and three distinct codeReceipt/archReceipt/dispatcherReceipt references with path/sha256.
Run in the bound loader environment, after those receipts and authorization:

```powershell
$ev='.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/o2-current-live-v12'
$req="$ev/sealed-candidate-request.json"
python -B .local/ops-acceptance/continue-o2-live-v12-root-execute.py --phase check --request $req --authorization "$ev/authorization.json"
if($LASTEXITCODE -ne 0){throw 'Request/review binding failed'}
python -B .local/ops-acceptance/continue-o2-live-v12-root-execute.py --phase execute --request $req --authorization "$ev/authorization.json"
```

Independent final NEW CODE/ARCH, public Base identity/IAM/audit/HTTP/browser/SSE, current VM/cold restart
and overall P0 remain open. Gate Outcomes: DLL closure caught primary/transitive drift; loader context caught drift.
