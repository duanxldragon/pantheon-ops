# E01 review amendment: claim failure and scheduler discovery

Parent: `2026-11-16-ops-filedist-product-closure`. Business/filedist only; L2 review continuation. Root owns canonical translations, startup integration, actual MySQL/CGO race and final acceptance.

Review evidence showed consumed Pending/Retrying queue entries were discarded when the GORM parent query, target update or COMMIT response failed. The due dispatcher also silently discarded discovery errors.

The minimum repair preserves the existing parent/target CAS and cancellation fences: distinguish `(false, nil)` stale work from claim SQL errors; before any remote IO, persist the still-unfinished target as failed. Retain active ownership and retry a failed terminal write with a five-second context-bound call and one-second cancellable wait. Never replay ambiguous remote writes. Emit only structured error types and IDs. Discovery keeps its existing one-second retry path and reports sanitized failure/recovery transitions.

Regression first: exercise actual GORM transaction calls through a standard-library SQL connector, injecting parent-read, target-update and both possible COMMIT-error outcomes. Cover Pending/Retrying, terminal cancellation, failed settlement retry, lifecycle stop and stale terminal queue items. Inject due discovery failure and require failure/recovery logs without raw SQL.

Replace only this implementer's six new short keys with the root-provided `business.filedist.task.*` canonical names. Keep the original frozen handoff unchanged; record new source hashes and RED/GREEN evidence in the amendment. Independent approval remains pending.
