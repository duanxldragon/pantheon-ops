---
title: Contract Template
doc_type: Contract
layer: platform / system/auth / system/iam / system/org / system/config / business/*
status: Draft
updated_at: YYYY-MM-DD
---

# Contract Template

Chinese version: [CONTRACT_TEMPLATE.md](./CONTRACT_TEMPLATE.md)

Use this template when creating a new contract document.

## Related Design

- `TBD`

## Related Acceptance

- `TBD`

---

# Title

One-sentence summary of the boundary this contract is intended to lock down.

## 1. Background

- why this contract is needed
- what current confusion, risk, or drift it is meant to address
- why a single design doc alone is not sufficient

## 2. Ownership Layer

- declare whether it belongs to `platform`, `system/auth`, `system/iam`, `system/org`, `system/config`, or `business/*`
- if it is cross-layer, state the primary ownership layer and dependent layers

## 3. Goals

- what this contract is intended to achieve
- what stability or consistency the project gains after it exists

## 4. Non-Goals

- what is explicitly out of scope for this round
- which capabilities are only reserved and not part of the current completion target

## 5. Boundaries

Covered objects:

- pages
- modules
- APIs
- data structures
- workflows

Excluded objects:

- `TBD`

## 6. Dependencies

- dependent design docs
- dependent shared components or workflows
- dependent database or module contracts

## 7. Hard Constraints

- red lines that later design and implementation must obey
- what may be depended on
- what must not be depended on
- what regressions are forbidden

## 8. Definition of Done

Describe at least:

- functional boundary
- visual or interaction boundary
- permission boundary
- i18n boundary
- audit or security boundary
- documentation and acceptance closure boundary

## 9. Acceptance Standards

List the required:

- checklists
- templates
- acceptance matrices
- build commands
- test commands
- scan rules

## 10. Related Documents

### 10.1 Design

- `TBD`

### 10.2 Acceptance

- `TBD`

## 11. Status Maintenance Rules

- when new design or remediation no longer fits this contract, update the contract first and then change the implementation
- when this contract is replaced by a newer contract, change its status to `Superseded`
- when it exits the active governance chain but still needs to be retained for reference, change its status to `Archived`

