# C04 request-context validated checkpoint

The request-context candidate passed coordinator real disposable-MySQL, CGO=1 race with all thirteen Service mutation, one permission and three RBAC top-level tests. Exit 0; Service package 7.613 seconds, RBAC package 1.355 seconds. Root receipt `../2026-11-17-ops-history-audit/continuation-c04-rbac-cancellation-fixed.log.receipt.json` records 2026-11-17T17:09:57.7032813Z–17:11:09.1160712Z and 729 backend Go/module hashes unchanged before/after execution.

`continuation-backend-rbac-cancellation-verification.json` verifies all eleven candidate hashes against both current files and root's executed after-manifest. The preceding opaque-port checkpoint, its receipt and all red evidence remain unchanged.

## Minimal repair and compatibility

The only new production change in this loop is `rbac/rbac.go`. `CheckHandlerPermission` keeps its signature and existing role/nil authorization handling. Request context reaches the existing namespace query and its repeated query path through `namespaceAllowsWithContext`. The old `namespaceAllows` helper remains a Background wrapper. GORM uses WithContext; the existing boolean error/allow result, LIKE matching, role behavior and table/query contract remain unchanged. No new DTO, dependency, UI field, shared Base code, alternative write path or write retry is introduced.

Test changes are limited to new `rbac/request_context_test.go` and the Service fixture/tests in `service/mutation_contract_test.go`. The fixture can now wire the real non-nil RBAC service; earlier nil-RBAC tests retain their old path. The bounded test provider adds a deterministic GET entry/exit gate with cleanup release.

## Genuine red and fresh green

Root red `continuation-c04-rbac-cancellation-red.log.receipt.json` exited 1 with stable backend source: both standalone RBAC request-cancellation tests failed, as did real RBAC-backed Service create/update/delete cancellation. Active binding/legacy-wrapper control, allowed provider create and both in-flight provider GET cancellation controls passed. This isolates the contextless RBAC query from the already-working client-go read cancellation.

Final green passes those cases and the earlier identity/precondition, concurrency, owner admission, provider cause, exact-route permission and AppProtocol-preservation regressions. Local gofmt, gopls, scoped diff check and K8s package compile passed. Source is frozen for independent review.

## Evidence limits and remaining gaps

The query-entry gate observes GORM Statement.Context and waits for the real request cancellation before the normal Count proceeds. It injects no SQL error. It proves context propagation and denial at the GORM/query boundary; it does not reproduce cancellation of an already-blocked MySQL driver/transport query.

The HTTP control uses non-nil real RBAC and real client-go against a simulated provider. Cancellation occurs after the provider GET entered during Update; the HTTP call fails, the provider read exits and no PUT/mutation occurs. The direct Manager control retains `errors.Is(context.Canceled)`. These cases do not establish full production authentication/Casbin behavior, live Kubernetes, or rollback after a provider already accepted a write.

Cluster/NamespaceBinding lifetime and revocation ordering through external commit remain explicit gaps. NodePort/LoadBalancer lifecycle/type-transition, full runtime role/environment/provider acceptance, complete unfiltered race, official rebuilt-product equivalence, final runtime/source identity, UI rendering and maintainer final acceptance remain coordinator-owned. This checkpoint grants no whole-task or archive approval; the context candidate's independent review remains pending.
