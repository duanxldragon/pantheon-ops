---
title: Repair PR auto-merge authentication
doc_type: Remediation
layer: ci-workflow
status: Active
updated_at: 2026-08-12
linked_contracts:
  - docs/designs/WORKFLOW.md
---

# Task Packet: 2026-08-12-pr-auto-merge-token

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
