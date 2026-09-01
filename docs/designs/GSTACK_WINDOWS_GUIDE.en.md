---
title: gstack Windows Usage Guide
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-04-20
---

# gstack Windows Usage Guide

Chinese version: [GSTACK_WINDOWS_GUIDE.md](./GSTACK_WINDOWS_GUIDE.md)

This document describes how to use the gstack built-in browser reliably on Windows for Pantheon local smoke testing and page inspection.

## 1. Scope

Applies to platform and system-domain local QA and smoke flows on Windows.

## 2. Current Environment Conclusion

Windows can support the expected local browser-driven QA path, but it requires known operational patterns rather than ad hoc command usage.

## 3. Confirmed Working Paths

The guide documents what has already been confirmed to work reliably in the current environment.

## 4. Recommended Usage

### 4.1 Prefer One `browse chain`

Use one chained browser flow rather than fragmented calls whenever possible.

### 4.2 Login by API First, Then Inject Tokens

Token injection after API login is the preferred path for stable local authenticated smoke flows.

## 5. Common Problems

### 5.1 `spawn EPERM`

Treat as a likely environment or elevation issue first.

### 5.2 `No active page`

Treat as a context-loss issue first, not immediately as a product defect.

### 5.3 Screenshot Timeout

Retry the browser chain and separate tool instability from true page failure.

## 6. Optional Setup Paths

### 6.1 `setup-browser-cookies`

Use only when authenticated cookie import is actually needed.

### 6.2 `open-gstack-browser`

Use when visible browser observation is necessary.

## 7. Recommended Pantheon Local Smoke Order

Use one stable sequence for local page verification rather than random page hopping.

## 8. Conclusion

The Windows path is workable and should be treated as a governed local QA workflow, not as improvised browser usage.
