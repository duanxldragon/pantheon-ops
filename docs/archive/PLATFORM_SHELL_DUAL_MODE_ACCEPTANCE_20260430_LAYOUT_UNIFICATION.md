# Platform 壳层双模式验收记录

验收批次：`platform-shell-2026-04-30-layout-unification`

更新时间：2026-04-30

类型：Acceptance
归属层：platform
状态：Archived

基于模板：`docs/acceptances/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md`

---

## 1. 基本信息

- 日期：`2026-04-30`
- 执行人：`Codex`
- 所属层级：`platform`
- 关联文档：
  - `docs/acceptances/ACCEPTANCE_CHECKLIST.md`
  - `docs/acceptances/PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md`
  - `docs/designs/FRONTEND_UI_SPEC.md`
- 改动范围：
  - 文件：`frontend/src/core/layout/index.css`
  - 目标：统一竖版侧栏与横版顶栏的菜单状态语言，收口 hover / selected / icon badge / popup 规则
  - 非目标：不调整 `system/*` 页面内部内容布局，不调整动态菜单数据契约，不调整权限判断链路

---

## 2. 触发原因

- 是否触达以下任一对象：`Yes`
  - `frontend/src/core/layout/index.tsx`：`No`
  - `frontend/src/core/layout/index.css`：`Yes`
  - 菜单渲染或菜单状态链路：`Yes`
  - 顶部横版导航：`Yes`
  - 左侧竖版导航：`Yes`
- 说明：本轮直接修改了 `platform` 壳层导航样式规则，因此必须执行双模式验收。

---

## 3. 固定验收输入

### 3.1 竖版侧栏展开态

- 访问路径：`/dashboard`、`/system/user`、`/system/role`
- 账号角色：`admin`
- 菜单层级样本：
  - 一级叶子：`/dashboard`
  - 一级分组：`访问控制`
  - 二级叶子：`/system/user`
- 结论：`Pass`
- 备注：展开态下一级分组、一级叶子和二级叶子均维持统一的 icon badge 与左侧激活导轨表达。

### 3.2 竖版侧栏折叠态

- 访问路径：`/dashboard`、`/system/user`
- 账号角色：`admin`
- tooltip / 高亮 / 折叠记忆是否正常：`Yes`
- 结论：`Pass`
- 备注：折叠后图标容器尺寸与展开态保持同源，当前菜单高亮不丢失。

### 3.3 横版主导航态

- 访问路径：`/dashboard`、`/system/user`
- 账号角色：`admin`
- 一级菜单样本：`工作台`、`访问控制`、`平台配置`
- 结论：`Pass`
- 备注：横版主导航已退出独立下划线标签风格，改为与竖版一致的轻强调卡片语义。

### 3.4 横版弹出子菜单态

- 访问路径：`/system/user`、`/system/setting`
- 账号角色：`admin`
- 分组菜单样本：`访问控制 -> 用户 / 角色 / 权限 / 菜单`
- 结论：`Pass`
- 备注：弹出层继续沿用 icon badge、左侧激活导轨和选中填充规则，与竖版分组菜单一致。

---

## 4. 固定验收项

| 验收项 | 竖版展开 | 竖版折叠 | 横版主导航 | 横版弹出子菜单 | 备注 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 当前菜单高亮一致 | Pass | Pass | Pass | Pass | 当前路径与 `activeMenu` 高亮一致 |
| hover / selected / open 三态一致 | Pass | Pass | Pass | Pass | 四种输入态已统一为同一套节奏 |
| icon badge 尺寸一致 | Pass | Pass | Pass | Pass | 一级 28px、二级 22px |
| 间距与文字节奏一致 | Pass | Pass | Pass | Pass | 横版不再显著偏胶囊化 |
| 普通叶子菜单表达一致 | Pass | Pass | Pass | Pass | 叶子菜单选中语义一致 |
| 分组菜单表达一致 | Pass | Pass | Pass | Pass | 分组标题与子级关系一致 |
| 外链菜单表达一致 | Pass | Pass | Pass | Pass | 保持同源 hover / selected 规则 |
| 品牌区与导航关系稳定 | Pass | Pass | Pass | Pass | 横版品牌区与导航壳层未打架 |
| 面包屑未错位 | Pass | Pass | Pass | Pass | 顶部信息层稳定 |
| 页签区未错位 | Pass | Pass | Pass | Pass | 横竖切换后页签区未挤压错位 |
| 滚动与溢出行为正常 | Pass | Pass | Pass | Pass | 横版可滚动，竖版长菜单可滚动 |

---

## 5. 固定扫描记录

### 5.1 壳层链路扫描

- 命令：`rg "ShellLayoutMode|isHorizontalLayout" frontend/src/core/layout`
- 结果：命中 `frontend/src/core/layout/index.tsx` 中的布局模式切换与判定逻辑，符合预期。

### 5.2 旧右栏类名扫描

- 命令：`rg "system-page-side|system-page-summary-card|system-page-note|system-page-main-grid|system-page-main" frontend/src`
- 结果：命中 `0`

### 5.3 原生浮层扫描

- 命令：`rg "<Modal|<Drawer" frontend/src/modules frontend/src/components`
- 结果：仅命中 `AppModal.tsx` 与 `AppDrawer.tsx` 平台封装

### 5.4 静态 Modal API 扫描

- 命令：`rg "Modal\\.confirm|Modal\\.(success|error|info|warning)" frontend/src`
- 结果：仅命中 `AppModal.tsx` 平台封装内部与 i18n 文案 key

### 5.5 构建结果

- 命令：`cmd /c npm run build`
- 结果：通过；前置 `check:menu-contract` 与 `check:i18n-hardcode` 同步通过

---

## 6. 例外与挂账

本轮无 `Pending` / `Fail` 挂账项。

| 类型 | 文件 | 状态 | 原因 | 是否阻断合入 | 后续动作 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 无 | - | Pass | 本轮双模式验收全部通过 | No | 继续作为基准记录复用 |

---

## 7. 最终结论

- 本轮是否通过双模式验收：`Yes`
- 是否允许标记为 `Target`：`Yes`
- 是否需要回写矩阵：`Yes`，已完成
- 结论摘要：
  - 本轮 `platform` 壳层导航改动已完成双模式验收闭环；
  - 横版顶栏与竖版侧栏现已共享同一套菜单视觉语言；
  - 本记录可作为后续壳层导航改动的首个基准样例。
