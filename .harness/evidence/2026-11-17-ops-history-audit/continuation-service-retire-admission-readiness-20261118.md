# Independent Service Retire admission readiness

Source: existing `architect` agent `/root/c01_handoff_architecture`, read-only final response received by Root on 2026-11-18. This file is a condensed coordinator transcription, not a verbatim saved agent document. No source edits or DB tests were performed by this lane.

Verdict: narrow RED-ready CLEAR; PATCH admission remains open and whole C01 WATCH.

Owner `TransitionState` is the minimal common entry (`service/handler.go:142`, `service/state.go:54`, `service/capability.go:23`). An HTTP-only guard leaves the typed callback path open. Actual Retire mutations should use the existing context-bound READ COMMITTED owner transaction, reuse caller transactions and acquire instance→service→application current row locks. Preserve replay, optional ExpectedLifecycleVersion semantics and unknown/stopped/failed→retired matrix; query `checkDeploymentReferences` before mutation and use the locked lifecycle version for CAS. Missing or unusable transaction-capable readers and query failures reject.

Existing typed reader covers running, pause_requested, paused and cancel_requested without locking Deploy rows. Completion writes terminal first (`recomputeTaskStatusTx:699`), then invokes owner callback (`:717`), so own reference disappears within the same transaction. Do not add excludeTaskID, host-wide locking or a stopped-only PATCH policy. The stricter historical wording belongs to an explicitly Superseded packet.

Production registration is already complete (`deploy.go:51–56` and `opsinit.go:69`) before recovery. Existing Service/Deploy fixtures register transaction-capable readers. Use typed fakes only in new focused Service tests, and real cross-owner/competition tests in Deploy fixtures. Keep absence-of-registration tests meaningful.

Readiness validation set: executing/nonexecuting references; reader binding/failure modes; uncommitted caller transaction visibility; deterministic Start-first and PATCH-first races; terminal self-callback versus another executing reference; replay/CAS/lifecycle/DataScope/outer rollback compatibility. Query-entry fault coverage must not be represented as blocked-driver context cancellation.

Existing error keys have delete-only text in seed, canonical error appendix and frontend runtime/fallback resources. Reuse requires consistent business wording synchronization. Full queue recovery, broader-host policy and current runtime/provider/role acceptance remain gaps.

Implementation and acceptance scope: `.harness/tasks/2026-11-16-ops-service-deploy-integrity/continuation-service-retire-admission-plan.md`.
