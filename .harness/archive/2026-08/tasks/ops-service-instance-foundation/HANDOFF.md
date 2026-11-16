# Task Handoff: ServiceInstance Foundation

Read the parent handoff, the three dependencies' completed summaries, then this
task packet/manifest/status. This is a single-owner model task; do not begin while
dependency contracts are still changing.

## Start

1. Record final BizScope, CMDB target, Deploy snapshot, and K8s target contracts.
2. Freeze the three-entity schema and target cardinality in `STATUS.md`.
3. Decide and test the legacy `installed_components` migration/compatibility path.
4. Reserve migration numbers and register the backend module before UI work.

## Resume

Read schema/API decisions and evidence before code. Continue only from `Next
atomic action`; do not expand the model because a future feature is imaginable.

## Stop

Stop for more entities, base/system changes, non-deterministic legacy migration,
cross-module GORM associations, or an incompatible Deploy/K8s public contract.
