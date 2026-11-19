---
title: Certify Pantheon Ops foundation consumer integrity
doc_type: Remediation
layer: ci-workflow
status: Active
updated_at: 2026-08-11
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
  - foundation-release.lock.json
---

# Task Packet: 2026-08-11-foundation-consumer-certification

## Goal

Make routine and explicit foundation consumption fail closed on archive, manifest, and release-tree integrity, then certify the Ops repository for the next immutable Base upgrade.

## Scope

### In

- Bind routine release resolution to the lock checksum and verification marker.
- Bind explicit local upgrades to the target archive checksum.
- Make archive extraction portable across Windows/MSYS paths.
- Align workflow Node and action pins with the certified Base baseline.
- Synchronize inheritance documentation and add regression coverage.
- Verify with Windows/MSYS Go race, frontend build, inheritance, and repository gates.

### Out

- Business-domain behavior.
- Product UI changes.
- Database schema changes.

## Boundaries

The change is limited to foundation release consumption, CI workflow baselines, regression tests, and inheritance documentation. Shared platform or system-domain behavior remains owned by `pantheon-base`; Ops business overlays are unchanged.

## Linkage

- Task ID: `2026-08-11-foundation-consumer-certification`
- Task Manifest: `.harness/tasks/2026-08-11-foundation-consumer-certification/manifest.json`
- Evidence Directory: `.harness/evidence/2026-08-11-foundation-consumer-certification/`
- Review File: `.harness/evidence/2026-08-11-foundation-consumer-certification/review.md`
