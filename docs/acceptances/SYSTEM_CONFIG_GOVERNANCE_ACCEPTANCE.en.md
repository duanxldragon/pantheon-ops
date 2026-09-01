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
