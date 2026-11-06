---
title: 报表中心接口契约
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
updated_at: 2026-11-04
---

# 报表中心接口契约

> 未实现声明：本文为设计稿，仓库内尚无对应实现；在实现落地前状态保持 Draft。
> Not implemented: this is a design draft with no corresponding implementation in the repository; it stays Draft until implemented.

主设计：[REPORT_CENTER_DESIGN.md](./REPORT_CENTER_DESIGN.md)
数据模型：[REPORT_CENTER_DB_DESIGN.md](./REPORT_CENTER_DB_DESIGN.md)
测试计划：[REPORT_CENTER_TEST_PLAN.md](./REPORT_CENTER_TEST_PLAN.md)

---

## 1. 挂载点与路由链

- 前缀：`/api/v1`（`backend/cmd/server/main.go:159` 的 `registerAPIRoutes`）。
- 本契约按主设计 §16 OD-2 的 **A 案**书写：`/api/v1/platform/report/*`。若裁决改成 B 案
  （`/system/report/*`），只有前缀与菜单归属变化，DTO、错误键与权限键不变。
- 路由链必须与系统域控制台同形（`backend/pkg/contracts/route_groups.go:137`）：
  `TokenAuth -> PlatformScopeTenantContext -> Casbin`，可选再叠 `PlatformCapability` / `SecureAction`。
  报表**列表/详情/模板管理**组不挂 `DataScopeMiddleware`（报表定义不是部门归属数据，
  见主设计 §9.2 末尾）；**渲染与导出**组必须在 handler 内消费
  `common.GetDataScope(c)`，因此需要挂该中间件的组只有 `run` / `export` / `artifact download`
  （`route_groups.go:70-84` 明确"省略某个 option 本身就是权限决定"）。

## 2. 统一信封与错误语义

| 事实 | 来源 |
| :--- | :--- |
| 信封 `{code, data, message}`，成功 `message = "success"` | `backend/pkg/common/http/response.go:15-38` |
| `Fail/FailWithData` 写 **HTTP 200**，业务码放 `code` | 同上 `:47-60` |
| `FailWithCode` 写**真实 HTTP 状态**（403 分支用这条） | 同上 `:113-118` |
| `message` 必须是 i18n key，key 形状 `^[a-z0-9_]+(?:\.[a-z0-9_]+)+$` | 同上 `:28` |

由此得到两条实现约束：

1. 本契约所有错误键**不得含连字符**（`report.code.conflict` 合法，`report-code-conflict` 会被当成自然语言
   直接透传给前端），并与 [ERROR_CODE_AND_I18N.md](./ERROR_CODE_AND_I18N.md) §3.1 一致：后端不返回自然语言。
2. 权限拒绝走 Casbin 中间件的既有键 `permission.denied`（`backend/internal/middleware/casbin_middleware.go`
   的 `failPermissionCheck`，业务码 403）；二次验证缺失走
   `auth.operation.verification_required` / `auth.operation.verification_expired`
   （`backend/internal/middleware/secure_action_middleware.go`）。报表侧**不新增**同义键。

业务码取值集合沿用 `backend/pkg/common/http/response.go:21-26`：`200 / 400 / 401 / 403 / 404 / 500`。

## 3. 路由清单

| # | 方法 | 路径 | 权限键 | 组链附加项 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | GET | `/platform/report/dataset/list` | `platform:report:list` | — | 只读列出已注册口径（OD-3 A 案：无写接口） |
| 2 | GET | `/platform/report/template/list` | `platform:report:list` | — | 分页；按 `visibility`/`category`/`sourceDomain` 过滤 |
| 3 | GET | `/platform/report/template/:id` | `platform:report:view` | — | 含当前发布版本的呈现规格 |
| 4 | POST | `/platform/report/template` | `platform:report:create` | — | 建模板 + 首版 draft |
| 5 | PUT | `/platform/report/template/:id` | `platform:report:update` | — | 只改 draft 版本，不改已发布版本 |
| 6 | DELETE | `/platform/report/template/:id` | `platform:report:delete` | **SecureAction** | 软删 + code 改名释放（DB 设计 §5） |
| 7 | GET | `/platform/report/template/:id/version/list` | `platform:report:view` | — | 版本历史（分页） |
| 8 | POST | `/platform/report/template/:id/publish` | `platform:report:update` | **SecureAction** | 发布 draft 为不可变版本，前移 `current_version_id` |
| 9 | POST | `/platform/report/template/:id/rollback` | `platform:report:update` | **SecureAction** | 把指针指回历史已发布版本 |
| 10 | POST | `/platform/report/run` | `platform:report:run` | DataScope | 同步渲染预览；行数上限见 §5 |
| 11 | POST | `/platform/report/export` | `platform:report:export` | DataScope（`sensitive` 时再叠 SecureAction） | 导出：小报表直接回 CSV，大报表返回 `runId` 异步产物 |
| 12 | GET | `/platform/report/run/list` | `platform:report:view` | — | 历史（分页） |
| 13 | GET | `/platform/report/run/:id` | `platform:report:view` | — | 单次执行详情 |
| 14 | GET | `/platform/report/artifact/:id/download` | `platform:report:export` | DataScope | 下载产物；过期返回 `report.artifact.expired` |
| 15 | POST | `/platform/report/run/cleanup` | `platform:report:clear` | **SecureAction** | 手动触发保留期清理（与 maintenance 任务同一实现） |

SecureAction 的选取依据：与既有高敏动作先例同形
（`backend/modules/auth/module.go:97-108` 的清理/批量、`backend/modules/system/system_modules.go:188-191`
的批量删除与重置密码）。

## 4. DTO 契约

命名遵守 `scripts/harness/check-backend-dto-contract.mjs` 的 `DTO_LIKE_SUFFIXES`
（`Resp` / `Result` / `Summary` / `Option` 等），handler 不返回裸 `map[string]any`。

### 4.1 请求

```jsonc
// POST /platform/report/run  和  POST /platform/report/export
{
  "templateId": 1024,
  "versionId": 0,             // 0 = 跟随 current_version
  "format": "csv",            // 只允许 report_template.supported_formats 中的值
  "params": {                 // 受 report_dataset.param_schema_json 白名单约束
    "dateFrom": "2026-10-01",
    "dateTo": "2026-10-31",
    "deptIds": [3, 7]
  },
  "idempotencyKey": "",       // 可选；空则由服务端按 §9 生成
  "async": false              // true 时立即返回 runId，不返回行数据
}
```

### 4.2 响应

```jsonc
// GET /platform/report/template/list -> data
{
  "list": [
    {
      "id": 1024,
      "tenantId": 0,
      "code": "iam-user-governance",
      "name": "system.menu.report.iam_user_governance",
      "reportType": "governance",
      "category": "governance",
      "sourceDomain": "system.iam",
      "datasetCode": "iam.user.governance",
      "visibility": "platform",
      "supportedFormats": ["csv"],
      "sensitive": false,
      "status": "published",
      "currentVersionId": 2048,
      "updatedAt": "2026-11-04T10:00:00+08:00"
    }
  ],
  "total": 1,
  "page": 1,
  "pageSize": 20
}

// POST /platform/report/run -> data（ReportRunResultResp）
{
  "runId": 4096,
  "status": "succeeded",
  "rowCount": 137,
  "truncated": false,
  "durationMs": 412,
  "columns": [
    { "key": "username", "titleKey": "system.column.username", "type": "string" }
  ],
  "rows": [ { "username": "alice" } ],
  "asOfAt": "2026-11-04T10:05:00+08:00",
  "scopeMode": "dept_and_children",
  "artifactId": 0
}

// POST /platform/report/export -> data（ReportExportResp）
{
  "mode": "sync",             // "sync" 时响应体已是 CSV 流；此处为 async 分支
  "runId": 4097,
  "artifactId": 0,
  "statusUrl": "/api/v1/platform/report/run/4097"
}
```

字段约束：

| 字段 | 约束 | 依据 |
| :--- | :--- | :--- |
| `rows` | 预览行数 <= `min(dataset.row_limit, 10000)` | `backend/pkg/impexp/export_limits.go:9` |
| `scopeMode` | 只读回显，取值来自 `backend/pkg/common/data_scope.go:11-15` | 让使用者知道"这次是按什么范围算的" |
| `truncated` | 与 `status = truncated` 一致，前端必须显式提示 | 主设计 §8.4 |
| `name` | 存 i18n key，不存自然语言 | [DESIGN.md](../../DESIGN.md) §6 |
| `asOfAt` | 由 `freshness_seconds` 推出，缓存命中时仍是真实口径时间 | 主设计 §10 |

## 5. 分页、排序与上限

| 项 | 规则 | 依据 |
| :--- | :--- | :--- |
| 分页参数 | `page`（默认 1）、`pageSize`（默认 10，**服务端夹到 100**） | `common.NormalizePagination`（`backend/pkg/common/handler.go:39`）；各系统域一致传 100 |
| 排序参数 | `sort` + `order`，字段必须过白名单 | `common.NormalizeSort`（`backend/pkg/common/handler.go:57`） |
| 渲染行数 | 同步上限 10000；超过必须转异步 | `impexp.MaxExportRows()` |
| 时间字段 | 列表响应统一返回带时区的 RFC3339，由前端按用户时区展示 | `check:system-datetime-presentation`（`frontend/package.json`） |

## 6. 权限键登记

| 键 | 语义 | 前端按钮权限 | 菜单挂载点 |
| :--- | :--- | :--- | :--- |
| `platform:report:list` | 报表中心访问与清单 | 否（页面准入） | 报表中心菜单 |
| `platform:report:view` | 查看单个报表与历史 | 是 | 报表中心 / 查看 |
| `platform:report:run` | 执行渲染（预览） | 是 | 报表中心 / 运行 |
| `platform:report:export` | 导出与下载产物 | 是 | 报表中心 / 导出 |
| `platform:report:create` | 新建模板 | 是 | 模板管理 / 新建 |
| `platform:report:update` | 编辑与发布 | 是 | 模板管理 / 编辑、发布 |
| `platform:report:delete` | 删除模板 | 是 | 模板管理 / 删除 |
| `platform:report:clear` | 清理 run / artifact | 是 | 模板管理 / 清理 |

既有同形先例（照抄形状，不另造词）：`view` 作动作词见
`backend/modules/system/seed.go:210`（`PagePerm: "platform:dashboard:view"`）；
清理类动作用 `clear` 并挂在 `Type: "F"` 节点上，见 `backend/modules/system/seed.go:548`
（`system:operation-log:clear`）。

同步义务（合同要求，不做则视为未完成）：

1. 每个键都要有 Casbin `p` 策略与权限点登记。命名形状沿用既有扫描约定
   `platform:<资源>:<动作>`（`scripts/harness/check-permission-contract.mjs:30` 的
   `PERMISSION_KEY_PATTERN`）。该 checker 的**真实约束不是动作词白名单**：`ACTION_TOKENS`（`:11-28`）
   只用于启发式上下文判断，它产生的阻断级 finding 是"动作菜单节点（`Type: "F"`）绑定了 `...:list` 权限"
   （`:125-136`），另有"`:list` 出现在动作语境的权限判断旁"的告警（Go `:138-165`、TS `:170-208`）。
   因此本表 8 个键都可实现，但**必须**做到：读/导航门禁用 `:list`，动作类菜单节点一律绑动作键，
   不得把 `list` 当作动作兜底（[DESIGN.md](../../DESIGN.md) §4.4）。
2. 权限点必须同时登记菜单节点（`Type = "F"`）与 i18n permission 组，写法参照
   `backend/modules/auth/login/login_menu.go:81-89`（`Key / ParentKey / TitleKey / Perms / Type / Sort`）。
3. 列表权限不得兜底写权限（[DESIGN.md](../../DESIGN.md) §4.4）。

## 7. i18n 命名空间

| 组 | 前缀 | 示例键 |
| :--- | :--- | :--- |
| menu | `platform.menu.report_*` | `platform.menu.report_center` |
| permission | `platform.permission.report_*` | `platform.permission.report.export` |
| messages（错误） | `report.*` | `report.render.row_limit_exceeded` |
| 页面文案 | `platform.report.*` | `platform.report.truncated.tip` |

登记方式：`backend/modules/system/i18n/seed_data.go` 的 `{Module, Locale, Group, Key, Value}` 四元组
（示例见 `:79`），中英文两份必须同时给（示例同文件 zh-CN / en-US 成对出现）。
前端不得硬编码文案，由 `check:i18n-hardcode` 拦截（`frontend/package.json` 的 `prebuild` 链）。

## 8. 审计约定

| 动作 | 记录方式 |
| :--- | :--- |
| 所有非 GET 动作 | 操作日志中间件自动落 `system_log_oper`（`backend/internal/middleware/operation_log_middleware.go`） |
| 导出/下载 | handler 显式设置标题 key 与来源域，`oper_param` 写参数摘要而非原文（[DATABASE.md](./DATABASE.md) §3.2 脱敏纪律） |
| 渲染执行 | `report_run` 行本身（含 `scope_fingerprint`、`param_digest`） |
| 高敏动作 | SecureAction 403 与成功都有留痕：403 由中间件写错误键，成功由操作日志写 |

## 9. 幂等与并发

1. **导出幂等**：`idempotencyKey` 缺省时由服务端按
   `sha1(tenantId|templateId|versionId|format|paramDigest|scopeFingerprint)` 生成。
   同一 key 命中未过期 artifact 时直接复用，不重复渲染（主设计 §11）。
2. **并发**：同一 `idempotencyKey` 的并发请求只允许一个真正执行渲染，其余等待并复用结果；
   实现必须证明"两个并发只产出一次取数"（测试计划 TC-R-CON-2）。
3. **无副作用**：`run` / `export` 除写 `report_run` / `report_artifact` 外不改任何业务表。
4. 重复发布（同一 draft 连续 publish 两次）第二次返回 `report.version.immutable`，不得产生第二个版本行。

## 10. 兼容与版本策略

- 第一阶段不引入 URL 版本段（沿用 `/api/v1`），契约破坏性变更须走
  [PLATFORM_CONTRACT.md](../contracts/PLATFORM_CONTRACT.md) §7.5 与文档治理合同流程。
- `supportedFormats` 增加新值属于**兼容扩展**，但必须先满足主设计 §8.2 的依赖前置。
- 字段只增不改语义；`scopeMode` / `asOfAt` 若缺失，前端必须按"未知范围"降级为不放行渲染。

## 11. 开放决策 / Open decisions

| 编号 | 决策 | 选项 | 决策人 |
| :--- | :--- | :--- | :--- |
| OD-API-1 | `run`（渲染）是否需要独立于 `view`（查看）的权限键 | A：独立键（当前契约，渲染是成本动作，需可与只读查看分离授权）；B：合并进 `platform:report:view`，靠行数上限与速率限制兜成本 | 维护者 |
| OD-API-2 | 大报表异步产物的进度查询形态 | A：轮询 `GET /run/:id`；B：新增 SSE 事件流 | 维护者（第一阶段倾向 A） |
| OD-API-3 | 产物下载是否走短期签名 URL | A：统一走应用内下载接口；B：S3 driver 时用 presigned URL | 维护者（B 需处理租户再校验，见主设计 OD-5） |
| OD-API-4 | `run` 与 `export` 是否合并成一个端点 | A：分离（当前契约）；B：合并为 `POST /platform/report/render`，用 `mode` 区分 | 维护者 |

## 12. 关联文档

- 主设计：[REPORT_CENTER_DESIGN.md](./REPORT_CENTER_DESIGN.md) / [REPORT_CENTER_DESIGN.en.md](./REPORT_CENTER_DESIGN.en.md)
- 数据模型：[REPORT_CENTER_DB_DESIGN.md](./REPORT_CENTER_DB_DESIGN.md) / [REPORT_CENTER_DB_DESIGN.en.md](./REPORT_CENTER_DB_DESIGN.en.md)
- 测试计划：[REPORT_CENTER_TEST_PLAN.md](./REPORT_CENTER_TEST_PLAN.md) / [REPORT_CENTER_TEST_PLAN.en.md](./REPORT_CENTER_TEST_PLAN.en.md)
- 调度中心接口：SCHEDULER_CENTER_API_CONTRACT.md（planned/unavailable：`./SCHEDULER_CENTER_API_CONTRACT.md`；无当前交付承诺）
- 目录索引：[README.md](./README.md)
