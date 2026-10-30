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

## Recorded False Positives (round 4)

- Ordering was the safety property: PR #158 merged as `52dfe28`, then the sensor and its 13 tests were re-run
  green on that tree, then every still-open sink was read at its reported line, and only then were the 36 keys
  transitioned. `node scripts/check-gorm-sql-literal.mjs` exits 0 on `52dfe28`,
  `node --test tests/scripts/check-gorm-sql-literal.test.mjs` is 13/13, and
  `go test ./modules/business/{filedist,service,cmdb,cmdb/host,cmdb/group,cmdb/label,deploy}/...` is green,
  which are the packages holding the reported sinks.
- The 18 issues still open at the start of this round are 12 `Updates(map)` calls whose map keys are assigned
  from quoted string literals inside the same function (`deploy_credential.go:112`, `host_service.go:219/460`,
  `host_state.go:117`, `deploy_service.go:341/1906`, `service.go:175/337/464`, `state.go:79`,
  `group_service.go:149`, `label_service.go:207`) and 6 `First(&model, id)` primary-key lookups that GORM sends
  as bind parameters (`deploy_credential.go:182`, `service.go:292/569`, `deploy_template_service.go:123/393`,
  `group_service.go:277`). That is the exact invariant the sensor asserts, so the transition is a recorded
  justification pointing at a blocking check, not an opinion about the rule.
- Read-back after the transition: all 36 keys are `RESOLVED / FALSE-POSITIVE` (33 `gosecurity:S3649` +
  3 `gosecurity:S2083`), open **VULNERABILITY 36 → 0**, **BUG 0**, hotspots to review **0**, and the branch gate
  reports **OK** on every condition — `new_security_rating` is no longer ERROR. The remaining 158 open findings
  are all `CODE_SMELL` legacy backlog and were left alone on purpose.
- Reversibility and honesty limits recorded in `.harness/evidence/2026-10-30-sonar-findings-remediation/sonar-triage.json`:
  every key is listed so `transition=to_reopen` restores it; `POST /api/issues/comment` is not exposed by
  SonarCloud (HTTP 404 `Unknown url`), so the written justification lives in this packet and the PR bodies rather
  than in the issue thread; `/api/issues/search` ignores `issueKeys` and only accepts
  `OPEN,CONFIRMED,FALSE_POSITIVE,ACCEPTED,FIXED` in `issueStatuses`, which is why a resumed run must match keys
  client-side instead of trusting a filtered query; and `new_coverage` reads OK only because no coverage report
  is uploaded, so it is not an evaluated condition and is not claimed as one.
- The driver script was kept as a scratch tool outside the repository and is not committed: it authenticates with
  a maintainer token from the environment, and a token-using script inside the tree would be an attack surface
  rather than evidence.

## Code-Smell Backlog (round 5)

- Target: the 158 findings left after the round-4 triage, all `CODE_SMELL` on `main`
  (`go:S1192` x68, `go:S3776` x27, `typescript:S3358` x20, `typescript:S3863` x10, `typescript:S6571` x6,
  `typescript:S3776` x6, `godre:S8196/S8205/S8209/S8193` x10, `typescript:S6551` x3, `go:S107` x2,
  `go:S1135` x2, `githubactions:S1192` x2). 109 are Go under `backend/modules/business`, 47 are TypeScript
  under `frontend/src/modules/business`, 2 are the workflow gate scripts.
- Work was partitioned by directory so no two writers share a file: k8s + filedist, deploy, service + cmdb,
  cicd + observability, and two frontend batches. The cross-cutting classes were kept out of the fan-out
  because they need repo-wide reference updates: the four `godre:S8196` single-method interface renames,
  the two `go:S1135` TODOs, and the two workflow literals.
- Every agent worked under the same constraints: behaviour-preserving extraction only, no `//nolint`,
  no `NOSONAR`, no `eslint-disable`, no `@ts-ignore`, no config or gate edit, no weakened assertion, and
  `node scripts/check-gorm-sql-literal.mjs` must stay green, which is what forces extracted SQL to be a
  package `const` rather than a `var`.
- `go:S1192` is cleared by moving the duplicated fragment to a package-level constant, which is the shape
  the SQL-literal sensor already accepts; `Updates(map…)` keys stay inline quoted literals on purpose, because
  a constant map key would break that sensor.
- `go:S3776` and `typescript:S3776` are cleared by extracting guard-clause and per-case helpers, not by adding
  nesting suppression; `typescript:S3358` by module-level `if/else` resolvers, `typescript:S3863` by merged
  import statements, `typescript:S6571` by a named union type instead of `… | string`, `typescript:S6551` by
  narrowing a form value with `typeof` instead of `String(…)`, `typescript:S7755` by `.at(-1)`,
  `godre:S8205` by named struct types with identical JSON tags, `godre:S8209` by grouping same-typed
  parameters, and `godre:S8193` by inlining a temporary into the condition that used it.
- Two gate couplings were found by the fan-out and are worth keeping as rules. First, `githubactions:S1192`
  was cleared by declaring `EXPECTED_RESULT: success` in the job env and comparing against it, which keeps the
  fail-closed behaviour (an unset variable yields an empty string and every comparison mismatches). That edit
  initially turned `tests/scripts/security-workflow.test.mjs` from green to red, because that base-owned test
  pins the literal text `[ "${DEPENDENCY_VULNERABILITIES_RESULT}" != "success" ]` in the dependency-vulnerability
  branch; the fix was to leave exactly one literal occurrence there and parameterize only the other four, which
  is below the duplication threshold and keeps the test green. The lesson is that a workflow literal is not free
  text — a governance test may be asserting on its exact shape. Second,
  base's `frontend/scripts/check-menu-contract.mjs` resolves a seed field that points at a Go identifier only
  through a package const table built from `name = "value"` lines; extracting `"business.service"` into an
  inline `const serviceModuleKey = …` therefore made the `/business/service` menu read back an empty module and
  `PANTHEON_CI=true npm run build` failed. The ops side was fixed by declaring that const inside a `const (…)`
  block, because the checker is a base-owned inherited file and a generic fix must go base-first rather than
  being hand-copied here.
- The two `go:S1135` pseudo-implementations were resolved differently, and the difference is deliberate.
  `processGitEvent` now parses the stored Argo CD `syncPolicy` and triggers a real sync only when
  `automated` is present, through an `applicationSyncer` seam satisfied by `*ArgoCDService` so the decision is
  testable without a server; `argocd_webhook_test.go` pins the parse table (8 cases, including
  `{"automated":false}` and a truncated column, both of which fail closed) and the trigger set (automated,
  manual, no policy, and an `Error` application), including that a failing sync still marks the event
  processed. The reconciler's `updateServiceInstanceState` stub was deleted instead of implemented: writing
  a service-instance transition needs an `InstanceID`, and `biz_argocd_application` has no association to a
  service instance, so implementing it would have meant a schema change plus API and UI work inside a
  code-smell cleanup. That gap is recorded in the manifest as a follow-up rather than hidden behind a comment.
- Local verification against the real MySQL from `.env` reproduces 11 failures in 5 packages
  (`deploy` x4, `bizscope` x3, `observability` x2, `cmdb/host` x1, `lowcode/dynamicmodule` x1). Each one was
  re-run in a clean worktree of `a5b5910` and fails with a byte-identical message there, including the four
  `deploy` cases that a first pass looked like refactor damage (`load task host: record not found`,
  `business.deploy.task.sshConnectFailed`, `… invalidDeleteState`, `… precheck failed: step failed`), so they are
  environment-dependent pre-existing failures and not regressions from this round; the isolated
  `-run` form reproduces them too, which rules out cross-package interference on the shared local schema. The
  hosted `backend-tests` job, which provisions a fresh `mysql:8.0`, stays the authority for them. The same HEAD
  worktree is the baseline for the governance suites: `node --test tests/scripts/*.test.mjs` fails 14 tests and
  `check:harness-generated` fails, identically before and after this round, so those two reds are not
  attributed to it either.
- Resolution was measured, not asserted. HEAD was mapped finding-by-finding to the enclosing function via
  `gocognit`, then the working tree was re-measured: all 27 `go:S3776` targets are at or below the threshold
  (SonarGo never reports a cognitive complexity higher than `gocognit`'s, so `gocognit ≤ 15` is a sound
  local oracle), and a duplicate-literal scan of `backend/modules/business` went 871 candidate clusters to
  802 with no new cluster appearing anywhere. The two apparent new clusters were tokenizer artifacts of the
  scan matching `"automated"` inside backtick raw strings in the new webhook test; SonarGo lexes those as one
  token, so nothing was introduced.
- The acceptance oracle is still the next hosted analysis, not these local numbers: local linters approximate
  SonarGo, and only `/api/issues/search` on branch `main` decides whether the counter reaches zero.

## PR-Side New Code (round 6)

- Round 5 was judged against a hosted oracle round 4 had not used: `/api/issues/search?pullRequest=160` returns
  the new-code findings for the PR analysis, so a push gives a fast read on exactly the lines this branch
  changes. It returned 12 open issues, and four of them were regressions that round 5 introduced.
- `go:S8242` at `deploy_service.go` was a genuine design smell: the extracted `taskStepHostRun` struct stored a
  `context.Context` as a field. The field is gone and `ctx` is threaded through `executeTaskStep` and
  `runTaskStepCheck` as a parameter, which is what the rule asks for and what every other call path in that
  file already does.
- `go:S3776` still fired on `buildDeployExecutionSnapshot` (Sonar 18) and on all three CMDB `Import` methods.
  The snapshot builder became four named helpers plus two reason constants
  (`deployStepSnapshotPackageDefault`, `deployStepSnapshotTemplateDefault`), preserving each branch: the
  missing-template, empty-template, per-step package resolution, component `action` parameter and the
  `TemplateID/Name/Version` carry-over. The three imports now share the base-owned `impexp` primitives
  (`BuildHeaderIndex`, `RequireHeaders`, `ForEachDataRow`, `AppendImportError`) with per-module
  `parse…ImportRows` helpers, instead of three near-identical copies of the header-and-row loop — a fourth copy
  would have raised `new_duplicated_lines_density` on the same analysis.
- `typescript:S6759` on `ClusterDetail.tsx` was fixed by annotating the two extracted card props as
  `Readonly<{…}>`. Three components whose declarations round 5 left in place still exceeded the complexity
  threshold on the PR analysis (`AlertCenter` 20, `ServiceCatalog` 17, `DeployTaskDetail` 25 at HEAD), and were
  converted to module-level subcomponents with readonly props shapes and `FormInstance<T>`-typed form props —
  the pattern `DeployTaskList.tsx` and `CmdbLabelSchemaList.tsx` already use. The extraction is render-identical
  and pinned by the 230-test unit suite.
- The round-4 FALSE_POSITIVE marks do not survive a rewrite of the marked statement: Sonar keys an issue
  instance on the text hash of its location, so replacing `Where("id = ?", id)` with a constant produces a
  new open instance at that line. Five marked sinks were candidates for restoration, and the binding constraint
  turned out to be `go:S1192` itself: the rule counts the constant declaration as an occurrence of the literal, so
  a file that declares the SQL constant may keep at most one inline copy of it (declaration plus two inline uses is
  three, which re-raises the smell — exactly what the first PR read-back reported on `argocd_service.go:17` and
  `backup_service.go:27`). Three sinks are now restored verbatim (`ArgoCDService.UpdateServer`,
  `BackupService.UpdateStorageLocation`, `NotificationService.UpdateChannel`; the last sits in a file that declares
  no such constant, so it costs nothing) and three stay on package constants (`ArgoCDService.UpdateRepository`,
  `ArgoCDService.UpdateApplication`, `BackupService.UpdateSchedule`). Those three are the unavoidable post-merge
  re-triage, each justified by `scripts/check-gorm-sql-literal.mjs` and recorded per key so it stays reversible.
- Verification this round: `gofmt`/`go build`/`go vet` clean, `golangci-lint run --new-from-rev=a5b5910` →
  0 issues, `scripts/check-gorm-sql-literal.mjs` exit 0 with its four visible waivers and 13/13 checker tests,
  `go test ./... -count=1` exit 0 with no `FAIL` line — the 11 MySQL-dependent failures reported in round 5 did
  not reproduce on this run, which is consistent with them being environment-sensitive rather than caused by
  the refactor. Frontend: eslint silent, `tsc -b` exit 0, 230/230 vitest, `PANTHEON_CI=true npm run build` green.
  The local TS-AST cognitive-complexity oracle (calibrated against Sonar HEAD values, deliberately over-counting
  JSX logical operators) puts every previously flagged component at or below the threshold:
  DeployTaskDetail 25 → 9, AlertCenter 20 → 8, ServiceCatalog 17 → 5, CmdbLabelSchemaList 27 → 12,
  ClusterDetail 49 → 10, WorkloadList 26 → 15.
- Residual gap for this round: the PR view shows only new code, so issues on unchanged lines are not visible
  until the squash merge triggers the `main` analysis. That read-back, and the single S3649 re-triage, are the
  next step.

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
- Triage: read the live set with `/api/issues/search` (accepted `issueStatuses` vocabulary is
  `OPEN,CONFIRMED,FALSE_POSITIVE,ACCEPTED,FIXED`, `issueKeys` is ignored so keys are matched client-side),
  `POST /api/issues/do_transition transition=falsepositive` per key, then re-read the gate, the open
  `VULNERABILITY,BUG` count and `/api/hotspots/search?status=TO_REVIEW`, and write the whole exchange to
  `sonar-triage.json` including the HTTP code per transition.

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
