v6 is implemented, verified, and sealed once. The correction permits only the pinned Deployment’s revision `"1"` after native ReplicaSet ownership/template checks. All 341 protected old assets remain unchanged.

Request SHA256: `b307329e8bd2e41ea4fd1056599a0ca8b5396cfc71138bd719afe41d5abfa0c8`

Verification passed:

- Full suite: 326 executions, 273 unique tests = 260 inherited + 13 new; zero failures/errors/skips.
- Forgery gates: 52/52 refused.
- Five-file Pyright: 0 errors/warnings; syntax compilation passed.
- Sealed `check-inputs` and hash readback: exit 0.

[Handoff with all five source SHAs and exact Root commands](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/continue-velero-alias-v6-handoff.md).

Fresh independent CODE/ARCH approvals and Root’s diagnostic → repair → validate → cleanup remain required. Runtime and P0 acceptance remain open.
