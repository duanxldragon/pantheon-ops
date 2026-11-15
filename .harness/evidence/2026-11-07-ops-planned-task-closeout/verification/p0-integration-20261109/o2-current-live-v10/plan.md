L2 parent: docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md.

Scope: NEW continue-o2-live-v10-* local helpers/tests/transient binaries and this
NEW evidence directory only. Business VM acceptance; logical helper adaptation,
no platform/system/Base/product changes. Generator owns offline work; Dispatcher
owns final source approval, candidate authorization and all actual VM execution.

Reuse immutable v9 driver, original Go tests and matrix/cleanup/SSH semantics.
Keep 45 original assertion calls, 12 cases, default Deploy SSH, exact pin failures,
one independent verifier connection per process, failure codes, no reconnect,
retry or replay, exact ownership cleanup, physical cold process crash/recovery,
and unmodified actual lease/timeouts. Inventory entries are never runtime cases.

1. Preserve OLD helper/evidence hashes without comparing CURRENT product bytes
   to the historical 185-file inventory. Derive new version-path copies only.
2. Discover named-file test, linked deploy/service/capability package tests and
   Linux build inputs using installed Go list -mod=readonly -deps -test; bind
   module/std-library/transitive inputs and installed tool bytes. Reject drift.
3. Run native Python diagnostics/regressions, gofmt, gopls, vet, task-cache CGO1
   MinGW race tests and both platform transient builds. Archive exact inputs and
   outputs. A drifted aggregate is FAILED even when individual commands passed.
4. Leave UNSEALED until the Service transaction final source-stable handoff,
   final CODE APPROVE and ARCH CLEAR are available. Later final verification and
   seal commands require those pins. No self-issued reviews or authorization.
5. Independently reviewed new receipts bind final v10 request/report hashes;
   Dispatcher authorizes the exact request. A durable consumed attempt rejects
   replay before credentials or resource imports. Root performs actual VM matrix.

Stop: concrete executable disabled proposal, fresh local evidence and <=800-word
handoff. Independent review, real VM/cold restart/public HTTP composition, full
repository gates and public Base identity/audit acceptance remain open. No P0
closeout, gate waiver, credentials, remote access or old artifacts modified.
