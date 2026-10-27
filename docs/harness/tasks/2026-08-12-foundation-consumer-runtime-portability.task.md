---
title: Make generated-module smoke portable across foundation consumers
doc_type: Remediation
layer: inheritance-sync
status: Active
updated_at: 2026-08-12
linked_contracts:
  - docs/designs/FOUNDATION_RELEASE_MODEL.md
---

# Task Packet: 2026-08-12-foundation-consumer-runtime-portability

## Goal

Make Base-owned generated-module runtime smoke use the consumer repository's Go module identity and restore the tracked feature-ledger baseline after smoke cleanup.

## Scope

### In

- Resolve generated registry import assertions from the active repository `go.mod` and backend layout.
- Restore tracked `schema/generated/feature-ledger.json` from the consumer Git index.
- Add focused producer regression coverage and real Ops runtime evidence.
- Distribute every Base-owned script referenced by the shared frontend package contract.
- Publish a new immutable patch release after exact-commit gates pass.

### Out

- Ops business behavior or schema changes.
- Consumer-local smoke or cleanup overrides.
- Mutation of published `pantheon-base-v0.10.18` assets.

## Stop Conditions

- Stop publication if producer tests, independent review, or exact-commit gates fail.
- Stop Ops merge until the immutable patch is consumed and full business smoke passes.
- Stop if cleanup leaves any tracked consumer runtime artifact modified.
- Stop if the consumer package contract references a script absent from the release bundle.

## Linkage

- Task ID: `2026-08-12-foundation-consumer-runtime-portability`
- Task Manifest: `.harness/tasks/2026-08-12-foundation-consumer-runtime-portability/manifest.json`
- Evidence Directory: `.harness/evidence/2026-08-12-foundation-consumer-runtime-portability/`
- Review File: `.harness/evidence/2026-08-12-foundation-consumer-runtime-portability/review.md`
