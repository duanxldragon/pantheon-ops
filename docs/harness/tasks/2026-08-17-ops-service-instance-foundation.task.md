# Task: Application, Service, And ServiceInstance Foundation

Task ID: `ops-service-instance-foundation`  
Priority: P0 model decision / P1 implementation  
Owner: new `business/service` bounded context  
Depends on: migrations, capability boundaries, BizScope ownership  
Coordinates with: Deploy reliability  
Blocks: VM lifecycle state machines and unified VM/K8s service identity

## Goal And Model Boundary

Add the minimum business-owned model between resource ownership and execution:

```text
BizScope -> Application -> Service -> ServiceInstance
CMDB Host -------------------------> VM instance target
K8s Cluster/Workload --------------> K8s instance target
Deploy/K8s Release ----------------> change history
```

This is not a generic CMDB extension. CMDB answers “what resource is this?”;
BizScope answers “who owns it?”; ServiceInstance answers “what service runs on
this resource?”; Deploy/K8s answer “what change was executed?”.

## Required Invariants

1. `Application` belongs to one active BizScope and has a stable unique code.
2. `Service` belongs to one Application and has a stable code within it.
3. `ServiceInstance` belongs to one Service and has exactly one typed runtime
   target: VM Host or K8s Workload identity.
4. Target references are validated through CMDB/K8s provider APIs; the service
   module never queries their tables or Kubernetes objects directly.
5. Deploy Task and K8s Release may reference the common service/instance identity,
   while preserving their different executors and histories.
6. Host `installed_components` stops being the source of service identity. Legacy
   data is read/migrated explicitly; no silent dual-write remains at completion.
7. Data scope is inherited from the authoritative BizScope/Application relation
   and enforced server-side.

## Why This Is Required

The current Host JSON cannot express service identity, desired/current version,
health, or target binding without lost updates. Adding those fields to CMDB would
make CMDB a runtime database. This is a bottom-model design gap, not a CRUD bug.

## Minimal Schema Contract

Names may follow repository conventions, but do not broaden the model:

- `biz_application`: ID, code, name, `business_scope_id`, optional display
  snapshot, `dept_id`, status, owner/remark, audit timestamps/actors, soft delete.
- `biz_service`: ID, `application_id`, code, name, runtime type/capability,
  description, status, audit timestamps/actors, soft delete.
- `biz_service_instance`: ID, `service_id`, environment, target type, nullable
  VM `host_id` or K8s `(cluster_id, namespace, workload_kind, workload_name)`,
  desired/current version fields, lifecycle version, timestamps/actors, soft delete.

Enforce exactly one target shape in Service validation and, where MySQL allows a
clear expression, a DB check. Use provider IDs as external references; do not add
cross-module GORM associations. Add unique constraints for application code,
service code within application, and one active service-instance target per
service according to the accepted cardinality.

Lifecycle state fields may be created here only with neutral defaults; legal
transitions and actions belong to task 6.

## In Scope

- New backend business module with Model/DTO/Service/Handler/Router, typed
  Reader/Command capability, migration, focused tests, and overlay registration.
- Minimal CRUD needed to create/list/get/disable Application, Service, and
  ServiceInstance, with delete protection. Avoid feature-complete catalog UI.
- BizScope provider validation and DataScope enforcement.
- CMDB/K8s target validation through typed APIs; no direct table or K8s client use.
- Add `service_id`/`service_instance_id` stable references and immutable display
  snapshots to Deploy/K8s history as coordinated migrations/API changes.
- Stop new Deploy success paths from writing service identity to
  `Host.installed_components`. Define a one-time migration/read-compatibility plan
  for existing JSON; delete dual-write after verification.
- Frontend: add compact Application/Service/Instance management or selectors only
  where required for the workflow; reuse base page/form/table and permission
  patterns. Register menu/component/permissions/i18n/audit via existing Ops overlay.

## Out Of Scope

- tenant/project/portfolio hierarchy;
- dynamic topology/CMDB universal-resource tables;
- start/stop/health/rollback behavior (task 6);
- merging VM Deploy and K8s Release executors;
- copying base IAM/audit/menu/i18n infrastructure;
- automatic discovery, service mesh, dependency graph, or configuration center;
- storing credentials or full Kubernetes objects.

## API Contract

External routes should follow existing `/business/...` style and DataScopedGroup.
At minimum expose stable refs/options for other modules and explicit commands for
target binding/state changes. Cross-module DTOs include IDs, codes/names, scope ID,
runtime type, target ref, and version only; no GORM model.

Deploy task creation must require `service_instance_id` for new VM lifecycle work.
For backward compatibility, legacy package-install tasks may be accepted only by
an explicit migration mode with a removal date; do not leave the field silently
optional forever. K8s Release linkage may be initially optional for unmanaged
workloads but must be explicit in DTO/status.

## Expected Files

- create `backend/modules/business/service/{model,dto,service,handler,module,capability}.go`
  and tests using repository naming conventions;
- modify `backend/modules/business/business.go`,
  `business_overlay_registry.go`, Deploy/K8s DTO/model/service integration points;
- add versioned migrations coordinated with task 1;
- modify permission/menu seeds, component registries, `frontend/src/modules/businessOverlay.ts`,
  API/pages/selectors, and zh-CN/en-US i18n only as required;
- update current design docs and add Service design/acceptance document.

Do not touch `../pantheon-base/**`, system/platform implementations, CMDB group/
label behavior, or unrelated K8s object pages.

## Verification Matrix

- duplicate application/service codes fail at Service and DB layers;
- inactive/unauthorized BizScope cannot create or view application/instance;
- VM instance target must be an authorized Host in the same scope;
- K8s target must resolve through K8s provider and match scope policy;
- mixed/missing target shape is rejected;
- deleting referenced application/service/instance is rejected;
- Deploy records stable service instance identity and retains historical snapshots;
- legacy `installed_components` migration is deterministic and no dual-write remains;
- permissions, menu, component registry, i18n, audit, frontend selector, and empty
  states are covered; UI claims require rendered evidence.

Commands:

```text
cd backend
go test -count=1 ./modules/business/service/... ./modules/business/deploy/... ./modules/business/k8s/...
go test -race ./modules/business/service/...
go vet ./...
cd ../frontend
npm run type-check
node scripts/business-overlay/check-business-overlay.mjs
```

Add a BizScope -> CMDB -> Service -> Deploy integration smoke and a K8s linkage
contract test. Evidence root: `.harness/evidence/ops-service-instance-foundation/`.

## Stop Conditions

Stop if the proposal expands beyond these three entities, requires a base/system
domain change, cannot define a deterministic legacy JSON migration, or introduces
cross-module GORM associations. Record the minimum safe alternative.
