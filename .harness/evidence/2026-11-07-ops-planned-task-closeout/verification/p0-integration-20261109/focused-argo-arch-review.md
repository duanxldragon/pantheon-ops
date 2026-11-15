## Summary

**Architectural Status: WATCH** for this bounded business-domain delta. The supplied source correctly separates HTTP acceptance from completion, distinguishes definite rejection from uncertain outcomes, and excludes provider failure bodies. One concrete error-key defect and a persistence failure limitation remain; neither establishes a blocking defect in the normal outcome classification.

This verdict does **not** approve the entire P0 or release.

## Analysis

1. **Acceptance remains nonterminal.** History starts as `Running`; successful submission leaves `FinishedAt` unset and does not update `SyncStatus` or `LastSyncedAt ([argocd_action.go:70](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:70), [argocd_action.go:79](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:79)). The supplied acceptance test asserts these distinctions ([argocd_action_test.go:44](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action_test.go:44)).

2. **The rejection/uncertainty split matches the stated contract.** Typed 4xx responses produce `Failed` with a finish timestamp. Other errors produce `Unknown` without one ([argocd_action.go:81](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:81)). The client supplies typed status errors for sync and rollback ([argocd_client.go:229](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_client.go:229), [argocd_client.go:364](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_client.go:364)). Supplied tests cover 403 rejection, mutation followed by 500, and cancellation ([argocd_action_test.go:30](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action_test.go:30), [:56](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action_test.go:56), [:106](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action_test.go:106)).

3. **Provider failure bodies are excluded on the reviewed paths.** `providerStatusError` retains only action and HTTP status; mutation methods return it without decoding unsuccessful bodies. The service then substitutes a business key or context error before persistence ([argocd_client.go:26](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_client.go:26), [argocd_action.go:83](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:83), [:115](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:115)). The operation-message helper likewise substitutes a fixed key ([argocd_action.go:125](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:125)). Reconciler implementation was not supplied, so its complete persistence behavior is outside this source verdict.

4. **Concrete source defect: rollback bookkeeping failures report a sync error.** The shared transaction-failure branch unconditionally returns `cicd.argocd.app_sync_failed`, including rollback requests ([argocd_action.go:101](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:101)). This is a bounded semantic defect, not evidence that the rollback request itself is incorrect.

5. **Persistence limitation: calculated outcomes can remain recorded as `Running`.** Initial history creation precedes provider submission; outcome changes are persisted in a later transaction. If that transaction fails without committing, the initial `Running` record remains—even when the calculated result was `Failed` or `Unknown` ([argocd_action.go:70](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:70), [:99](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:99)). The detached five-second context addresses request cancellation, but cannot guarantee database availability.

## Root Cause

The remaining weakness is the separation between provider submission and durable local outcome recording. That boundary permits incomplete bookkeeping; its shared failure branch also loses the sync/rollback distinction ([argocd_action.go:74](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:74), [:101](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:101)).

## Recommendations

1. **Small effort:** make the bookkeeping-error key action-specific, preserving the existing rollback key. Add a transaction-failure test for rollback.
2. **Moderate effort:** fault-inject outcome-write failure and abrupt transport loss after simulated mutation. Record the persistence limitation explicitly; any retry must retry **bookkeeping only**, never automatically resubmit the provider mutation.
3. **Verification:** attach fresh isolated-MySQL test results and the separately running full CGO race result. Supplied test source demonstrates assertions, not successful execution.

## Trade-offs

| Choice                              | Benefit                                          | Cost                                                              |
| ----------------------------------- | ------------------------------------------------ | ----------------------------------------------------------------- |
| Keep ambiguous outcomes `Unknown`   | Avoids false terminal claims                     | Requires subsequent resolution                                    |
| Detach bookkeeping for five seconds | Preserves recording after cancellation           | Adds post-cancellation latency; database failures remain possible |
| Retry bookkeeping only              | Improves durability without repeating deployment | Adds retry and concurrency considerations                         |

Review stops at the supplied-source boundary. Provider terminal correlation and real rollback/diff/DataScope acceptance remain explicitly open parent-P0 work; they are not counted as defects or completed verification here.
