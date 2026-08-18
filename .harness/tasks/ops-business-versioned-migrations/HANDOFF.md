# Task Handoff: Business Migrations

Read parent `.harness/tasks/2026-08-17-ops-foundation-hardening/HANDOFF.md` first,
then this task's `manifest.json`, `STATUS.md`, and task packet. Work from
`pantheon-ops/backend` for Go commands. This is a single-owner high-risk task.

## Start

1. Record `git status --short`, current migration head, and database version in
   `STATUS.md`.
2. Inspect every current business `Migrate()` call and determine whether it is
   test-only or production reachable.
3. Reserve a migration number only after checking all existing and dirty files.
4. Write the schema plan and rollback semantics into `STATUS.md` before editing.

## Resume

Read `STATUS.md` and evidence first. `Next atomic action` is authoritative. Re-run
only the failed migration scenario before changing SQL. Never reset another
executor's migration files.

## Stop

Stop for base-owned runner changes, destructive conversion, unavailable rollback,
or a migration-number collision. Record the exact decision required.
