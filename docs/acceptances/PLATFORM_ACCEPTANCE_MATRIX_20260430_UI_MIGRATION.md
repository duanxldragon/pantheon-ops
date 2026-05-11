# 平台层 UI 迁移验收矩阵（2026-04-30）

更新时间：2026-04-30

类型：Acceptance
归属层：platform
状态：Active

本文用于承接 2026-04-30 的第二阶段文档驱动盘点，重点回答一个平台层问题：

> 当前前端哪些页面和壳层能力已经进入“新骨架”，哪些仍停留在“旧样式遗留”，哪些属于平台策略尚未最终定稿的灰区。

本文不是代码改造记录，也不是视觉稿。

它的作用是为后续 UI 收口提供统一迁移锚点，避免继续出现“新旧两套样式都在被复制”的状态。

适用范围：

- `platform`
- `system/auth`
- `system/iam`
- `system/org`
- `system/config`
- `system/audit`
- 低代码辅助开发链路中的前端页面与浮层

---

## 1. 第二阶段结论

截至 2026-04-30 本轮代码收口完成后，Pantheon 前端的真实状态更新为：

- **平台公共骨架已扩展**：`PageContainer / PageHeader / FilterPanel / PageSplitLayout / SideRail / AppModal / AppDrawer`
- **旧右侧栏模板已从页面代码退出**
- **原生 `<Modal>` / `<Drawer>` 已从业务 JSX 退出，静态确认/反馈 API 也已收编进平台封装**
- **左侧导航与双导航模式已实现，但最终视觉策略仍需平台层确认**

因此当前结论是：

- 右侧辅助栏和主要浮层链路已经完成第一轮平台层收口
- 可以把 `PageSplitLayout + SideRail + AppModal / AppDrawer` 作为新的平台参考源
- 后续重点从“退出旧样式”切换为“收紧导航视觉策略与受控例外清单”

---

## 2. 迁移状态定义

| 状态 | 含义 | 是否可继续复制 |
| :--- | :--- | :--- |
| `Target` | 已符合当前平台层目标方向 | 可以 |
| `Mixed` | 已接入部分公共骨架，但仍夹带旧模式 | 不可以 |
| `Legacy` | 明确历史遗留，等待迁移 | 不可以 |
| `Pending` | 平台策略已实现但视觉方向尚未最终定稿 | 不可以 |

---

## 3. 盘点矩阵

### 3.1 平台壳层与导航策略

| 对象 | 归属层 | 当前状态 | 判断 | 后续要求 | 证据 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 左侧导航基础壳层 | `platform` | 已有独立壳层与折叠态 | `Target` | 保持中性 surface、弱边框、共享菜单状态语言，不回退到高装饰深色导航壳 | `frontend/src/core/layout/index.tsx` `frontend/src/core/layout/index.css` |
| 横版顶栏模式 | `platform` | 已实现布局切换与状态持久化 | `Target` | 继续纳入固定验收，但视觉规则已与竖版统一 | `frontend/src/core/layout/index.tsx` `frontend/src/core/layout/index.css` |
| 左导航滚动与折叠链路 | `platform` | 功能已存在 | `Target` | 继续沿平台壳层统一治理，不下放到页面 | `frontend/src/core/layout/index.css` |
| 左导航内容职责 | `platform` | 文档已锁定为“只做导航” | `Target` | 后续不在侧栏新增说明卡、统计卡、帮助卡 | `docs/designs/FRONTEND_UI_SPEC.md` |

判断补充：

- 2026-04-30 本轮收口后，左导航与横版顶栏已共享同一套 hover / selected / icon badge / popup 规则；
- 这属于 `platform` 壳层统一治理结果，不属于 `system/iam` 或 `system/config` 单页问题。

### 3.2 右侧辅助栏收口矩阵

以下页面已完成旧右栏类名退出，并统一迁移到 `PageSplitLayout + SideRail` 平台语义，可作为后续同类页面的参考源。

| 页面 / 组件 | 归属层 | 当前状态 | 当前承载方式 | 平台判断 | 备注 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `frontend/src/modules/system/user/UserList.tsx` | `system/iam` | `Target` | `SummaryRail` | 可继续复制 | 已退出旧摘要栏类名 |
| `frontend/src/modules/system/role/RoleList.tsx` | `system/iam` | `Target` | `SummaryRail` | 可继续复制 | 已退出旧摘要栏类名 |
| `frontend/src/modules/system/menu/MenuList.tsx` | `system/iam` | `Target` | `SummaryRail` | 可继续复制 | 已退出旧摘要栏类名 |
| `frontend/src/modules/system/permission/PermissionList.tsx` | `system/iam` | `Target` | `SummaryRail` | 可继续复制 | 已退出旧摘要栏类名 |
| `frontend/src/modules/system/dept/DeptList.tsx` | `system/org` | `Target` | 轻量治理栏 | 可继续复制 | 治理信息仍停留在右栏边界内 |
| `frontend/src/modules/system/post/PostList.tsx` | `system/org` | `Target` | `SummaryRail` | 可继续复制 | 已退出旧摘要栏类名 |
| `frontend/src/modules/system/dict/DictPage.tsx` | `system/config` | `Target` | 风险/说明型 Side Rail | 可继续复制 | 主从布局与辅助栏职责已分离 |
| `frontend/src/modules/system/i18n/I18nList.tsx` | `system/config` | `Target` | 风险/治理型 Side Rail | 可继续复制 | 已退出第二主内容列式右栏 |
| `frontend/src/modules/system/setting/SettingPage.tsx` | `system/config` | `Target` | `RiskRail` | 可继续复制 | 已收口到风险提示语义 |
| `frontend/src/modules/system/audit/OperationLogList.tsx` | `system/audit` | `Target` | `RiskRail` | 可继续复制 | 不再重复主内容摘要 |
| `frontend/src/modules/auth/LoginLogList.tsx` | `system/auth` | `Target` | `RiskRail` | 可继续复制 | 已退出旧摘要栏类名 |
| `frontend/src/modules/auth/SessionList.tsx` | `system/auth` | `Target` | `RiskRail` | 可继续复制 | 已退出旧摘要栏类名 |
| `frontend/src/modules/auth/SecurityCenter.tsx` | `system/auth` | `Target` | `RiskRail + Policy note` | 可继续复制 | 不再形成第二主内容列 |

统一退出条件：

- 不再出现 `system-page-side / system-page-summary-card / system-page-note`
- 每个右栏都能明确归类为 `SummaryRail / RiskRail / PolicyRail`
- 右栏内容能清楚回答“这是 Context 还是 Alert”

当前代码证据：

- `rg "system-page-side|system-page-summary-card|system-page-note|system-page-main-grid|system-page-main" frontend/src` 命中 `0`

### 3.3 原生浮层遗留矩阵

以下浮层链路已完成 JSX 层收口；业务层不再直接调用 Arco 静态 `Modal.*`，统一经平台封装入口触发。

| 文件 | 归属层 | 当前状态 | 当前实现 | 平台判断 | 备注 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `frontend/src/modules/system/dynamicmodule/ModuleManager.tsx` | `system/config` | `Target` | `AppModal` | 可继续复制 | 危险治理浮层已收口 |
| `frontend/src/components/feedback/SecondaryVerifyModal.tsx` | `platform` | `Target` | `AppModal` | 可继续复制 | 二次验证浮层已收口 |
| `frontend/src/modules/generator/pages/ModuleWizard.tsx` | 低代码辅助链路 | `Target` | `AppModal` + `showAppModalConfirm` | 可继续复制 | 静态确认已收口到平台入口 |
| `frontend/src/modules/generator/components/FieldEditor.tsx` | 低代码辅助链路 | `Target` | `AppModal` + `showAppModalConfirm` | 可继续复制 | 静态确认已收口到平台入口 |
| `frontend/src/modules/generator/components/CodePreview.tsx` | 低代码辅助链路 | `Target` | `AppDrawer` | 可继续复制 | Drawer 已纳入平台壳 |
| `frontend/src/components/patterns/AppModal.tsx` | `platform` | `Target` | 统一弹窗模式 | 作为后续平台弹窗基线继续演进 | 基线 |
| `frontend/src/components/patterns/AppDrawer.tsx` | `platform` | `Target` | 统一抽屉模式 | 作为后续平台抽屉基线继续演进 | 基线 |

判断补充：

- `generator/*` 虽然不属于核心系统域页面，但仍属于前端平台治理的一部分；
- `Modal.confirm`、`Modal.success/error` 目前仅允许存在于 `AppModal` 平台封装内部；
- 如果未来业务层继续新增静态 API 调用，应直接视为回归。

### 3.4 已可作为新参考源的公共骨架

| 能力 | 当前状态 | 结论 |
| :--- | :--- | :--- |
| `PageContainer` | 已广泛接入 | 可继续作为页面容器基线 |
| `PageHeader` | 已广泛接入 | 可继续作为页头基线 |
| `FilterPanel` | 已广泛接入 | 可继续作为筛选区基线 |
| `PageSplitLayout` | 已落地到系统页右栏迁移 | 可继续作为双栏骨架基线 |
| `SideRail` | 已落地到系统页右栏迁移 | 可继续作为右侧辅助栏基线 |
| `AppModal` | 已存在统一宽度与样式壳 | 可继续作为平台弹窗基线 |
| `AppDrawer` | 已存在统一宽度与样式壳 | 可继续作为平台抽屉基线 |

注意：

- “页面已经用了 `PageContainer`” 不等于“页面已经完成 UI 收口”；
- 只有在右栏、浮层、导航职责也一起收口后，才算真正进入 `Target`。

---

## 4. 第二阶段迁移顺序

按平台层优先级，建议顺序固定为：

1. 先定左导航视觉策略和横/竖导航共同验收口径
2. 再抽象统一 `SideRail` 语义和退出旧右栏类名
3. 再统一危险确认、二次验证、短表单编辑、详情查看四类浮层模式
4. 最后按系统域逐页替换遗留实现

当前进度补充：

- 第 1 步已完成：横版与竖版现已共享同一套菜单视觉语言；
- 第 2、3 步已完成：右栏与主要浮层也已完成平台层收口；
- 当前平台层剩余工作以持续验收、防回归和小幅细节优化为主。

不建议的做法：

- 继续拿 `system/user` 或 `system/role` 当前页面外观当“黄金模板”
- 先逐页改 CSS，再回头补壳层语义
- 放任 `generator/*`、`dynamicmodule`、`components/feedback` 持续扩散原生浮层

---

## 5. 第二阶段验收要求

本矩阵生效后，后续每轮 UI 改造至少补三类证据：

1. 旧右栏类名扫描结果
2. 原生 `Modal` / `Drawer` 使用点扫描结果
3. 左导航竖版 / 横版两种模式的验收记录

双模式验收记录不再接受自由发挥，统一至少包含以下字段：

- 改动范围：本轮触发了哪些 `platform` 壳层对象
- 竖版展开态结论
- 竖版折叠态结论
- 横版主导航结论
- 横版弹出子菜单结论
- 是否存在 `Pending` 例外；若存在，必须给出文件位置和阻断原因

固定模板：

- `docs/acceptances/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md`

建议最小扫描命令：

- `rg "system-page-side|system-page-summary-card|system-page-note" frontend/src`
- `rg "<Modal|<Drawer" frontend/src/modules frontend/src/components`
- `rg "Modal\\.confirm|Modal\\.(success|error|info|warning)" frontend/src`
- `rg "ShellLayoutMode|isHorizontalLayout" frontend/src/core/layout`

当前收口证据：

1. `cmd /c npm run build` 已通过，且前置 `check:menu-contract`、`check:i18n-hardcode` 同步通过
2. `rg "system-page-side|system-page-summary-card|system-page-note|system-page-main-grid|system-page-main" frontend/src` 命中 `0`
3. `rg "<Modal|<Drawer" frontend/src/modules frontend/src/components` 仅命中 `AppModal.tsx` 与 `AppDrawer.tsx` 两个平台封装
4. `rg "Modal\\.confirm|Modal\\.(success|error|info|warning)" frontend/src` 仅命中 `AppModal.tsx` 平台封装内部与 i18n 文案 key
5. `frontend/src/core/layout/index.css` 中横版导航已切换为与竖版共享的 icon badge、左侧激活导轨和轻强调选中规则
6. 首个真实双模式验收样例已落档：`docs/archive/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_20260430_LAYOUT_UNIFICATION.md`

纪律补充：

- 以后凡是触达 `frontend/src/core/layout/index.tsx`、`frontend/src/core/layout/index.css` 或菜单渲染链路的改动，都默认触发双模式验收；
- 未附双模式验收记录的壳层改动，不应判定为 `Target`。
- 后续壳层 PR 或阶段记录，必须显式附 `docs/acceptances/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md` 产出的验收文档链接；
- 首个可复用基准样例为 `docs/archive/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_20260430_LAYOUT_UNIFICATION.md`。

---

## 6. 与其他文档的关系

本矩阵与以下文档配套使用：

- `docs/designs/FRONTEND_UI_SPEC.md`
- `docs/remediations/BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md`
- `docs/acceptances/ACCEPTANCE_CHECKLIST.md`
- `docs/archive/PLATFORM_ACCEPTANCE_MATRIX_20260427.md`

边界说明：

- `FRONTEND_UI_SPEC.md` 负责定义目标规则
- `BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md` 负责定义整改方向与优先级
- `ACCEPTANCE_CHECKLIST.md` 负责定义验收门槛
- **本文负责定义 2026-04-30 这轮混合态盘点结果与页面级迁移归属**
