# Retire async rejection: independent architecture blocker

Provenance: Root's condensed transcription of c01_handoff_architecture's follow-up. New async path BLOCK / REQUEST CHANGES. Previous metadata-contract CLEAR is retained; whole C01 WATCH, archiveApproved=false.

The new runAsyncTask validation branch returns immediately. Queue reconciliation can still dispatch historical running SSH Retire/unknown-kind tasks, so releasing the worker slot alone leaves no persistent rejection reason, failure settlement or lease release. This is a new silent regression, not an exemption under the broader open queue-recovery work.

Minimal repair: retain the validator, record task ID and guard reason, reuse failQueuedTask, return its fencedTransaction error, and explicitly handle/log settlement errors at all caller sites. Existing fencing rejects running attempts, marks unfinished hosts failed, releases only this task's leases and recomputes status. Add real queue tests for rejection settlement, running-attempt protection and observed SQL/owner failure with rollback.

Do not add a generic localOnly/skipOwner callback option. Unknown-kind Install retains its existing failed-owner transition semantics. Retire failure does not Apply, but owner reads still precede that branch; stale/missing owner or SQL fault may roll back. Record that settlement failure and keep the task open rather than inventing a terminal success. A guarantee of terminal Retire even with owner loss would need separately bounded review.
