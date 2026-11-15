---
title: Root-only Velero unregistered BSL v7 recovery handoff
doc_type: Evidence
layer: business
status: Active
updated_at: 2026-11-10
---

L2 parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`.
Generator local preparation is complete; root owns independent inspection and
every actual remote phase. Business/k8s acceptance tooling only; logical
separation, no product/platform/system-domain change. No remote, credentials,
configuration, model/delegation, product source, old helper or old receipt edits.

The exact v6 failed diagnostic proves a preflight compatibility refusal after
official dry-run/native/mirror/auth guards. The exact read-only inspection finds
17 nonnamespace objects absent (BSL unregistered), 13 CRDs absent, and no advertised
`velero.io` group. The existing official object is the original `velero` namespace.

| Immutable predecessor                         | SHA256                                                             |
| --------------------------------------------- | ------------------------------------------------------------------ |
| v6 root-request.json                          | `d89a0624f6fa9bae53371dd37af9d9550ebb8c227e8334f04ae108d1e8389e08` |
| v6 root-diagnostic-20261110T033455865739.json | `4a486df4fa7f3b35f5a5c7596dd0045e76bfdd18fe1facb2a806af3d769b305e` |
| v6 absence-inspect-20261110T034909050953.json | `b8154da9dea668ca334fe3f6dae10134361ca79c722a035c83a6cea594507a7f` |

V7 permits only official `velero.io/v1 BackupStorageLocation velero/default`,
associated with manifest CRD `backupstoragelocations.velero.io` (exact names,
Namespaced scope, served/storage v1). Live CRD must return a complete normal
NotFound line; bounded authoritative raw `/apis` must parse correctly and omit
`velero.io`; live BSL GET must return the exact unregistered-kind error.
Registered groups, missing proofs, existing objects, malformed/duplicate discovery,
wrong identities/schema, 403/auth/network/timeout and mixed errors refuse before
CREATE. Other objects retain strict NotFound checks. This deliberately supports
only the observed group-absent case; no cached discovery or broad API fallback.

Complete repeats fresh guards before migration writes and immediately before
remaining CREATEs. Original `P.apply` awaits CRD Established before CR creation.
Original five UIDs, mirror relationship/namespace LIST, native/auth/unit/image,
binary/driver/policy/readiness, Backup/Restore and fixture cleanup rules remain.
No installer/native/image-import/first-five replay or provider-delete entry exists.

Final [request](resume-velero-s3-v7-root-request-final.json) binds 180 protected
files, seven scripts, 29 exact v6 remote baseline files and 17 original
product/driver sources. Existing v6 helpers are retained, never overwritten.
Only new v7 remote/seals/fixture helpers can upload exclusively or reuse exact
bytes. Unknown optional files refuse; original finite sealed v5 allowances remain.
Captured predecessor imports/JSON and helper commands hash/parse/execute the same
captured bytes. Earlier local v7 request/preparation remain immutable candidates;
use only the final request below. Preparation is exclusive; do not rerun it.

```powershell
$v7SHA = '05a39bef4df96a8c08ad78b17373d4968b6116be8ee8a2682e7708147517ad6e'
$v7EV = '.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109'
if ((Get-FileHash "$v7EV/resume-velero-s3-v7-root-request-final.json").Hash.ToLowerInvariant() -ne $v7SHA) { throw 'request drift' }
python -B .local/ops-acceptance/resume-velero-s3-v7-root.py check-inputs
if ($LASTEXITCODE -ne 0) { throw 'sealed inputs refused' }
$d = python -B .local/ops-acceptance/resume-velero-s3-v7-root.py diagnostic --request-sha256 $v7SHA | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $d.exitCode -ne 0) { throw 'diagnostic refused' }
Get-Content "$v7EV/$($d.artifact)"
$c = python -B .local/ops-acceptance/resume-velero-s3-v7-root.py complete --request-sha256 $v7SHA --receipt $d.artifact --sha256 $d.sha256 | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $c.exitCode -ne 0) { throw 'complete refused' }
Get-Content "$v7EV/$($c.artifact)"
$v = python -B .local/ops-acceptance/resume-velero-s3-v7-root.py validate --request-sha256 $v7SHA --receipt $c.artifact --sha256 $c.sha256 | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $v.exitCode -ne 0) { throw 'validate refused' }
Get-Content "$v7EV/$($v.artifact)"
$x = python -B .local/ops-acceptance/resume-velero-s3-v7-root.py cleanup --request-sha256 $v7SHA --receipt $v.artifact --sha256 $v.sha256 | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $x.exitCode -ne 0) { throw 'cleanup refused' }
Get-Content "$v7EV/$($x.artifact)"
```

Execute each phase separately; independently inspect its complete successful
artifact before the next command. Failed/stale/wrong-phase/hash/source predecessors
refuse. Preserve failures and partial state; never guess or replay installation.

[Fresh command evidence](resume-velero-s3-v7-commands.json): v7 45 tests, unchanged
v6 46, v5 32, v4 25 (148 total, zero skips); seven syntax compilations;
root `check-inputs`; all 180 protected seals pass. Exact commands/output/source
hashes/counts are recorded: seven scripts, 1,190 lines, 76,683 bytes.
[Final preparation](resume-velero-s3-v7-preparation-final.json),
[source comparison](resume-velero-s3-v7-source-final.diff),
[scope review](resume-velero-s3-v7-review.md). Initial Windows ctime mismatch was
corrected while retaining inode/mtime/mode/link/hash guards.
[Docs gates](resume-velero-s3-v7-governance-commands.json) pass. CodeGraph tools
unavailable; exact supplied paths were read directly.
No UI/product build changed; no fresh Go race or product-build claim.

Gate Outcomes: hostile proofs/source/receipt/listing guards caught unsafe cases.
No CODE/ARCH approval or P0 closure. Real BSL Available, Backup/Restore, S3
persistence and cleanup evidence remain root downstream; original D1/G2/G3 and
other acceptance gaps remain open.
