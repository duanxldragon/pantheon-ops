# Task Packets

Create task packets here for non-trivial business work and `base -> ops` sync work:

```text
docs/harness/tasks/YYYY-MM-DD-<task-name>.task.md
```

Use:

- `docs/TASK_PACKET_OPS_TEMPLATE.md`
- `../PROJECT_INHERITANCE.md`
- `.harness/templates/STATELESS_TOOL_HANDOFF_TEMPLATE.md` when a task must be
  resumed by a new chat, a different tool, or a lower-cost model without shared
  memory.

Reusable one-line handoff request:

```text
按 .harness/templates/STATELESS_TOOL_HANDOFF_TEMPLATE.md 为 <task/program> 生成无记忆工具交接包。
```

## Active Foundation Program

- Program task: `2026-08-17-ops-foundation-hardening.task.md`
- Stateless entry: `.harness/tasks/2026-08-17-ops-foundation-hardening/HANDOFF.md`
- Execution queue: `.harness/tasks/2026-08-17-ops-foundation-hardening/EXECUTION_QUEUE.md`
- Machine-readable plan: `.harness/tasks/2026-08-17-ops-foundation-hardening/manifest.json`
- Execution state: `.harness/tasks/2026-08-17-ops-foundation-hardening/STATUS.md`

Any tool without prior conversation memory must start from the stateless entry,
select only a dependency-ready child task from the execution queue, and update
that task's `STATUS.md` and evidence directory during execution.

Retention:

- Keep task packets while their evidence, review, remediation, or follow-up work is still active.
- If a task packet is fully superseded by a stable design, acceptance document, or release note, either link that successor from the task packet or delete the task packet during docs cleanup.
