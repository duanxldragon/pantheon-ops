---
title: Document Metadata And Status
doc_type: Contract
layer: platform
status: Active
linked_contracts:
  - docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md
  - docs/contracts/DOCUMENT_FRONTMATTER_SCHEMA.md
updated_at: 2026-10-31
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

## 2. Unified Metadata Fields (pointer)

The single source of truth for field definitions is
[DOCUMENT_FRONTMATTER_SCHEMA.md](./DOCUMENT_FRONTMATTER_SCHEMA.md): required base fields
(`title / doc_type / layer / status / updated_at`), retained-directory fields
(`index_group / retention_reason / linked_contracts`), type and format constraints, and the
enforced coverage boundary (schema §11). This document no longer repeats them.

This document only defines the value semantics of the `doc_type` enum (§3) and the
`status` enum (§4).

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

- `openspec/specs/`
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

### 6.5 Active review window

`Active` means "in force right now", so it is also an assertion about **now** — and assertions expire. Rules:

- `updated_at` older than **90 days** → **review overdue**.
- Overdue does not mean invalid and never auto-demotes; it means the doc must not be trusted solely because `status: Active`.
- Two disposal options: re-read the content and rewrite `updated_at`, or demote to `Draft` / `Superseded`.
- Reviewing is not date-bumping: pushing `updated_at` to today without reading a word is worse than overdue — it disguises "nobody looked" as "just looked".

`scripts/frontmatter-check.mjs` reports this as a **count** (informational, never a build failure); use `--report-overdue` for the list. Deliberately not an error: at 144 overdue docs (2026-10-31; 71 `.en.md` mirrors; oldest 198 days — `docs/designs/BACKEND.md` / `BACKEND.en.md`, `updated_at` 2026-04-17) the only feasible way to pass an error gate would be bulk-editing `updated_at`, destroying exactly what the rule protects. The bar is **visibility**, not **green**.

Scope note: audit report §11 counted "102 Active docs older than 60 days", a different window and population; not reproduced — the measured 90-day count is authoritative here.

Reading change (2026-11-01 re-check): 144 → **143**. The one that left the overdue set is `docs/designs/MODULE_CONTRACT.md`, whose `updated_at` was pushed to 2026-11-01 because §4.3 changed in substance (W11). That is exactly the disposal this section expects — re-review the content, then rewrite the date — not a bulk date bump.

## 8. First-Round Rollout

From this round forward:

1. new contracts must carry type / layer / status
2. new core design docs must carry type / layer / status
3. docs linked from main README indexes should gradually be upgraded with metadata
4. new review drafts without explicit type / status / linked contract should not enter primary indexes
