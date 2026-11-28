# Review — 服务实例到部署的前端入口与完整分页

## 结论

`approved with documented P2 follow-up`。代码、API、真实浏览器 smoke 与 1440/390 渲染证据已齐备；服务关联任务的运行态终态链路依赖真实执行目标（P1 运行态 gap），按用户安排与 T19/T17 一起延后，已逐条记录，未以“已验证”冒充。

## Findings

1. **P1 fixed — 计划前提与代码不符。**
   计划要求“去掉 `ServiceInstanceList` 的 `pageSize=100` 与 `pagination={false}`”，但该组件未在 `businessOverlayComponentRegistry` 登记，实际页面是 `ServiceCatalog` 的实例 tab，且早已服务端分页。已改为对真实页面补筛选与部署入口，并在 summary 记录更正。

2. **P1 fixed — 实例到部署没有入口。**
   实例行新增“部署”动作，按 `business:deploy:task:create` 显示，通过一次性路由意图复用既有创建弹窗，创建 payload 带 `serviceId` / `serviceInstanceId`。smoke 断言了 URL 意图参数、弹窗自动打开、预填任务名与业务域、刷新后意图不再触发。

3. **P1 fixed — 入口曾跳到不存在的路由。**
   收尾时发现入口指向 `/business/deploy/tasks`，真实注册路径是 `/business/deploy/task`（菜单 seed 与组件注册表），点击会落到 404 页面。已修正，并由 smoke 的 URL 断言与弹窗可见性覆盖。

4. **P1 fixed — 一次性意图在 StrictMode 下失效。**
   原实现先写 `createIntentHandled` 标记再起定时器，React StrictMode 二次调用 effect 时清理了定时器、二次调用又被标记拦住，弹窗永不打开。改为在定时器回调内写标记，smoke 在真实 dev server 下验证通过。

5. **P1 accepted-gap — 服务关联任务的手工结果上报被拒。**
   `requireHostWriter` 在存在 durable attempt 且没有 worker claim 时一律返回 `business.deploy.taskHost.staleReport`。因此 deploy-api 新用例在 preview 处停止（断言解析出实例所在主机 1 个目标），未断言 start→终态；未关联任务的手工 start→终态仍由既有用例覆盖。agent 执行器下的终态与日志需要真实执行目标，属真实环境项。

6. **P1 accepted-gap — 目标主机只预填意图，不预解析。**
   提交要求 `targetSelection` + `scopeVersion` 与业务域一致，预解析要承担并发/过期语义。本增量把主机作为 `initialHostIds` 种入选择器，由操作员点“解析目标”确认；没有伪造服务端校验结果。

7. **P2 fixed — 任务详情看不到所属服务/实例。**
   前端类型补齐 `serviceId` / `serviceInstanceId` / `serviceName` / `serviceInstanceName`，概览展示且所属服务可跳到该服务的实例列表；收尾又把 `serviceName` 快照进任务行，避免服务改名后详情变空。

8. **P2 fixed — 实例列表看不到当前版本。**
   新增“当前版本”列回显 `currentVersion`，配合任务详情回链满足“回显实例当前版本与状态”的可读部分。

9. **P2 accepted-gap — 部署任务无 environment 字段。**
   `CreateTaskRequest` 与任务表都没有环境列，环境无法预填；不新增迁移，只记录契约缺口。

10. **P2 accepted-gap — 无实例详情页。**
    实例只有列表路由，没有 `:id` 详情，因此“实例详情回显最近任务与终态”未实现；当前是列表行发起部署 + 任务详情回链 + 当前版本列。

11. **P2 fixed — i18n 缺失。**
    新增 10 个 key 覆盖五语言；deploy 模块 `locales/*.json` 同步三个 task key。`check:i18n-hardcode` 与生成作用域门禁在 build 中通过。

12. **P2 fixed — 无浏览器证据。**
    已在真实后端（MySQL/Redis 本机，health 200）上跑 `service-pages`（15 passed）与 `deploy-api`（5 passed），并落 1440/390 截图。`deploy-pages` 整包 9 passed / 1 failed，失败用例单跑通过，属既有 route-content-fallback 竞态，与本次改动无关。

## 验证要点

- `go build ./...`、`go vet ./modules/business/deploy/`、`tsc -b`、改动文件 ESLint 全部 exit 0。
- `service-pages.spec.ts` 15 passed（含 125 条分页第 6 页、桌面/390、只读角色、三条 T20 新用例）。
- `deploy-api.spec.ts` 5 passed（含服务关联任务的创建/过滤/详情/预览/实例过滤）。
- 截图见 `screenshots/`：`t20-deploy-entry-1440.png`、`t20-deploy-entry-390.png`、`t20-deploy-list-after-intent.png`、`service-catalog-pagination-edit.png`、`service-catalog-desktop.png`、`service-list-mobile.png`、`390x844-read-only.png`。
- 未触碰 `backend/modules/system/**`、`frontend/src/core/**`、`frontend/src/api/request.ts`、`foundation-release.lock.json`。

## Machine Readable

```json
{
  "taskId": "2026-11-25-ops-service-instance-deploy-entry",
  "reviewer": "primary agent self-review (no independent review on this candidate)",
  "verdict": "approved with documented P2 follow-up",
  "decision": "approved with documented P2 follow-up",
  "scope": "frontend: service-instance filters, instance-to-deploy entry via route intent, task-to-instance traceability, currentVersion column, i18n; backend: service-name snapshot on linked tasks",
  "linkage": {
    "taskManifest": ".harness/tasks/2026-11-25-ops-service-instance-deploy-entry/manifest.json",
    "taskPacket": "docs/harness/tasks/2026-11-25-ops-service-instance-deploy-entry.task.md",
    "evidence": ".harness/evidence/2026-11-25-ops-service-instance-deploy-entry/commands.json",
    "reviewFile": ".harness/evidence/2026-11-25-ops-service-instance-deploy-entry/review.md",
    "changeRef": "none",
    "planRefs": [".harness/tasks/2026-11-20-ops-review-remediation/plan.md"]
  },
  "findings": [
    {
      "id": "T20-R1",
      "severity": "P1",
      "status": "fixed",
      "summary": "planned ServiceInstanceList target is unregistered; real page already paginates server-side",
      "verification": "registry grep + service smoke (125 rows, page 6)"
    },
    {
      "id": "T20-R2",
      "severity": "P1",
      "status": "fixed",
      "summary": "no entry from a service instance to a deploy task",
      "verification": "service-pages smoke: URL intent + prefilled modal at 1440/390"
    },
    {
      "id": "T20-R3",
      "severity": "P1",
      "status": "fixed",
      "summary": "entry navigated to /business/deploy/tasks which is not a registered route (404)",
      "verification": "service-pages smoke asserts /business/deploy/task and modal visibility"
    },
    {
      "id": "T20-R4",
      "severity": "P1",
      "status": "fixed",
      "summary": "one-shot create intent never opened under React StrictMode",
      "verification": "service-pages smoke on real dev server; intent consumed once after reload"
    },
    {
      "id": "T20-R5",
      "severity": "P1",
      "status": "accepted-gap",
      "summary": "manual host report for a service-linked task is rejected as a stale attempt; terminal state needs a real executor",
      "verification": "deploy-api smoke stops at preview (1 resolved target); unlinked manual start/terminal covered by existing case"
    },
    {
      "id": "T20-R6",
      "severity": "P1",
      "status": "accepted-gap",
      "summary": "target host is seeded as intent only; operator must resolve before submit",
      "verification": "code review of DeployTargetSelector contract"
    },
    {
      "id": "T20-R7",
      "severity": "P2",
      "status": "fixed",
      "summary": "task detail could not show or link the owning service/instance",
      "verification": "deploy-api smoke asserts snapshotted serviceName and serviceInstanceId in detail"
    },
    {
      "id": "T20-R8",
      "severity": "P2",
      "status": "fixed",
      "summary": "instance list did not expose currentVersion",
      "verification": "ServiceCatalog column + service smoke rendering"
    },
    {
      "id": "T20-R9",
      "severity": "P2",
      "status": "accepted-gap",
      "summary": "deploy task has no environment field, so environment cannot be prefilled",
      "verification": "CreateTaskRequest and deploy task model inspection"
    },
    {
      "id": "T20-R10",
      "severity": "P2",
      "status": "accepted-gap",
      "summary": "no service-instance detail route, so recent tasks/terminal state on an instance is not implemented",
      "verification": "router and registry inspection"
    },
    {
      "id": "T20-R11",
      "severity": "P2",
      "status": "fixed",
      "summary": "missing i18n keys for the new filters, entry, traceability and version labels",
      "verification": "check:i18n-hardcode and check:i18n-generated-scope passed in npm run build"
    },
    {
      "id": "T20-R12",
      "severity": "P2",
      "status": "fixed",
      "summary": "no browser smoke or rendered evidence on this candidate",
      "verification": "service-pages 15 passed, deploy-api 5 passed, 1440/390 screenshots saved"
    }
  ]
}
```

## 11 月 28 日候选复核

本文件较早结论对应历史候选。本轮做了代码自审与浏览器验证，独立审批待 GitHub PR 记录。
