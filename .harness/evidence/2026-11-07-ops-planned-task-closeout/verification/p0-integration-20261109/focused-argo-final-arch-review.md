## Summary

**Architectural Status: CLEAR for this bounded business-domain delta.** No material source defect is established by the supplied code: acceptance remains nonterminal, ambiguous failures retain durable uncertainty, and provider failure bodies are excluded from action errors.

This is a source-review verdict only. It does **not** approve the entire P0 or release; test execution results and the separately running CGO race remain verification evidence to collect.

## Analysis

- **HTTP 200 records acceptance, not completion.** The client returns success on HTTP 200 without interpreting the response as terminal completion (`argocd_client.go:229–233`, `364–367`). The service records `Running` without setting `FinishedAt`, `SyncStatus`, or `LastSyncedAt` (`argocd_action.go:85–88`, `111–114`). The acceptance test explicitly checks these distinctions (`argocd_action_test.go:47–56`).

- **Rejected and uncertain outcomes are separated correctly under the supplied contract.** Typed 4xx responses produce `Failed` with a finish timestamp; other request errors produce `Unknown` without one (`argocd_action.go:89–104`). The supplied tests cover 403 rejection, simulated mutation followed by 500, and cancellation (`argocd_action_test.go:33–45`, `59–76`, `154–184`). Classification uses HTTP status alone; the definite-rejection interpretation therefore depends on the provider contract supplied for this review (`argocd_action.go:91`).

- **Bookkeeping failure preserves a durable uncertainty record.** History is inserted as `Unknown` before the mutation (`argocd_action.go:74–83`). Outcome history and application updates share a transaction with a separate, bounded context that survives request cancellation (`108–114`). The failure branch returns without replaying the provider action (`115–126`). Fault-injection tests assert one mutation and retained `Unknown` history (`argocd_action_test.go:101–119`).

- **Provider failure payloads are excluded from the reviewed action path.** HTTP failures retain only action and status (`argocd_client.go:26–35`, `229–230`, `364–365`). The service replaces request errors with a business key or standard context error (`argocd_action.go:92`, `134–141`), and persistence-failure logging records only the error type (`120–122`). The operation-message helper returns a fixed key rather than `OperationState.Message` (`144–150`).

- **Reconciliation is only partially inspectable here.** The supplied reconciliation test asserts redaction and use of the provider finish timestamp (`argocd_action_test.go:187–205`), but the reconciler implementation is absent. Those assertions establish test intent, not independently reviewed reconciliation correctness.

## Root Cause

Provider mutation and local outcome persistence cannot commit atomically: the POST occurs before the outcome transaction (`argocd_action.go:80–83`, `110–114`). Consequently, transport failure or bookkeeping failure cannot prove that deployment did not occur. Persisting `Unknown` before mutation is an appropriate conservative response to that boundary (`74–78`).

## Recommendations

1. **Complete verification evidence — low effort, high confidence impact.** Record actual isolated-MySQL test results and the pending full CGO race result. Supplied test source is not execution evidence.

2. **Exercise failure of the transaction’s second update — low effort, meaningful integrity coverage.** The existing injected failure targets the history update (`argocd_action_test.go:101–104`). Add an application-update failure case to verify that the preceding history update rolls back, leaving durable `Unknown` history and the prior application state (`argocd_action.go:111–114`). This is a coverage gap, not an identified production defect.

## Trade-offs

| Choice                                                           | Benefit                               | Concrete concern                                                                                                                                                                          |
| ---------------------------------------------------------------- | ------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Persist `Unknown` before POST                                    | Retains evidence across interruption  | A crash before POST can leave uncertainty for an action never submitted (`argocd_action.go:74–83`).                                                                                       |
| Return a safe error after accepted POST plus bookkeeping failure | Avoids claiming persistence succeeded | The same failure key also represents other failures; callers cannot infer rejection or safe retry from that key alone (`argocd_action.go:123–126`; `argocd_action_test.go:86–87`, `113`). |
| Continue bookkeeping after cancellation                          | Makes uncertainty visible             | Request return can await the additional five-second persistence budget (`argocd_action.go:108–109`).                                                                                      |
| Discard provider failure bodies                                  | Prevents payload disclosure           | Diagnosis loses provider detail; structured status remains available internally (`argocd_client.go:29–35`).                                                                               |

The review stops at the supplied source. Provider terminal correlation and real rollback/diff/DataScope acceptance remain explicitly open parent-P0 work.
