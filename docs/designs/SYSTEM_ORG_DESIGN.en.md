---
title: system/org Domain Design
doc_type: Design
layer: system/org
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_ORG_CONTRACT.md
updated_at: 2026-05-05
---

# system/org Domain Design

Chinese version: [SYSTEM_ORG_DESIGN.md](./SYSTEM_ORG_DESIGN.md)

This document defines the independent design anchor for `system/org`.

The organization domain already has departments, posts, governance pages, and backend capability, but historically much of its design was implicit or scattered. This document makes the domain boundary explicit.

## 1. Ownership and Boundary

`system/org` owns organizational structure and governance, not authentication or IAM authorization logic.

## 2. Core Objects

Core objects include:

- departments
- posts
- organization tree
- user organizational attachments where relevant

## 3. Department Governance

Department governance should cover:

- tree integrity
- parent-child correctness
- leader completeness where applicable
- organization-root handling

## 4. Post Governance

Post governance should cover:

- department ownership
- active versus disabled lifecycle
- user attachment safety

## 5. Boundary with IAM

IAM owns user, role, permission, and authorization semantics.

`system/org` owns structural organization semantics. The two cooperate, but neither should collapse into the other.

## 6. Data-Permission Extension Point

The organization domain is a likely future input to row-level data-permission policies, but that does not mean departments or posts are themselves the data-permission engine.

## 7. Acceptance

Acceptance should verify:

- tree integrity
- department and post governance behavior
- explicit boundary with IAM
- readiness for future data-permission cooperation without boundary collapse
