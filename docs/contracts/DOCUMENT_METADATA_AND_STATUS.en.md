---
title: Document Metadata And Status
doc_type: Contract
layer: platform
status: Active
linked_contracts:
  - docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md
  - docs/contracts/DOCUMENT_FRONTMATTER_SCHEMA.md
updated_at: 2026-04-30
---

# Document Metadata And Status

Chinese version: [DOCUMENT_METADATA_AND_STATUS.md](./DOCUMENT_METADATA_AND_STATUS.md)

This document defines the unified metadata fields, document-type enums, and lifecycle-status enums used by the Pantheon Base documentation system.

## 1. Goal

If a repository defines document type but not document status, it quickly regresses:

- docs know whether they are `Design` or `Assessment`, but not whether they are outdated
- main indexes mix `Draft`, `Active`, and `Superseded`
- AI and new contributors cannot tell which doc to trust
- old review drafts stay in place with no exit signal

So all core Pantheon docs should gradually adopt unified metadata.

## 2. Unified Metadata Fields

Core docs should carry at least:

- update time
- type
- ownership layer
- status

Important docs should additionally consider:

- owner
- last reviewed
- linked contract
- index group
- retention reason

New or rewritten docs should prefer YAML frontmatter. Legacy inline header text is only a transition format.

See [DOCUMENT_FRONTMATTER_SCHEMA.md](./DOCUMENT_FRONTMATTER_SCHEMA.md).

## 3. Document Type Enums

Recommended `doc_type` values:

- `Contract`
- `Design`
- `Assessment`
- `Remediation`
- `Acceptance`
- `Archive`

Interpretation:

- `Contract`: execution contract for a layer/module/topic
- `Design`: implementation-oriented design expansion under a contract
- `Assessment`: current-state review, gap review, or maturity review
- `Remediation`: remediation or governance convergence plan
- `Acceptance`: template, matrix, example, or acceptance result
- `Archive`: retained historical material with reference value

`Archive` is not a trash can. If a doc has no reuse value, no linkage, and no baseline value, it should be deleted rather than archived.

## 4. Document Status Enums

Recommended `status` values:

- `Draft`
- `Active`
- `Superseded`
- `Archived`

Interpretation:

- `Draft`: still being drafted; not a formal basis
- `Active`: formally in force and usable as an implementation/review basis
- `Superseded`: replaced by a newer document but not yet deleted
- `Archived`: out of the active governance chain but retained for sample/baseline/reference value

## 5. Type vs Status

Document type and document status are independent dimensions.

Examples:

- an `Assessment` may be `Active`
- the same `Assessment` may later become `Superseded`
- an `Acceptance` sample may remain `Archived`

Recommended representation:

```text
Type: Assessment
Status: Superseded
```

That tells both humans and agents:

- what the doc is for
- whether it is still a primary source

## 6. Index Rules

First-line entries:

- `Contract` + `Active`
- `Design` + `Active`

Second-line entries:

- `Remediation` + `Active`
- `Acceptance` + `Active`

Not first-line by default:

- `Assessment` + `Active`
- any `Draft`
- any `Superseded`
- any `Archived`

Exceptions are allowed when an assessment is still operationally required or when an acceptance template is a long-lived primary artifact.

## 7. Directory Mapping

Recommended directory-to-index mapping:

- `docs/superpowers/specs/`
  - usually `Design`
  - usually `Active` or `Superseded`
  - `index_group: superpowers-specs`
- `docs/archive/examples/`
  - usually `Archived`
  - `index_group: archive/examples`
- `docs/archive/baselines/`
  - usually `Archived` or `Superseded`
  - `index_group: archive/baselines`
- `docs/archive/upgrade/`
  - usually `Archived`
  - `index_group: archive/upgrade`

If a retained doc cannot justify an `index_group` and `retention_reason`, it likely should not be retained.

## 8. First-Round Rollout

From this round forward:

1. new contracts must carry type / layer / status
2. new core design docs must carry type / layer / status
3. docs linked from main README indexes should gradually be upgraded with metadata
4. new review drafts without explicit type / status / linked contract should not enter primary indexes

