# Pantheon Ops

pantheon-ops inherits pantheon-base as its foundation. Treat `pantheon-base` as the standard backoffice source of truth for platform, system domains, shared UI rules, backend contracts, permission, menu, i18n, audit, and acceptance standards.

pantheon-ops exists to add operations-domain business modules. It must not become a forked copy of the foundation.

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

## Layer ownership

- `pantheon-base` owns `platform` and `system/*`.
- `pantheon-ops` owns `business/*` and repository-local business docs.
- Business modules may use base extension points, shared contracts, and public packages.
- Business modules must not directly rewrite base platform or system-domain behavior.

Do not fix platform or system-domain drift locally in pantheon-ops. If the behavior should apply to the standard backoffice, change `pantheon-base` first, then upgrade `pantheon-ops`.

## Sync discipline

Before any non-trivial PR:

```powershell
node ../scripts/harness/check-inheritance-contract.mjs --root .. --strict
node ../scripts/harness/triage-base-drift.mjs --root .. --business pantheon-ops --json
```

Use drift categories as the decision gate:

- `generic drift`: backport to `pantheon-base` or record why it cannot be backported.
- `pseudo-drift`: do not expand it with ops-only edits.
- `business mount`: keep only as a narrow business integration point.
- `business-specific drift` and `business-only`: allowed when they stay inside the operations business scope.
- `base-only`: review during base upgrade.

## Design system

Always read `../pantheon-base/DESIGN.md` before visual or UI decisions, then read local business design docs if the task touches `business/*` UI.

Key inherited constraints:

- Font: Source Sans 3 for body/UI, JetBrains Mono for code.
- No Inter, no radial-gradient, no large button shadows, no non-standard font weights.
- Radius: 4/6/8/12px. No pill-radius cards.
- All colors via Pantheon CSS tokens, never raw Arco variables.
