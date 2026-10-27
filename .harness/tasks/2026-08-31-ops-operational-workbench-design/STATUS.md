# Program Status

- Status: `planned`
- Updated: `2026-08-31`
- Owner: unassigned
- Design readiness: complete
- Implementation: not started
- Foundation consumption: not started
- Runtime evidence: not applicable to this docs-only design turn
- Visual evidence: existing Ops screenshots are baseline only; target workbenches are not rendered

## Child Status

| Packet | Priority | Status | Base Dependency | Outcome |
| --- | --- | --- | --- | --- |
| O1 | P1 | blocked | B3 ContextSelector | unified context and target selector |
| O2 | P0 | blocked | B1 + B3 | deployment execution workbench |
| O3 | P0 | blocked | B3 ChangeDiff | pre-execution change preview |
| O4 | P0 | blocked | B2 + B3 | observability query workbench |
| O5 | P1 | blocked | B3 TaskLogViewer | Deploy and Pod log experience |
| O6 | P1 | blocked | B4 | business Dashboard widgets |
| O7 | P2 | deferred | relationship maturity gate | CMDB relationship/impact discovery |

## Decisions

- Base owns generic operational primitives; Ops owns business adapters and workflows.
- Target workbenches follow context -> risk preview -> execution -> failure -> recovery.
- O7 stays deferred until the relationship model has decision-grade semantics.
- No production code or external system is changed by this program setup.

## Next Atomic Action

After the required Base foundation release is locked, assign O1 or O4 and record current dirty-file collisions plus a focused behavioral baseline.

## Blockers

- O1-O6: required Base contracts are design-only and not yet released.
- O7: CMDB relationship maturity gate is not satisfied.
