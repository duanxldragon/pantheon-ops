# Pantheon Base v0.17.3 — Release Notes

**Release date**: 2026-10-23
**Tag**: `pantheon-base-v0.17.3`
**Previous release**: `pantheon-base-v0.17.2`

## Summary

v0.17.3 is the release that makes the post-publication review of v0.17.2 real. That review compared
the published code and assets against their evidence packet and found that the credential gate
shipped in v0.17.2 did not scan `repo.tar` — the exact asset that carried the v0.17.0 residue. The
published v0.17.2 assets were scanned afterwards and are clean, but the gate, not the artifact, was
the defect. This release widens the gate, corrects the record, and stabilises the smoke test that
misread an Arco popup animation as a layout regression.

There is **no runtime behaviour change** in this release. The only backend diff is one duplicated
doc-comment line removed. The database stays at schema version 29; no route, permission code, menu
seed or error envelope is added, removed or renamed.

## What changed

| Item | Before | Now |
|---|---|---|
| Credential preflight coverage | scanned `bundle/` only; reported "clean, 768 files" without opening `repo.tar` | scans both published shapes; `repo.tar` missing fails closed |
| `repo.tar` residue path | no regression case for the tracked evidence file that shipped in the snapshot | test pins `.harness/archive/.../commands.json` inside `repo.tar` |
| Tenant-picker smoke | sampled document overflow during Arco's first-frame popup positioning; CI saw 87px | waits until the popup aligns with its trigger, then asserts the unchanged 1px bound |
| v0.17.2 evidence | three statements did not match the shipped code | erratum records the correction and preserves the original claims; FR-015 records the gap |
| `GetTenantByID` comment | duplicated doc-comment line | one line |
| `.harness/STATUS.md` | said v0.17.2 was still publishing after the tag existed | records the verified tag commit |

## Fixed

### The release gate now opens both assets

`check-release-secrets.mjs` scanned the staged `bundle/` tree, which contains the built
application, configuration templates and docs. The v0.17.0 credential lived in an archived evidence
file under `.harness/archive/`, which is part of the tracked tree and therefore ships only inside
`repo.tar` — the full-commit snapshot. The v0.17.2 gate never opened that archive and could not have
seen the residue it was introduced to prevent.

The preflight now extracts and scans both shapes, and a missing `repo.tar` is an error rather than a
silent pass. The rules and their masking are unchanged: three concrete shapes (MySQL DSN, URL
credentials, `-p<value>` on a command line), with the secret replaced by a masked form in the
report. The broader rule set that reported 42 false positives on the real v0.17.1 bundle stays out.

The regression suite adds three cases: the archived `commands.json` residue inside `repo.tar`, a
clean `repo.tar`, and a missing `repo.tar` that must fail closed. `publish-foundation-release.mjs`
calls the widened preflight after the release is cut and before the tag exists.

### Tenant picker smoke waits for the popup to settle

Arco mounts a select popup at the trigger's centre for one frame and repositions it to the trigger's
left edge on the next. On a 390px viewport the 282px popup transiently reaches 477px, so measuring
`scrollWidth - innerWidth` during the appear animation races the reposition. CI sampled 87px and
reported a regression that did not exist in the settled layout.

The test now polls until the popup's horizontal position matches its trigger's within 1px, then
runs the same overflow assertion with the same bound. A popup that stays outside the viewport still
fails.

### Evidence corrections

The v0.17.2 evidence claimed three things the code did not do: that the deadlock fix retried
`Error 1213` (it replaces `INSERT ... SELECT` with a non-locking read and insert), that the scanner
had 18 cases including PEM/JWT detection (it has three rules and had 10 cases at publication), and
that the publication gate covered the whole release (it scanned `bundle/` only). The evidence now
carries an erratum that states the corrections and keeps the original claims visible, rather than
rewriting history.

`docs/harness/failure-registry.md` gains FR-015 for the gate-coverage gap: a static sensor that
reported success because it never opened the file it was meant to inspect.

## Added

- FR-015 in the failure registry, with the widened gate and its regression tests as the sensor.

## Changed

- None in the product. `VERSION`, `CHANGELOG.md`, `README.md`, `README.en.md` and `.harness/STATUS.md`
  move to v0.17.3.

## Compatibility

- No schema change: the database stays at version 29. No migration is added or edited.
- No route, permission code, menu seed or error envelope shape is removed or renamed.
- No runtime behaviour change: the only backend diff between v0.17.2 and v0.17.3 is a comment.
- `sharedPaths` stays at 52 entries. `pantheon-ops` may stay on v0.17.2; advancing to v0.17.3 picks
  up the corrected release gate and evidence, not a new application interface.

## Verification

Measured locally:

- `npm run test:foundation-release` — 57/57, including 12 scanner assertions (repo.tar residue,
  clean snapshot, missing snapshot, masking, exit codes, real fixture forms).
- Real v0.17.2 assets scanned under the widened gate — 3,079 files across the bundle and `repo.tar`,
  0 findings.
- Frontend: `tsc --noEmit`, ESLint and production build green; the tenant-picker spec passes 3/3 in
  the smoke-core project.
- Backend: `go build ./...`, `go vet ./...`, `gofmt -l .` clean. The package-level checks touched by
  the previous packet remain green; no runtime code changed in this release.
- Governance: docs frontmatter, task packet, structure, generated-module and failure-registry
  checkers pass.

Decided by CI, not locally: `Release Gate Summary` on the merge commit, the full required-check set
on the pull request, and `Foundation Release Verify` on the tag.

## Known residual

- The v0.17.0 asset still contains the development password. Published assets are immutable and the
  credential was rotated; the widened gate prevents a recurrence, it cannot recall the artifact.
- The published v0.17.2 assets are unchanged and clean. Their notes stay as published; the
  correction of record lives in the repository evidence and in this release.
- `system_role_data_scope` and `system_role_permission` writes still pre-check instead of relying on
  the unique index, so a lost race there can return the same 500 the tenant code used to.
  `IsDuplicateKey` makes that fixable per path; only the tenant path was fixed in v0.17.2.

  > **Correction (2026-10-23).** Only the `system_role_permission` half of the sentence above was
  > accurate. `system_role_data_scope` is written with a `clause.OnConflict` upsert
  > (`upsertRoleDataScopePolicyPreservingCustomDepts` in
  > `backend/modules/system/iam/role/role_service.go`), so it never pre-checked and could not lose
  > that race. The `system_role_permission` half was true when v0.17.3 was published; v0.17.4 made
  > the grant writes idempotent and mapped the failed-race response to the existing 409. The original
  > text is kept so the published record stays auditable; the `pantheon-base-v0.17.3` tag and its
  > assets still contain it unchanged.
- `tenants.deleted_at` remains a reserved, unwritten column; contract V1.3 records `status` as the
  deletion marker.
