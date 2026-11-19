# Claude UI small fixes — evidence record

Date: 2026-11-17
Scope: low-risk build/test regression fixes. Claude edited three owned files; a fourth (this file) records evidence. No commands were run by Claude — verification is Codex's responsibility.

## Files modified

1. `frontend/src/modules/business/deploy/task/DeployTargetSelector.tsx`
2. `frontend/src/modules/business/shared/targetSelector.ts`
3. `frontend/tests/smoke/business/service/ServiceCatalog.nav.spec.ts`

## Fix 1 — missing import (build failure)

`DeployTargetSelector.tsx:31` referenced `TARGET_SELECTION_BUDGET` but the shared-hook import on line 5 only pulled `parseManualLines, TARGET_CANDIDATE_PAGE_SIZE, useBusinessTargetSelector`. A pointer to the existing constant was added to the import. No export added; `TARGET_SELECTION_BUDGET` already existed.

## Fix 2 — unused `initialHostIds` (build/typecheck failure)

The extracted shared hook destructured `initialHostIds` (`targetSelector.ts:108`) but never used it, dropping the old selector's explicit-ID initialization compatibility. The useState initializer now:

- folds `initialHostIds` into the host set alongside an in-scope `initial` (dedup),
- otherwise seeds the intent from `initialHostIds` directly,
- and still drops a cross-domain `initial` (its `businessScopeId` ≠ `scopeId`) so out-of-scope intent is never carried.

Behavior confirmed against the existing unit contract `frontend/tests/unit/business/operational/deploy-selection.test.tsx:38-46` (`initialHostIds=[11]` seeds selection `11`; a subsequent different-scope remount clears it) and the caller `DeployTaskList.tsx:830`, which passes `initialHostIds` only when in scope. Scope-switch clearing, generation/AbortController stale-response guards, and provider revalidation were left intact.

## Fix 3 — strict-mode sidebar locator collision

Third case (`sidebar click services`) failed under strict mode: the flat `.app-shell__menu-entry--level-1` locator with exact text `服务` matched both `服务目录/服务` and `Kubernetes/服务`. The flat locator was replaced by a `serviceCatalogChild` helper that scopes each of the three child clicks to the real `服务目录` group ancestor using Arco's nested `.arco-menu-inline` structure (each group's children live in its own inline element), preserving real clicks, URL/tab/back/forward/reload assertions, and the full 30-entry navigation set. No `.first()` masking, no reduced coverage.

## Verification status

- Claude did not run npm/typecheck/Playwright or any commands (per hard-stop boundary).
- Rendering, `npm run build`, typecheck, Vitest, and Playwright runs are Codex's to execute and record.
- Do not mark this task completed or claim verification until Codex records pass output.
