# Task Handoff: BizScope Ownership

Read the parent handoff, then this task's task packet, manifest, and status. This
task handles authorization and concurrency; do not let a lower-cost executor
“simplify” it into a last-writer-wins update.

## Start

1. Revalidate base DataScope behavior and current dirty changes.
2. Freeze the minimal scope-owner mapping (recommended `dept_id`) in `STATUS.md`.
3. Freeze Reader/Command signatures from the capability task before implementation.
4. Write the bind/delete race proof and expected error keys before SQL changes.

## Resume

Read the last concurrency evidence and `Next atomic action`. Re-run the failing
MySQL scenario first. Preserve unrelated frontend/backend edits.

## Stop

Stop for base IAM changes, distributed coordination, destructive legacy mapping,
or a request to bypass CMDB ownership commands.
