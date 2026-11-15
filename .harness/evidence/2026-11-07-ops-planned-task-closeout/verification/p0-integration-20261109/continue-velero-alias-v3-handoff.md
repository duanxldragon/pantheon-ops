---
title: Sealed Velero alias v3 freshness handoff
doc_type: Evidence
layer: business
status: Active
updated_at: 2026-11-10
---

L2 parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`. Business acceptance tooling, logical separation. Ownership: five NEW v3 helpers and NEW v3 parent evidence. O2/current work and existing assets preserved.

Minimal repair: `fresh_predecessor(inputs)` immediately precedes EACH storage PUT/DELETE. Expiry refuses mutation and compensation. The durable failed repair receipt retains exact owned probe path,128-byte content hash, successful PUT and uncertain outcomes. `inspect-failed` reads that exact receipted object through the original captured S3 loader; GET/content or404 only, no replay. Original22/native/spec/fixture/source-AST/namespace/mirror/alias/content/preflight, finite uploads, BackupRestore/S3/persistence/cleanup assertions and original request authority remain intact. No timeout/freshness expansion.

| Helper                               | SHA256                                                           |
| ------------------------------------ | ---------------------------------------------------------------- |
| continue-velero-alias-v3-contract.py | bfac98f9c72ae3cbe3ff4bc3a9d97fe3d3c377f471e1ab851586c945772a4ca6 |
| continue-velero-alias-v3-remote.py   | 5e443950e04855771f6a36ad67041439a34902be9eb66dc0a960891e4dcb951b |
| continue-velero-alias-v3-root.py     | d3604678f345f258157a572add4fa8d8d367f30f27d66082e0e06589a1285897 |
| continue-velero-alias-v3-generate.py | a1c3738d2ebff357e18f20866f5ca9d84131d1f178332e321ef81ac2588ad88f |
| continue-velero-alias-v3-test.py     | da68651673b5a7281a152ecd0dce5d6138f1ccd6a9cd5efa70eb513585a8f176 |

Request: `continue-velero-alias-v3-request.json`, SHA256 `96fef023020ad1f8e880e01f04612acfa0480bd3009c009063fa8756703e3b07`. Generated/sealed ONCE after stable code/tests; do not rerun generators.

Protection: all inherited223 plus v2 five sources, request/verification/LSP/sealing/handoff, independent CODE/ARCH and preparation/commands =237 unchanged. Original v2 request remains `4bc6858b5f1ad5ff531339b8152fcd24b3a9ef53ed367f7f5720fbc158d22292`. Request/verification/LSP/sealing source maps agree before/after.

Verification: actual two-assertion RED on preserved v2; GREEN on v3. Original230 passes; v3 current53=45 preserved+8 new; full unique regression238, zero skips. All four original v1 counterexamples retained/reproduced. Five in-memory compilations pass. Actual native Pyright1.1.414 LSP, five files,0errors/0warnings, no suppression; basic/openFilesOnly cannot prove runtime.

- Verification SHA256: `d4e5924c72139c2b831fa062d09d913c4f8cd5f237d98849e90537b0a562e4a2`.
- LSP `continue-velero-alias-v3-lsp-final-20261110T071355818470.json`, SHA256 `e41921fd3f84d8266d21af692b0eb154b0767e75e8357c396b8cfc0966a4b98c`.
- Sealing SHA256: `e44b989f8b9f62195f5436ab3cd27d859226454d153d75b925d5d5d217bfc9ff`.
- Commands SHA256: `7421dd129b508414f3f6838017360eb787411fbbf87412592e02b95dae4bf3cf`; prior failures retained.

Dispatcher ONLY; root execution is blocked on both independent final-v3 approvals. Each `continue-velero-alias-v3-independent-{code,arch}-report.md` must contain exactly one JSON fence with `lane` CODE/ARCH, `decision` APPROVE, `independent` true, this `requestSHA256`, and exact full request `scriptHashes`. Root checks both before connection, including failure inspection; Generator supplies no approval.

```powershell
$aEV = '.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109'
$aTool = '.local/ops-acceptance/continue-velero-alias-v3-root.py'
$aSHA = '96fef023020ad1f8e880e01f04612acfa0480bd3009c009063fa8756703e3b07'
if ((Get-FileHash $aTool).Hash.ToLower() -ne 'd3604678f345f258157a572add4fa8d8d367f30f27d66082e0e06589a1285897') { throw 'root drift' }
if ((Get-FileHash "$aEV/continue-velero-alias-v3-request.json").Hash.ToLower() -ne $aSHA) { throw 'request drift' }
python -B $aTool check-inputs --request-sha256 $aSHA
if ($LASTEXITCODE) { throw 'inputs refused' }
function Inspect-Receipt($x) {
  $p = "$aEV/$($x.artifact)"
  if ((Get-FileHash $p).Hash.ToLower() -ne $x.sha256) { throw 'receipt drift' }
  Get-Content $p
  $receipt = Get-Content $p -Raw | ConvertFrom-Json
  if ($x.exitCode -or $receipt.exitCode -or !$receipt.remoteExecuted -or !$receipt.sourceHashesStable -or $receipt.rootRequestSHA256 -ne $aSHA) { throw 'inspect only; no replay' }
}
```

After approvals, run separately; inspect each complete successful receipt before its explicit predecessor name/hash is used. Predecessors expire after30minutes.

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

Bind `$failed` to the exact failed returned result; inspect its artifact/hash, then:

```powershell
python -B $aTool inspect-failed --request-sha256 $aSHA --receipt $failed.artifact --sha256 $failed.sha256
```

Inspection grants no continuation. Diagnostic reads two helpers; repair reads finite outputs and exact probe; validate/cleanup preserve `originalFixtureFailureInspectionRequired`. Incomplete inspection blocks continuation.

Stop reached: sealed local implementation/verification. Independent final-v3 CODE/ARCH and actual Ready/BSL, native tag concurrency, S3, BackupRestore, persistence and cleanup remain pending with Dispatcher. No remote execution, product changes, self-approval, waiver, release or P0 closure.

Gate Outcomes: per-effect expiry guards caught both v2 counterexamples; review gate blocks unapproved execution.
