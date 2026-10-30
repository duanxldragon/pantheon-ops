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

## Explicit gaps

- The legacy smell backlog outside the new-code period is untouched on purpose: 158 open findings, all
  `CODE_SMELL` — `go:S1192` ×68, `go:S3776` ×27, `typescript:S3358` ×20, `typescript:S3863` ×10, `godre:*` ×10,
  `typescript:S3776` ×6, `typescript:S6571` ×6, `typescript:S6551` ×3, `go:S107` ×2, `go:S1135` TODO comments ×2,
  `githubactions:S1192` ×2, `typescript:S7770` ×1, `typescript:S7755` ×1 (facets read back from
  `/api/issues/search` on `main` after the transition, so this is the post-change count rather than the
  pre-merge one). Clearing it is a maintainability program,
  not a security gate, and doing it in this diff would make a security review harder.
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
- The two `go:S1135` TODO comments are not just comments: `argocd_webhook.go:304` logs "triggering sync"
  without ever parsing the auto-sync policy, and `argocd_reconciler.go:183` only logs instead of updating
  service instance state through the capability interface. Both conflict with this repository's
  "extension points must not fake success" principle. Implementing them needs policy parsing and
  capability wiring, which is feature work; recording it is the honest option, not deleting the TODO.
