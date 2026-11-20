# C04 continuation plan

Status: in-progress. G003 and original C04 packet In/Out remain authoritative. Tier L2; business/k8s only, no foundation or shared-system change.

Ownership: c04_k8s_service_executor owns service CRUD, its tests, namespace admission helper, Cluster request-context projection seam, and exact K8s service action routes/seeds. C03 executor separately owns K8s capability error preservation and its tests. Coordinator owns governance, root race execution, runtime identity and evidence. Frontend remains assigned only after its explicit owner handoff.

1. Preserve the original red runs. The first fixture was blocked by production SSRF before reaching the provider; corrected envelope-only checks still do not prove mutation or target identity. Validate the revised fixture with a successful allowed create and provider call counter before interpreting rejection tests.
2. Reuse real client-go, disposable MySQL and an httptest proxy that responds only to the fixed test host/path and never forwards network traffic. Do not weaken production SSRF. This is simulated-provider contract evidence, not live Kubernetes acceptance.
3. Expose UID/resourceVersion in lists and detail; preserve numeric and named targetPort. Add PUT for selector/ports/type with a required expected resourceVersion and optional update UID check. Preserve Kubernetes allocated ClusterIP and unrelated metadata by updating the fetched object, not replacing it with a sparse request.
4. Public DELETE requires both expected UID and resourceVersion and submits Kubernetes preconditions. Missing identity is invalid; stale or recreated-name operations return conflict and leave the provider object intact. Internal legacy behavior must be explicit and cannot become an HTTP fallback fetching current identity.
5. Namespace owner admission must compare Binding scope/environment to an authorized Cluster projection and check the required action before provider mutation. Add a minimal Cluster GetByIDWithContext seam and pass the request context; no direct cross-owner table reads.
6. Bind service list/view/create/update/delete keys to their precise actual methods/routes through the existing registry. Do not broaden to cluster/node routes or grant roles directly. Synchronize the business seeds, five canonical locales, UI forms and smoke.
7. Run deterministic red→green, service/K8s compatibility tests, independent code/architecture reviews, current frontend and rendered evidence, final full CGO race and official-lock rebuild/source binding. Full production Kubernetes, kubectl interference, role/environment acceptance and maintainer final gate remain open until actually evidenced.

Stop only dependent work if production credentials, irreversible operations or gate waivers are required; continue safe local implementation. No archive or aggregate completion is implied by partial passing tests.
