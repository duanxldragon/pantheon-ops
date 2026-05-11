# Platform 壳层 PR 描述样例

更新时间：2026-04-30

类型：Acceptance
归属层：platform
状态：Archived

适用批次：`platform-shell-2026-04-30-layout-unification`

基于：

- PR 模板：`docs/acceptances/PLATFORM_SHELL_PR_TEMPLATE.md`
- 双模式验收记录：`docs/archive/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_20260430_LAYOUT_UNIFICATION.md`

以下正文可直接复制到后续 PR、阶段记录或阶段收口说明中。

---

```md
## 变更范围

- 层级：`platform`
- 目标：统一竖版侧栏与横版顶栏的菜单状态语言，收口 hover / selected / icon badge / popup 规则
- 非目标：不调整 `system/*` 页面内部内容布局，不调整动态菜单数据契约，不调整权限判断链路
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

## 简版样例

若提交场景只允许短摘要，可使用以下版本：

```md
## 变更范围

- 层级：`platform`
- 目标：统一竖版侧栏与横版顶栏的菜单状态语言
- 非目标：不调整 `system/*` 页面内容布局
- 影响文件：`frontend/src/core/layout/index.css`

## 结果摘要

- 双模式验收文档：`docs/archive/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_20260430_LAYOUT_UNIFICATION.md`
- 构建结果：`npm run build Passed`
- 扫描结果：`旧右栏 0 命中 / 原生浮层仅平台封装 / 静态 Modal API 仅平台封装内部 / 双模式链路已验证`
- 矩阵状态：`Target`
```

---

## 使用说明

- 若后续批次不是 `layout-unification`，只替换批次名、文档链接、影响文件和摘要结论，不改整体结构。
- 若存在 `Pending`，必须删除 `None` 并填写真实挂账项，不能沿用本样例。
- 若构建或扫描未通过，必须把 `Passed / Target` 改成真实结果，不能直接复制本样例结论。

