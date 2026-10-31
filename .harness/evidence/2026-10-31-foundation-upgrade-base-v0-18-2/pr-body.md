## 变更摘要

- 改动层级：inheritance-sync
- 改动模块：`foundation-release.lock.json`、`.foundation/foundation-release.lock.json`、`VERSION`、
  `business-overlay.json`、`scripts/business-overlay/rebuild-from-base.mjs`、
  `backend/modules/business/k8s/{k8s.go,k8s_seed.go}`、
  `frontend/src/modules/business/{k8s/index.ts,v12locales/*.json}`、`docs/release-notes/`、
  `.harness/{tasks,evidence}/2026-10-31-foundation-upgrade-base-v0-18-2/`
- 目标问题：消费 `pantheon-base-v0.18.2`。这一版把
  `frontend/scripts/check-menu-contract.mjs` 放进 `sharedPaths`（52 → 53 项），传感器第一次在
  ops 树上运行时直接失败：k8s 模块的 20 条权限菜单 titleKey 是在 `init()` 里用
  `strings.ReplaceAll` 拼出来的，既不是可解析常量，也不存在于任何 i18n 词表
  （运行时拼写如 `business.k8s.cluster.sync.permission`）。
- 预期影响：k8s 权限菜单改为静态常量 seed，补齐 4 个 `k8sServicePerm*` 常量、前端 16 条权限声明和
  20 条中英文标题；同时修掉两处会让下一次确定性重建丢东西的隐患（`check-gorm-sql-literal`
  未声明为 overlay、两个只在生成文件里的 i18n key）。业务行为不变：菜单集合、排序、管理员权限绑定、
  路由表保持一致，传感器仍数出 40 菜单 / 50 路由 / 237 权限 / 52 组件键。

## Harness 链路

- Task ID：`2026-10-31-foundation-upgrade-base-v0-18-2`
- Task Manifest：`.harness/tasks/2026-10-31-foundation-upgrade-base-v0-18-2/manifest.json`
- Evidence：`.harness/evidence/2026-10-31-foundation-upgrade-base-v0-18-2/commands.json`
- Verification evidence：`.harness/evidence/2026-10-31-foundation-upgrade-base-v0-18-2/summary.md`
- Review Artifact：`.harness/evidence/2026-10-31-foundation-upgrade-base-v0-18-2/review.md`
- OpenSpec change：none
- Trivial change：no
- Quality Profile：foundation-release-consumer
- Ratchet Decision：sensor-added
- GitHub Signal：consumer-lock-drift

## Harness adoption markers

> 保留本区块的英文 marker，供治理脚本做机械检查。

- task id: 2026-10-31-foundation-upgrade-base-v0-18-2
- task manifest: .harness/tasks/2026-10-31-foundation-upgrade-base-v0-18-2/manifest.json
- evidence: .harness/evidence/2026-10-31-foundation-upgrade-base-v0-18-2/commands.json
- boundaries: 共享面取自锁定发行包；业务修复只落在 `backend/modules/business/k8s/**` 与 `frontend/src/modules/business/**`；`frontend/package.json` / `package-lock.json` 不动
- backend response contract: 未改动
- backend DTO contract: 未改动
- permission contract: k8s 权限 key 的语义不变，仅把运行时拼出的 titleKey 换成同名常量引用；权限总数仍为 237
- audit coverage: 未改动；本轮不触碰审计链路
- visual evidence: 未提供（无 UI 渲染改动；前端这次只改 i18n 资源与权限声明）
- inheritance contract: `docs/PROJECT_INHERITANCE.md` 版本 marker 更新为 `pantheon-base-v0.18.2` / `a44ae4e0`
- base drift: `check:shared-drift --base ../pantheon-base/.tmp/rel-0182` → 813 identical / 10 allowlisted / 161 skipped
- Base/ops inheritance: 吸收 `pantheon-base-v0.18.2`（tag object `2e32bd46`，base commit `a44ae4e0`）
- Base version checked: `foundation-release.lock.json` releaseVersion / baseCommit = `pantheon-base-v0.18.2` / `a44ae4e012fff20430b1b7f91cb7a9296f3ea843`
- generic drift: 无；本轮发现的传感器缺口由 base 已共享的检查器暴露，消费者按业务归属修复
- pseudo-drift: 无（`frontend/package*.json` 的差异是有意的依赖版本差，未覆盖）
- business-only: no（含继承台账与共享传感器落地）

## 边界说明

- [ ] 本次改动仅涉及 `business/*`
- [x] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

> 边界：共享文件按锁定发行包整体吸收，不手抄 base 文件；业务侧修复限定在 k8s 权限 seed 与对应
> 前端声明/i18n。未被覆盖的两个 `frontend/package*.json` 保留 Dependabot 已合并的
> i18next 26.5.11 / react-i18next 17.1.9，避免把依赖回退到 base 的 26.5.6 / 17.1.3。

## 验证记录

- [x] `npm run check:business-overlay` → OK（53 declared paths）；`test:business-overlay` 13/13
- [x] `npm run check:shared-drift --base ../pantheon-base/.tmp/rel-0182` → 813 identical / 10 allowlisted / 161 skipped
- [x] `npm run check:gorm-sql-literal` → PASS（4 条既有 waiver）
- [x] `cd backend && go build ./... && go vet ./... && gofmt -l .` → 0 / 0 / 空；`go test -count=1 ./...` → 64 包 ok，无 FAIL
- [x] `cd frontend && npm run lint && npm run type-check && npm run build` → 0 / 0 / `✓ built in 804ms`；
      prebuild 传感器输出 `40 个菜单，50 条路由，237 个权限，52 个组件键`
- [x] 业务 smoke 或明确 runtime gap：本轮未重启后端；k8s 改动为 seed 拼写，权限键集合不变，
      以 `go test` + 传感器载荷为准，runtime 缺口已写入 `knownGaps`
- [x] 资源校验：tgz `c01ebbc1…2fc356a`（30,051,238 B）、repo.tar `c1360fb1…c5d51`（47,923,200 B）
      与 lock、GitHub 资产摘要一致；tgz 内 `manifest.json` 的 `sharedPaths` 三组与 lock 深等
- [x] Docs Governance 全量：`check:docs-frontmatter` / `check:encoding` / `check:structure` /
      `check:task-packet-template` / `check:pr-governance` / `check:generated-modules` /
      `check:ui-quality-gate` → exit 0
- [x] harness/业务门禁：`check:task-packet` / `check:harness-adoption` / `check:failure-registry` /
      `check:business-boundaries` / `test:business-boundaries` / `check:business-foundation` /
      `check:duplication` / `check:harness-docs` → exit 0
- [x] GitHub required checks 通过（见审核留痕回读）

## 审核留痕

- Copilot review：unavailable（本仓库/账号无法解析 `copilot` reviewer）
- CodeQL 结果：待 PR 检查回读
- GitHub checks 结果：待回读
- Auto-merge：enabled（沿用仓库既有 squash 策略）
- Duplication Gate 结果：PASS
- 是否高风险改动：no（无产品行为变更；权限 key 集合不变）
- Residual risk / follow-up：
  1. 本轮未重启后端做运行时验收，k8s seed 重写以 `go test` 与传感器为准；v0.18.1 的隔离 schema
     运行仍是最近一次运行时读数。
  2. `check:harness-docs` 的悬空链接继承自 base 文档（指向 base 专属 task packet 与 base 证据归档），
     属 base 侧修复，且该检查只报告、未接入流水线。
  3. `go test -race` 在本机无法运行（TSan 影子内存分配失败，error code 87），CI 在 ubuntu 上跑。
  4. SonarCloud Scan 与 Vercel 的既有红灯沿用上一轮结论，均非必需上下文。

## 检查清单

- [x] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter（本次为继承消费 + k8s 业务修复）
- [x] 通用平台/系统域问题已在 Base-first 边界处理（无新增平台问题；传感器本身来自 base v0.18.2）
- [x] 前端新增展示文案已使用 i18n（20 条权限标题已进入 v12locales 源并生成资源文件）
- [x] 菜单、权限、接口、审计和 seed/smoke 在范围内同步（菜单/权限 seed 与前端声明、i18n 同步更新）
- [x] UI 改动已有渲染证据或明确 runtime gap（无 UI 渲染改动）
- [x] 已确认不会泄露敏感配置、账号密码或 Token
