# T20 摘要 — 服务实例到部署的前端入口与完整分页

- 状态：completed（代码 + API + 渲染证据闭合；运行态终态链路按真实环境项延后，见末尾 gap）
- 候选：`831892d8` + 本包收尾工作树；分支 `feature/ops-t20-service-instance-deploy-entry`（基于 `origin/main` `5a1166c5`）
- 基线：`pantheon-base-v0.19.0`（`cc2fe737`）
- 来源发现：DEP-002

## 事实更正（开工核实）

- 计划中的 `ServiceInstanceList.tsx`（`pageSize: 100`、`pagination={false}`）**未在任何注册表登记**，实际入口是 `ServiceCatalog` 的“服务实例”tab（`/business/service-catalog/instances`），该 tab 早已使用服务端分页（smoke 覆盖 125 条、第 6 页编辑）。因此 S3 的“去掉 100 条硬上限”无对象可改，改为补齐该 tab 的筛选与部署入口。
- 分页上限方案选择 **(a)**：前端 `pageSize` 不超过 100，保持 service 包 `normalizePage` 现状，不改共享分页行为。
- 实例列表真实路由是 `/business/deploy/task`（单数），收尾时发现入口曾跳到 `/business/deploy/tasks` 并落到 404；已修正并由 smoke 断言 URL。

## 已交付

### S1 后端

- `TaskQuery` 增加 `ServiceID` / `ServiceInstanceID` 并下推 SQL（`ListTasks` 与导出共用过滤），`InstanceQuery` 增加 `Environment`。
- 收尾新增：创建服务关联任务时把 `serviceName` 快照进任务行（`serviceLinkTask` 校验通过后写入），任务详情与 CSV 导出不再依赖后续联表；该字段 `json:"-"`，不接受客户端输入。

### S2 前端 API 类型

- `service/api.ts`：`getServiceInstanceList` 补 `environment` / `targetType` / `status`；`ServiceInstanceRow` 补 `applicationId` / `applicationCode` / `businessScopeId`。
- `deploy/api.ts`：`DeployTaskRow` 补 `serviceId` / `serviceInstanceId` / `serviceName` / `serviceInstanceName`；`DeployTaskPayload` 补可选 service 关联字段。

### S3 实例列表筛选与分页

- 实例 tab 增加环境、目标类型、状态三个筛选（`allowClear`），筛选变化重置到第 1 页；`CatalogTabPanel` 新增 `filters` / `filtersActive`。
- 从 `?serviceId=` 进入时把实例查询初始化为该服务的实例，供任务详情回链使用。
- 新增“当前版本”列，实例行回显 `currentVersion`。

### S4 实例行“部署”入口

- 仅拥有 `business:deploy:task:create` 时显示；点击导航到 `/business/deploy/task?createFor=service-instance&…`。
- `DeployTaskList` 消费一次性创建意图：打开既有创建弹窗（未复制 1556 行表单），绑定服务/实例，预填业务域、模板与目标主机意图，生成默认任务名，然后清除意图参数。
- 收尾修复：意图标记改在实际执行时写入，避免 React StrictMode 二次调用清理定时器后弹窗永不打开。
- 提交时把 `serviceId` / `serviceInstanceId` 写入 payload；目标仍需操作员点“解析目标”确认。

### S5 回链

- 任务详情概览新增“所属服务”“服务实例”；所属服务可跳到 `/business/service-catalog/instances?serviceId=…`。
- 实例侧“最近任务/终态”未做（无 `:id` 详情路由，见 gap）。

### S6 i18n

- 新增 10 个 key（部署动作、三个筛选标签、实例启停状态、实例部署默认名、所属服务、服务实例、当前版本），五语言齐全；deploy 模块 `locales/*.json` 同步三个 task key。

## 验证（真实后端 + 真实浏览器）

后端：`go run ./cmd/server` 起在 127.0.0.1:8080，health 200（database/migrations/redis ok），MySQL/Redis 本机运行。

- `node scripts/run-smoke-suite.mjs … tests/smoke/business/service/service-pages.spec.ts` → **15 passed (1.9m)**，exit 0：
  - 125 条实例服务端分页、第 6 页编辑（既有）；
  - 桌面 1440 与 390 渲染状态、只读角色隐藏变更控件（既有）；
  - 新增：环境/目标类型/状态筛选把 `environment`/`targetType`/`status` 与 `page=1` 送到服务端；
  - 新增：无 `business:deploy:task:create` 时实例行不出现“部署”；
  - 新增：点实例行“部署”→ URL 带 `createFor=service-instance`/`serviceInstanceId`/`serviceId`/`businessScopeId` → 创建弹窗自动打开，任务名 `catalog-service-1 部署`、业务域 `catalog-scope`；刷新后意图已消费、弹窗不再打开；390×844 同样可打开且无横向溢出。
- `node scripts/run-smoke-suite.mjs … tests/smoke/business/deploy/deploy-api.spec.ts` → **5 passed (3.9s)**，exit 0：新增用例创建 scope/host/application/service/instance/package → 创建带 service 关联的任务 → `?serviceInstanceId=` 命中 → 详情 `serviceName` 为服务名快照 → preview 解析出 1 个目标（实例所在主机）→ `?serviceId=` 命中实例。
- `tests/smoke/business/deploy/deploy-pages.spec.ts`：整包 9 passed / 1 failed，失败用例单跑通过；失败是既有 `route-content-fallback` 竞态（`.page-container` 同时存在两个），与 T20 改动无关。
- `tsc -b` exit 0；改动文件 ESLint 0/0；`go build ./...`、`go vet ./modules/business/deploy/` exit 0。

截图（`.harness/evidence/2026-11-25-ops-service-instance-deploy-entry/screenshots/`）：`t20-deploy-entry-1440.png`、`t20-deploy-entry-390.png`、`t20-deploy-list-after-intent.png`、`service-catalog-pagination-edit.png`、`service-catalog-desktop.png`、`service-list-mobile.png`、`390x844-read-only.png`。

## 未关闭项（显式 gap）

1. **服务关联任务的手工结果上报被 attempt ownership 拒绝**（`business.deploy.taskHost.staleReport`）：存在 durable attempt 时无 worker claim 的上报一律视为过期。deploy-api 新用例因此在 preview 停止；未关联任务的手工 start→终态仍由既有用例覆盖。
2. **agent 执行器终态/日志**需要真实执行目标（SSH/Agent），按用户安排与 T19/T17 一起放到最后。
3. 部署任务无 `environment` 列 → 环境无法预填。
4. 实例无 `:id` 详情路由 → 实例侧“最近任务/终态”未做（以列表行入口 + 任务详情回链 + currentVersion 列替代）。
5. 主机以选择器意图种入，需操作员“解析目标”确认。
6. `deploy` 整包默认超时失败（既有，与本次无关）。

### 11 月 28 日基线复验

- 已移到 Ops main 875331ba 与锁定 Base v0.19.1。Go Lint 无新增问题，前端 lint、type-check、build，以及业务 overlay、干净重建和共享漂移检查通过。
- 首轮完整 Windows race 的可观测 HTTP 取消测试出现时序失败。隔离串行复验已通过。

- 同候选使用 versioned migrations、隔离 UTC MySQL 和 Redis，部署 API 5/5，服务目录与 CMDB 浏览器 smoke 25/25。二进制 SHA-256 b2e4a7e2f68bb656f9d156811b91740baad574b40becc51c2b84d26312e99153。桌面 1440px 和窄屏 390px 证据见 screenshots；相关布局和已覆盖权限状态可复验。

## 11 月 28 日交付复核

- 在 Windows MinGW CGO、隔离 UTC MySQL 和 Redis 上串行执行 `go test -race ./...`，78 个包通过，15 个包跳过，退出码 0。产品树 `730c46af902502d9abe1e05654ae63ccde7459f4`。[完整门禁回执](./windows-cgo-race-20261128.json)
- 同候选部署 API 5/5，服务目录与 CMDB 浏览器 smoke 25/25。已核看 390px 部署创建表单，表单在窄屏为单列，浏览器测试通过横向溢出检查。
- 实例到真实执行目标的启动、终态、日志和版本回写仍需验收。
- 本地证据已按候选绑定，暂留独立审批和 GitHub 门禁记录。
