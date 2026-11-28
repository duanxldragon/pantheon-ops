# CMDB 主机与服务实例、业务域关联视图

- Status: completed（代码 + 后端测试 + 真实浏览器 smoke + 1440/390 渲染证据）
- Updated: 2026-11-26
- Baseline: 0.19.0（`origin/main` `5a1166c5`，foundation `cc2fe737`），本包基于 T20 提交 `04c230dc`
- Owner: CMDB 与业务域负责人
- Priority: P1
- Activation: core
- Dependencies: 无（协同 2026-11-20-ops-service-deploy-business-closure）

## Source

NEW-02：主机详情只有资产信息与已装组件，看不到承载的服务实例；业务域详情只有主机数量，没有应用/服务/实例口径；bizscope 的引用保护缺少应用侧反例。

## Work Breakdown

- [x] S1 后端读取接口：`InstanceQuery.HostID`、`TaskQuery.HostID`（`JSON_CONTAINS`）、`GET /business/service/summary?businessScopeId=`（按数据范围计数，登记权限路由）
- [x] S2 一致性：环境与业务域的对应仍由 service 侧准入负责，cmdb 不复制规则；关联视图只读
- [x] S3 bizscope 反例测试：应用引用阻止删除（软删后可删）、并发删除单赢家；既有数据范围测试继续通过
- [x] S4 主机详情：「承载服务实例」表格 + 「最近部署」列表，无权限整块隐藏
- [x] S5 业务域详情：「服务目录统计」卡片（应用/服务/实例，可跳目录）
- [x] S6 受限角色：域外主机对 dept 调用方返回空、admin 可见（新增 deploy 用例覆盖）
- [x] S7 证据：`go test` 三包全绿、cmdb smoke 10 passed、1440/390 截图

## 验证记录（2026-11-26，真实 MySQL + 真实后端 + 浏览器）

- `go build ./...`、`go vet`（service/deploy/bizscope）exit 0。
- `go test -count=1 ./modules/business/bizscope/ ./modules/business/service/ ./modules/business/cmdb/...` 全绿：bizscope 7.7s、service 30.0s、cmdb 8.4s、cmdb/group 3.8s、cmdb/host 16.5s、cmdb/label 5.0s。
- 新增用例：
  - `TestListInstancesFiltersByHostAndScopeSummary`（host 55/56/未知、scope 10 计数 1/1/2、scope 11 全 0）；
  - `TestListTasksFiltersByHostWithinDataScope`（两台主机交叉任务、域外主机 dept 视图为空、admin 可见）；
  - `TestBizScopeDeleteRejectsApplicationReference`（inUse → 软删后成功且行消失）；
  - `TestBizScopeDeleteIsSingleWinnerUnderConcurrency`（恰好一个成功、无残留行）。
- `tsc -b` exit 0；改动文件 ESLint 0/0；`npm run build` 含全部 prebuild 契约门禁通过。
- `tests/smoke/business/cmdb/cmdb-pages.spec.ts` 10 passed（37.8s），新用例断言主机详情「承载服务实例」「最近部署」内容、业务域详情「服务目录统计」，390px 无横向溢出。
- 截图：`cmdb-host-relations-1440.png`、`cmdb-host-relations-390.png`、`bizscope-catalog-summary-1440.png`、`cmdb-host-detail.png`、`bizscope-list.png`。

## Known Gaps

- 主机详情为两个额外读请求（实例 + 部署），没有聚合端点。
- 实例无显示名，主机卡片以服务名 + 版本 + 状态呈现。
- 业务域统计的三个跳转落到未按业务域过滤的目录 tab。
- “跨租户”以部门数据范围验证（测试库 bizscope 表无 tenant 列）。
- deploy 整包默认超时问题未在本包处理（既有，见 T20 STATUS）。


## 11 月 28 日交付候选

- 当前为验证中。独立复核、完整 Windows CGO race 及 GitHub 收口按新候选留证。

- 11 月 28 日候选串行完整 CGO race 通过，仍按各任务验收项收尾。[验证汇总](../../evidence/2026-11-25-ops-cmdb-asset-relation-views/summary.md)
