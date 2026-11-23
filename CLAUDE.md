# Pantheon Ops

Chinese operational rules: [AGENTS.md](./AGENTS.md)

pantheon-ops inherits pantheon-base as its foundation. Treat `pantheon-base` as the standard backoffice source of truth for platform, system domains, shared UI rules, backend contracts, permission, menu, i18n, audit, and acceptance standards.

pantheon-ops exists to add operations-domain business modules. It must not become a forked copy of the foundation.

## Maintainer Contract: three touchpoints only

The maintainer intervenes at exactly three points — (1) requirement clarification at intake (batch ALL questions once, produce In/Out/acceptance criteria, then stop asking), (2) gate-policy decisions (red gates, exemptions, rule changes), (3) final visual/functional acceptance. Between those, run autonomously: no mid-task confirmations for reversible in-scope work; gates and evidence replace verbal confirmation. See `../pantheon-harness/architecture/methodology/workflow-routing.md` → Human Touchpoints.

## Capability-Adaptive Execution

Consume [capability-adaptive delivery](../pantheon-harness/patterns/capability-adaptive-delivery.md) and follow `AGENTS.md`. The current primary agent, including Claude Code, may plan, implement, verify, and fix authorized `business/*` work in one session. Planning-only/review-only assignments stay read-only. High-risk L2 keeps independent review or an explicit review gap and acceptance gate. Preserve the user's current model, effort, provider, and settings; old Base model-tier examples do not mandate a model switch.

Delegate only self-contained branches with ownership, budget, evidence, stop condition, and an integration owner. Use an actual available and authorized entrypoint; a model name does not grant a tool or permission.

Before work and at closeout run `node ../pantheon-harness/scripts/harness/check-method-consumption.mjs --root . --source ../pantheon-harness`. The receipt and these entrypoints are Ops-owned repository overlay assets. Direct method consumption does not change the locked Base snapshot or permit shared-source overrides.

## Business work reading order

Before implementing, reviewing, debugging, or designing in this repository, read only the relevant files in this order:

1. `../docs/WORKSPACE_INHERITANCE.md`
2. `AGENTS.md`
3. `docs/PROJECT_INHERITANCE.md`
4. `../pantheon-base/DESIGN.md`
5. `../pantheon-base/AGENTS.md`
6. `../pantheon-base/docs/README.md`
7. matching `../pantheon-base/docs/contracts/*`
8. matching `../pantheon-base/docs/designs/*`
9. matching `../pantheon-base/docs/acceptances/*`
10. local business docs under `docs/designs/BUSINESS_*`

Repository-local workflow skills live under:

- `.agents/skills/README.md`

## Layer ownership

- `pantheon-base` owns `platform` and `system/*`.
- `pantheon-ops` owns `business/*` and repository-local business docs.
- Business modules may use base extension points, shared contracts, and public packages.
- Business modules must not directly rewrite base platform or system-domain behavior.

Do not fix platform or system-domain drift locally in pantheon-ops. If the behavior should apply to the standard backoffice, change `pantheon-base` first, then upgrade `pantheon-ops`.

## Sync discipline

pantheon-ops consumes Base through a **locked official release snapshot** plus `business-overlay.json`. Read `docs/PROJECT_INHERITANCE.md` for the canonical fetch/rebuild/verification flow. Do not compare against Base HEAD, pass obsolete local Base source flags, manually edit the foundation lock, or copy shared source. Shared method/checker/config changes in Base reach the product tree through a formal foundation release.

For scoped overlay changes, run `node scripts/business-overlay/check-business-overlay.mjs` and relevant documentation checks. Product baseline replacement and non-trivial PR closeout retain the full verification set in `AGENTS.md` and `docs/PROJECT_INHERITANCE.md`, including the Windows CGO race gate.

## Design system

Always read `../pantheon-base/DESIGN.md` before visual or UI decisions, then read local business design docs if the task touches `business/*` UI.

Key inherited constraints:

- Font: Source Sans 3 for body/UI, JetBrains Mono for code.
- No Inter, no radial-gradient, no large button shadows, no non-standard font weights.
- Radius: 4/6/8/12px. No pill-radius cards.
- All colors via Pantheon CSS tokens, never raw Arco variables.
