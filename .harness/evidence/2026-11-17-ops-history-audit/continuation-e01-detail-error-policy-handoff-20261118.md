# E01 translated detail action and page-owned GET error handoff

Owner: `/root/e03_claude_partial_closeout`; parent E01 `2026-11-16-ops-filedist-product-closure`, coordinator history audit `2026-11-17-ops-history-audit`; observed UTC `2026-11-18T09:52:16Z`.

Exactly two business sources changed and are frozen:

- `frontend/src/modules/business/filedist/FiledistList.tsx`: the existing detail button reuses `common.detail` instead of leaking `common.view`.
- `frontend/src/modules/business/filedist/api.ts`: getTaskDetail/getTaskTargets/getTaskEvents pass the already-public `skipErrorMessage: true`, following existing start/cancel requests, so the page owns their error feedback.

The Base-supplied existing key resolves to Detail / 详情 / Détail / 詳細 / 상세 across the five locales. No locale data, shared API/interceptor, permission, signal, allSettled, polling/cancel or route logic changed. Direct rejected promises and their original error/signal identities remain intact. Complete pre-edit bytes and before hashes are preserved in `continuation-e01-detail-error-policy-before-20261118-01/`.

## Actual verification

Full commands/output and source bindings are in `continuation-e01-detail-error-policy-handoff-20261118.json`.

| Check                                                             | Result                                                                                               |
| ----------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| Scoped ESLint, max-warnings 0                                     | Exit0; no diagnostics                                                                                |
| Standard-config 28-case canonical FileDist lifetime suite         | Exit0;28 passed,0 skipped;6.62s                                                                      |
| Strict targeted TypeScript using the existing regular-unit config | Exit0; no diagnostics                                                                                |
| Business API adapter check for all three detail GETs              | 3 PASS: quiet-error flag, exact URL/method, original rejected promise/error and AbortSignal identity |
| Scoped git diff --check                                           | Exit0                                                                                                |
| Original/current normalized two-source comparison                 | PASS: one existing key substitution and three GET options only                                       |

The adapter check uses installed TypeScript transpilation and a strict API mock to exercise the actual business adapter exports; it does not exercise shared interceptors or prove visible toast suppression. Canonical test/production detail hook/detail page are unchanged and hash-bound. No dependency or production test was added. The existing Vite native-loader/dirname warning remains visible and was not suppressed.

## Visual evidence and remaining gates

Impeccable was used for the existing dense task list/detail error surface. Both real 1440×900 baseline screenshots were actually inspected: task actions show raw `common.view`; the missing-task page shows two shared not_found toasts plus its own error/retry. Those exact original screenshots are preserved and hash-bound in the JSON.

Fresh rendered output for the changed sources is owned by the QA/coordinator lane. This source-only handoff does not claim corrected rendering or mobile/overall visual acceptance. The coordinator must run a new formal locked-release reconstruction, source-equivalence and centralized frontend gates; the prior proof cannot authenticate these changed sources. Then actual desktop/mobile list/detail 404/403/503/retry checks confirm translated “详情” and no duplicate global toasts.

Frozen SHA256:

| Source           | SHA256                                                             |
| ---------------- | ------------------------------------------------------------------ |
| FiledistList.tsx | `a9aaaae9d510e5e844add157dacd941f789356ddafaf12d06ecd2726c6166ac5` |
| api.ts           | `77294421105d9bedde126dd4cb73c944f2f204c4866d9f891489b0ce464a5b45` |

No whole E01/C03 or ten-package closure is claimed. Independent review and final acceptance remain with the coordinator.
