---
title: 2026-06-18-pr-branch-cleanup-automation
doc_type: Remediation
layer: business
status: Archived
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-11-17
---

# Task Packet: 2026-06-18-pr-branch-cleanup-automation

## Goal

Make solo-maintainer PR flow delete merged same-repo head branches deterministically in `pantheon-ops`, instead of relying on GitHub repository auto-delete behavior alone.

## Primary Layer

platform/governance

## Dependency Layers

- none

## Harness Profile

- Template: pantheon-ops custom
- Overlay: none
- Quality Profile: ci-workflow
- Portable Failure Class: repo-quality-gate
- Owner Layer: repository-governance
- Coverage Dimensions:
  - maintainability
  - security

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `../pantheon-base/AGENTS.md`

## Scope

### In

- add `pull_request.closed` cleanup handling to PR automation
- explicitly delete merged same-repo head branches after merge
- lock cleanup behavior with workflow tests
- capture one-time evidence for current branch-hygiene cleanup

### Out

- business runtime behavior
- frontend runtime behavior
- shared base sync remediation

## Expected Files

### Create

- `docs/harness/tasks/2026-06-18-pr-branch-cleanup-automation.task.md`
- `.harness/evidence/2026-06-18-pr-branch-cleanup-automation/commands.json`
- `.harness/evidence/2026-06-18-pr-branch-cleanup-automation/summary.md`
- `.harness/evidence/2026-06-18-pr-branch-cleanup-automation/review.md`
- `.harness/evidence/2026-06-18-pr-branch-cleanup-automation/pr-body.md`

### Modify

- `.github/workflows/pr-automation.yml`
- `tests/scripts/pr-automation-workflow.test.mjs`

### Do Not Touch

- `backend/modules/business/*`
- `frontend/src/modules/business/*`
- `../pantheon-base/**`

## Implementation Notes

- Repository setting `deleteBranchOnMerge=true` remains enabled, but workflow cleanup owns the deterministic delete path after merge.
- Cleanup applies only to merged PRs whose head branch belongs to the same repository and differs from the base branch.

## Verification Plan

- `node --test tests/scripts/pr-automation-workflow.test.mjs`
- `node --test tests/scripts/run-github-feedback-loop.test.mjs`
- `npm run check:pr-governance`
- `npm run check:docs-frontmatter`
- `npm run check:task-packet-template`
- `npm run check:inheritance-contract`
- `npm run check:generated-modules`

## Linkage

- Task ID: `2026-06-18-pr-branch-cleanup-automation`
- Task Manifest: `.harness/tasks/2026-06-18-pr-branch-cleanup-automation/manifest.json`
- OpenSpec Change: `none`
- Evidence Directory: `.harness/evidence/2026-06-18-pr-branch-cleanup-automation/`
- Review File: `.harness/evidence/2026-06-18-pr-branch-cleanup-automation/review.md`

## Evidence Required

- workflow test output
- governance check output
- current remote branch cleanup record

## Human Gates

- none

## Completion Checklist

- [ ] Layer and landing side declared
- [ ] Cleanup rule added
- [ ] Verification run or exception recorded
- [ ] Evidence saved or summarized
- [ ] Review completed

## Result — History Closeout 2026-11-17

Manifest disposition: abandoned

原 pull_request.closed 耦合被独立 branch-hygiene 工作流明确取代，后继 hosted 删除及保护机制已获独立原范围批准。

Superseded by: 2026-06-18-branch-hygiene-fallback, 2026-06-18-branch-hygiene-slash-path-fix。原任务正文/evidence 保全；本地 divergent/未发布分支维护责任仍开放。

Review: .harness/evidence/2026-11-17-ops-history-audit/pr-branch-cleanup-review.md
Evidence: .harness/evidence/2026-11-17-ops-history-audit/second-batch-preflight.json
