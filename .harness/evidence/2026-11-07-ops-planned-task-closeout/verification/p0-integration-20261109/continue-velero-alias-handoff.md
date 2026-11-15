---
title: Velero alias recovery generator handoff
doc_type: Evidence
layer: business
status: Active
updated_at: 2026-11-10
---

L2 parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`.
Business runtime acceptance tooling, logical separation. Landing side is Ops local acceptance helpers; no platform/system/product change is needed for the captured containerd normalization failure. CodeGraph was unavailable; supplied Velero source paths were read directly.

Generator implementation and local verification are complete. Independent CODE/ARCH review and every remote phase belong to Root. Runtime provider success, parent/P0 and public HTTP approval remain unestablished. No credential read, remote execution, delegation, model override, installation, image import, resource CREATE replay, restart, forced cleanup or release occurred in this generator turn.

Five new `continue-velero-alias-{contract,remote,root,generate,test}.py` helpers reuse captured original v8/v6/v4 source buffers. The native adapter replaces only the obsolete absence blocks with current 22 UID/spec/label/Established/RBAC and controller-chain checks. Original credential, mount, process argv, cgroup, image/archive, private listener and authentication assertions remain. Official manifests/config/layers and imported single linux/amd64 index descriptors are independently checked against actual content. Only two absent normalized aliases can be tagged in `k8s.io`; unknown, malformed, conflicting or already-present references refuse. A matching present alias requires predecessor inspection.

Repair journals durable exclusive intent/outcome bytes for each alias. Partial success or lost response never becomes replay permission; Root captures only the exact expected partial filenames for inspection. Existing Deployment retries naturally. Readiness, zero observed restarts, BSL Available, authenticated Put/Get/Delete/absence and anonymous/bad-signature denial precede receipt recovery. Canonical 22-object inventory bytes remain unchanged and are archived verbatim. Provider phase is truthfully `alias-recovery`, with `installReexecuted=false`, original/current inventories and original failure pins. Original v4 terminal assertions, v8 sealed v6 finite fixture, live driver Backup/Restore, DB/client/fixture cleanup and S3 persistence remain the validation path.

Plan completed: inspect sealed failure → add bounded alias contract/diagnostic/repair → reuse original validate/cleanup → hostile local verification → exclusive request seal. In scope: five helpers/tests and new parent evidence only. Out of scope: all historical files and the user’s prohibited actions. Minimum complexity: existing sealed helpers plus Python stdlib/containerd native tag; no new dependency.

| Source   | SHA256                                                             |
| -------- | ------------------------------------------------------------------ |
| contract | `a18eaa2cac4b5e2358090b3c9aebe3eaa4cdb1450d22a0c3a1d2f80f890cdef6` |
| remote   | `906f24a70436eefc53245d5251c831b3828e5227edc8ffe8dcac3fb2f78dbfb6` |
| root     | `fd02f3dc08438891241d3c775dcb06124badb791fd9622c0a0b0358eb24f1e0d` |
| generate | `d8b8582cc0e227e1445845d12b63b71161b7e1868684b4a2e3a801931634958a` |
| test     | `46ad82310bb2de04e83a3c77ff20f1b7378b1b6c57027d2b03f435a7da6febc2` |

[Exclusive request](continue-velero-alias-request.json): `f053fb6dda5d44af557eabdee4ad04b676da47591d7add0e210a442425a81d70`. It pins the 35 current remote files, two new remote helpers and seven exact repair outputs. Original v8 request `aec0783ef37a191284c16f9fe69f96a06dcf94b846de62de2efdd0dcffae232f`, approved diagnostic `278dd8f45dfb02ad064c55419e7168312b93eea38bb5c7e20a125d5a6e943904`, failure `f27d06fcc7160e534d3a2d8dde304367620bc7bf1f4e66148476d3b405de2601` and all inherited seals remain preserved. Do not rerun generation.

[Post-seal verification](continue-velero-alias-verification.json), SHA `a8ab24072d2fb1d7e052d607068591602831a82081c677fe069beed02d7df1a0`: 27 adversarial tests, zero skips; five native Python compilations; all executed source hashes equal request and LSP hashes. Original regressions: v8 55 + v6 46 + v5 32 + v4 25 = 158 additional passes. Total 185. [Commands](continue-velero-alias-commands.json) records outcomes and the RED reproduction of both missing normalized references. Tests include wrong namespace/index/manifest/config/layer, unknown/conflicting/present aliases, UID/spec/source/predecessor tamper, partial/lost replies, Ready/BSL/S3 failure, exact byte preservation and no forbidden mutation calls.

[Actual Pyright 1.1.414 LSP](python-lsp-continue-velero-alias-sealed-20261110T055622323439.json), SHA `0e232c55179b1f0870148a5aec240c49997054097f577d8dfa43a0ae6369e44b`: five files, zero errors/warnings, basic/openFilesOnly, existing acceptance-tool extraPath. No suppressed diagnostics. No product build/Go race claim because product and Go sources were untouched and no commit/PR is requested.

ROOT NEXT COMMANDS: first obtain independent CODE APPROVE/ARCH CLEAR for these exact bytes. Then run separately and inspect each complete successful receipt before passing its explicit filename/hash onward; a predecessor expires after 30 minutes.

```powershell
$aEV = '.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109'
$aSHA = 'f053fb6dda5d44af557eabdee4ad04b676da47591d7add0e210a442425a81d70'
$aTool = '.local/ops-acceptance/continue-velero-alias-root.py'
python -B $aTool check-inputs --request-sha256 $aSHA
if ($LASTEXITCODE -ne 0) { throw 'inputs refused' }
$d = python -B $aTool diagnostic --request-sha256 $aSHA | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $d.exitCode -ne 0) { throw 'diagnostic refused' }
Get-Content "$aEV/$($d.artifact)"
```

```powershell
$r = python -B $aTool repair --request-sha256 $aSHA --receipt $d.artifact --sha256 $d.sha256 | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $r.exitCode -ne 0) { throw 'repair refused; inspect partial outputs' }
Get-Content "$aEV/$($r.artifact)"
```

```powershell
$v = python -B $aTool validate --request-sha256 $aSHA --receipt $r.artifact --sha256 $r.sha256 | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $v.exitCode -ne 0) { throw 'validation refused' }
Get-Content "$aEV/$($v.artifact)"
```

```powershell
$c = python -B $aTool cleanup --request-sha256 $aSHA --receipt $v.artifact --sha256 $v.sha256 | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $c.exitCode -ne 0) { throw 'cleanup refused' }
Get-Content "$aEV/$($c.artifact)"
```

Gate Outcomes: alias/predecessor/content/spec guards caught normalization mismatch, uncertain replay and tampered ownership before mutations; no weakening or runtime approval.
