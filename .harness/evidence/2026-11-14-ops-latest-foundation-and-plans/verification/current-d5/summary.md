# D5 official documentation dependency retention

Manifest implementation and fresh regression verification complete; final active installation and documentation closure remain pending the coordinator's source-frozen standard assembly.

The official v0.18.8 snapshot contains the three Base-owned documentation targets behind the active eight broken links. Ops previously replaced all of `.harness`, removing them. The manifest now replaces only `.github` and uses 168 precise Base-metadata exclusions to retain these three leaves. No shared document, official lock or assembly report was edited.

Three root-anchored `.gitignore` entries keep those rehydrated Base leaves out of subsequent Ops Git-selected overlay inputs. This is an ownership boundary: a future rebuild uses the official snapshot bytes, rather than an older installed local copy. Ops task/evidence files remain selected. The archive exclusion and legacy-review projection policy stay in force; no Base review or task manifest is imported and no historical review approval is invented.

Migration `000039_filedist_ownership_selection` is also registered as the paired business-owned source files requested by the O1 implementer. Its shared destinations must be generated through the standard assembler.

## Verification

- `node --test --test-name-pattern 'rebuilds a deterministic Base snapshot' tests/scripts/business-overlay/rebuild-from-base.test.mjs`: 1/1 passed. Actual standard assembler fixtures establish the old replacement's three missing targets, official byte inheritance, absent Ops ownership, rebuilt-consumer source-selection exclusion, retained Ops evidence, exclusion of unrelated Base reviews/manifests/state, and replacement by changed official bytes.
- `node --test tests/scripts/business-overlay/*.test.mjs`: 66/66 passed, zero failure/cancel/skip/todo. Raw output: `overlay-tests.log`.
- `git check-ignore -v <three exact paths>`: all three resolve to the intended root-anchored `.gitignore` entries.
- `node .../current-d5/verify-retention.mjs`: independently enumerates the official snapshot metadata and verifies that the manifest keeps exactly the three pinned leaves, their SHA256 values, and no Base review/task manifest. See `retention-readout.json`.
- `npm run check:business-overlay`: exit 1 because the newly declared migration39 generated targets have not yet been assembled. This is preserved in `overlay-contract.log`; no shared SQL destination was hand-written.

## Remaining gates

Final standard assembly must install the three Base leaves and migration39 destinations, generate a fresh report, and rerun active overlay/shared drift/document checks. The active eight link findings are not claimed closed by this proposal alone. The 85 historical strict-review errors are separate existing debt; this change neither hides nor resolves them.

The 168 exclusions describe the current official release's metadata siblings. Future Base releases may introduce new siblings, which must appear in candidate coverage and packaging review rather than be silently excluded by a broad `.harness` rule. This explicit release-specific list is a documented maintenance ceiling, not a runtime whitelist or gate waiver.

Gate Outcomes: actual assembly/source-selection regression caught the original loss and subsequent ownership hazard | current overlay contract accurately reports unassembled migration39 | no waiver.
