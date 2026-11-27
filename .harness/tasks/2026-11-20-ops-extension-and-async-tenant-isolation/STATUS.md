# 扩展业务租户归属 Schema（T14 第一阶段）

- Status: completed
- Updated: 2026-11-26
- Baseline: a2d69d4a759b5df2b26aaa9d26bdc5175babe9dc
- Owner: 扩展业务租户迁移负责人
- Priority: P1
- Activation: core
- Dependencies: T03, T04

## Phase 1 completed 2026-11-26

迁移 `000043` 给 18 张扩展表加 `tenant_id`，并把 5 个自然唯一键重建为租户优先；18 个模型同步。compat 数据不变。真实 MySQL migration 测试 5/5 PASS，PR #225 的 MySQL-backed Backend Tests 通过。独立 CODE/ARCH review 无发现。详见 `../../evidence/2026-11-20-ops-extension-and-async-tenant-isolation/summary.md`。

## Findings

- T01 清单有误：真实是 18 张表，不含 `biz_preflight_control`（测试合成表），漏了 `biz_application` 与 `obs_alert_*`、`obs_notification_*`。
- T04 只交付 schema：业务代码未使用 `tenant.Scoper`/`TenantOwnerID`，除 observability 外读写均未按租户强制；核心与扩展模块同样缺口。
- 告警评估器跳过 `TenantID != 0` 的数据源。
- T05 的 `000042` 此前未登记进 `businessMigrations`（已修）；`deploy_credential.go:59` 直接引用 `bizscope`，违反模块边界（未修，单独跟进）。

## Next Atomic Action

第二阶段已由维护者于 2026-11-25 决定：不并入 T14，拆为按模块的独立任务，见 [2026-11-25-ops-tenant-runtime-enforcement](../2026-11-25-ops-tenant-runtime-enforcement/PLAN.md)。先决条件是 CI 的 backend-tests 补 MySQL 服务。迁移 `000043` 已在真实 MySQL 8 上证明（5/5，含负例对照）。

## Known Gaps

- 第二阶段（运行时强制）未做，租户隔离尚未真正生效。
- 没有运行时租户读写隔离、异步 tenant identity 或双租户业务执行证据；这些均明确不在第一阶段交付范围。
- 这些 gap 已转至 `2026-11-25-ops-tenant-runtime-enforcement` 父计划与逐模块任务；本任务 completed 仅代表 schema 第一阶段，不代表租户隔离已在运行时生效。
