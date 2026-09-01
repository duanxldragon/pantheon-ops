---
title: Preserve consumer business overlays during generated cleanup
doc_type: Remediation
layer: inheritance-sync
status: Active
updated_at: 2026-08-12
linked_contracts:
  - docs/designs/FOUNDATION_RELEASE_MODEL.md
---

# Task Packet: 2026-08-12-foundation-consumer-cleanup-overlay

## Goal

Ensure Base-owned generated-module cleanup removes only untracked QA artifacts and never deletes a foundation consumer's tracked `business/*` overlays, registries, i18n resources, or schemas.

## Scope

### In

- Classify generated artifacts against the consumer repository Git index.
- Preserve tracked business source and restore tracked registries/i18n to their index baseline.
- Remove untracked generated modules, including generated modules nested under a tracked business domain.
- Add producer and real Ops-consumer regression evidence.

### Out

- Ops business behavior or schema changes.
- Consumer-local overrides of Base cleanup behavior.
- Mutation of the published `pantheon-base-v0.10.17` release.

## Stop Conditions

- Stop publication if cleanup changes any tracked consumer business overlay.
- Stop Ops merge until a new immutable release carries the fix and business smoke passes.
- Stop publication if exact-commit Full Smoke or Release Gate fails.

## Linkage

- Task ID: `2026-08-12-foundation-consumer-cleanup-overlay`
- Task Manifest: `.harness/tasks/2026-08-12-foundation-consumer-cleanup-overlay/manifest.json`
- Evidence Directory: `.harness/evidence/2026-08-12-foundation-consumer-cleanup-overlay/`
- Review File: `.harness/evidence/2026-08-12-foundation-consumer-cleanup-overlay/review.md`
