---
title: GitHub Governance Checklist
doc_type: Acceptance
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md
updated_at: 2026-06-10
---

# GitHub Governance Checklist

Use this checklist when enabling the workflow described in `docs/designs/WORKFLOW.md` and `docs/acceptances/CODE_REVIEW_STANDARD.md`.

## Repository settings

- Protect `main` and `release/*`
- Require conversation resolution
- Restrict direct pushes to protected branches
- Allow auto-merge
- Prefer squash merge
- Keep required approvals at `0` for the current solo-maintainer workflow
- Automatically delete head branches after merge

## Required checks

- `Quality Gates`
- `Security Gates`
- Keep PR-required checks limited to fast, deterministic merge-gate jobs
- Set branch-protection status checks to non-strict (`strict=false`) so auto-merge does not stall on "must be up to date"
- Keep `Full Smoke Suite` manual, scheduled, or release-precheck only
- Keep Sonar and Codacy out of required checks

## Secrets

- Any repository secrets needed by active security or deployment workflows

## Review evidence

- PR description records ownership layer, boundary, validation, `Quality Gates`, `Security Gates`, and CodeQL status
- PR description records Copilot review status or explicit unavailability
- PR description records auto-merge status
- High-risk PRs record residual risk and rollback notes explicitly
- Closeout records merged PR URL, merge commit, and branch cleanup status
