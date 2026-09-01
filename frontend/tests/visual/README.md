# Visual Regression Gate

Use this suite as the mechanical visual guardrail for frontend UI changes. It requires the
frontend and backend runtime expected by the smoke tests.

## Agent Workflow

1. Before a UI task, run `npm run test:visual` and confirm the committed baselines are green.
2. Make the UI change.
3. Run `npm run test:visual` again.
4. If the diff is intentional, run `npm run test:visual:baseline`, review the regenerated
   snapshots, and attach before/after images as delivery evidence.
5. If the diff is unintended, fix the UI and rerun `npm run test:visual`.

Never blind-update baselines to make a failing visual gate pass. Baseline updates are only for
reviewed, intentional visual changes.

The config has three named projects:

- `desktop-light`: existing `1440x900` page baselines.
- `mobile-light`: `390x844` tests marked `@mobile`.
- `desktop-dark`: `1440x900` tests marked `@dark`.

Every new shared UI surface must add the smallest meaningful state set to these projects and keep
interaction, permission, accessibility, and runtime assertions outside screenshot comparisons.
