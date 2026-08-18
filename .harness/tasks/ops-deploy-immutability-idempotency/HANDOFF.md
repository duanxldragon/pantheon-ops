# Task Handoff: Deploy Reliability

Read the parent handoff, task packet, manifest, and status. This is a high-risk
single-owner task. Do not split SQL/state transition edits across uncoordinated
tools.

## Start

1. Re-run the package mutability, Start, TaskHost, visibility, and CMDB writeback
   searches from the task packet.
2. Freeze snapshot, idempotency, callback, and lease contracts in `STATUS.md`.
3. Confirm migration number ownership with the migration task.
4. Implement provider API calls before deleting direct access; keep the code
   buildable after each bounded caller migration.

## Resume

Read the latest replay/concurrency evidence and `Next atomic action`. Reproduce
the failing race before changing state logic. Never broaden an update predicate
to make a test pass.

## Stop

Stop for distributed locking, secret storage, base executor changes, destructive
history rewrites, or an external API break.
