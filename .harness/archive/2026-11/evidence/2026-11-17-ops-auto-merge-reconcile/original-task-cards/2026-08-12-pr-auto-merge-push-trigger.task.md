---
title: Preserve post-merge workflow triggers
doc_type: Remediation
layer: ci-workflow
status: Active
updated_at: 2026-08-12
linked_contracts:
  - docs/designs/WORKFLOW.md
---

# Task Packet: 2026-08-12-pr-auto-merge-push-trigger

## Goal

Ensure an automated merge creates the exact-commit push workflow signals required by foundation publication.

## Scope

### In

- Prefer the existing `RELEASE_GATE_TOKEN` for `gh pr merge --auto`.
- Fail closed when `RELEASE_GATE_TOKEN` is unavailable; do not fall back to `github.token`.
- Add a manual CI trigger for exact-commit recovery.
- Add regression tests for both requirements.

### Out

- Branch-protection or required-check policy changes.
- Repository-token fallback for automated merges.
- Product, backend, frontend, or release validation changes.
- Bypassing the publisher's exact-commit Release Gate requirement.

## Verification

- `npm run test:pr-automation-workflow`
- `npm run check:docs-frontmatter`
- `npm run check:task-packet-template`
- Hosted PR checks and post-merge push workflows.
