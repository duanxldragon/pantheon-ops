# Document Frontmatter Spec

Type: Contract
Layer: platform
Status: Active

This document defines the portable YAML frontmatter convention for repository-level governance docs used by Harness Engineering.

## Goal

The purpose of this spec is to make document governance machine-readable across tools and repositories.

It standardizes:

- metadata fields
- contract linkage
- retained archive semantics
- README main-entry discipline

## Required Fields

Every governed Markdown document should use YAML frontmatter with at least:

```yaml
---
title: Example
doc_type: Design
layer: platform
status: Active
updated_at: 2026-05-18
---
```

## Contract Linkage

For non-contract governed docs, `linked_contracts` is required.

```yaml
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
```

## Contract Relationship Fields

Contract documents should progressively move their human-readable related sections into frontmatter fields:

```yaml
related_designs:
  - docs/designs/FOO.md
related_acceptances:
  - docs/acceptances/QUX.md
```

These fields are the machine-readable source of truth.
The contract body may retain readable `关联设计 / 关联评估 / 关联整改 / 关联验收` sections, but they should match the frontmatter.

## Status Rules

Recommended status enum:

- `Draft`
- `Active`
- `Approved`
- `Superseded`
- `Archived`

If a document uses `Superseded`, it should also declare:

```yaml
superseded_by: docs/designs/NEWER_DOC.md
```

## README Main Entry Rule

The repository `docs/README.md` should treat its main entry sections as an active navigation surface.

Recommended rule:

- main-entry links should point only to `Active` docs
- `Draft`, `Archived`, and `Superseded` docs should move to archive/history/template sections, or be mentioned without direct main-entry links

## Portable Gate

The repo shell includes `scripts/harness/check-doc-frontmatter.mjs` to validate:

- frontmatter presence
- required fields
- contract linkage
- superseded linkage
- README link drift
- contract body vs frontmatter relation drift

## Notes

This spec is intentionally lightweight.

It does not try to fully schema-encode every possible document body.
It focuses on the metadata and linkage layer that agents and CI can enforce reliably.
