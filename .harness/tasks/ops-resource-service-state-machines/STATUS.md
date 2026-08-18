# Status: State Machines

- Status: `complete`
- Updated: `2026-08-18`
- Executor: Codex `/root`
- Baseline commit: `ee213508a32ebdf27685128e0c719e9036272de5`
- Dirty-file collisions: existing dirty CMDB host/capability, Deploy,
  ServiceInstance, K8s, frontend business modules, migrations, registries,
  task/evidence artifacts, and newly added service design/evidence files.
  Preserve all unrelated work; no reset/restore/revert. The generated-module
  cleanup incident was recovered and the cleanup script now protects manifest
  declared overlay paths even when the report is stale.

## Completed

- Minimum Host and ServiceInstance state semantics and VM action matrix recorded.
- Dependency preflight revalidated: Deploy and ServiceInstance child manifests
  are `complete`, with evidence directories present.
- Legacy Host status values sampled from current model/import/test contracts:
  `pending`, `assigned`, `online`, `offline`, and `maintenance`.

## Decisions

- Resource lifecycle/connectivity and service desired/observed/health are separate.
- Component count never drives Host lifecycle.
- Compatibility mapping is explicit and non-destructive:
  `pending -> lifecycle pending + connectivity unknown`;
  `assigned -> lifecycle assigned + connectivity unknown`;
  `online -> lifecycle assigned + connectivity reachable`;
  `offline -> lifecycle assigned + connectivity unreachable`;
  `maintenance -> lifecycle maintenance + connectivity unknown`.
- Existing `status` remains a compatibility projection until the later
  model-generated production migration; new state fields must not be inferred
  from installed component count.

## Evidence

Implementation, transition, race, and review evidence is present in
`.harness/evidence/ops-resource-service-state-machines/`.

## Next Atomic Action

No remaining implementation action in this child. Production legacy-table
upgrade SQL remains deferred to the deployment migration pass.

## Blockers

- Production legacy-table upgrade migration remains deferred by maintainer
  instruction; only model and forward-compatible migration artifacts belong in
  this child.
