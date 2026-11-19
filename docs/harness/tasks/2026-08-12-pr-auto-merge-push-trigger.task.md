---
title: Preserve post-merge workflow triggers
doc_type: Remediation
layer: ci-workflow
status: Active
updated_at: 2026-11-17
supersedes: 2026-08-12-pr-auto-merge-token
linked_contracts:
  - docs/designs/WORKFLOW.md
---

# Task Packet: 2026-08-12-pr-auto-merge-push-trigger

> Reconcile links: decision `2026-11-17-ops-auto-merge-reconcile`. This packet
> is the in-force contract; `2026-08-12-pr-auto-merge-token` is superseded by it.

## Goal

Ensure an automated merge creates the exact-commit push workflow signals required by foundation publication.

## Scope

### In

- Prefer the existing `RELEASE_GATE_TOKEN` for `gh pr merge --auto`.
- Fail closed when `RELEASE_GATE_TOKEN` is unavailable; do not fall back to `github.token`.
- Exact-commit recovery trigger: reuse the existing `quality.yml` `workflow_dispatch`
  (manual) + `push`/`main` (post-merge) entrypoints; no separate `ci.yml` is created.
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
