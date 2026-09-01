---
title: Generator Module Design
doc_type: Design
layer: system/config
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-05-04
---

# Generator Module Design

Chinese version: [GENERATOR_MODULE_DESIGN.md](./GENERATOR_MODULE_DESIGN.md)

This document defines the formal boundary of `system/config -> generator`.

## 1. Module Positioning

The generator belongs to `system/config` as a controlled development-assistance capability.

## 2. Target Boundary

### 2.1 What `generator` Owns

- scaffold generation
- controlled business-module bootstrap outputs
- generation-time metadata and contract alignment

### 2.2 What `generator` Does Not Own

- runtime low-code orchestration
- redefinition of core system-domain boundaries
- arbitrary source rewriting outside governed generation paths

### 2.3 Current Constraints

The current system keeps generation constrained to governed business-module scenarios.

## 3. Relationship with `dynamicmodule`

`generator` creates module artifacts. `dynamicmodule` governs how those artifacts attach to the system.

## 4. Risk Classification

The generator is not a harmless UI convenience. It influences source code, module contracts, and governed registration behavior.

## 5. Permission Model

### 5.1 Page Permission

The generator surface should have explicit page access control.

### 5.2 Action Permissions

High-sensitivity generation actions must be separated from view-only capability.

### 5.3 Relationship Constraints

Generated modules must still obey module-contract, permission, and i18n rules. The generator cannot become a bypass around platform governance.

## 6. Current Migration Strategy

### 6.1 Phase 1: Compatible Split Permissions

Split coarse permissions into safer governed action layers.

### 6.2 Phase 2: Converged Page-Permission Semantics

Align generator page behavior and action gating with the wider permission model.

## 7. Page Behavior Requirements

The generator page should:

- make generated outputs and constraints explicit
- prevent fake capabilities
- show governed next steps rather than acting like a runtime designer studio

## 8. Audit and Security Requirements

Generation actions must remain auditable and environment-aware, with strong controls around workspace and registration impact.

## 9. Acceptance Requirements

### 9.1 Page Baseline

The page surface must be clear, governed, and operationally safe.

### 9.2 Permission Baseline

Permissions must separate read from high-sensitivity generation actions.

### 9.3 High-Sensitivity Action Baseline

Destructive or source-affecting operations must use the expected governance path.

### 9.4 Documentation Baseline

Generated-module behavior must still line up with the written module and i18n contracts.

## 10. P0 / P1 / P2 Completion List

### 10.1 P0: Controlled Generation Closure

Establish a safe, contract-aligned baseline.

### 10.2 P1: Multi-Table Business and Module Grouping

Support richer business modeling while preserving governance.

### 10.3 P2: Enterprise Governance Extensions

Deepen operational and policy control.

### 10.4 P2+: Enterprise Contract Boundaries

Keep expansion inside explicit governed boundaries.

## 11. Current Conclusion

The generator is a governed acceleration surface, not a replacement for architecture, contracts, or module review discipline.
