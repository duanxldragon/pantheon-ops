---
title: Sealed Velero alias v2 proposal handoff
doc_type: Evidence
layer: business
status: Active
updated_at: 2026-11-10
---

L2 parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`. Business runtime acceptance tooling in Ops, logical separation; platform/system/product sources are untouched. CODE/ARCH reviewed v1 and requested changes; both reports remain immutable. Final v2 requires independent CODE/ARCH review before Dispatcher performs any remote phase. Native reviewer failure503 does not grant self-approval.

Only five NEW helpers under `.local/ops-acceptance/` and NEW `continue-velero-alias-v2-*` parent evidence were written. The smallest repair reuses captured original inspection, finite fixture, driver, validation and cleanup functions plus stdlib. It keeps full native/22 UID/spec/Established/RBAC/namespace/controller/mirror checks and both repaired alias/index/manifest/config/layer checks at validation boundaries. The 11 original finite fixtures require exact captured specs, receipted UIDs and content; arbitrary fixture labels cannot bypass the original namespace allowlist. Predecessor freshness is checked after preflight and at original CREATE/token/DELETE/tag effects. Service defaults/Endpoints and native credential bytes remain pinned. Every adapted original function keeps CapturedSyntax and original loader globals; changed provider expressions refuse before sentinel effects.

Diagnostic uploads have finite intent/completed/uncertain records and exact two-name metadata/hash inspection. Lost replies and earlier attempted phases refuse replay. Tagging remains exactly TWO absent normalized aliases in `k8s.io`, with no force/delete/import. Original22inventory bytes, old requests, failures and partial journals remain unchanged. Provider phase remains truthful `alias-recovery`; original BackupRestore/S3/finite cleanup assertions and public authority gaps remain.

| Helper filename                      | SHA256                                                           |
| ------------------------------------ | ---------------------------------------------------------------- |
| continue-velero-alias-v2-contract.py | 4582eb92ef30ce23e3f8c20e513c0925de5b8ffea9a2d45cfeb8f2671a615a66 |
| continue-velero-alias-v2-remote.py   | acee8606947ed303e0eb2f9c31abbf61d2879192f7b61a1c814bd351fd55d2e6 |
| continue-velero-alias-v2-root.py     | f437de93590d61740b2e2a962078fcb92873a679ff445b0852091e915f6f1c7d |
| continue-velero-alias-v2-generate.py | d6835a601c43f46a1613194293dea9c1f016e2699504043f1c2b7231d8f8a3ad |
| continue-velero-alias-v2-test.py     | 47c614c02eabeefd703e6c3ba3067e1bd34fa0659e1ed8019f8fbf3c413407d8 |

EV: `.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109`.

- [Request](continue-velero-alias-v2-request.json): `4bc6858b5f1ad5ff531339b8152fcd24b3a9ef53ed367f7f5720fbc158d22292`; baseline equals unchanged original35files, plus two finite new helpers. New generation ran exactly once; do not rerun any generator.
- [Verification](continue-velero-alias-v2-verification.json): `42e0c57c09f5e40e802aafb9667b5b8d70b7d4ccb00a9f16120717cc54a083d5`. 45 v2 + 185 original tests = 230 passes, zero skips; five in-memory compilations. Two ARCH RED assertions fail unchanged v1; both independent CODE counterexamples reproduce. Current v2 refuses those defects.
- [Actual Pyright 1.1.414 LSP](continue-velero-alias-v2-lsp-final-20261110T064237081854.json): `01457ff5dcfb696818fa99de6138fb2143fa1864a164ebc0605bfe1b076e34b0`, five files, zero errors/warnings, no suppressed diagnostics.
- [Sealing](continue-velero-alias-v2-sealing.json): `47207c0cbe377d332eac146a3ab29e25a748319c973e23712bd600093cca58f2`; request/verification/LSP/sealing source hashes agree; all223 protected files unchanged. [Commands](continue-velero-alias-v2-commands.json) retains exact outputs, including RED evidence and failures.

Dispatcher commands, AFTER independent final-v2 review. Run each phase separately; inspect the complete successful receipt before using its explicit name/hash. Predecessors expire after30minutes.

```powershell
$aEV = '.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109'
$aTool = '.local/ops-acceptance/continue-velero-alias-v2-root.py'
$aSHA = '4bc6858b5f1ad5ff531339b8152fcd24b3a9ef53ed367f7f5720fbc158d22292'
if ((Get-FileHash $aTool -Algorithm SHA256).Hash.ToLower() -ne 'f437de93590d61740b2e2a962078fcb92873a679ff445b0852091e915f6f1c7d') { throw 'root hash drift' }
if ((Get-FileHash "$aEV/continue-velero-alias-v2-request.json" -Algorithm SHA256).Hash.ToLower() -ne $aSHA) { throw 'request hash drift' }
python -B $aTool check-inputs --request-sha256 $aSHA
if ($LASTEXITCODE) { throw 'inputs refused' }
function Inspect-Receipt($x) {
  $path = "$aEV/$($x.artifact)"
  if ((Get-FileHash $path -Algorithm SHA256).Hash.ToLower() -ne $x.sha256) { throw 'receipt hash drift' }
  Get-Content $path
  if ($x.exitCode) { throw 'failed phase: inspect only; no replay' }
}
```

```powershell
$d = python -B $aTool diagnostic --request-sha256 $aSHA | ConvertFrom-Json
Inspect-Receipt $d
```

```powershell
$r = python -B $aTool repair --request-sha256 $aSHA --receipt $d.artifact --sha256 $d.sha256 | ConvertFrom-Json
Inspect-Receipt $r
```

```powershell
$v = python -B $aTool validate --request-sha256 $aSHA --receipt $r.artifact --sha256 $r.sha256 | ConvertFrom-Json
Inspect-Receipt $v
```

```powershell
$c = python -B $aTool cleanup --request-sha256 $aSHA --receipt $v.artifact --sha256 $v.sha256 | ConvertFrom-Json
Inspect-Receipt $c
```

For failure, inspect its artifact/hash and partial records. Bind `$failed` to that exact returned result, then:

```powershell
python -B $aTool inspect-failed --request-sha256 $aSHA --receipt $failed.artifact --sha256 $failed.sha256
```

This performs read-only finite inspection: diagnostic reads only two helper names; repair captures only seven declared outputs and preserves unknown entries. Validate/cleanup inspection returns exit1 with `originalFixtureFailureInspectionRequired`; it never guesses timestamps/UIDs, reuploads, tags or retries cleanup. Incomplete inspection remains a blocker to continuation.

Remaining acceptance: independent CODE/ARCH on these exact bytes; then actual native tag concurrency semantics, Ready/BSL, S3, live BackupRestore, persistence and precise cleanup. Dispatcher owns integration/status and all remote work. No remote phase, release, product build/Go race, source approval or P0 completion is claimed.

Gate Outcomes: native/spec/source/freshness/upload guards caught the four reviewed defects and drift/uncertainty; no gate waiver.
