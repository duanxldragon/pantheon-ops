# Task Packet: github-governance-followup

## Goal

Make the Pantheon Ops GitHub governance PR self-contained and mergeable without turning existing product baseline debt into a blocker for a repository-governance-only patch.

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
- `../pantheon-base/DESIGN.md`
- `../pantheon-base/AGENTS.md`
- `../pantheon-base/docs/README.md`
- `../harness-engineering/docs/harness/VERIFICATION_EVIDENCE_SPEC.md`

## Scope

### In

- Keep the PR governance template, body validator, and duplication report scripts introduced by this branch.
- Pin GitHub Actions workflow references and disable credential persistence in required-check workflows.
- Scope frontend, smoke, and inheritance required gates so governance-only pull requests do not fail on unrelated repo baseline debt.
- Add task packet, commands evidence, and review artifact linkage required by the PR governance body checker.

### Out

- Fix existing frontend i18n hardcode debt.
- Fix existing system smoke regressions.
- Resolve current `base -> ops` shared backend drift.
- Reduce current repository-wide duplication baseline below the long-term threshold.

## Expected Files

### Create

- `docs/harness/tasks/2026-06-17-github-governance-followup.task.md`
- `.harness/evidence/2026-06-17-github-governance-followup/commands.json`
- `.harness/evidence/2026-06-17-github-governance-followup/review.md`
- `.harness/evidence/2026-06-17-github-governance-followup/pr-body.md`

### Modify

- `.github/workflows/quality.yml`
- `.github/workflows/security.yml`

### Do Not Touch

- `backend/modules/business/*`
- `frontend/src/modules/business/*`
- shared backend sync payloads from `pantheon-base`

## Implementation Notes

- This patch stays in `pantheon-ops` because it is repository governance and local PR closure logic, not shared platform runtime behavior.
- `frontend-contract` and `smoke-sanity` remain required for product-affecting pull requests, but are skipped for governance-only pull requests.
- `check:inheritance` stays available for inheritance-sensitive changes, but is no longer an unconditional docs-governance blocker while the repo baseline still carries shared backend drift.

## Method Readiness

- Consumer-Specific Controls: pantheon-ops repo-local governance
- Required Sensors: command
- Required Evidence: commands.json | review.md
- Ratchet Decision: gate-updated
- Deferred Code Issues: frontend hardcode debt | system smoke debt | base-sync drift | duplication baseline

## Delivery Governance

- Design Gate: short boundary note
- Development Gate: workflow pinning | task/evidence/review linkage
- QA Acceptance Gate: local script and workflow posture verification
- GitHub Governance Gate: repo-quality-gate

## Structural Scope

- Affected Subgraph: `pull_request -> governance scripts -> quality/security workflows -> required checks`
- Boundary Crossings: `none`
- Risk Nodes: `quality gate aggregator`, `security gate aggregator`, `workflow posture`
- Graph Focus: `none`

## Verification Plan

- `node --test tests/scripts/check-pr-governance.test.mjs`
- `node --test tests/scripts/check-duplication.test.mjs`
- `npm run check:pr-governance`
- `npm run check:docs-frontmatter`
- `npm run check:task-packet-template`
- `npm run check:generated-modules`
- `zizmor --format plain .github/workflows`

## Linkage

- Task ID: 2026-06-17-github-governance-followup
- OpenSpec Change: none
- Plan References: none
- Evidence Directory: `.harness/evidence/2026-06-17-github-governance-followup/`
- Review File: `.harness/evidence/2026-06-17-github-governance-followup/review.md`

## Evidence Required

- command result summary
- PR governance body
- workflow posture report

## Human Gates

- none

## Completion Checklist

- [ ] Layer and landing side declared
- [ ] Scope and deferred debt split recorded
- [ ] Required evidence files created
- [ ] Workflow posture validated
- [ ] Governance scripts re-verified
- [ ] PR body updated to real artifact paths
