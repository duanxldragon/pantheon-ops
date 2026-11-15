---
title: Root-only Velero Seaweed S3 v4 acceptance handoff
doc_type: Evidence
layer: business
status: Active
updated_at: 2026-11-10
---

Generator candidate is locally verified. Root owns all remote/credential execution
and final evaluation. Existing test-server authorization applies; no renewed user
scope/permission question is needed. This document is not P0 approval.

Parent task: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`.
L2, business/k8s acceptance tooling. No platform/system/product source changes,
SQL/ledger/lock changes, original script changes, credentials/configuration changes,
remote operations, commits or releases were performed by this Generator.

### Exact candidate and source evidence

Request: `resume-velero-s3-v4-root-request.json`
SHA256: `ea4d2d5e406f95cb3e838067f66ed32c40e0e7b02152e117d25e59ad1da4f812`.

| Local ignored script              | SHA256                                                             |
| --------------------------------- | ------------------------------------------------------------------ |
| `resume-velero-s3-v4-remote.py`   | `63b2e9a90fb7b54fc522e6190ff6ad0cb353f199752dd7fea76b7d93cb47183a` |
| `resume-velero-s3-v4-fixture.py`  | `e32b5ceb96e68edf0a3af76b3dbde6362b0679387460e1e748f95c2e7ee93910` |
| `resume-velero-s3-v4-native.py`   | `f116a423d4f5c207db032829fc22b4bb3b8179eec41624bb0d9d14f0c4be402a` |
| `resume-velero-s3-v4-root.py`     | `20209264b26af232a1b8c68e93943b5496e37ff93098f100928b81df5b186898` |
| `resume-velero-s3-v4-generate.py` | `45b9a273106739f87639130ae9bb93039fbf51209a94af4b60f8fc805badaf19` |
| `resume-velero-s3-v4-test.py`     | `3fb7373b6903ffd3f805e2c48378a58f1ce63573377dc1c5b9b8dab49cb27da0` |

The request binds all six scripts, the original eight scripts, 17 product/driver
sources, original root request, official cached archives/OCI metadata, actual
driver binary and retained prior diagnostic snapshots. `check-inputs` verifies
those locally without connecting or loading credentials. Each remote phase also
requires the exact request SHA256 and successful predecessor receipt SHA256.

Authoritative observations retained unchanged:

- `resume-velero-native-inspection-20261110T010514525598.json`: eth0 plus K3s overlays.
- `resume-velero-s3-v3-root-diagnostic-20261110T010950095179.json`, SHA256
  `3437101eb0cc50fdb48ebfd787cad47ada6d05be73f56007bb12c39192067a95`:
  actual binaries/server help and native dev eth0 diagnosis succeed.
- `resume-velero-s3-v3-install-result-20261110T011217586443.json`: failure at
  `private-native-seaweed`, ValueError, no owned Kubernetes objects.
- `resume-seaweed-listener-diagnosis-20261110T011723375932.json`: all ten listeners
  private, expected seven plus 8181/9101/28333; exact owned unit stopped.

Review `resume-velero-s3-v4-source.diff` and `resume-velero-s3-v4-derivation.json`.
Only remote `diagnostic()`/`install()` change; all 13 other functions have identical
AST, including strict dev eth0 selection, binding/auth/cgroup/OCI/ownership logic.
Fixture changes only its provider import; all six functions are AST-identical.
Root `validate()` changes only its fixture filename; all actual restore/non-PV,
RBAC, isolated MySQL, S3 persistence and finite UID cleanup assertions remain.

Actual help explicitly supports `-s3.port.iceberg int` and `-s3.port.lance int`,
both documented `0 to disable`, and `-s3.port.grpc int`. V4 adds exactly
`-s3.port.iceberg=0`, `-s3.port.lance=0`, `-s3.port.grpc=28333`, requires their
presence in fresh actual help and includes 28333 in the approved/preflight ports.
Approved listeners: 19333, 29333, 18080, 28080, 18888, 28888, 18333, 28333, all
at the single native eth0 RFC1918 IPv4. Installation/validation requires all eight.
Memory remains 160Mi + 320Mi = 480Mi, swap disabled, native image exactly 256Mi.

### Exact root execution phases

Use the existing root network/credential lane, from the Ops repository directory.
Do not run original `provision(diagnostic)` or original v1 install. Do not replay
the full installer over existing data/image/auth. First review these new helpers
and the actual help; then execute the already-authorized phases below.

```powershell
$veleroV4RequestSHA = 'ea4d2d5e406f95cb3e838067f66ed32c40e0e7b02152e117d25e59ad1da4f812'
$veleroV4Evidence = '.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109'
python -B .local/ops-acceptance/resume-velero-s3-v4-root.py check-inputs
if ($LASTEXITCODE -ne 0) { throw 'v4 local inputs changed' }

$veleroV4Inspect = python -B .local/ops-acceptance/resume-velero-s3-v4-root.py inspect-native --request-sha256 $veleroV4RequestSHA | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $veleroV4Inspect.exitCode -ne 0) { throw 'v4 native inspection refused; retain resources' }
Get-Content (Join-Path $veleroV4Evidence $veleroV4Inspect.artifact)
```

`inspect-native` first validates the known remote directory/metadata/hash set,
including weed/velero binary hashes and v3 script. Exactly one extra failed-install
timestamp receipt is permitted only with the exact typed v3 failure content in
the request. Unknown remote entries are refused before uploads. Only the three
new v4 remote helpers are uploaded, with exclusive `wx`; archives and originals
are never retransferred/overwritten. Previously uploaded v4 files require the
same exact hashes. A failed partial upload can be re-inspected without overwrite.

Root reviews the complete inspection receipt before passing its hash to cleanup:
root-owned fixed directories/files, single links, original artifact hashes;
unit not-found/inactive/MainPID 0 and no live process of the exact weed executable;
successful Kubernetes LIST proves no Velero CRDs or either dedicated namespace;
exact regular 256Mi image; native `findmnt` exact ext4 mount, required options and
no child mount; `losetup` exact canonical backing file, inode, source, offset and
read/write state; canonical loop block device matches mounted filesystem device;
single dedicated OWNER identity and credential count (values never printed).
The bounded full data tree records paths, root ownership, inode, counts and hashes.
Any unrecognized/foreign data-tree content must be refused by root; do not approve
its hash until the actual tree is understood as this failed native install's data.
No guessed file allowlist is substituted for actual evidence. Any change after
the reviewed snapshot makes cleanup refuse.

```powershell
$veleroV4Clean = python -B .local/ops-acceptance/resume-velero-s3-v4-root.py clean-native --request-sha256 $veleroV4RequestSHA --receipt $veleroV4Inspect.artifact --sha256 $veleroV4Inspect.sha256 | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $veleroV4Clean.exitCode -ne 0) { throw 'v4 exact native cleanup refused' }

$veleroV4Diagnostic = python -B .local/ops-acceptance/resume-velero-s3-v4-root.py diagnostic --request-sha256 $veleroV4RequestSHA --receipt $veleroV4Clean.artifact --sha256 $veleroV4Clean.sha256 | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $veleroV4Diagnostic.exitCode -ne 0) { throw 'v4 actual diagnostic failed' }
Get-Content (Join-Path $veleroV4Evidence $veleroV4Diagnostic.artifact)
```

`clean-native` rechecks the entire approved snapshot before unmounting only its
exact test mount. It removes only data.img/auth and the empty fixed mountpoint
with `rmdir`; no recursive host deletion. It verifies image metadata, dedicated
auth hash, absence of loop attachment, stopped unit and Kubernetes absence.
Unmount may write ext4's clean superblock: both pre-unmount and removed-image
hashes are retained, along with auth count/hash only. All original scripts,
archives, original receipts and their bytes/hashes remain. Unexpected underlying
mountpoint files refuse removal and leave image/auth intact.

`diagnostic` performs actual v4 private-IP/preflight/binary/help checks. The original
helper deliberately writes canonical help/diagnostic; prior v3 root full receipt
and typed timestamp remain unchanged. New root output retains actual canonical
help/diagnostic values, hashes and exactly one new typed timestamp output, rather
than replacing the unknown directory listing with a fresh unreviewed manifest.
Root reviews this output before installing.

```powershell
$veleroV4Install = python -B .local/ops-acceptance/resume-velero-s3-v4-root.py install --request-sha256 $veleroV4RequestSHA --receipt $veleroV4Diagnostic.artifact --sha256 $veleroV4Diagnostic.sha256 | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $veleroV4Install.exitCode -ne 0) { throw 'v4 install failed; inspect exact partial state, never replay' }

$veleroV4Validate = python -B .local/ops-acceptance/resume-velero-s3-v4-root.py validate --request-sha256 $veleroV4RequestSHA --receipt $veleroV4Install.artifact --sha256 $veleroV4Install.sha256 | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $veleroV4Validate.exitCode -ne 0) { throw 'v4 real validation failed; retain receipts and inspect any partial fixtures' }

$veleroV4FixtureCleanup = python -B .local/ops-acceptance/resume-velero-s3-v4-root.py cleanup --request-sha256 $veleroV4RequestSHA --receipt $veleroV4Validate.artifact --sha256 $veleroV4Validate.sha256 | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $veleroV4FixtureCleanup.exitCode -ne 0) { throw 'v4 final exact fixture cleanup failed' }
Get-Content (Join-Path $veleroV4Evidence $veleroV4FixtureCleanup.artifact)
```

Installation uses the reviewed v4 provider with unchanged official pinned OCI,
S3 SigV4 and anonymous/bad-signature denial, cgroup/memory/disk limits and Kubernetes
ownership. Successful output must show actual BSL Available and all eight private
ports. Newly created UID inventory and canonical provider/timestamp receipts are
accepted only as exact products of that invocation; unknown entries refuse.

Validation calls the actual original live driver binary with v4 fixture/global
hash adapters, real finite SA credentials and real isolated MySQL setup/cleanup.
It requires actual backup completed, restore completed, restored ConfigMap content
matches source, namespace-only non-PV restore intent, persisted S3 tarball,
finite SA/bindings/client-material absence, exact UID cleanup and retained provider.
Final cleanup verifies persisted S3 again and keeps successful provider/data.
Local mock tests are safety/adapter tests only; they never replace this provider
or establish live acceptance.

### Failure handling, evidence and remaining evaluation

Every root phase writes a new exclusive local timestamp JSON and prints a separate
single-line CLI summary with artifact/SHA256. The generator's intermediate request
candidates were preserved under their hashes; use only the final hash above.
Predecessor failure, stale snapshot, missing UID, changed source, foreign files,
mounts or credentials refuse and retain state. Root must inspect actual failure
state before devising bounded recovery; the wrapper never authorizes blind replay
or deletes unreceipted Kubernetes objects. Original validate failure paths remain
unchanged: failed/ambiguous partial fixture preparation may require inspected
recovery, and cannot be treated as successful cleanup.

Fresh local evidence: `resume-velero-s3-v4-commands.json`,
`resume-velero-s3-v4-derivation.json`, `resume-velero-s3-v4-review.json`.
25 local tests plus parse/compile, input/source/binary hashes, docs frontmatter
and task-packet-template checks pass. No new Go sources or driver builds were
introduced; retained original race-build/vet evidence and binary hash are bound
by the original request. No commit/PR action occurred, so no full Go race closure
is claimed. CodeGraph tools were not exposed in this leaf session; specific
provided source files were inspected directly.

Gate Outcomes: AST/source gate caught unintended provider/fixture/validation drift;
native snapshot gate caught foreign metadata/mount/identity and changed state;
private-listener gate reproduced the known v3 extra-port refusal; none false-positive.

Stop condition reached: scoped generator candidate, exact root continuation and
fresh local verification are ready for independent root review/execution. Real
v4 installation/Backup/Restore remain pending. D1/Base IAM HTTP audit, production
G2/G3, overall P0 and release decisions stay open; no final P0 approval is issued.
