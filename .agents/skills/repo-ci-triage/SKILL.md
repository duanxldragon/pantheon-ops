---
name: repo-ci-triage
description: Use when Pantheon Ops GitHub Actions is red and the failure must be mapped back to this repository's inheritance checks, business smoke expectations, and workflow names
---

# Repo CI Triage

Start from the failing workflow, job, and step. In Pantheon Ops, check inheritance drift before assuming the failure is a pure app regression.

## Gather First

- workflow name
- job name
- failing step name
- commit SHA or PR head
- short log excerpt

## Workflow Map

- `quality.yml` -> `docs-governance`
  - `npm ci`
  - `npm run check:docs-frontmatter`
  - `npm run check:task-packet-template`
  - `npm run check:generated-modules`
- `quality.yml` -> `frontend-contract`
  - `cd frontend && npm ci`
  - `npm run check:menu-contract`
  - `npm run lint`
  - `PANTHEON_CI=true npm run i18n:generate-module`
  - `PANTHEON_CI=true npx vite build`
- `quality.yml` -> `backend-tests`
  - `go test -race ./...`
- `quality.yml` -> `codeql-security`
  - usually not fully reproducible locally
  - first fix compile, sink/control, and trust-boundary issues
  - if the diff is risky, run `security-diff-scan`
- inheritance-related red flags not named directly in CI:
  - `npm run check:inheritance`
  - `npm run check:base-sync`
- `security.yml` -> dependency reports
  - `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`
  - `go run github.com/google/osv-scanner/v2/cmd/osv-scanner@latest scan --recursive .`
  - `npm ci && npm audit --registry=https://registry.npmjs.org --audit-level=moderate`
  - `cd frontend && npm ci && npm audit --registry=https://registry.npmjs.org --audit-level=moderate`
- `security.yml` -> secret scan report
  - `go run github.com/zricethezav/gitleaks/v8@latest detect --source . --redact --exit-code 0`
- `security.yml` -> workflow posture reports
  - `python -m pip install --user zizmor`
  - `zizmor --format json .github/workflows`

## Ops-Specific Notes

- `security.yml` is largely report-oriented. Vulnerability findings may need remediation, but they are not the same as a broken workflow run.
- If CI red follows a base upgrade or shared-path sync, reproduce `npm run check:inheritance` before deeper debugging.
- If the failure is in module i18n generation, keep the generated output and the business module changes in the same patch.

## Exit Condition

Report:

- failing workflow/job/step
- local reproduction command
- root cause
- fix applied
- remaining hosted-only or inheritance-only risk
