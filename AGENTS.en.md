# Pantheon Base Agent Rules

Chinese version: [AGENTS.md](./AGENTS.md)

This is the English quick guide for the Pantheon Base agent rules.

Core expectations:

- treat the repository contracts and canonical docs as the source of truth
- classify every task by ownership layer before editing
- keep `platform`, `system/*`, and `business/*` boundaries explicit
- respect harness protocol before non-trivial work
- do not collapse auth, IAM, org, and config into a single mixed module
- preserve key-first i18n, permission separation, audit discipline, and modular registration

Before substantial work, read in order:

1. `../pantheon-harness/patterns/README.md`
2. `../pantheon-harness/patterns/method-playbook.md`
3. `../pantheon-harness/patterns/execution-guardrails.md`
4. `../pantheon-harness/patterns/context-engineering-protocol.md`
5. `DESIGN.md`
6. `AGENTS.md`
7. `docs/README.md`
8. `docs/harness/HARNESS_METHOD_PLAYBOOK.md`
9. `docs/harness/PANTHEON_BASE_DELIVERY_WORKFLOW.md`
10. `docs/harness/AI_QUALITY_GOVERNANCE.md`
11. `docs/acceptances/AGENT_EXECUTION_CHECKLIST.md`
12. matching contracts
13. matching designs
14. matching acceptances

Additional defaults:

- for non-trivial work, state the implementer posture, reviewer posture, minimum evidence set, and human gates up front
- before implementation, apply the minimal complexity ladder: skip if unnecessary, reuse existing helpers/components/scripts/contracts, prefer standard library, native platform features, or installed dependencies, and only then write the minimum new code
- multi-agent work is coordinated by the active orchestrating agent, but the method is role-based: Planner/Dispatcher plans and routes review, Generator explores/implements/fixes, Reviewer/Evaluator judges evidence; the human owns goals, scope, risk acceptance, and key gates, not manual tool handoff
- runtime-sensitive changes such as auth, permissions, menu routing, import/export, lowcode, dynamic modules, async chains, and external integrations need runtime evidence or an explicit runtime gap in addition to tests
- when the same failure pattern repeats, ratchet it from closeout note to repo rule and then to checker, smoke path, or failure-registry entry

Use the Chinese source document for the full repository rules, task-order constraints, and architectural red lines.
