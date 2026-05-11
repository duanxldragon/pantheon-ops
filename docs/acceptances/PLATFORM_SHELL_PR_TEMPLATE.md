# Platform 壳层 PR 描述模板

更新时间：2026-04-30

类型：Acceptance
归属层：platform
状态：Active

本文用于 `platform` 壳层相关改动的 PR 描述、阶段记录说明或阶段收口说明。

快速 checklist 片段：

- `docs/acceptances/PLATFORM_SHELL_PR_CHECKLIST_SNIPPET.md`

适用范围：

- `frontend/src/core/layout/index.tsx`
- `frontend/src/core/layout/index.css`
- 动态菜单渲染链路
- 菜单图标映射
- 顶部横版导航
- 左侧竖版导航
- 与导航直接相邻的品牌区、页签区、顶部栏布局

不适用范围：

- 单个 `system/*` 页面内部 CRUD 微调
- 与壳层导航无关的局部视觉修补

---

## 1. 使用规则

- 后续壳层 PR、阶段记录、阶段收口说明统一使用本模板。
- 不允许只贴截图，不允许只写口头结论。
- 如果本轮触达壳层导航链路，必须附双模式验收文档链接。
- 如果本轮存在 `Pending`，必须同时附矩阵文档链接和挂账位置。

---

## 2. 固定模板

```md
## 变更范围

- 层级：`platform`
- 目标：
- 非目标：
- 影响文件：

## 结果摘要

- 双模式验收文档：`<doc link>`
- 构建结果：`npm run build Passed / Failed`
- 扫描结果：`旧右栏 / 原生浮层 / 静态 Modal API / 双模式链路`
- 矩阵状态：`Target / Pending`

## 文档链接

- 双模式验收记录：`<doc link>`
- 验收矩阵：`docs/acceptances/PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md`
- 验收清单：`docs/acceptances/ACCEPTANCE_CHECKLIST.md`
- 工作流要求：`docs/designs/WORKFLOW.md`

## 挂账说明

- Pending 项：
- 文件位置：
- 阻断原因：
- 后续动作：
```

---

## 3. 文档链接格式

统一使用以下格式：

- 双模式验收文档：`docs/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_<yyyymmdd>_<slug>.md`
- 验收矩阵：`docs/acceptances/PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md`
- 验收模板：`docs/acceptances/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md`
- PR 描述模板：`docs/acceptances/PLATFORM_SHELL_PR_TEMPLATE.md`

推荐 slug：

- `layout-unification`
- `nav-token-tightening`
- `menu-popup-alignment`
- `shell-brand-refresh`

---

## 4. 首个基准示例

当前首个真实验收样例：

- `docs/archive/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_20260430_LAYOUT_UNIFICATION.md`
- 对应 PR 正文样例：`docs/archive/PLATFORM_SHELL_PR_SAMPLE_20260430_LAYOUT_UNIFICATION.md`
- 可直接粘贴的 checklist 片段：`docs/acceptances/PLATFORM_SHELL_PR_CHECKLIST_SNIPPET.md`

基于该样例的摘要写法示例：

```md
## 变更范围

- 层级：`platform`
- 目标：统一竖版侧栏与横版顶栏的菜单状态语言
- 非目标：不调整 `system/*` 页面内部内容布局
- 影响文件：`frontend/src/core/layout/index.css`

## 结果摘要

- 双模式验收文档：`docs/archive/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_20260430_LAYOUT_UNIFICATION.md`
- 构建结果：`npm run build Passed`
- 扫描结果：`旧右栏 0 命中 / 原生浮层仅平台封装 / 静态 Modal API 仅平台封装内部 / 双模式链路已验证`
- 矩阵状态：`Target`

## 文档链接

- 双模式验收记录：`docs/archive/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_20260430_LAYOUT_UNIFICATION.md`
- 验收矩阵：`docs/acceptances/PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md`
- 验收清单：`docs/acceptances/ACCEPTANCE_CHECKLIST.md`
- 工作流要求：`docs/designs/WORKFLOW.md`

## 挂账说明

- Pending 项：`None`
- 文件位置：`-`
- 阻断原因：`-`
- 后续动作：`继续作为后续壳层改动的基准样例`
```

---

## 5. 禁止事项

- 不允许省略“层级：`platform`”
- 不允许省略双模式验收文档链接
- 不允许把 `Pending` 写成“后续再看”而不给文件位置
- 不允许把扫描结果写成“已检查”而不给结论摘要
- 不允许用单页视觉截图替代壳层双模式验收记录
