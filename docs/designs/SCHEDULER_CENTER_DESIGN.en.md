---
title: Scheduler Center Design
doc_type: Design
layer: platform / system/config / business/*
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-05-18
---

# Scheduler Center Design

Chinese version: [SCHEDULER_CENTER_DESIGN.md](./SCHEDULER_CENTER_DESIGN.md)

This document defines the scheduler center as the unified platform surface for scheduled execution governance.

## 1. Design Goals

- provide one scheduler-center entry instead of scattered cron or task execution views
- let different domains expose schedulable work through one governance surface
- keep execution state, schedule type, actions, and audit consistent

## 2. Non-Goals

Not in scope now:

- full workflow orchestration engine
- arbitrary script execution platform
- replacing all domain-local execution logic

## 3. Ownership Boundary

### 3.1 What `platform` Owns

- scheduler-center entry
- unified scheduling list and status views

### 3.2 What `system/config` Owns

- configuration-level scheduling governance where applicable

### 3.3 What Source Domains Own

- execution semantics
- task payload meaning
- domain-specific side effects

## 4. Scheduled-Task Model

Typical fields should include task type, source domain, schedule type, execution state, last run, next run, and jump target.

### 4.1 Execution State

Use explicit execution states such as pending, running, success, failed, paused, or cancelled.

### 4.2 Schedule Type

Schedule type should distinguish manual, recurring, one-off, or other clearly defined trigger classes.

## 5. Core Action Contract

Core actions may include:

- run now
- pause
- resume
- cancel
- jump to source context

## 6. UI Skeleton

The first phase should provide:

- scheduler list
- execution-state and schedule-type filters
- row-level actions
- source-context navigation

## 7. Audit and Permissions

At minimum, distinguish:

- center access
- task viewing
- scheduler action permissions
- audit traces for run, pause, resume, or cancel

## 8. Relationship with Alert Center

Scheduler center governs scheduled execution. Alert center governs operational signals. Failed or risky scheduled tasks may emit alerts, but the two centers must remain conceptually separate.

## 9. Minimum Pilot Requirement

At least one real schedulable source domain should integrate before the center is treated as complete.

## 10. Definition of Done

Done means:

- unified scheduled-task model exists
- platform entry exists
- action contract is explicit
- permissions and audit are defined
- a real pilot source is connected
