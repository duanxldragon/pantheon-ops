# Ops Operational Workbench Execution Queue

## Dependency Map

Reconciled 2026-11-07 against consumed Base v0.18.6: B3 primitives and table preferences are present. Ready means implementable, not implemented or accepted. O6 still requires a released Dashboard render/data contract.

| Packet | Priority | Required Base           | Business Dependencies                                                                     | State                                                                                                                                                                                                                         |
| ------ | -------- | ----------------------- | ----------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| O1     | P1       | B3 `ContextSelector`    | BizScope + CMDB typed capabilities                                                        | in-progress; Deploy resolve/create/update/UI/runtime verified; other consumers and full acceptance open                                                                                                                       |
| O2     | P0       | B1 + B3 step/log        | Deploy immutable task/events                                                              | in-progress; viewed-version controls and24/24 local runtime verified; full SSH/attempt/action and Base consumption open                                                                                                       |
| O3     | P0       | B3 `ChangeDiff`         | Deploy/K8s/Argo CD intent versions                                                        | in-progress; Deploy signed preview and frozen configuration implemented; single-package frozen rollback verified; K8s image review scoped tests pass; K8s manifest/config/resource, Argo/complex rollback and acceptance open |
| O4     | P0       | B2 + B3 condition/table | datasource query APIs                                                                     | in-progress; Loki returned-row modes locally verified; history/ConditionBuilder/multi-source/real acceptance open                                                                                                             |
| O5     | P1       | B3 `TaskLogViewer`      | Deploy SSE + K8s log API                                                                  | in-progress; scoped checks pass, acceptance open                                                                                                                                                                              |
| O6     | P1       | B4 widget kinds         | business summary endpoints; Base renderer implemented; formal release/consumption pending | blocked                                                                                                                                                                                                                       |
| O7     | P2       | none initially          | mature CMDB relationship model                                                            | deferred                                                                                                                                                                                                                      |

## Packets

- [O1 Unified Context And Target Selector](./packets/O1-context-and-target-selector.md)
- [O2 Deployment Execution Workbench](./packets/O2-deployment-execution-workbench.md)
- [O3 Change Preview And Risk](./packets/O3-change-preview-and-risk.md)
- [O4 Observability Query Workbench](./packets/O4-observability-query-workbench.md)
- [O5 Task And Pod Log Experience](./packets/O5-task-and-pod-log-experience.md)
- [O6 Business Dashboard Widgets](./packets/O6-business-dashboard-widgets.md)
- [O7 CMDB Relationship Impact Discovery](./packets/O7-cmdb-relationship-impact.md)

## Suggested Batches

1. Base release/lock gate, then O1 and O4 in parallel if file ownership does not overlap.
2. O2 and O5 after log/step adapters and Deploy event contracts are frozen.
3. O3 after immutable intent/version contracts are frozen; O6 after widget budget endpoints exist.
4. O7 only after its maturity gate passes; it is not required for O1-O6 closeout.

## Shared Acceptance Matrix

All packets cover permission, DataScope, audit, i18n, loading/empty/error/forbidden/conflict/stale/partial failure as applicable, 1440x900, 390x844, light/dark, long Chinese/English, keyboard/focus, 200% zoom and reduced motion. External actions require real evidence or an explicit endpoint/credential/production gap.

O2/O4 continuation 2026-11-08: actual ephemeral loopback SSH protocol and isolated MySQL credential/frozen-script/step/log/writeback evidence verify the local SSH path and context cancellation repairs; this is not real VM acceptance. O4 adds six-label/eighty-character previews with explicit truncation notices and retained bounded raw detail. Current validation/review/cleanup receipts are in the linked closeout summary. Both parent packets and all release/external/final gates remain open.
