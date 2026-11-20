# Summary — `2026-11-17-ops-auto-merge-reconcile`

Date 2026-11-17. Owner: ops-closeout Claude (G1). Task status: **in-progress /
awaiting-review** (independent Codex acceptance). Parent: `2026-11-16-ops-historical-closeout`.

## Decision (see `decision.md`)

**Approach A wins** — `2026-08-12-pr-auto-merge-token` is **superseded** by
`2026-08-12-pr-auto-merge-push-trigger`'s contract:
`secrets.RELEASE_GATE_TOKEN`, fail-closed, no `github.token` fallback.

## What changed

| File                                                               | Change                                                                                                                                                                                                                                                                                                                                                 |
| ------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `.github/workflows/pr-automation.yml`                              | "Enable squash auto-merge" step now uses `GH_TOKEN: ${{ secrets.RELEASE_GATE_TOKEN }}`; adds fail-closed guard (`if [ -z "${GH_TOKEN:-}" ] ... exit 1`); **no `github.token` literal**; `gh pr merge --auto --squash --delete-branch` preserved; governance/feedback/draft/same-repo/author-association `if:` gates and failure propagation untouched. |
| `.github/workflows/ci.yml`                                         | **Not created.** `quality.yml` already owns exact-commit triggers (`workflow_dispatch:` manual + `push:` on `main`/`release/**` post-merge). Reused — no empty/duplicate CI invented for the old test filename.                                                                                                                                        |
| `tests/scripts/ci-workflow-dispatch.test.mjs`                      | Repointed from the phantom `ci.yml` to `quality.yml` and asserted the real final contract: `workflow_dispatch:` for manual exact-commit recovery and `push:` → `branches: main` so the dispatched validation follows the merged SHA (not a substitute target).                                                                                         |
| `tests/scripts/pr-automation-workflow.test.mjs`                    | No edit needed — already asserts the RELEASE_GATE_TOKEN + fail-closed + no-github.token contract the workflow now implements.                                                                                                                                                                                                                          |
| `docs/harness/tasks/2026-08-12-pr-auto-merge-token.task.md`        | status → `Superseded`, `superseded_by: 2026-11-17-ops-auto-merge-reconcile`, `updated_at: 2026-11-17`, added reconcile note + decision; original bytes preserved in evidence.                                                                                                                                                                          |
| `docs/harness/tasks/2026-08-12-pr-auto-merge-push-trigger.task.md` | stays the in-force contract (status Active), `supersedes` the token record, `updated_at: 2026-11-17`; scope In updated to reflect quality.yml reuse (no ci.yml).                                                                                                                                                                                       |
| `.harness/.../2026-11-17-ops-auto-merge-reconcile/**`              | new reconcile package: task-manifest.json, decision.md, commands.json, summary.md, review.md, original-task-cards/ (both records' original bytes at HEAD).                                                                                                                                                                                             |

## Verification status

- **Not executed locally.** This headless session auto-denies `node`/`npm` (no approval
  surface): `node --test` and `npm run` were blocked, so the contract tests were **not
  run here**. I did not fake a run or claim `passed`.
- The workflow edit was verified **statically** against every regex assertion in
  `tests/scripts/pr-automation-workflow.test.mjs` and the rewritten
  `ci-workflow-dispatch.test.mjs` (match + doesNotMatch contours checked against the
  actual file bytes).
- Remaining hosted/runner gap: execute
  `npm run test:pr-automation-workflow`, `npm run check:docs-frontmatter`,
  `npm run check:task-packet-template`, `npm run check:docs-governance` (a normal
  interactive session or Codex acceptance), and record real outputs.

## Boundary honored

No Secret created/read/altered; no `gh merge`/publish/release run; no quality.yml /
dependabot-pr-governance / other-task changes; no pantheon-base change; no commit/push/
PR; no global STATUS / OPEN_TASKS / INVENTORY edits.
