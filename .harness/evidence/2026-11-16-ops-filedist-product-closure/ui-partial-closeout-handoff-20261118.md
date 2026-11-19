# E01 FileDist UI source handoff

Owner: `/root/e01_ui_partial_closeout`. Parent: `2026-11-16-ops-filedist-product-closure`. Status: **source frozen; scoped checks passed; E01 package acceptance remains open**. Claude's original resume exited 1/max-turns; its historical `continuation-ui-plan-20261118.md` label `implemented-ui-only` is preserved and is not acceptance evidence.

## Changes and boundaries

- `frontend/src/modules/business/filedist/FiledistList.tsx`: standard file/task pagination, remote bounded file choices, resolved authorized CMDB target selection, per-host fingerprints, write-only password/private-key form, schedule, permission-aware upload/delete, terminal-only task deletion, inline errors and warnings. File search/page changes clear file selection and invalidate asynchronous submit intent. Metadata lifetime is independent of submit intent.
- `createTaskPayload.ts` and `api.ts`: enforce current scope/version and eligible resolved host identities/endpoints; reject duplicate/malformed IDs or missing fingerprints; send only the selected authentication mode. No arbitrary editable host endpoint or browser persistence. Read target DTO has no authentication fields.
- `FiledistDetail.tsx` and `taskState.ts`: translated progress/results and statuses, pending cancel, partial retry, due-time start guard, single-flight control actions, route-bound completions, correct generic/404/403 initial failure classification.
- `useTaskDetail.ts`: aborted route/unmount responses cannot publish; terminal polling stops; refresh failures remain visible. `allSettled` waits for all sibling GETs after early failure, retaining the first observed error before scheduling the next poll.
- `useFiledistListQuery.ts`: query activation owns its controller. All manual callbacks use the current committed query. Old successes and errors cannot publish. Polling shares the pending GET. Foreground invalidations coalesce one serial fresh read after any already-running GET, preserving mutation freshness without overlap.
- `useFileChoices.ts`: reuse the existing bounded `useCatalogOptions` helper, 20 results per page, remote keyword search, 250ms debounce and abort guards. Pages beyond the first 100 are reachable. New dialog activation clears previous choices immediately. Items and total remain one activation-bound response.
- `filedist.css`: existing token spacing, wrapping detail toolbar and a responsive configuration grid. No shared UI styles or new dependency.

No backend, shared locale bundle, Base, dependency, manifest/status, regular unit-test or overlay declaration was edited by this lane. Shared locales are owned by `/root/extension_startup_integration`; the exact 44-key zh-CN/en-US delta is `ui-locale-delta-20261118.json`, with explicit English fallback for other locales. Regular test/overlay integration is owned by `/root/e03_claude_partial_closeout`.

## Fresh verification and frozen inputs

`ui-final-verification-20261118-03.json` records every command and actual exit, start/end UTC and all nine source SHA256 values before and after. All three commands exit 0; all nine hashes are unchanged during verification:

1. `node node_modules/eslint/bin/eslint.js src/modules/business/filedist --max-warnings 0`: zero errors and warnings.
2. `node .harness/evidence/2026-11-16-ops-filedist-product-closure/run-ui-lifetime-regression-20261118.mjs`: **28 passed**.
3. `git diff --check -- frontend/src/modules/business/filedist`: passed.

Frozen evidence fixture: `ui-lifetime.regression-20261118.test.ts`, SHA256 `0d3d66ea394e3c92dea41b5ce8fb19da5395fac4a23ff66b7a44ee54a0554256`. This is a focused fixture using controlled request futures and mocked Arco/router/translation for action and list tests, not rendered acceptance evidence. E03 received the exact hash and count for canonical unit-test integration. `run-ui-lifetime-regression-20261118.mjs` uses installed project Vitest config/dependencies, one thread worker and explicit module resolution for evidence outside the frontend root.

The parent instructed this lane to pause heavy app typecheck/build and browser work while running backend verification. The final verifier ran with `-RunSuffix 03 -SkipAppTypecheck`, which is explicit in its receipt. Earlier app typecheck passes in old receipts cover earlier source only; they do not verify this nine-file freeze. Final source diagnostics/build/render are centrally owned.

## Preserved failed evidence and repairs

- `ui-lifetime-green-20261118-01.log`: original malformed-NaN identity failure, repaired with `Object.is`; later green02 passed.
- `ui-lifetime-green-20261118-03.log`: router/mock module-resolution harness failures; fixed runner aliases, not production defects.
- `ui-initial-network-error-red-20261118-01.log`: initial connectivity failure incorrectly shown as task missing; repaired classification.
- `ui-sibling-request-overlap-red-20261118-01.log`: real sibling GET early-failure overlap; repaired with `allSettled`, then `ui-picker-sibling-green-20261118-01.log` passed all 21 cases.
- `ui-list-stale-refresh-red-20261118-01.log`: test-source parse error, no executed cases; not a production RED.
- `ui-list-stale-refresh-red-20261118-02.log`: 6 failed/21 passed. Four task cases are production RED: stale manual success, stale manual error, manual/poll overlap, and a mutation callback querying the prior keyword. Two file cases failed at a table-mock text matcher before their stale-response assertions; do not count those as production RED. The corrected name-span mock tests both file late outcomes in the final GREEN.
- `ui-list-stale-refresh-green-20261118-01.log`: same two file mock matcher failures after the production repair, 25 passed. Corrected fixture led to green02: 27 passed.
- `ui-list-mutation-revalidation-red-20261118-01.log`: real 1 failed/27 passed; a GET started before deletion left stale results because a mutation only joined it. One coalesced serial re-read repairs it; green01 and final03 pass all 28 cases.
- `ui-scoped-eslint-20261118-03.log`: one unnecessary revision dependency warning; a single activation query state makes reload identity explicit. Scoped lint04 and final03 have no warnings.

Standalone RED command wrappers displayed log tails after the runner and did not persist a separate process-exit receipt; use the failed assertions/counts in those preserved logs as RED evidence. Final verification03 does persist actual command exits. No failed log was overwritten or relabeled.

## Open acceptance items

The backend's production authorization provider remains nil/fail-closed in the inspected source; authorized runtime creation is not established by this lane. Provider integration, final app diagnostics/build, official overlay, actual SSH success/failure/cancel/restart and scheduled execution, desktop/narrow-screen rendered quality, and maintainer functional/visual acceptance remain central responsibilities.

Credential safety evidence is bounded to source: response DTOs omit password/privateKey/snapshots; authorized creation clears parent snapshot credentials and encrypts target columns; transfer failure uses a canonical message key; current events endpoint returns an empty array. This does not establish historical database cleanup, global logging/proxy sanitization, production provider admission or every native SSH path.

No E01 task is closed, no acceptance waiver is granted, and no fresh current-source app build or rendered evidence is claimed by this handoff. Root may start independent review and centralized validation against the recorded nine-file freeze.
