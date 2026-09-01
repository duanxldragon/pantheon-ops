---
title: Patch frontend security dependencies for foundation release
doc_type: Remediation
layer: ci-workflow
status: Archived
updated_at: 2026-08-11
linked_contracts:
  - docs/designs/FOUNDATION_RELEASE_MODEL.md
---

# Task Packet: 2026-08-11-base-security-dependencies

## Goal

Remove high-severity npm audit findings from the shared frontend toolchain before cutting the next foundation release.

## Scope

### In

- Upgrade `js-yaml` to `3.15.1`.
- Upgrade `nanoid` to `3.3.17`.
- Cut and verify `pantheon-base-v0.10.11` for Ops consumption.

### Out

- Business-domain changes and unrelated frontend dependency upgrades.

## Linkage

- Task ID: `2026-08-11-base-security-dependencies`
- Task Manifest: `.harness/tasks/2026-08-11-base-security-dependencies/manifest.json`
- Evidence Directory: `.harness/evidence/2026-08-11-base-security-dependencies/`
- Review File: `.harness/evidence/2026-08-11-base-security-dependencies/review.md`
