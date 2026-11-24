# 项目继承说明

English version: [PROJECT_INHERITANCE.en.md](./PROJECT_INHERITANCE.en.md)

`pantheon-ops` 采用“完整 Pantheon Base 快照 + 声明式 business overlay”的继承模式。升级时从 `foundation-release.lock.json` 锁定的正式 Base release 重建完整产品树，再注入 Ops 业务资产；不在旧树上逐文件同步，也不维护共享源码白名单。

## 方法消费与产品继承

方法直接引用 [harness 能力适配交付规则](../../pantheon-harness/patterns/capability-adaptive-delivery.md)，消费版本和规范化文本 SHA256 记录在 `.harness/method-consumption.json`。本仓 `AGENTS.md`、`CLAUDE.md` 和 `.harness` 已由 `repositoryOverlayPaths` 声明，重建时作为 Ops 仓库治理资产保留；不复制 harness 正文或 Base 共享脚本。

在 Ops 根目录运行 `node ../pantheon-harness/scripts/harness/check-method-consumption.mjs --root . --source ../pantheon-harness`。发现方法版本/哈希变化时先审查，才用同一命令追加 `--update` 更新 receipt。CI/离线环境必须提供同一 harness 快照；缺失来源是显式失败。产品 `VERSION` 与 `foundation-release.lock.json` 不因方法更新而改变；Base 共享产品配置或检查器仍须正式 release 后重建消费。

兼容性阅读以锁定快照内继承的 `docs/contracts/*` 为准；sibling Base HEAD 只提供演进参考，差异须标明。receipt 的完整 `sourceCommit` 固定 CI policy checkout，正常验证不刷新。未发布 helper/checker 另行发行，不能用 policy receipt 证明其已发布。

## 版本字段口径

仓库中并存多套版本字段，含义不同、不能相互替代：

- `foundation-release.lock.json`：唯一权威继承基线。`releaseVersion`、`baseCommit` 与资产 SHA256 决定重建内容与 CI policy。
- `VERSION`：继承自锁定 Base 快照的产品版本，必须与 lock 的 `releaseVersion` 数值一致（当前 `0.18.9`）。方法更新不改它；与 lock 不一致即视为继承漂移。
- `SHELL_VERSION.json`：锁定快照内 consumer shell 的 pin（`1.4.0`，`compatibleMethodKit` 1.4.0）。它描述 Base 仓库外壳，不等于 Ops 直接消费的方法版本。
- `.harness/method-consumption.json`：Ops 直接消费的方法 kit 版本（`methodVersion`，当前 `1.5.0`）与规范化文本哈希；方法更新只改这里与 receipt。
- `frontend/package.json` 的 `name`/`version`：Base UI 包身份（`@duanxldragon/pantheon-base-ui` `0.11.2`），随 Base release 变化，与 `VERSION` 无关。
- `CHANGELOG.md` 与 `docs/release-notes/`：历史交付记录；最新业务发布条目 `1.2.0` 不代表当前产品版本。

Ops 的 `quality.yml` 单独配置跨层边界与 checker wiring 检查；`check:coverage` 只是手动继承工具，须提供实际覆盖率文件与另行审查的阈值：`npm run check:coverage -- <artifact> --threshold <number>`。脚本存在、CI 配置引用、观察到的运行结果和合并必过检查必须分别报告。

## 1. 所有权

- `pantheon-base`：拥有 `platform`、`system/*`、共享前后端、生成器和产品级工程配置。
- `pantheon-ops`：拥有 `business/bizscope`、`business/cmdb`、`business/deploy`、业务 smoke、业务设计和仓库身份。
- `business-overlay.json`：Ops 唯一机器可读资产清单和装配清单。
- `.business-overlay-report.json`：每次重建输出的 Base commit、Ops 文件哈希、生成文件和 import 重写记录。

重建结果保留 Base 的目录布局和 Go module `pantheon-base`。业务源码在注入时将历史 import `pantheon-ops/backend/...` 改写为 `pantheon-base/...`，共享源码不再改写 module identity。

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
# 不指定版本时获取 GitHub 最新稳定版；本次明确锁定 v0.18.9：
npm run fetch:base-release -- --release pantheon-base-v0.18.9

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
- Base version: `pantheon-base-v0.18.9` (`6316f9b7189c86542424145fa49344ad737e952b`)
- If a foundation rule must change, update `pantheon-base` first

## 8. 业务数据迁移

业务 SQL 的唯一源文件位于 `backend/modules/business/<module>/.../migrations/`，使用六位编号与配对的 `.up.sql` / `.down.sql`。`businessMigrations` 显式列出这些源文件；来源必须属于已声明的 `businessPaths`。重建器把它们装配到固定的 `backend/pkg/database/migrations/`，并记录为生成文件，沿用 Base 的启动迁移失败处理。

不允许业务清单直接拥有 `backend/pkg/database`，也不允许指定任意目标路径。缺少配对、重复版本/方向、非法来源或与 Base 迁移编号冲突都会中止重建。新增 Base 版本占用业务编号时，只能调整尚未发布的业务迁移；已发布迁移不可重编号，必须另行设计兼容升级。当前未发布 host-state 回填使用 `000032`，Base 保留 `000031`。

## 9. Go 依赖图装配

重建器将 `goDependencies` 加入正式 Base 快照后，在生成文件哈希报告前执行 Go 的完整包与测试依赖加载。归一化产出的 `backend/go.mod` / `go.sum` 作为生成文件记录；缺少包、校验和或加载失败会中止重建，不生成成功报告。

暂存树必须直接通过 `go list -buildvcs=false -mod=readonly -deps -test ./...` 及适用构建/测试，不能在验收期间临时使用 `-mod=mod` 修补依赖。归档快照通过已校验 release/lock 记录来源，不读取外围仓库的 VCS stamp。离线缓存不足时补齐所需依赖后重建；不能跳过加载检查。正式发行与完整 race/业务 smoke 门禁继续适用。
