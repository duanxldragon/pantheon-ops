# Task Packet: github-governance-followup

## Goal

Make the Pantheon Ops GitHub governance and feedback-closure automation self-contained and mergeable without turning existing inheritance drift or product baseline debt into a blocker for a repository-governance-only patch.

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
- Add GitHub feedback fetch/apply loop scripts plus the `gh-address-comments` repo skill so PR, issue, and discussion comments can be closed through one automation path.
- Scope frontend, smoke, and inheritance required gates so governance-only pull requests do not fail on unrelated repo baseline debt.
- Ensure solo PR automation only enables squash auto-merge after PR governance and GitHub feedback gates both pass.
- Add task packet, commands evidence, review artifact, and PR body linkage required by the PR governance body checker.

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
- `.agents/skills/gh-address-comments/SKILL.md`
- `scripts/address-github-feedback.mjs`
- `scripts/fetch-github-feedback.mjs`
- `scripts/run-github-feedback-loop.mjs`
- `tests/scripts/address-github-feedback.test.mjs`
- `tests/scripts/fetch-github-feedback.test.mjs`
- `tests/scripts/pr-automation-workflow.test.mjs`
- `tests/scripts/run-github-feedback-loop.test.mjs`

### Modify

- `.github/workflows/pr-automation.yml`
- `.github/workflows/quality.yml`
- `.agents/skills/README.md`
- `.agents/skills/README.zh.md`
- `.agents/skills/repo-pr-gate/SKILL.md`
- `AGENTS.md`
- `README.md`
- `README.en.md`
- `docs/PROJECT_INHERITANCE.md`
- `docs/PROJECT_INHERITANCE.en.md`
- `docs/README.md`
- `docs/README.en.md`
- `package.json`

### Do Not Touch

- `backend/modules/business/*`
- `frontend/src/modules/business/*`
- shared backend sync payloads from `pantheon-base`

## Implementation Notes

- This patch stays in `pantheon-ops` because it is repository governance and local PR closure logic, not shared platform runtime behavior.
- `frontend-contract` and `smoke-sanity` remain required for product-affecting pull requests, but are skipped for governance-only pull requests.
- `check:inheritance` stays available for real inheritance-sensitive changes, but documentation-only edits to `docs/PROJECT_INHERITANCE.md` should not be treated as a backend drift gate by themselves.
- The GitHub feedback loop is repo-governance only. It does not modify business runtime behavior.

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

- `node --test tests/scripts/address-github-feedback.test.mjs`
- `node --test tests/scripts/fetch-github-feedback.test.mjs`
- `node --test tests/scripts/run-github-feedback-loop.test.mjs`
- `node --test tests/scripts/pr-automation-workflow.test.mjs`
- `node --test tests/scripts/check-pr-governance.test.mjs`
- `npm run check:pr-governance`
- `npm run check:docs-frontmatter`
- `npm run check:task-packet-template`
- `npm run check:inheritance-contract`
- `npm run check:generated-modules`

## Linkage

- Task ID: 2026-06-17-github-governance-followup
- Task Manifest: `.harness/tasks/2026-06-17-github-governance-followup/manifest.json`
- OpenSpec Change: none
- Plan References: none
- Evidence Directory: `.harness/evidence/2026-06-17-github-governance-followup/`
- Review File: `.harness/evidence/2026-06-17-github-governance-followup/review.md`

## Evidence Required

- command result summary
- PR governance body
- workflow posture and gating summary

## Human Gates

- none

## Completion Checklist

- [ ] Layer and landing side declared
- [ ] Scope and deferred debt split recorded
- [ ] Required evidence files created
- [ ] Workflow posture validated
- [ ] Governance scripts re-verified
- [ ] PR body updated to real artifact paths
