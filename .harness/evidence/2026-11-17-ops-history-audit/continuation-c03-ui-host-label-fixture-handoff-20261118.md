# C03 live smoke fixture host-label correction

Owner: `/root/e03_claude_partial_closeout`; observed UTC `2026-11-18T08:36:03Z`; test-only scope.

Only line 499 of `frontend/tests/smoke/business/service/service-pages.spec.ts` changed from `Host #${hostId}` to `主机 #${hostId}`. The suite sets Chinese in beforeEach via primeChineseLocale. The exact dynamic host ID, all other assertions/API requests and the full finally cleanup are preserved. The normalized original/current comparison passes with this single replacement.

The coordinator's real `continuation-c03-ui-ten-packages-20261118-live.log` exit1 is preserved: expected `Host #532`, actual `主机 #532`; sourceUnchanged true and no cleanup failure reported. This RED records an outdated translated fixture expectation, not a product-label regression. Original test bytes remain at `continuation-c03-ui-host-label-fixture-before-20261118-01.spec.ts.before`, SHA256 `70fe8d4176011e72f902ac3ebf454f005b5e769d463df9fa5ff7b17eed1237dd`. Corrected test SHA256: `ad0e5c0b32e3f78f47c620d3c23e7b7666d28e04861d312c10720b8f50f1c526`.

## Fresh results

| Check                                                                | Actual result                                                 |
| -------------------------------------------------------------------- | ------------------------------------------------------------- |
| Scoped ESLint, max-warnings 0                                        | Exit0; no output                                              |
| Scoped git diff --check                                              | Exit0; no output                                              |
| Original/current normalized one-line comparison                      | PASS; cleanup and all other assertions unchanged              |
| Strict test TypeScript with inherited app/Vite and Node declarations | Exit1; two unchanged transitive diagnostics, documented below |
| Existing live runner, new suffix ten-packages-20261118-live-02       | Exit0; 1 passed, 0 skipped;5.7s; sourceUnchanged true         |

The live runner used the existing hidden-child preload and unchanged task-owned local admin/API/UI lifecycle. It created temporary scope/CMDB host/application/service/VM instance, asserted the resulting catalog UI, and completed every original finally-cleanup assertion. The new log, receipt and screenshot artifacts are retained; no previous evidence was overwritten. No SSH/deploy/provider execution occurred.

## TypeScript gap retained

The first ad-hoc typecheck lacked Vite/CSS/import-extension configuration and mixed browser/Node declarations. Its actual failed output is retained in the adjacent JSON (tool output was truncated, explicitly marked). Subsequent configuration results are retained too.

The final check uses the inherited app configuration, frontend vite-env, Node typeRoots and allowJs for the existing mjs helper, with strict/noEmit. It reports only:

- `frontend/src/core/refresh/refreshBus.ts:174` TS2322: Node Timeout versus browser number.
- `frontend/tests/smoke/helpers/auth.ts:68` TS2339: refreshToken absent from the existing mjs return-shape inference.

There is no diagnostic on the changed smoke fixture. Shared source/helper corrections are outside this narrow ownership and were not made. Targeted test TypeScript is not claimed green; this remains an explicit gap for the coordinator.

## Rendered evidence and boundary

The fresh desktop screenshot was actually inspected and shows the service-instance target `主机 #533`, plus the temporary application/service/instance totals. It is `continuation-c03-ui-ten-packages-20261118-live-02-artifacts/smoke-business-service-ser-26279-plete-the-catalog-lifecycle/service-catalog-desktop.png`. Impeccable evidence scope is the existing Chinese label in its rendered table; this lane changed no product layout and claims no overall visual polish or mobile acceptance.

Production sources, other tests, generators and cleanup logic are untouched. The corrected single test is frozen. Whole C03 acceptance, least-privilege/provider proof and the explicit targeted-TypeScript gap remain separate. All command output and source/log/receipt/screenshot hashes are in `continuation-c03-ui-host-label-fixture-verification-20261118.json`.
