## Test-only architecture addendum

**Narrow architecture verdict remains CLEAR; whole C01/backend acceptance remains WATCH.** The fixture migration aligns the legitimate caller with the approved RC boundary without weakening its lifecycle or rollback assertions.

The full backend run failed at the old generic caller transaction: `continuation-retire-isolation-full-race-20261118-01.log:401` records `TestStateTransactionValidationRevisionAndLifecycle`, and `:402` records `service.state.transactionIsolationUnknown`. This failure record remains valid and retained.

### Fixture review

- The caller now uses `bizcap.ReadCommittedTransaction` (`backend/modules/business/service/state_transaction_test.go:35`).
- The local no-executing-reference reader verifies the same pool, request context, disabled default transactions, and valid RC proof (`state_transaction_test.go:44`, `:55`). Exactly one reference binding is required (`:114`).
- Wrong-scope, invalid-action, stale-version, successful transition, replay, and Retire checks remain (`:75`, `:80`, `:85`, `:88`, `:91`, `:95`).
- Lifecycle version 10, two writes, outer rollback, unchanged long-lived owner DB, `sql.ErrTxDone`, and zero connections in use remain asserted (`:103`, `:111`, `:117`, `:124`, `:127`, `:134`).

The local reader supplies this lifecycle fixture’s explicit no-reference condition. Actual Deploy reference behavior remains covered by the earlier real cross-owner tests; this addendum does not substitute the fixture for those tests.

### Verification and source binding

The complete Service package passed fresh disposable-MySQL CGO race verification:

`continuation-retire-isolation-service-full-green-20261118-03.log.receipt.json:5896` records exit 0; `:2955` records unchanged source; `:2` and `:13` record disposable MySQL and CGO.

The reviewed fixture matches that receipt:

```text
backend/modules/business/service/state_transaction_test.go
sha256: ed6c3767efa80de590d9e80cc99c5efa22e1c050f07b956bd29421d8acb10574
```

All ten production files checked against the prior core/compatibility receipt remain byte-identical. No production architecture change requires reopening the narrow verdict. Whole-backend acceptance requires a successful run against the refreshed candidate.

Evidence filenames above are relative to `.harness/evidence/2026-11-17-ops-history-audit/`.

### Machine Readable

```json
{
  "lane": "architect",
  "type": "test-only-addendum",
  "architecturalStatus": "CLEAR",
  "scope": "Service transaction lifecycle fixture migration",
  "wholeC01Status": "WATCH",
  "wholeBackendAcceptance": "WATCH",
  "previousFullBackendFailureRetained": true,
  "productionHashMatchesAgainstPriorCandidate": 10,
  "fixture": {
    "path": "backend/modules/business/service/state_transaction_test.go",
    "sha256": "ed6c3767efa80de590d9e80cc99c5efa22e1c050f07b956bd29421d8acb10574",
    "assertionsWeakened": false
  },
  "validation": {
    "receipt": ".harness/evidence/2026-11-17-ops-history-audit/continuation-retire-isolation-service-full-green-20261118-03.log.receipt.json",
    "exitCode": 0,
    "realDisposableMySQL": true,
    "cgoRace": true,
    "sourceUnchanged": true
  },
  "unresolvedArchitecturalBlockers": [],
  "filesModifiedByReviewer": []
}
```
