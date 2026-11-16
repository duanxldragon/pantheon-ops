# Status: ServiceInstance Foundation

- Status: `complete`
- Updated: `2026-08-18`
- Executor: Codex `/root`
- Baseline commit: `ee213508a32ebdf27685128e0c719e9036272de5`
- Dirty-file collisions: existing dirty backend business modules, overlay
  registries, migrations, frontend business modules, task/evidence artifacts,
  and untracked K8s implementation; no existing `business/service` files.
  Preserve all unrelated changes in place.

## Completed

- Minimum three-entity bounded context implemented and registered.
- Application, Service, and ServiceInstance schema/API/DTO/handler paths are
  present under `/business/service`.
- VM and K8s target shape validation uses typed CMDB/K8s provider APIs.
- Deploy/K8s stable service references and overlay/menu/i18n contracts are wired.
- Duplicate-code, mixed-target, target-scope, and target-cardinality tests pass.
- Frontend menu, i18n, type-check, build, and business-boundary gates pass.
- MinGW CGO race test passes for the service package.

## Decisions

- Service identity is business-owned, not CMDB or pantheon-base owned.
- VM and K8s share identity, not an executor or copied runtime objects.
- Minimal model is exactly `Application -> Service -> ServiceInstance`.
- Application owns `business_scope_id` and `dept_id`; Service inherits scope
  through Application; ServiceInstance inherits scope through Service.
- A ServiceInstance has exactly one target shape: VM `host_id`, or K8s
  `(cluster_id, namespace, workload_kind, workload_name)`. The accepted
  cardinality is one active target per instance and one active instance per
  `(service_id, target identity)` tuple.
- Existing `installed_components` JSON remains read-compatible for legacy
  inspection only. New Service/Deploy paths do not dual-write it; a
  deterministic future migration must report entries and must not guess a
  Service/Instance identity.

## Evidence

- `.harness/evidence/ops-service-instance-foundation/commands.json`
- `.harness/evidence/ops-service-instance-foundation/summary.md`
- `.harness/evidence/ops-service-instance-foundation/review.md`

## Next Atomic Action

Start `ops-resource-service-state-machines` with the Host/service state matrix
already frozen in its task packet.

## Blockers

- None for this child task. Production model-to-migration upgrade scripting
  remains explicitly deferred by maintainer instruction.
