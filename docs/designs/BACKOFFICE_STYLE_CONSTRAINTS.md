# 后台 UI 风格约束

更新时间：2026-05-01

类型：Design
归属层：platform
状态：Active

本文用于约束 Pantheon Base 后台的**共享风格实现**，重点解决以下回归模式：

- 右侧辅助栏存在多种 `Card`/标题/边框写法；
- 对话框与抽屉的圆角、边框、阴影来源不统一；
- 系统页摘要卡、治理卡、工作台卡片各自维护一套半独立边框语言；
- 页面继续通过散落 CSS 直接“补丁式”修正样式，而不是收口到平台层组件与 token。

本文不是替代 `docs/designs/FRONTEND_UI_SPEC.md`，而是作为其**实现级补充约束**，用于阻断新的 UI 风格漂移。

## 1. 归属与边界

### 1.1 所属层级

本约束归属 `platform` 层。

- `platform` 负责定义共享风格 token、壳层布局、浮层体系、辅助栏体系、全局反馈视觉；
- `system/auth`、`system/iam`、`system/org`、`system/config`、`system/audit` 只消费这些规则，不得各自发明新的边框/圆角/浮层语言；
- `business/*` 默认继承同一套后台 UI 规则，除非有明确品牌隔离需求并单独立项。

### 1.2 本文治理对象

- 顶部栏、页签、右侧辅助栏、主从布局；
- `Modal / Drawer / Message / Notification / Dropdown / Popover` 的共享面板风格；
- 系统页中的摘要卡、上下文卡、治理卡、列表卡片；
- 与上述区域直接相关的 CSS token、共享组件和约束文档。

## 2. 审计结论

本轮仓库级审计确认，后台当前不是“某一个页面写得不好”，而是存在明确的**双轨样式状态**：

### 2.1 右侧辅助栏双轨

- 已有共享结构：`PageSplitLayout`、`SideRailPanel`、`SideRailItem`、`SideRailNote`
- 仍有大量页面直接手写：`page-split-layout + Card + side-rail-*`
- 结果：同样是右侧辅助栏，有的走 `Card` 自带 head，有的把标题塞进 body，有的靠 `title` prop，有的靠自定义 `<span>`

### 2.2 浮层双轨

- 已有共享封装：`AppModal`、`AppDrawer`、`showAppModalConfirm / Success / Error`
- 同时仍存在两层风格来源：
  - `AppModal / AppDrawer` 局部样式
  - 全局 `.arco-modal-content / .arco-drawer-content` 覆盖
- 结果：不同入口的对话框/抽屉可能表现出“看起来都是 Arco，但圆角、边框、阴影不完全一样”

### 2.3 卡片双轨

- 一部分页面已经使用 `page-panel / panel-border / radius-md`
- 另一部分系统页摘要卡仍在使用 `var(--color-border-2)`、`14/16/18px` 混合圆角
- 结果：同屏中会出现“右栏一套、摘要卡一套、组织卡一套”的边框语言

## 3. 强制实现约束

### 3.1 右侧辅助栏

右侧辅助栏必须满足以下约束：

- 页面布局必须优先使用 `PageSplitLayout`
- 右栏容器必须优先使用 `SideRailPanel`
- 右栏统计条目必须优先使用 `SideRailItem`
- 右栏说明/风险提示必须优先使用 `SideRailNote`

禁止：

- 禁止继续在页面中直接拼 `Card className="page-panel side-rail-panel"` 作为长期方案
- 禁止在右栏同时混用 `Card title`、自定义 body 标题、裸 `<span>` 标题三种模式
- 禁止右栏退化为“第二主内容列”
- 禁止在 `system/iam` 默认列表页中为了视觉对称强行加右栏

例外：

- 若页面没有真实上下文摘要或风险信息，直接回退为单列布局
- 若页面有强约束的审计/安全语义，允许在右栏中使用 warning/danger tone，但仍必须走 `SideRail*` 组件

### 3.2 浮层体系

后台共享浮层必须满足以下约束：

- 页面级弹窗统一使用 `AppModal`
- 页面级抽屉统一使用 `AppDrawer`
- 静态确认/成功/错误弹窗统一使用 `showAppModalConfirm / Success / Error`

禁止：

- 禁止在业务或系统页面中直接使用原生 `Modal.confirm / success / error`
- 禁止在页面代码里单独覆盖 modal/drawer 的圆角和面板边框
- 禁止一个页面同时出现“圆角 overlay”和“方角 overlay”两套视觉

### 3.3 卡片与面板

后台共享卡片面板必须遵守：

- 普通面板：`radius-md`
- 辅助栏内条目：`radius-sm`
- 浮层面板：`radius-overlay`
- 输入类控件：`radius-control`
- 操作按钮：`radius-action`
- 系统列表表格面板：`page-panel system-list__table-card`
- 系统列表表格容器：`.app-table .arco-table-container` 使用 `radius-md`
- 系统列表表格面板留白：统一使用 `--shell-table-card-padding`
- 系统列表筛选面板：统一使用 `FilterPanel` 与 `--shell-filter-*` token
- 系统列表页头动作：统一使用 `ListHeaderActions` 与 `--shell-list-actions-gap`
- 系统列表批量/治理动作：统一使用 `TableBatchActionBar` / `GovernanceCleanupBar` 与 `--shell-action-bar-*` token

禁止：

- 禁止在系统页摘要卡中随意使用 `14px / 18px / 22px` 等自由圆角
- 禁止同屏卡片混用 `var(--panel-border)` 与 `var(--color-border-2)` 作为主边框语言
- 禁止为“看起来更高级”额外叠加局部高阴影
- 禁止单个系统页通过 `.xxx-page .system-list__table-card .arco-card-body` 私自改表格面板左右 padding
- 禁止单个系统页通过 `.xxx-page .filter-panel ...` 私自改筛选区 padding、控件高度、label 间距或查询按钮对齐
- 禁止单个系统页通过 `.xxx-page .list-header-actions...` / `.xxx-page .table-batch-action-bar...` 私自改按钮间距、高度和主轴对齐
- 禁止保留 Arco fixed column 默认阴影，让表头左上角、固定列边缘呈现渐变边框观感

### 3.4 系统表格视觉契约

用户、角色、部门、岗位、字典、国际化、登录日志、操作日志等系统域列表页必须共享同一套表格视觉契约：

- 表格外层面板统一走 `page-panel system-list__table-card`
- 表格面板左右留白、顶部批量操作栏间距、表格主体起点必须一致
- 表头左上角不得出现渐变、阴影或主题色边缘
- 表头背景使用中性 surface，不随主题色变成绿色、紫色或蓝色染色
- 表格容器圆角统一由 `--radius-md` 控制，不允许同屏出现圆角表格和方角表格
- 如果页面有 Tabs 或主从布局，Tabs/主从容器可以包住表格，但不能改变表格自身的 TableCard 契约

### 3.5 系统筛选与操作视觉契约

用户、角色、权限、菜单、部门、岗位、字典、国际化、会话管理、登录日志、操作日志等系统域页面必须共享同一套筛选与操作契约：

- 筛选区一律由 `FilterPanel` 渲染，body 留白来自 `--shell-filter-body-padding`
- 筛选控件高度来自 `--shell-filter-control-min-height`，不得出现 32px、34px、自由高度混用
- 筛选项底部节奏来自 `--shell-filter-form-item-margin-bottom` 和 `--shell-filter-label-padding-bottom`
- 查询/重置按钮通过 `filter-panel__action-item` 与输入控件底线对齐
- 页头操作条通过 `ListHeaderActions` 分离次级动作和主动作，gap 来自 `--shell-list-actions-gap`
- 批量操作条通过 `TableBatchActionBar`，治理清理条通过 `GovernanceCleanupBar`
- 治理清理条的保留期选择宽度来自 `--shell-governance-select-width`，额外动作统一靠右
- 视觉密度只能通过平台层 `data-pantheon-density` token 生效，不得在页面局部重新声明压缩值

## 4. 共享 Token 约束

### 4.1 必须使用的共享 token

- `--panel-border`
- `--panel-border-strong`
- `--panel-bg-solid`
- `--panel-shadow-soft`
- `--panel-shadow-strong`
- `--radius-sm`
- `--radius-md`
- `--radius-overlay`
- `--radius-control`
- `--radius-action`

### 4.2 禁止扩散的旧写法

以下写法视为历史兼容，不得新增：

- `var(--color-border-2)` 作为后台共享容器主边框
- 新页面继续手写 `border-radius: 14px`
- 新页面继续手写 `border-radius: 18px`
- 通过散落在页面 CSS 中的局部规则覆盖 `Modal/Drawer` 的壳子样式

## 5. 组件使用规则

### 5.1 页面布局

- 双列页：`PageSplitLayout`
- 单列页：`PageContainer + PageHeader + FilterPanel + AppTable`
- 右栏说明：`SideRailPanel + SideRailNote`
- 右栏摘要：`SideRailPanel + SideRailStack + SideRailItem`
- 右栏摘要数据结构：`GovernanceRailSummary` 与 `StandardRailSummary` 必须共用 `RailSummaryItem`

### 5.2 浮层

- 表单编辑：`AppModal`
- 详情/预览：`AppDrawer` 或 `AppModal(size="detail")`
- 危险确认：`showAppModalConfirm`

### 5.3 不允许的直写模式

- `div.page-split-layout > div.page-side-column > Card` 作为长期公共模式
- 页面 CSS 私自定义另一套 `.xxx-dialog`、`.xxx-drawer`
- 在页面内复制 `side-rail-*` 结构但不复用共享组件

## 6. 验收清单

涉及后台共享视觉修改时，至少检查以下项目：

1. 右侧辅助栏是否仍通过共享组件生成
2. 同一页面中的面板边框是否统一使用 `--panel-border`
3. 对话框与抽屉是否都呈现同一套 overlay 圆角
4. 摘要卡、治理卡、辅助栏卡片是否仍保持同一套圆角等级
5. 是否新增了页面级硬编码圆角/边框/阴影
6. 是否破坏 `FRONTEND_UI_SPEC.md` 中的壳层、导航、状态规范

## 7. 推荐治理顺序

当发现后台 UI 风格回归时，按以下顺序治理：

1. 先查是否绕过共享组件
2. 再查是否新增了局部 token 或自由样式值
3. 再查是否存在全局覆盖与局部覆盖双轨并存
4. 最后才做页面级微调

不要直接在单页上“修到看起来差不多”，否则会继续累积样式碎片。

## 8. 与现有文档关系

- 顶层视觉/布局规范：`docs/designs/FRONTEND_UI_SPEC.md`
- 后台整改路线：`docs/remediations/BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md`
- 本文职责：把“风格一致性”落实成可执行、可搜索、可阻断回归的实现约束
