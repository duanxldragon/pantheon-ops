---
title: Publish Pantheon Base foundation release v0.10.12
doc_type: Remediation
layer: ci-workflow
status: Active
updated_at: 2026-08-11
linked_contracts:
  - docs/designs/FOUNDATION_RELEASE_MODEL.md
---

# Task Packet: 2026-08-11-foundation-release-v0-10-12

## Goal

Record, certify, and publish `pantheon-base-v0.10.12` as a new immutable foundation release without modifying `pantheon-base-v0.10.11`.

## Scope

### In

- Generate release metadata for Base commit `16918771e2650f8c045b0e086144eb290e774704`.
- Build the foundation archive from the manifest-declared shared paths.
- Require the exact-commit Release Gate before publishing.
- Publish an annotated tag, immutable GitHub Release, archive, and SHA-256 file.
- Upgrade the Ops lock only after the published asset is independently verified.

### Out

- Changes to the existing `pantheon-base-v0.10.11` release.
- Product UI, business-domain behavior, or database schema changes.

## Linkage

- Task ID: `2026-08-11-foundation-release-v0-10-12`
- Task Manifest: `.harness/tasks/2026-08-11-foundation-release-v0-10-12/manifest.json`
- Evidence Directory: `.harness/evidence/2026-08-11-foundation-release-v0-10-12/`
- Review File: `.harness/evidence/2026-08-11-foundation-release-v0-10-12/review.md`
