Implemented the scoped `business/deploy` O2 repair. [Handoff](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/o2-engine-ownership-20261110/handoff.md) includes exact files, old/new hashes and RED evidence.

## Verification

- Source-stable CGO race: **157 passed**, two live-VM tests skipped.
- Vet, build, gofmt and native gopls checks passed; one pre-existing suggestion remains.
- Required real MySQL regressions passed; Redis returned `+PONG`.
- Exact owned timeout fixtures cleaned.

## Remaining gaps

Live-VM acceptance, root full-tree gates and independent CODE/ARCH review remain open. `ServiceInstanceStateApplier` retains its separate commit boundary; cross-module atomicity remains unresolved.

Implementation is reviewable. O2/P0 were not self-approved or closed.
