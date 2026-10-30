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

## Round 5: the smell fan-out, and what the gates caught

- Partitioning was by directory so no two writers shared a file (k8s + filedist, deploy, service + cmdb,
  cicd + observability, two frontend batches). The cross-cutting classes stayed with the orchestrator because
  they need repo-wide reference updates: the four `godre:S8196` interface renames, the two `go:S1135` TODOs and
  the two workflow literals.
- Two independent adversarial audits were run over the agent diffs, plus a line-by-line personal read of the
  highest-risk extractions (`service/state.go`, `cmdb/host/host_state.go`, the seed files, the CMDB/group/label
  import paths and every frontend helper lift). Both audits returned no blocker: extracted constants are
  byte-identical to the literals they replace, error-append ordering is preserved, optimistic-lock and CAS
  guards are unchanged, hook dependency lists are unchanged, React helpers moved to module scope rather than
  into the component body (so no remount identity change), and no assertion was weakened.
- One agent hit its turn budget with an unknown completion state, so its package was re-derived from
  measurement rather than trusted: `gofmt`/`build`/`vet` clean, all nine `go:S1192` literals gone, all eleven
  `go:S3776` targets under the threshold, `go:S107` and `godre:S8209` fixed. That suspicion was justified — the
  full new-code lint then found two issues in it that review alone had missed (an unreachable extracted helper
  and a `goconst` duplicate), both fixed and re-linted to 0.
- The `applicationSyncer` seam was reviewed for the shape of the failure mode, not just the happy path:
  `{"automated":null}` and `{"automated":false}` must both be false, a truncated `syncPolicy` column must be
  false, and a sync that returns an error must still mark the webhook event processed, otherwise a transient
  Argo CD outage would re-drive the same event forever. All four are pinned by tests.
- Deleting the reconciler stub was checked against the schema before it was accepted as the honest option:
  `biz_argocd_application` has no service-instance column, so the alternative was inventing an association.
  The gap is recorded in the manifest instead.
- What is deliberately not claimed here: that the local Go suite is green (11 failures reproduce identically at
  HEAD), and that the backlog is cleared in the metric rather than in the code — only the next hosted analysis
  decides that.

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

## Round 6: the PR analysis as an independent reviewer

- The PR analysis is the only hosted oracle available before the merge, and it behaved like a reviewer: it
  rejected two of round 5's own refactors (`go:S8242` on the struct-held context, `go:S3776` on a snapshot
  builder that had only been partially decomposed) that every local gate had accepted. Local `gocognit` is
  conservative for Sonar, but it cannot see the struct-field-context rule at all, so "all local gates green" was
  not evidence of new-code cleanliness.
- The component extractions were reviewed for render equality rather than for compiling: the JSX, the conditional
  expressions, the callbacks and the `pagination`/`scroll` table props are unchanged, `TaskActionsCard` still
  receives the same seven permission booleans (now produced by `resolveTaskActionPermissions`, whose `!task`
  branch returns an all-false constant instead of seven `Boolean(task && …)` expressions), and `buildTaskDetailItems`
  is now a concatenation of the profile and execution halves in the original order. The `taskFlowItems` memo
  dependency moved from the derived status string to `task`, which recomputes more often and yields the same
  value.
- `resolveTaskActionPermissions` was checked status-by-status against the round-5 expressions it replaces,
  including the deliberate asymmetry that `canDeleteTask` excludes the three request states
  (`running`, `pause_requested`, `cancel_requested`) rather than allow-listing terminal ones.
- Restoring the FALSE_POSITIVE-marked sink statements is a trade, not a fix, and it is recorded as one: those
  lines are literal SQL again because rewriting them would silently re-open hosted security findings that were
  closed with a machine-checked argument. The bound is `go:S1192` counting the constant declaration as an
  occurrence, so only one restored sink per declaring file is possible, and three provably-literal sinks stay on
  package constants and must be re-triaged after the merge. The trade is acceptable only while
  `scripts/check-gorm-sql-literal.mjs` stays blocking, because that sensor is what prevents the restored text
  from drifting into a genuinely dynamic clause later.
- What is still not claimed: that the merged `main` analysis is at zero. The PR view cannot show findings on
  unchanged lines, so the post-merge read-back remains the acceptance step.
