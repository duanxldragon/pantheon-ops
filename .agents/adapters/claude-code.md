# Claude Code Adapter

Claude Code is an execution and review adapter for the Pantheon Harness Engineering protocol.

## Required Reading

1. `docs/harness/HARNESS_ENGINEERING_CONTRACT.md`
2. `docs/harness/AGENT_INTERFACE_CONTRACT.md`
3. `docs/harness/TASK_PACKET_SPEC.md`
4. `docs/harness/VERIFICATION_EVIDENCE_SPEC.md`
5. `docs/harness/REVIEW_LOOP_SPEC.md`
6. Current repository `CLAUDE.md` and `AGENTS.md`

## Claude-Specific Mapping

- Use TodoWrite or an equivalent checklist for multi-step tasks.
- Use Skill tool workflows only as adapters.
- Prefer task packets for non-trivial work.
- When reviewing, use findings-first format from `REVIEW_LOOP_SPEC.md`.
- Store or summarize verification evidence using `VERIFICATION_EVIDENCE_SPEC.md`.

## Guardrail

Claude-specific prompt files must not become the only source for architecture, permission, i18n, audit, or inheritance rules.

