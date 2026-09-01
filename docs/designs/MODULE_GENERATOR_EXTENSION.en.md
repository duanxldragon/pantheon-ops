---
title: Module Generator Extension Design
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-06-26
---

# Module Generator Extension Design

Chinese version: [MODULE_GENERATOR_EXTENSION.md](./MODULE_GENERATOR_EXTENSION.md)

## 1. Background

The current `backend/internal/scaffold/` module generator does not include best practices:

- Does not automatically import `pkg/common` utilities
- Does not use `StatusEnabled/StatusDisabled` constants
- Missing pagination, import/export utilities

This document defines the generator template extension specification.

---

## 2. Extension Plan

### 2.1 Backend Template

```go
// New imports in generated code
import (
    "pantheon-platform/backend/pkg/common"
    "pantheon-platform/backend/pkg/impexp"
)

// Use status constants
status := common.NormalizeEnabledStatus(req.Status)

// Use batch deduplication
ids := common.NormalizeUint64IDs(req.IDs)
```

### 2.2 Frontend Template

```typescript
// New imports
import { normalizePageQuery } from "@/utils/pagination";
import { downloadFile, uploadImportFile } from "@/api";

// Auto-generated API methods
export const downloadTemplate = () => downloadFile("/module/import-template");
export const exportItems = (params) => downloadFile("/module/export", params);
export const importItems = (file: File) =>
  uploadImportFile("/module/import", file);
```

---

## 3. Acceptance Criteria

- [ ] Generated modules auto-import `pkg/common`
- [ ] Status fields use `common.StatusEnabled/StatusDisabled`
- [ ] Batch operations use `common.NormalizeUint64IDs`
- [ ] Frontend APIs include import/export methods
- [ ] Generated code passes lint checks
