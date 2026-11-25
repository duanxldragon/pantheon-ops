# 扩展业务与异步链路租户隔离

- Status: in-progress
- Updated: 2026-11-25
- Baseline: a2d69d4a759b5df2b26aaa9d26bdc5175babe9dc
- Owner: 扩展业务租户迁移负责人
- Priority: P1
- Activation: core
- Dependencies: T03, T04

## Phase 1 delivered 2026-11-25

迁移 `000043` 给 18 张扩展表加 `tenant_id`，并把 5 个自然唯一键重建为租户优先；18 个模型同步。compat 数据不变。迁移测试在本机未对真实库运行，交 CI 证明。详见 `../../evidence/2026-11-20-ops-extension-and-async-tenant-isolation/summary.md`。

## Findings

- T01 清单有误：真实是 18 张表，不含 `biz_preflight_control`（测试合成表），漏了 `biz_application` 与 `obs_alert_*`、`obs_notification_*`。
- T04 只交付 schema：业务代码未使用 `tenant.Scoper`/`TenantOwnerID`，除 observability 外读写均未按租户强制；核心与扩展模块同样缺口。
- 告警评估器跳过 `TenantID != 0` 的数据源。
- T05 的 `000042` 此前未登记进 `businessMigrations`（已修）；`deploy_credential.go:59` 直接引用 `bizscope`，违反模块边界（未修，单独跟进）。

## Next Atomic Action

第二阶段已由维护者于 2026-11-25 决定：不并入 T14，拆为按模块的独立任务，见 [2026-11-25-ops-tenant-runtime-enforcement](../2026-11-25-ops-tenant-runtime-enforcement/PLAN.md)。先决条件是 CI 的 backend-tests 补 MySQL 服务。迁移 `000043` 已在真实 MySQL 8 上证明（5/5，含负例对照）。

## Known Gaps

- 第二阶段（运行时强制）未做，租户隔离尚未真正生效。
- CI 不运行真实库测试（backend-tests 无 MySQL）；迁移已在本机真实库证明。
- 无独立 review。
