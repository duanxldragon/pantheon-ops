---
title: Velero v8 LSP clean generator handoff
doc_type: Evidence
layer: business
status: Active
updated_at: 2026-11-10
---

L2 parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`.
Generator local implementation and verification are complete. Business/k8s
acceptance tooling, logical separation; platform/system/product source unchanged.
Root must obtain independent CODE/ARCH review before executing remote phases.
Source checks do not establish actual provider success or P0 closure.

Seven new `resume-velero-s3-v8-{fixture,generate,native,remote,root,seals,test}.py`
files add typed AST call traversal, nested-call/function shape guards, required
source segments, checked exception metadata, required spec/loaders, explicit
predecessor guards and a truthful heterogeneous test-record type. The policy
function supplies empty type parameters required by Python 3.12's AST signature.
No diagnostic suppression, stubs, checking-mode override or broad fallback.

Captured JSON, predecessor import bytes and helper execution still consume the
same sealed buffers. Unchanged v6 native (`14f9ca3cc471bbf754da27fcdb71b2e22bd01153eefb02088dc43bb543527081`)
and fixture (`e53facea2f9b9822855de75f19aeab8210f9b4debe6da5f76b2fb84803d86fa3`)
execute through exact seals. V7 sources are protected history, not executed.
All original request/five/mirror/runtime/product/driver source identities remain.
The fixed 17 absent objects, exact BSL/CRD association, current authoritative
group-absence proof and forbidden/auth/network failure refusal remain. Native,
mirror UID/spec/controller/namespace LIST, S3 auth, official dry-run, CRD
Established-before-CR CREATE, driver Backup/Restore, cleanup and Unknown refusal
remain. No install, image, native or first-five replay.

| Bound artifact                                                       | SHA256                                                             |
| -------------------------------------------------------------------- | ------------------------------------------------------------------ |
| [Final v8 request](resume-velero-s3-v8-root-request-final.json)      | `aec0783ef37a191284c16f9fe69f96a06dcf94b846de62de2efdd0dcffae232f` |
| Immutable v7 final request                                           | `05a39bef4df96a8c08ad78b17373d4968b6116be8ee8a2682e7708147517ad6e` |
| Immutable failed v6 diagnostic                                       | `4a486df4fa7f3b35f5a5c7596dd0045e76bfdd18fe1facb2a806af3d769b305e` |
| Immutable v6 absence inspection                                      | `b8154da9dea668ca334fe3f6dae10134361ca79c722a035c83a6cea594507a7f` |
| [True LSP artifact](python-lsp-velero-v8-20261110T041748927902.json) | `298bc97e478df2a79db1348473198a1938933432dc2ecbf0fb75d0927cb0e82e` |

[Exclusive preparation](resume-velero-s3-v8-preparation-final.json) preserves all
180 v7 protected pins plus seven v7 source pins; 198 protected files and seven v8
scripts verify. The exact 29-file v6 remote baseline remains; v7 was never uploaded.
Earlier candidates, failures and receipts remain. Do not rerun generation.

[Fresh verification](resume-velero-s3-v8-tests.json): 55 v8 tests (45 existing
logical tests plus 10 malformed-AST/None/loader guards), unchanged v6 46, v5 32,
v4 25: 158 passed, zero skips. Seven syntax compilations and actual root
`check-inputs` pass. Real Pyright 1.1.414 LSP: seven files, zero errors/warnings,
`basic`, `openFilesOnly`, actual acceptance-tool extraPath. Every LSP source hash
matches the final request. Exact commands, outputs and hashes are recorded.

Root supplies the request hash externally; never infer success from old failures:

```powershell
$v8SHA = 'aec0783ef37a191284c16f9fe69f96a06dcf94b846de62de2efdd0dcffae232f'
$v8EV = '.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109'
if ((Get-FileHash "$v8EV/resume-velero-s3-v8-root-request-final.json").Hash.ToLowerInvariant() -ne $v8SHA) { throw 'request drift' }
python -B .local/ops-acceptance/resume-velero-s3-v8-root.py check-inputs
if ($LASTEXITCODE -ne 0) { throw 'inputs refused' }
```

Run each phase separately. Inspect its complete successful artifact before
supplying that receipt name and explicit output hash to the next phase.

```powershell
$d = python -B .local/ops-acceptance/resume-velero-s3-v8-root.py diagnostic --request-sha256 $v8SHA | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $d.exitCode -ne 0) { throw 'diagnostic refused' }
Get-Content "$v8EV/$($d.artifact)"
```

```powershell
$c = python -B .local/ops-acceptance/resume-velero-s3-v8-root.py complete --request-sha256 $v8SHA --receipt $d.artifact --sha256 $d.sha256 | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $c.exitCode -ne 0) { throw 'complete refused' }
Get-Content "$v8EV/$($c.artifact)"
```

```powershell
$v = python -B .local/ops-acceptance/resume-velero-s3-v8-root.py validate --request-sha256 $v8SHA --receipt $c.artifact --sha256 $c.sha256 | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $v.exitCode -ne 0) { throw 'validate refused' }
Get-Content "$v8EV/$($v.artifact)"
```

```powershell
$x = python -B .local/ops-acceptance/resume-velero-s3-v8-root.py cleanup --request-sha256 $v8SHA --receipt $v.artifact --sha256 $v.sha256 | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $x.exitCode -ne 0) { throw 'cleanup refused' }
Get-Content "$v8EV/$($x.artifact)"
```

Failed/stale/wrong-phase/hash predecessors refuse. Preserve partial state.
No remote, credential/configuration, old database/native S3, delegation/model or
release operations occurred. Root downstream must establish actual diagnostic,
BSL Available, Backup/Restore, S3 persistence and cleanup evidence. Independent
approval, runtime and P0 closure remain pending. No product build/Go race claim.
CodeGraph tools were unavailable; supplied affected paths were read directly.

Gate Outcomes: AST/None/source/receipt guards caught malformed inputs before
command/CREATE/connect; no checking suppression or provider-success claim.
