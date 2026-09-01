---
title: GitHub 治理清单
doc_type: Acceptance
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md
updated_at: 2026-06-10
---

# GitHub Governance Checklist

Use this checklist when enabling the execution flow described in `docs/designs/WORKFLOW.md` and `docs/acceptances/CODE_REVIEW_STANDARD.md`.

## Repository settings

- Protect `main` and `release/*`
- Require conversation resolution
- Restrict direct pushes to protected branches
- Allow auto-merge
- Keep required approvals at `0` for the current solo-maintainer workflow
- Prefer squash merge
- Automatically delete head branches after merge

## Required checks

- `Quality Gates`
- `Security Gates`
- Keep PR-required checks limited to fast, deterministic merge-gate jobs
- Set branch-protection status checks to non-strict (`strict=false`) so auto-merge does not stall on "must be up to date"
- Keep `Full Smoke Suite` manual, scheduled, or release-precheck only
- Keep Sonar 和 Codacy out of required checks

## Secrets

- Any repository secrets needed by active security or deployment workflows

## Review evidence

- PR description records ownership layer, boundary, validation, `Quality Gates`, `Security Gates`, and CodeQL status
- PR description records Copilot review status or explicit unavailability
- PR description records auto-merge status
- Auto-merge automation only opens after the PR body passes repository governance validation
- Auto-merge automation only opens after `npm run check:github-feedback -- --repo <owner/repo> --pr <number>` confirms that PR, linked issue, and linked discussion feedback are fully closed
- High-risk PRs record residual risk and rollback notes explicitly
- Closeout records merged PR URL, merge commit, and branch cleanup status
