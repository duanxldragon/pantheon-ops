# Summary: SonarCloud findings remediation (2026-10-30)

## What the gate was failing on

`duanxldragon_pantheon-ops` on `main` reported 242 open issues: 174 code smells, 62 vulnerabilities and
6 bugs. The quality gate returned ERROR on two conditions, `new_security_rating` and
`new_reliability_rating`; maintainability, duplication and hotspot-review coverage were already OK. The
vulnerability/bug set was fully enumerable, so this round closed it by class instead of by count:

| Rule | Count | Disposition |
|---|---|---|
| `go:S4830`, `go:S5527` | 2 | Fixed: ArgoCD client verifies the server certificate unconditionally |
| `gosecurity:S2083` | 3 | Fixed: filedist paths confined to the storage root, upload names reduced to one element |
| `gosecurity:S3649` | 33 | Verified false positives, now backed by an enforced invariant, then transitioned in Sonar |
| `shell:S5332` | 18 | Fixed: e2e scripts require an explicit target instead of shipping a cleartext default |
| `shell:S6505` | 3 | Fixed: `npm ci --ignore-scripts`, `npx` replaced by the pinned local Playwright binary |
| `typescript:S5332` | 2 | Fixed: monitoring placeholders moved to i18n keys that state the https requirement |
| `typescript:S3923` | 1 | Fixed: `playwright.config.ts` CI/local worker branches no longer return the same value |
| `typescript:S9383` | 5 | Fixed: Arco `Popconfirm.onOk` and Select handlers await or explicitly void their promises |
| `githubactions:S8541` | 1 | Fixed: zizmor installed with `--only-binary :all:` (matching base) |

Also fixed in the same pass because they are cheap and in touched files: `go:S1871` duplicated
upgrade/rollback branches in `service/state.go`, `go:S1313` hardcoded example IP in the CMDB import
template (now RFC 5737 `192.0.2.10`), `typescript:S2681` misleading one-line bodies in
`FiledistDetail.tsx`, `typescript:S4624` nested template literal in `BackupList.tsx`, and `typescript:S2925`
fixed wait in the filedist smoke spec.

## The S3649 family became a checked invariant, not an argument

SonarGo flags every GORM `Where/Or/Order/Updates` argument that is not a source-level literal, including
package-level constants that are provably literal. Dismissing 33 findings by hand would not survive the
next analysis, so the invariant the argument depends on is now enforced:

- `scripts/check-gorm-sql-literal.mjs` walks `backend/modules/business/**/*.go`, recognises GORM receivers
  (and refuses to treat same-named methods on other receivers as sinks), and requires every SQL text
  operand of a `+` chain to be a literal, a `clause`/`gorm` expression or a package-level literal constant,
  and every map column key to be a quoted literal.
- Waivers are line-scoped, require a written justification, and are printed. An empty justification is not
  a waiver, so a waiver cannot decay into silence.
- `tests/scripts/check-gorm-sql-literal.test.mjs` pins both directions (10 tests).
- `.github/workflows/quality.yml` `docs-governance` runs the checker and its tests as blocking steps, and
  `.github/PULL_REQUEST_TEMPLATE.md` lists the command.
- `backend/modules/business/retired_modules.go` was the one place that genuinely built a clause from a
  column parameter; it now uses three package constants, matching the `condNameEquals` /
  `menuPathWhereClause` convention already used by the CMDB modules.

The two remaining waivers are in that file and are the price of keeping the helper generic; they name the
exact constants allowed, and the checker prints them.

## Review pass changed the shape of this round

An independent review of the diff before commit found five defects, all fixed rather than argued about. Two
were material:

- `sanitizeUploadFileName` only reduced Windows-style paths on Windows. `filepath.FromSlash` is a no-op on
  Linux, so the stored name kept its backslashes there and the new unit test would have failed on
  `ubuntu-latest`. Normalizing `\` to `/` before `filepath.Base` closes the containment gap on both OSes.
- The checker did not cover `Raw`, `Exec` or `Table`, which is the most direct injection surface, and its
  constant allowlist matched by name, so a reassigned `var` or a parameter shadowing a constant name was
  accepted as literal. Both were rewritten; widening the sinks is what surfaced the two additional waivers
  above, and the whole checker suite now pins 13 behaviours including those two bypasses.

The other three were a receiver heuristic too narrow to see `permissionQuery.Where(...)`, a stored ArgoCD
CA bundle that could never be cleared back to system trust through the API (now a tri-state `*string`), and
the documentation of that semantics in the ArgoCD design. Full record in `review.md`.

## Verification

Full matrices are recorded in `commands.json`. Headline results: backend `gofmt`/`build`/`vet`/`test`
green (64 packages, exit 0), new checker green with 4 visible waivers, checker tests 13/13, frontend
lint/type-check/build green, `bash -n` green for the three rewritten shell files, and the governance suite
(structure contract, frontmatter, task-packet template, encoding strict, overlay contract, module
boundaries, UI quality gate, generated modules) green.

Round 2 was driven by the first PR CI run rather than by review: `Go Lint` failed on a new gosec **G304**
(`os.Create(fullPath)`) that round 1's own rewrite introduced, and the SonarCloud gate was still ERROR, this
time only on `new_duplicated_lines_density` (11.90% against a 3% limit; 74 of 622 new lines) with the
security and reliability conditions now **OK**. The duplicated lines were in the brand-new checker test file,
whose twelve test bodies were copies of each other, so the file was collapsed into a `CASES` table plus one
driver loop — the same 12 fixtures and assertions, 13/13 still passing — instead of relaxing the gate. The
G304 line carries a file-local annotation pointing at the containment proof, matching the three G304
annotations already in that module. Re-running the lint job's own arguments
(`golangci-lint run --new-from-rev=<main> ./...`) locally returns 0 issues, the Go suite is 64 packages ok,
the checker and the duplication gate are green, and frontend was untouched in round 2.

## Post-merge read-back

PR #156 squash-merged into `main` as `9522048` with both required contexts green, the head branch was deleted
on merge, and the repository now holds only `main`. The push-triggered scan moved the numbers the way the
code should: open issues **242 → 194**, vulnerabilities **62 → 36**, bugs **6 → 0**. `new_reliability_rating`
turned OK, duplication and maintainability stayed OK, and the gate is now ERROR on a single condition,
`new_security_rating`.

That last condition is entirely made of two families, both machine-verified rather than argued: 33
`gosecurity:S3649` behind `scripts/check-gorm-sql-literal.mjs`, and 3 `gosecurity:S2083` on the
`os.Remove(fullPath)` cleanup calls in `UploadFile`. SonarGo accepted `resolveWithinRoot` as a sanitizer for
the `os.Create` site this round — that finding is gone — but it does not treat the same value as sanitized
by the time it reaches `os.Remove`, so the code fix alone did not clear the trio. Rather than assert the
containment in prose, `TestUploadPathStaysInsideRootForAdversarialNames` now runs the flagged data flow: the
seven hostile upload names (unix traversal, Windows separators, absolute path, `....//` doubling, nested
traversal, embedded NUL, CRLF) go through `sanitizeUploadFileName`, get joined as `UploadFile` joins them,
and must resolve under the storage root. Names the sanitizer rejects are accepted as safe outcomes, so the
test asserts the property instead of a hoped-for behaviour.

## Recorded false positives, and the check that carries them

Marking an issue won't-fix is only defensible after the fix ships and a fresh analysis runs against it, so the
transition happened last: PR #158 merged, the containment tests and the GORM sensor were on `main`, the checker
was re-run green there (13/13 tests), and each of the 18 sinks that were still open was read at its reported
line before any state was changed. All 36 captured keys are now `RESOLVED / FALSE-POSITIVE` and the branch gate
reads **OK** on every condition — open vulnerabilities **36 → 0**, bugs stay **0**, hotspots to review **0**.

What makes this a record rather than a suppression:

- The justification is per rule and lives in version control (`sonar-triage.json`), naming the artifact that
  re-checks it: `scripts/check-gorm-sql-literal.mjs` for the 33 `gosecurity:S3649`, and
  `TestUploadPathStaysInsideRootForAdversarialNames` plus its two siblings for the 3 `gosecurity:S2083`.
- Both artifacts block: `.github/workflows/quality.yml` runs `npm run check:gorm-sql-literal` and
  `npm run test:gorm-sql-literal` with no `continue-on-error`, and that job feeds the required `Quality Gates`
  context. If someone reintroduces a dynamic SQL argument, the sensor fails the merge instead of the finding
  silently coming back.
- It is reversible and keyed: every issue key is listed, and `POST /api/issues/do_transition
  transition=to_reopen` restores any of them. No metric, new-code baseline or gate definition was touched, and
  nothing was marked won't-fix.
- SonarCloud does not expose `POST /api/issues/comment` for this client (HTTP 404 `Unknown url`), so the written
  justification could not be attached to the issue thread; it is carried by this packet and both PR bodies.

## The maintainability residue (round 5)

The maintainer's target is `Issues = 0`, and security alone does not get there: 158 open findings remained, all
`CODE_SMELL`. This round cleared them by rule, in the file that produced each one — 109 Go findings under
`backend/modules/business`, 47 TypeScript findings under `frontend/src/modules/business`, and 2 workflow
literals. Nothing was suppressed: no `//nolint`, no `NOSONAR`, no `eslint-disable`, no `@ts-ignore`, no config
edit, and no assertion changed. The only test-file edits in the whole diff are the mechanical rename of
`recordingServiceStateCommand` and one SQL-fragment constant substitution.

How each rule was satisfied, and where the shape was forced by an existing gate rather than chosen:

- `go:S1192` → package-level `const` (never `var`, because `scripts/check-gorm-sql-literal.mjs` only accepts
  literals or package constants), except `Updates(map…)` keys, which stay inline quoted literals on purpose.
- `go:S3776` / `typescript:S3776` → guard clauses and per-case helper extraction.
- `typescript:S3358` → module-level `if/else` resolvers; `typescript:S3863` → merged imports;
  `typescript:S6571` → a named union with `(string & {})` instead of `… | string`; `typescript:S6551` →
  `typeof` narrowing instead of `String(…)`; `typescript:S7755` → `.at(-1)`; `typescript:S7770` → `String` used
  directly.
- `godre:S8205` → named struct types with identical JSON tags; `godre:S8209` → grouped same-typed parameters;
  `godre:S8193` → the temporary inlined into its condition; `godre:S8196` → agentive names for single-method
  interfaces (`InstanceCreator`, `StateTransitioner`, `ServiceInstanceStateApplier`, `LogWriter`).
- `go:S107` → `appendTaskEvent` and `runTaskStepCheck` take one purpose-built struct (`taskEventInput`,
  `taskStepHostRun`) instead of nine and eight positional parameters.
- `go:S1135` → resolved by evidence, differently for each: the ArgoCD webhook auto-sync is implemented behind an
  `applicationSyncer` seam with three tests, and the reconciler's service-instance stub was deleted because
  `biz_argocd_application` carries no service-instance association.
- `githubactions:S1192` → `EXPECTED_RESULT: success` in the aggregate steps.

Resolution was measured rather than asserted. HEAD was mapped finding-by-finding to the enclosing function with
`gocognit` and re-measured on the working tree: **27 of 27** `go:S3776` targets are at or below the threshold,
and the whole `backend/modules/business` duplicate-literal inventory went from 871 candidate clusters to 802
with **no new cluster anywhere** (the two apparent additions were this scanner matching `"automated"` inside
backtick raw strings in the new webhook test — SonarGo lexes those as one token).

Two regressions were caught by running the gates instead of trusting the fan-out, and both are the interesting
kind:

1. `tests/scripts/security-workflow.test.mjs` went from green to red, because that base-owned test pins the
   literal text `[ "${DEPENDENCY_VULNERABILITIES_RESULT}" != "success" ]`. One literal occurrence was restored
   there and only the other four became `${EXPECTED_RESULT}`; the test is green again and the cluster is still
   below the duplication threshold.
2. `golangci-lint --new-from-rev` reported two new-code issues in `deploy`: an extracted helper that was
   unreachable because its only caller `executeSSHHost` is dead code already at HEAD (`unused`), and three
   `"paused"` occurrences with `TaskStatusPaused` already declared (`goconst`). The helper was inlined back into
   its caller and the literals now use the constants; the package re-lints at 0 issues.

Verification for this round: `gofmt` clean, `go build ./...` and `go vet ./...` exit 0,
`golangci-lint run --new-from-rev=a5b5910` 0 issues, the GORM sensor green with 4 visible waivers plus 13/13
tests, `check-duplication` green, all governance gates green, frontend `eslint` 0 output, `tsc -b` exit 0,
`vitest` 230/230, `PANTHEON_CI=true npm run build` exit 0 (which is the run that caught the menu-contract
const-block coupling), and `node --test tests/scripts/*.test.mjs` failing the same 14 tests as a clean HEAD
worktree. Eleven Go tests fail against the local MySQL; every one of them fails with a byte-identical message
in that same HEAD worktree, so none is damage from this round.

## PR-side new code (round 6)

Round 5 was pushed as PR #160 and then judged against the PR analysis, which reports only new-code findings and
therefore acts as a fast per-push oracle on exactly the lines this branch touches. It returned 12 open issues;
four were real damage introduced by round 5.

- `go:S8242` — the extracted `taskStepHostRun` stored a `context.Context` as a struct field. The field is gone;
  `ctx` is a parameter of `executeTaskStep` and `runTaskStepCheck`, matching the rest of that file.
- `go:S3776` — `buildDeployExecutionSnapshot` (Sonar 18) became `defaultExecutionSnapshotSteps`,
  `buildTemplateStepSnapshot`, `resolveStepSnapshotPackage` and `validateStepSnapshotParams` plus the two
  snapshot reason constants, with every branch preserved. The three CMDB `Import` methods now share the
  base-owned `impexp` header primitives instead of repeating the header-and-row loop three more times.
- `typescript:S3776` — `AlertCenter` (20), `ServiceCatalog` (17) and `DeployTaskDetail` (25) were decomposed
  into module-level subcomponents with readonly props shapes and `FormInstance<T>` form props, the pattern the
  repo already uses in `DeployTaskList.tsx` and `CmdbLabelSchemaList.tsx`.
- `typescript:S6759` — the two `ClusterDetail` card props are now `Readonly<{…}>`.

The finding worth keeping as a rule: Sonar keys an issue instance on the text hash of its location, so editing a
statement that was marked FALSE_POSITIVE in round 4 produces a new open instance. The four previously marked GORM
sinks in `argocd_service.go` and `backup_service.go` were restored to their HEAD text and only the other
occurrences became package constants; `go:S1192` allows two inline literals per file, so both rules stay satisfied
and the post-merge re-triage shrinks to the single `ArgoCDService.UpdateApplication` instance.

Verification: `gofmt`/`go build`/`go vet` clean, `golangci-lint run --new-from-rev=a5b5910` 0 issues, GORM sensor
exit 0 with 4 visible waivers and 13/13 tests, `go test ./... -count=1` exit 0 with no `FAIL` line, frontend
eslint silent, `tsc -b` exit 0, vitest 230/230, `PANTHEON_CI=true npm run build` exit 0,
`check:harness-adoption --strict` 0 findings, and `node --test tests/scripts/*.test.mjs` still 177 pass / 14 fail —
the same 14 pre-existing base-drift failures as the HEAD baseline. The local TS complexity oracle puts every
previously flagged component at or below the threshold (max 15 across the six files).

## Explicit gaps

- The 158-finding `CODE_SMELL` backlog is addressed in round 5 rather than deferred, but the acceptance oracle
  is the next hosted analysis: local `gocognit`/duplicate-literal scans approximate SonarGo and are not the
  metric, so "Issues = 0" is only claimed after `/api/issues/search` on `main` reads it.
- Eleven Go tests failed against the developer MySQL in round 5 (`deploy` x4, `bizscope` x3, `observability` x2,
  `cmdb/host` x1, `lowcode/dynamicmodule` x1). Each reproduced with an identical message in a clean worktree of
  `a5b5910`, and the isolated `-run` form reproduced them too, so they are environment-dependent and pre-existing.
  In round 6 the same local MySQL run (`go test ./... -count=1`) exited 0 with no `FAIL` line, which supports that
  reading rather than the idea that this branch damages them; the hosted `backend-tests` job provisions a fresh
  `mysql:8.0` and stays the authority.
- Dependabot-triggered runs still cannot read `SONAR_TOKEN`, so `SonarCloud Scan` stays red on Dependabot
  PRs. Provisioning Dependabot secrets or skipping that actor is a maintainer gate-policy decision.
- The Vercel check is out of scope by maintainer instruction ("drop the Vercel task"). There is no committed
  Vercel config to change, the deployment log needs the Vercel team's credentials, and it is not a required
  context on `main`, so it neither blocks the merge nor is fixable from this host; disabling it is a GitHub
  App / Vercel dashboard action.
- `test:quality-workflow` has 4 pre-existing failures that assert base's `paths-filter` / governance-only
  job shape, which ops' `quality.yml` does not have. That base drift is recorded, not fixed here.
- No UI form exists for ArgoCD server registration, so `caCert` is API-only in this round; the response
  exposes only `hasCaCert`.
- `sonar.sources=.` still makes the scanner read PNG evidence as source text; narrowing that property is a
  Sonar configuration decision left with the maintainer.
- The gate's `new_coverage` condition (LT 80) reports OK with no value because no coverage report is uploaded,
  so it is not actually evaluated. Treating coverage as a real gate needs a Go coverage pipeline in
  `sonarcloud.yml` first; claiming a passing coverage condition here would be false evidence.
- The two `go:S1135` comments are closed, one by implementation and one by deletion. `processGitEvent` now parses
  the stored Argo CD `syncPolicy` and triggers a sync only when `automated` is present, through
  `applicationSyncer`, pinned by `argocd_webhook_test.go` (policy table, trigger set, and a failing sync still
  marks the event processed). The reconciler's `updateServiceInstanceState` stub was deleted instead of
  implemented: writing a service-instance transition needs an `InstanceID`, and `biz_argocd_application` has no
  association to a service instance, so the honest outcome is that association gap — recorded above as a
  follow-up — not a comment that pretends the path exists.
