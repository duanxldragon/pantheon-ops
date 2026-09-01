# Harness Method Playbook

Chinese version: [HARNESS_METHOD_PLAYBOOK.md](./HARNESS_METHOD_PLAYBOOK.md)

Type: Playbook
Layer: platform
Status: Active

This file no longer carries the full method definition.

## Current Relationship

- `pantheon-harness/`: the method source of truth, responsible for method orchestration, templates, schema, and portable checks
- `docs/harness/*`: the repo-local contract and landing layer for this repository
- `scripts/harness/*`: the mechanical gates for this repository

## Reading Order

1. First read `../../../pantheon-harness/patterns/README.md`
2. Then read `../../../pantheon-harness/patterns/harness-core-model.md`
3. Then read `../../../pantheon-harness/patterns/harness-coverage-model.md`
4. Then read `../../../pantheon-harness/patterns/harness-template-taxonomy.md`
5. Then read `../../../pantheon-harness/patterns/tool-adapter-matrix.md`
6. Then read `../../../pantheon-harness/patterns/method-playbook.md`
7. Then read `../../../pantheon-harness/patterns/execution-guardrails.md`
8. Then read `../../../pantheon-harness/patterns/context-engineering-protocol.md`
9. Then read `../../../pantheon-harness/patterns/method-first-delivery-policy.md`
10. Then read `../../../pantheon-harness/patterns/cross-agent-ratchet-model.md`
11. Then read `../../../pantheon-harness/patterns/minimal-complexity-ladder.md`
12. Then read `../../../pantheon-harness/patterns/templates/task-packet.template.md`
13. Then read `../../../pantheon-harness/patterns/templates/review.template.md`
14. Then read `docs/harness/PANTHEON_BASE_DELIVERY_WORKFLOW.md`
15. Then read `docs/harness/AI_QUALITY_GOVERNANCE.md`
16. Then read the contracts this repository executes locally:
    - `HARNESS_ENGINEERING_CONTRACT.md`
    - `TRIVIALITY_CLASSIFICATION_POLICY.en.md`
    - `TASK_PACKET_SPEC.md`
    - `VERIFICATION_EVIDENCE_SPEC.md`
    - `REVIEW_LOOP_SPEC.md`
    - `VISUAL_QUALITY_PROTOCOL.md`
    - `VISUAL_EVIDENCE_PROMOTION_POLICY.en.md`
    - `FAILURE_RATCHET_POLICY.en.md`
    - `FAILURE_REGISTRY_PROMOTION_POLICY.en.md`
    - `HARNESS_RETIREMENT_REVIEW.en.md`

## Default Execution Guardrails

Before implementation, run through `../../../pantheon-harness/patterns/context-engineering-protocol.md` by default:

- separate confirmed facts, working assumptions, and open questions instead of guessing through permission, audit, or boundary ambiguity
- walk the minimal complexity ladder and keep the approach at the smallest load-bearing rung
- decide entry sources, retrieval order, and sensitive-context boundaries before implementation
- for long-running, cross-session, delegated, or cost-sensitive work, also declare `Response Budget`, `Retrieval Helpers`, `Promotion Target`, and `Economics Watch`
- declare `Do Not Touch` and `Structural Scope` before making surgical edits
- define `Success Criteria`, `Verification Plan`, and evidence linkage before declaring the task done

## Responsibilities In This Repository

This repository keeps the following because they directly serve local execution:

- `docs/harness/*`: contracts, formats, governance rules
- `scripts/harness/*`: validators and CI gates
- added generic gates: review, template health, runtime evidence, docs links, and sync drift
- `.agents/*` / `.codex/skills/*`: tool adaptation layers

If the method layer conflicts with the repository landing layer, the method definition in `pantheon-harness/` wins, and the repository landing layer should then be synchronized.
