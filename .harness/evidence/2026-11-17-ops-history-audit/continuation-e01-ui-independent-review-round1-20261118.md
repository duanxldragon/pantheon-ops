# E01 UI independent review — round1

REQUEST CHANGES, two MEDIUM, no HIGH/CRITICAL. Seven files. Review2026-11-18T07:51:03Z binds historical final02 receipt2026-11-18T07:43:39.5189970Z, not later edits. Adjacent JSON preserves recorded hashes, original findings and evidence provenance. Root persists returned read-only report.

E01-UI-R1: useTaskDetail.ts51 Promise.all rejects after one GET failure while siblings may remain pending. Five-second timer at76 can start new poll before siblings' ten-second requests end. Await all batch settlement preserving original error, then timer. Regress early failure+slow sibling; current normally-slow request test misses this.

E01-UI-R2: FiledistList.tsx125/147/169 loads and poll write shared state; upload/delete/retry248/260/272/603/671 invoke no-signal old closures outside effect ownership. After keyword/page changes succeed, delayed mutation completion can fetch old query and overwrite state. Unify query generation/controller and manual/poll lifecycle. Regress late success/error and coordinated refresh.

Source safety: authorized CMDB resolution produces stable IDs/version/selection/scopeVersion; no arbitrary endpoint editor; perhost fingerprints, mutually selected authfield, scopeintent invalidation and submit mutex. DTOs omit credentials/parent snapshots; safe canonical errors. Permission/loading/error/empty/cancel/retry/future-start states reviewed. Refresh stale-data warning retains failure evidence. Fortyfour keys in five locales,88 zh/en values exact; source verification only.

Read final02 actual logs: ESLint/appTS/diff exit0,14tests PASS1.74s (old13PASS belongs prior candidate). No independent fixture rerun while source edited. Classifier mocks always false leave404/forbidden dynamic gap. First100 filepicker search/pagination amendment pending. Impeccable/checklist read, no rendered evidence. Native LSP/astgrep/CodeGraph unavailable, no APPROVE. Public job authorizer/actualSSH/cancel/restart/officialruntime/human gaps remain.

First REQUEST CHANGES retained; new freeze needs separate independent review.
