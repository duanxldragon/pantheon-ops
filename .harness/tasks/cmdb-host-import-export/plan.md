# Task: CMDB Host Import/Export (CSV, Excel-compatible)

Implement batch import (Excel/CSV) + asset export for CMDB host management in pantheon-ops, following the exact patterns already established by the system modules (role/user/dept/post/audit).

## Read these first (existing patterns to mirror)

- Backend host module: `backend/modules/business/cmdb/host/` — `host_handler.go`, `host_service.go`, `host_model.go`, `host_dto.go`
- Import/export infra: `backend/pkg/impexp/` — `CSVFile`, `WriteCSV`, `ReadCSV`, `ImportResult`, `AppendImportError`, `ReadCSVField`, `IsCSVRecordEmpty`, `ParseCSVInt`
- Reference implementation (mirror closely): `backend/modules/system/iam/role/role_export.go` + `role_handler.go` (Export / Import / BuildTemplate)
- Frontend reference (mirror closely): `frontend/src/modules/system/post/PostList.tsx` + `post/api.ts` (utility buttons: export / template / import via `ImportCsvButton`, `downloadFile`, `uploadImportFile`, `showImportResult`)
- Permission/menu/i18n seeds: `backend/modules/business/cmdb/module.go`

## Data model

Table `biz_cmdb_host` (`Host` struct in `host_model.go`). Key fields: `Hostname` (string), `IP` (string, unique index `uk_ip_deleted`), `SSHPort` (int, default 22), `OS` (string), `OSVersion`, `CPUCores` (int), `MemoryGB` (float64), `DiskGB` (float64), `LabelValues` (datatypes.JSON = `[]LabelEntry{Key,Val}`), `Status` (string, default pending), `BusinessScopeID/Code/Name` (uint64/string/string), `DeptID` (uint64), `Owner` (string), `Remark` (text), `CreatedAt/UpdatedAt`, `CreatedBy/UpdatedBy`, `DeletedAt` (gorm.DeletedAt).

The `biz_business_scope` table has columns `id`, `code`, `name` (used to resolve `businessScopeCode` → id/code/name during import).

## Backend changes

### 1. `backend/modules/business/cmdb/host/host_handler.go`

Add three routes to `RegisterRoutes` and three handlers, mirroring `role_handler.go`:

```go
r.GET("/hosts/export", h.Export)
r.GET("/hosts/import-template", h.DownloadImportTemplate)
r.POST("/hosts/import", h.Import)
```

Handler details:
- `Export`: `common.SetAuditMetadata(c, "cmdb.host.export.title", common.BusinessExport)`; `var query HostListQuery`; `c.ShouldBindQuery(&query)`; on bind error `common.Fail(c, common.CodeParamInvalid, msgParamInvalid)`; `file, err := h.svc.Export(query, common.GetDataScope(c))`; on err `common.Fail(c, common.CodeError, "cmdbhost.export_failed")`; `impexp.WriteCSV(c, *file)`.
- `DownloadImportTemplate`: `file := h.svc.BuildImportTemplate()`; `impexp.WriteCSV(c, *file)`.
- `Import`: `common.SetAuditMetadata(c, "cmdb.host.import.title", common.BusinessImport)`; `fileHeader, err := c.FormFile("file")` → err `common.Fail(c, common.CodeParamInvalid, "import.file.required")`; `file, err := fileHeader.Open()` → err `common.Fail(c, common.CodeError, "import.file.read.error")`; `records, err := impexp.ReadCSV(file)` → err `common.Fail(c, common.CodeParamInvalid, "import.file.invalid_csv")`; `createdBy := strconv.FormatUint(common.GetUserID(c), 10)`; `result, err := h.svc.Import(records, common.GetDataScope(c), createdBy)` → err `common.Fail(c, common.CodeError, "cmdbhost.import_failed")`; `common.Success(c, result)`.

Note: `impexp`, `common`, `strconv` imports already available in the package (add `impexp` import to host_handler.go).

### 2. New file `backend/modules/business/cmdb/host/host_import_export.go` (package `host`)

Add `var maxHostExportRows = 10000` (var so tests can lower it).

`func (s *HostService) Export(query HostListQuery, dataScope *common.DataScopeReq) (*impexp.CSVFile, error)`:
- If `s.db == nil` return `nil, common.NewBadRequest("database.not_initialized")`.
- Build query mirroring `List` filters: keyword → `hostname LIKE ? OR ip LIKE ?` (use `common.EscapeLikePattern`), status, os, businessScopeID, deptID, scoped via `s.hostQuery(dataScope)`, `Order("id DESC")`, `Limit(maxHostExportRows)`, `Find(&hosts)`.
- Headers: `hostname, ip, sshPort, os, osVersion, cpuCores, memoryGb, diskGb, labels, businessScopeCode, deptId, owner, status, remark`.
- One row per host. `labels` serialized as `key=val` pairs joined by `"; "` (e.g. `env=prod; region=cn-east-1`); empty labels → empty string. Numeric fields via `strconv.FormatInt` / `strconv.FormatFloat(...,'f',-1,64)`.
- Filename: `"cmdb-host-export.csv"`.

`func (s *HostService) BuildImportTemplate() *impexp.CSVFile`:
- Same headers as export, plus a leading `#`-comment row and one example row (mirror `BuildRoleImportTemplate` style):
  - Row 1 (comment): `{"#说明：保留第一行表头；hostname/ip/os 必填；labels 多个用 \"; \" 分隔（key=val）；businessScopeCode 对应业务域编码；status 取值 pending/assigned/online/offline/maintenance，留空默认 pending。", "", "", "", "", "", "", "", "", "", "", "", "", ""}`
  - Row 2 (example): `{"web-01", "10.0.0.1", "22", "linux", "Ubuntu 22.04", "4", "8", "100", "env=prod; region=cn-east-1", "volcano-prod", "0", "ops", "pending", ""}`
- Filename: `"cmdb-host-import-template.csv"`.

`func (s *HostService) Import(records [][]string, dataScope *common.DataScopeReq, createdBy string) (*impexp.ImportResult, error)`:
Mirror `role.ImportRoles` structure:
- `result := &impexp.ImportResult{Applied: false, Errors: []impexp.ImportError{}}`.
- If `s.db == nil` return `nil, common.ErrDatabaseNotInitialized`.
- If `len(records) == 0`: `impexp.AppendImportError(result, 0, "file", "import.file.empty")`; return result.
- Build `headerIndex` map (trim header whitespace). Validate required headers `hostname`, `ip`, `os`; missing → `impexp.AppendImportError(result, 0, header, "import.header.missing")`. If `result.Failed > 0` return.
- Preload existing hosts: `s.db.Find(&hosts)` → `map[string]*Host` keyed by `IP` (only non-deleted; GORM soft-delete auto-filters).
- Preload scopes: `SELECT id, code, name FROM biz_business_scope WHERE deleted_at IS NULL` → `map[string]scopeRef{id, code, name}` keyed by `code`.
- Parse each data row (skip `impexp.IsCSVRecordEmpty`), rowNumber = index+1:
  - `hostname` required → error key `cmdbhost.hostname_required` on field `hostname`.
  - `ip` required → `cmdbhost.ip_required` on field `ip`.
  - `os` required → `cmdbhost.os_required` on field `os`.
  - `sshPort` = `impexp.ParseCSVInt`, default 22 if empty; parse error → `param.invalid` on field `sshPort`.
  - `cpuCores` = ParseCSVInt; `memoryGb`/`diskGb` = `strconv.ParseFloat` (empty → 0); parse error → `param.invalid`.
  - `labels`: split on `";"` and `","` (accept both), each `key=val`; trim; skip empty; build `[]LabelEntry`. A malformed entry (no `=`) → `param.invalid` on field `labels`.
  - `businessScopeCode` = trim; if non-empty: resolve via scope map; not found → `business.bizscope.notFound` on field `businessScopeCode`; found → capture id/code/name.
  - `deptId` = ParseCSVInt; `owner`, `remark` = trim.
  - `status` = trim; if empty → leave as "" (default applied later); else validate in `{pending, assigned, online, offline, maintenance}` → invalid → `cmdbhost.status.invalid` on field `status`.
  - Duplicate IP within the file: track `seenIP map[string]int`; repeat → `impexp.AppendImportError(result, rowNumber, "ip", fmt.Sprintf("import.duplicate.row.%d", firstRow))`.
- After parse loop, if `result.Failed > 0` return result (no apply).
- Apply in a single `s.db.Transaction`:
  - For each parsed row:
    - If existing host (by IP) exists → UPDATE: set `hostname`, `os`, `os_version`, `ssh_port`, `cpu_cores`, `memory_gb`, `disk_gb`, `label_values` (marshaled JSON), `business_scope_id/code/name`, `dept_id`, `owner`, `remark`, `updated_by=createdBy`, `updated_at=time.Now()`. Set `status` ONLY if the row provided a non-empty status; otherwise leave unchanged. Increment `Updated`.
    - Else → CREATE new `Host` mirroring `Create` semantics: default `SSHPort=22` if 0; `status` = provided status if non-empty, else `assigned` if scope resolved else `pending`; set `CreatedBy/UpdatedBy=createdBy`. Increment `Created`.
  - On any DB error return err (transaction rolls back).
- `result.Applied = true`; return result.

### 3. `backend/modules/business/cmdb/module.go`

- Add consts:
  - `cmdbHostPermExportKey = "business.cmdb.host.permission.export"`
  - `cmdbHostPermImportKey = "business.cmdb.host.permission.import"`
- In `hostMenuSeeds()`, append two F-type seeds (after the status seed):
  - `{Key: "operations-cmdb-host-export", ParentKey: cmdbHostListRoute, TitleKey: cmdbHostPermExportKey, Perms: "business:cmdb:host:export", Type: "F", Module: cmdbModuleKey, Sort: 7}`
  - `{Key: "operations-cmdb-host-import", ParentKey: cmdbHostListRoute, TitleKey: cmdbHostPermImportKey, Perms: "business:cmdb:host:import", Type: "F", Module: cmdbModuleKey, Sort: 8}`
- In `seedHostI18n`, add permission i18n (zh-CN + en-US, Group "permission"):
  - `cmdbHostPermExportKey`: zh "导出主机资产" / en "Export host assets"
  - `cmdbHostPermImportKey`: zh "导入主机" / en "Import hosts"
- Add error i18n (zh-CN + en-US, Group "error") for any new keys: `cmdbhost.hostname_required`, `cmdbhost.ip_required`, `cmdbhost.os_required`, `cmdbhost.status.invalid`, `cmdbhost.export_failed`, `cmdbhost.import_failed`. (Reuse existing `cmdbhost.ip_exists`, `cmdbhost.not_found`, `business.bizscope.notFound`, and `import.*` keys — they already exist.)

### 4. `backend/modules/business/cmdb/seeds_test.go`

Add assertions mirroring existing ones for the two new F-menu records:
- `assertCmdbRecordCount(t, db, "system_menu", "perms = 'business:cmdb:host:export' AND type = 'F'", 1)`
- `assertCmdbRecordCount(t, db, "system_menu", "perms = 'business:cmdb:host:import' AND type = 'F'", 1)`
- (optional) role permission bindings for the two new keys if the existing test asserts per-key `system_role_permission` rows.

### 5. Tests (recommended)

Add `backend/modules/business/cmdb/host/host_import_export_test.go` mirroring role's `role_service_test.go` import/export tests:
- export caps rows at `maxHostExportRows` (lower the var in test, restore after).
- import empty file returns error result.
- import creates new hosts + updates existing hosts by IP (assert created/updated counts).
- import duplicate IP in file appends error.
- import invalid status appends error.

Follow the existing test conventions in the host/role packages (in-memory DB setup identical to role_service_test.go).

## Frontend changes

### 1. `frontend/src/modules/business/cmdb/host/api.ts`

Add imports `downloadFile` from `'../../../api/file'` and `uploadImportFile` from `'../../../api/importExport'`. Add three functions:

```ts
export function exportHosts(params?: HostListQuery) {
  return downloadFile({
    url: '/business/cmdb/hosts/export',
    method: 'get',
    params,
    filename: 'cmdb-host-export.csv',
  });
}

export function downloadHostImportTemplate() {
  return downloadFile({
    url: '/business/cmdb/hosts/import-template',
    method: 'get',
    filename: 'cmdb-host-import-template.csv',
  });
}

export function importHosts(file: File) {
  return uploadImportFile('/business/cmdb/hosts/import', file);
}
```

### 2. `frontend/src/modules/business/cmdb/host/CmdbHostList.tsx`

- Add `IconDownload` to the `@arco-design/web-react/icon` import; add `ImportCsvButton` to the components import; add `import { showImportResult } from '../../../../api/importExport';`; extend the `./api` import with `exportHosts, downloadHostImportTemplate, importHosts`.
- Add `const canExport = hasPerm('business:cmdb:host:export');` and `const canImport = hasPerm('business:cmdb:host:import');`.
- Add handlers:
  - `const handleExport = async () => { await exportHosts(query); };`
  - `const handleDownloadTemplate = async () => { await downloadHostImportTemplate(); };`
  - `const handleImport = async (file: File) => { const result = await importHosts(file); showImportResult(result, t); if (result.applied) loadData(query); };`
- In the `TableBatchActionBar` `prefixActions`, the existing `ListHeaderActions` currently only has `primary`. Add a `utility` prop with the three buttons, mirroring `PostList.tsx` exactly:

```tsx
<ListHeaderActions
  utility={
    <>
      <Button icon={<IconDownload />} onClick={handleExport} disabled={!canExport}>
        {t('common.export')}
      </Button>
      <Button onClick={handleDownloadTemplate} disabled={!canImport}>
        {t('common.downloadTemplate')}
      </Button>
      <ImportCsvButton disabled={!canImport} onSelect={handleImport}>
        {t('common.import')}
      </ImportCsvButton>
    </>
  }
  primary={
    canCreate ? (
      <Button type="primary" icon={<IconPlus />} onClick={handleCreate}>
        {t('common.add')}
      </Button>
    ) : undefined
  }
/>
```

Note: `common.export`, `common.downloadTemplate`, `common.import` i18n keys already exist.

## Verification (run all)

- Backend: `cd backend && go build ./... && go test ./modules/business/cmdb/...`
- Frontend: `cd frontend && npm run type-check && npx eslint src/modules/business/cmdb/ && npx vite build`
- Overlay contract: from repo root `node scripts/business-overlay/check-business-overlay.mjs`

## Constraints

- Module path is `pantheon-base` (imports rewritten); import `pantheon-base/pkg/impexp`, `pantheon-base/pkg/common`, `pantheon-base/pkg/database`.
- Follow existing code style (Chinese doc comments matching neighbors; `common.Fail` / `common.FailWithError` / `common.Success`).
- Do NOT add `excelize`. Use the existing CSV impexp infra — CSV opens in Excel (BOM-prefixed), and it is the codebase-wide convention for every import/export feature.
