# C03 catalog option activation lifetime

Parent: 2026-11-16-ops-service-catalog-usability. L2; business/service frontend only. This is a bounded continuation of the original form/target-invalidity scope, not closure of the backend cross-owner target-lifetime or NamespaceBinding/environment/action contract.

## Confirmed finding

The existing useCatalogOptions hook hides state when disabled and aborts superseded effects, but identifies cached results only by lookup function. Closing and reopening a form with the same lookup callback immediately returns the prior rows/loading=false until the next 250ms debounce clears them. That can expose stale candidates from a prior form activation before the current lookup completes. Existing lookup-change and aborted-response checks do not cover activation identity.

## Minimum change

Extract the existing hook into a business-local useCatalogOptions.ts without changing its initial behavior, then bind result state to a memoized enabled/lookup request identity. A new disabled-to-enabled activation has a distinct identity even when the callback is unchanged, so prior rows remain unavailable and loading remains true until this activation returns fresh options. Preserve the existing 250ms debounce, abort cleanup, error isolation and bounded owner query callers. No shared hook, dependency, backend/API, locale or visual layout change.

## Validation and ownership

Generator owns ServiceCatalog.tsx and the business-local hook. Request a narrowly owned tests/unit/business/service/useCatalogOptions.test.ts using installed Vitest/jsdom/renderHook. Meaningful regression: reopen unchanged callback after a successful lookup, require old options absent and pending state until fresh response; controls cover disabled no-query and late aborted/superseded result rejection. Run only this focused threads-pool unit file and scoped ESLint/syntax/diff checks; preserve actual RED/GREEN receipts. Root centralizes full lint/type/build, business smoke and desktop/mobile rendered evidence after all concurrent writers freeze.

Execution adjustment: root was unavailable with a rate-limit error before approving unit/business ownership; the current business-overlay does not own that path. Keep the regression in this task's evidence folder and use the installed Vitest programmatic API with its existing frontend config, repository root, exact single-file include, threads pool and one worker. No shared config or overlay change. Root must relocate/register the test before claiming normal unit-suite coverage. Original hook: RED 2 failed/5 passed; corrected hook: GREEN 7 passed. Scoped source ESLint, isolated hook strict noEmit typecheck, runner syntax and tracked source diff checks passed. Full project and rendered acceptance remain deferred.

Use impeccable: preserve compact operational form layout, loading/error visibility and existing translated feedback. No UI polish or actual rendered claim from this hook unit test. Original C03 paging/edit/read-only and backend target/role/provider acceptance remain separate. Provide source hashes, precise unit checks and an empty locale delta to root; do not self-approve, edit task status or claim whole C03 completion.
