---
title: Root-only pinned Velero EndpointSlice v6 recovery handoff
doc_type: Evidence
layer: business
status: Active
updated_at: 2026-11-10
---

V6 is a locally verified Generator candidate. Root must independently inspect
the sources/request and execute the real diagnostic, completion, validation and
fixture cleanup. No v6 remote phase has run in this Generator lane. The original
v4 install and v5 diagnostic remain failed. Real controller/Backup/Restore, P0,
D1/Base IAM HTTP audit, production G2/G3 and release acceptance remain open.

Parent task: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`.
L2, `pantheon-ops`, business/k8s acceptance tooling only. This is a logical
separation of recovery tooling from the unchanged product; no physical product
split or platform/system-domain change is needed. Reuse v5 helpers, the sealed
v4 install tail and original validation/fixture assertions with Python stdlib.
Only new `.local/ops-acceptance/resume-velero-s3-v6-*.py` and distinct current EV
files were written. No delegation/model override, credential reads, remote
operations, product/API/SQL/lock edits, commit or release were performed.

## Starting facts and preserved failures

Read [v5 handoff](resume-velero-s3-v5-handoff.md) and the actual v5
root/remote/native/fixture/test/generate sources, then the sources they derive
from. V5's corrected official dry-run succeeded with 18 objects, `v1-List`,
201205 bytes, raw SHA256
`65e25156d194c147a43757d347cd070949ffd3fd99d7261cf3e226bd70de3c24` and
original policy SHA256
`aa71df7b02a1b1af54b3a051e688100452d0556f708d5cbf6161fb46377b5d33`.
Its subsequent namespace guard refused the mirrored EndpointSlice. That is a
failed diagnostic, not a successful install or a waived ownership gate.

| Sealed original input                                                                          | SHA256                                                             |
| ---------------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| [v5 failed root diagnostic](resume-velero-s3-v5-root-diagnostic-20261110T024105028756.json)    | `4c9ae739221233ae3d3c0f1a10706b97e9ed2557fdc690c78d5086bbfc5aab53` |
| [v5 root request](resume-velero-s3-v5-root-request.json)                                       | `723824066805739acad3f52011c299a947f01c8035b872193c9113b4663949d6` |
| [actual read-only mirror inspection](velero-v6-mirror-inspect-20261110T024616246362.json)      | `b792321b9c3ce9240cd2e6f0664e93b2396ea0defe003f0f45f9c923a5f2224a` |
| [original v4 failed root install](resume-velero-s3-v4-root-install-20261110T021156151544.json) | `9c2048e5ff4d4f2ce139a8a19fe281afa26b539ba93a7c988cdd445a8a61683b` |

Original five successful CREATE UIDs are retained verbatim:

| Kind      | Namespace/name                                | UID                                    |
| --------- | --------------------------------------------- | -------------------------------------- |
| Namespace | velero                                        | `36e0c54b-185e-47c8-9ef4-a726dc808323` |
| Namespace | pantheon-p0-seaweed-velero-s3-v1              | `ea045ce0-48bf-47c3-9f2c-f3b933e459d6` |
| Service   | pantheon-p0-seaweed-velero-s3-v1/seaweedfs-s3 | `97c4a5d0-203d-473f-8068-cdc1a29e1490` |
| Endpoints | pantheon-p0-seaweed-velero-s3-v1/seaweedfs-s3 | `b6ab7b83-3a24-4b10-87d1-fae129a51dca` |
| Secret    | velero/cloud-credentials                      | `29994a5b-c619-4277-b461-0e59c1632a8e` |

Owner is fixed to `velero-s3-seaweed-20261109-v1`. Native Seaweed/private
disk/auth, imported public images and all receipts remain retained. Completion
must not replay native installation, image import or any of these five creates.
The original pre-failure authenticated 128-byte object PUT/GET/DELETE remains
historical evidence; v6's diagnostic runs authenticated bucket GET and actual
anonymous/bad-signature denials, not a new claim to have rerun that object probe.

## Narrow compatibility contract

The sole admitted mirror is `discovery.k8s.io/v1 EndpointSlice`
`pantheon-p0-seaweed-velero-s3-v1/seaweedfs-s3-rp65g`, UID
`e34fea74-5bd1-4c47-adf6-f637e7a0dee6`. Its four labels must match exactly:
the original owner, component `seaweedfs-s3`, service-name `seaweedfs-s3`, and
managed-by `endpointslicemirroring-controller.k8s.io`. Its sole ownerReference
must be `v1 Endpoints seaweedfs-s3`, exact original Endpoints UID, with
`controller=true` and `blockOwnerDeletion=true` as booleans. No deletionTimestamp,
annotations or finalizers are admitted.

The live Service must retain its pinned UID and exact inspected selectorless
ClusterIP spec (`10.43.75.62`, IPv4 SingleStack, Cluster internal traffic,
session affinity None, one TCP `s3` port/targetPort 18333). The original Endpoints
must retain its pinned UID, original exact labels and one address
`192.168.94.127` with TCP `s3/18333`. The slice must be IPv4, with precisely one
ready endpoint at that address and that exact single port. Extra conditions,
addresses/ports, targetRef/spec keys, unknown metadata, extra ownerReferences,
unknown labels, another mirror, replaced UID and any unknown namespace object
all refuse. No generic owner-label adoption is used.

Inspection returned no managedFields. V6 explicitly records
`managedFieldsEvidenceAvailable=false`; it never uses managedFields as ownership
proof. Ordinary API resourceVersion, creationTimestamp and generation are
validated for type/format but do not freeze a live object's version. The exact
generated prefix `seaweedfs-s3-` and an empty Service loadBalancer status are
accepted API envelope fields, not additional inspection evidence. A natural
replacement of the slice refuses with bounded identity/count/owner-match
metadata. Root must inspect that failure and prepare a separately sealed narrow
request; never broaden ownership or silently refresh this request.

All listable namespaced API types must be discovered and successfully LISTed,
including core types and EndpointSlices. Missing discovery, failed LIST,
duplicates, malformed/missing identities and more than 64 items per type refuse.
Original owned namespace objects must match their exact UIDs in the LIST as well
as the direct GET. Kubernetes `default` ServiceAccounts without credentials and
CA-only `kube-root-ca.crt` ConfigMaps remain separately recorded system defaults.
The mirror is separately recorded controller evidence, never appended to APPLIED.

Cleanup retains the provider, owned Endpoints/Service/namespaces and the mirror.
The mirror's only possible future destructive cleanup route is controller
garbage collection through the exact owned Endpoints or namespace, after fresh
foreign-object checks. `guard_provider_cleanup()` is a read-only guard and
refuses direct slice or foreign cleanup entries; v6 exposes no provider-delete
or native-delete phase. The final fixture cleanup checks the retained store
relationship and all types in both fixture namespaces before its first DELETE.
Each namespace DELETE repeats successful full discovery/LIST checks and retains
the original UID-precondition DELETE and absence verification. The restored
target ConfigMap must have the original exact content hash and test labels.
Only the finite additional `velero.io/backup-name=velero-final-backup` and
`velero.io/restore-name=velero-final-restore` labels are permitted for the
original restore attempt; these are bounded compatibility allowances, not
claimed observations of a real v6 restore.
This adds checks without dropping the original cleanup assertions.

## Request, source seals and retained remote files

Final immutable [v6 root request](resume-velero-s3-v6-root-request.json), SHA256:
`d89a0624f6fa9bae53371dd37af9d9550ebb8c227e8334f04ae108d1e8389e08`.
It binds 156 protected original files, all v1–v5 family scripts/requests/failures,
the actual inspection and its source, six new v6 helpers, the original 17
product/driver sources, actual driver executable, public native binary/archive
hashes and original resource policies. Earlier local v6 request iterations were
preserved as `resume-velero-s3-v6-root-request-candidate-<SHA256>.json`; none was
executed remotely. Root must use only the final request hash above. Rerunning
local preparation is unnecessary after sealing; any input drift is a refusal.

| New helper                      | SHA256                                                             |
| ------------------------------- | ------------------------------------------------------------------ |
| resume-velero-s3-v6-root.py     | `68e1ea283aaf67e82363e120ea749d27d002d2004537a7c32f609619ed4cfcea` |
| resume-velero-s3-v6-native.py   | `14f9ca3cc471bbf754da27fcdb71b2e22bd01153eefb02088dc43bb543527081` |
| resume-velero-s3-v6-remote.py   | `5f54ba6f6229793d072588733eb522810607e25fdcd3a37f4665d895c68d8bc7` |
| resume-velero-s3-v6-fixture.py  | `e53facea2f9b9822855de75f19aeab8210f9b4debe6da5f76b2fb84803d86fa3` |
| resume-velero-s3-v6-test.py     | `a0230535f33d45e2646dfee8aeeaab8f8dd73f507b41d35210ac9645ce3e40a3` |
| resume-velero-s3-v6-generate.py | `f13cf4c51b9aed499a6f44f564b04d98979ce019cf094695303c025a7408e1a0` |

The v5 failed diagnostic's actual `remoteFiles` is the new baseline, not the
earlier v4 manifest. It explicitly includes the failed remote timestamp
`resume-velero-s3-install-20261110T021156210245.json`, 1337 bytes, SHA256
`3904bb56d051519743880d1cc252847031aff4be2b7372840fc26b7ad58ebe63`, and the
original five-UID progress, 986 bytes, SHA256
`7640f7541fbdb28bdb0b28e67e0957968a989c4c31559a0fa1318f27c0c2d43b`.
These original exact bytes remain unchanged until the explicit progress migration.

V5 may have left six helper files. Three uploads are directly observed in the
actual failure receipt and are required; the other three may already exist only
at the exact original v5 request bytes/hash/private mode below. They are never
uploaded by v6, overwritten or adopted at a different seal.

| Retained v5 file                | Presence rule                     | Bytes | SHA256                                                             |
| ------------------------------- | --------------------------------- | ----- | ------------------------------------------------------------------ |
| resume-velero-s3-v5-remote.py   | Required/observed                 | 12502 | `b558d5809f671100c240f8648812819d92052ee0fa0571fd77723d770cc62029` |
| resume-velero-s3-v5-native.py   | Required/observed                 | 12681 | `540ca34b11dcabfebb3c1554142c83fb2b22ae2728d1ca3bb7672c1bfa9192d2` |
| resume-velero-s3-v5-fixture.py  | Required/observed                 | 13093 | `e32b5ceb96e68edf0a3af76b3dbde6362b0679387460e1e748f95c2e7ee93910` |
| resume-velero-s3-v5-root.py     | Optional/exact original seal only | 16081 | `a20989cacaf2e89d79e5918b5d0e2b1a9d28d14e24db78ce91d0c452b7a6c28b` |
| resume-velero-s3-v5-generate.py | Optional/exact original seal only | 4350  | `600f6159a77ccc75558ffc89d07347565f18b50f602cc01ec101032696f46ce1` |
| resume-velero-s3-v5-test.py     | Optional/exact original seal only | 21630 | `1278d8dfb6db9363a309ac574a07db039b7fc5d1642c047d61239f347e7f6b6b` |

All six must be root-owned regular single-link files, mode 0600 if present.
Every old baseline file and native data/img/auth metadata is rechecked before
the three new remote/native/fixture helpers are exclusively uploaded (`wx`,
0600). Only already present exact copies of those same three v6 helpers can be
reused. Unknown filenames, missing originals, extra bytes, wrong hashes/mode or
replacement paths refuse before uploads. No old helper, binary, archive, data
or auth file is overwritten/retransferred. Native credentials are excluded from
public content manifests; only the root lane forwards credentials and finite
client material in memory during the original validation.

## Root execution commands and evidence review

From the Ops root, in the existing authorized root remote/credential lane.
These commands are supplied for root; this Generator did not execute them.

```powershell
$veleroV6RequestSHA = 'd89a0624f6fa9bae53371dd37af9d9550ebb8c227e8334f04ae108d1e8389e08'
$veleroV6Evidence = '.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109'
if ((Get-FileHash -Algorithm SHA256 (Join-Path $veleroV6Evidence 'resume-velero-s3-v6-root-request.json')).Hash.ToLowerInvariant() -ne $veleroV6RequestSHA) { throw 'v6 request hash drift' }
python -B .local/ops-acceptance/resume-velero-s3-v6-root.py check-inputs
if ($LASTEXITCODE -ne 0) { throw 'v6 sealed local input check failed' }

$veleroV6Diagnostic = python -B .local/ops-acceptance/resume-velero-s3-v6-root.py diagnostic --request-sha256 $veleroV6RequestSHA | ConvertFrom-Json
$veleroV6DiagnosticCode = $LASTEXITCODE
Get-Content (Join-Path $veleroV6Evidence $veleroV6Diagnostic.artifact)
if ($veleroV6DiagnosticCode -ne 0 -or $veleroV6Diagnostic.exitCode -ne 0) { throw 'v6 diagnostic refused; retain all state and inspect safe metadata' }
```

Diagnostic records bounded shape/count/public argv/hashes and safe identities,
never raw manifests, Secrets or kubeconfig contents. It retains successful
official dry-run metadata if a later native/mirror check fails. CLI stdout is
one safe JSON artifact envelope; its artifact is a distinct timestamped root
receipt with SHA256. ExitCode=1 is a refusal, never permission to continue.
Root must review the entire successful diagnostic artifact, including:

- exact original five UIDs and separately pinned live controller relationship;
- all successful namespace discovery/LISTs and absent remaining official objects;
- native binary/archive/image digests, private root kubeconfig metadata and source seals;
- exact loop image/mount/argv/unit/cgroup and eight private listeners;
- actual SigV4 authenticated read, anonymous denial and bad-signature denial;
- exact memory/disk caps and live headroom, with no PV or uncontrolled installation.

Root then explicitly supplies the successful predecessor artifact and SHA256:

```powershell
$veleroV6Complete = python -B .local/ops-acceptance/resume-velero-s3-v6-root.py complete --request-sha256 $veleroV6RequestSHA --receipt $veleroV6Diagnostic.artifact --sha256 $veleroV6Diagnostic.sha256 | ConvertFrom-Json
$veleroV6CompleteCode = $LASTEXITCODE
Get-Content (Join-Path $veleroV6Evidence $veleroV6Complete.artifact)
if ($veleroV6CompleteCode -ne 0 -or $veleroV6Complete.exitCode -ne 0) { throw 'v6 completion failed; retain receipts/resources and do not replay' }

$veleroV6Validate = python -B .local/ops-acceptance/resume-velero-s3-v6-root.py validate --request-sha256 $veleroV6RequestSHA --receipt $veleroV6Complete.artifact --sha256 $veleroV6Complete.sha256 | ConvertFrom-Json
$veleroV6ValidateCode = $LASTEXITCODE
Get-Content (Join-Path $veleroV6Evidence $veleroV6Validate.artifact)
if ($veleroV6ValidateCode -ne 0 -or $veleroV6Validate.exitCode -ne 0) { throw 'v6 real validation failed; retain partial fixture evidence' }

$veleroV6Cleanup = python -B .local/ops-acceptance/resume-velero-s3-v6-root.py cleanup --request-sha256 $veleroV6RequestSHA --receipt $veleroV6Validate.artifact --sha256 $veleroV6Validate.sha256 | ConvertFrom-Json
$veleroV6CleanupCode = $LASTEXITCODE
Get-Content (Join-Path $veleroV6Evidence $veleroV6Cleanup.artifact)
if ($veleroV6CleanupCode -ne 0 -or $veleroV6Cleanup.exitCode -ne 0) { throw 'v6 fixture cleanup or retained S3 persistence unverified' }
```

Only `check-inputs`, `diagnostic`, `complete`, `validate`, `cleanup` are exposed.
There is no install/import/first-five replay or clean-native entry. Each phase
requires the final request hash; every phase after diagnostic requires an exact
successful predecessor of the correct phase, bound to v6 script hashes and the
original failure/inspection chain. A v5 failure cannot serve as a successful
predecessor. No latest-receipt selection or automatic permissions widening occurs.

Complete repeats the official dry-run/native diagnostic, approved stable
snapshot comparisons and a fresh exact five/mirror check before migration and
again before the first new CREATE. It checks all remaining original official
object names absent. Exclusive `resume-velero-s3-v6-original-partial-owned.json`
archives the original progress bytes; exclusive `resume-velero-s3-v6-migration.json`
binds the root request, v4 failure, v5 failure, inspection, failed remote filename,
original five UIDs, approved manifest and canonical paths. A previously existing
provider receipt refuses. APPLIED starts with the original five and grows only
from actual CREATE results. The original v4 tail performs only remaining
CRD/SA/RBAC/BSL/controller creation, rollout, actual BSL Available/no restarts
and the compatible provider success receipt. A distinct v6 complete timestamp
is retained. A failure after migration requires root inspection; do not rerun
the full installer or replace the failed completion receipt.

Provider caps remain 160Mi Seaweed plus 320Mi Velero, combined 480Mi; disk is
256Mi ext4/no PV. Public pinned images and Never pull policy, original resource
limits/credential mount/SigV4/denial assertions remain. Native installation,
image import, credential generation and the original five creates never repeat.

Validate retains the exact v4 `validate` and `remote_phase` function bodies,
with only explicit v6 fixture/global adapters. It executes the original sealed
driver and all finite-SA authorization/product assertions, real Backup/Restore,
exact restored ConfigMap content and namespace-only non-PV intent. It uses the
original isolated MySQL setup and cleanup only in root's authorized lane. Its
finally cleanup must prove finite SA/bindings/fixtures/private client material
absence and actual S3 persistence. Final explicit cleanup rechecks the original
fixture assertions and retained native provider/data without deleting the mirror.

## Local verification and review limits

[Commands and captured results](resume-velero-s3-v6-commands.json),
[source comparison](resume-velero-s3-v6-source-final.diff), and
[self-review](resume-velero-s3-v6-review.md) are separate new artifacts.
46 v6 tests, 32 unchanged v5 tests and 25 unchanged v4 tests pass. Six-helper
syntax compile, ignored-path check and 156 original-file hashes pass. Tests
include valid relationship; wrong UID/owner/controller/service/address/port;
extra mirror/unknown object/malformed metadata; LIST/discovery errors;
source drift/extra bytes; exclusive old-evidence protection; exact optional
retained v5 files; failed/stale/wrong-hash predecessors and last-preflight drift.
All Kubernetes/native/S3/SQL actions in these regressions are mocks. Their
success proves guard behavior only, never real controller or backup success.
Docs frontmatter and task-packet-template checks also pass. Ruff/pyflakes are
not installed; syntax compilation and the targeted executable regressions were
used without installing dependencies. The initial source diff is retained;
`resume-velero-s3-v6-source-final.diff` is the final helper comparison.

CodeGraph tools were not exposed; the task provided exact helper/source paths,
which were read directly. No UI changed, so visual evidence is not applicable.
No product/Go/frontend build changed; this local preparation does not claim a
fresh product build or full Go race. Root's original executable/source seals
remain required. Independent review was not delegated because this Generator
lane expressly prohibits delegation; the next review/execution belongs to root.

Gate Outcomes: mirror/namespace/source/predecessor guards catch unsafe variants;
initial malformed mock UID/path/resourceVersion fixtures were corrected without
weakening gates. Original failed evidence remains unchanged. Stop condition:
local v6 preparation and fresh safety verification complete; root-only actual
diagnostic/completion/Backup/Restore/cleanup and final acceptance remain pending.
