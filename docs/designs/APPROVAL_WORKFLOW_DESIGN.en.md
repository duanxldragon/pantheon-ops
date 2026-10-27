---
title: Approval Workflow Skeleton Design
doc_type: Design
layer: platform / business/*
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-05-18
---

# Approval Workflow Skeleton Design

Chinese version: [APPROVAL_WORKFLOW_DESIGN.md](./APPROVAL_WORKFLOW_DESIGN.md)

This document defines the “approval workflow skeleton” for Pantheon’s standard backoffice product surface.

The current goal is not to build a full BPM platform. The goal is to converge on:

- one unified approval center entry
- one shared approval-item model
- one minimal action contract
- one consistent audit story

for views such as pending approvals, processed approvals, and my submissions.

## 1. Design Goals

The skeleton should ensure that:

1. the platform has one approval entry instead of each module inventing its own todo center
2. different business domains can plug into one unified approval view while preserving their own domain semantics
3. approval actions, state transitions, and audit traces are expressed consistently

## 2. Non-Goals

Not in scope now:

- visual workflow designers
- general-purpose rule orchestration engines
- arbitrary node-level script execution
- complex multi-party organizational approval choreography

## 3. Ownership Boundary

### 3.1 What `platform` Owns

- the approval center entry
- the unified approval-item list model
- shared status views
- the aggregate relationship with the task center

### 3.2 What `business/*` Owns

- business meaning of the approval item
- step-transition rules
- detail pages and handling actions
- domain-specific audit supplements

In one sentence:

`platform` owns the approval product surface, while business domains own the actual approval handling logic.

## 4. Approval Item Model

Recommended minimum fields:

- `id`
- `workflowType`
- `sourceDomain`
- `bizType`
- `bizId`
- `title`
- `status`
- `currentStep`
- `applicantId`
- `currentApproverId`
- `submittedAt`
- `processedAt`
- `jumpTarget`

### 4.1 States

Minimum states:

- `draft`
- `submitted`
- `pending`
- `approved`
- `rejected`
- `cancelled`

### 4.2 Default Views

Minimum views:

- `pending-for-me`
- `processed-by-me`
- `submitted-by-me`

## 5. Action Contract

First-phase unified action semantics:

- `approve`
- `reject`
- `transfer`
- `cancel`

Rules:

- action availability is decided by the source domain
- the platform layer does not guarantee that every approval item supports every action
- the frontend must not show fake buttons without backend semantics

## 6. Relationship with the Task Center

Approval workflows and the task center should not become two parallel, unrelated models.

Recommended relationship:

- the approval item is the business object
- approval todos are one task type inside the task center
- the task center shows approval summaries
- the approval center provides approval-specific filters and handling entry points

## 7. UI Skeleton

First-phase minimum UI:

- approval-center list page
- three tabs: pending for me, processed by me, submitted by me
- status filters
- row-level jump to source-domain detail or handling page

Not required yet:

- drag-and-drop flow diagrams
- node-level visual workflow composition
- approval analytics dashboards

## 8. Audit and Permissions

At minimum, the system needs:

- approval-center access permission
- approval-item view permission
- approval-action permissions
- audit records for approval actions

Audit must retain:

- applicant
- processor
- action
- time
- result
- summary note

## 9. Minimum Pilot Requirements

The first phase must connect at least one real business scenario so the approval center does not remain a shell.

Recommended pilots:

- approval for enabling or disabling business resources
- high-sensitivity configuration-change approval
- asset-entry review

## 10. Definition of Done

First phase is complete when Pantheon has:

- a unified approval-item model
- an approval-center entry
- a minimal action contract
- at least one real business pilot
- permission and audit traces

It is not measured by whether a full BPM platform already exists.
