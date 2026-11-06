---
title: Data Permission Hook
doc_type: Design
layer: system/iam
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-11-04
---

# Data Permission Hook

Chinese version: [DATA_PERMISSION_HOOK.md](./DATA_PERMISSION_HOOK.md)

This document defines Pantheon’s implemented shared contract for data permissions, especially row-level filtering for department scope, self scope, or future custom scope dimensions.

## 1. Core Contract

### 1.1 Data Structure

The shared data-scope structure lives in a common contract area and is meant to be reused rather than reinvented by modules.

### 1.2 GORM Hook

The shared database scope hook provides current row-level filtering and the extension point for additional dimensions.

## 2. Usage Examples

### Backend Service Layer

Services should consume shared data-scope structures rather than inventing module-local filtering models.

### Backend Handler Layer

Handlers should pass authenticated and permission-resolved context into shared scope handling rather than scattering direct conditions.

## 3. Future Extension Path

This hook is the current extension point for richer data-scope strategy evolution.

### 3.1 Multi-Role Merge Rules

Current additive merge: `admin` bypasses filtering; any `all` grants all; custom department IDs form a union; otherwise choose `dept_and_children`, then `dept`, then `self`. Results are independent of database row order. No deny/intersection semantics are implied.

## 4. Why Keep a Shared Hook

The shared hook protects current IAM behavior. System tenant scope follows TENANT_CONTRACT_V1; V1 business modules remain single-tenant even when using DataScope. Ops CMDB is a downstream example, not a Base-owned module.
