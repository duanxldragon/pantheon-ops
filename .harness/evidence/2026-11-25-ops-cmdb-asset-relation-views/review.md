# Review — CMDB 主机与服务实例、业务域关联视图

## 结论

`approved with documented P2 follow-up`。主机与业务域两侧的关联视图已用真实后端与浏览器 smoke 验证；剩余的是聚合端点与目录过滤这类体验性跟进，不影响只读关联的正确性。

## Findings

1. **P1 fixed — 主机详情看不到承载它的服务实例。**
   `CmdbHostDetail` 原本只有基础信息、标签、已装组件与匹配分组，没有任何服务实例入口。新增只读「承载服务实例」表格：服务名链接到该服务的实例列表，并显示环境、当前版本、运行状态。数据由 `GET /business/service/instances?hostId=` 提供，复用服务目录的数据范围。

2. **P1 fixed — 主机详情看不到最近部署。**
   新增「最近部署」只读列表，按 `GET /business/deploy/tasks?hostId=` 取最近 5 条，任务名链接到任务详情。过滤用 `JSON_CONTAINS(target_ids, ?)` 在 SQL 内完成，任务可见性仍由既有 host 数据范围把关，域外主机对 dept 调用方不泄露任何任务（新增后端用例覆盖）。

3. **P1 fixed — 业务域详情没有服务目录口径。**
   新增 `GET /business/service/summary?businessScopeId=`，按调用方数据范围统计应用/服务/实例，并登记到 `business:service:list|view` 路由；`BizScopeDetail` 增加「服务目录统计」卡片，三个计数可跳到目录对应 tab。

4. **P1 fixed — 业务域删除的应用引用没有回归测试。**
   `Delete` 早已会数 `biz_application` / `biz_k8s_cluster` / `biz_deploy_task`，但测试没覆盖。新增用例证明应用引用时返回 `business.bizscope.inUse`，软删除应用后可删且行消失；并发删除用例证明两个 goroutine 只有一个成功、不残留半删状态。

5. **P2 accepted-gap — 主机详情多两次读请求。**
   没有新增聚合端点：实例与任务各读一次列表（pageSize 50 / 5）。主机详情页本来就按区块异步加载，当前体量可接受；聚合端点留作跟进。

6. **P2 accepted-gap — 业务域计数跳转未按业务域过滤。**
   目录 tabs 尚不接受 `businessScopeId`，三个按钮落到全局列表。要真正“带域过滤”需要目录侧新增过滤参数，属另一个包的契约变更。

7. **P2 fixed — 无权限时区块空白。**
   服务实例块按 `business:service:list|view` 显示、部署块按 `business:deploy:task:view` 显示，无权限时整块隐藏；业务域统计块在无目录读权限时隐藏。

8. **P2 fixed — i18n 与渲染证据缺失。**
   新增 15 个 key 五语言齐全（cmdb 模块 locales 同步 zh/en，bizscope 同步五语言）；`npm run build` 的 i18n 门禁通过。`cmdb-pages.spec.ts` 10 passed，主机详情与业务域统计在 1440 与 390 下都有截图。

## 验证要点

- `go test ./modules/business/bizscope/ ./modules/business/service/ ./modules/business/cmdb/...` 全绿（6 个包）。
- 新增：`TestListInstancesFiltersByHostAndScopeSummary`、`TestListTasksFiltersByHostWithinDataScope`、`TestBizScopeDeleteRejectsApplicationReference`、`TestBizScopeDeleteIsSingleWinnerUnderConcurrency`。
- `tsc -b`、改动文件 ESLint、`npm run build`（含全部 prebuild 门禁）exit 0。
- `tests/smoke/business/cmdb/cmdb-pages.spec.ts` 10 passed（37.8s），截图落在 `screenshots/`。
- 未触碰 `backend/modules/system/**`、`backend/internal/**`、`frontend/src/core/**`、`frontend/src/api/request.ts`、`foundation-release.lock.json`。

## Machine Readable

```json
{
  "taskId": "2026-11-25-ops-cmdb-asset-relation-views",
  "reviewer": "primary agent self-review (no independent review on this candidate)",
  "verdict": "approved with documented P2 follow-up",
  "decision": "approved with documented P2 follow-up",
  "scope": "backend: hostId filters for instances and tasks, business-scope catalog summary; frontend: host detail relation cards, bizscope catalog counts; bizscope delete reference tests; i18n and rendering evidence",
  "linkage": {
    "taskManifest": ".harness/tasks/2026-11-25-ops-cmdb-asset-relation-views/manifest.json",
    "taskPacket": "docs/harness/tasks/2026-11-25-ops-cmdb-asset-relation-views.task.md",
    "evidence": ".harness/evidence/2026-11-25-ops-cmdb-asset-relation-views/commands.json",
    "reviewFile": ".harness/evidence/2026-11-25-ops-cmdb-asset-relation-views/review.md",
    "changeRef": "none",
    "planRefs": [".harness/tasks/2026-11-20-ops-review-remediation/plan.md"]
  },
  "findings": [
    {
      "id": "T21-R1",
      "severity": "P1",
      "status": "fixed",
      "summary": "host detail had no view of the service instances it hosts",
      "verification": "service instance hostId filter + cmdb-pages smoke"
    },
    {
      "id": "T21-R2",
      "severity": "P1",
      "status": "fixed",
      "summary": "host detail had no recent deployment view",
      "verification": "task hostId JSON_CONTAINS filter + dept-scope leak test + cmdb-pages smoke"
    },
    {
      "id": "T21-R3",
      "severity": "P1",
      "status": "fixed",
      "summary": "business scope detail had no catalog counts",
      "verification": "/business/service/summary + bizscope summary card smoke"
    },
    {
      "id": "T21-R4",
      "severity": "P1",
      "status": "fixed",
      "summary": "business scope delete application-reference guard had no regression test",
      "verification": "TestBizScopeDeleteRejectsApplicationReference + concurrent delete test"
    },
    {
      "id": "T21-R5",
      "severity": "P2",
      "status": "accepted-gap",
      "summary": "host detail issues two extra list reads instead of an aggregate endpoint",
      "verification": "page review; no aggregate API added"
    },
    {
      "id": "T21-R6",
      "severity": "P2",
      "status": "accepted-gap",
      "summary": "catalog tiles open unfiltered tabs because the catalog has no businessScopeId filter",
      "verification": "ServiceCatalog query parameters review"
    },
    {
      "id": "T21-R7",
      "severity": "P2",
      "status": "fixed",
      "summary": "relation blocks would render empty for roles without catalog permissions",
      "verification": "hasPerm gating in both detail pages"
    },
    {
      "id": "T21-R8",
      "severity": "P2",
      "status": "fixed",
      "summary": "missing i18n keys and rendered evidence",
      "verification": "15 keys in five locales, build gates, cmdb smoke 10 passed with 1440/390 screenshots"
    }
  ]
}
```

## 11 月 28 日候选复核

本文件较早结论对应历史候选。本轮做了代码自审与浏览器验证，独立审批待 GitHub PR 记录。
