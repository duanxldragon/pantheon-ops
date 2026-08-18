# State Machines Evidence

## Delivered

- Host lifecycle and connectivity are separate owner fields with explicit
  compatibility projection for legacy `pending`, `assigned`, `online`,
  `offline`, and `maintenance` values.
- Host transitions use conditional `state_version` updates. Host retirement
  checks active ServiceInstance references through a typed Service-owned Reader.
- ServiceInstance owns desired, observed, health, version, rollback target,
  correlation, timestamp, and lifecycle CAS state.
- Duplicate callbacks with the same correlation and projection are idempotent;
  stale lifecycle versions and illegal transitions are rejected.
- Upgrade and rollback completion requires a healthy result before observed
  `running` is accepted. Failed transitions retain the prior stable version and
  rollback target.
- Deploy is wired to ServiceInstance only through the typed capability contract.
  It records begin and terminal callback transitions for install, start, stop,
  health, upgrade, rollback, uninstall, and retire actions.
- Stuck transitional instances expose a reconciliation command that marks the
  observed state failed and health degraded instead of fabricating success.
- Service UI now shows desired/observed/health/version columns and exposes a
  reconciliation action for stale instances.

## Verification

The command results are recorded in `commands.json`. Focused tests include
illegal transitions, CAS stale rejection, healthy upgrade gating, timeout
reconciliation, active-host retirement references, and Deploy typed callback
routing.

## Explicit Gap

No production upgrade SQL was added for the new model fields because the
maintainer deferred model-generated migration work to the production
deployment phase. The frontend type-check and build passed; a rendered
desktop/mobile screenshot was not generated in this environment and remains a
visual QA follow-up.
