# Findings-First Review

## Findings

1. **Medium: rendered visual evidence is still missing.** The Service page
   passed type-check and production build, but this run did not capture
   desktop/mobile screenshots or exercise the reconciliation control in a
   browser. Run the existing Playwright visual/smoke harness before release.
2. **Medium: production schema upgrade remains deferred.** `rollback_version`
   and state fields rely on the later model-generated migration pass, per
   maintainer instruction. Do not deploy this model change against an existing
   production schema until that migration is generated and applied.

## Reviewed Controls

- Cross-module Deploy writes use the ServiceInstance typed Command.
- Host retirement uses the Service-owned reference Reader rather than a direct
  ServiceInstance table query.
- No Host lifecycle transition is inferred from installed component count.
- State writes use lifecycle/state CAS and correlation-based duplicate handling.
- Upgrade/rollback running projection is health-gated.

## Residual Risk

The current Deploy executor still maintains the legacy CMDB installed-component
projection for inventory compatibility. It is not used as ServiceInstance
identity or lifecycle state, but a later cleanup should remove any remaining
component-derived Host status projection after consumers migrate.
