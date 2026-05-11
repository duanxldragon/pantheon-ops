# 系统域导入导出 Smoke 指南

更新时间：2026-04-21

类型：Acceptance
归属层：system/config
状态：Active

本文用于验证 `system/iam`、`system/org`、`system/config`、`system/auth`、`system/audit` 已开放的导入导出能力是否真实可用。

除本文的手工联调步骤外，仓库也已提供一套可复用的 API 自动 smoke：

```powershell
cd frontend
cmd /c npm run test:smoke:impexp
```

默认会请求 `http://127.0.0.1:8080/api/v1`，并使用 `admin / 123456` 登录；如需覆盖，可设置：

- `PANTHEON_API_BASE_URL`
- `PANTHEON_SMOKE_ADMIN_USERNAME`
- `PANTHEON_SMOKE_ADMIN_PASSWORD`

## 1. 范围与边界

- `system/iam`
  - 用户：导入 / 导出
  - 权限策略：导入 / 导出
  - 角色：仅导出、批量启用 / 禁用，不支持导入
- `system/org`
  - 部门：导入 / 导出
  - 岗位：导入 / 导出
- `system/config`
  - 字典类型：导入 / 导出
  - 字典项：导入 / 导出
- `system/auth`
  - 登录日志：仅导出
- `system/audit`
  - 操作日志：仅导出

## 2. 前置条件

- 服务已启动，默认地址：`http://127.0.0.1:8080`
- 已具备管理员账号，例如 `admin / 123456`
- 使用 Windows PowerShell 时，建议显式使用 `curl.exe`

## 3. 登录获取 Token

```powershell
$base = "http://127.0.0.1:8080/api/v1"
$loginBody = @{
  username = "admin"
  password = "123456"
} | ConvertTo-Json

$loginResp = Invoke-RestMethod `
  -Uri "$base/auth/login" `
  -Method Post `
  -ContentType "application/json" `
  -Body $loginBody

$token = $loginResp.data.accessToken
```

如果你的响应结构不是 `data.accessToken`，也可以改成：

```powershell
$token = $loginResp.accessToken
```

## 4. 通用验证步骤

每个模块都建议走以下闭环：

1. 下载后端模板
2. 用本文提供的示例 CSV 或模板复制一份测试文件
3. 执行导入
4. 调用列表接口确认数据落库
5. 执行导出
6. 打开导出文件确认字段与内容正确

说明：

- 模板中的 `#` 注释行会被导入器自动忽略
- 导入接口返回 `applied / created / updated / failed / errors`
- 导出接口统一走 `POST /export`

## 5. 示例 CSV 文件位置

可直接使用以下样例文件：

- `tests/fixtures/system-import-export/user-import.csv`
- `tests/fixtures/system-import-export/dept-import.csv`
- `tests/fixtures/system-import-export/post-import.csv`
- `tests/fixtures/system-import-export/permission-import.csv`
- `tests/fixtures/system-import-export/dict-type-import.csv`
- `tests/fixtures/system-import-export/dict-item-import.csv`

## 6. 用户导入导出

### 下载模板

```powershell
curl.exe -L `
  -H "Authorization: Bearer $token" `
  "$base/system/user/import-template" `
  -o ".tmp/user-import-template.csv"
```

### 导入

```powershell
curl.exe -L `
  -H "Authorization: Bearer $token" `
  -F "file=@tests/fixtures/system-import-export/user-import.csv" `
  "$base/system/user/import"
```

### 列表校验

```powershell
curl.exe -L `
  -H "Authorization: Bearer $token" `
  "$base/system/user/list?username=sample_user&page=1&pageSize=10"
```

### 导出

```powershell
curl.exe -L `
  -H "Authorization: Bearer $token" `
  -H "Content-Type: application/json" `
  -d "{\"username\":\"sample_user\"}" `
  "$base/system/user/export" `
  -o ".tmp/user-export.csv"
```

## 7. 部门导入导出

注意：`parentDeptPath` 必须使用系统内已有完整路径，默认根节点通常是 `Pantheon Base`。

```powershell
curl.exe -L -H "Authorization: Bearer $token" "$base/system/dept/import-template" -o ".tmp/dept-import-template.csv"
curl.exe -L -H "Authorization: Bearer $token" -F "file=@tests/fixtures/system-import-export/dept-import.csv" "$base/system/dept/import"
curl.exe -L -H "Authorization: Bearer $token" "$base/system/dept/tree?deptName=%E7%A0%94%E5%8F%91"
curl.exe -L -H "Authorization: Bearer $token" -H "Content-Type: application/json" -d "{\"deptName\":\"研发\"}" "$base/system/dept/export" -o ".tmp/dept-export.csv"
```

## 8. 岗位导入导出

```powershell
curl.exe -L -H "Authorization: Bearer $token" "$base/system/post/import-template" -o ".tmp/post-import-template.csv"
curl.exe -L -H "Authorization: Bearer $token" -F "file=@tests/fixtures/system-import-export/post-import.csv" "$base/system/post/import"
curl.exe -L -H "Authorization: Bearer $token" "$base/system/post/list?postCode=developer&page=1&pageSize=10"
curl.exe -L -H "Authorization: Bearer $token" -H "Content-Type: application/json" -d "{\"postCode\":\"developer\"}" "$base/system/post/export" -o ".tmp/post-export.csv"
```

岗位 CSV 表头为 `deptPath,postCode,postName,sort,status,remark`；`deptPath` 必须指向已存在的普通部门，不能直接使用组织根节点。

## 9. 权限策略导入导出

注意：这里只验证 Casbin 路由策略，不验证菜单/按钮授权。

```powershell
curl.exe -L -H "Authorization: Bearer $token" "$base/system/permission/import-template" -o ".tmp/permission-import-template.csv"
curl.exe -L -H "Authorization: Bearer $token" -F "file=@tests/fixtures/system-import-export/permission-import.csv" "$base/system/permission/import"
curl.exe -L -H "Authorization: Bearer $token" "$base/system/permission/list?roleKey=admin&page=1&pageSize=10"
curl.exe -L -H "Authorization: Bearer $token" -H "Content-Type: application/json" -d "{\"roleKey\":\"admin\"}" "$base/system/permission/export" -o ".tmp/permission-export.csv"
```

## 10. 字典类型 / 字典项导入导出

### 字典类型

```powershell
curl.exe -L -H "Authorization: Bearer $token" "$base/system/dict/type/import-template" -o ".tmp/dict-type-import-template.csv"
curl.exe -L -H "Authorization: Bearer $token" -F "file=@tests/fixtures/system-import-export/dict-type-import.csv" "$base/system/dict/type/import"
curl.exe -L -H "Authorization: Bearer $token" "$base/system/dict/type/list?dictCode=biz_status"
curl.exe -L -H "Authorization: Bearer $token" -H "Content-Type: application/json" -d "{\"dictCode\":\"biz_status\"}" "$base/system/dict/type/export" -o ".tmp/dict-type-export.csv"
```

### 字典项

```powershell
curl.exe -L -H "Authorization: Bearer $token" "$base/system/dict/item/import-template" -o ".tmp/dict-item-import-template.csv"
curl.exe -L -H "Authorization: Bearer $token" -F "file=@tests/fixtures/system-import-export/dict-item-import.csv" "$base/system/dict/item/import"
curl.exe -L -H "Authorization: Bearer $token" "$base/system/dict/item/list?dictCode=biz_status"
curl.exe -L -H "Authorization: Bearer $token" -H "Content-Type: application/json" -d "{\"dictCode\":\"biz_status\"}" "$base/system/dict/item/export" -o ".tmp/dict-item-export.csv"
```

## 11. 角色导出与批量状态

角色仍不支持导入。

### 导出角色基础信息

```powershell
curl.exe -L `
  -H "Authorization: Bearer $token" `
  -H "Content-Type: application/json" `
  -d "{\"roleKey\":\"admin\"}" `
  "$base/system/role/export" `
  -o ".tmp/role-export.csv"
```

### 批量禁用 / 启用

```powershell
curl.exe -L `
  -H "Authorization: Bearer $token" `
  -H "Content-Type: application/json" `
  -d "{\"roleIds\":[2,3],\"status\":2}" `
  "$base/system/role/batch-status"
```

## 12. 登录日志 / 操作日志导出

```powershell
curl.exe -L -H "Authorization: Bearer $token" -H "Content-Type: application/json" -d "{\"username\":\"admin\"}" "$base/system/login-log/export" -o ".tmp/login-log-export.csv"
curl.exe -L -H "Authorization: Bearer $token" -H "Content-Type: application/json" -d "{\"title\":\"导出\"}" "$base/system/operation-log/export" -o ".tmp/operation-log-export.csv"
```

## 13. 成功判定标准

- 导入返回：
  - `applied=true`
  - `failed=0`
  - `created` 或 `updated` 大于 0
- 列表接口能查到刚导入的数据
- 导出得到非空 CSV 文件
- CSV 表头与接口定义一致
- 模板说明行不会被导入成真实数据

## 14. 当前确认结果

本仓库已通过以下自动化验证：

- 用户导入/导出
- 部门导入/导出
- 岗位导入/导出
- 权限策略导入/导出
- 字典类型 / 字典项导入/导出
- 角色导出 / 批量状态
- 登录日志导出
- 操作日志导出

如需进一步做接口级自动 smoke，可在 `tests/smoke/` 下继续补 Playwright / API 混合用例。
