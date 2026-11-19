# C03 catalog-options regular-unit integration receipt

Parent: `2026-11-16-ops-service-catalog-usability`; implementer `/root/e03_claude_partial_closeout`; observed UTC `2026-11-18T07:42:38Z`.
Plan: `../../tasks/2026-11-16-ops-service-catalog-usability/continuation-regular-unit-integration-plan-20261118.md`.

The existing seven activation/error/abort regressions now live at `frontend/tests/unit/business/service/useCatalogOptions.test.ts`, covered by the standard Vitest `tests/unit/**/*.{test,spec}.{ts,tsx}` glob. The exact new test path is added to `business-overlay.json`. The original evidence fixture and manual runner are retained. A normalized byte comparison proves the regular test differs from the original fixture only in its source import. Production `useCatalogOptions.ts` and ServiceCatalog were not modified.

## Actual verification

All Node entries used the existing `hide-child-windows.mjs` preload; Vitest used its threads pool with one worker and no file parallelism. No new hidden-process implementation, shell shim or dependencies were introduced. This records the launch configuration, not a new desktop-window visibility observation.

| Command                                                                                                                                                                                                                                                                                   | Working directory | Actual result                                                                  |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------- | ------------------------------------------------------------------------------ |
| `node --import ../.local/manual-preview-20261116/hide-child-windows.mjs node_modules/vitest/vitest.mjs run tests/unit/business/service/useCatalogOptions.test.ts --pool=threads --maxWorkers=1 --fileParallelism=false --reporter=verbose`                                                | frontend          | Exit0; standard config/default include; 1 file,7 tests passed,0 skipped;1.71s. |
| `node --import ../.local/manual-preview-20261116/hide-child-windows.mjs node_modules/typescript/bin/tsc --ignoreConfig --noEmit --skipLibCheck --strict --module esnext --moduleResolution bundler --jsx react-jsx --target es2023 tests/unit/business/service/useCatalogOptions.test.ts` | frontend          | Exit0; no diagnostics; isolated test and imported hook only.                   |
| `node --import ../.local/manual-preview-20261116/hide-child-windows.mjs node_modules/eslint/bin/eslint.js tests/unit/business/service/useCatalogOptions.test.ts --max-warnings 0`                                                                                                         | frontend          | Exit0; no diagnostics.                                                         |
| `node --import ./.local/manual-preview-20261116/hide-child-windows.mjs scripts/business-overlay/check-business-overlay.mjs`                                                                                                                                                               | repository        | `OK business overlay contract (54 declared paths, rebuilt tree)`.              |
| scoped `git diff --check`                                                                                                                                                                                                                                                                 | repository        | Exit0.                                                                         |
| Original/regular fixture comparison excluding import and line-ending representation                                                                                                                                                                                                       | repository        | PASS; original evidence fixture preserved.                                     |

The unit run emitted the pre-existing Vite configuration warning about `__dirname` and a future native config loader. This lane did not modify shared `vitest.config.ts` or suppress that warning. The overlay contract check is not an official locked-release reconstruction or drift proof.

The seven cases cover immediate old-row removal on reopen, old-denial removal pending fresh authorization, initially-disabled no-query, close-before-debounce cancellation, ignored late resolve/reject after abort, and lookup-change/current-error behavior. They retain the prior real RED→GREEN evidence in `target-lifetime-ui-handoff-20261118.json`; this integration does not manufacture another RED.

## Evidence boundary

Impeccable was applied to the form option loading/empty/error/stale-result states. No layout or production UI changes were made, and no desktop/mobile rendering was produced by this test-integration lane. No visual polish is claimed. Backend target lifetime, NamespaceBinding/environment/action, actual roles/provider, C02 dependencies, full frontend/build/smoke, official reconstruction and whole C03 acceptance remain open with the coordinator.

Current test/overlay and unchanged production hook/original fixture SHA256 values are in the adjacent JSON. Owned test and manifest declaration are frozen for coordinated assembly.
