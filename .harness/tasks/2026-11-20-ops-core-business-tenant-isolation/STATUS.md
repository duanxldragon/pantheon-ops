# 核心业务资源租户归属与兼容迁移

- Status: in-progress
- Updated: 2026-11-24
- Baseline: a2d69d4a759b5df2b26aaa9d26bdc5175babe9dc
- Owner: 核心业务租户迁移负责人
- Priority: P1
- Activation: core
- Dependencies: T01

## Implemented

- `000041_core_business_tenant_ownership.{up,down}.sql`：48 条 `SET @sql := IF(...); PREPARE; EXECUTE; DEALLOCATE` 幂等 DDL，为既有业务表加 `tenant_id` 并把全局 name 唯一键重建为 tenant-first。
- 业务模块源文件与 `backend/pkg/database/migrations/` 装配副本 up/down 字节一致（`cmp` 验证）。

## Verification

- `TestTenantOwnershipMigrationUpgradesLegacyNameKeys` PASS（0.635s，真实 MySQL）：旧库升级后双租户可复用同名、单租户内重名仍拒绝。
- 证据：`../evidence/2026-11-20-ops-core-business-tenant-isolation/`。

## Known Gaps

1. **Base 上游阻塞**：`keyMatch2` 使 `GET /packages/:id` 同时匹配 `/packages/export`，详情路由保持未绑定（anti-widening），运行时租户隔离需 Base 正式 release 的静态路由守卫。
2. ~~compat/multi 真实库升级、失败恢复与回滚行数/归属核对未执行。~~ 2026-11-24 已补：`TestTenantOwnershipMigrationPreservesLegacyRowsAndRecoversPartialState`（真实 MySQL PASS 0.42s）验证旧行保留、部分应用恢复、down no-op 保持数据；真实回滚仍为 pre-T04 备份恢复。
3. 异步任务重启后保持原租户未端到端验证。

## Next Atomic Action

等 Base 发布静态路由守卫后恢复详情路由绑定；在此之前先在隔离库补 compat/multi 升级与回滚核对。
