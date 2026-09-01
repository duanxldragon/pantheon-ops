---
title: Distribute shared CSRF request client in foundation release
doc_type: Remediation
layer: system/auth
status: Active
updated_at: 2026-08-10
linked_contracts:
  - docs/designs/FOUNDATION_RELEASE_MODEL.md
  - docs/designs/WORKFLOW.md
---

# Task Packet: 2026-08-10-shared-csrf-release

## Goal

Ensure the HttpOnly CSRF cookie/header contract is shipped to foundation-release consumers.

## Scope

### In

- Include the shared request client and error classifier in the foundation release manifest.
- Validate release manifest generation and downstream Ops consumption.

### Out

- Business-domain changes, cookie policy changes, and unrelated frontend files.

## Verification Plan

- Foundation release manifest unit test.
- Ops foundation sync, frontend build/type-check, and hosted smoke.

## Linkage

- Task ID: `2026-08-10-shared-csrf-release`
- Task Manifest: `.harness/tasks/2026-08-10-shared-csrf-release/manifest.json`
- Evidence Directory: `.harness/evidence/2026-08-10-shared-csrf-release/`
- Review File: `.harness/evidence/2026-08-10-shared-csrf-release/review.md`
