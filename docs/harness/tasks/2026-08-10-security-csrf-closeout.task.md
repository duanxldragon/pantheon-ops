---
title: Close Dependabot security debt and consume shared CSRF foundation fix
doc_type: Remediation
layer: inheritance-sync
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-08-10
---

# Task Packet: 2026-08-10-security-csrf-closeout

## Goal

Remediate the 29 open Dependabot alerts and consume the Base release that fixes the HttpOnly CSRF cookie/header contract exposed by hosted smoke.

## Scope

### In

- Upgrade vulnerable Go and frontend dependencies to patched versions.
- Consume the next immutable `pantheon-base` foundation release.
- Run inheritance, build, race, audit, and hosted smoke validation.

### Out

- New Ops business functionality or follow-up Ops development.

## Linkage

- Task ID: `2026-08-10-security-csrf-closeout`
- Task Manifest: `.harness/tasks/2026-08-10-security-csrf-closeout/manifest.json`
- Evidence Directory: `.harness/evidence/2026-08-10-security-csrf-closeout/`
- Review File: `.harness/evidence/2026-08-10-security-csrf-closeout/review.md`
