---
title: Alert and Monitoring Center Design
doc_type: Design
layer: platform / system/config / business/*
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-05-18
---

# Alert and Monitoring Center Design

Chinese version: [ALERT_MONITORING_CENTER_DESIGN.md](./ALERT_MONITORING_CENTER_DESIGN.md)

This document defines the alert and monitoring center as a platform-level product surface that aggregates alert signals without collapsing source-domain ownership.

## 1. Design Goals

- provide one unified alert entry instead of scattered warning surfaces
- let multiple source domains publish alerts into one platform view
- keep severity, state, actions, audit, and routing behavior consistent

## 2. Non-Goals

Not in scope now:

- full observability platform replacement
- arbitrary metrics pipeline design
- domain-specific incident playbooks embedded into the platform shell

## 3. Ownership Boundary

### 3.1 What `platform` Owns

- the alert-center entry
- unified list and status views
- cross-domain aggregation behavior

### 3.2 What `system/config` Owns

- configuration-level alert policy and related governance hooks where applicable

### 3.3 What Source Domains Own

- alert semantics
- rule triggers
- domain-specific handling logic

## 4. Alert Model

Typical fields should include source domain, alert type, severity, state, title, summary, timestamps, and jump target.

### 4.1 Severity

Use a stable severity model such as info, warning, and critical rather than domain-local ad hoc labels.

### 4.2 State

Use explicit states such as open, acknowledged, resolved, or muted rather than ambiguous free text.

## 5. Core Action Contract

The first-phase action contract should stay small and explicit, for example:

- acknowledge
- resolve
- mute
- jump to source detail

Action availability is still decided by the source domain and permission model.

## 6. UI Skeleton

The center should provide:

- a unified list
- severity and state filters
- source-domain grouping or filtering
- row-level navigation to source context

## 7. Audit and Permissions

At minimum, the platform must distinguish:

- center access
- alert viewing
- alert action permissions
- audit traces for acknowledgment and resolution

## 8. Relationship with Notice Center, Scheduler Center, and Dashboard

- dashboard shows summary-level attention signals
- notice center carries user-facing informational messages
- scheduler center focuses on scheduled execution
- alert center is for operational signals that need explicit monitoring and handling

## 9. Minimum Pilot Requirement

At least one real alert-producing domain should connect before the center is treated as complete.

## 10. Definition of Done

Done means:

- unified alert model exists
- platform entry exists
- source-domain jump behavior is defined
- severity and state rules are consistent
- permissions and audit traces exist
