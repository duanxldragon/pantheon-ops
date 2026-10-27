---
title: Dynamic Module Governance Design
doc_type: Design
layer: system/config
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-04-29
---

# Dynamic Module Governance Design

Chinese version: [DYNAMIC_MODULE_GOVERNANCE_DESIGN.md](./DYNAMIC_MODULE_GOVERNANCE_DESIGN.md)

This document defines the formal governance boundary of `system/config -> dynamicmodule`.

It answers:

- why dynamic modules are high-sensitivity capabilities
- how they relate to the generator
- which operations are allowed and which require stronger protection
- how environment guards, secondary confirmation, audit, and rollback should be understood

## 1. Module Positioning

Dynamic module capability belongs to:

- `system/config`

But it is not a normal settings-page feature. It is a platform module-assembly governance capability.

It can affect:

- the code workspace
- generated registries
- frontend and backend module assembly results
- the final set of accessible modules in the platform

It must therefore be treated as a high-sensitivity governance capability.

## 2. Target Boundary

### 2.1 What `dynamicmodule` Owns

- module registry maintenance
- registered-module status queries
- module registration
- module uninstall
- putting generated modules into a pending-activation state
- alignment of frontend and backend generated registries

### 2.2 What `dynamicmodule` Does Not Own

- defining business-module semantics directly
- replacing module-contract design
- providing mature runtime hot-plug behavior
- rewriting core `auth`, `iam`, `org`, or `config` boundaries

### 2.3 Relationship with `generator`

- `generator` produces the module scaffold
- `dynamicmodule` governs registration and assembly status

In one sentence:

`generator` produces artifacts, `dynamicmodule` governs integration.

## 3. Current Capability Scope

Current real capabilities include:

- querying module lists
- querying single-module status
- reattaching previously uninstalled modules
- uninstalling modules
- generating and registering modules

Frontend page:

- `/system/modules`

Backend API prefix:

- `/api/v1/system/dynamic-modules`

## 4. Environment Constraints

Dynamic-module capability should not be treated as permanently enabled production functionality.

Rules:

- enabled by default in development
- disabled by default in production
- production enablement must be explicit

Why:

- it changes the active module landscape
- it may write into the workspace or rewrite registries
- it is not ordinary business-data CRUD

It must not be treated like a casual admin page that any administrator can use freely.

## 5. Risk Classification

- list modules: low risk
- query module state: low risk
- reattach module: high risk
- uninstall module: high risk
- generate and register: high risk

## 6. Permission Model

Page permission:

- `system:module:list`

Action permissions:

- `system:module:register`
- `system:module:unregister`

Rules:

- page permission only grants entry to the module-management page
- register, uninstall, and generate flows require dedicated action permissions
- backend APIs still require Casbin policy enforcement
- if `system:module:generate` becomes separate later, it should stay distinct from generic register permission

## 7. Secondary Confirmation and Environment Guards

Dynamic-module write operations require two protection layers.

### 7.1 Environment Guard

Used to block prohibited environments, for example:

- production closed by default
- explicit enablement required before write operations are accepted

### 7.2 Secondary Confirmation

Used to protect high-sensitivity actions such as:

- register module
- uninstall module
- generate and register module

These are not ordinary form saves. One mistake can directly affect system availability.

## 8. State Model

Dynamic modules should not be modeled as only present or absent.

Minimum state model:

- unregistered
- pending activation
- attached
- uninstalled
- failed

The crucial distinction is between “files written” and “runtime actually effective”.

## 9. Audit Requirements

All dynamic-module write operations must enter unified audit.

At minimum record:

- operator
- module name
- module scope
- action type
- execution result
- failure reason
- timestamp

For generate-and-register flows, also record:

- target generation path
- affected generated registries
- result summary

Audit must be specific enough to reconstruct what module changed and which assembly files were affected.

## 10. Rollback and Failure Handling

Dynamic-module governance is incomplete if it has no rollback mindset.

### 10.1 Registration Failure

- return a clear failure reason
- do not leave half-registered state behind

### 10.2 Uninstall Failure

- avoid partial frontend-backend registry desynchronization
- default uninstall should detach integration rather than delete workspace source code

### 10.3 Generate-and-Register Failure

- clearly distinguish scaffold generation failure, file-write failure, registration failure, and environment blocking

Recommended rollback principle:

- repair registration-state consistency first
- then clean generated artifacts or hand off to manual recovery

Full automatic rollback is not required today, but the need for it must be acknowledged as high-priority governance work.

## 11. Collaboration Flow with `generator`

The generator creates the scaffold and passes it into the dynamic-module governance flow, which then manages pending activation, registry alignment, and final attach state.

## 12. Acceptance Requirements

### 12.1 Page Baseline

Verify module list and status surfaces are readable and stable.

### 12.2 Permission Baseline

Verify page-entry and action permissions are separated correctly.

### 12.3 High-Sensitivity Action Baseline

Verify environment guards and secondary confirmation protect write actions.

### 12.4 Result Baseline

Verify success, failure, pending activation, and audit traces are all distinguishable.

## 13. Current Conclusion

`dynamicmodule` is a high-sensitivity `system/config` governance capability. It manages module integration state and must be protected, auditable, and rollback-aware rather than treated as ordinary admin CRUD.
