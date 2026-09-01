---
title: Platform Dashboard Design
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-05-01
---

# Platform Dashboard Design

Chinese version: [PLATFORM_DASHBOARD_DESIGN.md](./PLATFORM_DASHBOARD_DESIGN.md)

This document defines the Pantheon platform dashboard as a cross-domain aggregate view owned by the `platform` layer.

## 1. Ownership Boundary

The platform dashboard belongs to `platform`, not to any single `system/auth`, `system/iam`, `system/org`, or `system/config` subdomain.

It remains a logical platform layer even if the physical source directory is `backend/modules/dashboard/` or `frontend/src/modules/dashboard/`.

Why:

- the dashboard is a cross-domain aggregate view
- it reads multiple system-domain summaries without owning them
- future business-domain summary cards should still attach to the platform aggregate layer

## 2. Current API

- `GET /api/v1/platform/dashboard/summary`

Authentication rule:

- login required
- no extra page permission required
- used as the data source for the home workbench and platform overview cards

## 3. Current Metrics

Current summary includes metrics such as:

- user counts
- role, department, and post counts
- dictionary and setting counts
- visible menu count
- active session count
- login success and failure counts
- today operation count
- last successful login time
- recent login activity
- period days

These are lightweight aggregate indicators, not replacements for full audit or security-detail pages.

## 4. Frontend Presentation Rules

The homepage uses a light-workbench mode:

- first row for primary metrics, capped tightly
- second row for security posture, attention items, and quick actions
- third row for domain-level summaries such as IAM, org, config, and audit
- lower sections for recent activity

Rules:

- do not build a same-sized card wall
- do not repeat the same KPI in multiple first-screen zones
- separate hero, attention, and overview responsibilities clearly
- cap quick-entry count and route overflow back to menu navigation or global search

Recommended structure:

- `StatusStrip`
- `PrimaryActions`
- `AttentionPanel`
- `DomainOverview`
- `RecentActivity`

## 5. Future Evolution

The dashboard may later add:

- todo and notice cards
- audit trend charts
- setting-change reminders
- registered workbench cards from business modules

Rules:

- every new card must declare its owning layer
- cross-domain aggregation stays in `platform`
- business-domain cards must register rather than being hardcoded
- business widgets must also declare cleanup behavior when their source module is removed

## 6. Current Registration Contract

The frontend workbench currently converges on `ModuleConfig.dashboardWidgets`.

Rules:

- each module declares its own dashboard widgets
- the platform aggregates widgets from registered modules
- `system/*` and `business/*` both use the same registration surface
- business widgets must additionally declare ownership and cleanup policy
- widget visibility still respects navigation and permission boundaries
