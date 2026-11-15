# Velero live adapter handoff — live blocked, offline fallback verified

Task: `2026-11-07-ops-planned-task-closeout`, P0 D1 continuation; L2 business-domain acceptance. Parent: `.harness/tasks/2026-09-28-ops-delivery-readiness-gap-closure/packets/D1-external-integration-acceptance.md`.

## Finding first

**Confirmed:** `BackupService.CreateRestore` accepts a foreign department's known backup/cluster, persists a restore and returns `New` before checking DataScope in its asynchronous worker. A real isolated MySQL reproduction used caller Dept20 against a Dept10 cluster. The worker then set `Failed`; there was no remote API request. This is a registry/authorization defect, not proof of a successful unauthorized Kubernetes restore. Source fixes are left to the root owner. See `velero-live-negative-paths.json` and `velero-live-closeout.json`.

## Finished evidence

- 30 offline checks passed using actual public product Services and a fresh isolated MySQL schema. Seeded registry rows are explicit fixtures and do not represent completed backups/restores.
- Coverage: owned/foreign DataScope list/get/delete, missing backup/restore, denied create with no persistence, department and credential-reference revocation including an already cached lazy client, deletion failure retaining registry, cancellation/expired deadline, public-origin lazy config validation, private/metadata/proxy/insecure-TLS rejection, and the async authorization finding.
- Driver: `CGO_ENABLED=1`, MinGW `D:/msys64/mingw64/bin`, `go build -mod=readonly -race` and `go vet -mod=readonly`; both exit0.
- Fresh tests: `go test -mod=readonly -race -count=1 -timeout=120s ./modules/business/k8s/backup ./modules/business/k8s/pkg`; both packages pass. `GOMAXPROCS=2` bounds local verification. These include client mocks and are not live Velero acceptance.
- Python drivers passed AST syntax checks. Exact commands, durations, receipts and artifact SHA256 values are in `velero-live-verification.json` / `velero-live-closeout.json`; compiler/vet/race logs use the same prefix.

## Live blocker and contracts

`remote.py connect()` was used with existing credentialref1 and pinned SSH host policy, without printing credentials or configs. The current execution environment rejected external SSH socket connection with `PermissionError`, errno13, before remote commands ran. `velero-live-preflight.json` records sanitized frames. This is not evidence that the server/provider is down. No attempt was made to bypass the runtime restriction.

Actual memory headroom, reboot health, Argo960Mi cap, provider ownership, Velero/MinIO installation, trusted server CA and SAN, finite SA permissions and official release provenance could not be checked. No release tags/images were guessed, downloaded or installed. No successful live backup, S3 object, restore data, live RBAC or cleanup result is claimed.

Current public client fixes its namespace to `velero`; it cannot select a differently named test provider namespace. Actual inventory must determine whether `velero` is safely owned/available. `IsInstalled` reads a cluster-scoped Namespace, which is outside a purely namespaced Role. Storage-location methods only manage DB rows and cannot install/configure the real store. Product backup deletion issues a direct CR delete; S3 content cleanup requires separate evidence. These are source boundaries, with locations recorded in `velero-live-closeout.json`, not claims based on a live provider.

## Cleanup and ownership

The exact fresh isolated schema was dropped and `information_schema.SCHEMATA` confirmed absence. No remote resource, token, MinIO key or credential file was created. No core provider was installed or retained by this adapter; existing Argo, datasources and other remote resources were not inspected or modified. Local ignored driver/source/binary artifacts remain for root reuse. Observed backend source hashes were unchanged. Only `velero-live-*` drivers and this evidence family were written; no checkpoint/commands/summary/overlay, config, backend/frontend source, commit, push, publication or baseline replacement.

Live D1 remains blocked by the executor's outbound SSH restriction. Independent maintainer review, whole D1/P0 acceptance and D4 production signed G2/G3 remain open. Findings-first self-review was used because the assigned leaf role forbids recursive agents/CLI; root owns integration and later source fixes.

Gate Outcomes: DataScope diagnostic caught unauthorized restore registry creation | SSH preflight caught denied external connection | none
