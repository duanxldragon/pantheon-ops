# C03 continuation — catalog usability and owner editing

Status: in-progress. Parent G003; original C03 In/Out and acceptance are retained. No dependency on C01; C05/C06 wait for actual C03 acceptance. Tier L2; business/service only, shared platform/system changes remain Base-first.

Implementer ownership:

- `c03_catalog_executor`: ServiceCatalog, service API wrapper, five canonical v12 locale sources, service-page smoke. Preserve existing navigation and generated fallback changes. Apply impeccable before implementation and completion.
- `c03_service_owner_executor`: service-owned UpdateInstance DTO/Manager/handler and focused real-MySQL tests. Preserve C01 transaction readers, state transitions and deletion/reference locks. No Deploy implementation ownership.
- Coordinator: original packet/manifest/evidence/ledger, generated fallback timing, local preview and fixture identity, root test execution, final revision binding and independent review routing.

Read-only evidence: catalog requests fixed page 1/pageSize 100, disables table pagination and derives summaries from loaded rows; one Promise.all includes optional CMDB/K8s queries; full edit forms are absent. Owner UpdateInstance accepts versions/status but not target fields or expected lifecycle version. UI-only work cannot prove target-edit or stale-write conflict acceptance.

Minimum approach:

1. Use per-tab server pagination, search and totals; test objects beyond 101, not only a local array slice. Load optional parent/target options when the form needs them using bounded server queries.
2. Reuse existing PUT and shared form/modal/pagination patterns. Keep optional CMDB/K8s failures local; a Service read role must still load authorized catalog lists.
3. Select K8s cluster in the business scope, then bound namespace/workload; clear stale dependent choices. Reuse owner target readers, never query owner tables from another module.
4. Initial frontend patch keeps target/currentVersion read-only until the owner target/CAS seam exists. This intermediate behavior is not full C03 acceptance. Backend lane first proposes exact contract/lock order, adds red stale/target/active-reference tests, then implements after root verifies the red stage.
5. Reject stale editor updates via owner lifecycle CAS and reuse target/scope validation. Preserve observed currentVersion as an owner/runtime fact in the UI. Do not weaken active deployment/reference protection or add schema without evidence.

Verification: scoped lint/type-check; current UI smoke for pagination/edit/optional-target failure/service-read permissions; real MySQL CGO race for owner stale-write/target/reference invariants; final complete race, official v0.18.9 rebuild/overlay/drift and source binding. Desktop/mobile rendered evidence and meaningful interaction states are required; the historical preview binary does not establish current backend behavior.

Review: distinct code-reviewer and architect lanes on final scope. Historical failed runs and original artifacts are preserved. Shared ja/ko/fr fallback runtime wiring, external provider, production and final maintainer gates remain open; no archive or goal completion is implied.

Success: all original C03 acceptance criteria are demonstrated. Missing owner contract, permission/runtime/render evidence or final source binding keeps C03 in-progress.
