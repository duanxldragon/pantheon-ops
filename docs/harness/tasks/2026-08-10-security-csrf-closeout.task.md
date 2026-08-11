---
title: Close Dependabot security debt and consume shared CSRF foundation fix
doc_type: Remediation
layer: inheritance-sync
status: Archived
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-08-11
---

# Task Packet: 2026-08-10-security-csrf-closeout

## Goal

Remediate the Dependabot security backlog, consume the Base release that fixes the HttpOnly CSRF cookie/header contract exposed by hosted smoke, and close the remaining CodeQL and quality-gate debt.

## Scope

### In

- Upgrade vulnerable Go and frontend dependencies to patched versions.
- Consume the next immutable `pantheon-base` foundation release.
- Remediate the remaining CodeQL allocation-size-overflow finding in `business/deploy`.
- Apply Base-derived Ops quality controls for stale-run cancellation and Go new-code linting.
- Run inheritance, build, race, audit, CodeQL, and hosted smoke validation.

### Out

- New Ops business functionality or follow-up Ops development.

## Linkage

- Task ID: `2026-08-10-security-csrf-closeout`
- Task Manifest: `.harness/tasks/2026-08-10-security-csrf-closeout/manifest.json`
- Evidence Directory: `.harness/evidence/2026-08-10-security-csrf-closeout/`
- Review File: `.harness/evidence/2026-08-10-security-csrf-closeout/review.md`

## Closeout

- Dependabot and CodeQL open alerts: `0`.
- CSRF foundation fix consumed from immutable release `pantheon-base-v0.10.10` (`a95e6e52eee8ae9aeb4fd115d18c7c37609290f6`); final Ops lock advanced to `pantheon-base-v0.10.11` (`48c7ca5dcb8fd3c7235055dbeec57fb5b165b13e`) for the patched dependency set.
- Hosted Security run `31445476446`, Quality run `31445476431`, and Smoke job `93638664396`: passed.
- Local release, workspace-head, inheritance, i18n, overlay, structure, encoding, and foundation-release checks: passed.
