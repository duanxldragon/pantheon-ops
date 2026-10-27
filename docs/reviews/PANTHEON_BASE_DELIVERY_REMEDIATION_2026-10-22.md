---
title: 交付可交付性独立验收结论与 v0.17.1 整改记录（2026-10-22）
doc_type: Remediation
layer: platform
depends_on_layers:
  - system/iam
status: Active
linked_contracts:
  - docs/contracts/TENANT_CONTRACT_V1.md
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-10-22
---

# 交付可交付性独立验收结论与 v0.17.1 整改记录（2026-10-22）

## 0. 结论

独立验收针对的问题是「当前最新代码是否已到可冻结基线平台、把精力转向 `pantheon-ops` 的阶段」，
不是功能开发评审，也不是通用代码走查。两个判定对象分别给出结论：

| 判定对象 | 基线 | 结论 | 说明 |
|---|---|---|---|
| **A. 最新主干代码** | `origin/main = 92b94474` | **不通过（FAIL，仅交付完备性维度）** | 无 P0、无功能/安全回归；`Release Gate` 在主干为红，主干切不出任何发布；整改成果只存在于未提交工作区；消费侧文档与实现/已发布产物互相矛盾 |
| **B. 已发布产物** | `pantheon-base-v0.17.0 @ 728feb2b` | **有条件通过** | 四个 Foundation 资产齐全、尺寸与校验和一致、空库迁移到 29 且 `dirty=false`；但落后主干 2 个提交，缺 3 个 P1 修复与凭据脱敏 |

严重度分布：**P0 = 0，P1 = 3，P2 = 9，P3 = 5**。
功能维度（单租户、多租户生命周期/初始化/鉴权/数据隔离、安全与性能、UI）在实测中未发现 P0/P1 缺陷；
三条 P1 全部落在**工程化与发布能力**和**变更管理**上。

本文件记录这 17 项的最终处置。v0.17.1 关闭了全部 3 条 P1、7 条 P2（P2-1/2/3/5/6/7/10）、
1 条 P3（P3-4 改判为 P2-10），其余项显式登记为已知残余，附理由与责任方。

> **审计后新增的 P2-10**：整改过程中对工作区里一处「未评审的 down 文件改动」做了实测，
> 发现它修的是真实的回滚断链（`Error 1091`，schema 停在 dirty）。该项不在原审计的 17 项之内，
> 记为第 18 项；原 P3-4 相应改判。


## 1. 基线与判定方法

- 工作区：分支 `fix/tenant-plan-entitlements-and-audit` @ `3a446e3b`，34 个未提交文件，
  `git rev-list --left-right --count origin/main...HEAD` → `1 3`（已分叉）。
- 主干：`origin/main = 92b94474`；发布：`pantheon-base-v0.17.0 @ 728feb2b`（落后主干 2 个提交）。
- 已知参考值 `pantheon-base-v0.17.0` / `728feb2b` / `aef57e30118921c84ccdf52241e83869e98c21a0`
  均重新校验，未直接采信。`git fetch` 因网络代理 502 失败，改用
  `gh api repos/.../commits/main` 与 `git ls-remote origin -h refs/heads/main` 交叉确认 SHA。
- 判定依据按优先级：合同（`TENANT_CONTRACT_V1` V1.2）> 实测行为 > 代码 > 文档。
  文档只作为被检对象，不作为证据。
- 全程只读；未修改 `backend/` 与 `frontend/src/` 的任何被审逻辑，未发布、未削弱断言、
  未把未运行的测试记为通过。

## 2. 三条 P1 的定位、证据与整改

### P1-1 主干 `Release Gate` 红灯，主干切不出发布

- **位置**：`.github/workflows/release-gate.yml` → `sonarcloud-gate`（`resolved=false` 必须为 0）。
- **证据**：`92b94474` 上 `Release Gate Summary = failure`、`SonarCloud Gate = failure`
  （run `39733491324`，日志 `Release blocked: 8 unresolved SonarCloud issue(s).`）；
  SonarCloud API `/api/issues/search?resolved=false` → `total=8`，
  而 `/api/qualitygates/project_status` → `status=OK`（**质量门是绿的，未处理问题数不是**）。
  同期 `CI` / `Quality Gates` / `Security Gates` / `Core Smoke` / `Full Smoke` /
  `Tenant Smoke (multi-mode)` 在 `92b94474` 上**全部 success**。
- **判定**：失败面只有「未处理 issue 数」，不是功能或安全回归；但 `Release Gate` 是
  `publish-foundation-release.mjs` 的前置条件，红灯等于主干无发布能力。
- **整改**：8 条逐一消除（`go:S1192` ×2 提取常量 `sqlSelectMenuIdentity` /
  `condTenantScopedRow`；`go:S3776` 拆分 `collectPlatformOnlyMenuIDs` /
  `deleteTenantMenuGrants` / `collectTenantPlatformPermissionKeys` /
  `deleteTenantPermissionGrants`；`typescript:S6759` ×3 改 `Readonly<Props>`；
  `S3776` 提取 `useLayoutMenu` 纯函数；`S6582` 可选链）。
  **未**使用 `solo-override`，**未**改 Sonar 规则——`AGENTS.md` 明确该 label 仅维护者本人可用。
- **验证**：Sonar 判定只能由 CI 给出，本地无法替代；以 `Release Gate` 在 v0.17.1 提交上转绿为准。

### P1-2 整改成果停留在未提交工作区，没有可冻结的交付物

- **位置**：`git status`（30 个已跟踪修改 + 4 个未跟踪）；
  `.harness/tasks/2026-10-22-delivery-audit-remediation/manifest.json` 为 `in-progress`。
- **证据**：工作区分叉且脏；`npm run check:task-packet` 报
  `Linked review file does not exist: .harness/evidence/2026-10-22-delivery-audit-remediation/review.md`；
  新增测试 `tenant_lifecycle_guard_test.go`、`user_tenant_notfound_test.go` 未跟踪。
- **判定**：冻结时冻结的不是这份代码；工作区一旦被清理，整改即丢失；CI 从未在该状态上跑过，
  等于「未验证的修复」。
- **整改**：补齐证据目录 → 按逻辑切片提交 → 推送 → CI 在该提交上跑绿 →
  相关 task packet 置 `completed`。本文件即该动作的组成部分。

### P1-3 主干消费侧文档与实现/已发布产物互相矛盾

- **证据（主干原文）**：
  - `README.md` 宣称的当前版本与 `VERSION` / `CHANGELOG` 不一致；
  - `docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md:22` 仍写「当前运行态是单租户」，
    而 `platform.tenant_mode` 已可在 multi 下运行并有 Tenant Smoke 覆盖；
  - `.harness/STATUS.md` 声称「无 P0–P3 残留」，而主干 `Release Gate` 为红；
  - 实测 API 面与文档不符：`/api/v1/tenants` 与 `/api/v1/tenants/my` → **404**，
    `/api/v1/system/tenant/list` → 200。文档写的端点在运行时不存在。
- **判定**：消费侧文档（README / STATUS / 矩阵 / 部署指南）是外部判断交付状态的唯一入口，
  与实现矛盾即构成交付缺陷，而不是「文档待润色」。
- **整改**：README（中/英）、`CHANGELOG`、`RELEASE_NOTES_v0.17.1.md`、`.harness/STATUS.md`
  统一到 v0.17.1 的实测事实；scope 矩阵的运行态表述改为与 `tenant_mode` 现状一致。

## 3. P2 处置

| ID | 问题 | 处置 | 验证 |
|---|---|---|---|
| P2-1 | 已开通租户无法通过 API 删除：§2.1（archived 是退役路径）与 §2.2（最后 owner 不可移除）互斥 | **已修复**。`DeleteTenant` 先读 `id, code, status`；archived 租户即使有活跃成员也放行，在同一事务内先把成员置 `disabled` 再释放 code、置 `deleted`；活跃租户有成员仍拒 `tenant.has_active_members` | `TestDeleteTenantArchivedWithMembersSucceeds`（真 MySQL，绿） |
| P2-2 | 租户初始化失败后行残留：占着 UNIQUE `code`、无 owner，只能手工修 | **已修复**。`CreateTenant` 在 `ProvisionTenant` 失败后补偿 `Unscoped().Delete`；补偿本身失败时把两个错误都返回，不静默 | `TestCreateTenantRollsBackWhenProvisioningFails`（GORM callback 注入故障，真 MySQL，绿） |
| P2-3 | v0.17.0 不含主干后续 2 个提交，缺 3 个 P1 修复与脱敏 | **由 v0.17.1 关闭**：v0.17.1 从包含这些提交的主干切出 | 发布后校验 commit 祖先关系 |
| P2-4 | 并行跑包时 `TestBatchRevokeSessions_IsTenantScoped` 偶发失败（Redis 键空间共享） | **登记为已知残余**。单包 3/3 通过、`-p 1` 全量通过；非产品缺陷，是测试隔离度问题 | 见 §5 残余 |
| P2-5 | 主干其余文档漂移（CHANGELOG / scope 矩阵 / 部署指南 / STATUS） | **已修复**（同 P1-3） | 文档一致性走查 |
| P2-6 | 主干未落实 §2.1（archived 终态）与 §2.2（最后 owner） | **已修复**。`tenantStatusTransitionAllowed` 拒绝 archived→active；`ensureNotLastOwner` 拦截最后 owner 移除；被拒的迁移不再顺带写入同请求其他字段 | `TestTenantStatusTransitionAllowed`（8 表用例）、`TestUpdateTenantRejectsArchivedReactivation`、`TestRemoveMemberRefusesLastOwner`（真 MySQL，绿） |
| P2-7 | 跨租户用户读写返回 500 而非 404，违反 §5（外来 id 与不存在 id 不可区分） | **已修复**。`loadTenantUser` → `common.NewNotFound("user.not_found")`，handler 走 `common.FailWithError(c, common.CodeForError(err), ...)`；写路径先解析行再变更 | `user_tenant_notfound_test.go` |
| P2-8 | `Core Smoke` 仍是 advisory（`continue-on-error: true`），主干红灯不会让工作流失败 | **保留并显式声明**。`smoke-tenant` 是硬门（无 `continue-on-error`），Core Smoke 的 advisory 是 v0.16 分层决策，文档已说明；不构成「静默降级」 | 工作流文件走查 |
| P2-9 | 并发租户初始化未验证、无并发锁 | **登记为已知残余**。provision 在单事务内且 `code` 有 UNIQUE 约束，重复 code 会被数据库拒绝；但两个不同 code 的并发 provision 未串行化 | 见 §5 残余 |
| P2-10 | **回滚链中断**：`000020.down` 无条件删索引/列，而 `000024.down` 已删掉其中几个，`migrate down` 走到一半报 `Error 1091: Can't DROP 'tenant_id'; check that column/key exists`，schema version 停在 dirty。运维照 runbook 回滚会得到一个半删的库 | **已修复**。每个破坏性步骤改为先查 `information_schema` 再执行；回滚可完成，且可重新 up | `TestMigrationChainRollsBackAndReapplies`（真 MySQL，up→down→up 回到同一干净版本）。**反向验证**：把 `000020.down` 换回上一版，该测试即以 `Error 1091` 失败 —— 证明缺陷真实存在且测试有效 |

## 4. P3 处置

| ID | 问题 | 处置 |
|---|---|---|
| P3-1 | 已发布 v0.17.0 产物内含已失效的开发凭证明文 | **不回收、登记**。凭据已失效（实测 MySQL 拒绝旧口令），且历史 tag 产物不可变；v0.17.1 起证据文件不再落明文 |
| P3-2 | 租户存在合同外的第四态 `deleted` | **登记为合同债**。`deleted` 由软删产生，`DeleteTenant` 需要它；`TENANT_CONTRACT_V1` §2.1 只写了三态。补合同属维护者动作（改合同需 L2 流程），本轮不改合同 |
| P3-3 | 套餐门禁仅前端，后端不按套餐限制 | **设计边界，非缺口**。已在 `PANTHEON_BASE_TENANT_PLAN_AND_AUDIT_FIXES_2026-10-22.md` §0 显式声明：套餐是 UX 呈现层，授权最终由角色权限 + `PlatformCapabilityMiddleware` 判定 |
| P3-4 | 已发布迁移的 down 文件在工作区被修改 | **改判并已修复（见 P2-10）**。初判为「未评审的改动」，实测该改动修的是一条真实断链：原文件会让回滚中途失败。现由 `TestMigrationChainRollsBackAndReapplies` 覆盖 |
| P3-5 | 开发态默认管理员口令为 `123456` | **保留**。仅 `APP_ENV=development` 生效；生产启动校验拒绝默认口令 |

## 5. 已知残余（v0.17.1 时登记，v0.17.2 已全部关闭）

v0.17.1 冻结时这 5 项被显式登记为不阻塞交付。v0.17.2 逐项关闭，其中两项在实测时暴露为
**真实缺陷**（不是"未验证"，而是"验证即失败"），因此本轮不是重新标注，而是修复。

| # | 登记时的表述 | v0.17.2 的处置 |
|---|---|---|
| 1 | **P2-4 测试隔离度**：`TestBatchRevokeSessions_IsTenantScoped` 多包并行时偶发失败，根因是 Redis 键空间共享 | **已关闭**。`testredis.OpenDB` + 预留索引注册表，每包独立逻辑库（authtoken=2 / login=3 / middleware=4 / session=5），默认库 1 不变。4 包并行 3 轮全绿 |
| 2 | **P2-9 并发 provisioning**：不同 `code` 的并发 `CreateTenant` 未串行化，资源竞争未验证 | **已关闭，且实测为缺陷**。设置/字典复制用 `INSERT ... SELECT` 读同一张表，两个租户并发创建触发 `Error 1213 Deadlock found when trying to get lock`，调用方看到 500。改为「非锁定读 + 插入」；另修同 code 竞态下 loser 收到 `Error 1062` 原始驱动错误的 500。8 轮并发全绿 |
| 3 | **P3-1 v0.17.0 凭据残留**：历史产物不可变，凭据已失效 | **条件已消除**。产物本身仍不可回收（这一点无法改变），但新增 `check-release-secrets.mjs` 作为发布硬前置：DSN / URL / `-p<值>` 三种形态命中即拒绝打标。v0.17.2 发布时该门禁只扫描了 `bundle/`；事后复核发现 v0.17.0 残留实际位于 `repo.tar`，已补扫并确认 v0.17.2 的 `repo.tar` 干净。门禁现同时扫描 `bundle/` 与 `repo.tar`，缺失 `repo.tar` 时失败关闭；对 v0.17.0 的两种残留形态均有回归用例 |
| 4 | **P3-2 合同第四态**：`deleted` 未进合同，补合同需走 L2 评审 | **已关闭**。合同 V1.3 澄清 §2.1（墓碑语义、不可寻址、不可作为输入、删除标记是 `status` 而非预留的 `deleted_at`），按既有 contract-change 流程记录触发与范围，不自行宣告 G4 通过。补写过程中实测发现并修复了「墓碑可被复活」缺陷 |
| 5 | **race 检测的本地缺口**：`go test -race` 因 MinGW 链接器失败，race 覆盖仅来自 CI | **已关闭**。真实原因是 `PATH` 上的 Cygwin `gcc`；`CC=/d/msys64/mingw64/bin/gcc.exe` 下 race 构建与运行正常。`pkg/tenant`、`pkg/testredis`、`internal/middleware`、`modules/auth/session` 与租户生命周期/并发用例 race 全绿 |

证据：`.harness/evidence/2026-10-23-residual-closeout/`；发布说明 `RELEASE_NOTES_v0.17.2.md`。

## 6. v0.17.1 的验证边界声明

明确区分「本地实测通过」与「仅 CI 可判定」：

| 项 | 状态 |
|---|---|
| `go build` / `go vet` / `gofmt -l` | 本地通过 |
| `go test -p 1 ./...`（真 MySQL 8.0.36 + Redis 7，`*_REQUIRED=true`） | 本地通过 |
| `tsc --noEmit` / `eslint src --max-warnings 0` / `vitest run` | 本地通过 |
| `npm run prebuild` / `vite build` | 本地通过 |
| 浏览器走查（17 条路由，0 console error，0 `/api/` ≥400） | 本地通过 |
| 空库迁移到版本 29 且 `dirty=false` | 本地通过（一次性库 `pantheon_accept_fresh`） |
| **SonarCloud 未处理数 = 0 / `Release Gate` 绿** | **仅 CI 可判定**，本地不声称 |
| `go test -race` | **本地不可执行**（链接器限制），仅 CI |

## 7. 关联

- 验收矩阵：`docs/reviews/PANTHEON_BASE_ACCEPTANCE_MATRIX_2026-10-21.md`（72 通过 / 14 不通过 / 4 未验证）
- 前一轮整改：`docs/reviews/PANTHEON_BASE_ISSUES_AND_REMEDIATION_2026-10-21.md`
- 租户套餐与审计：`docs/reviews/PANTHEON_BASE_TENANT_PLAN_AND_AUDIT_FIXES_2026-10-22.md`
- 证据：`.harness/evidence/2026-10-22-v0171-release-closeout/`
- 发布说明：`RELEASE_NOTES_v0.17.1.md`
