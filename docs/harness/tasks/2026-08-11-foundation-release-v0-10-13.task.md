---
title: Publish Pantheon Base foundation release v0.10.13
doc_type: Remediation
layer: ci-workflow
status: Active
updated_at: 2026-08-11
linked_contracts:
  - docs/designs/FOUNDATION_RELEASE_MODEL.md
---

# Task Packet: 2026-08-11-foundation-release-v0-10-13

## Goal

Publish `pantheon-base-v0.10.13` as an immutable certified foundation release that closes the shared frontend ownership gap discovered during the Ops SonarCloud classification.

## Scope

### In

- Certify Base commit `ac62d71581865d4649691095ae46216f07726681` through its exact-commit GitHub Release Gate.
- Publish the manifest-bound archive, annotated tag, GitHub Release, and SHA-256 checksum.
- Preserve the explicit ownership of the generic frontend shell roots plus `frontend/src/api` and `frontend/src/hooks`.
- Deliver the verified release to Pantheon Ops through its consumer lock.

### Out

- Mutating existing foundation releases or tags.
- Product UI redesign, business-domain behavior, and database-schema changes.
- SonarCloud issue-level remediation before a fresh Ops hosted analysis completes.

## Linkage

- Task ID: `2026-08-11-foundation-release-v0-10-13`
- Task Manifest: `.harness/tasks/2026-08-11-foundation-release-v0-10-13/manifest.json`
- Evidence Directory: `.harness/evidence/2026-08-11-foundation-release-v0-10-13/`
- Review File: `.harness/evidence/2026-08-11-foundation-release-v0-10-13/review.md`
