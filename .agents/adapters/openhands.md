# OpenHands Adapter

OpenHands is an execution adapter for the Pantheon Harness Engineering protocol.

## Required Reading

1. `docs/harness/HARNESS_ENGINEERING_CONTRACT.md`
2. `docs/harness/AGENT_INTERFACE_CONTRACT.md`
3. `docs/harness/TASK_PACKET_SPEC.md`
4. `docs/harness/VERIFICATION_EVIDENCE_SPEC.md`
5. Current repository `AGENTS.md`

## OpenHands-Specific Mapping

- Use task packets as issue-level instructions for non-trivial work.
- Run verification commands inside the workspace when possible.
- Persist command summaries and artifacts using the evidence spec.
- Do not bypass human gates for deletion, schema, base contract, permission, i18n, audit, or inheritance changes.

