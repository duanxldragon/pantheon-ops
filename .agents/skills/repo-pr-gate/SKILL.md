---
name: repo-pr-gate
description: Use when preparing a Pantheon Ops pull request or merge candidate and deciding the required inheritance, review, evidence, and security gates
---

# Repo PR Gate

Pantheon Ops PR closure must prove both business correctness and inheritance discipline.

## Required Sequence

1. Run `repo-verify`.
2. Decide whether the landing side is really `pantheon-ops`.
3. Attach evidence.
4. Request independent review.

## Mandatory Landing-Side Check

If the issue touches any shared platform, system-domain, permission, menu, i18n, audit, shell, table, upload, or smoke-helper behavior, the PR must explain why the change stays in ops instead of going back to `pantheon-base`.

## Risk Split

- Standard business change:
  - at least one non-author approval
- High-risk change:
  - at least two non-author approvals
  - one reviewer should be domain, security, or architecture responsible

High-risk scope in this repo includes:

- any `base -> ops` shared-path sync
- changes under inherited `system/*` or auth flows
- permission, audit, menu, or i18n behavior
- generator or dynamic-module governance
- `.github/workflows/*`
- secrets, credentials, dependency posture

## Extra Gates

- UI change:
  - use `impeccable`
  - attach rendered evidence or a concrete runtime gap
- Runtime-sensitive or security-sensitive change:
  - run `security-diff-scan`
- Inheritance-sensitive change:
  - include `npm run check:inheritance` result in the PR summary

## PR Body Minimum

- owning layer
- landing-side decision
- change boundary
- commands run
- evidence summary
- known gaps
