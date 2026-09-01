---
title: Document Frontmatter Schema
doc_type: Contract
layer: platform
status: Active
updated_at: 2026-05-18
---

# Document Frontmatter Schema

Chinese version: [DOCUMENT_FRONTMATTER_SCHEMA.md](./DOCUMENT_FRONTMATTER_SCHEMA.md)

This document defines the unified YAML frontmatter convention used by the Pantheon Base documentation system.

It supplements [DOCUMENT_GOVERNANCE_CONTRACT.md](./DOCUMENT_GOVERNANCE_CONTRACT.md) and [DOCUMENT_METADATA_AND_STATUS.md](./DOCUMENT_METADATA_AND_STATUS.md) by moving key metadata from body text into a stable machine-readable header.

## 1. Goal

This convention solves three problems:

- lets AI and scripts read document type, status, linked contracts, and index groups reliably
- makes the retention logic for `docs/superpowers/specs/` and `docs/archive/*` truly machine-readable
- creates stable input for later lint, drift checks, and linkage checks

The current goal is not to rewrite the whole `docs/` tree at once. The goal is to establish a unified format for new docs and already-governed directories first.

## 2. Base Format

Governed Markdown docs should prefer YAML frontmatter:

```yaml
---
title: Platform Config Governance And Low-Code Menu Redesign
doc_type: Design
layer: platform
depends_on_layers:
  - system/config
status: Approved
index_group: superpowers-specs
retention_reason: retained as the design anchor for platform navigation IA changes and low-code workbench promotion
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-05-13
---
```

The body should still keep its own H1:

```md
# Platform Config Governance And Low-Code Menu Redesign
```

Principle:

- frontmatter is for machine readers
- headings and body remain for human readers
- frontmatter does not replace body semantics

## 3. Field Definitions

### 3.1 Required Fields

- `title`
- `doc_type`
- `layer`
- `status`
- `updated_at` in `YYYY-MM-DD`

### 3.2 Conditionally Required Fields

- `linked_contracts`
  - expected for `Design / Assessment / Remediation / Acceptance / specs / archive` docs
- `index_group`
  - required for `docs/superpowers/specs/` and `docs/archive/*`
- `retention_reason`
  - required for `docs/superpowers/specs/` and `docs/archive/*`

### 3.3 Optional Fields

- `depends_on_layers`
- `superseded_by`
- `owner`
- `last_reviewed_at`

## 4. Field Types

Scalar string fields:

- `title`
- `doc_type`
- `layer`
- `status`
- `index_group`
- `retention_reason`
- `updated_at`
- `superseded_by`
- `owner`
- `last_reviewed_at`

Array fields:

- `depends_on_layers`
- `linked_contracts`

Even when only one value exists, arrays should stay arrays to avoid script branching.

## 5. Recommended Enums

### 5.1 `doc_type`

- `Contract`
- `Design`
- `Assessment`
- `Remediation`
- `Acceptance`

### 5.2 `status`

- `Draft`
- `Active`
- `Approved`
- `Superseded`
- `Archived`

### 5.3 `index_group`

Current fixed values:

- `superpowers-specs`
- `archive/examples`
- `archive/baselines`
- `archive/upgrade`

Add new enums only when directories really expand. Do not pre-allocate empty categories.

## 6. Directory Mapping Rules

### 6.1 `docs/superpowers/specs/`

- usually `doc_type: Design`
- usually `status: Approved` or `Superseded`
- `index_group: superpowers-specs`
- `retention_reason` required
- `linked_contracts` required

### 6.2 `docs/archive/examples/`

- `status: Archived`
- `index_group: archive/examples`
- `retention_reason` required
- `linked_contracts` required

### 6.3 `docs/archive/baselines/`

- usually `status: Archived` or `Superseded`
- `index_group: archive/baselines`
- `retention_reason` required
- `linked_contracts` required

### 6.4 `docs/archive/upgrade/`

- `status: Archived`
- `index_group: archive/upgrade`
- `retention_reason` required
- `linked_contracts` required

## 7. Migration Strategy

Use progressive migration, not a hard cutover.

Phase 1:

- new docs use YAML frontmatter first
- `docs/superpowers/specs/` and `docs/archive/*` migrate first

Phase 2:

- `contracts/`
- `designs/`
- `acceptances/`
- `remediations/`
- `assessments/`

Phase 3:

- decide whether to remove old inline metadata headers
- until scripts fully take over, do not force-delete every legacy trace

## 8. Examples

### 8.1 Specs Example

```yaml
---
title: Language Session And Cross-Page Selection Design
doc_type: Design
layer: platform
depends_on_layers:
  - system/auth
  - system/config
  - system/iam
status: Approved
index_group: superpowers-specs
retention_reason: retained as the cross-module design anchor for language runtime priority and cross-page batch selection
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-05-16
---
```

### 8.2 Archive Example

```yaml
---
title: Platform Shell PR Example
doc_type: Acceptance
layer: platform
status: Archived
index_group: archive/examples
retention_reason: retained as a reusable PR-description example for platform shell changes
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-04-30
---
```

## 9. Current Non-Goals

This schema currently solves only metadata machine readability. It does not yet solve:

- body-structure schema
- evidence-chain field schema
- automatic cross-file validation across change / task packet / evidence / review

Those should be added later after frontmatter is stable.

## 10. Current Validation Commands

The repository currently provides a lightweight frontmatter validator:

```bash
npm run check:docs-frontmatter
```

To also inspect which docs are still legacy and not migrated:

```bash
npm run check:docs-frontmatter:legacy
```

Current checks include:

- full `docs/*.md` scan
- base field requirements for docs already using frontmatter
- directory-specific rules for `docs/superpowers/specs/` and `docs/archive/*`
- `index_group` vs directory-semantic consistency
- allowed `status` values by directory
- `linked_contracts` array validity and target existence
- `linked_contracts` required on non-contract core docs
- `superseded_by` required on `Superseded` docs
- reporting legacy docs that still use old inline metadata headers

Related test command:

```bash
npm run test:docs-frontmatter
```

