# Stateless Program Handoff

This file is the entry point for any tool or model with no prior conversation
memory. Do not begin by editing code.

## Start Protocol

1. Set the repository root to `pantheon-ops` and read `AGENTS.md` plus
   `docs/PROJECT_INHERITANCE.md`.
2. Read `docs/business-module-review-summary.md`, then the parent task packet and
   this directory's `manifest.json`, `STATUS.md`, and `EXECUTION_QUEUE.md`.
3. Run `git status --short`. Treat every existing change as owned by someone else
   until proven otherwise; never reset, restore, or rewrite it.
4. Choose only a child whose dependencies are `complete` in its manifest/status.
5. Read that child's task packet, `manifest.json`, `HANDOFF.md`, and `STATUS.md`.
6. Revalidate every cited file/method against current code. Audit references are a
   baseline, not permission to overwrite newer work.
7. Before implementation, update the child `STATUS.md` with executor, timestamp,
   baseline commit, observed dirty-file collisions, and the first concrete step.

## Resume Protocol

On every new tool/model session:

1. Read only the five files above plus the selected task's files before exploring.
2. Inspect the selected task's evidence directory and recent diff.
3. Re-run the smallest verification that establishes the current failure/success
   baseline; record the result in `STATUS.md`.
4. Continue from `Next atomic action`; do not redo items recorded with evidence.
5. After each atomic change, update `Completed`, `Decisions`, `Evidence`, and
   `Next atomic action`. Never rely on chat history.

## Status Vocabulary

- `planned`: untouched task packet.
- `in_progress`: one named owner is changing implementation.
- `blocked`: maintainer decision is required and the exact question is recorded.
- `verification`: implementation frozen; only tests/review/evidence remain.
- `complete`: all acceptance criteria and evidence are present.
- `deferred`: maintainer explicitly moved the task out of this program.

Only one implementation owner may set a high-risk child to `in_progress`.

## Parallel Work Rules

- Safe early parallel pair: migrations and capability contract design.
- Safe second batch after contracts: BizScope ownership, Deploy reliability, and
  K8s reliability, provided file ownership does not overlap.
- Single-owner tasks: ServiceInstance foundation and state machines.
- Lower-cost tools may own isolated tests, quality scripts, documentation, and
  frontend type fixes after contracts are frozen.
- No tool may create cross-module table access as an interim shortcut.

Use `EXECUTION_QUEUE.md` as the source of truth for batch order, owner profile,
cost allocation, and per-child minimum evidence. If it conflicts with a child
manifest, stop and update the parent `STATUS.md` with the exact conflict.

## Required Evidence Per Child

- `commands.json`: command, workdir, exit code, timestamp, and output artifact.
- `summary.md`: outcome, changed contracts/schema, validation, and residual risk.
- `review.md`: findings-first self-review and resolution.
- migration tasks: empty/upgrade/repeat/failure-recovery evidence.
- UI tasks: desktop/mobile rendered evidence or explicit runtime gap.

## Stop Conditions

Stop without speculative implementation when a maintainer gate in the parent or
child manifest is reached. Record one precise question, the options, impact, and
recommended choice in `STATUS.md`. Do not ask for routine implementation choices.

## Program Closeout

The final owner verifies all child statuses and dependency links, runs the parent
verification set, records unresolved gaps, and writes the parent evidence summary.
No completion claim is valid from task checkboxes alone.
