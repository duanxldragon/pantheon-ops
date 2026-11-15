# 项目继承说明

English version: [PROJECT_INHERITANCE.en.md](./PROJECT_INHERITANCE.en.md)

`pantheon-ops` 采用“完整 Pantheon Base 快照 + 声明式 business overlay”的继承模式。升级时从 `foundation-release.lock.json` 锁定的正式 Base release 重建完整产品树，再注入 Ops 业务资产；不在旧树上逐文件同步，也不维护共享源码白名单。

## 1. 所有权

- `pantheon-base`：拥有 `platform`、`system/*`、共享前后端、生成器和产品级工程配置。
- `pantheon-ops`：拥有 `business/bizscope`、`business/cmdb`、`business/deploy`、业务 smoke、业务设计和仓库身份。
- `business-overlay.json`：Ops 唯一机器可读资产清单和装配清单。
- `.business-overlay-report.json`：每次重建输出的 Base commit、Ops 文件哈希、生成文件和 import 重写记录。

重建结果保留 Base 的目录布局和 Go module `github.com/duanxldragon/pantheon-base/backend`。业务源码在注入时将历史 import `pantheon-ops/backend/...` 改写为 `pantheon-base/...`，共享源码不再改写 module identity。

## 2. 装配扩展点

业务模块通过 Ops 重建器生成的静态 overlay 注册表接入：

- `backend/modules/business/business_overlay_registry.go`
- `frontend/src/modules/businessOverlay.ts`
- `frontend/src/core/router/businessOverlayComponentRegistry.ts`
- `backend/modules/system/iam/menu/business_overlay_component_registry.go`

重建器只在 Base 的聚合入口接入这些静态注册表。Base 的 `generated_*` 注册表仍由低代码生成器独占，可以反复生成和 purge，不得存放 Ops 内置业务注册。菜单、权限、数据库 migration、seed、运行态 i18n 和 API 路由由各业务模块负责。

Base 的菜单契约门禁（`check-menu-contract.mjs`）按命名约定发现组件注册表文件（前端 `*Registry.ts`、后端 `*registry.go`），因此 `businessOverlayComponentRegistry.ts` / `business_overlay_component_registry.go` 会被门禁自动纳入一致性校验，无需在 base 侧登记文件清单。

## 3. 升级流程

以下命令在 Ops 仓库根目录执行：

```powershell
# 1. 从 GitHub 查询最新稳定的正式 Base release，下载四个资产，校验 SHA256、
# manifest 和不可变 tag，全部成功后才更新 Ops 锁文件。
# 不指定版本时获取 GitHub 最新稳定版；以下为当前活动版本 v0.18.8 的获取示例：
npm run fetch:base-release -- --release pantheon-base-v0.18.8

# 2. 从该正式锁定版本重建暂存树，不覆盖当前 Ops。
npm run rebuild:from-base -- --target .tmp/clean-base-overlay
node scripts/business-overlay/check-business-overlay.mjs --root .tmp/clean-base-overlay
npm run check:shared-drift -- --root .tmp/clean-base-overlay

# 3. 验证 backend CGO race、frontend、业务 smoke 和 GitHub 质量门禁。
# 4. 全部通过后，才以暂存树替换 Ops 工作树。
```

每周 GitHub workflow 使用相同的最新正式 release 获取和锁定重建流程。重建与漂移校验均核验 GitHub 官方资产，读取解压快照的真实字节，不读取父目录仓库的 Git HEAD。入口拒绝本地 Base 来源参数；指定 `--release` 时必须获取该正式版本，不能退回 latest 或其他 tag。缓存归档必须符合锁文件及官方 release 的 SHA256；解压目录每次从验证过的归档重建，避免旧文件或被修改的缓存成为基线。

## 4. 禁止路径

- 不发布 Base patch 只为补消费白名单。
- 不把 Base release producer 脚本复制到 Ops。
- 不把 Ops 全树 import 改写成 `pantheon-ops`。
- 不保留未在 `business-overlay.json` 声明的通用产品文件。
- 不在未完成暂存验证前覆盖当前 Ops。

## 5. 变更落点

- 通用平台、系统域、共享 UI、共享生成器问题回 `pantheon-base` 修复。
- 业务模型、业务页面、业务权限和业务验收留在 `pantheon-ops`。
- Base 升级只处理真实 contract break；代码行数差异和共享文件差异不再作为人工同步任务。

## 6. 验证门禁

- `npm run test:business-overlay`
- `npm run check:business-overlay`
- Windows `go test -race ./...`（在 `backend/` 执行并启用 CGO）
- `frontend`: lint、type-check、build
- business smoke
- SonarCloud revision、代码量和重复率复核

当前仓库替换属于 L2 human gate：暂存验证通过后才能执行最终基线替换。

## 7. 继承锚点

> 以下英文 marker 供 `../pantheon-base/scripts/harness/check-inheritance-contract.mjs` 机械校验，请勿翻译或删除；版本变化时必须与 `foundation-release.lock.json` 同步更新。

- Base repository: `../pantheon-base`
- Base version: `pantheon-base-v0.18.8` (`34428814a9efc997fd93d4434406a53df6460323`)
- If a foundation rule must change, update `pantheon-base` first

## 8. 业务数据迁移

业务 SQL 的唯一源文件位于 `backend/modules/business/<module>/.../migrations/`，使用六位编号与配对的 `.up.sql` / `.down.sql`。`businessMigrations` 显式列出这些源文件；来源必须属于已声明的 `businessPaths`。重建器把它们装配到固定的 `backend/pkg/database/migrations/`，并记录为生成文件，沿用 Base 的启动迁移失败处理。

不允许业务清单直接拥有 `backend/pkg/database`，也不允许指定任意目标路径。缺少配对、重复版本/方向、非法来源或与 Base 迁移编号冲突都会中止重建。新增 Base 版本占用业务编号时，只能调整尚未发布的业务迁移；已发布迁移不可重编号，必须另行设计兼容升级。当前未发布 host-state 回填使用 `000032`，Base 保留 `000031`。

## 9. Go 依赖图装配

重建器将 `goDependencies` 加入正式 Base 快照后，在生成文件哈希报告前执行 Go 的完整包与测试依赖加载。归一化产出的 `backend/go.mod` / `go.sum` 作为生成文件记录；缺少包、校验和或加载失败会中止重建，不生成成功报告。

暂存树必须直接通过 `go list -buildvcs=false -mod=readonly -deps -test ./...` 及适用构建/测试，不能在验收期间临时使用 `-mod=mod` 修补依赖。归档快照通过已校验 release/lock 记录来源，不读取外围仓库的 VCS stamp。离线缓存不足时补齐所需依赖后重建；不能跳过加载检查。正式发行与完整 race/业务 smoke 门禁继续适用。

## 10. 当前业务空库基线

业务迁移 `000033_core_business_schema` 在关闭 AutoMigrate 时补齐 CMDB、BizScope、Deploy、Service、K8s 的 29 个核心表，与 `000032` 配对装配到正式 Base 迁移链。源文件属于 `business/opsinit/migrations`；禁止手工修改共享目标目录。

该基线使用 CREATE TABLE IF NOT EXISTS 保留既有表和数据，down 同样保留表和数据。它不修复不兼容历史列，也不授权生产升级或多租户迁移；历史生产数据库仍需单独迁移评审、备份与演练。空库、版本 32 到 33、重复执行和保留式 down 的当前本地证据见 `.harness/evidence/2026-11-07-ops-planned-task-closeout/summary.md`。

The fresh baseline uses generated active-row keys to enforce active natural-key uniqueness while permitting soft-deleted identities to be reused. Existing incompatible tables are still preserved, so legacy index corrections require a separate upgrade migration.

## 11. 部署日志尝试元数据

业务迁移 `000034_deploy_log_attempt` 在已有 `000033` 表基线上增加日志 `attempt_no`。新日志保存写入时的步骤尝试编号，步骤检查点更新不会重写旧日志归属。历史日志保留为 `0`（未归属），仅在明确选择所有尝试时合并展示；不推测历史归属。迁移有表/列存在保护，重复执行和保留式 down 不丢失输出。通过业务清单装配到正式迁移链，禁止手工修改共享目标源码。生产升级仍遵守单独评审、备份和演练门禁。

## 12. 部署步骤检查点历史

业务迁移 `000035_deploy_step_history` 提供仅含身份、步骤尝试、状态、版本和时间的追加式检查点观察记录。每次既有检查点更新与快照在同一事务提交，不改变引擎重试/租约/恢复策略。历史视图使用该步骤尝试最后记录的状态，旧检查点不回填；这不是完整执行时间线或终态保证。现有任务详情按 CMDB 权限过滤主机、步骤、日志尝试和历史元数据，历史投影最多500项并明确截断。无新公开路由、秘密/配置/错误原文或共享源码 override。配对迁移由业务清单正式装配，down 保留历史；生产升级、独立评审和正式最终验收仍开放。

业务迁移 `000036_deploy_log_sequence_scope` 将部署日志唯一键从全局 `sequence` 修正为 `(task_id, sequence)`。序号分配本来就以任务为单位，旧索引会丢弃第二个任务的同序号日志。升级只接受已知旧索引形状，未知形状明确失败；缺表跳过、正确索引重放幂等，down 保留新索引和历史。业务清单正式装配 up/down，禁止手改共享生成目标。真实 MySQL 验证跨任务保留、任务内去重、并发写入、旧库升级/重放及保留式 down；真机成功部署后的逆操作也验证第二任务日志。此证据不替代完整 P0 验收或生产迁移门禁。

## Earlier r5 checkpoint (historical; superseded by the installation below)

Official GitHub latest stable v0.18.8 and its four immutable assets were acquired and verified. The fully assembled candidate is `.tmp/ops-v0188-final-r5`; its lock/report target commit `34428814a9efc997fd93d4434406a53df6460323`. A working-tree install was denied by Windows directory permissions at `backend/modules/system/iam/tenant/tenant_owner_lifecycle_test.go`. All 158 install preimages were restored; the active lock/report remain v0.18.6. Candidate validation does not establish active adoption. No local Base/Ops Git comparison or shared-source copying was used. Later asset replays record `freshGitHubLookup:false`.

Business menu/environment migration000038, bounded recorded-state Dashboard adapters and CMDB permission-key registration are implemented. The candidate passes frontend lint/type-check/build, 414 unit tests and 24 actual browser tests via an isolated IPC/CDP adapter (`nativeRunnerPassed:false`). Dashboard renders cover four empty-state widgets in light/dark at 1440/390; full provider/state acceptance remains open. Six P0 parent packets remain open, as do final visual/functional acceptance and production gates. See [current execution evidence](../.harness/evidence/2026-11-14-ops-latest-foundation-and-plans/summary.md).

## v0.18.8 installation continuation

The active installation now consumes official `pantheon-base-v0.18.8`, commit `34428814a9efc997fd93d4434406a53df6460323`, assembled from verified GitHub release assets and the current declared business overlay. Prior denied-installation records describe the earlier attempt and are preserved. The new installation uses fresh preimages; six P0 parents, external-provider acceptance and final visual/functional acceptance remain open. See [current evidence](../.harness/evidence/2026-11-14-ops-latest-foundation-and-plans/summary.md).
