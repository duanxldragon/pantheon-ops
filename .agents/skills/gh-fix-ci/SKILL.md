---
name: gh-fix-ci
description: Use when a Pantheon Ops GitHub Actions run is red and local verification is green or inconclusive, so CI repair must be driven from GitHub run details and inheritance-aware reproduction
---

# GH Fix CI

This is the Pantheon Ops adaptation of CI-fix workflow for GitHub Actions.

## Before Using

- Reproduce locally first with `repo-ci-triage`.
- Run `npm run check:inheritance` when the patch may involve shared paths or a recent base sync.
- Do not use CI debugging as a substitute for local red tests.

## Minimal Loop

1. Identify the failing workflow, job, and step.
2. Pull the failed-run details with GitHub CLI when available.
3. Map the failing job to local commands from `repo-ci-triage`.
4. Fix the smallest real cause.
5. Re-run the proof commands plus `repo-verify`.

## GitHub CLI Hints

- `gh run list --limit 10`
- `gh run view <run-id> --json jobs`
- `gh run view <run-id> --log-failed`

## Pantheon Ops-Specific Rules

- `quality.yml` focuses on buildable quality; reproduce the exact failing job, not a broader smoke suite unless the scope requires it.
- `security.yml` is report-oriented. Do not treat every vulnerability report as a hosted-run failure requiring YAML surgery.
- If `.github/workflows/*` changed, include local `zizmor` reproduction before claiming the workflow fix is ready.
- If the failure follows base-sync or inheritance drift, repair the sync boundary first rather than layering an ops-only workaround.

## Final Report

- failing run identifier
- failing job and step
- local reproduction command
- root cause
- fix
- remaining nonlocal risk
