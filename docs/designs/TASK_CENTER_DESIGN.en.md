---
title: Task Center Design
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-05-18
---

# Task Center Design

Chinese version: [TASK_CENTER_DESIGN.md](./TASK_CENTER_DESIGN.md)

This document defines the task center as the unified platform surface for cross-domain actionable work items.

## 1. Design Goals

- provide one task-center entry instead of per-module todo islands
- let task-producing domains converge on one product surface
- keep task state, view, jump, and permission behavior consistent

## 2. Non-Goals

Not in scope now:

- replacing every domain workflow screen
- generic workflow engine design
- unrelated notification or alert functionality inside the task center itself

## 3. Ownership Boundary

### 3.1 What `platform` Owns

- task-center entry
- unified task views
- task aggregation behavior

### 3.2 What Task Source Domains Own

- task semantics
- completion conditions
- source-page handling logic

## 4. Task Model

The unified task model should include at least title, source domain, task type, status, assignee, timestamps, and jump target.

### 4.1 Status

Use explicit states such as pending, in progress, completed, or cancelled.

### 4.2 Views

The center should distinguish views such as tasks assigned to me, tasks created by me, or other platform-defined work lists.

## 5. Jump Contract

Task-center rows must navigate safely to known source pages while respecting permissions and route validity.

## 6. UI Skeleton

First-phase UI should include:

- unified task list
- view tabs
- state filtering
- safe source-page jump behavior

## 7. Relationship with Notice Center and Dashboard

- dashboard may surface task summaries
- notice center may inform users about events
- task center owns actionable work lists

## 8. Audit and Permissions

At minimum, define:

- center access permission
- task visibility rules
- task-action permissions if the center allows inline actions
- audit for task-handling actions where required

## 9. Definition of Done

Done means:

- unified task model exists
- task-center entry exists
- source-domain jump behavior is explicit
- at least one real task source is connected
