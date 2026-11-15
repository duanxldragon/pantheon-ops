# Velero resume: local authorization repair verified, live phases staged

L2 business-domain implementation under the D1 parent packet: `.harness/tasks/2026-09-28-ops-delivery-readiness-gap-closure/packets/D1-external-integration-acceptance.md`. The change stays in `backend/modules/business/k8s/backup`; shared platform/system behavior is reused through existing cluster authorization and transaction helpers. This is a logical business-domain repair, with no physical domain split.

The original CreateRestore returned New and persisted foreign-department attempts before its worker checked scope. Fresh required MySQL red evidence reproduced that behavior and also caught canceled/deadline requests and revoked cached credentials. The repair authorizes before source lookup or provider calls, validates the matching registry and actual live Completed/unexpired source, then rechecks scope/credentials under the existing cluster lock and locks the source row before commit. The worker reauthorizes and revalidates the source. Public constructor/service signatures are retained. The successful mapping test also exposed and fixed map[string]string normalization within the Velero unstructured boundary.

## Concrete file plan and verification

- `backup_service.go`: reuse scoped/context-aware cluster helpers; reject before persistence; live source validation; transactional source/cluster checks; worker recheck.
- `restore_authorization_test.go`: actual required isolated MySQL with only provider IO replaced; foreign ID/name guessing, missing/revoked source/credentials including cached clients, canceled/deadline contexts, registry/live eligibility, revocation during the live read, row-lock probes, failed worker source and authorized enqueue with namespace mapping.
- `velero_client.go`: normalize namespace mapping to a JSON-native string-keyed map.

Fresh `go test -mod=readonly -race -count=1 -timeout=120s -v ./modules/business/k8s/backup`, `go vet -mod=readonly ./modules/business/k8s/backup` and `go build -mod=readonly ./modules/business/k8s/backup` all exit0. Required MySQL is enabled, cgo=1, MinGW path is D:/msys64/mingw64/bin, GOMAXPROCS=2. The new ignored live driver also passes race build and vet. Python drivers parse successfully. `git diff --check -- backend/modules/business/k8s/backup` is clean. Exact source/driver hashes and command receipts are in `velero-resume-closeout.json`; red/green logs are separate. No skipped tests are claimed as coverage.

## Runtime handoff and remaining gaps

Root should run the concrete script/arguments in `velero-resume-phase-request.json`. It starts with inspect; each successful phase writes the next request. Do not retry leaf SSH or interpret errno13 as provider failure. This continuation did not attempt SSH or mutate the server. Local default Go cache was denied, so verification used this lane's ignored cache.

1. Inspect: fresh memory/provider ownership, pinned SSH helper, verified public API TLS with server-name kubernetes, official fixed release API records.
2. Download-install: verified Velero v1.17.0 archive checksum, official AWS plugin v1.13.0 and MinIO RELEASE.2025-09-07T16-13-09Z release provenance, registry response digests, then labeled exact resources. Velero and MinIO are capped at 256Mi each; new scheduling requires at least 1024Mi available. Core Argo remains untouched. Controller BSL must become Available.
3. Validate: finite 15m SA with namespace-scoped provider access and only get Namespace/velero at cluster level; public product service create/get/list/delete, actual Completed backup/restore, matching ConfigMap data, real provider RBAC/missing storage, revocation, deadlines. Inspect S3 object bytes/hashes and delete only exact owned object prefixes and exact UID/label checked test resources. Retain explicitly labeled provider core/CRDs and their provider credential Secrets; remove finite client SA/bindings and transient client credential file. Drop the exact isolated local MySQL schema and verify absence.

These are staged runtime assertions, not observed live success. Full D1/P0 acceptance, root integration gates/full repository race/overlay projection and independent review remain open. D4 signed production recovery is out of scope. No UI, shared K8s/pkg, frontend, SQL, lock, overlay, common ledger, commit/push/publication or baseline was changed by this lane. CodeGraph MCP was not exposed; focused native reads were used. Findings-first self-review found no remaining local authorization defect in the covered path; independent review belongs to root under this leaf's delegation prohibition.

Gate Outcomes: required MySQL hostile regression caught unauthorized persistence and revoked cached credentials | successful mapping case caught unsupported unstructured map | row-lock probes verified serialized commit.

## Machine-readable self-review

```json
{
  "approvedScope": "local backup repair only",
  "localVerdict": "verified",
  "liveVerdict": "pending root phase execution",
  "independent": false,
  "findings": [],
  "residualRisks": [
    "Root must execute staged inspect, download-install and validate through its authorized execution path; this leaf must not retry known sandbox SSH errno13.",
    "Fixed official releases, image digests, actual memory and TLS SAN are runtime assertions in staged scripts; not yet observed by this continuation.",
    "Successful live backup/S3 persistence/restore content, actual provider RBAC and remote cleanup are pending.",
    "Whole D1/P0, independent review, full repository race and overlay/gate integration belong to root; no common ledger or overlay changed.",
    "D4 signed production recovery/RPO/RTO excluded."
  ],
  "sourceHashes": {
    "backend/modules/business/k8s/backup/backup.go": "bb6162665a7b710351ed6df4f2c204ad3914088aa7119d7cbbaa4a3170b61c8f",
    "backend/modules/business/k8s/backup/backup_dto.go": "b6d0430c93bb33dada0f1155b2bb3ff17cf403fc619ab687ba141e64616aa7e4",
    "backend/modules/business/k8s/backup/backup_handler.go": "320bf427300d6cb88038adfbf581600c8c82cd2f760bdfa755dfca4711d99e96",
    "backend/modules/business/k8s/backup/backup_model.go": "0119686da49fa62c3106ca6a90206592122153eaf5f858ae884a824590c3ea51",
    "backend/modules/business/k8s/backup/backup_service.go": "4320ceb9bfdfa85eb660ef8fcf83dc0b4de02ecddf3bce8b0babd3f91a88e934",
    "backend/modules/business/k8s/backup/drill.go": "0de6e13dc1a2a4f0ea0acb3cd2478d636d2de8ff9a5b07024ba4e18abcddc702",
    "backend/modules/business/k8s/backup/reconciler.go": "044fccce5bf558a6542d33f7d4dcfd7ddd65937d820cde2cb217814de3bbff5c",
    "backend/modules/business/k8s/backup/restore_authorization_test.go": "16b228b1416bda79debb068fe0b00545c14e1c0921d35b3697a27e2e7df197ea",
    "backend/modules/business/k8s/backup/velero_client.go": "b24de4061c69f37daa7b8bfcd13cf5a7132d5f2f0c4fbf286ef46da9aeb0552b",
    "backend/modules/business/k8s/backup/velero_client_test.go": "0272dd81e1e6dee02185d8fbba9119c8767f796edc7073cfecf31d86fcc0a3d9"
  }
}
```
