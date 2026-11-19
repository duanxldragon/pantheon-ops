# C04 backend candidate handoff

Task: original `2026-11-16-ops-k8s-service-mutations` and its continuation plan. Business/K8s scope only. No shared platform/system or foundation-lock changes by this lane.

## Contract and owner boundary

- List/detail publish the provider UID and resourceVersion. targetPort is the native Kubernetes integer-or-string JSON value, including named ports.
- PUT `/api/v1/business/k8s/clusters/:id/services/:name?namespace=N` accepts type, selector, ports, required resourceVersion and optional UID. It checks the caller revision against the fetched Service, edits a DeepCopy's routing, and submits the same provider revision. Unrelated metadata and allocated ClusterIP fields remain intact; provider admission decides provider-specific constraints and allocations.
- DELETE requires namespace, resourceVersion and UID query values. It sends both Kubernetes preconditions; missing identity never fetches a replacement identity. The legacy unversioned Manager method deliberately refuses the operation.
- Service writes ask Namespace owner to compare the binding scope/environment with the scoped Cluster owner projection and allow the exact requested action. Service does not query other owner tables. Cluster identity/credential queries and client-go operations receive the caller context with a bounded 30-second operation deadline.
- Five provider-owned permission bindings map list/view to the two GET routes, create to POST, update to PUT and delete to DELETE. The registry creates no direct role grants and includes no node or cluster mutation routes.
- Known provider NotFound/Forbidden/Conflict/Invalid errors retain their cause and a typed public classification. Conflicts use actual HTTP/code 409. Unknown provider causes remain unchanged and use common logging; unknown detail-read failures use the existing generic `request.failed` key.

## Evidence

Source checkpoints: `continuation-backend-expanded-source.json`, then `continuation-backend-provider-cause-source.json` for the subsequent narrow detail-read error correction.

Coordinator real disposable-MySQL/CGO=1 race receipts in `../2026-11-17-ops-history-audit/`:

- `continuation-c04-mutation-provider-bound-red.log.receipt.json`: effective corrected red with allowed provider control. Earlier red fixtures were SSRF-blocked and do not establish provider arrival.
- `continuation-c04-permission-routes-red.log.receipt.json`: five missing action bindings.
- `continuation-c04-mutation-first-fixed.log.receipt.json`: exit 0, six mutation plus one permission top-level tests; Service 2.802 seconds.
- `continuation-c04-mutation-expanded-fixed.log.receipt.json`: exit 0, eight mutation plus one permission top-level tests; Service 5.346 seconds. Includes provider-generated after-GET conflict, missing UID, cancellation, and create/update/delete scope/environment/action denials.
- `continuation-c04-mutation-provider-cause-fixed.log.receipt.json`: exit 0, nine mutation plus one permission top-level tests; Service 5.816 seconds. Its extra real REST regression distinguishes 404, 403 and 500 through Manager and HTTP; no separate dynamic-red assertion is made for this source-inspected fallback defect.

Local gofmt, scoped diff check and Service diagnostics passed; `go build -p=1 -mod=readonly ./modules/business/k8s/...` passed. See `continuation-validation-boundary.md` for exact limits.

## Remaining handoff

Backend seeds now match the UI lane's eighteen canonical keys in zh/en: thirty-six exact values, with unique seed entries and matching canonical locale values. `continuation-seed-canonical-verification.json` records the check and hashes. This lane owns only `k8s_seed.go`, and does not edit generated frontend fallback. Frontend structured forms and permission controls belong to the UI lane.

Independent review found MEDIUM opaque-port loss `C04-R1-01`: replacement ports drop existing AppProtocol even for selector-only edits. Coordinator red receipt `../2026-11-17-ops-history-audit/continuation-c04-opaque-port-red.log.receipt.json` records exit 1: three preservation cases failed and two new-port controls passed, with real GET/PUT controls. Root released the minimal repair after this red. The new logic carries forward AppProtocol for a unique named-port name, or exact protocol + port for unnamed ports. Renamed/new unnamed identities do not inherit unrelated port data; every exposed port field remains caller-controlled. This does not add a UI/DTO field, an alternate write path or a retry. Source `continuation-backend-opaque-port-fixed-source.json` is frozen for coordinator green and independent re-review; earlier green receipts describe the previous checkpoint only.

The owner admission check does not lock Cluster/NamespaceBinding lifetime through the external Kubernetes commit. The test provider is a bounded in-memory API reached by real client-go, not live Kubernetes or production role/admission evidence. Full compatibility/race, official rebuild, final source/runtime binding, real provider/role verification, desktop/mobile rendering, independent reviews and maintainer final acceptance remain coordinator-owned. This handoff neither closes C04 nor authorizes archival.

## Final opaque-port checkpoint update

The candidate above subsequently passed coordinator CGO=1 race with real MySQL: ten mutation top-level tests and one permission top-level test, Service 7.226 seconds. `continuation-backend-opaque-port-verification.json` matches all nine source hashes and binds the final root receipt; code round 2 is pending. See `continuation-backend-summary.md`. Earlier receipts remain unchanged. The next RBAC cancellation loop adds tests in a separate candidate and must obtain fresh red/green evidence.
