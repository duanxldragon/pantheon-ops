## Relocation architecture addendum

**Architectural Status: CLEAR for the narrow relocation; whole C01 remains WATCH.** Moving the implementation to `business/shared/transaction` restores the capability boundary without changing transaction behavior.

### Findings and closure

The root cause was package placement: the checker explicitly prohibits direct GORM types in capability production contracts (`scripts/check-business-module-boundaries.mjs:110`). The relocated helper belongs in the existing shared utility boundary (`backend/modules/business/shared/timeout.go:1`), and the overlay already covers that directory (`business-overlay.json:56`).

The frozen candidate closes this finding:

- Capability contains only its existing contract file, with no helper reexports.
- The transaction leaf imports only standard-library packages and GORM (`backend/modules/business/shared/transaction/transaction.go:3`), introducing no reverse dependency on Service, Deploy, or capability.
- Actual RC BEGIN, private pointer proof, context propagation, savepoint nesting, and commit/error hooks remain unchanged (`transaction.go:36`, `:74`, `:95`, `:118`, `:135`).
- Consumers retain the Retire proof gate and Deploy ownership hooks (`backend/modules/business/service/state.go:112`; `backend/modules/business/deploy/attempt_ownership.go:192`).

### Independent validation

The unchanged boundary checker passed on the frozen candidate.

Strict SHA256 comparisons also passed:

- Helper and tests match their prior bytes after reversing **only the package declaration**: 2/2.
- Eight consumers match the successful prior full-race candidate after reversing **only helper imports, qualifiers, and import ordering**: 8/8.

The prior full backend race PASS remains historical evidence in `continuation-retire-isolation-full-race-20261118-02.log.receipt.json`. This verdict does not claim a completed full race run for the relocated candidate; that verification remains separate.

### Recommendation and trade-off

Keep this leaf package limited to the transaction implementation. It preserves the existing contract/implementation distinction and avoids adding an overlay entry; the cost is one explicit implementation import in each consumer.

### Machine Readable

```json
{
  "lane": "architect",
  "type": "implementation-package-relocation-addendum",
  "architecturalStatus": "CLEAR",
  "scope": "Capability transaction helper relocation to business/shared/transaction",
  "wholeC01Status": "WATCH",
  "boundaryGate": "PASS",
  "unresolvedArchitecturalBlockers": [],
  "normalizedSourceEquivalence": {
    "helperAndTestsPackageDeclarationOnly": 2,
    "consumerImportsQualifiersAndImportOrderingOnly": 8,
    "allComparisonsPassed": true
  },
  "previousFullBackendRace": {
    "receipt": ".harness/evidence/2026-11-17-ops-history-audit/continuation-retire-isolation-full-race-20261118-02.log.receipt.json",
    "exitCode": 0,
    "sourceUnchanged": true,
    "historicalCandidate": true
  },
  "relocatedCandidateFullBackendRace": "Separate verification pending",
  "sourceHashes": {
    "backend/modules/business/shared/transaction/transaction.go": "621748b4a5ce8e9a79cecc61f75907a99b5c5705d10adbcd0552cb72390d38ce",
    "backend/modules/business/shared/transaction/transaction_test.go": "4f62610221ee819a6f5bb49ee08e74b4bcb55c2a85ed5f8df207e72af735f0cb",
    "backend/modules/business/service/state.go": "c1283d807faada5679ed8223d58f749e112b93b18fc977fdb72f58f735732717",
    "backend/modules/business/service/reader_transaction.go": "3b04d94ae70536752805c151573d305cd67600c7d07c6c07742401fdd2ca4e50",
    "backend/modules/business/service/deployment_reference.go": "a3cd941ca9c74829bfe064afd9263c96548ff97e34e5be696e49ccc7fb3196e4",
    "backend/modules/business/service/service.go": "08483efa0f60f7e042b7982ca81f3bdff9241135026ce02ec7cdfbceeb8930f8",
    "backend/modules/business/service/retire_admission_test.go": "66e739b15842407b0fef5249e4c5efa249484ec0c82c16bdab2334949597fc2a",
    "backend/modules/business/service/state_transaction_test.go": "a3e72a081ed5632ac9e0acadd34c12722a9770a407268eccd1fa0cca8a6a3b44",
    "backend/modules/business/deploy/attempt_ownership.go": "f423ebe0c7c1e7819385c5ba9195c1a618021ff0629fb594a3c2722138231ac8",
    "backend/modules/business/deploy/service_reference.go": "398d90b1bec54d6e37f636c23dba5b92260249b5b9d26770a1596c614a8dafe3"
  },
  "reviewMode": "Independent read-only architecture lane",
  "filesModifiedByReviewer": []
}
```
