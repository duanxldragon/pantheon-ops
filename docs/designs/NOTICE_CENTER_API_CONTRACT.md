---
title: 通知中心接口契约
doc_type: Contract
layer: platform
status: Draft
related_designs:
  - docs/designs/NOTICE_CENTER_DESIGN.md
  - docs/designs/NOTICE_CENTER_DB_DESIGN.md
  - docs/designs/NOTICE_CENTER_TEST_PLAN.md
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-11-04
---

# 通知中心接口契约

English version: NOTICE_CENTER_API_CONTRACT.en.md（planned/unavailable：`./NOTICE_CENTER_API_CONTRACT.en.md`；无当前交付承诺）

> 未实现声明：本文为设计稿，仓库内尚无对应实现；在实现落地前状态保持 Draft。
> Not implemented: this is a design draft with no corresponding implementation in the repository; it stays Draft until implemented.

关联设计：
- `NOTICE_CENTER_DESIGN.md`
- `NOTICE_CENTER_DB_DESIGN.md`
- `NOTICE_CENTER_TEST_PLAN.md`

---

本文是通知中心的 REST 面契约：方法、路径、DTO、错误码、分页、权限 key、审计头行为与 i18n 命名空间。实现前本契约不生效；一旦落地，本文即成为评审与门禁的引用对象。

## 1. 范围与挂载点

| 项 | 决定 | 依据 |
|----|------|------|
| 前缀 | `/api/v1/notice` | 所有模块都在 `api := r.Group("/api/v1")` 下注册（`backend/cmd/server/main.go:159`） |
| 归属模块 | `backend/modules/platform/notice/`，由 `RegisterPlatformRoutes` 挂载 | 平台面先例：`backend/modules/platform/routes.go:16-42` |
| 中间件链 | `TokenAuth → PlatformScopeTenantContext → Casbin`，destructive 追加 `SecureActionMiddleware` | 同上 `:34-37`；系统形态若走 `contracts.SystemProtectedGroup` 则为 `TokenAuth → PlatformScopeTenantContext → Casbin →（可选 Capability/DataScope/RefreshSync）`（`backend/pkg/contracts/route_groups.go:137-150`，顺序是安全属性） |
| 摄入面 | **不出货 HTTP 端点**，源域走进程内 service（设计 §17 OD-8 的建议） | 若 OD-8 裁定为 HTTP，本节追加一条声明式内部路由并同步 §6 的权限 key |
| 限流 | 继承 `/api/v1` 的全局限流（按 IP，默认 6000/60s，`PANTHEON_API_RATE_LIMIT_*` 可配） | `backend/cmd/server/main.go:160-167`；组级按用户限流预设存在但底座未注册（`backend/internal/middleware/rate_limit_presets.go:33-36` 注释），本文不假设它生效 |

## 2. 通用约定

### 2.1 响应信封

```json
{ "code": 200, "data": {}, "message": "success" }
```

`Response{Code, Data, Message}` 定义在 `backend/pkg/common/http/response.go:14-19`，经 `pkg/common/re_export.go` 全站复用。

### 2.2 HTTP 状态 vs 业务码（不可混淆的两条路）

| 写法 | 实际 HTTP | body `code` | 使用者 |
|------|-----------|-------------|--------|
| `common.Fail(c, code, key)` / `FailWithError` / `FailWithData` | **200** | 业务码 | Service 层业务失败（`response.go:47-59` 与 `:98-111`，最终都经 `FailWithData` 固定写 `http.StatusOK`） |
| `common.FailWithCode(c, code, key)` | 真实状态码 | 同码 | 中间件级拒绝（例：`secure_action_middleware.go:20`、`:27`） |

本契约的**业务错误**一律是 HTTP 200 + 业务码；只有认证/授权/step-up 类拒绝才是真实 4xx。Casbin 拒绝走 `common.Fail(c, common.CodeForbidden, "permission.denied")`（`backend/internal/middleware/casbin_middleware.go:48`），因此也是 HTTP 200 + `code=403`，前端必须按 `code` 而非 HTTP status 判定。

### 2.3 错误码到业务码的映射

`common.CodeForError` 是唯一映射源（`backend/pkg/common/error.go:87-104`）：`ErrBadRequest → 400`、`ErrConflict → 400`、`ErrNotFound → 404`、`ErrForbidden → 403`、`ErrUnauthorized → 401`、其余 `→ 500`。

**注意（可核对的行为，不是偏好）**：本仓库没有 409 通道——`common.NewConflict(...)` 落库冲突也会以 `code=400` 返回。通知中心不得在文档或实现里承诺 409。

### 2.4 message 是 i18n key

- 业务失败时 `message` 必须是 i18n key，形状受 `backend/pkg/common/http/response.go:29` 的正则约束 `^[a-z0-9_]+(?:\.[a-z0-9_]+)+$`：**小写字母、数字、下划线，点号分层**。因此 `notice.not_found` 合法，`notice.not-found` 与 `Notice.NotFound` 非法（非法 key 会被 `resolveErrorMessageKey` 替换为 `request.failed`，即错误信息静默丢失，见 `response.go:81-96`）。
- key 必须进 i18n 资源（§8），否则前端显示原始 key。

### 2.5 分页

- 查询参数：`page`（默认 1）、`pageSize`（默认 10，**上限 100**）。
- 由 `common.NormalizePagination(page, pageSize, 100)` 归一（`backend/pkg/common/handler.go:39-50`），与 `audit_service.go:47`、`dict_service.go:24`、`permission_service.go:937`、`role_helper.go:534`、`post_service.go:663` 同形。
- 超限不报错而是钳到 100：这是该 helper 的既有语义，本契约不新立规则；同时不得声称「超限被拒绝」（与 `TENANT_CONTRACT_V1` §2.5 规则 4 的表述纪律一致——合同只承诺「有界」）。
- 列表响应形状：`{ items, total, page, pageSize }`，对齐 `OperationLogPageResp`（`backend/modules/system/audit/audit_dto.go:24-33`）。

### 2.6 排序

`sort` 走白名单映射 `common.NormalizeSort(input, allowed, fallback)`（`backend/pkg/common/handler.go:57-62`）；未登记字段回落默认 `created_at desc`，不进入 SQL 拼接。

### 2.7 追踪头

`X-Request-ID` / `X-Trace-ID` 常量定义在 `backend/pkg/common/http/request_context.go:10-11`，审计与会话排障依赖它们；通知中心不自定义新头（`X-Operation-Token` 除外，见 §7）。

### 2.8 租户

- `tenant_id` 只从上下文取，绝不从请求体取（`TENANT_CONTRACT_V1` §2.3.1 第 2 条）。
- 收件人列表天然只含当前主体的行；跨收件人读只存在于 §5 声明的平台面。
- 寻址他人消息：返回空集/404，**不区分**「不存在」与「属于他人」，与 `TENANT_CONTRACT_V1` §5 最后一条同源。

## 3. 端点总表

| # | 方法 | 路径 | 权限 key | 幂等 | 审计 | step-up |
|---|------|------|----------|------|------|---------|
| 1 | GET | `/api/v1/notice/list` | `platform:notice:view` | 是（读） | 否（GET） | 否 |
| 2 | GET | `/api/v1/notice/unread-count` | `platform:notice:view` | 是（读） | 否 | 否 |
| 3 | GET | `/api/v1/notice/:id` | `platform:notice:view` | 是（读） | 否 | 否 |
| 4 | PUT | `/api/v1/notice/:id/read` | `platform:notice:update` | 是（设计 §9.4） | 是 | 否 |
| 5 | PUT | `/api/v1/notice/read-all` | `platform:notice:update` | 是 | 是 | 否 |
| 6 | PUT | `/api/v1/notice/:id/archive` | `platform:notice:update` | 是 | 是 | 否 |
| 7 | PUT | `/api/v1/notice/:id/unarchive` | `platform:notice:update` | 是 | 是 | 否 |
| 8 | DELETE | `/api/v1/notice/:id` | `platform:notice:update` | 是 | 是 | 否 |
| 9 | POST | `/api/v1/notice/batch-update` | `platform:notice:update` | 是 | 是 | 否 |
| 10 | POST | `/api/v1/notice/admin/publish` | `platform:notice:publish` | 依赖 `dedupKey` | 是 | 否 |
| 11 | GET | `/api/v1/notice/admin/list` | `platform:notice:manage` | 是（读） | 否 | 否 |
| 12 | POST | `/api/v1/notice/admin/:id/withdraw` | `platform:notice:manage` | 状态机守卫 | 是 | **是** |
| 13 | POST | `/api/v1/notice/admin/replay-dead` | `platform:notice:manage` | 否（每次产生新尝试） | 是 | **是** |
| 14 | POST | `/api/v1/notice/admin/cleanup` | `platform:notice:manage` | 否（清理量随时间变化） | 是 | **是** |

路由形状说明：`/notice/:id` 与 `/notice/list`、`/notice/unread-count` 等静态兄弟共存，是本仓库已经在跑的既有形状——审计组同时注册了 `/operation-log/list`、`/operation-log/:id`、`/operation-log/export`、`/operation-log/cleanup`（`backend/modules/system/system_modules.go:531-536`）。新形状必须在同一 patch 内进 §6.2 的 workbench 探针表，理由与 `TENANT_CONTRACT_V1` §2.5 规则 1 对路由形状的记录相同。

## 4. DTO

字段命名 lowerCamelCase（对齐 `backend/modules/system/audit/audit_dto.go` 的 json tag 风格）。

### 4.1 `NoticeItem`（列表行）

```json
{
  "id": 1024,
  "category": "alert",
  "severity": "warning",
  "status": "unread",
  "sourceDomain": "system.auth",
  "title": "...",
  "summary": "...",
  "jumpType": "route",
  "jumpTarget": "/system/security/security-event",
  "jumpPerm": "system:security-event:list",
  "createdAt": "2026-11-04T09:00:00+08:00",
  "readAt": null,
  "archivedAt": null,
  "expiresAt": null
}
```

约定：`id` 序列化为 JSON 数字（后端 `uint64`、前端 `number`），沿用既有形状——`audit_dto.go:5` 与 `user_dto.go:4` 的 `ID uint64 \`json:"id"\``，前端对应 `frontend/src/modules/system/audit/api.ts:22` 的 `id: number`；本契约不得新立「id 一律转字符串」的规则（仓库仅 `generator_dto.go:4` 一处字符串 id，且那是名称型 id，不是雪花/自增 id）。`status` 是设计 §4.5/§5.1 的派生值（由 `readAt`/`archivedAt` 推出，不落列），`content`/`payload` **不**出现在列表响应中。

### 4.2 `NoticeDetail`

`NoticeItem` 加：`content`、`contentType`、`payload`、`sourceType`、`sourceId`、`createdBy`、`deliveredAt`；站外渠道启用时加 `deliveries: [{channel, status, attempts, lastError, sentAt}]`。

### 4.3 列表查询参数

`GET /notice/list`：`page` `pageSize` `status`(`unread|read|archived|all`，默认 `unread`) `category` `severity` `sourceDomain` `keyword` `sort`。
- `keyword` 只匹配 `title/summary`，不做全文检索（设计 §2 非目标）。
- `archived` 与 `all` 之外不再提供自定义组合状态。

### 4.4 写请求

| 端点 | 请求体 | 成功响应 `data` |
|------|--------|-----------------|
| `PUT /:id/read` | 无 | `{ "id": 1024, "status": "read", "changed": true }` |
| `PUT /read-all` | `{ "category": "task" }`（可空=全部） | `{ "updated": 12 }` |
| `PUT /:id/archive` | 无 | `{ "id": 1024, "status": "archived" }` |
| `PUT /:id/unarchive` | 无 | `{ "id": 1024, "status": "read" }` |
| `DELETE /:id` | 无 | `{ "deleted": 1 }` |
| `POST /batch-update` | `{ "ids": [1024, 1025], "action": "read\|archive\|delete" }`，`ids` 上限 500 | `{ "updated": 2, "skipped": 0 }` |
| `POST /admin/publish` | `{ "category","severity","title","summary","content","contentType","jumpType","jumpTarget","jumpPerm","dedupKey","expiresAt","recipients":[userId...] 或 "broadcast": true }` | `{ "id": 1024, "deduplicated": false, "targets": 42 }` |
| `POST /admin/:id/withdraw` | 无 | `{ "id": 1024, "status": "withdrawn", "targetsHidden": 42 }` |
| `POST /admin/replay-dead` | `{ "channel": "email", "limit": 50 }` | `{ "replayed": 3 }` |
| `POST /admin/cleanup` | 无 | `{ "task": "notice.retention", "outcome": "succeeded", "deleted": 812 }` |

`batch-update` 的 `ids` 上限 500 与拒绝方式（`param.invalid`）直接沿用 `backend/modules/system/audit/audit_service.go:357` 的既有守卫形状。

`admin/cleanup` 的响应字段形状对齐 `maintenance` 的结果语义（`Outcome` ∈ `succeeded|failed|skipped|overlap|unknown`，`backend/pkg/maintenance/maintenance.go:38-58`）：`skipped`/`overlap` 是**正常返回**而非错误，客户端不得把它渲染成失败。

### 4.5 未读计数

```json
GET /api/v1/notice/unread-count
{ "code":200, "data": { "total": 7, "critical": 1, "byCategory": { "task": 3, "alert": 4 } }, "message": "success" }
```

设计 §9.2；`byCategory` 的键集合就是 §4.3 的枚举，不返回 0 值键。

## 5. 平台面声明

`/notice/admin/*` 是 `TENANT_CONTRACT_V1` §3.3/§3.4 意义上的**显式声明平台作用域**路由：

1. 需要平台级能力（全局角色或无 domain 的 Casbin policy），租户内角色不构成平台能力（§3.4 第 1 条）。
2. 权限 key 保持 `platform:` 前缀；若将来挪进 `system:` 前缀，则 `pkg/tenant.IsPlatformOnlyPermissionKey` 与前端 `core/layout/useLayoutMenu.tsx` 的判定必须在同一 patch 内补齐（§2.5 规则 2 的义务）。
3. 跨租户读只允许出现在 `admin/list`，且只返回聚合与本体事实，不返回他人收件人的 `read_at`（设计 §9.5）。

## 6. 权限与门禁义务

### 6.1 权限 key

| key | 授权对象 | 说明 |
|-----|----------|------|
| `platform:notice:view` | 所有登录主体角色 | 页面准入 + 自己列表；默认随基础角色 seed |
| `platform:notice:update` | 所有登录主体角色 | 只作用于自己的视图行 |
| `platform:notice:publish` | 平台运营角色 | 发布与源域代投 |
| `platform:notice:manage` | 平台管理员 | 撤回/重放/清理，三条 destructive 面 |

前缀与三段式形状受 `scripts/harness/check-permission-contract.mjs` 的 `PERMISSION_KEY_PATTERN = /(?:system|business|platform):[a-z0-9:-]+/g`（同文件 :30）识别；既有同层 key 见 `backend/modules/system/seed.go:210` 的 `platform:dashboard:view`。

### 6.2 权限↔路由绑定

新增的每条**受权限保护的**路由必须登记进 `requiredAPIRoutesByPermission`（`backend/modules/system/iam/permission/permission_workbench.go:622-652`），并由 `permission_workbench_routes_test.go` 的 `TestRequiredAPIRoutesExistOnEngine` 对照真实引擎校验：该表的注释（同文件 :607-613）写明「每条都必须匹配实注册路由，漂移由该测试挡住，补齐流程绝不能指向不存在的路由」。只读导出面通过 `ReadOnlyExportRoutes` 追加（同文件 :648-650）。

### 6.3 菜单与页面准入

菜单 `menuSeed` 的 `PagePerm` 必须等于 `platform:notice:view`，`Component` 必须能在 `frontend/src/core/router/componentRegistry.ts` 解析（该文件把 `'dashboard'` 映射到 `modules/platform/Dashboard`，是新页面的模板）。菜单与权限解耦规则见 `docs/designs/PERMISSION_MENU_DECOUPLING.md`。

## 7. 审计头与审计行为

| 行为 | 约定 | 依据 |
|------|------|------|
| 非 GET 自动留痕 | 端点 4-14 全部自动进 `system_log_oper`，无需额外接线 | `backend/internal/middleware/operation_log_middleware.go:224-231`（GET 直接放行不记录） |
| `X-Operation-Token` | 端点 12/13/14 必需；缺失→403 `auth.operation.verification_required`，过期→403 `auth.operation.verification_expired`，用户/会话/scope 不匹配→403 `auth.operation.verification_mismatch` | `backend/internal/middleware/secure_action_middleware.go:16-53` |
| step-up 的 HTTP 语义 | 这三条是**真实 403**（`FailWithCode`），与 §2.2 的「业务失败 HTTP 200」不同 | 同文件 `:20/:27/:32` |
| 审计标题 | 12/13/14 与 `read-all` 显式调用 `common.SetAuditMetadata(c, titleKey, businessType)` | `backend/pkg/common/audit.go:30-34`；用法先例 `backend/modules/auth/login/login_handler.go:451` |
| 业务类型 | `cleanup` 用 `BusinessClean = 8`（先例 `backend/modules/system/audit/audit_handler.go:90`）；`withdraw` 用 `BusinessDelete = 3`；`publish` 用 `BusinessInsert = 1`（先例 `backend/modules/lowcode/dynamicmodule/dynamic_module_handler.go:26`）；授权类才用 `BusinessGrant = 4`（先例 `backend/modules/system/iam/tenant/tenant_platform_handler.go:237`） | `backend/pkg/common/audit.go:18-28` |
| 参数脱敏 | `oper_param` 落库前需脱敏，`payload` 中禁止携带凭证 | `docs/designs/DATABASE.md` §3.2（`oper_param` 脱敏要求） |
| 读操作 | 端点 1/2/3/11 是 GET，不落审计；「谁读过」只有 `read_at` 事实字段 | 设计 §12 |

审计标题 key 建议：`notice.audit.publish.title`、`notice.audit.withdraw.title`、`notice.audit.replay.title`、`notice.audit.cleanup.title`（小写下划线点分，受 §2.4 正则约束）。

## 8. i18n key 命名空间

`system_i18n` 的 `(module, group, key)` 三元组语义见 `docs/designs/DATABASE.md` §3.1；canonical 条目形态见 `backend/modules/system/i18n/i18n_seed.go:109+`，语言集合为 `zh-CN/en-US/ja-JP/ko-KR/fr-FR`（`backend/modules/system/i18n/i18n_helper.go:815`）。

| 命名空间 | module | group | 例子 |
|----------|--------|-------|------|
| 菜单标题 | `platform` | `menu` | `platform.menu.notice` |
| 权限说明 | `platform` | `permission` | `platform.permission.notice.view` |
| 页面文案 | `platform` | `messages` | `notice.page.title`、`notice.filter.unread` |
| 错误/结果 key | `platform` | `messages` | `notice.not_found`、`notice.state.invalid`、`notice.batch.too_many` |
| 审计标题 | `platform` | `messages` | `notice.audit.cleanup.title` |
| 设置说明 | `system` | `messages` | `system.setting.remark.notice.read_target_retention_days`（对齐 `backend/modules/system/config/setting/setting_seed.go:65-68` 的 remark key 形态） |

迁移义务：壳层现有 `app.notice.*`（`frontend/src/i18n/resources/zh-CN.ts:14+`，被 `frontend/src/core/layout/AppLayout.tsx` 消费）在实现时并入 `notice.*`，**不留两套 key**；前端 fallback 资源五语同步（`frontend/src/i18n/resources/*.ts`）。

## 9. 错误码表

`code` 是业务码（HTTP 恒为 200，§2.2）。

| key | code | 触发 | 前端处置 |
|-----|------|------|----------|
| `param.invalid` | 400 | 绑定失败、`ids>500`、`category` 非法枚举 | 表单/提示，不重试 |
| `permission.denied` | 403 | Casbin 拒绝 | 进入 forbidden 态（设计 §11） |
| `auth.operation.verification_required` | 403（真实 HTTP） | 缺 `X-Operation-Token` | 触发 step-up 流程 |
| `auth.operation.verification_expired` | 403（真实 HTTP） | 令牌过期 | 重新验证后重试一次 |
| `auth.operation.verification_mismatch` | 403（真实 HTTP） | 用户/会话/scope 不匹配 | 重新验证，不自动重试 |
| `notice.not_found` | 404 | id 不存在或不属于当前收件人（**不可区分**，§2.8） | 单行移除 + 列表刷新 |
| `notice.state.invalid` | 400 | 非法状态迁移（设计 §5.3） | 提示并刷新该行状态 |
| `notice.publish.invalid_jump` | 400 | `jumpType=route` 但目标不在路由表 / `external-url` 不在白名单 | 表单错误 |
| `notice.publish.recipients_required` | 400 | 既无 `recipients` 又非 `broadcast` | 表单错误 |
| `request.failed` | 500 | 未分类内部错误（`resolveErrorMessageKey` 的兜底值） | 通用错误态 |

不引入 `409`（§2.3）；不新增全局错误码枚举文件，key 直接进 i18n——与 `TENANT_CONTRACT_V1` §5 的做法一致（能用既有码就不新立）。

## 10. 版本与兼容

- 路径不带版本号，版本在 `/api/v1` 前缀层（既有约定）。
- 破坏性变更（字段删除、语义改变、状态枚举变化）属 `TENANT_CONTRACT_V1` §8 意义上的 contract change：先改本文并给下游（`pantheon-ops`）留 foundation release 同步窗口。
- 兼容承诺：新增可选字段允许；`status`/`category`/`severity`/`jumpType` 四个枚举**只增不改**，前端遇到未知值必须原样展示而非崩溃（与 `TENANT_CONTRACT_V1` §2.5 规则 5「未知值如实呈现」同方向）。

## 11. 开放决策（接口层）

| 编号 | 议题 | 选项 | 建议 | 决策人 |
|------|------|------|------|--------|
| AI-1 | 摄入面形态（设计 OD-8） | A 进程内 service；B 内部 HTTP 端点 | A | 维护者 |
| AI-2 | `broadcast` 是否在第一阶段开放 | A 开放但分批 fan-out；B 只允许显式 `recipients` | B（V1），A 需要 §6.3 的体量论证先落地 | 维护者 |
| AI-3 | 是否提供 `/notice/export` | A 提供并复用 `ReadOnlyExportRoutes`；B 不提供 | B：导出会引入跨租户读面与新的权限组合，V1 无需求 | 维护者 |
| AI-4 | 未读计数是否合并进列表响应 | A 独立端点；B 列表响应带 `unread` 字段 | A：徽章轮询频率远高于列表访问，合并会让每次轮询都拉列表数据 | 维护者 |

## 12. 完成定义

本契约算「可实现」当且仅当：

- §3 每条路径都有中间件链与权限 key 的对应实现，并登记进 §6.2 的 workbench 表且有探针测试
- §9 的每个 key 都有五语 i18n 条目
- §7 的 step-up 与审计标题按既有中间件行为生效（不新造审计通道）
- §2.2/§2.3 的 HTTP 语义在前端拦截器里被正确消费（业务失败读 `code`，不读 HTTP status）
