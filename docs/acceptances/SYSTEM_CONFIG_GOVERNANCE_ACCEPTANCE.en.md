---
title: system/config Sensitive Governance Acceptance Baseline
doc_type: Acceptance
layer: system/config
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-05-19
---

# system/config Sensitive Governance Acceptance Baseline

Chinese version: [SYSTEM_CONFIG_GOVERNANCE_ACCEPTANCE.md](./SYSTEM_CONFIG_GOVERNANCE_ACCEPTANCE.md)

This baseline puts `/system/i18n`, `/system/modules`, and `/system/generator` under a fixed acceptance surface.

### V2.0-O4 Incremental Generator Acceptance (in progress)

These are new targets under the [tenant contract section 2.7](../contracts/TENANT_CONTRACT_V1.en.md), not covered by older passing baselines. Parent `2026-11-07-ops-formal-queue-foundation-gaps` records final evidence and independent conclusions.

| Boundary                | Required evidence                                                                                                                                                       |
| :---------------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Consistent entry        | Wizard/scaffold/direct export reject V1 tenant, system tenant, conflicting ownership fields and unsupported relations; legacy V1 runs                                   |
| Generated runtime       | Actual flat V2 export runs default/development MySQL A/B CRUD/options, duplicate-key isolation and missing-context denial; tenant predicates survive disabled DataScope |
| Required initialization | Bootstrap/schema/migrate/seed failure exposes no adopted ready route; test actual generated Init and legacy V1 runner separately                                        |
| UI/i18n                 | All supported locales, desktop/mobile, light/dark, keyboard and validation rendering; explicit V2/tenant opt-in and V1 reset                                            |
| Inheritance             | Stable final-source hashes, candidate pairing, independent review and later official release consumption; template-string tests alone do not pass                       |

The [generator guide](../designs/LOWCODE_GENERATOR_GUIDE.en.md) provides the runtime entry. Scoped fixtures do not prove the full consuming product's current authorization, migrations or private-data isolation.

It exists to close the gap between design and implementation for high-sensitivity configuration governance pages.

## Covered Areas

- translation assets and cache refresh
- dynamic-module registration, unregister, purge, and record cleanup
- generator execution and datasource governance

## Common Acceptance Themes

- page and action permissions must stay separated
- backend permissions must cover real write paths
- failures must distinguish permission, environment, verification, business, and tool errors
- all display text must remain i18n-driven
- high-risk writes must be audited
- dangerous operations need secondary verification or equivalent safety gates

Use the Chinese source document for the full permission tables, automation commands, and completion definition.
