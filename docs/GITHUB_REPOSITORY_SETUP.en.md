---
title: GitHub Repository Setup Guide
doc_type: Acceptance
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md
updated_at: 2026-06-10
---

# GitHub Repository Setup Guide

Use this guide to activate the workflow defined in `docs/designs/WORKFLOW.md` and `docs/acceptances/CODE_REVIEW_STANDARD.md`.

## 1. Base repository

### 1.1 Branch protection

Open `Settings -> Branches -> Branch protection rules` and add rules for:

- `main`
- `release/*`

Enable:

- Require a pull request before merging
- Require conversation resolution before merging
- Restrict pushes that create matching branches
- Allow auto-merge
- Prefer squash merge as the standard merge method
- Automatically delete head branches after merge so merged worktree branches do not linger on GitHub

For a solo-maintainer repository, keep required approvals at `0` and let GitHub-native required checks plus PR traceability carry the merge gate.

### 1.2 Required status checks

Add these checks to the required checks list:

- `Quality Gates`
- `Security Gates`

Set the required status checks policy to non-strict (`strict=false`) so auto-merge does not stall on "must be up to date with base branch" in a single-maintainer flow.

Do not add `Full Smoke Suite`, Sonar, Codacy, or other external scanners to required checks. `Quality Gates` must stay fast and deterministic for PR feedback, while `Security Gates` is the only required GitHub-native security aggregate and already includes CodeQL.

### 1.3 Copilot review

If your account or organization supports GitHub Copilot code review, enable automatic review or let the repository workflow request `@copilot` for each PR.

### 1.4 Secrets

Configure only the repository secrets required by active GitHub Actions workflows.

Do not add inactive scanner secrets to the repository. The active workflow stack is GitHub-native plus CodeQL.

### 1.5 Review gate evidence

Require the PR template to record:

1. `Quality Gates` result
2. `Security Gates` result
3. CodeQL result or alert link
4. Copilot review status
5. auto-merge status
6. residual-risk / rollback note for high-risk changes

## 2. Ops repository

Repeat the same branch-protection, required-check, and Copilot/auto-merge steps in `pantheon-ops`.

The ops repository should keep the same review discipline as base, but only own business-specific drift and business-domain changes.

## 3. Verification

After configuration, verify:

- branch protection blocks direct push to protected branches
- required checks show only GitHub-native merge gates
- PRs enable auto-merge after the gate workflow runs
- Copilot review is requested automatically when available
- merged PR head branches are deleted automatically
- the PR template records gate status, Copilot status, and rollback notes
