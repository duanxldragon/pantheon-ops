# C03 Service catalog UI implementation checkpoint

Task: `2026-11-16-ops-service-catalog-usability`.
Packet: `docs/harness/tasks/2026-11-16-ops-service-catalog-usability.task.md`.
Parent: `2026-11-16-ops-pilot-delivery`.
Status: in-progress; implementer checkpoint, not independent approval or whole-task completion.

## Implemented scope

- Application, Service and Instance catalog lists use owner server pagination and totals. Health count is explicitly current-page.
- Application and Service edits use existing PUT endpoints, excluding immutable ownership/code fields.
- Instance version edit sends expectedLifecycleVersion; currentVersion and existing binding are read-only.
- An explicit change-target switch lazily resolves the owning Service detail and its business scope, then enables a complete VM/K8s target replacement. Default version editing never queries CMDB/K8s or owner options.
- Create-form options are lazy and bounded (25 rows plus keyword for Application/Service/Host/Cluster). K8s namespace/workload selection is structured, scope/namespace/kind-filtered, and dependent values reset after context changes. Stale lookups are aborted or ignored.
- Optional-query errors remain inside the modal; service-only catalog reading does not request CMDB/K8s. Authorized buttons use existing permission checks.
- Added ten catalog UI keys and five existing owner error keys in five canonical v12 locales. Parent owns generated fallbacks and error appendix synchronization.

## Validation boundary

Scoped lint and frontend type checking are recorded in `ui-commands.json` and their receipts. One intermediate JSX conditional parse failure was corrected and is retained there. Playwright specs have six new mocked contract regressions in addition to the two existing live smoke tests: 125-object pagination/edit, read-only/no optional requests, forbidden target isolation, structured K8s scope/reset, explicit versioned rebind, and stale-conflict input preservation. Mocked tests do not prove backend authorization, owner CAS or provider behavior.

Fresh final checkpoint: scoped ESLint exit 0 (14:06:12Z), frontend `tsc -b` exit 0 (14:06:36Z), and Playwright `--list` exit 0 (eight registered tests; no test execution). Canonical JSON validation exit 0: five files parse and contain all 15 C03 keys (75 pairs). Scoped `git diff --check` exit 0. Final owned-file hashes are in `ui-source-manifest.json` (14:06:51Z; working tree, HEAD `71c86cfe664ef9fde310cc68efbf2e8992e648f2`).

Parent runtime selection for the six mocked cases: `--grep "server totals and page six|read-only catalog|denied optional target|K8s options are scope-bound|instance rebind is explicit|a stale instance update"`. Tests reuse existing admin login/session and Chinese locale setup, with parent-supplied API/web environment and `PANTHEON_EXTERNAL_WEB_SERVER=1`. Business data and targets are intercepted only in the browser; no business DB fixtures are created by these six tests. The read-only case reads the real `/auth/me` response before replacing browser permission fields.

UI surface: operational table and modal forms. Intended feel: compact, calm administration with visible ownership and actionable errors. Impeccable and its checklist applied; existing Base/Arco surfaces, controls, icons, spacing, error/loading states and shared pagination reused.

Rendered desktop/mobile and primary-state evidence, live role/owner verification, full build, candidate/rebuild binding and independent review remain owned by the parent execution lane. No preview/DB tests or process start/stop were run by this UI lane. No rendered-polish claim is made. Full C03 target-edit/CAS acceptance remains open until backend production and runtime verification pass; K8s target owner validation gaps reported by the owner lane remain separate.

No commit, push, archive, production action or foundation-lock change performed by this lane.

Gate Outcomes: scoped lint/typecheck caught one incomplete JSX conditional | none.

## Reviewer item 3 follow-up: cross-tab parent availability

The reviewer found that a zero-result Application search disabled Create Service in another tab, and a zero-result Service search disabled Create Instance. Those totals describe the filtered table, not parent availability. The two create entry points now remain available for authorized users; existing modal-only parent queries resolve available Application/Service options independently.

Minimal production diff: two `addDisabled` values changed in `ServiceCatalog.tsx`. Two parameterized browser regressions search for no matches, switch to the dependent tab, open its create modal, and confirm an unfiltered bounded parent query returns a visible option without writing any business fixture.

Exact new-case grep: `--grep "ignores an empty .* search in another tab"`. To select all eight mocked C03 cases, append `|ignores an empty .* search in another tab` to the previous six-case grep. Parent executes browser smoke, rendered checks and independent re-review; this lane only discovers tests.

The parent also assigned the five generated business fallback files. The real generator convention is `frontend/src/i18n/resources/business/<locale>.ts`, not the low-code `resources/generated/` folder. Before replacing them, all 24 candidate/live business canonical JSON files were checked, raw originals and SHA256 were saved in fresh `ui-fallback-refresh-20261117T143632701Z/`, and only the five candidate-generated fallbacks were copied. `direct-byte-binding.json` records successful direct `Buffer.equals` comparison for 24 canonical JSON files and all five refreshed fallbacks; no generated content was hand-edited.

Final seven-file SHA256 checkpoint: `ui-cross-tab-source-manifest.json` (14:37:17Z). Scoped lint/typecheck and two-case discovery receipts have the `ui-cross-tab-` prefix; the final checks after fallback refresh have `ui-cross-tab-refreshed-` prefixes. Scoped diff check exit 0. Initial validation and intermediate failed JSX diagnostics remain preserved. C03 remains in-progress pending parent runtime and independent re-review; no review approval or rendered visual completion is asserted.

Gate Outcomes: independent reviewer caught filtered-table counts incorrectly controlling cross-tab creation | none.

## Mock smoke follow-up: stale-error envelope

Parent run `continuation-c03-ui-optimizer-refreshed` passed seven mocked cases and failed the stale-instance localized-message assertion. The original run receipt, error context and trace remain under `.harness/evidence/2026-11-17-ops-history-audit/`; they bind UI hash `21299fee2258954b6c0408d7abdea71130952d2a4557af3b8679cfcb54646f7e` and the previous spec hash `2c7fa03908c9696569d7c86e6cd4704a1fbff9dd7015ba745a9e53190a55c9e9`. Earlier Vite prebundle 504 failures remain separate parent evidence.

Cause: the new fixture sent HTTP 409, `messageKey: business.service.state_stale`, and `message: Lifecycle version conflict`. The shared wire envelope only has code/data/message (`backend/pkg/common/http/response.go`); `frontend/src/api/request.ts` creates both transport and business RequestError.messageKey from envelope.message. Service update calls common.FailWithError, which uses HTTP 200 with business code 409 and the machine key in message. Trace inspection found the actual error feedback `请求失败，请稍后重试 (Lifecycle version conflict)`, consistent with the malformed fixture. Canonical frontend and backend seed sources contain the expected Chinese stale text; no shared bug or historical-seed mismatch is established by this failure.

Minimal fix is confined to the smoke spec: simulate HTTP 200 with `{ code: 409, data: null, message: 'business.service.state_stale' }`, and scope exact Chinese text lookup to `.arco-message-error`. The first match accommodates existing global/local error publishers without making the message assertion generic. The test still requires visible localized stale feedback, retained dialog and `3.0.0` input, exact lifecycle-version payload, and no visible service translation keys. UI, shared request handling, backend and canonical/generated translations were not changed in this follow-up.

Current spec hash is in `ui-stale-envelope-source-manifest.json`; UI hash remains unchanged. Fresh diagnostic receipts use the `ui-stale-envelope-` prefix. Parent executes a fresh eight-case mock run and render acceptance; this lane did not start a browser, process or DB test. Targeted discovery grep remains `a stale instance update`; the full eight-case grep is unchanged.

Gate Outcomes: browser mock smoke caught a malformed stale-error fixture outside the shared wire contract | none.

## Parent eight-case mock verification and rendered-state handoff

The parent subsequently ran `continuation-c03-ui-envelope-fixed`: exit 0, eight mocked tests passed, and all three tracked sources remained unchanged during execution. Receipt/log: `.harness/evidence/2026-11-17-ops-history-audit/continuation-c03-ui-envelope-fixed.receipt.json` and `.log`. That run binds UI hash `21299fee2258954b6c0408d7abdea71130952d2a4557af3b8679cfcb54646f7e`, API hash `6206773022d1ba4774e66cab246a2b568e42a7dd668995aac8682bfb638eee57`, and corrected spec hash `9486c5f5c97ab1ab69d84075dcb3bbb66f2fc9ecab4f90ebde25fa0d7dcb417a`. This verifies browser behavior against mocked business responses with real local login; it does not establish live business authorization, owner concurrency, providers or production readiness. Earlier failures remain preserved.

Two additional reusable tests now match `--grep "catalog desktop and mobile rendered states"`: desktop 1440x900 and mobile 390x844. Each captures and attaches five viewport screenshots through testInfo.outputPath: catalog, instance editor, localized stale error with retained version input, read-only catalog, and read-only empty state. Every capture checks body width and absence of visible service translation keys; primary creation and modal buttons must be in the viewport, version input must be focusable, stale error text and exact lifecycle-version payload remain required. Horizontal scrolling within a table is permitted. Business lists/targets and browser permission projection remain mocked; no business fixture is written.

These multistate capture tests have a 60-second per-case bound. Screenshot names begin with the exact viewport dimensions, and image/png attachments use the parent-selected fresh artifact directory. Render test execution and actual screenshot viewing are parent-owned and remain pending here; no screenshot or visual-polish claim is made by this lane.

Only the smoke spec changed in this handoff; production UI and shared files did not. `ui-render-states-source-manifest.json` binds the extended spec; `ui-render-states-` receipts record fresh lint, typecheck and two-case discovery. The earlier eight-case pass applies to its preserved corrected-spec checkpoint, rather than being presented as an execution of the new rendered tests.

## Render review follow-up: duplicate localized error feedback

Parent `continuation-c03-ui-rendered-states` passed both capture tests in 27.4 seconds (exit 0, source unchanged). Its receipt/log remain in the history-audit evidence directory and bind the pre-deduplication UI and spec. Root's actual review of the stale desktop/mobile screenshots found two identical conflict toasts; the second mobile toast sat immediately above the modal title. Passing capture checks did not close that visual defect.

Read-only request-contract inspection found no handled/messageShown flag on RequestError. Instead, default shared request interceptors show localized business/transport errors before throwing RequestError; every C03 mutation wrapper uses this default feedback configuration without skipErrorMessage. The C03 operationFailed catch was displaying the same error again.

The narrow fix returns for RequestError only in that C03 callback, with comments explicitly recording the default-wrapper assumption and denying any global handled-marker interpretation. Other exceptions still receive the local operation-failed fallback. Independent local targetUnavailable and Arco form-validation branches remain unchanged; shared request/UI code and API wrappers were not edited.

The stale regression and both rendered-state tests now wait for submission loading to finish, then require exactly one visible scoped error toast with the precise Chinese stale message. They retain the dialog/version input, lifecycle-version payload and translation-key checks. No first-match shortcut remains in these assertions.

Current two-file hashes are in `ui-single-error-source-manifest.json`; fresh diagnostics use `ui-single-error-` receipts. Parent owns fresh eight-case mock and two-case render execution plus inspection of ten new screenshots. Earlier captures, passes and the duplicate-toast finding remain preserved; final deduplication runtime/visual verification is not claimed by this lane.

Gate Outcomes: actual screenshot review caught duplicate stale-error feedback; exact single-toast regression added after submission completes | none.

## Final LOW review follow-up: count all error toasts

Parent verification of the deduplication checkpoint passed eight mocks (`continuation-c03-ui-single-error-fixed.receipt.json`) and two render cases (`continuation-c03-ui-single-error-rendered.receipt.json`), each exit 0 with source unchanged. Those receipts under the history-audit evidence directory bind UI `f748584a67685c25d219d715a904cd941cf27350561e8679d56028bab2282d97` and spec `f1cd15bbe8c7934b9e34c78ac7380c708e2819f8fbc61cfc5b6c5bc3bb9da0db`. Root reports actual inspection of all ten final screenshots; this remains mocked business UI evidence, not live backend acceptance.

The independent LOW finding was that counting only the canonical stale message could miss an extra generic error toast. Two spec-only assertions now require total `.arco-message-error` count 1 immediately after submission loading ends, alongside the existing exact Chinese count 1 and visibility checks. Retained input, lifecycle-version payload and translation-key assertions remain. C03 business UI and C04 files were not edited.

`ui-total-error-source-manifest.json` binds the frozen revised spec. Fresh `ui-total-error-lint` and `ui-total-error-discovery` receipts have exit 0; targeted discovery finds three cases (stale plus desktop/mobile render), and scoped diff check is clean. Grep remains `a stale instance update|catalog desktop and mobile rendered states`. Parent owns fresh execution and final binding of this strengthened spec; earlier green runs are not presented as its execution.

Gate Outcomes: independent LOW review caught incomplete total error-toast coverage; two total-count assertions added | none.
