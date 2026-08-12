---
title: Consume and certify foundation v0.10.14
doc_type: Remediation
layer: inheritance-sync
status: Active
updated_at: 2026-08-12
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
  - foundation-release.lock.json
---

# Task Packet: 2026-08-12-foundation-v0-10-14-clean-consumer

## Goal

Consume the immutable `pantheon-base-v0.10.14` artifact and independently prove that Ops contains no unauthorized historical shared-source residuals.

## Scope

### In

- Upgrade the lock and shared trees through the existing release consumer.
- Preserve `business/{bizscope,cmdb,deploy}` and documented backend/frontend overlays.
- Build a temporary clean consumer from the same release, overlay only approved Ops-owned paths, and compare product source trees.
- Verify inheritance, Windows Go race, frontend lint/type/build, business smoke, GitHub gates, and hosted SonarCloud revision and measures.

### Out

- New business behavior or schema changes.
- Hand-copying Base source or fixing shared platform behavior locally.
- Deleting or replacing Ops source without a classified diff.

## Stop Conditions

- Stop publication if the Base release is not bound to a successful exact-commit Release Gate.
- Stop merge if the clean consumer comparison finds unauthorized shared residuals.
- Stop merge if inherited gates, business smoke, or hosted SonarCloud quality gate fail.
