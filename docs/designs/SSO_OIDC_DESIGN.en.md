---
title: SSO / OAuth2 / OIDC Design
doc_type: Design
layer: system/auth
status: Draft
linked_contracts:
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
updated_at: 2026-05-05
---

# SSO / OAuth2 / OIDC Design

Chinese version: [SSO_OIDC_DESIGN.md](./SSO_OIDC_DESIGN.md)

This document defines the future boundary for Pantheon SSO / OAuth2 / OIDC integration.

Pantheon does not implement real SSO yet. This document exists so that future identity-source integration does not get patched in without auth-domain discipline.

## 1. Goal

Define a stable future SSO boundary before implementation begins.

## 2. Non-Goals

Do not treat this draft as proof that a real provider is already integrated.

## 3. Core Model

The design should eventually define:

- provider identity
- subject mapping
- local-user binding
- callback and session behavior

## 4. Login Flow

The future flow should be explicit about redirection, callback handling, local-user resolution, and Pantheon session issuance.

## 5. Local Login Fallback

Local fallback must remain available for administrative recovery and provider-failure scenarios.

## 6. Security Constraints

The eventual implementation must define:

- callback safety
- provider trust boundaries
- state and nonce handling
- session safety
- audit and risk-event traces

## 7. Acceptance

When implementation starts, acceptance should verify:

- real provider connectivity
- safe callback handling
- local-user mapping correctness
- fallback-login behavior
- permission and audit consistency
