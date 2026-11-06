---
title: 报表中心测试计划
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
updated_at: 2026-11-04
---

# 报表中心测试计划

> 未实现声明：本文为设计稿，仓库内尚无对应实现；在实现落地前状态保持 Draft。
> Not implemented: this is a design draft with no corresponding implementation in the repository; it stays Draft until implemented.

主设计：[REPORT_CENTER_DESIGN.md](./REPORT_CENTER_DESIGN.md)
数据模型：[REPORT_CENTER_DB_DESIGN.md](./REPORT_CENTER_DB_DESIGN.md)
接口契约：[REPORT_CENTER_API_CONTRACT.md](./REPORT_CENTER_API_CONTRACT.md)

---

## 1. 范围与判定原则

1. 本计划只覆盖**第一阶段承诺**（主设计 §15 完成定义）。xlsx/pdf（OD-1）、SSE 进度（OD-API-2）、
   presigned URL（OD-API-3）不在用例范围内——它们未进入承诺，为其写用例等于把开放决策变成需求。
2. 每条用例必须落到**一个既有门禁**（`package.json` / `frontend/package.json` 的脚本，或
   `.github/workflows/` 的具名 job）。落不到门禁的用例必须显式写进 §9「运行时缺口」，
   不允许以"应当没问题"的形式留在文档里。
3. 优先级定义：
   - **P0**：证明安全属性（权限、租户隔离、行数上限、fail-closed、幂等）。P0 不全绿即不视为完成。
   - **P1**：证明契约形状（DTO、信封、分页、i18n、菜单）与主链路可用。
   - **P2**：体验与可观测性（状态覆盖、提示文案、执行历史可读）。
4. 用例编号：`TC-R-<类别>-<序号>`，类别 `N`（正常）/ `B`（边界）/ `E`（错误）/ `CON`（并发）/
   `SEC`（权限与租户）/ `DB`（迁移与保留期）/ `CT`（静态契约门禁）/ `FE`（前端与 smoke）。

## 2. 验收标准 → 用例组（DoD 映射）

| 主设计 §15 完成标准 | 覆盖用例 | 落点门禁 |
| :--- | :--- | :--- |
| 四张业务表落地并有迁移 | TC-R-DB-1 ~ TC-R-DB-5 | `unit-tests` / `backend-tests`（CI job） |
| 报表中心入口（菜单 + 路由 + i18n + 权限点） | TC-R-CT-1 ~ TC-R-CT-6、TC-R-FE-1 | `check:harness-permission-contract`、`check:menu-contract`、`check:i18n-hardcode` |
| 查看与导出契约全部有实现与测试 | TC-R-N-*、TC-R-E-* | `backend-tests`（`go test -race ./...`） |
| 权限与审计留痕每条约束有测试或 smoke | TC-R-SEC-1 ~ TC-R-SEC-9 | `internal/middleware` 既有测试族 + `smoke-tenant` |
| 大报表边界可被测试证明 | TC-R-B-1 ~ TC-R-B-7 | `backend-tests` |
| 至少一个真实报表样板接入 | TC-R-N-5、TC-R-FE-3 | `pr-smoke-modules` / `test:smoke:system:governance` |
| P0 全绿或有显式豁免 | §10 豁免流程 | `quality-gates` |

## 3. 正常路径用例

| 编号 | 级别 | 场景 | 期望 | 落点 |
| :--- | :--- | :--- | :--- | :--- |
| TC-R-N-1 | P1 | 已发布模板 + 合法参数 → `POST /platform/report/run` | 200、`data.status = succeeded`、`columns/rows` 非空、`report_run` 落一行 | `backend-tests`（handler 集成测试） |
| TC-R-N-2 | P1 | 小报表 `POST /platform/report/export`（同步分支） | 响应头 `Content-Disposition` 为附件、CSV 带 BOM、单元格经公式注入清洗 | 复用 `impexp.WriteCSV` 既有断言（`backend/pkg/impexp/csv_test.go`） |
| TC-R-N-3 | P1 | 大报表 `async = true` | 立即返回 `runId`，`statusUrl` 可达，产物完成后 `artifactId` 非空且可下载 | `backend-tests` |
| TC-R-N-4 | P1 | 模板发布 → 回滚 → 再导出 | 导出命中回滚后的版本；`current_version_id` 前移/回退正确；已发布版本行内容不变（不可变性） | `backend-tests` |
| TC-R-N-5 | P0 | 真实样板（用户与权限治理报表）端到端 | 口径来自真实表、导出行数与直接 SQL 统计一致、审计有导出留痕 | `test:smoke:system:governance` + `system_log_oper` 断言 |
| TC-R-N-6 | P1 | 执行历史分页与详情 | `list` 按 `tenant_id` 过滤、`total` 正确、详情含 `scopeMode` 与 `paramDigest` | `backend-tests` |
| TC-R-N-7 | P2 | 详情页预览 + 分页 10/20/50/100 | 前端 SearchToolbar 契约形状、无硬编码文案 | `check:search-toolbar-contract`、`check:i18n-hardcode` |

## 4. 边界用例

行数上限全部以既有常量为准，不在测试里另写魔法数字。

| 编号 | 级别 | 边界 | 期望 | 依据 |
| :--- | :--- | :--- | :--- | :--- |
| TC-R-B-1 | P0 | 请求 `pageSize = 500` 的列表 | 服务端夹到 100 返回，不报错也不放大 | `common.NormalizePagination`（`backend/pkg/common/handler.go:39`），既有断言见 `backend/pkg/common/pagination_test.go` |
| TC-R-B-2 | P0 | dataset 命中 10000 行、同步导出 | 转异步或 `status = truncated` + 显式提示；**禁止**静默截断 | `impexp.MaxExportRows()`（`backend/pkg/impexp/export_limits.go:9`） |
| TC-R-B-3 | P0 | `row_limit` 注册为 >10000 | 参数校验 400 拒绝，不落库 | 主设计 §8.4 |
| TC-R-B-4 | P1 | 渲染耗时逼近 `WriteTimeout` | 必须走异步分支；同步分支不得把连接挂死 | `backend/cmd/server/main.go` 的 server 超时配置 |
| TC-R-B-5 | P1 | `versionId = 0` 与 `versionId = 已发布版本` 两分支 | 两者结果一致（跟随 current）；`versionId` 指向 draft 时拒绝 | API 契约 §4.1 |
| TC-R-B-6 | P1 | 时间/数值列格式化（含跨年、时区、零值） | 导出值与预览值一致；列表时间为带时区 RFC3339 | `check:system-datetime-presentation`（`frontend/package.json`） |
| TC-R-B-7 | P2 | 参数含非白名单键 | `report.param.invalid`，不回显非法键值 | API 契约 §2 |

## 5. 错误路径用例

| 编号 | 级别 | 故障注入 | 期望 | 依据 |
| :--- | :--- | :--- | :--- | :--- |
| TC-R-E-1 | P0 | 取数器返回 error | `report_run.status = failed` + 错误键；**不得**返回空集合冒充"无数据" | 主设计 §7 约束 |
| TC-R-E-2 | P0 | 数据范围策略缺失/表不存在/mode 非法 | 渲染判为 `failed`（或 500 信封），绝不能渲染成空报表；与中间件 fail-closed 一致 | `backend/internal/middleware/data_scope_middleware.go:117-131`、`:210-213`，`backend/pkg/database/scope.go:66-68`；既有测试族 `backend/internal/middleware/data_scope_failclosed_test.go` |
| TC-R-E-3 | P0 | 无 `platform:report:run` 权限 | `permission.denied` + 业务码 403，走 `FailWithCode` 的真实状态 | `backend/internal/middleware/casbin_middleware.go`，既有 `casbin_middleware_test.go` |
| TC-R-E-4 | P0 | 高敏动作缺 `X-Operation-Token` | `auth.operation.verification_required`；过期/不匹配各自键 | `backend/internal/middleware/secure_action_middleware_test.go` |
| TC-R-E-5 | P1 | 对已发布版本再次 `PUT` / 连续 `publish` 两次 | 第二次返回 `report.version.immutable`，不产生第二个版本行 | API 契约 §9.4 |
| TC-R-E-6 | P1 | 下载已过期 artifact | `report.artifact.expired`，不落新行 | DB 设计 §6 |
| TC-R-E-7 | P1 | 未实现声明格式（请求 `format = xlsx`） | 明确拒绝（不在 `supported_formats` 内），不是"降级为 CSV" | 主设计 §8.2 |
| TC-R-E-8 | P2 | 错误键形状回归 | 所有 `report.*` 键匹配 `^[a-z0-9_]+(?:\.[a-z0-9_]+)+$`，无连字符 | `backend/pkg/common/http/response.go:28`，参照 `backend/pkg/common/error_code_test.go` |

## 6. 并发与幂等用例

并发断言依赖既有夹具，不自造等待逻辑：`backend/pkg/testmysql` 的 `Open/OpenTB`
与 `AssertRowLocked`（`backend/pkg/testmysql/locking.go:18`，用 `SELECT ... FOR UPDATE NOWAIT` 探测）、
`AssertSerialisedWriters`（`:39`，证明同一父行的两个写者不可交错）。

| 编号 | 级别 | 场景 | 期望 | 落点 |
| :--- | :--- | :--- | :--- | :--- |
| TC-R-CON-1 | P0 | 同一 `idempotencyKey` 两个并发导出 | 只写一行 `report_artifact`；第二个请求复用产物 | `backend-tests`（`go test -race`） |
| TC-R-CON-2 | P0 | 同 key 并发下取数计数 | **两个并发只产出一次取数**（provider 调用计数 = 1）；API 契约 §9.2 的显式证明点 | `backend-tests` |
| TC-R-CON-3 | P0 | 同租户并发创建同名模板 code | 唯一键冲突时后到者收 409 语义，不产生重复行；删除后同名可复用（改名释放） | DB 设计 §5；既有夹具 `AssertSerialisedWriters` |
| TC-R-CON-4 | P1 | 并发 `publish` 同一 draft | 仅一次成功，其余 `report.version.immutable` | `backend-tests` |
| TC-R-CON-5 | P1 | 渲染中取消（`status = cancelled`） | 取消后不再产出 artifact；已产出 artifact 标记可清理 | `backend-tests` |
| TC-R-CON-6 | P1 | 保留期清理与渲染同时发生 | 清理任务靠 overlap 保护不重入；不删除正在写入的行 | `backend/pkg/maintenance/maintenance.go`，既有 `maintenance_test.go` |

## 7. 权限与租户安全用例（P0 集中区）

| 编号 | 场景 | 期望 | 依据 |
| :--- | :--- | :--- | :--- |
| TC-R-SEC-1 | `scope_binding = dept`，用户为 `dept_and_children` | 结果集只含本部门子树行；`scopeMode` 回显一致 | `backend/pkg/database/scope.go:38-57`、`data_scope_middleware.go:324`（`system_dept.ancestors` 展开由中间件完成，报表侧不复制） |
| TC-R-SEC-2 | 同 dataset、两个不同范围用户先后请求 | 第二个用户**不得**命中第一个用户的缓存结果 | 主设计 §10 约束 1（`scopeFingerprint` 不可省略） |
| TC-R-SEC-3 | 多角色用户（一个窄、一个宽） | 取最宽模式 | `data_scope_middleware.go:368` `resolveDataScopeMode` |
| TC-R-SEC-4 | `admin` / `tenant_admin` 请求 | 跳过 dept 策略但在自身租户内收窄 | `data_scope_middleware.go:455`、`:478` |
| TC-R-SEC-5 | 跨租户读模板/artifact | 只按 `tenant_id` + `visibility` 放行；他租户私有模板返回 not-found 语义而非"存在但拒绝" | [TENANT_CONTRACT_V1.en.md](../contracts/TENANT_CONTRACT_V1.md) §6 |
| TC-R-SEC-6 | `compat` 与 `multi` 双模各跑一遍主链路 | 行为一致；缺租户上下文时 `TENANT_CONTEXT_MISSING` fail-closed，不静默放大 | 合同 §3.2/§5；既有 `backend/internal/middleware/tenant_platform_scope_test.go` |
| TC-R-SEC-7 | 租户配额已超限 | 报表侧只展示，不拦截写入（配额为展示期语义） | 合同「配额展示期」条目 |
| TC-R-SEC-8 | 列表页是否兜底写权限 | 只有 `list` 的用户能看清单，不能运行/导出 | [DESIGN.md](../../DESIGN.md) §4.4 |
| TC-R-SEC-9 | hostile 浏览器直调报表 URL（越权参数） | 与既有 hostile 套件同形拒绝 | `frontend` 的 `test:smoke:tenant`（`tests/smoke-core/tenant-hostile-browser-matrix*.spec.ts`），CI job `smoke-tenant`（`.github/workflows/smoke-core.yml:138`） |

## 8. 迁移与保留期用例

| 编号 | 场景 | 期望 | 落点 |
| :--- | :--- | :--- | :--- |
| TC-R-DB-1 | 全新库 up 到报表迁移号 | 表/索引/字符集与 DB 设计 §2 一致（`ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`） | `backend/pkg/database/migrate_test.go` 所在包，`go test ./pkg/database/` |
| TC-R-DB-2 | down 形状 | down 只撤销本包新增对象，不触碰他域；`down` 的表名顺序与 `up` 逆序 | `backend/pkg/database/migrate_down_shape_test.go`、`migrate_down_test.go` 的既有断言风格 |
| TC-R-DB-3 | 唯一键不含 `deleted_at` | 软删后同 code 可复用；唯一索引定义与 `backend/pkg/database/migrations/000001_init_schema.up.sql:117-120` 的约定一致 | `migrate_preflight_test.go`（`backend/pkg/database/migrations/preflight/`） |
| TC-R-DB-4 | 保留期清理 | 分批 `LIMIT` 删除、不锁全表；`report.artifact_retention_days` / `report.run_retention_days` 生效；产物对象同步删除 | 复用 `RegisterMaintenanceTasks`（先例 `backend/modules/system/audit/audit_service.go:317`） |
| TC-R-DB-5 | 清理任务在 maintenance 关闭时 | 不产生"看起来会执行"的假承诺；文档与测试都写明 `PANTHEON_MAINTENANCE_ENABLED=false` 时的运维动作 | `backend/cmd/server/main.go` 的 runner env 门 |

## 9. 静态契约与文档门禁用例

| 编号 | 门禁 | 命令 | 断言点 |
| :--- | :--- | :--- | :--- |
| TC-R-CT-1 | 权限键形状 | `npm run check:harness-permission-contract`（`scripts/harness/check-permission-contract.mjs`） | 报表的 `Type: "F"` 动作节点全部绑动作键（`view/run/export/create/update/delete/clear`），无一个绑 `platform:report:list`——这正是该 checker 的唯一阻断级 finding（`:125-136`） |
| TC-R-CT-2 | DTO 形状 | `npm run check:backend-dto-contract` | handler 返回 `*Resp` 类命名，无裸 `map[string]any` |
| TC-R-CT-3 | 响应信封 | `npm run check:backend-response-contract` | 统一 `Success/Fail/FailWithCode`，无裸 `c.JSON` |
| TC-R-CT-4 | 文档 frontmatter | `npm run check:docs-frontmatter`（`scripts/frontmatter-check.mjs`） | 本包 8 份文档字段齐备、`linked_contracts` 指向存在文件 |
| TC-R-CT-5 | 文档链接 | `npm run check:harness-docs` | 无死链；新增文档已登记索引 |
| TC-R-CT-6 | 菜单契约 | `npm run check:menu-contract`（`frontend/scripts/check-menu-contract.mjs`） | 报表菜单节点 key/权限点/标题 key 齐备 |
| TC-R-CT-7 | i18n | `npm run check:i18n-hardcode`、`check:i18n-generated-scope` | 页面无硬编码；中英文两份同时登记 |
| TC-R-CT-8 | 编码与结构 | `npm run check:harness-encoding`、`check:structure`（CI job `docs-governance`，`.github/workflows/quality.yml:100`） | UTF-8 无 mojibake；文档放在 `docs/designs/` |
| TC-R-CT-9 | 后端 lint | CI job `go-lint`（`.github/workflows/ci.yml:249`） | 新增包不引入 lint 回归 |

## 10. 前端与 smoke 用例

| 编号 | 场景 | 落点 |
| :--- | :--- | :--- |
| TC-R-FE-1 | 报表中心菜单可达、无权限时页面拒绝进入 | `check:system-page-admission`（`frontend/package.json`） |
| TC-R-FE-2 | 无权限 / 空 / 加载 / 提交中 四态齐备 | [EMPTY_LOADING_ERROR_STATES.md](./EMPTY_LOADING_ERROR_STATES.md) 纪律 + 新增 spec |
| TC-R-FE-3 | 真实样板报表的列表/导出交互 | 新增 `frontend/tests/smoke/system/report-center.spec.ts`，并注册 `test:smoke:system:report` 脚本 |
| TC-R-FE-4 | 导出截断提示可见（`truncated`） | 同上新增断言 |
| TC-R-FE-5 | 前端单测（呈现层纯函数：列映射、参数摘要） | `npm run test:unit`（vitest，`frontend/package.json`） |

**新增 spec 的强制登记义务**：`frontend/scripts/check-smoke-coverage-contract.mjs` 要求
`tests/smoke/**/*.spec.ts` 每个文件都被某个 `test:smoke:*` 叶子脚本引用，且映射写入
`frontend/tests/smoke/README.md`（该脚本 `:85`、`:91`、`:96` 三处 finding）。
因此 TC-R-FE-3 的落地必须是"spec + package.json 脚本 + smoke README 行"三处同一 PR 修改，
否则 `check:smoke-coverage-contract`（在 `prebuild` 链内）直接失败。

## 11. 运行时缺口（本计划不证明的东西）

显式声明，避免"文档看起来覆盖了其实没覆盖"：

| 缺口 | 现状 | 需要的动作 | 决策人 |
| :--- | :--- | :--- | :--- |
| 大报表性能 | 仓库只有 **advisory** 的基准 job（`bench-perf-smoke`，`.github/workflows/ci.yml:310`，单迭代、且只跑审计侧 `BenchmarkAuditServiceListOperationLogs`），没有报表容量基线 | 需要一次真实数据量（≥10 万行）的人工压测并记录证据；本计划不声称已达标 | 维护者 |
| 同步导出与 `WriteTimeout` 的实测余量 | 只有配置事实，没有测量值 | 压测时一并测量；超限即转异步的阈值需实测校准 | 维护者 |
| 多实例缓存失效广播 | 第一阶段允许靠 TTL 收敛（主设计 §10 约束 3） | 若要实时失效，需要接入既有 watcher 方向并补跨实例测试 | 维护者 |
| Redis 不可用时的行为 | 生产或 `PANTHEON_REDIS_REQUIRED=true` 时 fail-fast（`backend/pkg/database/redis.go`，测试 `redis_required_test.go`） | 报表侧不自建降级逻辑；实现时只需确认沿用该前提 | — |
| 覆盖率 | CI 阈值来自 `vars.COVERAGE_THRESHOLD`（默认 50，`coverage-gate`，`.github/workflows/ci.yml:396`） | 新增包需证明不拉低该门；不在文档里承诺具体数字 | — |

## 12. 执行清单（实现 PR 自检）

```bash
# 后端（CI job backend-tests 等价，本地需 MySQL 8 + Redis 与测试 env）
cd backend && go test -race ./...
cd backend && go test ./pkg/database/ ./pkg/impexp/ ./pkg/maintenance/ ./internal/middleware/

# 静态契约门禁
npm run check:docs-frontmatter
npm run check:harness-docs
npm run check:harness-permission-contract
npm run check:backend-dto-contract
npm run check:backend-response-contract

# 前端
cd frontend && npm run test:unit
cd frontend && npm run check:menu-contract && npm run check:i18n-hardcode \
  && npm run check:system-datetime-presentation && npm run check:smoke-coverage-contract

# smoke（需本地服务，脚本来自 frontend/package.json）
cd frontend && npm run test:smoke:core
cd frontend && npm run test:smoke:tenant
```

MySQL/Redis 测试夹具由 `backend/pkg/testmysql` / `backend/pkg/testredis` 读取；CI 中相关 env 为
`PANTHEON_TEST_DSN` / `PANTHEON_TEST_DSN_REQUIRED` / `PANTHEON_TEST_REDIS_ADDR` / `PANTHEON_TEST_REDIS_REQUIRED`
（见 `.github/workflows/quality.yml:379` 的 `backend-tests` job），缺 DSN 时该门设计为**失败而非静默跳过**。

## 13. 豁免流程

P0 用例若需豁免，必须在实现 PR 里同时写明：豁免编号、原因、补偿证据（人工执行记录或 runbook 链接）、
复审日期。缺任一信息视为未豁免。豁免记录落在 `.harness/evidence/`（先例见 `coverage-gate` 引用的
`.harness/evidence/2026-09-10-tenant-ready-guardrails/`，见 `.github/workflows/ci.yml` 的阈值注释）。

## 14. 开放决策 / Open decisions

| 编号 | 决策 | 选项 | 决策人 |
| :--- | :--- | :--- | :--- |
| OD-TEST-1 | 报表 smoke 是否进入核心套件（`test:smoke:core` / CI `pr-smoke-core`） | A：不进核心，独立 `test:smoke:system:report`；B：进核心 | 维护者（倾向 A，核心套件当前固定 9 个 spec，见 `frontend/package.json`） |
| OD-TEST-2 | 大报表压测证据形态 | A：一次性人工记录进 `.harness/evidence/`；B：新增 advisory bench job | 维护者 |
| OD-TEST-3 | 并发证明是否必须依赖真实 MySQL 夹具 | A：用 `testmysql`（更接近真实锁语义）；B：用内存替身（快但证明力弱） | 维护者（倾向 A，P0 用例不接受 B） |

## 15. 关联文档

| 文档 | 中文 | 英文 |
| :--- | :--- | :--- |
| 主设计 | [REPORT_CENTER_DESIGN.md](./REPORT_CENTER_DESIGN.md) | [REPORT_CENTER_DESIGN.en.md](./REPORT_CENTER_DESIGN.en.md) |
| 数据模型与迁移 | [REPORT_CENTER_DB_DESIGN.md](./REPORT_CENTER_DB_DESIGN.md) | [REPORT_CENTER_DB_DESIGN.en.md](./REPORT_CENTER_DB_DESIGN.en.md) |
| 接口契约 | [REPORT_CENTER_API_CONTRACT.md](./REPORT_CENTER_API_CONTRACT.md) | [REPORT_CENTER_API_CONTRACT.en.md](./REPORT_CENTER_API_CONTRACT.en.md) |
| 测试计划（本文） | `REPORT_CENTER_TEST_PLAN.md` | [REPORT_CENTER_TEST_PLAN.en.md](./REPORT_CENTER_TEST_PLAN.en.md) |

调度侧配套：SCHEDULER_CENTER_TEST_PLAN.md（planned/unavailable：`./SCHEDULER_CENTER_TEST_PLAN.md`；无当前交付承诺）。
数据库约定：[DATABASE.md](./DATABASE.md)。目录索引：[README.md](./README.md)。

## 变更日志

| 日期 | 变更 |
| :--- | :--- |
| 2026-11-04 | 建立验收标准到用例的映射、四类用例、权限与租户安全用例、迁移与保留期用例、静态门禁清单、运行时缺口与豁免流程 |
