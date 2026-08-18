# Task: Resource And Service Lifecycle State Machines

Task ID: `ops-resource-service-state-machines`  
Priority: P0 for VM lifecycle  
Owner: CMDB + Service lifecycle coordinator  
Depends on: Deploy reliability and ServiceInstance foundation  
Blocks: production start/stop/health/upgrade/rollback/retire workflows

## Goal

Separate resource lifecycle, resource connectivity, service desired state, service
observed state, and health. Implement explicit legal transitions and route all VM
lifecycle actions through Deploy plus ServiceInstance commands.

## State Contract

Freeze these semantics before implementation; names may adapt to repository style
but meanings cannot collapse into one field:

- Host lifecycle: `pending`, `assigned`, `maintenance`, `retired`.
- Host connectivity: `unknown`, `reachable`, `unreachable`, with observed time.
- Service desired state: `stopped`, `running`, `retired`.
- Service observed state: `unknown`, `installing`, `stopped`, `starting`,
  `running`, `stopping`, `upgrading`, `failed`, `retired`.
- Health: `unknown`, `healthy`, `degraded`, `unhealthy`, with check definition,
  time, message, and revision/correlation ID.

Existing Host `online/offline` values require an explicit migration mapping. The
recommended mapping is lifecycle `assigned` plus connectivity
`reachable/unreachable`; do not apply it without data sampling and a recorded
rollback/forward decision. Host state must never be derived from the number of
installed components.

## Required Invariants

1. CMDB owns Host lifecycle/connectivity. Service module owns desired/observed/
   health. Deploy can change service state only through ServiceInstance Command.
2. Every transition has allowed source states, actor/system source, preconditions,
   version/CAS, timestamp, correlation/task ID, and audit event.
3. Out-of-order or duplicate callbacks cannot overwrite newer state.
4. Start requires installed version and valid Host ownership/connectivity; stop,
   upgrade, rollback, uninstall, and retire each have explicit preconditions.
5. Upgrade is successful only after health criteria pass. Failure preserves the
   previous stable version and exposes a rollback target.
6. Host maintenance/retired blocks new executions; retiring a Host requires no
   active ServiceInstance or an explicit drain/retire workflow.
7. Desired state records intent; observed state records executor/resource fact.
   A timeout leaves a reconcilable discrepancy, not a fabricated success.

## Why This Is P0

Current Host status is a free string, Deploy writes component-derived Host state,
and there is no service state for start/stop/health/rollback. VM lifecycle cannot
close safely until state ownership and transitions are explicit. This is a base
model design issue, not an isolated handler bug.

## In Scope

- Add Host lifecycle/connectivity fields, transition service, conditional updates,
  migration, compatibility mapping, audit, and tests.
- Add ServiceInstance desired/observed/version/health fields and transition Command
  with optimistic version/CAS and stale-callback protection.
- Add VM lifecycle commands: install, start, health-check, stop, upgrade, rollback,
  uninstall, and retire. Each creates/reuses an immutable Deploy change record;
  it does not execute arbitrary shell commands directly in a handler.
- Define package/template requirements for lifecycle commands and health checks.
  Health definitions are versioned execution data, not ad hoc UI input at report
  time.
- Remove component-count-driven Host online/offline updates and all direct service
  state writes from Deploy. Use provider Commands.
- Add reconciliation query/command for stuck transitional states using task ID,
  lease, timeout, and latest observed result. A full distributed worker remains a
  later task unless already available.
- Update API/frontend to show resource lifecycle separately from service desired/
  observed/health; actions use permissions, confirmations, disabled reasons, and
  server-side validation.

## Out Of Scope

- K8s controller/release state implementation (task 7);
- generic workflow engines, event sourcing, or distributed sagas;
- automatically retiring Hosts during service uninstall;
- a full monitoring platform or continuous probe scheduler;
- copying base audit/permission UI infrastructure;
- treating Kubernetes Workloads as CMDB rows.

## Transition Matrix Minimum

| Action          | Required source/precondition              | Intermediate | Success                                  | Failure                         |
| --------------- | ----------------------------------------- | ------------ | ---------------------------------------- | ------------------------------- |
| assign Host     | pending + no owner                        | assigned     | assigned                                 | unchanged/conflict              |
| maintenance     | assigned + no conflicting lease           | maintenance  | maintenance                              | unchanged                       |
| retire Host     | pending/maintenance + no active instances | retired      | retired                                  | reference conflict              |
| install         | instance stopped/unknown + reachable Host | installing   | stopped + version                        | failed                          |
| start           | stopped + installed version               | starting     | running + health pending                 | failed                          |
| health          | running                                   | running      | healthy/degraded                         | unhealthy/timeout               |
| stop            | running/failed                            | stopping     | stopped                                  | failed                          |
| upgrade         | running/stopped + rollback version known  | upgrading    | desired=current new version after health | failed + prior version retained |
| rollback        | failed/running + prior version            | upgrading    | prior version + target desired state     | failed                          |
| retire instance | stopped/uninstalled + no active lease     | retired      | retired                                  | conflict                        |

The implementation must publish the complete matrix in design docs and tests;
the table above is the minimum, not permission for arbitrary missing transitions.

## Expected Files

- `backend/modules/business/cmdb/host/host_model.go`, `host_dto.go`,
  `host_service.go`, `host_handler.go`, tests;
- ServiceInstance model/DTO/state service/capability created by task 5;
- `backend/modules/business/deploy/deploy_model.go`, `deploy_dto.go`,
  `deploy_service.go`, handler/permissions/tests only for lifecycle orchestration;
- coordinated migration SQL, service/CMDB/deploy design docs;
- frontend CMDB Host and ServiceInstance/Deploy pages, API, permissions, i18n;
- integration/smoke/visual evidence.

Do not touch `../pantheon-base/**`, K8s release code, system/platform logic, or
unrelated CMDB group/label pages.

## Verification Matrix

- every illegal transition returns a stable error and leaves data unchanged;
- two concurrent transitions with one version allow exactly one commit;
- duplicate callback is idempotent; stale callback cannot overwrite newer state;
- Host maintenance/retired rejects new Deploy lease;
- install/start/health/stop/upgrade/rollback/retire happy and failure paths;
- failed health prevents upgrade success and retains rollback target;
- service state changes never mutate Host lifecycle automatically;
- restart/reconciliation resolves or clearly flags stuck transitional states;
- permission and Business Scope DataScope deny unauthorized actions;
- UI separates lifecycle/connectivity/desired/observed/health with confirmations
  and no overlap; rendered desktop/mobile evidence is required.

Commands:

```text
cd backend
go test -count=1 ./modules/business/cmdb/... ./modules/business/service/... ./modules/business/deploy/...
go test -race ./modules/business/cmdb/... ./modules/business/service/... ./modules/business/deploy/...
go vet ./...
cd ../frontend
npm run type-check
npm run test:smoke:business:cmdb
npm run test:smoke:business:deploy:api
npm run test:smoke:business:deploy
```

Evidence root: `.harness/evidence/ops-resource-service-state-machines/`.

## Stop Conditions

Stop if legacy Host status cannot be mapped safely, lifecycle semantics conflict
with an already published API, a new base workflow engine is required, or task 4/5
contracts are not frozen. Record state mapping alternatives and data impact.
