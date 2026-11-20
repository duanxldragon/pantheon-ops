# D3: Base-first Consumer Re-review

**Priority:** P0  
**Estimate:** 0.5-1 engineer-day  
**State:** ready

## In

- Verify `foundation-release.lock.json` and `.business-overlay-report.json` agree on v0.13.0.
- Re-run the overlay and boundary checks and, if possible, a clean temporary rebuild.
- Classify any shared middleware, common package, frontend tooling, or registry drift as Base-owned.
- Record the current `frontend/src/modules/generated/system.ts` dirty change as a separate ownership decision.

## Out

- Editing Base from this repository.
- Hand-copying Base source into Ops.
- Treating a green local build as proof that external integration is complete.

## Exit Evidence

- Lock/release checksum/rebuild comparison.
- Business foundation checks and staged tree results.
- Findings-first review with resolved, deferred, and Base follow-up items.
