# Failure Registry

This registry turns repeated agent or process failures into concrete harness changes.

Seeded 2026-07-26 from `pantheon-harness/patterns/templates/failure-registry.template.md` (HOT-002 adoption step). Field vocabulary lives in that template; ratchet rules live in the harness `failure-ratchet-policy.md`.

## Scope

- Repository: `pantheon-ops`
- Owner: method maintainer
- Review cadence: after each release or after repeated agent failure
- Last reviewed: 2026-07-26

## Registry

| Failure ID | Category | Failure Class | Owner Layer | Occurrences | Example | Impact | GitHub Signal | Current Guide | Current Sensor | Current Gate | Detected By | Missed By | Recommended Harness Change | Promotion Decision | Promotion Deadline | Status |
|---|---|---|---|---:|---|---|---|---|---|---|---|---|---|---|---|---|

No entries yet. Add a row when a failure repeats or a one-off failure has high impact; classify it per the ratchet rule (guide / sensor / gate / template / adapter / registry-only / no-action).

## Review Notes

- Repeated failures: none recorded yet.
- Sensors with false positives: none recorded.
- Sensors with known false negatives: none recorded.
- Rules to remove or downgrade: none.
- Next harness changes: none.
