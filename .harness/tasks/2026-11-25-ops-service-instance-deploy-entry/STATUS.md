# 服务实例到部署的前端入口与完整分页

- Status: completed（代码 + API + 真实浏览器 smoke + 1440/390 渲染证据闭合；运行态终态链路按真实环境项延后）
- Updated: 2026-11-26
- Baseline: 2026-11-26 迁移到 0.19.0 基线（`origin/main` `5a1166c5`，foundation `cc2fe737`），本包提交 `831892d8` + 收尾工作树
- Owner: 服务部署闭环负责人
- Priority: P0
- Activation: core
- Dependencies: 无（协同 2026-11-20-ops-service-deploy-business-closure、2026-11-20-ops-deploy-credential-ownership-and-ui）

## Source

2026-11-25 站在用户主路径复核新增。事实依据：评审时 frontend/src/modules/business/deploy 下 serviceId/serviceInstanceId 零引用；ServiceInstanceList.tsx 以 getServiceInstanceList({page:1,pageSize:100}) 加载且 pagination={false}，仅有“同步”操作。后端 service_link.go、ApplyServiceInstanceState 已就绪并有测试。

## Next Atomic Action

无（本包关闭）。延后项：agent 执行器下的服务关联任务终态/日志，按用户安排与 T19/T17 的真实环境项一起处理。

## Work Breakdown

- [x] S1 后端：TaskQuery 增加 serviceId/serviceInstanceId 过滤并下推到 SQL（deploy_dto.go、deploy_service.go ListTasks，导出复用同一过滤），service 包 InstanceQuery 增加 environment 过滤；补带数据范围的反例测试。收尾追加：服务关联任务创建时把 `serviceName` 快照进任务行（`json:"-"`，不接受客户端输入）
- [x] S2 前端 API：`getServiceInstanceList` 参数补 `environment`/`targetType`/`status`，`ServiceInstanceRow` 补 `applicationId`/`applicationCode`/`businessScopeId`；`DeployTaskRow` 补 service 关联字段，`DeployTaskPayload` 补可选 service 关联字段
- [x] S3 实例列表：核实后 `ServiceInstanceList.tsx` 未注册，真实页面是 `ServiceCatalog` 实例 tab（已有服务端分页）。补环境/目标类型/状态筛选、`?serviceId=` 初始过滤与“当前版本”列；分页上限选择方案 **(a)**
- [x] S4 部署入口：实例行“部署”按 `business:deploy:task:create` 显示，导航到 `/business/deploy/task?createFor=service-instance&…`；`DeployTaskList` 消费一次性意图打开既有创建弹窗，预填业务域与目标主机意图、默认任务名并写入 service 关联。收尾修复了路由单复数（曾跳 404）与 StrictMode 下意图被吞的问题
- [x] S5 回链：任务详情展示所属服务/实例，所属服务可跳转到该服务的实例列表；实例→任务由 S4 提供。实例详情页不存在（gap）
- [x] S6 异常路径与 i18n：五语言 10 个新 key（含 deploy 模块 locales 三个 task key）；无权限时入口隐藏由 smoke 覆盖。stale 目标提示与实例并发删除的可恢复提示未实现（gap）
- [x] S7 测试与证据：`service-pages` 15 passed、`deploy-api` 5 passed、1440/390 截图落 `.harness/evidence/2026-11-25-ops-service-instance-deploy-entry/`

## 2026-11-26 收尾验证（真实后端 + 真实浏览器）

环境：`go run ./cmd/server` 监听 127.0.0.1:8080，health 200（database/migrations/redis ok）；本机 MySQL + Redis 6379；Playwright 走 `scripts/run-smoke-suite.mjs` 起 Vite smoke server。

- `service-pages.spec.ts` → **15 passed (1.9m)**，exit 0。新增三条用例：
  1. 环境/目标类型/状态筛选把 `environment`/`targetType`/`status` 与 `page=1` 送到服务端；
  2. 无 `business:deploy:task:create` 时实例行不出现“部署”，也不出现变更控件；
  3. 点实例行“部署”→ URL 带 `createFor=service-instance`/`serviceId`/`serviceInstanceId`/`businessScopeId` → 创建弹窗自动打开，任务名 `catalog-service-1 部署`、业务域 `catalog-scope`；刷新后意图已消费不再打开；390×844 同样打开且无横向溢出。
- `deploy-api.spec.ts` → **5 passed (3.9s)**，exit 0。新增用例覆盖 scope/host/application/service/instance/package → 创建带 service 关联的任务 → `?serviceInstanceId=` 命中 → 详情 `serviceName` 为快照名 → preview 解析 1 个目标（实例所在主机）→ `?serviceId=` 命中实例。
- `deploy-pages.spec.ts` → 9 passed / 1 failed，失败用例单跑通过；失败是既有 `route-content-fallback` 竞态（`.page-container` 同时存在两个），与本次改动无关。
- `go build ./...`、`go vet ./modules/business/deploy/`、`tsc -b`、改动文件 ESLint 全部 exit 0。
- 截图：`t20-deploy-entry-1440.png`、`t20-deploy-entry-390.png`、`t20-deploy-list-after-intent.png`、`service-catalog-pagination-edit.png`、`service-catalog-desktop.png`、`service-list-mobile.png`、`390x844-read-only.png`。

## Known Gaps

- **服务关联任务的手工结果上报被 attempt ownership 拒绝**（`business.deploy.taskHost.staleReport`）：存在 durable attempt 且无 worker claim 时上报视为过期。deploy-api 新用例在 preview 停止；未关联任务的手工 start→终态仍由既有用例覆盖。
- **agent 执行器终态/日志**需要真实执行目标，属真实环境项，按用户安排与 T19/T17 一起延后。
- 契约：部署任务无 `environment` 列，环境无法预填；实例无 `:id` 详情路由，“实例侧最近任务/终态”未做。
- 交互：主机以选择器意图种入，需操作员“解析目标”确认。
- **deploy 整包：默认超时下无法通过，放宽超时后通过**（2026-11-25 记录，`-timeout 3000s` 下 251 PASS / 0 FAIL / 2 SKIP，耗时约 19 分钟）。2 个 SKIP 需要真实 SSH 目标，未验证。本次未重跑整包。
- `deploy-pages` 整包存在 route-content-fallback 竞态（单测通过、整包偶发）。


## 11 月 28 日交付候选

- 当前为验证中。独立复核、完整 Windows CGO race 及 GitHub 收口按新候选留证。

- 11 月 28 日候选串行完整 CGO race 通过，仍按各任务验收项收尾。[验证汇总](../../evidence/2026-11-25-ops-service-instance-deploy-entry/summary.md)
