---
title: Close foundation generated smoke ownership
doc_type: Remediation
layer: inheritance-sync
status: Active
updated_at: 2026-08-12
linked_contracts:
  - docs/designs/FOUNDATION_RELEASE_MODEL.md
---

# Task Packet: 2026-08-12-foundation-smoke-closure

## Goal

Make the foundation release own the complete generated-business runtime smoke closure so consumers cannot retain stale setup, configuration, helper, or specification copies.

## Scope

### In

- Add generated-business smoke setup, cleanup, configuration, helper, and specification paths to the release manifest.
- Fail release construction when the generic runtime QA closure is left unowned.
- Publish the next immutable patch release and verify Pantheon Ops consumes it without shared smoke drift.

### Out

- Product UI or business behavior changes.
- Ops-specific CMDB or deploy smoke specifications.
- Mutating any existing release tag or artifact.

## Stop Conditions

- Stop publication if exact-commit Release Gate or Full Smoke fails.
- Stop Ops merge if shared generated smoke assets differ from the published release.
- Stop closeout if the Ops business smoke suite does not pass against a running backend.

## Linkage

- Task ID: `2026-08-12-foundation-smoke-closure`
- Task Manifest: `.harness/tasks/2026-08-12-foundation-smoke-closure/manifest.json`
- Evidence Directory: `.harness/evidence/2026-08-12-foundation-smoke-closure/`
- Review File: `.harness/evidence/2026-08-12-foundation-smoke-closure/review.md`
