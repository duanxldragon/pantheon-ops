# Task Packets

Current program: [2026-11-16-ops-pilot-delivery.task.md](./2026-11-16-ops-pilot-delivery.task.md).

- [Canonical plan](../../../.harness/tasks/2026-11-16-ops-pilot-delivery/plan.md)
- [Execution queue](../../../.harness/tasks/2026-11-16-ops-pilot-delivery/EXECUTION_QUEUE.md)
- [Machine state](../../../.harness/tasks/2026-11-16-ops-pilot-delivery/manifest.json)
- [Current status](../../../.harness/STATUS.md)
- [Archive index](../../../.harness/ARCHIVE.md)

Create non-trivial business and Base-consumption task packets here using [the Ops template](../../TASK_PACKET_OPS_TEMPLATE.md) and [inheritance guide](../../PROJECT_INHERITANCE.md). Select a dependency-ready child, preserve other owners' edits, and update its manifest and evidence with actual results. The parent remains planned until implementation starts; planning completion does not complete delivery.

The August foundation-hardening program and eight children are archived within their original scope. Keep task documents at stable addresses; archive task state and evidence together under .harness/archive/YYYY-MM/. The archive-aware resolver preserves canonical task/evidence references. Do not archive unresolved runtime acceptance, missing-summary logs, or tasks whose completion fields conflict with pending evidence.

For memory-free handoff, use [.harness/templates/STATELESS_TOOL_HANDOFF_TEMPLATE.md](../../../.harness/templates/STATELESS_TOOL_HANDOFF_TEMPLATE.md). Canonical state belongs in the repository; .omx/plans/ is only a local launch pointer.
