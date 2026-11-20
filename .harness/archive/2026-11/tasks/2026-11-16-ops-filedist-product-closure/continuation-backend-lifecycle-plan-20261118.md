# E01 bounded backend lifecycle continuation

Parent: 2026-11-16-ops-filedist-product-closure. L2; business/filedist only. Other writers own the frontend, shared composition, C01, C04 and Helm. No DTO, schema, Base or credentials changes.

Observed gaps: CancelTask changes only the parent status; the transfer engine has no active cancellation and SFTP connect/upload lack context. Retry waits unconditionally. Future scheduled tasks have no dispatcher. Startup does not reconcile interrupted work. Existing pending->running claim and stable-ID live authorization remain required.

Minimum approach:

1. Propagate per-target contexts, keyed by parent task ID, through a task-owned transfer engine. Register cancellation before the DB transfer claim; cancel parent and unfinished targets atomically, then cancel its active contexts. Conditional target/task writes preserve terminal cancellation and prevent duplicate transfer claims.
2. Reuse SFTP transport with context-aware connect/upload. Cancellation closes the owned connection and stops retry waiting. Previously written remote bytes may remain; this is interruption, not atomic remote rollback.
3. Add explicit module/service Start(ctx) for root composition after migration and authorizer/provider wiring. A bounded dispatcher discovers due pending tasks; claims enforce scheduledAt. On startup, interrupted running work becomes failed, preserving completed targets and never automatically replaying ambiguous remote writes. Module Stop cancels dispatcher/monitors/workers.
4. Add focused in-process cancellation/claim/recovery tests. Preserve existing authorization/dequeue guards. Root coordinates actual MySQL/CGO race after all writers freeze; no heavy DB tests or live SSH here.

Verification handoff: gofmt/scoped diff and diagnostics/compile locally if approved; meaningful test filters and exact source hashes go to root. No whole-E01 completion or independent approval asserted. New safe failure keys and startup integration requirements are explicitly handed to root.
