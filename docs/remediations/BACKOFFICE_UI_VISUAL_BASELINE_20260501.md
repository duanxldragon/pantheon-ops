# 后台 UI 截图基线与视觉回归清单

更新时间：2026-05-01

类型：Baseline
归属层：platform
状态：Active

## 1. 目标

本清单用于固定 Pantheon Base 后台 UI 的平台层视觉基线，防止以下回归：

- 壳层导航只校验竖版、不校验横版；
- 视觉密度偏好接入后，只在局部页面生效；
- 平台工作台重新出现未注册的业务域卡片；
- 系统页、登录页和浮层文案在多语言或样式收口后悄然退化。

## 2. 当前自动化入口

前端视觉 smoke 统一入口：

- `frontend/tests/smoke/backoffice-ui-visual.spec.ts`
- 运行命令：`cd frontend && npm run test:smoke:backoffice-ui`

截图产物目录：

- `frontend/test-results/backoffice-ui/`

## 3. 当前截图基线

### 3.1 默认壳层基线

- `login-desktop.png`
- `login-mobile.png`
- `dashboard-desktop.png`
- `system-user-desktop.png`
- `system-role-desktop.png`
- `system-permission-desktop.png`
- `system-menu-desktop.png`
- `system-dept-desktop.png`
- `system-post-desktop.png`
- `system-setting-desktop.png`
- `auth-security-desktop.png`

### 3.2 平台偏好多模式基线

- `dashboard-horizontal-compact.png`
- `system-user-horizontal-compact.png`

说明：

- 这组截图用于覆盖 `platform` 壳层的横版导航 + 紧凑密度组合；
- 后续任何壳层导航、表格密度、页头节奏调整，都必须同时复核默认模式与这组偏好模式。

### 3.3 关键浮层与表单提示基线

- `secondary-verify-dialog-validation.png`
- `system-i18n-create-validation.png`

## 4. 回归验收规则

### 4.1 平台壳层

- 必须同时检查竖版侧栏和横版顶栏；
- 必须至少覆盖一种非默认偏好组合，当前固定为 `horizontal + compact`；
- 顶部偏好开关只允许改变节奏和呈现，不允许改变菜单、路由与权限装配边界。

### 4.2 工作台

- 工作台首屏只允许展示注册过的 widget；
- `business/*` widget 必须声明来源域、权限、跳转路径、注册责任人与清理策略；
- 未注册业务卡片不得直接进入 `Dashboard.tsx`。

### 4.3 系统域列表页

- 紧凑密度下仍需保持表格可读性、操作区可点性和分页完整性；
- Hero、筛选区、表格区之间的节奏变化必须一致，不允许只缩一处；
- 不允许因为密度收紧导致首屏重新堆出长说明段落。

## 5. 人工复核清单

- 登录页桌面 / 移动首屏是否仍保持最短任务路径；
- 工作台第一屏是否仍遵守“4 个以内一级指标 + 6 个以内快捷入口”；
- 右侧栏页面在中屏下是否退化为单列或稳定双列，而不是互相挤压；
- 中英文本切换后，偏好面板和横版导航是否仍不溢出。

## 6. 后续扩展纪律

- 新增需要长期维护的视觉基线截图时，必须同时更新本文件；
- 新增 `business/*` 工作台 widget 时，必须在工作台设计文档和本清单中追加对应验收点；
- 若未来接入真正的截图 diff 流程，本文件继续作为人工验收与自动化覆盖范围的对照表。
