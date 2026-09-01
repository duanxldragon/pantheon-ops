---
title: Notice Center Design
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-05-18
---

# Notice Center Design

Chinese version: [NOTICE_CENTER_DESIGN.md](./NOTICE_CENTER_DESIGN.md)

This document defines the notice center as the unified platform surface for user-facing notices and message routing.

## 1. Design Goals

- provide one notice center instead of scattered module-local message boxes
- let system and business domains publish into one user-facing channel
- keep message category, severity, state, and jump behavior consistent

## 2. Non-Goals

Not in scope now:

- full enterprise messaging platform
- arbitrary campaign or marketing notification capabilities
- replacing all domain-specific in-page feedback

## 3. Ownership Boundary

### 3.1 What `platform` Owns

- the notice-center entry
- shared list and reading experience
- navigation and jump orchestration

### 3.2 What System or Business Domains Own

- message semantics
- source payloads
- jump targets
- domain-specific generation logic

## 4. Message Model

Messages should carry at least:

- category
- severity
- state
- title
- summary
- source domain
- jump target
- timestamps

### 4.1 Category

Categories should be stable and product-meaningful rather than arbitrary source strings.

### 4.2 Severity

Severity should express urgency consistently across domains.

### 4.3 State

States such as unread, read, archived, or dismissed should stay explicit.

## 5. Jump Contract

### 5.1 Jump Rules

Jump targets must:

- route to known platform paths
- respect permissions
- avoid broken or stale targets

The notice center should not expose messages that lead to impossible or fake destinations.

## 6. UI Skeleton

The first-phase UI should include:

- unified message list
- category and state filters
- unread/read handling
- safe source-page jump behavior

## 7. Audit and Trace

At minimum, record:

- message publication
- read or dismiss actions where relevant
- jump behavior for operational traceability if needed

## 8. Relationship with Dashboard

Dashboard may show summaries or highlights, but the notice center owns the full user-facing notice stream and reading experience.

## 9. Definition of Done

Done means:

- platform notice entry exists
- message model is explicit
- jump behavior is governed
- state handling is consistent
- at least one real source domain is connected
