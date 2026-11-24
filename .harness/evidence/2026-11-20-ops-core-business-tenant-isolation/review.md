# T04 Review — 核心业务资源租户归属与兼容迁移

## 结论

`changes requested`（就完整验收而言）。schema 迁移部分已完成并验证；运行时租户隔离被 Base `keyMatch2` 上游缺口阻塞，其余 live 核对未执行，本包不能整体关闭。

## Findings

1. **P1 blocked — Base `keyMatch2` 阻止详情路由安全绑定。**
   `GET /packages/:id` 同时匹配 `GET /packages/export`，绑定详情会放宽导出权限（anti-widening）。因此详情路由保持未绑定，`tenant.Scoper` 路由化无法完成；需 Base 正式 release 提供静态路由守卫，不在 Ops override。

2. **P2 — fixed（2026-11-24）— compat/multi live 升级、回滚核对。**
   新增 `TestTenantOwnershipMigrationPreservesLegacyRowsAndRecoversPartialState`（真实 MySQL PASS 0.42s）：预置旧行后迁移，行数/名称不变且 `tenant_id=0`；模拟部分应用（某表已加列）后幂等恢复成功；down 脚本按合同为 no-op，验证不破坏 schema/数据。真实回滚仍以 pre-T04 备份恢复为准。

3. **P2 — 异步任务原租户保持未验证。**

4. **info — 迁移幂等 DDL 模式正确。**
   golang-migrate 逐语句执行；48 条动态 DDL 用 `SET @sql := IF(...)` 保护，重复执行安全；业务源与装配副本字节一致。

## Machine Readable

```json
{
  "taskId": "2026-11-20-ops-core-business-tenant-isolation",
  "reviewer": "primary agent self-review (no independent review on this candidate)",
  "verdict": "changes requested",
  "decision": "changes requested",
  "scope": "core business tenant_id schema migration and legacy key rebuild",
  "taskManifest": ".harness/tasks/2026-11-20-ops-core-business-tenant-isolation/manifest.json",
  "evidenceDir": ".harness/evidence/2026-11-20-ops-core-business-tenant-isolation",
  "summaryFile": ".harness/evidence/2026-11-20-ops-core-business-tenant-isolation/summary.md",
  "reviewFile": ".harness/evidence/2026-11-20-ops-core-business-tenant-isolation/review.md",
  "changeRef": "none (uncommitted worktree)",
  "findings": [
    {"id": "T04-R1", "severity": "P1", "status": "blocked-upstream", "summary": "Base keyMatch2 prevents binding detail routes; runtime tenant isolation cannot land in Ops"},
    {"id": "T04-R2", "severity": "P2", "status": "fixed", "summary": "compat/multi live upgrade and rollback verified", "verification": "TestTenantOwnershipMigrationPreservesLegacyRowsAndRecoversPartialState PASS 0.42s on real MySQL: legacy rows retained with tenant_id=0, partial-state resume succeeds, down script no-op preserves schema/rows"},
    {"id": "T04-R3", "severity": "P2", "status": "open", "summary": "async task original-tenant retention not verified"},
    {"id": "T04-R4", "severity": "info", "status": "verified", "summary": "idempotent 48-statement dynamic DDL; up/down mirrored byte-identically; tenant-first key test PASS"}
  ],
  "verification": {
    "migrationTest": "PASS 0.635s",
    "migrationMirror": "cmp identical (up and down)",
    "idempotentStatements": 48
  },
  "linkage": {
    "taskManifest": ".harness/tasks/2026-11-20-ops-core-business-tenant-isolation/manifest.json",
    "taskPacket": "docs/harness/tasks/2026-11-20-ops-core-business-tenant-isolation.task.md",
    "evidence": ".harness/evidence/2026-11-20-ops-core-business-tenant-isolation/commands.json",
    "reviewFile": ".harness/evidence/2026-11-20-ops-core-business-tenant-isolation/review.md",
    "changeRef": "none",
    "planRefs": [".harness/tasks/2026-11-20-ops-review-remediation/plan.md"]
  }
}
```
