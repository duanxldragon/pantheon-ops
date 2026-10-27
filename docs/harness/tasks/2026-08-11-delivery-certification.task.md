---
title: Certify Pantheon Base delivery for Ops development
doc_type: Remediation
layer: ci-workflow
status: Active
updated_at: 2026-08-11
linked_contracts:
  - docs/designs/FOUNDATION_RELEASE_MODEL.md
  - docs/designs/QUALITY_AND_SECURITY_STRATEGY.md
---

# Task Packet: 2026-08-11-delivery-certification

## Goal

Close the foundation release certification gaps, synchronize deployment documentation with runtime behavior, and produce an immutable Base release that Pantheon Ops can consume.

## Scope

### In

- Bind Release Gate and SonarCloud evidence to the immutable candidate commit.
- Require successful release certification before publishing.
- Prevent release asset overwrite and reject dirty bundle sources.
- Run Actionlint and Full Smoke for every `main` candidate.
- Align Node, Docker, runtime version, environment, README, and deployment documentation.
- Verify with Windows/MSYS Go race tests and repository quality gates.

### Out

- Product UI changes.
- Business-domain behavior.
- Database schema changes.

## Boundaries

The change is limited to release tooling, CI workflows, runtime build metadata, and operational documentation. It does not change API, permission, menu, i18n, audit, or database contracts.

## Linkage

- Task ID: `2026-08-11-delivery-certification`
- Task Manifest: `.harness/tasks/2026-08-11-delivery-certification/manifest.json`
- Evidence Directory: `.harness/evidence/2026-08-11-delivery-certification/`
- Review File: `.harness/evidence/2026-08-11-delivery-certification/review.md`
