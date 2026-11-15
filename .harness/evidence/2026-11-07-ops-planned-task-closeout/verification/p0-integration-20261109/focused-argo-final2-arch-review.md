## Summary

**Architectural Status: CLEAR for this bounded business-domain delta.** No material source defect is established by the supplied code: acceptance remains nonterminal, ambiguous outcomes remain Unknown, and provider failure bodies are excluded from the reviewed action errors and persistence paths.

This is a source-review verdict, **not approval of the entire P0 or release**. Test execution and the separately running CGO race gate remain unverified here.

All citations below refer to the supplied files under `backend/modules/business/cicd/argocd/`.

## Analysis

- **HTTP200 records acceptance without claiming completion.** The clients return success on HTTP200; the service records `Running` without assigning `FinishedAt`. Its application update contains only `status` and `last_error`, so it does not manufacture a synchronized revision or completion timestamp.  
  Evidence: `argocd_client.go:231–235`, `argocd_client.go:366–369`, `argocd_action.go:85–88`, `argocd_action.go:111–114`. The acceptance test checks `Running`, absent finish time, unchanged `SyncStatus`, and absent `LastSyncedAt`: `argocd_action_test.go:47–56`.

- **Rejection and uncertainty have distinct outcomes.** Typed provider 4xx responses produce `Failed` with a local finish timestamp. Other errors retain `Unknown` without a finish timestamp. Cancellation is returned through the safe error helper.  
  Evidence: `argocd_action.go:89–104`, `argocd_action.go:134–141`. Supplied tests cover 403 rejection, a 500 after a simulated mutation, and cancellation: `argocd_action_test.go:33–45`, `argocd_action_test.go:59–76`, `argocd_action_test.go:154–184`.

- **Bookkeeping failure preserves durable uncertainty without replaying the mutation.** An Unknown history row is created before either POST. Subsequent history/application updates share a transaction with an independent, bounded context. If that transaction fails, the service returns an error without issuing another POST.  
  Evidence: `argocd_action.go:74–83`, `argocd_action.go:108–126`. Fault-injection tests assert one mutation and the surviving Unknown row: `argocd_action_test.go:79–119`.

- **The reviewed failure paths exclude provider payloads.** POST failures retain only action and HTTP status; the service replaces those errors with safe business keys or context errors before persisting them. Application-status decoding failures are sanitized, and terminal operation messages are replaced with a fixed business key.  
  Evidence: `argocd_client.go:26–35`, `argocd_client.go:194–197`, `argocd_client.go:231–232`, `argocd_client.go:366–367`; `argocd_action.go:92–104`, `argocd_action.go:144–150`. Redaction assertions appear in `argocd_client_test.go:110–155`.

- **Rollback submission uses provider history rather than the sync endpoint.** It selects the greatest nonnegative history ID matching the requested revision and submits that ID to `/rollback`.  
  Evidence: `argocd_action.go:59–71`, `argocd_action.go:80–83`, `argocd_client.go:347–367`. The supplied fixture verifies the selected ID and rejects an unknown revision: `argocd_action_test.go:123–151`. This establishes submission behavior, not real rollback acceptance.

## Root Cause

The semantic failure mode addressed is collapsing **request acceptance, definite rejection, and uncertain delivery** into a single success/failure result. The implementation separates those outcomes and persists uncertainty before crossing the provider mutation boundary (`argocd_action.go:74–104`).

No additional material root-cause defect is demonstrated within this delta.

## Recommendations

1. **Complete the existing verification gates — low additional effort, required execution evidence.** Record focused test results and the full CGO race result. The supplied tests show intended coverage, but their presence does not establish that they passed.
2. **Strengthen bookkeeping fault coverage — low effort, targeted confidence gain.** Add failure injection on the application update after the history update succeeds. The current injection targets the first write (`argocd_action_test.go:101–104`); the second-write case would exercise rollback of the whole transaction (`argocd_action.go:111–114`).
3. **Consider safe diagnostic classification — low effort, operational benefit.** Persistence-failure logging currently records identifiers and error type only (`argocd_action.go:120–122`). A narrowly allowlisted failure category could improve diagnosis while retaining payload redaction.

## Trade-offs

| Choice                                       | Benefit                                                       | Concern                                                                                                           |
| -------------------------------------------- | ------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| Persist Unknown before POST                  | Preserves uncertainty across crashes and bookkeeping failures | A row may represent an action never submitted (`argocd_action.go:74–83`)                                          |
| Use safe keys instead of provider messages   | Prevents payload disclosure                                   | Less diagnostic detail (`argocd_action.go:92–104`)                                                                |
| Detach bounded bookkeeping from cancellation | Preserves the observable outcome                              | Cancellation can be followed by additional database latency, bounded to five seconds (`argocd_action.go:108–114`) |

The review stops at the supplied source. Reconciler implementation was not supplied; its test demonstrates an assertion, not independently reviewed terminal correlation (`argocd_action_test.go:187–205`). Provider correlation and real rollback/diff/DataScope acceptance remain the explicitly open parent-P0 work.
