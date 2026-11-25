# Summary: ops consumes official Base v0.19.0

## Result

The lock now names `pantheon-base-v0.19.0` (`cc2fe7370a98c2862e2610d79880012971cf57f5`). The four official assets were checked against the published digest before locking. The rebuilt tree passes overlay, drift, backend build/vet/test, frontend lint/type-check/build/unit and the script suites. Hosted CI and the maintainer's acceptance have not happened yet, so consumption is not claimed.

## What the first full materialization since v0.18.9 exposed

v0.18.9 was consumed lock-only, so many shared files were stale. v0.19.0 rebuilt them all and surfaced ops inputs that sat outside declared overlay paths:

| Problem                                                | Cause                                                                                        | Fix                                                                                                                                     |
| ------------------------------------------------------ | -------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| 12 `business.k8s.namespace.*` zh/en labels disappeared | Hand-added to the generated `i18n/resources/business/*.ts`; no source JSON had them          | Moved into `modules/business/k8s/locales/{zh-CN,en-US}.json`; regenerated output matches the old key/value set exactly in all 5 locales |
| 3 business files could not resolve `SecretTextArea`    | It lived in `frontend/src/components/`, a generic path the overlay is forbidden to own       | Moved to `modules/business/shared/`; 3 import paths updated                                                                             |
| `check:checker-wiring` red                             | Base re-pointed `check:structure`, orphaning ops' `scripts/check-structure-contract.mjs`     | Removed the orphan and its overlay declaration                                                                                          |
| `check-inheritance-contract` test red                  | Base's checker now cross-checks the doc version against the lock and requires `package.json` | Updated the ops-owned fixture                                                                                                           |
| `task-manifest-status` test red                        | The rebuild overwrote ops' archive-aware version                                             | Restored the ops version                                                                                                                |
| Undeclared ops-owned paths                             | `frontend/tests/unit/business/*`, `docs/reviews`                                             | Declared in `business-overlay.json`                                                                                                     |

## Verified

- Overlay contract OK (54 declared paths); shared drift OK (878 compared, 868 identical, 10 allowlisted).
- Backend: dependency graph, build, vet clean; full non-race tests pass (observability needs its own Redis and passes with one).
- Frontend: lint 0 errors, type-check and build clean, 54 files / 430 unit tests pass.
- Scripts: 253/253; business-overlay 59/59; the governance checks listed in `commands.json`.
- Dependency changes are all upgrades (checked key by key), no scripts lost from either `package.json`.

## Gaps (explicit)

- **Race**: local Windows ThreadSanitizer fails with allocation error 87 on 12 packages with zero assertion failures; the same error reproduces on unmodified ops main. The Linux hosted race job is the real gate.
- **Business smoke** needs the full stack and was not run locally; left to hosted CI.
- **Independent CODE/ARCH review** was not performed (`review.md`).
- **`check:harness-docs`**: 2 inherited Base docs link to Base-only artifacts. Fix is Base-first; no ops hand-edit was made.
- `check:generated`, `check:security-patterns`, `check:harness-encoding`, `check:task-packet` are red on unmodified ops main too and are not wired into CI.
- Several shared files in ops still carry earlier hand-edits that the rebuild would overwrite (the TENANT docs, the release-notes index, `RELEASE_NOTES_v0.18.7.md`). They were kept as they were so main does not regress; they are undeclared drift to settle later.
