# Codex Adapter

Codex is an execution adapter for the repository Harness Engineering protocol.

## Required Reading

1. `docs/harness/HARNESS_ENGINEERING_CONTRACT.md`
2. `docs/harness/AGENT_INTERFACE_CONTRACT.md`
3. `docs/harness/TASK_PACKET_SPEC.md`
4. `docs/harness/VERIFICATION_EVIDENCE_SPEC.md`
5. `docs/harness/REVIEW_LOOP_SPEC.md`
6. Current repository `AGENTS.md`

## Codex-Specific Mapping

- Use `.codex/skills/*` only as helper workflows.
- Prefer repository-local workflow skills in `.agents/skills/*` when the task matches Pantheon Base verification, PR gating, or CI triage.
- If a skill conflicts with `docs/harness/*` or repository contracts, follow the repository contracts.
- Use shell commands for verification and report command results.
- Use `apply_patch` for manual file edits.
- Respect sandbox approval rules.
- Do not treat chat history as verification evidence.

## Output Requirements

Final responses must include:

- changed files
- commands run
- verification result
- evidence path or summary
- known gaps
