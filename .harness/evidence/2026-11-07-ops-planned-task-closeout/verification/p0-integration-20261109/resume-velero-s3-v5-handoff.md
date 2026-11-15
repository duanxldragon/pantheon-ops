---
title: Root-only bounded Velero Seaweed S3 v5 recovery handoff
doc_type: Evidence
layer: business
status: Active
updated_at: 2026-11-10
---

V5 is a locally verified Generator candidate for root review and execution.
The observed v4 install remains failed; provider installation and real
Backup/Restore are pending. This document does not approve P0.

Parent task: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`.
L2, Ops business/k8s acceptance only. This is a logical separation between
recovery tooling and the unchanged product, without a platform/system change.
Only new ignored `.local/ops-acceptance/resume-velero-s3-v5-*.py` helpers and
distinct evidence in this directory were prepared. No remote/credential reads,
product/API/SQL/assertion/lock edits, delegation, model override, commit or
release were performed. Existing root test-server authorization continues to
apply; root does not need to ask the maintainer again.

### Sealed starting state and minimal plan

Start from the original [v4 handoff](resume-velero-s3-v4-handoff.md) and actual
v4 remote/root/fixture/native sources. Root completed exact native cleanup
(including the retained new image-only snapshot), diagnostic and the attempted
install. Preserve all these observations and their failed receipts.

Canonical failure:
`resume-velero-s3-v4-root-install-20261110T021156151544.json`, SHA256
`9c2048e5ff4d4f2ce139a8a19fe281afa26b539ba93a7c988cdd445a8a61683b`.
It failed at `owned-kubernetes-provider` after five successful creates:

| Kind      | Namespace/name                                | Exact UID                            |
| --------- | --------------------------------------------- | ------------------------------------ |
| Namespace | velero                                        | 36e0c54b-185e-47c8-9ef4-a726dc808323 |
| Namespace | pantheon-p0-seaweed-velero-s3-v1              | ea045ce0-48bf-47c3-9f2c-f3b933e459d6 |
| Service   | pantheon-p0-seaweed-velero-s3-v1/seaweedfs-s3 | 97c4a5d0-203d-473f-8068-cdc1a29e1490 |
| Endpoints | pantheon-p0-seaweed-velero-s3-v1/seaweedfs-s3 | b6ab7b83-3a24-4b10-87d1-fae129a51dca |
| Secret    | velero/cloud-credentials                      | 29994a5b-c619-4277-b461-0e59c1632a8e |

No Velero CRDs/controller were created. Native disk/auth/Seaweed were healthy;
the original actual SigV4/anonymous/bad-signature checks and 128-byte object
PUT/GET/DELETE ran before the failure. The exact source control flow and failure
stage establish that provenance; v5 does not claim to rerun that object probe.
Fresh diagnostic uses authenticated bucket GET and real denial checks only.

Actual root-only read-only diagnostic
`resume-velero-v4-install-diagnostic-20261110T021316400477.json` shows that the
official Velero CLI dry-run failed with no kubeconfig, although native K3s
kubectl can use its own configuration automatically. V5 derives the exact
original public dry-run argv and adds only
`--kubeconfig /etc/rancher/k3s/k3s.yaml`. Require canonical root-owned regular
single-link config, private mode 0600/0400 and trusted parent directories. Config
contents stay in the native CLI process; the helper records metadata only.

Plan: seal original bytes and the failure -> collect corrected actual dry-run
shape and native ownership evidence -> recheck the approved snapshot -> run
only remaining original manifest/controller/BSL statements -> run original
real validation and exact fixture cleanup. Reuse existing helpers and Python
stdlib; no dependency or product implementation is added.

### Candidate files and local verification

Final [root request](resume-velero-s3-v5-root-request.json), SHA256:
`723824066805739acad3f52011c299a947f01c8035b872193c9113b4663949d6`.
It binds all six v5 helpers, all six v4 helpers, original script hashes,
17 product/driver sources, the actual driver executable, native binary/archive
records, the five exact UIDs and 135 protected historical inputs/evidence files.
Earlier v5 request candidates remain under their own SHA256 filenames.

| New ignored helper              | Responsibility                                                                      |
| ------------------------------- | ----------------------------------------------------------------------------------- |
| resume-velero-s3-v5-root.py     | Explicit recovery predecessor, upload/hash gates and original validation adapters   |
| resume-velero-s3-v5-native.py   | Read-only exact native/config/five-UID/Secret comparison and unknown-object guards  |
| resume-velero-s3-v5-remote.py   | Corrected dry-run, source-derived remaining install and explicit progress migration |
| resume-velero-s3-v5-fixture.py  | Byte-identical v4 fixture; still imports the unchanged v4 provider                  |
| resume-velero-s3-v5-generate.py | Local-only immutable request and identical fixture preparation                      |
| resume-velero-s3-v5-test.py     | Local mock/adapter refusal tests; never real backup evidence                        |

Fresh local evidence: [commands](resume-velero-s3-v5-commands.json),
[derivation](resume-velero-s3-v5-derivation.json),
[source comparison](resume-velero-s3-v5-source.diff),
[review](resume-velero-s3-v5-review.md).
32 v5 tests and 25 unchanged v4 tests pass, along with six-helper syntax compile,
sealed input checks, ignored-path checks, docs frontmatter and task template.
Ruff/pyflakes are not installed; no package installation was attempted.
No Go sources/builds changed, so this preparation does not claim a new Go race
or product build. All original executable/source verification remains bound.
CodeGraph tools were not exposed; the explicitly supplied source files were
read directly. No UI changed, so rendered UI evidence is not applicable.

### Root commands: local check, then reviewed native diagnostic

From the Ops root in the existing authorized root network/credential lane:

```powershell
$veleroV5RequestSHA = '723824066805739acad3f52011c299a947f01c8035b872193c9113b4663949d6'
$veleroV5Evidence = '.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109'
python -B .local/ops-acceptance/resume-velero-s3-v5-root.py check-inputs
if ($LASTEXITCODE -ne 0) { throw 'v5 sealed local input check failed' }

$veleroV5Diagnostic = python -B .local/ops-acceptance/resume-velero-s3-v5-root.py diagnostic --request-sha256 $veleroV5RequestSHA | ConvertFrom-Json
$veleroV5DiagnosticCode = $LASTEXITCODE
Get-Content (Join-Path $veleroV5Evidence $veleroV5Diagnostic.artifact)
if ($veleroV5DiagnosticCode -ne 0 -or $veleroV5Diagnostic.exitCode -ne 0) { throw 'v5 diagnostic refused; retain all state and review the safe evidence' }
```

Diagnostic first accounts for the two exact v4 failure products omitted from
the failed root remote manifest: one typed timestamp failure and the canonical
five-UID progress file. Their exact bytes/hashes must match the original v4
persist serialization. Unknown/missing entries fail before uploads. Uploads
use exclusive `wx`; only the three new v5 remote/native/fixture helpers are
transferred. Existing helpers require exact local hashes; no originals or
archives are overwritten or retransferred.

The native diagnostic does not create/update/delete Kubernetes resources,
import images, start/stop units, mount/unmount, or alter disk/auth/config.
It runs the corrected official CLI only with `--dry-run --no-secret`.
Actual v1 List or separate JSON document output is parsed with duplicate-field,
kind/schema/scope/identity bounds, pinned-image checks and the original limits.
Root receives output shape/counts/hashes, safe object identities and policy hash.
The full manifest never enters retained evidence. If later native checks
refuse, safe official output evidence is still retained in the failure record.

Root reviews the complete receipt before using its hash for completion:
all five live UID/owner/component checks; exact selectorless ClusterIP service
and private endpoint; in-memory live Secret versus strict dedicated native auth
comparison; no existing Velero CRD/controller or extra owner/cluster objects;
successful discovery/LIST of every listable type in both namespaces; canonical
256Mi ext4 loop backing inode/source/options with no child mounts; exact weed
PID/executable/argv, private eight listeners, cgroup 160Mi/no swap/25% CPU;
native image digests/binary/archive/source hashes, memory/disk headroom and real
authenticated S3 read/anonymous/bad-signature denial.

Kubernetes-created `default` ServiceAccounts without Secrets/imagePullSecrets
and `kube-root-ca.crt` ConfigMaps with only the CA key are recorded separately
with UID/shape hash, never adopted into APPLIED. Every other extra object,
including an unreviewed EndpointSlice mirror, refuses and reports only safe
identity metadata. If this happens, root must inspect the concrete receipt and
prepare a separately sealed narrow compatibility path; v5 never guesses that
such an object is owned or deletes it. This preserves the no-extra-object gate.

### Root commands: bounded completion, then unchanged real validation

After root review of the diagnostic receipt:

```powershell
$veleroV5Complete = python -B .local/ops-acceptance/resume-velero-s3-v5-root.py complete --request-sha256 $veleroV5RequestSHA --receipt $veleroV5Diagnostic.artifact --sha256 $veleroV5Diagnostic.sha256 | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $veleroV5Complete.exitCode -ne 0) { throw 'v5 completion failed; retain receipts/resources and do not replay' }
Get-Content (Join-Path $veleroV5Evidence $veleroV5Complete.artifact)

$veleroV5Validate = python -B .local/ops-acceptance/resume-velero-s3-v5-root.py validate --request-sha256 $veleroV5RequestSHA --receipt $veleroV5Complete.artifact --sha256 $veleroV5Complete.sha256 | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $veleroV5Validate.exitCode -ne 0) { throw 'v5 actual validation failed; retain partial fixture evidence' }

$veleroV5Cleanup = python -B .local/ops-acceptance/resume-velero-s3-v5-root.py cleanup --request-sha256 $veleroV5RequestSHA --receipt $veleroV5Validate.artifact --sha256 $veleroV5Validate.sha256 | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or $veleroV5Cleanup.exitCode -ne 0) { throw 'v5 final fixture cleanup/persistence verification failed' }
Get-Content (Join-Path $veleroV5Evidence $veleroV5Cleanup.artifact)
```

There is deliberately no `install`, `clean-native` or full-installer entry.
Completion reruns dry-run and native checks and requires the approved manifest
hash/policy/shape and stable source/UID/auth/mount/unit/config metadata. Dynamic
memory/disk usage is observed freshly; a running data image is not required to
have a frozen content hash. Every remaining object is checked absent before
the first new create, then original `apply()` repeats the absence check.

Before canonical progress evolves, exclusive
`resume-velero-s3-v5-original-partial-owned.json` preserves its original exact
bytes. Exclusive `resume-velero-s3-v5-migration.json` binds the root request,
canonical failure hash, retained failed remote timestamp, original five UIDs,
old progress hash, approved manifest hash and canonical output paths.
The canonical provider receipt must have been absent. This explicitly accounts
for migration of a known failure/progress state rather than treating the v4
install as successful or refreshing an unknown remote manifest.

APPLIED starts with the five verified original UIDs and adds only actual new
CREATE UIDs. Only the original v4 tail from `images = ...` through controller
rollout, real BSL Available/no restart, and success persist executes. Its sole
statement replacement supplies the reviewed corrected manifest. Original
namespace/Service/Endpoints/Secret creates, OCI import, disk creation/mount,
auth generation, native unit launch and object probe are never replayed.
Actual success writes compatible `resume-velero-s3-provider.json`, retains
the original failure bytes, and gets a distinct v5 completion timestamp.
Root accepts only these exact typed products and their UID inventory/hashes.

Validation and `remote_phase()` use the original v4 function bodies with only
the explicit v5 fixture filename/global hash adapters. Fixture bytes/functions
are identical. Root runs the original executable, finite-SA authorization and
isolated MySQL setup, actual Backup/Restore, exact ConfigMap content and namespace
non-PV intent assertions, then original UID-precondition fixture cleanup.
Validation already attempts cleanup in `finally`; the final explicit cleanup
rechecks finite SA/bindings/fixtures/private client material absence and real S3
persistence while retaining provider/disk/auth/data. Failed/ambiguous fixture
preparation is not asserted as successful cleanup.

### Review and stop condition

Independent root review/execution is mandatory for this candidate. The local
self-review covers source derivation, source seals, non-replay, migration,
credential redaction, finite cleanup and refusal paths; no delegated review was
performed because this Generator's scope prohibits delegation.

Gate Outcomes: source/AST checks caught adapter/native path mismatch in an
initial Windows-only mock (corrected by POSIX fixture paths); refusal tests catch
changed source/UID/owner/credentials, unknown objects, stale manifest/snapshot and
failed predecessor reuse; none of these mocks establish live backup acceptance.

Stop condition reached: scoped local preparation and fresh verification are
complete; root-only diagnostic/completion/real Backup/Restore remain pending.
D1/Base IAM HTTP audit, production G2/G3, overall P0 and release decisions remain
open. A failed v5 completion requires inspected recovery; never rerun the full
v4 installer over retained resources or convert a failure receipt into success.
