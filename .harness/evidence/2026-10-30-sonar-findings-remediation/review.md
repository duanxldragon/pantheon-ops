# Review: SonarCloud findings remediation (2026-10-30)

Reviewer pass run against the full working-tree diff before commit. Five findings were raised; all five
were acted on. Nothing was declined without a reason.

## High

1. **`sanitizeUploadFileName` was only safe on Windows.** The unit test asserted
   `` `C:\Users\ops\report.txt` `` reduces to `report.txt`, but `filepath.FromSlash` is a no-op on Linux,
   so on `ubuntu-latest` (where the Backend Tests job runs) the function returned the whole backslashed
   string: a guaranteed CI failure and a stored name that still carried path structure.
   Fixed in `backend/modules/business/filedist/storage_service.go` by normalizing `\` to `/` before
   `filepath.Base`, which makes both separators behave the same on either OS. Re-verified with
   `go test ./... -count=1` (exit 0, 64 packages ok).

## Medium

2. **The new checker did not cover its most direct injection surface.** `TEXT_SINKS` omitted `Raw`,
   `Exec` and `Table`, so a future `tx.Exec("... " + req.X)` or `db.Table(req.Sort)` would have passed the
   gate silently while the gate claimed to lock SQL construction to literals.
   Fixed by adding all three sinks. Running the widened checker surfaced exactly two pre-existing dynamic
   shapes, both handled with visible, prose-justified waivers rather than an exemption list:
   - `backend/modules/business/bizscope/bizscope_service.go:335` - `resource.table` comes from a
     three-entry inline slice of literal table names in the same function.
   - `backend/modules/business/k8s/storage/storage.go:385` - `s.tableName(model)` falls back to the
     passed model's own `TableName()`, which is a literal declared by that model.

3. **The package-constant allowlist matched names, not declarations.** `var col = "id"; col = req.Sort;
   db.Order(col)` and a function parameter shadowing a constant name were both accepted as literal.
   Fixed by rewriting the constant collection as a package-wide line scan: only `const` declarations at
   package level (single form or `const ( ... )` block) qualify, and any name that also appears as an
   assignment target, a `:=` target, or a function parameter anywhere in the package is dropped.

## Low

4. **The GORM receiver heuristic was too narrow.** `permissionQuery.Where(...)` or `txn.Order(...)` were
   skipped. Fixed by matching the receiver identifier against a name tail (`db`, `tx`, `query`,
   `session`, `gorm`, `conn`, `q`) instead of a fixed vocabulary, while still ignoring chained calls that
   end in `)`. Same-named non-GORM methods stay excluded (`s.handler.Update(id, payload)` and the
   three-argument clientset `Update` are both pinned by tests).

5. **A stored ArgoCD CA bundle could not be cleared.** `UpdateServer` guarded on `req.CACert != ""`, so an
   operator who removed a private CA had no way back to system-trust-only through the API. Fixed by making
   `UpdateServerRequest.CACert` a `*string` with tri-state semantics (absent = keep, empty = clear,
   non-empty = replace and validate), and the semantics are now stated in
   `docs/designs/BUSINESS_ARGOCD_MODULE_DESIGN.md` section 3.

## CI round 2 (after the first PR run)

The first CI run on PR #156 was green on SonarCloud Scan, Backend Tests, Docs Governance, Frontend Contract,
Security Gates and CodeQL, and red on two things this review owns:

- **Gosec G304 on the new code** — `os.Create(fullPath)` in `UploadFile` was introduced by the round-1
  rewrite, so it counted as new code and blocked `Go Lint`, which in turn blocked the required
  `Quality Gates` context. Fixed with a file-local `#nosec G304` whose justification is the containment
  proof (`fullPath` is `resolveWithinRoot(sanitizeUploadFileName(...))`), consistent with the three G304
  annotations already in this module. Verified by re-running the lint job's own invocation locally: 0 issues.
- **New-code duplication 11.90% against a 3% gate limit** — traced to the new checker test file: twelve test
  bodies that repeated the same fixture/run/assert shape, 68 of the 74 duplicated new lines. Restructured
  into a `CASES` table with one driver loop rather than suppressing the metric; the fixture content and the
  assertions are unchanged and all 13 tests still pass. This is the honest fix: the duplication was real,
  and a copy-paste test file is also harder to extend.

Not acted on, recorded instead: the `Vercel` commit status is red and is not a required context (main
requires only `Quality Gates` and `Security Gates`, `strict: false`), so it cannot block this merge. The
maintainer dropped it from scope for this task ("delete the Vercel item"), and it is not actionable from this
host: nothing in the repository configures it, the deployment log needs Vercel team credentials, and removing
the check is a GitHub App / Vercel dashboard action.

## CI round 3 and the triage decision

- PR #158 (the filedist containment test plus the post-merge read-back) merged as `52dfe28` and the branch list
  is again exactly `[main]`. The pre-existing `Vercel` red on that PR was not treated as a signal to change
  code, because no gate on `main` depends on it.
- Decision reviewed before acting: transitioning 36 BLOCKER issues to false-positive is the kind of change that
  can quietly launder a real vulnerability, so it was ordered last — after both fixes were on `main`, after the
  sensor and its 13 tests were re-run green on that tree, and after each of the 18 still-open sinks was read at
  its reported line to confirm it is an `Updates(map)` with literal keys or a `First(&model, id)` primary-key
  lookup. Nothing was transitioned on the strength of the rule name alone.
- The API quirks that shaped the record, so a later reader does not mistake them for missing evidence:
  `/api/issues/search` ignores `issueKeys` and rejects `RESOLVED/CLOSED/REOPENED` in `issueStatuses`, so an
  already-transitioned issue looks absent unless the statuses are read with the accepted vocabulary; and
  `/api/issues/comment` is not exposed, so the written justification is carried by this packet. The first driver
  run therefore reported "0 of 36" while 18 keys were in fact already `RESOLVED / FALSE-POSITIVE` — the batch was
  resumed against the live state instead of being replayed blindly.
- What is deliberately not claimed: `new_coverage` passes only because no coverage is uploaded, the 158 remaining
  `CODE_SMELL` findings are untouched legacy backlog, and the three `os.Remove` taints are still a SonarGo
  sanitizer-propagation limitation — the code is safe because the test says so, not because the scanner agreed.

## Checked and found correct

- `veleroState(status)` produces byte-identical output to the inlined nested template it replaced.
- The `FiledistDetail` `load` reformat and the two `if … return` splits preserve semantics.
- `applyMenuFieldFilter` reproduces the original inline `Where`-then-`Or` chaining exactly.
- Merging the `TransitionActionUpgrade` and `TransitionActionRollback` cases is an exact body match, so
  behaviour is unchanged.
- The `Popconfirm onOk` awaits are safe because the handlers use `try/finally`; a rejecting handler keeps
  the dialog open rather than reporting a false success.
- `run-real-machine-test.sh` captures `TEST_EXIT_CODE` correctly under `set -e`; the pre-existing `-n`
  ping flag and the `cd e2e-tests` assumption were left alone because they are out of this scope.
- No credential or target value leaks into logs or the generated report after the IP removal.
- Dropping `page.waitForTimeout(200)` is safe because the following `expect.poll` retries the assertion.
- `workers: process.env.CI ? 1 : undefined` is a valid Playwright configuration.
- zizmor publishes wheels, so `--only-binary :all:` installs it without a source build.
- CI wiring for the two new steps is real: `docs-governance` installs root dependencies with `npm ci`
  before them, and both npm scripts exist.
