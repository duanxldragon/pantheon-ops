---
title: Pantheon Base 租户演进任务分解与开发计划
doc_type: Design
layer: platform
status: Active
updated_at: 2026-09-15
linked_contracts:
  - docs/contracts/TENANT_CONTRACT_V1.md
---

# Pantheon Base 租户演进任务分解与开发计划

> **2026-09-15 执行对账**：队列 5 的 S3 下载授权剩余项已收口（`ServeUploadedFile` 现以授权路由服务 s3 驱动：`upload.Service.OpenS3Object` + `EnforceTenantObjectScope`，租户判定仅来自已解析上下文，跨租户在触达对象存储前拒绝，compat 保持旧键位；6 项隔离测试，DB-backed `go test -short` 37 包全绿，evidence 见 `s3-download-authorization.md`）。
> durable job 框架确认不存在于代码库（仅 operation-log 进程内异步队列，已带显式 `TenantID` 持久化测试）；合同 §3.2 已约束未来 worker 必须携带 `tenant_id`，记录为 future work 而非隔离缺口。
> 队列 4/5 剩余仅 runtime/生产项（hostile 双租户浏览器矩阵扩展、G1--G4 人工门禁），随队列 6 收口。
>
> **2026-09-11 执行对账**：队列 0--3 已完成（manifest 均为 `done`，evidence 齐全）——
> 任务 0 `tenant-ready-guardrails`（伪能力下线 + 就绪检查 + CI DB-backed 覆盖率对账，阈值 11→50，维护者已批准口径）；
> 任务 1 `tenant-contract-design`（[合同 V1 冻结](../../../contracts/TENANT_CONTRACT_V1.md)）；
> 任务 2 `tenant-migration-runbook`（[Runbook](../../../runbooks/TENANT_MIGRATION_RUNBOOK.md) + 副本三类演练通过，生产执行待 G1--G4）；
> 任务 3 `tenant-canary-slice`（dict 资源垂直切片，flag `platform.tenant_mode` 默认 `compat`，10 项双租户 hostile tests 全绿，flag-off 回归通过）。
> 下一步为队列 4--6（core-auth-iam / core-data-infrastructure / verification-and-gray），开工前提：canary 隔离 evidence 的维护者 review（"canary 放量" gate）。
>
> **2026-09-13 队列 4 slice 3 + 队列 5 slices 4-5 验证收口**：前端租户 picker（密码验证后展示候选、显式 `tenantId` 重试、MFA 挑战租户绑定与分歧拒绝、5 locale 文案）+ auth 日志租户列（迁移 000015/000016，additive 带索引）+ dashboard 聚合/上传下载授权/operation-log 异步租户持久化/lowcode 路由租户护栏；
> Playwright picker E2E 3/3 与本地 MySQL 双租户 HTTP 探针通过（见 queue-6 evidence）。收口复验时发现并修复 3 处 `WithTenantContext` 浅拷贝锁复制（`Runtime`/`LoginService`/`security.Service`，vet lock-copy），修复后 build/vet/`go test -short ./...`/type-check/lint 全绿。
> 队列 4/5 剩余均为 runtime/生产项：hostile 双租户浏览器矩阵扩展、flag 回滚运行态证据、S3 下载授权（当前仅本地存储）、G1--G4 人工门禁；详见各 manifest statusNote。
>
> **2026-09-12 队列 5 开工（audit + upload + settings 切片）**：`system_log_oper.tenant_id`（additive，索引，默认 0）+ 解析上下文戳记（不信请求字段）+
> list/detail/export/batch-delete 的租户硬过滤 + `tenantIdFilter` 仅平台层可用；上传对象 key 多租户模式加 `t<tenantId>/` 前缀（compat 布局不变）；
> `system_setting` 租户 override/继承（迁移 000014 换组合唯一键 `(tenant_id, setting_key)`，override 优先、租户写生成 override 副本不改全局行、缓存按租户命名空间隔离）；
> 21 项隔离测试 DB-backed 全绿，compat 回归通过。剩余：security-event/login-log 列、dashboard/异步上下文、下载授权、生成器护栏。
>
> **2026-09-12 队列 4 slice 2 完成**：多 membership 用户登录租户选择（`LoginReq.TenantId` 显式选择走同一 membership 门禁、错租户拒绝、歧义消解）+ `GET /auth/login-tenants` 候选列表端点 + MFA 挑战租户值透传；
> 租户 scoped Casbin policy 写入面打通（create/update policy 增加 `tenantId`，服务端派生 `role:<key>@tenant:<id>` domain subject，per-tenant 命名空间唯一性校验），middleware domain 6 项隔离测试 + permission 4 项测试 + auth-gate 扩至 18 项全绿；
> 前端 picker UI 留 UX gate（flag 仍 `compat`，UI 惰性），其余队列 4 后端范围收口。
>
> **2026-09-11 队列 4 开工（slice 1 完成）**：维护者已消费 "canary 放量" gate（canary review 结论为 Approved，evidence 已被确认）。
> `tenant-core-auth-iam` slice 1 已交付并验证：session/token 租户 claim（`SessionData.TenantID` + `system_user_session.tenant_id`，additive 迁移）、
> 多租户登录 membership 发现与签发门禁（单 membership 确定解析 / 无 membership 落平台层 / 歧义拒绝）、refresh 时 membership 复检（会话不过期于 membership）、
> Casbin domain subject 展开（global 优先 → `role:<key>@tenant:<id>`，contract §4）、membership 变更会话吊销 helper；
> 12 项 DB-backed 双租户 hostile tests 全绿 + flag-off 回归通过，evidence 见 `.harness/evidence/2026-09-10-tenant-core-auth-iam/`。
> 队列 4 剩余：多 membership 用户的登录租户选择 UI（需 UX gate）、租户 scoped policy 写入面；队列 5/6 未开工。

## 目标与边界

本计划把 `docs/reviews/PANTHEON_BASE_INDEPENDENT_AUDIT_PLAN_20260910.md` 的结论拆成可回退的 task packet。当前系统继续按单租户模式运行；本计划本身不批准 schema、认证、权限或生产迁移改动。

交叉复核记录：`docs/reviews/PANTHEON_BASE_DESIGN_SUPPLEMENT_AUDIT_20260910.md`（2026-09-10）独立核实了本计划的租户零实现、伪能力和覆盖率三项前提，结论一致；其中「覆盖率对账」一项的归因已按复核结果修正（见下方"当前判断"末条）。该审计文档同时记录了一项不属租户范围但需在 1.0 冻结前处理的缺口：`permission_workbench.go:548` 的 API 权限映射仅 12 条而全库 200 条 route。

多租户的目标是“可验证的共享库/共享 schema MVP”，而不是一次性支持套餐、计费、域名发现、独立数据库和完整品牌中心。所有实现都必须保留单租户兼容模式，并通过双租户隔离证据后才能扩大范围。

## 当前判断

- 当前底座没有已实现的 Tenant master、membership、tenant middleware、tenant-aware Casbin 或 tenant migration。
- `MULTI_TENANT_DESIGN.md` 中的 16 小时估算不能作为实施承诺；从护栏到生产验证的合理总量为约 6--10 周（单工程师、中等置信度）。
- 生成器暴露的 `tenant` 选项是伪能力，必须在运行态实现之前移除或明确禁用。触点共 8 处：`backend/internal/scaffold/contract.go:73`、`frontend/src/modules/lowcode/generator/schema.ts:25`、`ModuleWizard.tsx:1737-1738`，以及 5 个 locale 的 `generator.wizard.dataScopeMode.tenant` 键。
- 现有 `2026-09-08-p1-test-coverage-phase1` 的 evidence 声称阶段目标已达成（`test-summary.md:32`：auth 70.8% / iam 55.7--64.9% / audit 63.9% / 整体 57.4%），但 manifest 仍为 `in-progress`（`manifest.json:5`）且 CI 阈值未提升（backend 11% / frontend 0%，自 v0.9.0 引入后未变）；在进入 auth/IAM 改造前必须完成状态对账。
- **对账的根因是两条测量线之间没有桥，不是状态字段填错。** evidence 的达成数字建立在本地 MySQL + Redis 环境（`test-summary.md:133` 自述），而 CI `unit-tests` job（`ci.yml:103-124`）无 MySQL/Redis service、无 `PANTHEON_TEST_DSN`，测的是 DSN-less 路径——实测该路径下 auth/login 8.9%、auth/security 6.0%、iam/permission 5.5%、iam/user 10.8%、audit 26.5%，与原始基线逐行吻合，即 CI 可见的覆盖率从未改善。因此对账动作是**补 CI 的 DB 环境让门禁测到真实数字，再按实测值设定阈值**，不是把 manifest 状态改成 done。在两条线打通前，队列 4 声明的 `blockedBy: 2026-09-08-p1-test-coverage-phase1` 是空转依赖，挡不住任何东西。

## 执行队列

| 顺序 | Task ID | 交付 | 依赖 | 估算 | 是否可改变运行态 |
| --- | --- | --- | --- | --- | --- |
| 0 | `2026-09-10-tenant-ready-guardrails` | 伪能力下线（8 处触点）、租户就绪检查、资源/唯一键登记模板、覆盖率测量口径对账 | 审计结论 | 1--3 日 | 否，单租户回归；CI 门禁配置变更需 gate |
| 1 | `2026-09-10-tenant-contract-design` | Tenant、membership、context、scope、Casbin、兼容模式合同 | 队列 0 | 1--3 日 | 否 |
| 2 | `2026-09-10-tenant-migration-runbook` | 表/索引/缓存/文件/审计盘点、回填与回滚演练方案 | 队列 1 | 2--4 日 | 否，副本演练 |
| 3 | `2026-09-10-tenant-canary-slice` | 一个真实低风险 CRUD 垂直切片及双租户隔离证据 | 队列 0--2 | 1--2 周 | 是，必须 feature flag/可回退 |
| 4 | `2026-09-10-tenant-core-auth-iam` | membership、token/session/MFA、角色与 Casbin 域隔离 | 队列 3；覆盖率对账 | 2--3 周 | 是，高风险 |
| 5 | `2026-09-10-tenant-core-data-infrastructure` | 配置/字典/审计/导出/上传/缓存/动态模块边界 | 队列 4 | 1--2 周 | 是，高风险 |
| 6 | `2026-09-10-tenant-verification-and-gray` | 双租户 hostile smoke、迁移/回滚/性能/观测、灰度结论 | 队列 4--5 | 1--2 周 | 是，生产 gate |

## 依赖图与停止条件

```text
审计报告
   |
   v
tenant-ready-guardrails --> tenant-contract-design --> tenant-migration-runbook
                                      |                         |
                                      +-----------> tenant-canary-slice
                                                       |
                                                       v
                                        tenant-core-auth-iam
                                                       |
                                                       v
                                  tenant-core-data-infrastructure
                                                       |
                                                       v
                                  tenant-verification-and-gray
```

必须停止而不是继续堆临时分支的情况：

1. Tenant master、用户 membership、可信 tenant resolution、平台/租户资源 scope 中任一项未冻结。
2. 迁移副本无法完成默认租户回填、唯一键冲突处理或回滚。
3. canary 出现跨租户读取/写入、越权 ID、伪造 Header/JWT、批量/导出/统计泄漏、缓存/文件串租户。
4. auth/IAM 任务无法同时给出单租户回归、双租户隔离和运行态证据。
5. 任何“已支持”能力只在设计文档存在、但 API、向导或 runtime 尚未完成。

## 分阶段开发计划

### Phase 0：护栏与事实对账

先完成任务 0；同时把 coverage phase 1 的 manifest 状态、CI 阈值和 evidence 对齐。该阶段不能引入 `tenant_id`、token claim 或权限分支。

覆盖率对账的具体口径与停止条件见 `2026-09-10-tenant-ready-guardrails/task.md` 的 Implementation Notes 与 Stop Points：若无法在 CI 内取得可复现的 DB-backed 读数，该项转为独立任务并记录 gap，**不得用本地读数替代门禁读数放行队列 4**。CI 测量方案与新阈值取值属门禁政策变更，需维护者 gate。

### Phase 1：合同与迁移准备

按任务 1、2 顺序产出合同和 runbook。合同必须冻结：用户是否多租户、登录发现方式、平台管理员边界、资源 scope 矩阵、Casbin domain、兼容模式期限和唯一键策略。runbook 必须能在数据库副本完成回填、冲突、失败、恢复和回滚演练。

### Phase 2：可回退 canary

任务 3 只选择一个真实低风险资源；默认候选为 `system/config` 下、且经 scope 矩阵确认可租户化的 CRUD 资源。不得同时改全仓系统表、域名解析、套餐计费或第二种 tenant resolution。canary 必须覆盖 list/detail/create/update/delete、批量、导出、审计、缓存/文件边界和两个租户的交叉访问。

### Phase 3：核心隔离

任务 4 先做身份、membership、session/token/MFA 与 IAM/Casbin 语义；任务 5 再做设置、字典、审计、导出、上传、缓存、dashboard、后台任务和动态模块。按垂直切片提交，每个切片都有 migration、contract test、cross-tenant test、rollback note 和 runtime evidence。

### Phase 4：验证与灰度

任务 6 在副本和预发布环境执行 hostile smoke、ID/claim/header 篡改、批量/导出/统计、并发、缓存/文件/session 隔离、性能基线、备份恢复和回滚演练。只有全部证据齐全，才能称为“受控生产候选”；否则保持“单租户兼容/受控试点”。

## 风险与控制

| 风险 | 控制 |
| --- | --- |
| 一次性给全表加 `tenant_id` 导致大 diff/不可回退 | 先合同、runbook、canary，再按切片迁移 |
| 把部门数据权限当租户隔离 | Tenant filter 作为独立授权维度，与 DataScope 明确组合顺序 |
| Header/子域名伪造租户上下文 | Phase 1 只采用合同批准的可信来源，默认拒绝缺失/冲突 context |
| 全局唯一索引改造失败 | 先做重复值盘点、冲突策略和副本演练 |
| 伪能力继续扩散 | 任务 0 加生成器/DDL/模块验收护栏，未实现能力不出现在向导 |
| 测试覆盖不足掩盖串租户 | auth/IAM/审计高风险包先完成 coverage 对账和双租户 hostile tests |
| 覆盖率"已达成"结论在 CI 不可复现，形成虚假门禁 | 任务 0 打通 CI 的 DB-backed 测量；阈值按门禁实测值设定，禁止用本地读数放行队列 4 |

## 统一完成门禁

- 每个任务有 `.harness/tasks/<task-id>/task.md`、`manifest.json` 和 evidence 链接。
- 所有 runtime-sensitive 任务都有运行态证据；无法运行时明确记录 gap，不得用静态检查替代。
- schema、迁移、权限、审计、认证和跨仓继承变更必须经过 human gate。
- Base 共享合同如影响 `pantheon-ops`，必须另开 foundation release/consumer lock 任务；本计划不直接修改 ops。
- 生产就绪前，单租户回归、双租户隔离、越权、导出、统计、缓存、文件、异步、迁移恢复和性能证据必须全部存在。

## 预估与决策点

- 只做任务 0：1--3 个工作日（含覆盖率测量口径对账），可支撑近期单租户开发并降低未来返工。
- 做到任务 2：约 3--7 个工作日，得到可评审的租户合同与迁移决策，但仍不是多租户运行态。
- 做完任务 3：约 2--4 周，得到受控试点，可据此重新估算核心实现。
- 做完任务 4--6：约 6--10 周，才有资格评估生产候选。

维护者只需要在四个节点做 gate：合同冻结、迁移 runbook 批准、canary 放量、最终灰度/生产批准。其余执行由任务包和 evidence 驱动。
