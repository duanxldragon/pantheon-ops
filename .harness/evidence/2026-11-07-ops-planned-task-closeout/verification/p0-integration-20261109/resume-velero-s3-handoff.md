# SeaweedFS S3 transport and root handoff

Parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`. L2, business/K8s acceptance tooling. Only new ignored `.local/ops-acceptance/resume-velero-s3-*` scripts/cache and this evidence directory were written. No backend, Base, UI, SQL source, lock, ledger, commit or release edits. Existing providers, credentials, data, source changes and failure receipts were preserved.

## Finished local work

- Implemented official SeaweedFS **4.53** transport with exact GitHub release/tag/origin/asset checks, HTTPS-only approved redirects, 250 MiB download bound and transfer deadline. Required GitHub asset SHA256: `d1a7dec9f48d8b33895a0ea8951971defc2dc5312906dcc76fa5a5f74a6ee90d`. Extraction validates all member paths/types and emits only a regular amd64 ELF into its owned directory; no `extractall`.
- Reverified and cached the actual official Velero **1.17.0** archive (58,499,469 bytes, SHA256 `f507a956cf8ec00f483f5d7b04901614a9016a10feb32b3ecc968da36b4fc7be`) and binary ELF/hash. The prior official archive/checksum evidence is reused without altering it.
- Reused `resume-velero-transport-prep.py` registry manifest, blob, layer and OCI-layout verification for exactly Velero **1.17.0** and AWS plugin **1.13.0**. Both image stages are attempted independently of Seaweed transport failure. No MinIO manifest, credential request, unavailable registry retry, mirror, mock provider or product dependency was introduced.
- Prepared versioned root diagnostic, conditional installer, fixture, real Service driver and exact cleanup phases. Native Seaweed is limited to 160 MiB/25% CPU, one 256 MiB disk image/one 64 MiB volume; a single Velero Pod is limited to 320 MiB/500m CPU (init and controller run sequentially). Combined provider limit is **480 MiB <= 512 MiB**. Installation refuses less than 1 GiB available memory or 3 GiB disk headroom.
- All native listener addresses must use the actual unique private VM address, with exact ports 19333/29333, 18080/28080, 18888/28888 and 18333. All new remote fixture paths are under `/opt/pantheon-p0-20261109/velero-s3-seaweed-v1`. Namespace/Service/Endpoints labels identify SeaweedFS correctly; the Velero component is labelled separately.
- Installation requires genuine anonymous/bad-signature S3 rejection, SigV4 PUT/GET/DELETE and actual BSL `Available`. Random credentials are confined to memory, a 0600 native auth fixture and the owned Kubernetes Secret. They are never emitted. Finite client/denied SAs use a verified 15-minute token TTL, exact namespace/role permissions and stdin-only driver input.
- The new driver retains the final driver's real backup, restore/ConfigMap equality, department/source/remap/RBAC/missing/deadline/credential-revocation cases and inspects actual Restore namespace/PV/cluster-resource intent. The fixture cleanup uses owner plus UID preconditions and rejects unexpected workloads/content. It retains provider cores, S3 backup contents and evidence.

## Fresh verification

[`resume-velero-s3-verification-20261109T100633950177.json`](resume-velero-s3-verification-20261109T100633950177.json) records **9 passed offline checks**, including parse/compile, actual official archive/indices, tampered release metadata, unsafe tar members/wrong ELF, wrong owner/UID and SSH RejectPolicy guards. These are integrity tests, not simulated provider acceptance.

Fresh driver `go build -mod=readonly -race` passed (8.937s); `go vet -mod=readonly` passed (1.969s), both with explicit `CGO_ENABLED=1`, `D:/msys64/mingw64/bin` and the owned Go cache. Business source and Go module hashes were identical before/after. Driver binary SHA256: `7204fc7cabbfe4ed9b73c4edf83fedd9f917c2f614b943f287baab8da0d5244e`. No fresh full repository race or real provider test was run in this lane.

The first offline failure was a Windows path representation assertion, corrected with `as_posix()`; its [failure receipt](resume-velero-s3-check-failed-20261109T095724511279.json) is retained. Whole-worktree `git diff --check` reports five unrelated EOF whitespace findings in existing task/business/frontend files; those files were left untouched. Scoped ignored-script diff check returned 0; Python compilation and driver build/vet are the substantive diagnostics here.

## Actual transport blocker and concrete next phase

The final [transport attempt](resume-velero-s3-transport-20261109T100823969535.json) freshly reverified Velero bytes, then encountered **URLError / PermissionError / errno 13 / WinError 10013** for Seaweed GitHub metadata and both public image stages. No HTTP response was received. Seaweed release bytes and OCI image blobs are **not cached or certified in this lane**. Existing MinIO 401 and all earlier transport failures remain distinct evidence.

Use [`resume-velero-s3-root-request.json`](resume-velero-s3-root-request.json). It contains every script and current product-source hash, exact commands, scope/revert and review constraints. Primary hashes:

- transport preparer: `0b5219ab2fe498a81b5d224c35dc8eaa7827f68467ab4dbee413a45a6445cc02`
- root wrapper: `65d8155901e32eb76318658972f995b24ec241a38df71d2adb92e0ae694675d5`

Root's already-authorized execution lane must run these first:

```powershell
python .local/ops-acceptance/resume-velero-s3-prep.py
python .local/ops-acceptance/resume-velero-s3-root.py diagnostic
```

The first command must emit exitCode 0 and complete verified Seaweed/Velero/OCI artifact records. The second uses the existing encrypted credential/pinned RejectPolicy helper, verifies uploaded SHA256 and executes actual `weed version`/`weed server -help`. It creates **no Kubernetes objects, image imports or provider process**. Actual **4.53 server flags, including explicit gRPC overrides, have not been confirmed here**. Missing flags return exitCode 1 and retain the actual help; root must review that diagnostic instead of guessing flags. This is a safe diagnostic handoff, not an installation-ready claim.

Only after successful actual diagnosis and root review of resource/bind constraints may the request's `install` and `validate` commands run. The installer requires digest-preserved containerd references and `imagePullPolicy: Never`, official CRDs Established, controller rollout and actual S3 readiness. The validator requires real Backup/Restore Completed with zero errors, matching restored ConfigMap, namespace/PV intent, S3 persisted bytes/hashes and exact cleanup. A failed partial CREATE without a UID is retained for root review; cleanup does not guess ownership. No credentialed remote command was executed in this leaf lane and no permission bypass or new permission question was used.

## Review and remaining gates

Self-review only, recorded in [`resume-velero-s3-review.json`](resume-velero-s3-review.json), because the assigned leaf scope prohibits delegation. The blockers are real network access and unexecuted actual-binary diagnosis/runtime. No independent approval, real Seaweed compatibility, OCI import, provider installation, backup or restore is claimed.

Full Base live actor/session/current-action authorization, audit/HTTP, live lost-response acceptance, full-repository gates and production G2/G3 remain open. Existing deterministic lost-response tests remain separate historical evidence; this driver does not certify a real lost-response transport case. **S3 alone cannot close D1**, and neither parent P0 nor D1 is marked complete.

Gate Outcomes: archive/release/extraction/UID guards caught tampering and unsafe inputs; transport guard caught blocked sockets; initial Windows path assertion was corrected and retained. No provider-success simulation or registry retry was applied.
