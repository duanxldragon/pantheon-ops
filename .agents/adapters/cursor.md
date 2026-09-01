# Cursor Adapter

Cursor is an implementation adapter for the Pantheon Harness Engineering protocol.

## Required Reading

1. `docs/harness/HARNESS_ENGINEERING_CONTRACT.md`
2. `docs/harness/AGENT_INTERFACE_CONTRACT.md`
3. `docs/harness/TASK_PACKET_SPEC.md`
4. Current repository `AGENTS.md`

## Cursor-Specific Mapping

- Cursor rules should point to `docs/harness/*`; they should not duplicate the full protocol.
- Composer tasks should start from a task packet for non-trivial work.
- Before broad edits, inspect current diff and avoid user-owned changes.
- Verification commands must be run in the terminal or explicitly marked not run.

## Review

Cursor can implement or review, but review output must follow `docs/harness/REVIEW_LOOP_SPEC.md`.

