# C04 architecture round1 — historical report transcription

Provenance: coordinator condensed the original full round1 text returned by the independent `/root/c03_owner_architecture` lane, available in the tool mailbox. Transcription date 2026-11-18 Asia/Shanghai; original exact review time not supplied. This is a historical review and does not attribute round2 fixes or later UI to round1.

**WATCH. C04 not approved for archive.** Eight manifest source hashes matched; nine Service mutation tests plus one permission test were observed passing. No tests or runtime edits by the reviewer.

Finding C04-ARCH-R1-01 (P2): sparse replacement of Spec.Ports drops existing AppProtocol because DTO/toServicePorts supplies only name/port/nodePort/protocol/targetPort (`service.go:32`, `:232`, `:306`). At this round it was source inference, not dynamically reproduced. Existing preservation fixture checked ClusterIP/metadata/named targetPort only (`mutation_contract_test.go:160`, `:265`). Add a retention RED and define logical port matching; NodePort/LB full behavior remains unproven.

UID/RV protection was accepted within scope: GET/list identity, PUT stale comparison, DELETE requires both identities and Kubernetes preconditions (`service.go:218`, `:242`, `:254`, `:278`). Provider-revision change and delete cases proved zero stale mutations with real client-go and simulated REST, not a live cluster (`mutation_contract_test.go:282`, `:297`, `:32`).

Namespace owner checks exact Cluster-scoped projection against Binding scope/environment/action (`namespace_service.go:80`, `:101`; `service.go:328`) before mutations; denial tests check zero provider calls (`mutation_contract_test.go:331`). No common owner lifetime lock covers admission to external commit (`namespace_service.go:77`, `service.go:211`, `:263`). RV does not order Binding withdrawal or Cluster rebinding.

Handler passes request context, Service sets timeout, new Cluster/Binding queries are contextual; existing production RBAC Binding SQL is not (`handler.go:116`, `rbac/rbac.go:122`). HTTP fixture uses nil RBAC (`mutation_contract_test.go:196`), and cancellation is pre-provider (`:366`), so production RBAC/in-flight cancellation were open.

Five Service action policies match only actual GET/list/detail and POST/PUT/DELETE routes; nodes are not granted (`permission_routes.go:7`, `permission_routes_test.go:16`, `:55`). No runtime role/seed login acceptance was inferred. Provider classification retains causes; known status maps to correct business errors, otherwise returned unchanged (`service.go:335`, `handler.go:153`).

Original final provider-cause GREEN: exit 0, CGO/race/disposable MySQL, 9 mutation +1 permission, Service5.816s (`continuation-c04-mutation-provider-cause-fixed.log.receipt.json`; log `:450`, `:452`). The valid third RED included successful allowed-create/provider counters; earlier SSRF-blocked fixtures were not mutation evidence. Original manifest 8/8 matched and explicitly liveProvider=false/archiveApproved=false.

Recommended narrow fixes: appProtocol red→green, request-context RBAC seam, non-nil admission, provider-entry cancellation barrier. Owner lifetime, NodePort/LB, actual roles/environment/provider, UI/seed and final full race/rebuild/binding remained open. Later round2 independently closes the appProtocol finding.

## Machine Readable

```json
{
  "task": "C04",
  "review": "independent-architecture-round1-historical-transcription",
  "agentRole": "architect",
  "reviewer": "/root/c03_owner_architecture",
  "architecturalStatus": "WATCH",
  "sourceHashMatches": 8,
  "sourceHashTotal": 8,
  "newTestsExecuted": false,
  "wholeTaskComplete": false,
  "archiveApproved": false,
  "finding": {
    "id": "C04-ARCH-R1-01",
    "priority": "P2",
    "statusAtOriginalReview": "OPEN",
    "evidenceType": "source-inference",
    "dynamicReproductionAtOriginalReview": false
  },
  "verification": {
    "exitCode": 0,
    "topLevelMutationTests": 9,
    "topLevelPermissionTests": 1,
    "servicePackageSeconds": 5.816,
    "liveProvider": false
  },
  "provenance": "Condensed historical transcription from original full mailbox report; later fixes excluded"
}
```
