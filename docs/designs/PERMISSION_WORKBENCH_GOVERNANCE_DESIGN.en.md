---
title: Permission Workbench Governance Deep-Dive
doc_type: Design
layer: system/iam
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-05-05
---

# Permission Workbench Governance Deep-Dive

Chinese version: [PERMISSION_WORKBENCH_GOVERNANCE_DESIGN.md](./PERMISSION_WORKBENCH_GOVERNANCE_DESIGN.md)

This document defines the next-stage boundary for the `system/iam` permission workbench: evolving from discovery, export, and controlled repair into discovery, remediation, and tracking.

## 1. Current State

The current workbench already supports meaningful governance visibility and controlled remediation for permission gaps.

## 2. Non-Goals

Not in scope:

- unconstrained freehand permission mutation
- replacing the full permission model with one workbench screen

## 3. Governance Tracking Model

The workbench should not only surface issues. It should also track remediation progress and governance state explicitly.

## 4. Governance Loop

The intended loop is:

- discover
- explain
- remediate
- track

### 4.1 Default Browsing Behavior

Default browsing should prioritize explanation of integrity gaps, page gaps, and API-policy gaps in a readable way for administrators.

## 5. Permission and Security Constraints

The workbench must still obey:

- page permission rules
- action permission rules
- backend policy constraints
- controlled-write expectations

It is a governance surface, not a shortcut around IAM safety.

## 6. Acceptance

Acceptance should verify:

- issue discovery
- explanation clarity
- controlled remediation behavior
- remediation tracking
- consistency with the overall permission model
