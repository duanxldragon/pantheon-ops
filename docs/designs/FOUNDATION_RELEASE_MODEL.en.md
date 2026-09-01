---
title: Foundation Release Model
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-08-04
---

# Foundation Release Model

Chinese version: [FOUNDATION_RELEASE_MODEL.md](./FOUNDATION_RELEASE_MODEL.md)

This document defines how `pantheon-base` should evolve continuously while downstream repositories consume only a standard foundation release instead of the unstable middle state of `main`.

## 1. Goal

`pantheon-base` must support both:

1. continuous security, quality, and architecture work
2. a stable consumption surface for downstream repositories such as `pantheon-ops`

The default model should move from:

```text
consumer repo -> track base/main
```

to:

```text
consumer repo -> consume base foundation release
```

## 2. Core Rules

### 2.1 `main` is not the consumer interface

`pantheon-base/main` is the continuous development line, not the default inheritance surface for downstream business repositories.

### 2.2 release is the consumer interface

Downstream repositories should consume one of:

- an explicit tag such as `pantheon-base-v0.10.0`
- an explicit release line such as `release/0.10`
- an emergency exception commit with a documented reason

Tracking `main` should not be the default.

### 2.3 base publishes, consumers upgrade

- `pantheon-base` publishes a gated foundation release
- `pantheon-ops` upgrades its local business overlay to that release

This is an upgrade model, not an informal file-sync model.

## 3. Asset Layers

### 3.1 Foundation-Owned

Owned by `pantheon-base`:

- `platform`
- `system/*`
- shared admin shell
- shared frontend components, shell, table, upload, and smoke helpers
- shared permission, i18n, audit, and menu contracts
- shared generator and governance foundation constraints

### 3.2 Consumer-Owned Overlay

Owned by downstream repositories:

- `business/*`
- business design and acceptance docs
- business smoke
- business seed, business i18n, and business menu integration

### 3.3 Integration Seams

Explicit seams that downstream repositories may extend:

- business module registry
- business component registry
- `business.*` i18n entries
- business menu mount points

The `business component registry` seam is discovered by the menu-contract gate (`frontend/scripts/check-menu-contract.mjs`) by name convention (frontend `*Registry.ts`, backend `*registry.go`). Consumer-injected overlay registries are therefore picked up by the gate without editing the base check script.

These seams must stay explicit, reviewable, and upgradeable.

## 4. Minimum Foundation Release Definition

A release fit for downstream consumption should include at least:

1. version identifier
2. release notes
3. consumer impact summary
4. upgrade notes
5. verification conclusion

Recommended tag format:

- `pantheon-base-v<major>.<minor>.<patch>`

### 4.1 Frontend shared-path ownership

`manifest.sharedPaths.frontend` is an executable ownership contract, not a one-time copy list. The foundation shell's `App.tsx`, `main.tsx`, and `vite-env.d.ts` must be declared explicitly. Shared transport and permission helpers must be declared through the `frontend/src/api` and `frontend/src/hooks` directories so new leaf files cannot silently remain in a consumer repository.

Release bundling rejects an existing generic frontend root that is not covered by this contract. `business/*`, declared overlays, and business-i18n seams remain consumer-owned and continue to follow the overlay upgrade rules.

## 5. Release Gate

A foundation release should not be cut until:

- GitHub required checks are green
- CodeQL has no unexplained reachable high-severity issues
- Security Hotspots are reviewed
- high-risk shared-foundation changes have independent review
- consumer upgrade impact is documented when relevant

The consumer impact must be made explicit when the release changes:

- permission model
- i18n-key semantics
- menu, route, or registry contracts
- generator or dynamic-module governance
- shared shell, table, upload, or smoke helpers
- `.github/workflows/*`

## 6. Consumer Upgrade Model

Recommended downstream sequence:

1. choose the target foundation release tag
2. update `docs/PROJECT_INHERITANCE.md`
3. run inheritance, sync, and drift checks
4. repair only real overlay breakpoints
5. run the downstream minimum verification set
6. record the upgrade result and any remaining gap

Downstream repositories should not:

- track `main` by default
- keep long-lived local overrides for shared foundation behavior
- copy base files continuously without a release boundary

## 7. Versioning Guidance

- `major`: breaking consumer contract or inheritance change
- `minor`: backward-compatible shared capability or governance improvement
- `patch`: security, quality, or compatibility fix

Even before a full packaging model exists, the minimum standard should be:

- stable tag
- release notes
- upgrade notes
- consumer impact summary

## 8. Direct Requirement For `pantheon-ops`

`pantheon-ops` should move toward:

- Base branch or release line: `release/<x.y>`
- Base version: `pantheon-base-v<x.y.z>`
- Inheritance mode: `foundation-release-consumer`

instead of:

- Base branch: `main`
- Base version: temporary commit pin

An unpublished commit should be an emergency exception only, with rollback and later-release reconciliation documented.
