# Pantheon Base v0.17.7 — Release Notes

Patch release on the `release/0.17` line. Supersedes `pantheon-base-v0.17.6`.

This release carries **thirteen inserted comment lines and nothing else**. It exists because
`pantheon-ops` cannot close three gate findings inside its own tree: the ops consumer tree is rebuilt
from a locked base snapshot, and `.foundation/allowed-drift.json` states that generic drift must flow
back to base rather than be hand-copied into `business/*`. Base-first is the only landing side.

## Summary

- **No schema change.** The migration chain still ends at `000030_tenant_plan_definitions` (schema 30).
- **No route, permission code, menu seed, i18n copy, config key or exported symbol changes.**
- **No behaviour change.** Nothing executes differently; `git diff` adds lines and deletes none.
- What changes is what the gates say about the tree: `golangci-lint` reaches 0 new findings, and the
  CodeQL alerts on base-owned files now carry their reason in the source instead of only in GitHub.

## What changed

### `backend/modules/system/config/setting/setting_service.go`

`revive` reports the `SettingService` stutter (the type is referenced as `setting.SettingService` from
outside the package). Renaming it to `setting.Service` would break every consumer's compile, including
the ops overlay's generated call sites, and that rename is not this round's scope. The finding is
suppressed in place with `//nolint:revive` plus the reason, following the repository's existing shape
for stutter suppressions.

### `backend/pkg/logging/logger.go`

CodeQL `go/log-injection` flags the four sink calls that forward a formatted message. They are not
injectable: `SanitizeLogValue` strips CR, LF, Unicode line separators and other control characters, and
`sanitizeLogFields` applies the same sanitizer to every field before it reaches a sink. Two alerts of
the same rule were already dismissed on exactly that argument. All four sinks now carry a
`// codeql go/log-injection: this is a false positive` annotation at the flagged line, because a
dismissal recorded in GitHub is keyed to the alert fingerprint rather than to the code, and a comment
that shifts line numbers must not silently re-open a settled conclusion. `TestSanitizeLogMessage` and
`TestSanitizeLogFields` are the regression cover.

### `frontend/scripts/tenant-matrix-fixture-setup.mjs`

CodeQL `js/file-access-to-http` flags seven sites in the tenant-matrix smoke fixture. The taint source
is the manifest this fixture writes itself into the gitignored `.tmp` directory, and every value read
back from it is used only to call the backend instance the fixture has just provisioned. Each site now
states that in a `// codeql js/file-access-to-http: this is a false positive` annotation.

## Compatibility

Consumers advance `foundation-release.lock.json` from `pantheon-base-v0.17.6` to
`pantheon-base-v0.17.7` and re-run `npm run rebuild:from-base`. Nothing else is required:

- No migration to run (schema unchanged at 30).
- No permission grant or menu re-seed.
- No API or DTO surface moved.
- Business overlays are untouched: all three files are base-owned snapshot paths, so a consumer that
  had hand-edited them to silence the same finding loses nothing — the rebuild now carries the
  suppression from the snapshot.

## Verification

- `go build ./...`, `go vet ./...` and `gofmt -l .` clean.
- `golangci-lint run --max-issues-per-linter 0 --max-same-issues 0 --new-from-rev=main ./...` →
  **0 issues**, which is the direct proof that the revive suppression works.
- `go test ./pkg/logging/...` passes, including `TestSanitizeLogMessage` and `TestSanitizeLogFields`.
- Frontend `tsc -b` exits 0; `check-duplication` reports `Result: PASS`.
- Governance chain green locally: docs frontmatter (363 docs / 266 with frontmatter), harness encoding
  (0 findings over 2382 files), structure contract (0 over 2720), task packet (0 errors),
  PR governance, UI quality gate, generated artifacts (0 over 11).
- CI: `CodeQL`, `Go Lint`, `Backend Tests` (`go test -race ./...` on Linux), `Docs Governance`,
  `Frontend Contract`, `Duplication Gate` and `Release Gate` as recorded in the publication facts.

## Known residual

- **SonarCloud Scan stays red on every PR run**, with
  `Failed to query JRE metadata: GET https://api.sonarcloud.io/analysis/jres → HTTP 403 Forbidden.
  Please check the property sonar.token or the environment variable SONAR_TOKEN`. The workflow has
  failed this way on every PR since 2026-09-17, so it is a credential/permission defect and not a
  regression introduced here. Repair needs a valid `SONAR_TOKEN` from the maintainer; per the standing
  rule, neither the Sonar rule set nor the project configuration was touched.
- `go test -race` cannot run on this Windows host (cgo disabled), so the race reading is the Linux CI
  job's. The generator tests in `backend/modules/lowcode` need `PANTHEON_NODE_BIN` and
  `PANTHEON_WORKSPACE_ROOT` locally, or they fail with `module.generate.server_export_failed` (the
  already-recorded environment limitation F5).
- The `SettingService` rename is deferred, not rejected; if it happens it must land as a breaking
  minor with a consumer migration note.
