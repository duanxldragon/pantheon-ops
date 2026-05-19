# Pantheon Base Enterprise Backoffice Design

Chinese version: [DESIGN.md](./DESIGN.md)

This document defines the high-level design stance inherited by `pantheon-ops` from the broader Pantheon platform approach.

The goal is not another generic admin shell with login, menu, and CRUD only. The target is a multilingual, contract-driven enterprise backoffice foundation with clear separation between platform capabilities and business modules.

## Core Principles

- stable foundation domains: auth, IAM, org, config, audit, i18n
- business decoupling from foundation implementation details
- registration-first extension for menus, permissions, dictionaries, and settings
- AI-readable structure, naming, and boundaries
- restrained backoffice visual language instead of marketing-style UI noise

## Layering

Recommended layers:

- platform shell
- system foundation domains
- business domains

Business modules should consume shared contracts and context, not direct internals from system modules.

## Foundation Domains

Key base domains remain:

- `auth`
- `iam`
- `org`
- `i18n`
- `audit`
- `dict`
- `setting`

`auth` should be logically separated from mixed user-management code, even if deployment remains a modular monolith.

## Required Enterprise Capabilities

Priority bands:

- P0: auth, authorization, org, navigation, i18n, audit
- P1: session management, permission workbench, dictionary management, system settings
- P2: multi-tenant readiness, SSO/OIDC, MFA and risk control, data permissions, theming, dashboards

## Authorization Model

The intended model separates:

- menus
- page permissions
- action permissions
- API permissions

List permission must not stand in for create, update, delete, or detail access.

## Dynamic Menus and I18n

Menus should remain metadata-driven and store `title_key` rather than localized strings directly.

All user-facing content must be localized, including:

- menus
- buttons
- forms
- errors
- empty states
- confirmation dialogs

Backend responses should return stable keys rather than natural-language messages.

## UI Guidance

The inherited visual language is a calm, trustworthy, tool-like enterprise backoffice:

- stable shells
- neutral surfaces
- restrained borders and spacing
- unified page states
- shared typography and token rules

The document also defines anti-patterns to avoid, such as decorative gradients, heavy glow effects, and ad hoc visual systems.

## Reading Order

Before implementation, read in order:

1. `DESIGN.md`
2. `AGENTS.md` / `agent.md`
3. `docs/README.md`
4. contracts
5. architecture and frontend/backend design docs
6. module and acceptance docs

For the full Chinese-first design rationale, visual token rules, roadmap status, and the detailed 45-item reading order, use the Chinese source document.
