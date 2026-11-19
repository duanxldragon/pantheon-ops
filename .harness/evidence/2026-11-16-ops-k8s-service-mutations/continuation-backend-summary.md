# C04 backend validated checkpoint

The structured Service backend candidate passed coordinator-run real disposable MySQL, CGO=1 race: ten mutation top-level tests and one permission top-level test, including all opaque-port cases. Exit 0; Service package 7.226 seconds. Root receipt: `../2026-11-17-ops-history-audit/continuation-c04-opaque-port-fixed.log.receipt.json`, 2026-11-17T16:05:53.5812651Z–16:07:03.3712943Z.

`continuation-backend-opaque-port-verification.json` binds the original receipt to all nine current source hashes; every hash matches `continuation-backend-opaque-port-fixed-source.json`. Prior source checkpoints and red/green receipts remain unchanged.

## Implemented contract

- Read UID/resourceVersion and native numeric-or-named targetPort; revision-checked structured PUT preserving object metadata and ClusterIP allocations.
- Required UID/resourceVersion on DELETE with provider preconditions; no identity-fetch fallback or retry around a conflict.
- Scoped Cluster/NamespaceBinding environment, business-scope and exact-action admission before provider mutation; caller context reaches the new owner seams and client-go.
- Exact Service list/view/create/update/delete route bindings, with no Cluster/Node grants.
- Provider fault classification retains real causes; known 404/403/409 remain distinct from unknown failures.
- Existing AppProtocol survives matching logical ports: named ports use unique name; unnamed ports use exact protocol + port. New identities do not inherit hidden data; exposed routing fields remain request-controlled.
- Eighteen canonical text keys × zh/en seed synchronization: thirty-six exact comparisons, zero failures, matching canonical locales. See `continuation-seed-canonical-verification.json`.

## Findings and verification boundary

`C04-R1-01` had a genuine provider-bound red: three preservation subcases failed, two new-port controls passed, with actual GET/PUT/mutation controls. The final root green passes all five. Local gofmt, gopls, scoped diff checks and K8s package compile passed. Code round 2 is pending; root reports the narrow architecture scope CLEAR and the whole task WATCH.

The HTTP fixture uses a simulated bounded Kubernetes API via real client-go. It does not prove live role/provider admission, NodePort/LoadBalancer lifecycle/type-transition behavior or common owner lifetime through external commit. Existing RBAC query context and provider in-flight cancellation are the next scoped loop. Full unfiltered race, official rebuilt-product equivalence, final runtime/source binding, rendered visual acceptance and maintainer final acceptance remain coordinator-owned. No archival or full-task completion is approved.

The executor freezes backend source while root reruns C01. Only evidence preparation proceeds until root releases the shared backend window; the new RBAC tests are not included in this validated checkpoint.
