# C04 independent architecture review — round 2

Provenance: coordinator transcription, condensed from `/root/c03_owner_architecture` independent read-only report of 2026-11-18 Asia/Shanghai. No reviewer tests or source/runtime mutations.

**C04-ARCH-R1-01 CLOSED; reviewed appProtocol fix CLEAR; whole C04 WATCH.** The manifest alias C04-R1-01 identifies the same finding. No whole-task or archive approval.

## Evidence

The fetched object's DeepCopy is retained. Named ports match unique name; unnamed ports match unnamed + protocol + port. Only matched logical ports copy old AppProtocol; rename/new identities do not inherit it (`service.go:232`, `:238`, `:241`). Normalization rejects duplicate names and requires names for multi-port requests (`:388`). UID/RV checks still precede object mutation and the current revision is submitted (`:222`, `:247`). No new cross-owner access or public DTO was introduced.

Five cases at `mutation_contract_test.go:270` check success/provider count/routing and AppProtocol (`:292`, `:299`, `:302`): same-name selector edit, same-name protocol/port edit and unchanged unnamed identity retain the field; renamed and changed unnamed-port identities do not inherit it. The preserved RED has three retention failures (`continuation-c04-opaque-port-red.log:12`, `:23`, `:34`); all five pass in the final GREEN (`continuation-c04-opaque-port-fixed.log:169`, `:220`). These use real client-go serialization with a simulated REST provider, not a live cluster.

Final receipt: 2026-11-17T16:05:53.5812651Z–16:07:03.3712943Z, required disposable MySQL, CGO=1/race/count=1/readonly, exit 0. Ten mutation tests and one permission test pass; Service package 7.226s (`continuation-c04-opaque-port-fixed.log:83`, `:507`, `:509`). Nine of nine current source hashes match `continuation-backend-opaque-port-fixed-source.json`. Service SHA 38956b4db476fa365fa2daf49c42bab5c8e52184a19c3312c1918bfb25e35f56; test SHA 8c5759893970cd112585491c4d38b0166d3f049a8b5f186562856921905c28d4.

Nine-file hash verification is not a new full semantic review of all nine files. New code review scope is port matching/preservation and corresponding tests. Seed did not receive new semantic or runtime acceptance.

## Remaining gaps and next steps

Real Handler RBAC still precedes the Service operation context and Binding SQL lacks request context (`handler.go:116`, `rbac/rbac.go:122`); fixture RBAC is nil (`mutation_contract_test.go:196`). Add a compatibility context seam and non-nil RBAC cancellation evidence. Existing cancellation is pre-request (`:409`); reuse the bounded provider-entry barrier approach in `capability_error_test.go:107` for in-flight cancellation.

Namespace admission and external provider commit lack a common owner-lifetime protocol (`namespace_service.go:77`, `service.go:211`, `:247`). Kubernetes RV cannot order local Binding withdrawal/Cluster rebinding. A repeat check only narrows the window; it does not prove atomic authorization. First reproduce the admission-to-provider race and define the promise before introducing a large protocol.

NodePort/LoadBalancer allocation/type transitions, complete roles/environment/live provider, UI/seed acceptance, full race/rebuild/final source binding and final maintainer acceptance remain open. This review accepts existing appProtocol preservation only.

## Machine Readable

```json
{
  "task": "C04",
  "review": "independent-architecture-round2",
  "agentRole": "architect",
  "reviewer": "/root/c03_owner_architecture",
  "reviewDateLocal": "2026-11-18",
  "timezone": "Asia/Shanghai",
  "mode": "read-only",
  "newTestsExecuted": false,
  "architecturalStatus": "WATCH",
  "reviewedFixStatus": "CLEAR",
  "finding": {
    "id": "C04-ARCH-R1-01",
    "manifestAlias": "C04-R1-01",
    "status": "CLOSED"
  },
  "sourceManifest": "continuation-backend-opaque-port-fixed-source.json",
  "sourceHashMatches": 9,
  "sourceHashTotal": 9,
  "seedSemanticAcceptanceGranted": false,
  "verification": {
    "redExitCode": 1,
    "greenExitCode": 0,
    "cgoEnabled": true,
    "raceEnabled": true,
    "topLevelMutationTests": 10,
    "topLevelPermissionTests": 1,
    "opaquePortCases": 5,
    "servicePackageSeconds": 7.226,
    "liveProvider": false
  },
  "wholeTaskComplete": false,
  "archiveApproved": false,
  "openGaps": [
    "Production RBAC request context",
    "Provider in-flight cancellation",
    "Cluster/Binding lifetime through commit",
    "NodePort/LoadBalancer allocation and type transitions",
    "Runtime roles/environments/live provider",
    "UI/seed",
    "Full race/rebuild/source binding",
    "Final visual/functional acceptance"
  ]
}
```
