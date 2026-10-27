# O6 Business Dashboard Widgets

- Priority: `P1`
- Layer: `business/*` dashboard overlay
- Status: `blocked`
- Base Dependency: B4 dashboard semantic slots
- Business Dependency: bounded summary endpoints

## Outcome

在平台 Dashboard 注册少量高价值业务摘要，让用户看到异常和待处理对象并能带上下文下钻。

## In Scope

- deployment status summary and failed-task attention queue。
- K8s cluster/workload abnormal summary。
- alert/datasource health summary。
- recent business resources/tasks where Base contract permits。
- owner、permission、cleanup、freshness、query budget 和 deep-link filters。

## Out Of Scope

- 不复制完整业务表格、日志或查询工作台。
- 不在 Ops 修改 Dashboard registry 核心。
- 不提供无限制自定义布局或高频轮询。

## Expected Files

- business widget registration and components under `frontend/src/modules/business/**`。
- business overlay registry、i18n、permission and tests。
- provider-owned bounded summary endpoints if current APIs cannot satisfy budget。

## Acceptance

1. 无权限 widget 不请求、不渲染、不泄露标题/计数。
2. 每个计数显示时间范围/更新时间，stale/error 独立，不阻断 Dashboard。
3. 点击下钻保留 BizScope、时间范围和对应筛选。
4. endpoint 有固定上限和缓存/超时策略，不出现 N+1 或高频 fan-out。
5. mobile 顺序符合异常优先，长标题/数值不改变稳定尺寸。

## Evidence

- registry/overlay：registration、permission、cleanup、unknown kind tests。
- backend：summary scope、budget、timeout、empty/partial behavior。
- Playwright：mixed permissions、stale/error isolation、deep links、mobile order。
- screenshots：desktop/mobile x populated/empty/partial/forbidden x light/dark。

## Gates

- Base lock、新聚合 schema/API、首页信息优先级和最终维护者视觉验收。
