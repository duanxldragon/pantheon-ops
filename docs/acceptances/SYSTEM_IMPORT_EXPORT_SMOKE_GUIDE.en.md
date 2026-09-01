---
title: System-Domain Import/Export Smoke Guide
doc_type: Acceptance
layer: system/config
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/contracts/SYSTEM_ORG_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-05-19
---

# System-Domain Import/Export Smoke Guide

Chinese version: [SYSTEM_IMPORT_EXPORT_SMOKE_GUIDE.md](./SYSTEM_IMPORT_EXPORT_SMOKE_GUIDE.md)

This guide validates whether import and export capabilities across `system/iam`, `system/org`, `system/config`, `system/auth`, and `system/audit` are actually usable.

The repository already provides reusable API smoke coverage:

```powershell
cd frontend
cmd /c npm run test:smoke:system:api
```

## Main Coverage

- users, permissions, departments, posts, dictionaries
- export-only flows for login logs and operation logs
- shared token acquisition and API-based smoke verification

## Typical Validation Loop

1. download the backend template
2. prepare a sample CSV
3. run import
4. confirm data persistence through list APIs
5. run export
6. inspect the exported file

Use the Chinese source document for the complete PowerShell examples, fixture paths, and per-module smoke steps.
