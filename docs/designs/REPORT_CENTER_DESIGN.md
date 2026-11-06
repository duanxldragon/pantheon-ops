---
title: 报表中心设计
doc_type: Design
layer: platform / business/*
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
updated_at: 2026-11-04
---

# 报表中心设计

> 未实现声明：本文为设计稿，仓库内尚无对应实现；在实现落地前状态保持 Draft。
> Not implemented: this is a design draft with no corresponding implementation in the repository; it stays Draft until implemented.

关联合同：
- `PLATFORM_CONTRACT.md`
- `SYSTEM_IAM_CONTRACT.md`
- `TENANT_CONTRACT_V1.md`

关联路线：
- `P2_SCALE_ROADMAP.md`（§8.3 调度中心 / 报表中心 / 监控告警中心）
- `PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md`

本包的其余三份文档（数据模型、接口契约、测试计划）见 §17。

---

本文定义 Pantheon 标准后台产品位中的"报表中心"最小骨架。

目标不是立刻做成 BI 平台，而是先统一报表清单、查看入口、导出边界和权限审计语义。

## 1. 设计目标

报表中心要解决：

1. 平台层对各域报表有统一入口，而不是散落在业务页面里
2. 报表定义、适用权限、导出动作有一致治理模型
3. 后续 dashboard、调度中心和报表归档可以复用同一套元数据
4. 大报表有明确的行数/超时/异步边界，不允许无界查询

## 2. 非目标

当前阶段不做：

- 自助式拖拽分析器
- 多维 OLAP 引擎
- 可视化数据建模工作台
- 海量交互式图表设计器
- 面向最终用户的自由 SQL 查询入口（对齐 §5.1；`system/generator` 的外部数据源至今只允许
  `information_schema` 结构探查，见 [DATABASE.md](./DATABASE.md) §3.3.4）
- 跨租户联邦查询（[TENANT_CONTRACT_V1.md](../contracts/TENANT_CONTRACT_V1.md) §1 Out）

## 3. 归属边界

### 3.1 `platform` 负责

- 报表中心入口与聚合清单
- 报表元数据视图（dataset / template / version / run / artifact）
- 渲染管线的编排与状态机
- 导出动作入口治理（权限键、审计、行数与超时边界）

### 3.2 源域负责

- 报表数据生成逻辑（取数器实现）
- 数据口径定义与列语义
- 报表详情实现
- 导出内容正确性

边界依据：[PLATFORM_CONTRACT.md](../contracts/PLATFORM_CONTRACT.md) §7.1「聚合层可以读取多个子域数据，但不能侵入子域内部职责」，
以及 [DESIGN.md](../../DESIGN.md) §2.3「业务模块禁止直接依赖底座某个具体 Service/Repository」——反向同样成立：
报表中心不 import 任何 `modules/system/*` 或 `modules/business/*` 的 Service，只消费源域注册的取数器接口
（装配参照 `backend/modules/platform/routes.go` 的 `OrgGovernanceTaskLoader` 注入方式，边界规则见
[REPOSITORY_LAYOUT.md](./REPOSITORY_LAYOUT.md)）。

## 4. 端到端链路

```text
dataset（口径注册） -> template（呈现与参数） -> version（发布） ->
run（渲染：鉴权 -> 取数 -> 格式化 -> 落 artifact） -> export（下载） / schedule（定时）
```

| 阶段 | 责任对象 | 落库位置 | 关键约束 |
| :--- | :--- | :--- | :--- |
| dataset | 源域开发 | `report_dataset` | 只有已注册口径可被报表引用 |
| template | 报表作者 | `report_template` | 可见性与权限键在定义期声明 |
| version | 报表作者 | `report_template_version` | 发布后不可变，被 run/schedule 引用 |
| run | 报表中心 | `report_run` | 必须记录 scope 指纹与参数摘要 |
| artifact | 报表中心 | `report_artifact` | 对象 key 必须带租户维度 |
| schedule | 调度中心 | `schedule_job` | 见 §11，本中心不自己实现 cron |

## 5. 数据源模型（dataset）

### 5.1 口径即注册能力，不是查询入口

`report_dataset` 是"一个可复用的取数口径"的最小登记单元。第一阶段的 dataset 只有一种来源类型：

| 字段 | 语义 |
| :--- | :--- |
| `code` | 口径标识，平台内唯一 |
| `name` / `description` | 展示与说明（展示文本走 i18n key，见 §13） |
| `source_domain` | 归属域，如 `system.iam` / `system.audit` / `business.*` |
| `provider_key` | 取数器标识，由源域在模块注册时登记，报表中心按 key 调用 |
| `param_schema_json` | 参数白名单：名称、类型、是否必填、最大值/最大集合长度 |
| `scope_binding` | 行级范围绑定：`none` / `dept` / `self`（见 §9） |
| `row_limit` | 该口径单次取数的行数上限，必须 <= 全局同步上限 |
| `timeout_ms` | 该口径的取数超时 |
| `cacheable` | 是否允许结果缓存（默认 false） |
| `freshness_seconds` | 口径允许的数据新鲜度，用于展示"截至"时间 |

约束：

1. 报表中心不接收 SQL 文本、表名或列表达式；dataset 只声明"用哪个口径 + 传哪些参数"。
2. dataset 定义是 `platform-global` 资源（与 `system_menu` / `system_permission` 同一裁决方向，
   见 [TENANT_RESOURCE_SCOPE_MATRIX.md](./TENANT_RESOURCE_SCOPE_MATRIX.md) §2 的 2026-10-15 条目）：
   口径是发布物，不由租户各自维护；租户差异由 Casbin 与行级范围表达。
3. dataset 由代码侧注册（模块 seed / 注册表），运行期只读；控制台的 dataset CRUD 不在第一阶段。

### 5.2 与既有导入导出面的关系

`backend/pkg/impexp` 已经承载了系统域的 CSV 导入导出。报表中心是它的**消费方**，不新增第二套导出实现：

| 既有能力 | 文件 | 报表中心如何用 |
| :--- | :--- | :--- |
| CSV 写出（BOM、公式注入中和、RFC 5987 文件名） | [../backend/pkg/impexp/csv.go](../../backend/pkg/impexp/csv.go)（`WriteCSV` 第 56 行、`sanitizeCSVCell` 第 33 行） | 所有 CSV 产物必须走该函数，列值经 `sanitizeCSVRow` |
| 同步导出行数 hard cap = 10000 | [../backend/pkg/impexp/export_limits.go](../../backend/pkg/impexp/export_limits.go)（`MaxExportRows()` 第 9 行） | 同步渲染的天花板；SQL 必须 `Limit()` 下推，参照 `backend/modules/system/org/post/post_service.go:537` |
| 导入接收五步流水线 | `backend/pkg/impexp/pipeline.go`（`ReadImportCSV`、`ImportFileField = "file"`） | 报表中心不做导入；如需"报表参数批量导入"必须复用同一接收面 |
| 导入模板 CSV 组装 | `backend/pkg/impexp/template.go`（`BuildImportTemplate`） | 参数模板/示例行下载复用 |
| 治理型导出列头与标签 | `backend/pkg/impexp/governance_template.go` | 治理类报表（§14）直接复用其标签映射，不再造一份 |

## 6. 报表模板定义与版本化

### 6.1 模板

`report_template` 描述"面向使用者的一份报表"：绑定 dataset、声明可见性、声明支持的导出格式、声明权限键与默认参数。

最小字段（继承草稿的元数据模型并落到可执行约束）：

| 字段 | 语义 |
| :--- | :--- |
| `id` / `tenant_id` | 主键 / 租户维度（见 §9.3） |
| `code` | 租户内唯一（删除后由重命名释放，见 DB 设计 §5） |
| `report_type` | 分类语义：`list` / `summary` / `governance` / `audit` |
| `name` / `description` | 展示名（i18n key）与说明 |
| `category` | 列表页筛选用分类 |
| `dataset_code` | 绑定的口径 |
| `visibility` | `private` / `team` / `platform`（草稿裁决保留） |
| `owner_user_id` | `private` 可见性的归属人 |
| `supported_formats` | 逗号分隔，取值只允许已实现格式（第一阶段 `csv`，见 §8） |
| `default_param_json` | 默认参数 |
| `permission_key` | 查看该报表所需的权限键（见 §9.1） |
| `status` | `draft` / `published` / `disabled` |
| `current_version` | 指向已发布版本 |

`visibility` 只解决"哪些人能在清单里看到这张报表"；"看到哪些行"由 §9 的行级范围决定，两者不可混写
（对齐 [SYSTEM_IAM_CONTRACT.md](../contracts/SYSTEM_IAM_CONTRACT.md) §「菜单不是权限模型本体」的同源纪律）。

### 6.2 版本

1. 版本内容 = 呈现规格（列集合、列序、格式化、参数默认值）+ 绑定 dataset 的快照。
2. `published` 版本**不可变**：编辑已发布模板时写入新的 `draft` 版本行，发布后 `current_version` 指针前移。
3. 每次 `report_run` 记录 `template_version_id`，保证历史导出可复现口径。
4. 定时任务（§11）引用 `template_id + version` 语义：默认跟随 `current_version`，
   允许显式钉住版本；钉住版本被删除时调度任务必须进入 `disabled`，不留悬空引用。
5. 回滚 = 把 `current_version` 指回某个历史已发布版本，不改写历史行。

## 7. 渲染管线

单次渲染的状态机（`report_run.status`）：

```text
pending -> running -> succeeded
                   -> failed
                   -> cancelled
running  -> truncated（命中行数上限并仍产出文件时）
```

执行步骤，顺序是安全属性而不是风格：

1. **路由鉴权**：Casbin（`backend/pkg/contracts/route_groups.go:137` 的
   `TokenAuth -> PlatformScopeTenantContext -> Casbin` 链）。
2. **模板准入**：`status = published` + `visibility` + `permission_key`。
3. **行级范围解析**：读取 `common.GetDataScope(c)`（由
   `backend/internal/middleware/data_scope_middleware.go:102` 注入），生成 scope 指纹。
4. **参数校验**：按 `param_schema_json` 白名单校验，超界直接 `param.invalid`。
5. **取数**：按 `provider_key` 调用源域取数器，强制 `LIMIT row_limit`、`timeout_ms`；
   `scope_binding != none` 时通过 `db.Scopes(database.WithDataScope(...))`
   （`backend/pkg/database/scope.go:12`）下推，**不允许**在应用层过滤全表。
6. **格式化**：列映射 + 数值/时间格式化（时间展示遵循 `frontend` 的系统时间呈现门禁
   `check:system-datetime-presentation`，见 `frontend/package.json`）。
7. **产出**：
   - 同步：直接 `impexp.WriteCSV` 流回；
   - 异步：写入 artifact（本地或 S3 driver，复用 `backend/pkg/upload`，见 §8.3），`report_run` 记录 artifact id。
8. **落审计**：写 `report_run`，并由操作日志中间件（`backend/internal/middleware/operation_log_middleware.go`）
   记录请求级审计。

约束：

- 步骤 5 失败时**不得**降级返回空集合冒充"无数据"：必须落 `failed` 并给出错误 key。
- 渲染过程不写业务表；`report_run` / `report_artifact` 是仅有的两张写表。

## 8. 导出格式

### 8.1 第一阶段只承诺 CSV

`csv` 是当前仓库唯一已实现的导出编码路径（§5.2）。`supported_formats` 允许声明的取值集合第一阶段为 `{csv}`。

### 8.2 xlsx / pdf 不写进承诺

草稿建议第一阶段支持 `csv / xlsx / pdf`。落地事实核对结果：`backend/go.mod` 的 require 列表里没有
xlsx（excelize 等）也没有 pdf（maroto/gotenberg 等）依赖，`backend/pkg/impexp` 只有 CSV 实现。
因此本设计把 xlsx / pdf 明确降级为**开放决策**（§16 OD-1），在依赖获批前：

- 接口层不暴露 xlsx/pdf 选项；
- 前端不展示对应按钮（对齐 [DESIGN.md](../../DESIGN.md) §7.5「未实现真实行为的控件不展示为可点击能力」）。

### 8.3 产物载体

异步产物落对象存储时复用 `backend/pkg/upload/service.go` 的 driver 抽象
（`upload.storage_driver` 取 `local` / `s3`，对象 key 由 `scope + 日期 + uuid` 组装）。
报表侧的 scope 段必须携带租户维度——理由与 [TENANT_RESOURCE_SCOPE_MATRIX.md](./TENANT_RESOURCE_SCOPE_MATRIX.md) §2
"导出文件 / 上传对象：`derived`，key 必须可携带租户维度，避免串租户命中"一致。

### 8.4 大报表边界（禁止无界查询）

| 边界 | 取值来源 | 违反时行为 |
| :--- | :--- | :--- |
| 同步渲染行数 | `impexp.MaxExportRows()` = 10000（`backend/pkg/impexp/export_limits.go:9`） | 转为异步任务，或 `truncated` + 明确提示，禁止静默截断 |
| dataset 行数 | `report_dataset.row_limit`，且 `<= 10000` | 参数校验拒绝（400） |
| 列表分页 | `common.NormalizePagination(page, pageSize, 100)`（`backend/pkg/common/handler.go:39`；各系统域一致传 100） | pageSize 被夹到 100 |
| HTTP 写超时 | `backend/cmd/server/main.go` 的 `WriteTimeout`（同步下载必须留余量） | 超阈值必须走异步 |
| 单请求速率 | `middleware.RateLimiter`（`backend/cmd/server/main.go` 的 `registerAPIRoutes`） | 429 |

任何"全表扫描后再筛"的实现形态都在本设计禁止之列。

## 9. 权限模型：谁能看到哪些行

三层正交，逐层收窄，任何一层不得替代另一层。

### 9.1 路由层（Casbin 权限键）

- 权限键命名沿用既有形状 `platform:<资源>:<动作>`（`scripts/harness/check-permission-contract.mjs:30` 的
  `PERMISSION_KEY_PATTERN`）。该 checker 的阻断级 finding 不是"动作词必须在白名单内"，而是
  "动作菜单节点（`Type: \"F\"`）绑定了 `...:list` 权限"（`:125-136`）；`ACTION_TOKENS`（`:11-28`）
  只用于启发式告警上下文。
- 本包用的键集合（完整登记见 [REPORT_CENTER_API_CONTRACT.md](./REPORT_CENTER_API_CONTRACT.md) §6）。
- 每个动作必须有独立键，不得用 `...:list` 兜底写动作
  （[DESIGN.md](../../DESIGN.md) §4.4「前端按钮权限不应再使用 `system:user:list` 同时兜底新增/编辑/删除」）。

### 9.2 行级层（复用数据范围中间件语义，不重新实现）

报表中心**直接复用** `backend/internal/middleware/data_scope_middleware.go` 的语义，不自建第二套行级模型：

| dataset 的 `scope_binding` | 复用机制 | 语义来源 |
| :--- | :--- | :--- |
| `none` | 不加行级谓词 | 口径本身不含行归属（如全局统计摘要） |
| `dept` | `database.WithDataScope` → `dept_id = ?` / `dept_id IN ?` | `backend/pkg/database/scope.go:38-57` |
| `self` | `database.WithDataScope` → `created_by` / `create_by` / `id` | `backend/pkg/database/scope.go:70-96` |

必须原样继承的既有裁决（逐条来自已读实现）：

1. **模式集合固定**：`all` / `self` / `dept` / `dept_and_children` / `custom`
   （`backend/pkg/common/data_scope.go:11-15`），多角色取最宽（`data_scope_middleware.go:368` `resolveDataScopeMode`）。
2. **fail-closed**：策略读不到、表缺失、mode 非法时，非管理员请求被拒（HTTP 500 或 JSON 错误信封），
   且 scope 被写成 `custom` + 空 dept 集合，使 `WithDataScope` 落入 `1 = 0`
   （`data_scope_middleware.go:117-131`、`:210-213`、`backend/pkg/database/scope.go:66-68`）。
   报表渲染必须把这种情形判为 `failed`，绝不能渲染成"空报表"。
3. **管理员放宽只作用于自身请求**：`admin` 与 `tenant_admin` 视为其作用域内的完整管理员，跳过 dept 策略
   （`data_scope_middleware.go:455` `hasAdministrativeRole`）。
4. **租户内角色豁免 dept 策略**：`tenant_admin` / `tenant_member` 的行收窄来自租户 scope，
   不受平台 dept 策略管辖（`data_scope_middleware.go:478` `hasTenantLocalRole`）。
5. **策略按租户读取**：`system_role_data_scope` 的唯一键是 `(tenant_id, role_key)`，
   策略读必须带 tenant 谓词（`data_scope_middleware.go:45-51`、`:250`）。
6. **`dept_and_children` 依赖 `system_dept.ancestors`** 展开子部门（`data_scope_middleware.go:324`），
   报表侧不复制这段展开逻辑。

报表中心自己的表**不套用** dept 行级模型（报表定义不是部门归属数据）；行级只作用于 dataset 产出的业务行。

### 9.3 租户层

- 路由链必须走 `contracts.SystemProtectedGroup(...)` 同形的
  `TokenAuth -> PlatformScopeTenantContext -> Casbin`，使 `compat` / `multi` 双模语义与系统域一致
  （`backend/pkg/contracts/route_groups.go:137`；双模语义见
  [TENANT_CONTRACT_V1.md](../contracts/TENANT_CONTRACT_V1.md) §6）。
- 若报表入口最终挂在 `platform` 模块而非 `/system` 前缀下，也必须显式声明平台作用域，
  否则缺上下文时按合同 §3.2/§5 走 `TENANT_CONTEXT_MISSING` fail-closed，不做静默放大。
- 资源分类登记（写进 DB 设计）：dataset `platform-global`；template/version `tenant-owned`；
  run/artifact `derived`（跟随来源）。
- `business/*` 路由在 V1 不参与租户隔离（合同 §2.6、`backend/pkg/contracts/route_groups.go:21-26`）：
  业务报表的租户维度只能来自业务行自身的列，不能靠给业务路由挂租户中间件"造隔离"。

### 9.4 高敏导出

以下动作按既有先例要求二次验证 `X-Operation-Token`
（`backend/internal/middleware/secure_action_middleware.go`，先例见 `backend/modules/auth/module.go:97-108`
与 `backend/modules/system/system_modules.go:188-191`）：

- 删除模板
- 发布版本
- 触发被标记为高敏（`sensitive = true`）的报表导出
- 批量清理 run / artifact

## 10. 缓存

| 层 | 内容 | key 组成 | 失效 |
| :--- | :--- | :--- | :--- |
| 结果缓存 | dataset 取数结果（仅 `cacheable = true`） | 连接命名空间 + `tenantID` + `datasetCode` + `paramDigest` + `scopeFingerprint` + `freshness` | TTL（`freshness_seconds`） |
| 产物缓存 | artifact 文件 | 对象 key（含租户段） | 保留期（DB 设计 §6） |
| 元数据缓存 | template/version 定义 | `tenantID` + `templateID` + `current_version` | 发布动作显式广播 |

约束：

1. **`scopeFingerprint` 不可省略**：由 `mode` + 归一化后的 `deptIDs` + `userID` 派生。
   缺失该段就等于让 `dept` 范围的用户命中 `all` 范围用户的结果——这是行级权限泄漏，不是性能问题。
2. 缓存命名空间必须来自连接身份而非连接池指针，沿用既有教训
   （`backend/internal/middleware/data_scope_middleware.go:315-322` 与 `database.ConnectionCacheNamespace`）。
3. 元数据失效广播复用既有跨实例机制方向：`PANTHEON_CASBIN_WATCHER` + Redis（
   [BACKEND.md](./BACKEND.md) §「Casbin 多实例广播」）。第一阶段允许"无跨实例广播，靠 TTL 收敛"，
   但必须把它写成显式 gap，不得声称已实时同步。
4. 缓存只在进程内或 Redis；不引入新的存储组件。

## 11. 与调度中心的衔接

分工（不重复定义，细节见 [SCHEDULER_CENTER_DESIGN.md](./SCHEDULER_CENTER_DESIGN.md)）：

- 调度中心负责 cron 解析、触发、多实例 single-flight、重试与执行历史。
- 报表中心只提供两样东西：
  1. 任务处理器 `report.generate`（`handler_key`），入参为 `templateId`、`versionPolicy`、`paramJson`、`receiverScope`；
  2. 产出写回：一次调度执行生成 `report_run`（`trigger_source = schedule`）与 `report_artifact`。
- 报表侧**不得**自建 cron、自建定时器、自建锁；注册与启停只通过调度中心的接口契约
  （SCHEDULER_CENTER_API_CONTRACT.md（planned/unavailable：`./SCHEDULER_CENTER_API_CONTRACT.md`；无当前交付承诺） §5）。
- 失败语义：报表渲染失败 → 处理器返回分类错误 key → 调度中心按重试策略处理；
  达到 `retry_max` 后由调度中心产出失败摘要，供告警中心消费
  （[ALERT_MONITORING_CENTER_DESIGN.md](./ALERT_MONITORING_CENTER_DESIGN.md)）。
- 幂等：同一次计划触发的重复投递由调度中心的幂等键抑制；报表侧的去重键是
  `(template_id, schedule_fire_key)`，重复命中时复用既有 artifact，不重复产出。

## 12. UI 骨架

第一阶段最小 UI（列表页遵循 [FRONTEND_PAGE_TEMPLATES.md](./FRONTEND_PAGE_TEMPLATES.md) 与 `SearchToolbar` 契约，
状态覆盖遵循 [EMPTY_LOADING_ERROR_STATES.md](./EMPTY_LOADING_ERROR_STATES.md)）：

- 报表中心列表页：分类筛选、来源域筛选、可见性指示、更新时间、说明
- 报表详情页：参数区 + 预览表格（分页 10/20/50/100）+ 导出入口
- 模板管理页：草稿编辑、版本列表、发布/回滚
- 我的导出/历史页：`report_run` 列表 + artifact 下载 + 状态（含 `truncated`）
- 无权限 / 空 / 加载 / 提交中 四态必须齐备

暂不强制：

- 大屏设计器、图表布局编辑器、订阅能力

## 13. 审计与权限留痕

页面动作 → 权限键 → 审计记录三段必须齐全：

| 必须记录 | 载体 |
| :--- | :--- |
| 谁查看了报表 | `report_run`（`trigger_source = manual`）+ `system_log_oper` |
| 谁导出了报表、什么格式 | `report_run.format` + `report_artifact` |
| 过滤条件摘要 | `report_run.param_digest` + `scope_fingerprint`（存摘要，不存明文大参数） |
| 谁改了模板/发布了版本 | `system_log_oper` + `report_template_version` 行本身 |
| 高敏动作的二次验证 | `SecureActionMiddleware` 的 403 分支已有错误键，成功动作靠操作日志 |

- `system_log_oper` 已有 `source_domain / source_page / failure_category / cost_time` 派生列与索引，
  报表侧只需设置标题 key 与来源域，不再新增审计列（[DATABASE.md](./DATABASE.md) §3.2、
  `backend/internal/middleware/operation_log_middleware.go`）。
- i18n：菜单标题走 `title_key`，权限点走 permission 组，错误返回 key 不返回自然语言
  （[ERROR_CODE_AND_I18N.md](./ERROR_CODE_AND_I18N.md) §3.1；seed 结构见
  `backend/modules/system/i18n/seed_data.go:79`，菜单/权限登记方式见
  `backend/modules/auth/login/login_menu.go:81-89`）。
- 保留期与清理走 `maintenance` 注册表，不在请求路径内做全表 DELETE
  （`backend/pkg/maintenance/maintenance.go` 的包注释；先例 `backend/modules/system/audit/audit_service.go:317`）。

## 14. 最小样板要求

第一阶段至少接入一类真实报表（优先选已有导出实现、可直接映射到 `impexp` 的）：

1. 审计事件汇总报表（`system.audit`，`scope_binding = none`）
2. 用户与权限治理报表（`system.iam`，可复用 `backend/pkg/impexp/governance_template.go` 的标签集）
3. 业务资源盘点报表（`business/*`，注意 §9.3 的业务域边界）

样板要求的是"真实口径 + 真实权限 + 真实导出留痕"，不是 demo 数据。

## 15. 完成定义

第一阶段完成标准：

- 有统一报表元数据模型（DB 设计四张业务表落地并有迁移）
- 有报表中心入口（菜单 + 路由 + i18n + 权限点同步登记）
- 有查看与导出动作契约（API 契约文档 §3 的路由全部有实现与测试）
- 有权限与审计留痕（§9/§13 的每条约束有对应测试或 smoke）
- 有大报表边界（§8.4 的行数/超时/异步转换可被测试证明）
- 有至少一个真实报表样板接入
- 测试计划（[REPORT_CENTER_TEST_PLAN.md](./REPORT_CENTER_TEST_PLAN.md)）中的 P0 用例全绿或有显式豁免

不是以"已经做成完整 BI 平台"作为完成标准。

## 16. 开放决策 / Open decisions

| 编号 | 决策 | 选项 | 影响 | 决策人 |
| :--- | :--- | :--- | :--- | :--- |
| OD-1 | xlsx / pdf 是否进入第一阶段 | A：只做 CSV；B：新增纯 Go xlsx 依赖；C：pdf 走外部渲染服务 | B/C 需要新增依赖（AGENTS「不引入未要求的新依赖」），且 pdf 引入外部服务边界 | 维护者 |
| OD-2 | 报表入口挂在 `platform` 还是 `system` 路由组 | A：`/api/v1/platform/report/*`；B：`/api/v1/system/report/*` | 影响 Casbin 策略形状、`PlatformScopeTenantContext` 是否复用 `SystemProtectedGroup`、菜单归属 | 维护者 |
| OD-3 | dataset 注册方式 | A：代码注册表（模块 seed，类似菜单/权限）；B：控制台可建（需要 SQL 白名单治理） | B 会把"自由查询入口"风险引回，本设计倾向 A | 维护者 |
| OD-4 | `visibility = team` 的"团队"语义 | A：按 `system_dept` 子树；B：按角色；C：第一阶段不做 team | 仓库当前无"团队"实体，只有 dept 与 role；C 是最小可行 | 维护者 |
| OD-5 | 异步产物是否需要保留原始行级可见性再校验 | A：下载时重新校验请求者 scope；B：产物一旦生成即同租户可见 | A 更安全但要求 artifact 记录 scope 指纹并可二次过滤 | 维护者 |
| OD-6 | 报表租户分类是否需要回写 scope 矩阵 | A：实现 PR 同步更新 `TENANT_RESOURCE_SCOPE_MATRIX.md`；B：另开文档任务 | 本设计包只写设计，不改矩阵；实现时必须同步 | 维护者 |

## 17. 关联文档

| 文档 | 中文 | 英文 |
| :--- | :--- | :--- |
| 主设计（本文） | `REPORT_CENTER_DESIGN.md` | [REPORT_CENTER_DESIGN.en.md](./REPORT_CENTER_DESIGN.en.md) |
| 数据模型与迁移 | [REPORT_CENTER_DB_DESIGN.md](./REPORT_CENTER_DB_DESIGN.md) | [REPORT_CENTER_DB_DESIGN.en.md](./REPORT_CENTER_DB_DESIGN.en.md) |
| 接口契约 | [REPORT_CENTER_API_CONTRACT.md](./REPORT_CENTER_API_CONTRACT.md) | [REPORT_CENTER_API_CONTRACT.en.md](./REPORT_CENTER_API_CONTRACT.en.md) |
| 测试计划 | [REPORT_CENTER_TEST_PLAN.md](./REPORT_CENTER_TEST_PLAN.md) | [REPORT_CENTER_TEST_PLAN.en.md](./REPORT_CENTER_TEST_PLAN.en.md) |

调度侧配套：[SCHEDULER_CENTER_DESIGN.md](./SCHEDULER_CENTER_DESIGN.md)。
目录索引：[README.md](./README.md)。

## 变更日志

| 日期 | 变更 |
| :--- | :--- |
| 2026-11-04 | 补齐数据源模型、模板版本化、渲染管线、导出边界、权限三层复用、缓存与大报表限制、调度衔接；新增开放决策与本包文档索引 |
| 2026-10-31 | 初始草稿 |
