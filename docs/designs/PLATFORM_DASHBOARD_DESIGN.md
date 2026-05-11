# 平台仪表盘设计

更新时间：2026-05-01

类型：Design
归属层：platform
状态：Active

## 1. 归属边界

平台仪表盘属于 `platform` 层，不属于单一 `system/auth`、`system/iam`、`system/org` 或 `system/config` 子域。

补充说明：

- `platform` 在这里是**逻辑层**，不是必须存在的物理目录；
- 当前代码已将 dashboard 物理放置在顶层 `backend/modules/dashboard/` 与 `frontend/src/modules/dashboard/`；
- 即使物理目录不再叫 `platform`，它的职责仍然是平台聚合层。

原因：

- 仪表盘是跨域聚合视图；
- 它读取多个系统子域的统计结果，但不反向拥有这些子域；
- 后续若增加业务域摘要卡片，也应继续挂在 `platform` 聚合层，而不是塞回某个系统模块。

## 2. 当前接口

- `GET /api/v1/platform/dashboard/summary`

鉴权策略：

- 只要求已登录；
- 不依赖额外页面权限点；
- 作为首页工作台与平台概览卡片的数据来源。

## 3. 当前统计项

- `totalUsers`
- `enabledUsers`
- `totalRoles`
- `totalDepts`
- `totalPosts`
- `totalDictTypes`
- `totalSettings`
- `visibleMenuCount`
- `activeSessionCount`
- `loginSuccessCount`
- `loginFailureCount`
- `todayOperationCount`
- `lastSuccessfulLoginAt`
- `recentLogins`
- `periodDays`

说明：

- `visibleMenuCount` 统计 `type <> 'F'` 且 `is_visible = 1` 的导航节点；
- `activeSessionCount` 统计未吊销且 refresh 未过期的会话；
- 登录成功/失败趋势默认统计最近 `7` 天；
- 最近登录活动用于首页快速感知安全状态，不替代完整审计页。
- `todayOperationCount` 用于首页表达“今日平台活跃度”，不替代完整操作日志页。
- `lastSuccessfulLoginAt` 用于轻量表达最近安全态势，不替代安全中心详情。

## 4. 前端呈现规范

首页仪表盘采用“轻工作台”模式：

- 第一行展示关键指标卡片，最多保留 4 个一级指标；
- 第二行展示安全态势、待关注事项与快捷入口；
- 第三行展示按 `iam / org / config / audit` 聚合的域概览；
- 下方展示最近登录活动或最近操作摘要；
- 不在首页堆过多图表，避免后台首页视觉噪音与 AI 模板感；
- 不把平台首页做成同尺寸卡片墙，卡片必须有明确的信息层级和真实数据来源。
- 同一 KPI 不得在第一屏多个区域重复出现，例如“成功率 / 活跃会话 / 今日操作”只能有一个主表达位置；
- Hero、Attention、Overview 三块区域必须职责分离：Hero 负责总述，Attention 负责需要处理的事项，Overview 负责域级摘要；
- 快捷入口上限保持 `6` 个，超过后回到动态菜单与全局搜索，不继续扩写工作台首屏；
- 第一屏优先呈现“需要管理员行动的内容”，解释型说明和重复性平台统计后移。

推荐布局：

```text
DashboardPage
  ├── StatusStrip
  ├── PrimaryActions
  ├── AttentionPanel
  ├── DomainOverview
  └── RecentActivity
```

## 5. 后续演进

后续可以按模块化方式增量加入：

- 待办与公告卡片
- 审计趋势图
- 系统设置变更提醒
- 业务模块注册的工作台卡片

约束：

- 新增卡片必须声明归属层级；
- 跨域聚合继续放在 `platform`；
- 单域详情不要反向塞进平台层，应该通过跳转进入对应系统域或业务域页面。
- 业务模块卡片不得硬编码在平台工作台中，必须通过后续 widget 注册机制接入。
- 当前前端已收口为注册式 widget 清单；后续新增卡片必须声明：
  - `sourceDomain`
  - `permission`
  - `path`
  - `cleanupPolicy`
  - 若属于 `business/*`，还必须声明 `registrationOwner`
- `business/*` widget 的清理策略必须为“模块退出时移除”，不能把失效卡片长期留在平台首页。

## 6. 当前注册契约

当前前端工作台已收口到 `ModuleConfig.dashboardWidgets`：

- 每个模块在自己的 `index.ts` 中声明可挂入工作台的 widget；
- `platform` 层只负责从已注册模块聚合 widget，不再在 `dashboard/widgets.tsx` 中维护静态大列表；
- `system/*` 与 `business/*` 通过同一注册面接入，但 `business/*` 必须额外声明 `registrationOwner` 与 `remove_with_source_module` 清理策略；
- widget 可见性仍由导航可达性与权限共同决定，`platform` 不绕过 `system/*` 的权限边界。
