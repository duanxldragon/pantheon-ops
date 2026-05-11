# 前端页面模板规范

更新时间：2026-04-17

本文定义 Pantheon Base 的标准页面模板，用于指导后续所有前端页面实现。

目标是：

- 不让每个页面各写一套布局
- 不让 AI 每次生成不同风格
- 不让后台页面变成随机拼接的 CRUD
- 让企业级后台有一致的信息架构和交互节奏

本文必须和以下文档一起使用：

- `docs/designs/FRONTEND_UI_SPEC.md`
- `docs/designs/MODULE_CONTRACT.md`
- `docs/designs/PERMISSION_MODEL.md`
- `docs/designs/ERROR_CODE_AND_I18N.md`

## 1. 页面类型总览

Pantheon Base 前端页面分为以下几类：

| 类型 | 用途 | 示例 |
| :--- | :--- | :--- |
| `ListPage` | 列表、筛选、表格、分页 | 用户、角色、岗位、权限策略 |
| `TreePage` | 树结构管理 | 部门、菜单 |
| `DetailPage` | 详情展示 | 用户详情、角色详情 |
| `FormPage` | 独立创建/编辑流程 | 高复杂度配置 |
| `ConfigPage` | 系统配置 | 系统设置、上传配置 |
| `DashboardPage` | 工作台 | 仪表盘 |
| `AuthPage` | 认证入口 | 登录、二次验证 |
| `ProfilePage` | 当前用户自助 | 个人资料、安全设置 |
| `StatePage` | 全局状态页 | 403、404、500、网络错误 |

## 2. 通用页面结构

所有业务页面都应尽量采用：

```text
Page
  ├── PageHeader
  ├── PageToolbar / ActionBar
  ├── PageContent
  └── PageState
```

## 2.1 PageHeader

包含：

- 标题
- 描述
- 面包屑（由布局层负责时可不重复）
- 右侧主操作

规则：

- 标题使用 i18n key
- 描述可选，但复杂页面建议写
- 主操作最多 2 个，更多操作放进 dropdown
- 如果布局层已经提供面包屑，页面内容区不要再重复渲染同名路径标题
- 如果页面在 `PageHeader` 之后还有 hero / summary 卡片，该卡片只承载摘要、说明和 KPI，不再重复页面主标题

## 2.2 PageContent

承载页面主体。

规则：

- 页面主体使用 surface 背景
- 区块之间使用 `16 / 24px` 间距
- 不在页面里随意写大面积 inline style

## 2.3 PageState

所有页面必须考虑：

- loading
- empty
- error
- forbidden
- submitting

## 3. ListPage 模板

适用于：

- 用户管理
- 角色管理
- 岗位管理
- 权限策略
- 字典项

## 3.1 标准结构

```text
ListPage
  ├── PageHeader
  │   ├── title
  │   ├── description
  │   └── primaryAction
  ├── FilterPanel
  ├── TableCard
  │   ├── Table
  │   └── Pagination
  └── CreateEditModal / Drawer
```

## 3.2 FilterPanel

规则：

- 默认放在表格上方
- 简单筛选不超过一行
- 超过 4 个筛选项时支持折叠
- 查询按钮在右侧，顺序为：重置、查询
- 筛选项必须能随分页、排序联动
- 筛选区必须使用共享 `FilterPanel`，不得在页面内直接手写 `Card + Form` 复制筛选样式
- 筛选区 body padding 必须使用平台变量 `--shell-filter-body-padding`
- 筛选控件高度必须使用 `--shell-filter-control-min-height`，Input / Select / TreeSelect / Picker 不得单页私自压缩
- 筛选项底部间距必须使用 `--shell-filter-form-item-margin-bottom`，label 底部间距必须使用 `--shell-filter-label-padding-bottom`
- 查询/重置按钮必须放在 `filter-panel__action-item` 内，按钮高度与筛选控件一致，桌面端右对齐，移动端纵向撑满
- 禁止通过 `.xxx-page .filter-panel ...` 覆盖 padding、控件高度、label 间距；视觉密度只能通过 `data-pantheon-density` 的平台 token 调整

## 3.3 TableCard

规则：

- 表格 loading 使用表格自身 loading
- 搜索无结果要显示“搜索无结果”空态
- 初始无数据要显示“暂无数据”空态
- 操作列固定右侧
- 操作列顺序：查看、编辑、删除、更多
- 表格必须放入统一 `TableCard` 语义容器，系统页默认使用 `page-panel system-list__table-card`
- `TableCard` 左右留白必须统一使用平台变量 `--shell-table-card-padding`，页面不得为某个列表页单独压缩左右 padding
- `AppTable` 的 `.arco-table-container` 必须使用 `--radius-md` 圆角，并由容器裁切表头背景，避免同一批系统页出现圆角和方角混用
- 表头背景必须是中性 `--panel-muted`，不得用品牌色 `color-mix`、渐变或主题色轻染
- 固定列产生的 Arco 阴影伪元素不得显示为表头左上角或右侧的“渐变边框”，平台层应统一关闭或替换为中性边界
- 分页区左右 padding 与表格内容区对齐，不得在单页内单独偏移

## 3.3.1 列表操作条

列表页头部操作和表格批量/治理操作必须统一使用共享组件：

- 页头导入、导出、刷新等次级操作使用 `ListHeaderActions.utility`
- 页头新增、生成、注册等主操作使用 `ListHeaderActions.primary`
- 批量选择后的启用、禁用、清空选择等使用 `TableBatchActionBar`
- 登录日志、会话管理、操作日志等保留期清理使用 `GovernanceCleanupBar`
- `ListHeaderActions` 间距必须使用 `--shell-list-actions-gap`
- `TableBatchActionBar` 主行间距和高度必须使用 `--shell-action-bar-gap`、`--shell-action-bar-min-height`
- `GovernanceCleanupBar` 的保留期 Select 宽度必须使用 `--shell-governance-select-width`
- 治理条的清理动作与保留期选择在左侧元信息区，额外动作在右侧对齐；不得出现有的页面左对齐、有的页面右对齐
- 禁止页面通过 `.xxx-page .list-header-actions...` 或 `.xxx-page .table-batch-action-bar...` 私自改 gap、按钮高度和主轴对齐

## 3.4 Modal 与 Drawer

简单新增/编辑：

- 使用 Modal

复杂表单：

- 使用 Drawer

危险操作：

- 使用 Popconfirm 或 Modal confirm

## 3.5 权限规则

- 页面进入需要 `view` 权限
- 查询列表需要 `list` 权限
- 新增按钮需要 `create`
- 编辑按钮需要 `update`
- 删除按钮需要 `delete`

## 3.6 完成定义

一个 ListPage 完成时必须具备：

- 筛选
- 表格
- 分页
- loading
- empty
- error
- 按钮权限
- i18n
- 创建/编辑/删除反馈

## 4. TreePage 模板

适用于：

- 部门管理
- 菜单管理
- 组织树

## 4.1 标准结构

```text
TreePage
  ├── PageHeader
  ├── FilterPanel
  ├── TreeTableCard
  └── CreateEditDrawer
```

## 4.2 树表规则

- 同层级排序稳定
- 父子层级清晰
- 操作列固定
- 不允许删除有子节点的节点
- 支持展开/收起

## 4.3 菜单树特殊规则

菜单管理需要展示：

- 菜单类型
- 路由 path
- 组件路径
- 权限标识
- 显示状态
- 排序

后续建议补：

- 图标选择器
- 组件路径校验
- route name
- cache 标记

## 5. DetailPage 模板

适用于：

- 用户详情
- 角色详情
- 策略详情
- 操作日志详情

## 5.1 标准结构

```text
DetailPage
  ├── PageHeader
  ├── SummaryCard
  ├── SectionCard[]
  └── StickyActionBar
```

## 5.2 SummaryCard

用于展示：

- 核心标识
- 状态
- 创建时间
- 更新时间
- 关键关联对象

## 5.3 SectionCard

用于分组展示：

- 基本信息
- 权限信息
- 组织信息
- 安全信息

## 5.4 权限规则

- 无查看权限时显示 403
- 无编辑权限时隐藏编辑按钮

## 6. FormPage 模板

适用于复杂流程，不适合塞进 Modal 的场景：

- 复杂系统设置
- 多步骤配置
- 大型业务对象创建

## 6.1 标准结构

```text
FormPage
  ├── PageHeader
  ├── FormCard
  ├── FormSections
  └── StickyFooter
```

## 6.2 表单规则

- 长表单必须分区
- 危险字段必须有帮助文案
- 提交按钮固定底部或显著位置
- 离开页面前如有未保存变更，应提示

## 7. ConfigPage 模板

适用于：

- 系统设置
- 安全策略
- 上传配置
- 通知配置

## 7.1 标准结构

```text
ConfigPage
  ├── PageHeader
  ├── ConfigLayout
  │   ├── CategoryNav
  │   └── ConfigPanel
  └── StickySaveBar
```

## 7.2 配置分类

建议左侧分类：

- 基础信息
- 安全策略
- 登录策略
- 上传配置
- 国际化

## 7.3 保存规则

- 配置修改后显示未保存状态
- 保存成功给明确反馈
- 危险配置修改需要二次确认

## 8. DashboardPage 模板

适用于：

- 仪表盘
- 工作台首页

## 8.1 标准结构

```text
DashboardPage
  ├── WelcomeHeader
  ├── MetricGrid
  ├── QuickActions
  ├── RecentActivity
  └── AlertSummary
```

## 8.2 设计规则

- 不做营销页 hero
- 不堆无意义统计卡
- 每个卡片都要回答“用户下一步该做什么”
- 彩色点缀可以存在，但业务内容优先

## 8.3 推荐指标

底座阶段可展示：

- 用户数
- 角色数
- 菜单数
- 活跃会话
- 最近登录
- 最近操作

## 9. AuthPage 模板

适用于：

- 登录
- 忘记密码
- 二次验证

## 9.1 标准结构

```text
AuthPage
  ├── BrandPanel
  ├── AuthCard
  │   ├── title
  │   ├── form
  │   └── actions
  └── Footer
```

## 9.2 设计规则

- 登录页可以有轻微品牌感
- 禁止大面积紫蓝渐变
- 表单区域保持清晰、稳定
- 错误提示贴近表单，不要只弹 toast
- 语言切换入口必须可见

## 10. ProfilePage 模板

适用于：

- 个人中心
- 账号资料
- 安全设置

## 10.1 标准结构

```text
ProfilePage
  ├── ProfileHeader
  ├── Tabs
  │   ├── BasicProfile
  │   ├── Security
  │   └── Sessions
  └── SaveActions
```

## 10.2 分区建议

- 基本资料
- 密码修改
- 在线会话
- 登录日志

## 11. StatePage 模板

适用于：

- 403
- 404
- 500
- 网络错误
- 空态

## 11.1 标准结构

```text
StatePage
  ├── Illustration / Icon
  ├── Title
  ├── Description
  └── Actions
```

## 11.2 403

必须提供：

- 返回首页
- 返回上一页
- 联系管理员提示

## 11.3 404

必须提供：

- 返回首页
- 检查路径提示

## 11.4 500 / 网络错误

必须提供：

- 重试
- 返回首页
- 错误 ID（后续可接入 request id）

## 12. 模板选择规则

新增页面前必须先判断页面类型：

| 如果页面主要是 | 使用 |
| :--- | :--- |
| 表格 CRUD | `ListPage` |
| 树结构 | `TreePage` |
| 只读详情 | `DetailPage` |
| 大表单 | `FormPage` |
| 分类配置 | `ConfigPage` |
| 首页概览 | `DashboardPage` |
| 登录认证 | `AuthPage` |
| 当前用户自助 | `ProfilePage` |
| 异常/空态 | `StatePage` |

## 13. AI 生成页面要求

AI 生成前端页面时必须：

1. 先声明页面类型
2. 套用对应模板
3. 补齐状态设计
4. 补齐权限点
5. 补齐 i18n key
6. 避免无规范内联样式
7. 不新增随机视觉风格

## 14. 当前代码改造建议

后续建议按以下顺序改造现有页面：

1. 抽 `PageHeader`
2. 抽 `FilterPanel`
3. 抽 `DataTableCard`
4. 抽 `StatePage`
5. 逐步改造用户、角色、菜单、部门、岗位、权限页

## 15. 完成定义

当以下条件满足时，页面模板体系算完成：

- 新增列表页不需要重新设计布局
- 所有系统页状态处理一致
- 所有页面标题、按钮、空态都走 i18n
- 所有页面权限表达一致
- 页面视觉不再出现明显 AI 拼装感

## 16. 下一份建议补的文档

下一份建议补：

- `docs/designs/FRONTEND_COMPONENT_PLAN.md`

因为页面模板定下来后，下一步最应该补的是“组件层沉淀计划”，否则后续还是会在每个页面里重复拼装。
