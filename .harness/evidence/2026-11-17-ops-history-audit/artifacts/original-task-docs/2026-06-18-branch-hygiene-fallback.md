# Task Packet: branch-hygiene-fallback

## Goal

Add an independent GitHub branch-hygiene fallback so `pantheon-ops` can automatically delete stale same-repo PR head branches that remain after merge or close, without re-binding branch cleanup to `pull_request.closed`.

## Primary Layer

platform/governance

## Dependency Layers

- none

## Harness Profile

- Template: pantheon-ops custom
- Overlay: none
- Quality Profile: ci-workflow
- Portable Failure Class: ci-signal-noise
- Owner Layer: repository-governance
- Coverage Dimensions:
  - maintainability
  - architecture-fitness

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `../pantheon-base/AGENTS.md`
- `../pantheon-base/DESIGN.md`
- `../pantheon-base/docs/README.md`

## Scope

### In

- add an independent `branch-hygiene` GitHub Actions workflow for fallback branch cleanup
- add a deterministic Node cleanup script that deletes only stale same-repo PR head branches
- add focused workflow and script tests for the fallback cleanup contract

### Out

- changing the existing PR governance or auto-merge decision flow
- restoring `pull_request.closed` cleanup coupling
- changing business-domain code or inheritance overlays
- deleting local branches or worktrees from developer machines

## Structural Scope

- Affected Subgraph: `main push or schedule -> branch hygiene workflow -> GitHub PR history -> remote branch deletion`
- Boundary Crossings: `none`
- Risk Nodes: `workflow trigger scope`, `head branch deletion guardrails`, `same-branch reuse safety`
- Graph Focus: `sensitive-input-flow`

## Expected Files

### Create

- `.github/workflows/branch-hygiene.yml`
- `scripts/cleanup-github-branches.mjs`
- `tests/scripts/branch-hygiene-workflow.test.mjs`
- `tests/scripts/cleanup-github-branches.test.mjs`
- `docs/harness/tasks/2026-06-18-branch-hygiene-fallback.task.md`

### Modify

- `package.json`

### Do Not Touch

- `.github/workflows/pr-automation.yml`
- `backend/modules/business/*`
- `frontend/src/modules/business/*`
- shared backend sync payloads from `pantheon-base`

## Implementation Notes

- The fallback must run independently from PR-close events so GitHub auto-merge race conditions do not strand stale branches.
- The deletion guard must only remove branches whose current remote HEAD SHA still matches a closed same-repo PR head SHA.
- A recreated branch with the same name but a different SHA must be preserved.
- A branch still referenced by an open PR must be preserved.

## Method Readiness

- Consumer-Specific Controls: pantheon-ops repo-local governance
- Required Sensors: command
- Required Evidence: workflow test output | script test output
- Ratchet Decision: gate-updated
- Deferred Code Issues: none

## Delivery Governance

- Design Gate: short boundary note
- Development Gate: expected files declared
- QA Acceptance Gate: local workflow and script verification
- GitHub Governance Gate: repo-quality-gate

## Execution Roles

- Implementer Posture: implementer
- Reviewer Posture: architecture | mechanical

## Stop Points

- stop before widening cleanup beyond same-repo PR head branches
- stop before changing merge strategy or PR governance requirements

## Verification Plan

- `node --test tests/scripts/cleanup-github-branches.test.mjs`
- `node --test tests/scripts/branch-hygiene-workflow.test.mjs`

## Linkage

- Task ID: `2026-06-18-branch-hygiene-fallback`
- Task Manifest: `.harness/tasks/2026-06-18-branch-hygiene-fallback/manifest.json`
- OpenSpec Change: none
- Plan References: none
- Evidence Directory: `.harness/evidence/2026-06-18-branch-hygiene-fallback/`
- Review File: `.harness/evidence/2026-06-18-branch-hygiene-fallback/review.md`

## Evidence Required

- branch hygiene workflow test output
- branch hygiene script test output
- final branch cleanup rule summary

## Human Gates

- none

## Completion Checklist

- [ ] Layer and landing side declared
- [ ] Scope and safety guards recorded
- [ ] Required evidence files created or summarized
- [ ] Workflow posture validated
- [ ] Script behavior re-verified
