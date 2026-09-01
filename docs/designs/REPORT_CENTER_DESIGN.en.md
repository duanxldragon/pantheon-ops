---
title: Report Center Design
doc_type: Design
layer: platform / business/*
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-05-18
---

# Report Center Design

Chinese version: [REPORT_CENTER_DESIGN.md](./REPORT_CENTER_DESIGN.md)

This document defines the report center as a unified platform surface for report discovery, export, and governance while leaving report semantics with source domains.

## 1. Design Goals

- provide one report-center entry
- unify report metadata and visibility behavior
- keep export and access rules explicit

## 2. Non-Goals

Not in scope now:

- general BI platform replacement
- free-form SQL authoring for end users
- domain-specific analytics logic inside the platform shell

## 3. Ownership Boundary

### 3.1 What `platform` Owns

- report-center entry
- report metadata presentation
- cross-domain report discovery

### 3.2 What Source Domains Own

- report content semantics
- report generation logic
- domain-specific filters and constraints

## 4. Report Metadata Model

Metadata should include at least:

- report type
- source domain
- title
- visibility
- supported export formats
- jump or download target

### 4.1 Visibility

Visibility must follow explicit permission and menu rules rather than implicit existence.

### 4.2 Export Format

Supported formats should be declared explicitly and auditable.

## 5. Core Action Contract

Core actions may include:

- view report definition
- run or preview
- export
- jump to source context

## 6. UI Skeleton

The first phase should include:

- report list
- source-domain filters
- visibility-aware actions
- export entry points

## 7. Audit and Permissions

At minimum, distinguish:

- report-center access
- report visibility permission
- export permission
- audit traces for report execution or export where required

## 8. Relationship with Scheduler Center and Dashboard

- scheduler center may automate report generation
- dashboard may surface summary metrics
- report center owns discoverability and report-specific access

## 9. Minimum Pilot Requirement

At least one real report-producing domain should integrate before the center is treated as complete.

## 10. Definition of Done

Done means:

- unified report metadata model exists
- center entry exists
- export and visibility rules are explicit
- one or more real report sources are connected
