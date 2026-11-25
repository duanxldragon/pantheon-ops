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

Start with `AGENTS.md`, `.harness/STATUS.md` and `.harness/TASK_INDEX.md`; use `docs/README.md` to select only the matching contract/design/acceptance. Read relevant DESIGN sections for architecture/UI decisions and delivery governance for L2. Progress belongs in the target task summary; manifests own task status. Product VERSION, official publication and dated checkpoints are separate facts.

Before execution, consume [capability-adaptive delivery](../pantheon-harness/patterns/capability-adaptive-delivery.md) and verify `.harness/method-consumption.json` with `node ../pantheon-harness/scripts/harness/check-method-consumption.mjs --root . --source ../pantheon-harness`. The primary agent may plan, implement, verify, and fix in one session; planning-only/review-only assignments stay read-only. Keep current model/settings, risk tiers, authority, independent high-risk review or an explicit review gap, and evidence. Update the receipt only after reviewing an intentional source change.

Additional defaults:

- for non-trivial work, state the implementer posture, reviewer posture, minimum evidence set, and human gates up front
- before implementation, apply the minimal complexity ladder: skip if unnecessary, reuse existing helpers/components/scripts/contracts, prefer standard library, native platform features, or installed dependencies, and only then write the minimum new code
- multi-agent work is coordinated by the active orchestrating agent, but the method is role-based: Planner/Dispatcher plans and routes review, Generator explores/implements/fixes, Reviewer/Evaluator judges evidence; the human owns goals, scope, risk acceptance, and key gates, not manual tool handoff
- runtime-sensitive changes such as auth, permissions, menu routing, import/export, lowcode, dynamic modules, async chains, and external integrations need runtime evidence or an explicit runtime gap in addition to tests
- when the same failure pattern repeats, ratchet it from closeout note to repo rule and then to checker, smoke path, or failure-registry entry

Use the Chinese source document for the full repository rules, task-order constraints, and architectural red lines.
