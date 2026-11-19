---
title: Repair PR auto-merge authentication
doc_type: Remediation
layer: ci-workflow
status: Superseded
superseded_by: 2026-11-17-ops-auto-merge-reconcile
updated_at: 2026-11-17
linked_contracts:
  - docs/designs/WORKFLOW.md
---

# Task Packet: 2026-08-12-pr-auto-merge-token

> ⚠️ Superseded — see `2026-11-17-ops-auto-merge-reconcile` for the consolidated
> decision. Original bytes preserved in `.harness/evidence/2026-11-17-ops-auto-merge-reconcile/original-task-cards/`.

## Goal

Restore reliable squash auto-merge after required PR checks pass.

## Scope

### In

- Authenticate the `gh pr merge --auto` workflow step with `github.token`.
- Fail the workflow when enabling auto-merge fails.
- Add a regression test for both requirements.

### Out

- Branch-protection policy changes.
- Product, backend, frontend, or release content changes.

## Verification

- `npm run test:pr-automation-workflow`
- `npm run check:docs-frontmatter`
- Hosted workflow security and actionlint checks.

## Decision (2026-11-17)

Superseded by `2026-11-17-ops-auto-merge-reconcile`. `github.token` suppresses
push-triggered workflows after the merge, so this record's auth approach is
rejected in favor of `secrets.RELEASE_GATE_TOKEN` with fail-closed (no
`github.token` fallback), matching the `2026-08-12-pr-auto-merge-push-trigger`
contract. Its fail-on-enable requirement (failure must fail the workflow) is
retained and now satisfied by the fail-closed guard + `set -euo pipefail`.
