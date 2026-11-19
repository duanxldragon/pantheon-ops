# C03 live host-label smoke fixture correction

Owner: `/root/e03_claude_partial_closeout`; test-only scope, one assertion in `frontend/tests/smoke/business/service/service-pages.spec.ts`.

The suite fixes the locale to Chinese in beforeEach with primeChineseLocale. Actual `continuation-c03-ui-ten-packages-20261118-live.log` failed because line 499 expected `Host #532` while the current Chinese UI correctly displayed `主机 #532`. Preserve that real exit1/log/receipt and the complete original test bytes/hash. Replace only the assertion's English label with `主机 #${hostId}`, preserving its exact dynamic ID and every other assertion, lifecycle API request and finally-cleanup path.

Require scoped ESLint, isolated strict test TypeScript, scoped diff and original/current comparison to pass. Then run the unchanged native hidden-child-preloaded `run-c03-ui-verification.mjs ten-packages-20261118-live-02 live`, retaining fresh log/receipt/artifacts. This reuses the task-owned temporary API/DB/UI lifecycle and existing asserted cleanup; do not run SSH/deploy/provider operations or change production source/other tests.

Impeccable applies to the Chinese product label, but this lane corrects a stale fixture expectation rather than changing rendered product behavior. Record actual screenshot output if the fresh live run reaches it; do not claim whole C03 closure, least-privilege/provider or production acceptance.

Observed verification limitation: the properly configured strict browser/Node test program retains TS2322 in unchanged refreshBus.ts and TS2339 in unchanged auth.ts. Record both as an explicit failed type gate, without changing shared source/helper or declaring it waived. The separately authorized live rerun completed successfully; the handoff preserves the type gap for coordinator disposition.
