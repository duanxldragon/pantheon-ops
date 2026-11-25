# 核心业务资源租户归属与兼容迁移

- Status: completed — schema 迁移、兼容恢复与异步恢复路径已验证；残余路由绑定转入 `2026-11-25-ops-tenant-detail-route-binding`；维护者最终验收仍开放
- Updated: 2026-11-25
- Baseline: a2d69d4a759b5df2b26aaa9d26bdc5175babe9dc
- Owner: 核心业务租户迁移负责人
- Priority: P1
- Activation: core
- Dependencies: T01

## 2026-11-25 收口

按维护者指示关为本包 completed。已交付且验证：schema 迁移与装配副本一致、compat 旧行保留与部分应用恢复、双租户同名/单租户重名约束、异步重启后的租户保持（`TestAsyncRecoveryKeepsOriginatingTenant`，真实 MySQL `-race`）。

残余不在本包内继续，已拆出：

- `2026-11-25-ops-tenant-detail-route-binding`（P1，planned）：消费含静态路由守卫的正式 foundation release 后恢复详情路由租户绑定。
- 写入路径（claim 时落 `tenant_id`）与 `tenant_mode=multi` 端到端归 T14 `2026-11-20-ops-extension-and-async-tenant-isolation` 与 `2026-11-25-ops-residual-live-acceptance`。

本包 completed 只表示范围内交付完成，不代表生产或多租户验收。

## Implemented

- `000041_core_business_tenant_ownership.{up,down}.sql`：48 条 `SET @sql := IF(...); PREPARE; EXECUTE; DEALLOCATE` 幂等 DDL，为既有业务表加 `tenant_id` 并把全局 name 唯一键重建为 tenant-first。
- 业务模块源文件与 `backend/pkg/database/migrations/` 装配副本 up/down 字节一致（`cmp` 验证）。

## Verification

- `TestTenantOwnershipMigrationUpgradesLegacyNameKeys` PASS（0.635s，真实 MySQL）：旧库升级后双租户可复用同名、单租户内重名仍拒绝。
- 证据：`../evidence/2026-11-20-ops-core-business-tenant-isolation/`。

## Known Gaps

1. **Base 上游阻塞**：`keyMatch2` 使 `GET /packages/:id` 同时匹配 `/packages/export`，详情路由保持未绑定（anti-widening），运行时租户隔离需 Base 正式 release 的静态路由守卫。
2. ~~compat/multi 真实库升级、失败恢复与回滚行数/归属核对未执行。~~ 2026-11-24 已补：`TestTenantOwnershipMigrationPreservesLegacyRowsAndRecoversPartialState`（真实 MySQL PASS 0.42s）验证旧行保留、部分应用恢复、down no-op 保持数据；真实回滚仍为 pre-T04 备份恢复。
3. 异步任务重启后保持原租户：恢复/重载路径已补 `TestAsyncRecoveryKeepsOriginatingTenant`（真实 MySQL `-race` PASS）；写入路径（claim 时落 `deploy_task.tenant_id`）与 multi 运行环境归 T05，未做端到端证明。

## 2026-11-24 剩余项归因（为何仍 in-progress）

迁移与兼容恢复已由真实 MySQL 测试覆盖（旧行保留、部分应用恢复、down no-op）。剩余两条不是本包单独可关闭的：

1. **详情路由绑定**：`GET /packages/:id` 与 `/packages/export` 的静态/参数同层冲突需要 `pantheon-base` 正式 foundation release 携带静态路由守卫（Base PR #525 `63940437` 已合并但未发布），ops 不能手抄或绕过 release 流程。
2. **异步任务重启保持原租户**：恢复路径已由 `TestAsyncRecoveryKeepsOriginatingTenant` 固定——重启后的新 worker 重载任务行，执行器收到的 `Task.TenantID` 与库中原租户一致。剩余是**写入路径**（业务写路径显式 `TenantOwnerID()` 落库）与 `tenant_mode=multi` 真实验收，二者属于 T05 `2026-11-20-ops-extension-and-async-tenant-isolation`（当前 planned）；当前验收库未配置 V4 多租户，故不做端到端声明。

因此 T04 保持 in-progress，不使用未验证声明收口。

## Next Atomic Action

等 Base 发布静态路由守卫后恢复详情路由绑定；迁移与异步恢复路径已覆盖，剩余仅在 T05 完成写入路径后做 multi 端到端验收。
