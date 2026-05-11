# 前端组件规划

更新时间：2026-04-17

类型：Design
归属层：platform
状态：Active

本文用于承接：

- `docs/designs/FRONTEND_UI_SPEC.md` 的视觉与交互规范
- `docs/designs/FRONTEND_PAGE_TEMPLATES.md` 的页面骨架模板

它回答的是另一个更落地的问题：

- 后续前端到底应该沉淀哪些公共组件？
- 哪些组件属于页面骨架，哪些属于业务可复用组件？
- 如何避免每个页面都重新拼一套筛选区、表格区、状态区？

这份文档的目标是：**在真正开始大规模写页面之前，先把组件分层和复用策略定下来。**

## 1. 设计目标

- 统一后台页面骨架
- 降低 CRUD 页面重复开发成本
- 保持企业级后台的克制感和一致性
- 避免“页面能跑，但组件体系失控”
- 让 AI 和人工都能按同一套组件心智写页面

## 2. 组件分层

前端组件建议分为四层。

### 2.1 Design Token 层

负责：

- 颜色
- 间距
- 圆角
- 阴影
- 字号
- 行高
- z-index
- 多主题 token 映射（如 `indigo / emerald / violet / slate`）

特点：

- 不直接承载业务逻辑
- 不直接与页面耦合
- 优先通过 Arco Theme / CSS Variables 落地
- 登录页、Layout、Modal、Drawer、Message、Notification 必须共用同一套 token

### 2.2 Primitive 层

负责最基础的 UI 原语封装，例如：

- `AppIcon`
- `AppLink`
- `AppTag`
- `AppStatusBadge`
- `AppEmptyIllustration`

特点：

- 语义轻
- 复用范围广
- 不绑定具体业务模块

### 2.3 Pattern 层

这是企业后台最重要的一层。

负责封装常见页面模式，例如：

- 页头
- 筛选区
- 表格工具栏
- 列表状态区
- 表单区
- 详情区
- 权限态

这一层应该成为业务页面的主力基础设施。

### 2.4 Module 层

指业务模块自己特有的组件，例如：

- 订单状态时间线
- 工单 SLA 卡片
- 项目成员矩阵

特点：

- 可复用范围局限在某个模块或某类业务
- 不应反向污染公共组件层

## 3. 目录建议

推荐逐步演进为：

```text
frontend/src/components/
  primitives/
  patterns/
  feedback/
  data-display/
  data-entry/
  navigation/
  auth/
  permission/
```

业务模块内部保留：

```text
frontend/src/modules/{system|business}/{module}/components/
```

约束：

- 跨模块通用组件进入 `src/components`
- 仅本模块可复用的组件留在模块目录
- 不允许把模块业务组件直接堆到全局组件目录

## 4. 第一批必须沉淀的公共组件

## 4.1 页面骨架类

### `PageContainer`

负责：

- 页面主容器
- 内边距
- 最大宽度策略
- 页面垂直节奏

### `PageHeader`

负责：

- 标题
- 副标题
- 面包屑承接区
- 右侧主操作区
- 预留平台层全局能力入口（如主题切换、语言切换的对齐节奏）

### `ThemeSwitcher`

负责：

- 切换平台层主题
- 持久化主题偏好
- 让登录页与系统域页面共用同一套视觉 token

要求：

- 放在 `platform/core` 层，不挂到某个系统子域
- 只切换 token，不复制页面结构和组件实现

### `PageSection`

负责：

- 页面内容分区
- 区块标题
- 区块说明
- 分区间距

### `PageActions`

负责：

- 新建
- 导出
- 批量操作
- 更多操作

要求：

- 对齐按钮优先级
- 控制危险操作位置

## 4.2 列表页类

### `FilterPanel`

负责：

- 查询表单
- 展开/收起
- 重置/搜索
- 高级筛选区

### `DataToolbar`

负责：

- 批量操作
- 列显示控制
- 刷新
- 导出
- 右侧辅助操作

### `AppTable`

基于 Arco Table 封装统一能力：

- loading
- empty
- rowKey
- 权限列控制
- 长文本省略
- 固定操作列
- 密度/滚动策略

### `TableSelectionBar`

负责：

- 已选择数量
- 批量删除/导出/状态切换

## 4.3 表单页类

### `FormPage`

负责：

- 页面级表单布局
- 主操作按钮
- 返回/取消
- 离开确认

### `FormSection`

负责：

- 表单分组标题
- 分组说明
- 栅格布局

### `SubmitBar`

负责：

- 保存
- 提交
- 取消
- 提交中状态

## 4.4 详情页类

### `DetailPage`

负责：

- 详情头部
- 状态标签
- 关键摘要信息
- 右侧操作区

### `DetailSection`

负责：

- 键值对布局
- 分组信息块
- 长文本和附件展示

### `TimelineCard`

负责：

- 审批记录
- 操作日志摘要
- 状态流转时间线

## 4.5 反馈与状态类

### `PageLoading`

用于页面级首屏加载。

### `PageEmpty`

用于无数据、首次使用、筛选无结果等场景。

### `PageError`

用于接口失败或初始化失败。

### `PageForbidden`

用于无页面权限场景。

### `InlineError`

用于局部模块失败而不是整页失败。

### `ConfirmDangerAction`

用于删除、停用、强制下线等危险操作确认。

## 4.6 权限与路由类

### `PermissionGuard`

负责：

- 页面访问保护
- 无权限 fallback

### `ActionGuard`

负责：

- 按钮权限
- 操作位显隐

### `RouteMetaTitle`

负责：

- 页面标题
- 文档标题
- 面包屑标题 key 对齐

## 4.7 导航类

### `AppSider`

负责：

- 动态菜单渲染
- 折叠
- 高亮
- 超长菜单滚动

### `AppTopbar`

负责：

- 全局搜索入口（预留）
- 语言切换
- 用户菜单
- 通知入口（预留）

### `BreadcrumbBar`

负责：

- 面包屑展示
- 长路径截断策略

## 4.8 Rail 模式类

### `PageSplitLayout`

负责：

- 双列后台页面骨架
- 主内容区与右侧辅助栏的共享布局节奏
- 在无辅助信息时平滑回退为单列

### `GovernanceRailPanel`

负责：

- 治理摘要 rail 的标题区
- 关闭按钮承接
- 摘要内容与说明 note 的组合承接

要求：

- 仅承接治理语义较强的右栏场景
- 与 `useGovernanceRail`、`GovernanceRailSummary` 配套使用

### `GovernanceRailSummary`

负责：

- 治理摘要条目列表渲染
- `warning / danger / neutral` tone 透传

要求：

- 条目结构统一使用 `RailSummaryItem`
- 不允许在系统页内重复声明 `label/value/description/tone` 的局部 item 类型

### `StandardRailSummary`

负责：

- 非治理类右栏摘要信息渲染
- 普通系统页上下文摘要复用

要求：

- 与 `GovernanceRailSummary` 共用 `RailSummaryItem`
- 仅允许在面板标题、面板包裹方式上与治理 rail 区分

### `StandardRailNotePanel`

负责：

- 非治理类右栏说明 note
- 统一承接普通提示、背景说明、只读辅助信息

## 5. 不建议过早抽象的组件

在项目早期，不要一上来就抽这些高度不稳定组件：

- 超复杂图表封装
- 通用业务审批流引擎组件
- 可配置表单引擎
- 可视化页面搭建器
- 全量低代码 schema renderer

原因：

- 现在边界还在收敛期
- 过早抽象很容易做成“看起来通用，实际上谁都不好用”

## 6. 组件设计约束

### 6.1 命名约束

- 页面模式组件优先使用语义名
- 避免 `CommonTable`、`BaseCard` 这类过泛命名
- 优先表达用途，而不是表达“通用”

### 6.2 样式约束

- 优先复用 Arco Design 能力
- 公共组件不随意内置大量魔法数
- 间距、字号、颜色尽量走统一 token

### 6.3 权限约束

- 权限判断优先集中到 `Guard` 类组件
- 不要在业务页面里到处散写字符串判断

### 6.4 i18n 约束

- 所有组件展示文案必须支持 `t()`
- 公共组件内置默认文案时，也必须走 key

### 6.5 组合优先于继承

- 优先通过 `children`、`slots`、render props 扩展
- 少做大而全的超级组件

## 7. 页面模板与组件映射

| 页面模板 | 推荐组件组合 |
| :--- | :--- |
| 列表页 | `PageContainer` + `PageHeader` + `FilterPanel` + `DataToolbar` + `AppTable` |
| 树表页 | `PageContainer` + `PageHeader` + `FilterPanel` + `AppTable` |
| 详情页 | `PageContainer` + `PageHeader` + `DetailPage` + `DetailSection` |
| 表单页 | `PageContainer` + `PageHeader` + `FormPage` + `FormSection` + `SubmitBar` |
| 配置页 | `PageContainer` + `PageHeader` + `PageSection` + `FormSection` |
| 安全页 | `PageContainer` + `PageHeader` + `PageSection` + `TimelineCard` + 状态组件 |

## 8. 推荐实施顺序

先做稳定、复用率最高的组件。

### P0

- `PageContainer`
- `PageHeader`
- `FilterPanel`
- `AppTable`
- `PageLoading`
- `PageEmpty`
- `PageError`
- `PageForbidden`

### P1

- `PageActions`
- `DataToolbar`
- `FormPage`
- `FormSection`
- `SubmitBar`
- `DetailSection`

当前落地状态：

- 已完成 `PageActions`
- 已完成 `FormSection`
- 已完成 `SubmitBar`
- 已完成 `AppTable` 作为列表页统一表格封装第一版
- `UserList`、`RoleList`、`PermissionList`、`ProfileCenter`、`DeptList`、`MenuList`、`PostList` 已开始接入第二批组件

### P2

- `PermissionGuard`
- `ActionGuard`
- `BreadcrumbBar`
- `AppTopbar` 细化
- `TimelineCard`

## 9. 组件完成定义

一个公共组件只有满足以下条件，才算可以进入复用：

- 责任单一
- 命名清晰
- 不耦合具体业务字段
- 支持多语言
- 支持权限态或状态态
- 在至少两个页面场景中可复用

如果只服务单一页面，不要急着放进公共组件层。

## 10. 与其他文档的边界

| 文档 | 负责什么 | 不负责什么 |
| :--- | :--- | :--- |
| `docs/designs/FRONTEND_UI_SPEC.md` | 视觉、布局、交互原则 | 具体组件清单与沉淀顺序 |
| `docs/designs/FRONTEND_PAGE_TEMPLATES.md` | 页面模板与骨架模式 | 组件分层和目录规划 |
| `docs/designs/MODULE_CONTRACT.md` | 模块如何注册 | 组件如何抽象 |
| `docs/designs/BUSINESS_MODULE_TEMPLATE.md` | 业务模块页面应写什么 | 公共组件应如何分层 |

## 11. 当前阶段结论

Pantheon Base 现阶段最需要的不是“更多页面”，而是**先把页面公共骨架和状态组件稳定下来**。

这样后续无论补：

- `auth`
- `security center`
- `dict`
- `setting`
- `business/order`

都不会重新拼一套页面结构。
