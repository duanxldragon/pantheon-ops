# C01 HTTP/factory correction — independent architecture round 4

Reviewer: `/root/c01_architecture_round2` (`architect`). Coordinator-persisted synopsis of the completed independent report.

Architectural status: **CLEAR**, bounded to the HTTP/factory correction. No C01, merge, runtime or visual approval is inferred.

The unusable transaction-state marker is explicitly rejected before owner reads and superseded cancellation classification. Retry freezes a new execution generation; Resume validates the frozen owner binding transactionally. Manual report tokens include that generation and frozen owner/hosts; the UI submits the selected host token.

HTTP manual reports bind request context. Unknown Reader failures retain their cause; invalid association maps to 400, stale lifecycle to 409. The reviewer inspected real MySQL CGO race logs for missing/nil/typed-nil factories, HTTP cause/classification, and the canceled-request regression (`continuation-c01-state-factory-fixed.log`, `continuation-c01-http-cause-fixed.log`, `continuation-c01-http-context.log` and receipts).

Current desktop/mobile behavior and deployment interactions remain unverified because the running preview uses a historical binary. The subsequent complete race run exited 1; its two Deploy failures and Redis availability failures require separate resolution. Retire lifecycle acceptance is **WATCH**, see [owner gap](./continuation-retire-lifecycle-owner-gap.md).
