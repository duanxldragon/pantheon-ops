# Platform 壳层双模式验收模板

更新时间：2026-04-30

类型：Acceptance
归属层：platform
状态：Active

本文是 `platform` 壳层改动的固定验收模板。

提交要求：

- 本模板产出的记录文件，应用于后续壳层 PR、阶段记录和阶段收口说明；
- 提交说明中必须直接附本文档的实例化记录链接，不接受只贴截图或口头结论；
- 首个基准记录见 `docs/archive/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_20260430_LAYOUT_UNIFICATION.md`。
- PR 描述正文模板见 `docs/acceptances/PLATFORM_SHELL_PR_TEMPLATE.md`。

适用范围：

- `frontend/src/core/layout/index.tsx`
- `frontend/src/core/layout/index.css`
- 动态菜单渲染链路
- 菜单图标映射
- 顶部横版导航
- 左侧竖版导航
- 与导航直接相邻的品牌区、页签区、顶部栏布局

不适用范围：

- 单个 `system/*` 页面内部的普通 CRUD 视觉调整
- 不影响导航承载方式的页面内容样式
- 与壳层菜单状态无关的局部弹窗或表单细节

---

## 1. 基本信息

- 验收批次：
- 日期：
- 执行人：
- 所属层级：`platform`
- 关联文档：
  - `docs/acceptances/ACCEPTANCE_CHECKLIST.md`
  - `docs/acceptances/PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md`
  - `docs/designs/FRONTEND_UI_SPEC.md`
- 改动范围：
  - 文件：
  - 目标：
  - 非目标：

---

## 2. 触发原因

- 是否触达以下任一对象：`Yes / No`
  - `frontend/src/core/layout/index.tsx`
  - `frontend/src/core/layout/index.css`
  - 菜单渲染或菜单状态链路
  - 顶部横版导航
  - 左侧竖版导航
- 若为 `Yes`，本模板必须完整填写。
- 若为 `No`，说明为什么本次改动不需要双模式验收：

---

## 3. 固定验收输入

### 3.1 竖版侧栏展开态

- 访问路径：
- 账号角色：
- 菜单层级样本：
  - 一级叶子：
  - 一级分组：
  - 二级叶子：
- 结论：`Pass / Pending / Fail`
- 备注：

### 3.2 竖版侧栏折叠态

- 访问路径：
- 账号角色：
- tooltip / 高亮 / 折叠记忆是否正常：
- 结论：`Pass / Pending / Fail`
- 备注：

### 3.3 横版主导航态

- 访问路径：
- 账号角色：
- 一级菜单样本：
- 结论：`Pass / Pending / Fail`
- 备注：

### 3.4 横版弹出子菜单态

- 访问路径：
- 账号角色：
- 分组菜单样本：
- 结论：`Pass / Pending / Fail`
- 备注：

---

## 4. 固定验收项

逐项填写 `Pass / Pending / Fail`：

| 验收项 | 竖版展开 | 竖版折叠 | 横版主导航 | 横版弹出子菜单 | 备注 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 当前菜单高亮一致 |  |  |  |  |  |
| hover / selected / open 三态一致 |  |  |  |  |  |
| icon badge 尺寸一致 |  |  |  |  |  |
| 间距与文字节奏一致 |  |  |  |  |  |
| 普通叶子菜单表达一致 |  |  |  |  |  |
| 分组菜单表达一致 |  |  |  |  |  |
| 外链菜单表达一致 |  |  |  |  |  |
| 品牌区与导航关系稳定 |  |  |  |  |  |
| 面包屑未错位 |  |  |  |  |  |
| 页签区未错位 |  |  |  |  |  |
| 滚动与溢出行为正常 |  |  |  |  |  |

---

## 5. 固定扫描记录

### 5.1 壳层链路扫描

- `rg "ShellLayoutMode|isHorizontalLayout" frontend/src/core/layout`
- 结果：

### 5.2 旧右栏类名扫描

- `rg "system-page-side|system-page-summary-card|system-page-note|system-page-main-grid|system-page-main" frontend/src`
- 结果：

### 5.3 原生浮层扫描

- `rg "<Modal|<Drawer" frontend/src/modules frontend/src/components`
- 结果：

### 5.4 静态 Modal API 扫描

- `rg "Modal\\.confirm|Modal\\.(success|error|info|warning)" frontend/src`
- 结果：

### 5.5 构建结果

- `cmd /c npm run build`
- 结果：

---

## 6. 例外与挂账

若存在未通过项，必须逐条记录，不接受口头说明。

| 类型 | 文件 | 状态 | 原因 | 是否阻断合入 | 后续动作 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 导航视觉 |  | `Pending / Fail` |  | `Yes / No` |  |
| 交互状态 |  | `Pending / Fail` |  | `Yes / No` |  |
| 响应式 / 溢出 |  | `Pending / Fail` |  | `Yes / No` |  |
| 其他 |  | `Pending / Fail` |  | `Yes / No` |  |

规则：

- 只要存在 `Pending`，就必须同步回写矩阵文档；
- 只要存在 `Fail` 且影响当前模式可用性，就不应判定本轮为完成；
- 不允许出现“竖版通过，横版后补”但仍记为 `Target` 的结论。

---

## 7. 最终结论

- 本轮是否通过双模式验收：`Yes / No`
- 是否允许标记为 `Target`：`Yes / No`
- 是否需要回写矩阵：`Yes / No`
- 结论摘要：

---

## 8. 填写示例

- 验收批次：`platform-shell-2026-04-30-layout-unification`
- 日期：`2026-04-30`
- 执行人：`Codex`
- 改动范围：`frontend/src/core/layout/index.css`
- 目标：`统一竖版与横版导航的 hover / selected / icon badge 规则`
- 非目标：`不调整 system 页面内部内容布局`

示例结论：

- 竖版展开态：`Pass`
- 竖版折叠态：`Pass`
- 横版主导航态：`Pass`
- 横版弹出子菜单态：`Pass`
- 是否允许标记为 `Target`：`Yes`
