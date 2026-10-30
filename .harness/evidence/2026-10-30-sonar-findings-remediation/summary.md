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
green (64 packages, exit 0), new checker green with 2 visible waivers, checker tests 10/10, frontend
lint/type-check/build green, `bash -n` green for the three rewritten shell files, and the governance suite
(structure contract, frontmatter, task-packet template, encoding strict, overlay contract, module
boundaries, UI quality gate, generated modules) green.

## Explicit gaps

- The legacy smell backlog outside the new-code period is untouched on purpose: `go:S1192` ×68, `go:S3776`
  ×27, `typescript:S3358` ×22, `typescript:S3863` ×10, `godre:*`, `shell:S7688` ×7, `typescript:S6571` ×6,
  `go:S1135` TODO comments ×2, `go:S107` wide parameter lists ×2. Clearing it is a maintainability program,
  not a security gate, and doing it in this diff would make a security review harder.
- Dependabot-triggered runs still cannot read `SONAR_TOKEN`, so `SonarCloud Scan` stays red on Dependabot
  PRs. Provisioning Dependabot secrets or skipping that actor is a maintainer gate-policy decision.
- The external Vercel deployment check cannot be removed through the APIs available on this host; it needs
  a maintainer action in the GitHub UI.
- `test:quality-workflow` has 4 pre-existing failures that assert base's `paths-filter` / governance-only
  job shape, which ops' `quality.yml` does not have. That base drift is recorded, not fixed here.
- No UI form exists for ArgoCD server registration, so `caCert` is API-only in this round; the response
  exposes only `hasCaCert`.
- `sonar.sources=.` still makes the scanner read PNG evidence as source text; narrowing that property is a
  Sonar configuration decision left with the maintainer.
- The two `go:S1135` TODO comments are not just comments: `argocd_webhook.go:304` logs "triggering sync"
  without ever parsing the auto-sync policy, and `argocd_reconciler.go:183` only logs instead of updating
  service instance state through the capability interface. Both conflict with this repository's
  "extension points must not fake success" principle. Implementing them needs policy parsing and
  capability wiring, which is feature work; recording it is the honest option, not deleting the TODO.
