---
title: GitHub Governance Checklist
doc_type: Acceptance
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md
updated_at: 2026-10-20
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

## Security surfaces

- `SECURITY.md` stays the Security-tab policy entry, with `SECURITY.en.md` as its English mirror; a control change updates both in the same PR
- Private vulnerability reporting stays enabled, so the private advisory path documented in `SECURITY.md` actually exists
- Code scanning (CodeQL) runs on every push to `main` and every PR
- Dependabot alerts and security updates stay enabled; no high/critical alert may be open at release time
- Secret scanning stays enabled with push protection; a blocked push is resolved by rotating the credential, never by bypassing the protection
- SonarCloud is not a required check, but the release gate requires a green quality gate and zero unresolved issues
- An alert is closed by a fix or by a dismissal that records its basis (`false positive`, `won't fix`, `used in tests`) and the file it covers

## Review evidence

- PR description records ownership layer, boundary, validation, `Quality Gates`, `Security Gates`, and CodeQL status
- PR description records Copilot review status or explicit unavailability
- PR description records auto-merge status
- High-risk PRs record residual risk and rollback notes explicitly
- Closeout records merged PR URL, merge commit, and branch cleanup status
