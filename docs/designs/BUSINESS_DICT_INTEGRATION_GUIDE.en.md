---
title: Business Dictionary Integration Guide
doc_type: Design
layer: system/config / business/*
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-05-05
---

# Business Dictionary Integration Guide

Chinese version: [BUSINESS_DICT_INTEGRATION_GUIDE.md](./BUSINESS_DICT_INTEGRATION_GUIDE.md)

This document defines how business modules should consume `system/config/dict` without hardcoding enums or mixing business dictionary semantics into the base system vocabulary.

## 1. Boundary

Business dictionaries should consume the shared dictionary capability while preserving their own business meaning.

## 2. Naming Rules

Dictionary names and keys should stay explicit, stable, and distinguish business-domain meaning from shared system-domain enums.

## 3. Integration Method

Business modules should:

- resolve options through shared dictionary APIs
- keep user-facing labels key-driven
- avoid scattering local enum constants that diverge from dictionary truth

## 4. Reference Protection

Modules must protect against stale or deleted dictionary references and avoid silent business-state corruption when dictionary values change.

## 5. CMDB Example

CMDB remains the concrete reference example for how business domains can consume shared dictionary capability without polluting system semantics.

## 6. Acceptance

Acceptance should verify:

- dictionary consumption uses shared capability
- labels remain translatable
- business meaning stays explicit
- references remain protected against invalid values or drift
