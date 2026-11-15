## Summary

**Architectural Status: `WATCH` for this bounded business-domain delta.** The submission paths correctly separate acceptance from completion and protect provider failure bodies, but the rejection classifier leaves definite representation rejections such as HTTP 415 as `Unknown`. This verdict does not approve the entire P0 or release.

## Analysis

**1. Current source defect: definite rejection coverage is incomplete.**

Both POST methods convert every non-200 response into `providerStatusError` ([argocd_client.go:231](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_client.go:231), [argocd_client.go:366](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_client.go:366)). However, `definitiveApplicationRejection` recognizes only seven status codes; HTTP 415, whose semantics explicitly reject the request representation, falls through to `false` ([argocd_action.go:134](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:134)).

Consequently, a 415 rejection persists `Unknown` without `FinishedAt`, rather than the required `Failed` with a finish timestamp ([argocd_action.go:89](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:89)). This is a classification defect in the implemented delta. The supplied evidence does **not** establish that a deployed Argo endpoint has actually returned 415.

The service rejection test exercises only 403, so it does not cover this gap ([argocd_action_test.go:34](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action_test.go:34)).

**2. The central safety properties are implemented coherently.**

- An `Unknown` history is written before the provider mutation. HTTP 200 subsequently promotes it to `Running`, without setting a finish timestamp ([argocd_action.go:74](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:74)).
- Transport errors and unclassified responses retain uncertainty. Outcome persistence uses a separate, bounded context after request cancellation ([argocd_action.go:89](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:89), [argocd_action.go:106](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:106)).
- HTTP failure errors contain status and action only; submission errors become a business key or context error. Provider operation messages are replaced with a business key ([argocd_client.go:26](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_client.go:26), [argocd_action.go:146](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:146)).

**3. Verification remains separate from source findings.**

The tests cover acceptance, uncertain server responses, cancellation, and failure of the history outcome update ([argocd_action_test.go:48](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action_test.go:48), [argocd_action_test.go:60](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action_test.go:60), [argocd_action_test.go:84](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action_test.go:84), [argocd_action_test.go:159](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action_test.go:159)).

The injected database fault targets the **first** transaction update. Failure of the application update after a successful history update is not exercised by that test. Reconciler implementation is not supplied, so its test alone cannot establish its architecture or correlation correctness. No test execution results or completed race result were supplied.

## Root Cause

The rejection policy is encoded as a narrow status whitelist. Every omitted status inherits the uncertainty fallback, including statuses that definitively reject the request ([argocd_action.go:134](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:134)).

## Recommendations

1. **Small effort — correct definite rejection coverage.** Add HTTP 415 and document other supported definite rejection codes. Add service-level cases for sync and rollback asserting `Failed`, non-null `FinishedAt`, and sanitized errors. Preserve `Unknown` for 408, 499, transport errors, and 5xx.
2. **Small effort — strengthen transaction verification.** Inject failure specifically into the application update; verify the history update rolls back to its durable initial `Unknown` state and the provider receives exactly one mutation ([argocd_action.go:110](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:110)).
3. **Verification only — obtain fresh test and CGO race results.** These remain pending evidence, not additional diagnosed source defects.

## Trade-offs

| Option                        | Pros                                                           | Cons                                        |
| ----------------------------- | -------------------------------------------------------------- | ------------------------------------------- |
| Keep the current whitelist    | Conservative about uncertain provider outcomes                 | Misclassifies omitted definite rejections   |
| Expand a documented whitelist | Meets rejection semantics while preserving timeout uncertainty | Requires explicit status-policy maintenance |

A blanket “all 4xx are terminal” rule would undermine the existing timeout safeguards. Provider correlation and real rollback/diff/DataScope acceptance remain explicitly open parent-P0 work; they are outside this verdict.
