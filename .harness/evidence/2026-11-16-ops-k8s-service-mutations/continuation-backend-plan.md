# C04 backend continuation plan

Task: `2026-11-16-ops-k8s-service-mutations`; parent: `2026-11-16-ops-pilot-delivery`; durable goal remains leader-owned G003. Original In/Out and acceptance remain unchanged.

Owner: business/k8s/service, plus the narrow namespace admission method and owner API bindings approved by the coordinator. No platform/system, foundation lock, frontend, production resources, or global goal changes.

1. Preserve the current red checkpoint with real MySQL `testmysql.Open`, encrypted fixture kubeconfig, real client-go HTTP and a bounded httptest Kubernetes API. It enforces supplied resourceVersion/UID for update/delete, and is not live provider acceptance.
2. Add PUT for structured Service type/selector/ports. Publish resourceVersion and UID on list/detail. Preserve Kubernetes-assigned fields and named targetPort values. Refuse missing/stale versions before modifying a current object and retain optimistic concurrency on the Kubernetes write.
3. Delete requires resourceVersion and uses UID/resourceVersion preconditions; an unversioned compatibility seam must fail closed rather than read and adopt the current resource version.
4. Extend Namespace owner admission with a scoped Cluster projection and exact BusinessScopeID/environment matching, then reuse its existing action allow-list. Service never reads NamespaceBinding/Cluster tables directly.
5. Add only Service list/view/create/update/delete provider-owned API bindings. Test actual registered routes and no mutation grants from read keys. Synchronize stable error/audit keys through backend seed and the separate UI/canonical lane.
6. Coordinator runs red and green CGO race, broader K8s compatibility, full race, formal rebuild/source binding, live role/provider validation and rendering. Independent reviewer and architect assess the final candidate. No completion/archival claim before these gates.

Initial production findings: no PUT endpoint; ServiceItem drops resourceVersion/UID and named targetPort; DELETE supplies empty options; namespace action guard does not compare binding scope/environment with the authorized Cluster. These are static findings until the coordinator's red receipt confirms the dynamic cases.

Planned source files: `backend/modules/business/k8s/service/{service.go,handler.go,mutation_contract_test.go}`, `backend/modules/business/k8s/namespace/namespace_service.go`, `backend/modules/business/k8s/{permission_routes.go,permission_routes_test.go,k8s.go,k8s_seed.go}`. Additional tests are added only to verify required owner semantics; no new dependency.
