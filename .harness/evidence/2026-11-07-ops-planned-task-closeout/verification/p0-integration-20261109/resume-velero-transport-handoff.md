# Velero alternate transport preparation — blocked before official image bytes

Parent task: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md` (`2026-11-07-ops-planned-task-closeout`, L2). Scope: business/k8s acceptance fixture, with no platform/system/core/product changes. All scripts and cache are new ignored `resume-velero-*` assets. Existing scripts, evidence, configuration, manifests, locks and ledgers were not overwritten. No remote commands, installation, container changes, commit, push or publication were executed.

## Verified local result

- Reverified and copied the actual official Velero v1.17.0 Linux amd64 archive: 58,499,469 bytes, SHA256 `f507a956cf8ec00f483f5d7b04901614a9016a10feb32b3ecc968da36b4fc7be`. Its official CHECKSUM matches; the archive contains a Linux amd64 ELF binary.
- Parsed the existing official Velero and AWS plugin index bytes and selected their unique Linux amd64 manifests. These remain metadata, not locally cached image layers.
- Added a local-only preparer that verifies an older MinIO release and upstream README registry authority, matches official response bytes to registry digests, then tries digest-pinned Docker pull/save or direct official blobs in an OCI image layout. The OCI path checks compressed blob sizes/hashes, config platform and uncompressed layer digests. No mirrors or TLS/network setting changes.
- Fresh offline check: `python .local/ops-acceptance/resume-velero-transport-selfcheck.py` exited 0; 7 checks passed, including script parse/compile, real archive/indices and rejection of tampered manifests, unknown origins, invalid blob paths and oversized blobs. It performs no provider or backup simulation.

## Exact blockers

| Transport/stage                                                              | Observed result                                                                                                  |
| ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| Existing latest MinIO attempt                                                | Original `velero-root-cache-result.json`: official MinIO stage failed with HTTP 401. No usable image proof.      |
| Local Docker daemon preflight                                                | Exit 1; Docker Desktop Linux engine named pipe absent. Daemon was not started.                                   |
| Fresh official HTTPS (Docker auth, Quay API, GitHub older MinIO release API) | `URLError` caused by `PermissionError`, errno 13, WinError 10013. No HTTP response was received.                 |
| Authenticated existing `gh api` older MinIO release retrieval                | Exit 1; stderr present and suppressed.                                                                           |
| Independent `curl.exe` to official Quay API                                  | Exit 7, HTTP status `000`.                                                                                       |
| Latest preparer attempt                                                      | Exit 1 at `official-image-manifest-velero/velero`, same errno 13 / WinError 10013. No image payloads downloaded. |

The older MinIO tags in the script are candidates only. Neither their release origin nor their registry manifests/digests could be freshly verified here. No MinIO image is certified, no OCI/containerd transport is certified end to end, and no P0/provider/backup/restore success is claimed.

## Concrete root next phase

Use `resume-velero-transport-root-request.json`: phase `local-transport-preparation`, script `.local/ops-acceptance/resume-velero-transport-prep.py`, SHA256 `1c2b41d64b18643833b4ed42af60f86226b1a6e95d9bb67c151362b09f451b2c`. Run it through the existing authorized path where official HTTPS is available. It adds zero remote memory and executes no remote installation. Do not infer installation readiness until it emits exitCode 0 with all three complete image archive/hash records. If official transports remain unavailable, the next alternative is root's read-only inventory for an already installed upstream S3 fixture; no such fixture is assumed present.

Only after verified bytes exist may root serialize its separate remote phase against O2/Argo budgets. That phase must use `remote.connect`'s encrypted credential loading and pinned RejectPolicy; verify uploaded SHA256 and Linux amd64 image content before containerd import; verify exact imported digest references before scheduling with `imagePullPolicy: Never` if registry egress is unavailable. Docker save/import digest preservation is an untested runtime gap and must be checked rather than replaced with a mutable tag.

Later installation must freshly verify `velero` and `pantheon-p0-minio-velero-resume` namespaces and all `velero.io` CRDs are absent, and bound transfer/image/content disk use and the aggregate fixture memory to 512 MiB (MinIO 256 MiB plus Velero 256 MiB; plugin init at most 128 MiB, no node agent). Install official CRDs/controller, wait for Established/rollout/readiness, create a real authenticated S3 bucket and perform actual object PUT/GET or listing verification, then require BackupStorageLocation Available. Keep owner-labelled object and namespace UID receipts, including partial apply failures, for root's scoped cleanup. No broad resource deletion is permitted. Actual backup/restore remains a later required acceptance phase.

## Review and gaps

Self-review only: native leaf ownership forbids recursive delegation. The offline integrity guards passed; the full downloader, Docker save/import, real older MinIO runtime, S3 availability, provider readiness and backup/restore have not been exercised because this lane cannot open outbound sockets. Root must review the new script/hash and execute the next network-capable preparation phase; no existing installer should be presented as green.

Gate Outcomes: official-origin/integrity guard caught tampered metadata and unverified origins; runtime transport guard caught missing Docker daemon and blocked outbound sockets. No bypass was applied.
