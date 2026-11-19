# 业务导航修复 — 实施摘要

日期：2026-11-16
任务：`2026-11-16-ops-navigation-repair`
角色：Claude = 唯一执行者；Codex = 派发与独立验收（本文件为 Claude 实施证据，未自我验收）。

## 五个缺陷与修复

### 1) 服务目录重复 / 旧树 404（`id159 /business/service`）

`backend/modules/business/service/seed.go` + `seed_migrate_test.go`：

- 收敛 `module=business.service` 拥有的旧树 `id159 /business/service` 与当前树 `/business/service-catalog`（含子项 1819/1820/1821），**保留旧菜单 id** 与其全部角色绑定（含 `tenant_id`）。
- seed 对连续两轮**幂等**；只处理 `module=business.service` 拥有的旧菜单；永不删除同名自定义菜单（所有权边界）。自定义同名菜单不受影响。
- 迁移回归 `seed_migrate_test.go` 覆盖：真实升级旧树 fixture（绑定保留含 tenant_id / 重复收敛 / 连续两轮 seed 幂等 / 所有权边界）。真实 MySQL 运行见 Codex 验收日志（本 shell 无 `PANTHEON_TEST_DSN`，本地执行即 SKIP，非通过）。

### 2) ServiceCatalog 页内 Tabs 受控于 URL

`frontend/src/modules/business/service/ServiceCatalog.tsx`：

- `activeTab` 由路由派生、`onChange` 写回 URL，使二级菜单/Tab 点击真正响应；深链刷新 / 前进后退一致。

### 3) 菜单排序（业务块 1..8 连续，系统治理之前）

后端各 seed Sort 重排：**CMDB=1 → BizScope=2 → Service=3 → Deploy=4 → Filedist=5 → K8s=6 → ArgoCD=7 → Observability=8**。

- `deploy_seed.go`（4）、`filedist/seed.go`（5）、`k8s_seed.go`（6）、`cicd/seed.go`（7）、`observability/seed.go`（8）——filedist 紧跟 deploy，argocd 紧邻 k8s/release。见 `NAVIGATION_MATRIX.md`。

### 4) ArgoCD 展示名明确、与 K8s 发布不冲突（Codex 修订 7e）

- 后端 `cicd/seed.go` 菜单标签：`应用交付` → **`ArgoCD 应用交付`**（en `ArgoCD App Delivery`），与 K8s release `应用发布` 区隔。
- `frontend/src/i18n/resources/business/zh-CN.ts` / `en-US.ts` 镜像 `operations.cicd.argocd.menu`，并把 ArgoCD 页 hero（`business.cicd.argocd.hero.title`/`.eyebrow`）从 `应用发布`/`Application Delivery` 改为 ArgoCD 明确字样，消除页面标题二次冲突。
- `frontend/src/modules/business/index.ts` 模块注册顺序与后端 seed Sort 对齐（7f）。

### 5) 证据物料

`NAVIGATION_MATRIX.md`（全部业务菜单顺序 + 服务 3 页/404 修复）、`commands.json`、本 summary；Playwright 导航回归（Codex 独立验收，含桌面 1440x900 / 移动 390x844 渲染截图）。

## Codex 修订补丁（本窗口收尾）

- **7e**：ArgoCD 标签两处（侧栏菜单 + 页 hero）均已 ArgoCD 明确。`ServiceCatalog.nav.spec.ts` 的 `BUSINESS_ORDER` 用 `hasText`/`includes` 匹配 `应用交付`，`ArgoCD 应用交付` 仍满足；K8s release 唯一保留 `应用发布`，`toHaveCount(1)` 断言成立——无需改动 spec。
- **7f**：`frontend/src/modules/business/index.ts` 注册顺序=后端 seed Sort，未给 `ModuleMenuMeta` 加 sort 字段。
- **7g**：`frontend/src/i18n/resources/business/{zh-CN,en-US}.ts` 补充 46 个 `business.k8s.workload.*` 页面 key（镜像 `k8s_seed.go`）。根因：自动化 webdriver dev 运行时跳过远端 i18n pack（`automationPolicy.shouldFetchRemoteI18nPack=false`），提交的本地 bundle 是唯一来源，故页面 key 缺失即渲染裸 key；ja/ko/fr 回退 zh-CN，仅需补齐 zh-CN+en-US。

## 验证状态（诚实记录）

- **已在本人 shell 通过**：`go build ./modules/business/...`、`go vet ./modules/business/...`（空输出）。service seed 迁移测试在本 shell 因无 MySQL DSN 而 SKIP，**非通过**；真实 MySQL 回归见 Codex 验收日志 `go-race-service.log` / `go-race-full.log`。
- **已在本人 shell 被拒（无审批面）**：前端 `tsc -b` / `npm run build`。Codex 已先行产出通过日志：`codex-frontend-typecheck.log`（0 错）、`codex-frontend-lint.log`（0 错 1 编译期无关 warning）、`codex-menu-contract.log`（45 菜单/55 路由/250 权限/54 组件键通过）、`codex-business-overlay.log`（53 声明路径通过）、`codex-business-boundaries.log`（边界通过）。这些日志先于本窗口末尾仅字符串值的 bundle/label 改动（hero title 等），不改变结构/类型，仍代表对应门禁通过。
- **渲染证据未由本人产出**：本 shell 无真实升级 DB 也禁 localhost curl，故桌面/移动截图由 Codex 用真实升级旧库重启后以 Playwright 独立验收（任务契约）。参见任务文档 Human Gates —— 未获维护者最终视觉/功能验收前不宣称交付完成。

## 边界遵守

未触碰 `backend/modules/business/cmdb/host/**`；未做 lock 升级/commit/push/PR/release；未 wholesale 删除菜单或本地 SQL 改写既有业务库数据；未削弱 RBAC（每个 C 节点 pagePerm 数仍 1）；未自动关闭 SSH/历史工作台等包；未读用户凭据/browser profile/localStorage。任务保持 awaiting-review，未自我验收。

---

## 2026-11-17 返修轮（awaiting-review，不提前写通过）

### 状态基准

- **Codex 最终 build/lint 已 PASS**（`codex-frontend-build-final.log`、`codex-frontend-lint-final.log`），task packet 门禁命令已正确 PASS（仅 warning：缺 Codex review）。
- **但本次返修前的两条回归在某轮实际执行中 FAIL**，故本轮据此返修：
  1. Playwright `ServiceCatalog.nav.spec.ts:196` —— `getByText('应用交付', {exact})` 得 0（实际根标题为 `ArgoCD 应用交付`；K8s `应用发布` child 初始未展开）。见 `codex-navigation-spec.log`。
  2. `seed_migrate_test.go` `TestMergeLegacyServiceMenuReadFailurePropagatesAndRollsBack` —— 期望 merge 失败却得 `nil`：`Table.Scan` 走的是 GORM **Row** callback（`gorm:row`）而非 Query callback，故原 `injectNextReadError` 从未命中真实 Scan。见 `go-race-service.log:8`。

### 本轮改动（Claude 实现，未自我验收）

- `frontend/tests/smoke/business/service/ServiceCatalog.nav.spec.ts`：
  - 根标签断言改为 `ArgoCD 应用交付`；断言 K8s `应用发布` 前先点击 level-0 `Kubernetes` 展开（避开初始折叠）。
  - `rootLabels` 含平台工作台根，不再固定 `slice(0)`：以 CMDB 根为首定位业务块起始，断言业务 8 项在 seed 序下连续、且块后无任何业务根复发（系统治理随之在后）。
  - 30 入口 walk 改为**逐项命中 `BUSINESS_ENTRIES` 计划**（不再是遍历活树再静默 `continue`）：缺 seed 行即硬失败，不静默跳过；整条 walk `test.setTimeout(180000)`。
  - 第三个测试：服务侧栏 3 项与页内 tab 三项均真实 click 并校验 URL + `aria-selected` + 对应「新增」按钮，locator 均限定 `main` 避顶部同名 tab；back/forward 一致；三个深链均真正 reload；保留 404 收集。
  - 已逐字对齐 `probe-navigation.mjs` 的正文定位（工作台 `高频管理入口` 与 spec 现用文本一致）。
- `backend/modules/business/service/seed_migrate_test.go`：
  - `injectNextReadError` 改为 `db.Callback().Row().Before("gorm:row")`（Register/Remove 同步），使注入落到真实 Scan。
  - 新增 `TestMergeLegacyServiceMenuWriteFailureRollsBackWholeTx`：一次命中 Raw（`gorm:raw`）的写注入，令重挂 child 之后的角色绑定 INSERT 失败，断言整事务回滚且 custom child / tenant 绑定 / 两 parent 完整保留。

### 验证诚实记录

- 本 shell 无审批面，**playwright、node、go 命令均被权限系统拒绝**（无批准者），无法在本轮复跑上面两条回归。改动未在本轮「通过」意义上得到运行证据，**不宣称修复已由本人验证 PASS**；需 Codex 用真实 MySQL + 真实 8080/5173 复跑确认。
- task packet、custom/behaviour 枚举本轮未改；任务维持 awaiting-review / not self-approved。
