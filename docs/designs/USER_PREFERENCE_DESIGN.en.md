---
title: User Preference Design
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
updated_at: 2026-05-18
---

# User Preference Design

Chinese version: [USER_PREFERENCE_DESIGN.md](./USER_PREFERENCE_DESIGN.md)

This document defines Pantheon Base user preferences as platform-shell behavior rather than as ad hoc page-local settings.

## 1. Design Goals

- keep shell-level preferences explicit and stable
- separate session overrides from long-term user preferences
- prevent domain-local settings from polluting the platform shell

## 2. Non-Goals

Not in scope:

- arbitrary business-domain personal settings inside the platform preference model
- collapsing every UI behavior into one generic JSON blob without boundary

## 3. Ownership Boundary

### 3.1 What `platform` Owns

- shell preference semantics
- layout, density, and similar shared behavior

### 3.2 What Other Domains Own

- their own business-specific user settings
- auth-domain persistence entry points where platform preferences are stored or retrieved

## 4. Preference Scope

Typical platform preferences include:

### 4.1 `navMode`

Navigation presentation mode.

### 4.2 `tableDensity`

Shared visual density for table-like admin surfaces.

### 4.3 `workbenchLayout`

Platform workbench layout preference.

## 5. Session Overrides and Long-Term Preferences

Pantheon must distinguish:

- temporary session overrides
- durable user preferences

Runtime resolution should not blindly overwrite session choices with long-term stored values.

## 6. Storage Model

Platform preferences may be physically stored through auth-related user preference persistence, but their semantics remain platform-owned.

## 7. Frontend Constraints

- preferences should affect shared shell behavior consistently
- page-local components should consume them through one stable preference path
- preference changes should not require page reload

## 8. Backend Constraints

- preference persistence must validate allowed values
- storage shape should stay bounded and documented
- platform preference storage must not become a dumping ground for unrelated business settings

## 9. Audit and Security

Preference changes that alter shell behavior meaningfully should remain auditable, while sensitive or unrelated domain settings stay outside this model.

## 10. Definition of Done

Done means:

- preference scope is explicit
- session-versus-long-term semantics are explicit
- storage and validation rules are explicit
- platform shell behavior consumes preferences consistently
