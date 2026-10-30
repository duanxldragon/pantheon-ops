---
title: Remediate the open SonarCloud security and reliability findings on business scope
doc_type: Remediation
layer: business-scope
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
  - docs/designs/BUSINESS_ARGOCD_MODULE_DESIGN.md
updated_at: 2026-10-30
---

# Task Packet: 2026-10-30-sonar-findings-remediation

Target repository: `pantheon-ops`
Layer: `business-scope (backend/modules/business, frontend/src/modules/business, e2e-tests, .github)`
Delivery tier: `L2`
Mode: close the SonarCloud quality-gate findings that keep `duanxldragon_pantheon-ops` in ERROR

## Goal

Turn the hosted SonarCloud quality gate for `duanxldragon_pantheon-ops` from ERROR-by-default to
green-by-construction. The restored scan reports 242 open issues, of which 62 are vulnerability-class
and 6 are bug-class; the gate fails on `new_security_rating` and `new_reliability_rating`. This packet
fixes every vulnerability and bug in the touched scope with real code, converts the largest false-positive
family into a machine-checked invariant, and only then transitions the verified findings in Sonar with a
recorded, reversible justification.

## Primary Layer

business/*

## Workspace Context

- Target Repository: `pantheon-ops`
- Repository Role: `business-consumer`
- Upstream Dependencies: `pantheon-base`
- Downstream Consumers: `none`
- Sync Expectation: `not-required`
- Release Requirement: `none`

## Dependency Layers

- `backend/modules/business/cicd/argocd` — server TLS trust and SSRF validator
- `backend/modules/business/filedist` — storage root containment
- `backend/modules/business/**` GORM condition builders — the taint surface Sonar reports as S3649
- `scripts/check-gorm-sql-literal.mjs` + `tests/scripts/check-gorm-sql-literal.test.mjs` — new sensor
- `.github/workflows/quality.yml` `docs-governance` job — where the new sensor blocks
- `frontend/src/modules/business/**` — floating promises, hotfix placeholders, nested templates
- `frontend/src/i18n/resources/business/{zh-CN,en-US}.ts` — placeholder keys for the https guidance
- `e2e-tests/{auto-test.sh,run-real-machine-test.sh,playwright.config.ts}` — cleartext defaults,
  hardcoded target IP, `npx` at runtime, duplicated CI worker config
- SonarCloud organization `duanxldragon`, project `duanxldragon_pantheon-ops`, branch `main`

## Harness Profile

- Template: api-service
- Overlay: pantheon-ops
- Quality Profile: auth-security
- Portable Failure Class: static-sensor-gap
- Owner Layer: business-boundary
- Coverage Dimensions:
  - architecture-fitness
  - method-health
  - maintainability

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `scripts/check-business-module-boundaries.mjs`
- `scripts/check-gorm-sql-literal.mjs`
- `.github/workflows/quality.yml`
- `.github/workflows/security.yml`
- `sonar-project.properties`

## Scope

### In

- Fix `go:S4830` / `go:S5527`: the ArgoCD client must always verify the server certificate; private CAs
  are supported through a stored PEM bundle instead of `InsecureSkipVerify`.
- Fix `gosecurity:S2083`: filedist storage paths must stay inside the configured storage root, and upload
  file names must be reduced to one path element.
- Convert the `gosecurity:S3649` family (33 open issues) into a checked invariant: SQL text and column
  names in `backend/modules/business/**` must be literals or package constants, asserted by
  `scripts/check-gorm-sql-literal.mjs`, unit-tested, and blocking in CI.
- Fix the remaining frontend and e2e vulnerabilities/bugs: floating promises (`typescript:S3923`,
  `S7755`), cleartext endpoints in shell scripts (`shell:S5332`, `typescript:S5332`), duplicated CI
  worker config (`typescript:S3923` class duplicates), `npx` at runtime, and the `S4624` nested template.
- Remove the hardcoded reachable target IP from `e2e-tests/auto-test.sh` and make the target an explicit
  required environment input; move the CMDB import template example into the RFC 5737 documentation range.
- Record the exact Sonar issue keys for the S3649 set before any transition, then transition them to
  FALSE_POSITIVE with the checker as the written justification, and read back the gate.

### Out

- Mass refactoring of the legacy smell backlog that is outside the new-code period
  (`go:S1192` ×68, `go:S3776` ×27, `typescript:S3358` ×22, `typescript:S3863` ×10, `godre:*`,
  `shell:S7688`): recorded as legacy backlog, not silenced.
- Any change to SonarCloud quality-gate definitions, the new-code period, or the organization plan.
- Provisioning Dependabot secrets or changing the Dependabot scan policy.
- Removing the external Vercel deployment check, which needs a maintainer action in the GitHub UI.
- Adding a UI form for ArgoCD CA-certificate storage; `caCert` is API-only in this round.

## Expected Files

### Create

- `scripts/check-gorm-sql-literal.mjs`
- `tests/scripts/check-gorm-sql-literal.test.mjs`
- `docs/harness/tasks/2026-10-30-sonar-findings-remediation.task.md`
- `.harness/tasks/2026-10-30-sonar-findings-remediation/manifest.json`
- `.harness/evidence/2026-10-30-sonar-findings-remediation/{commands.json,summary.md,review.md,pr-body.md,sonar-s3649-keys.json,sonar-triage.json}`

### Modify

- `backend/modules/business/cicd/argocd/{argocd_client.go,argocd_client_test.go,argocd_dto.go,argocd_model.go,argocd_service.go,argocd_reconciler.go}`
- `backend/modules/business/filedist/{storage_service.go,storage_service_test.go}`
- `backend/modules/business/{retired_modules.go,service/state.go,cmdb/host/host_import_export.go,bizscope/bizscope_service.go,k8s/storage/storage.go}`
- `frontend/src/modules/business/cicd/argocd/api.ts`
- `frontend/src/modules/business/{bizscope/BizScopeList.tsx,cmdb/host/CmdbHostList.tsx,deploy/task/DeployTaskList.tsx,filedist/FiledistDetail.tsx,k8s/backup/BackupList.tsx,observability/MonitoringCenter.tsx}`
- `frontend/src/i18n/resources/business/{zh-CN.ts,en-US.ts}`
- `frontend/tests/smoke/business/observability-filedist-pages.spec.ts`
- `e2e-tests/{auto-test.sh,run-real-machine-test.sh,playwright.config.ts}`
- `.harness/v1.2-tasks/generate_tasks.sh`
- `docs/designs/BUSINESS_ARGOCD_MODULE_DESIGN.md`
- `.github/workflows/{quality.yml,security.yml}`
- `.github/PULL_REQUEST_TEMPLATE.md`
- `package.json`

### Do Not Touch

- `sonar-project.properties` and `.sonarcloud.properties` (project identity and scope stay as restored)
- `foundation-release.lock.json` and `.foundation/foundation-release.lock.json` (no lock advance)
- any repository secret value inside the working tree; `SONAR_TOKEN` is only ever used as a shell variable

## Implementation Notes

- `NewArgoCDClient` now returns an error when the stored CA bundle parses to no certificate
  (`ErrInvalidCACert`) and never sets `InsecureSkipVerify`. Tests that previously bypassed verification
  inject the `httptest` server certificate as the stored bundle, so the production code path under test is
  the real one.
- The S3649 family is not one bug but a shape: SonarGo treats any `Where/Or/Order/Updates` argument that
  is not a literal as tainted, including package constants that are provably literal. Rather than argue
  33 findings by hand, `scripts/check-gorm-sql-literal.mjs` asserts the invariant the argument depends on,
  every `+` operand must be a literal, a `clause`/`gorm` expression, or a package-level literal constant,
  and map column keys must be quoted literals. Line-scoped waivers require a written justification and are
  printed, so a waiver cannot become invisible drift.
- `retired_modules.go` previously built `module IN ?` from a column parameter. The three clauses are now
  package constants, matching the `condNameEquals` / `menuPathWhereClause` convention already used in the
  CMDB modules, and the two remaining call sites carry the visible waivers.
- `backend/modules/business/service/state.go` had byte-identical `TransitionActionUpgrade` and
  `TransitionActionRollback` branches (`go:S1871`); they are now one case, which is the same behavior with
  one place to change.
- `e2e-tests/auto-test.sh` shipped a reachable production IP and `http://` defaults (`shell:S5332` ×18).
  Both scripts now fail fast unless `API_BASE_URL` and `WEB_BASE_URL` are supplied, which removes the
  finding class instead of marking it won't-fix, and `npx playwright` became a pinned local binary.
- `typescript:S5332` on the monitoring placeholders was fixed by i18n keys that state the https
  requirement, not by deleting the hint.
- `.github/workflows/security.yml` now installs zizmor with `--only-binary :all:`, matching base, so a
  source-only PyPI package cannot execute arbitrary setup code in the runner.
- `scripts/check-gorm-sql-literal.mjs` is an Ops-only sensor: it scans `backend/modules/business/**`, which
  base does not ship. It therefore does not need a base-first round trip, and must not be hand-copied into
  base as a generic checker.
- The first hosted CI run on the PR drove a second round: `Go Lint` flagged one new gosec **G304** on
  `os.Create(fullPath)`, a line this round's own rewrite introduced, and the SonarCloud gate stayed ERROR
  only on `new_duplicated_lines_density` (11.90% against a 3% limit) once `new_security_rating` and
  `new_reliability_rating` were already OK. The G304 line takes a file-local annotation justified by the
  same containment proof that closes `gosecurity:S2083` (`fullPath` is
  `resolveWithinRoot(sanitizeUploadFileName(...))`), matching the three G304 waivers already present in that
  module, and is verified by re-running the lint job's own `--new-from-rev=<main>` invocation locally. The
  duplication was real, not noise: the new checker test file copied its twelve test bodies, 68 of the 74
  duplicated new lines. It is now a `CASES` table plus one driver loop, with the same fixtures and the same
  assertions, so the metric is fixed by removing the copy rather than by relaxing the gate.
- `test:quality-workflow` has 4 pre-existing failures asserting base's `paths-filter` / governance-only
  job shape that ops' `quality.yml` does not have; that drift is recorded, not fixed here, because fixing
  it means changing which jobs are allowed to skip.

## Post-Merge Read-Back (round 3)

- The squash-merge landed on `main` as `9522048` and the push-triggered `SonarCloud Scan` succeeded, so the
  gate was read back against the merged tree, not the PR. Open issues on `main` went **242 → 194**;
  **BUG count went 6 → 0**, and every shell / typescript / githubactions vulnerability class cleared. The
  gate condition `new_reliability_rating` is now **OK**, and `new_security_rating` is the only remaining
  ERROR. That is the honest shape of what code fixed versus what still needs triage: the code fixes worked,
  and one condition is left to the recorded-false-positive path.
- What is still open under `new_security_rating` is exactly 36 BLOCKER vulnerabilities: 33 `gosecurity:S3649`
  (the GORM literal-argument family) and 3 `gosecurity:S2083` on `os.Remove(fullPath)` in
  `filedist/storage_service.go:319/325/345`. The S2083 trio is the cleanup path of `UploadFile`: SonarGo
  cleared the `os.Create` site this round, which shows it accepts `resolveWithinRoot` as a sanitizer for the
  create flow, but it still taints the three remove calls that use the same value.
- Rather than argue the S2083 trio from the code alone, `TestUploadPathStaysInsideRootForAdversarialNames`
  now pins the exact flagged data flow: an attacker-controlled multipart name goes through
  `sanitizeUploadFileName`, is joined the way `UploadFile` joins it, and the resolved result is asserted to
  stay under the storage root — including `../../etc/passwd`, a Windows path with backslashes, an absolute
  path, `....//` doubling, a nested traversal, an embedded NUL and a CRLF name. Names that the sanitizer
  rejects are treated as equally safe outcomes, so the test does not encode a false expectation that every
  hostile name must succeed. This is the machine-checkable justification the triage record leans on; the
  S3649 family leans on `scripts/check-gorm-sql-literal.mjs`, which blocks in CI.
- No transition was performed before this test was on `main`, and no metric, new-code baseline or gate was
  touched to make the reading look better.

## Verification Plan

- `cd backend && gofmt -l modules/business && go build ./... && go vet ./... && go test ./... -count=1`
- `cd backend && golangci-lint run --new-from-rev=<main sha> ./...`, the same arguments and new-code mode
  the `Go Lint` job uses, so the hosted gate is reproduced locally instead of guessed at
- `node scripts/check-gorm-sql-literal.mjs` and `node --test tests/scripts/check-gorm-sql-literal.test.mjs`
- `cd frontend && npm run lint && npm run type-check && PANTHEON_CI=true npm run build`
- `bash -n e2e-tests/auto-test.sh e2e-tests/run-real-machine-test.sh .harness/v1.2-tasks/generate_tasks.sh`
- Governance suite: `check:structure`, `check:docs-frontmatter`, `check:task-packet-template`,
  `check:business-overlay`, `check:business-boundaries`, `check:generated-modules`,
  `check:ui-quality-gate`, `check:encoding`
- Hosted: all GitHub Actions workflows green on `main`, then a `sonarcloud.yml` dispatch on `main`,
  then `/api/qualitygates/project_status` for branch `main`.

## Linkage

- Task ID: `2026-10-30-sonar-findings-remediation`
- Task Manifest: `.harness/tasks/2026-10-30-sonar-findings-remediation/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `docs/harness/tasks/2026-10-30-sonar-findings-remediation.task.md`
- Evidence Directory: `.harness/evidence/2026-10-30-sonar-findings-remediation/`
- Review File: `.harness/evidence/2026-10-30-sonar-findings-remediation/review.md`
- Related packet: `docs/harness/tasks/2026-10-29-sonar-ci-analysis-restore.task.md`

## Evidence Required

- Sonar issue inventory before the change and the S3649 key list captured in
  `sonar-s3649-keys.json`.
- Command evidence for the Go, TypeScript, shell and governance matrices in `commands.json`.
- The S3649 transition result and the gate read-back after it in `sonar-triage.json`.
- Hosted workflow conclusions on `main` after merge.
- Review summary with the remaining gaps stated explicitly.

## Human Gates

- Credential boundary: `SONAR_TOKEN` stays a repository secret and is used only as a shell variable; no
  token value is written into the tree.
- Gate policy: the legacy smell backlog and the Dependabot scan failure are maintainer decisions and are
  recorded rather than resolved here.
- Sonar triage authority: marking 33 findings FALSE_POSITIVE changes hosted analysis state. It is executed
  only after the checker that justifies it is merged and blocking in CI, with the key list recorded so the
  transition can be reversed issue by issue.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
