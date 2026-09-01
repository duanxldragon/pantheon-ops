---
title: Visual Quality Protocol
doc_type: Contract
layer: method
status: Active
updated_at: 2026-06-18
---

# Visual Quality Protocol

本文定义 UI 任务的视觉质量门。它是工具无关 Harness 协议的一部分。

## 1. 适用范围

任何任务只要影响以下内容，就必须执行视觉质量门：

- 页面布局
- 前端组件
- dashboard/admin/workbench
- 表格、表单、图表
- 导航、弹窗、抽屉、工具栏
- 交互状态、loading、empty、error、permission denied
- responsive/mobile viewport
- 视觉设计系统、颜色、字体、间距、图标、动效

## 2. 默认 Skill

首选视觉质量 skill：

```text
impeccable
```

Codex 全局位置：

```text
C:\Users\xiaolong\.codex\skills\impeccable\SKILL.md
```

如果当前工具没有 Codex skill 能力，也必须按同一协议执行：

1. 先读取本文。
2. 使用 `.agents/prompts/implementation.md`、`.agents/prompts/review.md`、`.agents/prompts/qa.md`。
3. 将视觉证据写入 `.harness/evidence/<task-id>/`。

## 3. 与其他设计工具关系

- `impeccable` 是视觉质量门。
- `ui-ux-pro-max` 可用于颜色、字体、布局、可访问性、设计系统细化。
- gstack/browser/Playwright/人工截图可用于视觉证据。
- Figma 或其他设计工具只能作为输入或证据，不是唯一事实源。

## 4. UI Task Manifest 要求

UI 任务必须在 task manifest 的 `verificationPlan.visualEvidence` 中补充：

- UI surface 类型。
- 目标视觉感受。
- desktop/mobile/relevant viewport 验证计划。
- empty/loading/error/permission 等 state 验证计划。
- 如有固定入口，补充 route 验证计划。

如果仓库仍保留 task packet，它可以保留同样的人类可读说明；但自动化闭环以 task manifest 为准，不再从 task packet 推断视觉校验语义。

## 5. Review Gate

UI review 必须检查：

- 是否使用 `impeccable` 或同等视觉质量门。
- 是否保留 rendered evidence 或记录未运行原因。
- 是否存在文本溢出、重叠、错位、弱对比、状态缺失。
- 是否符合当前项目定义的 UI 风格：克制、清晰、可扫描，不引入无理由的第二套视觉语言。
- 是否有清晰 keyboard focus：可交互元素必须有 `:focus-visible` 或等效可见焦点，不允许只依赖 hover。
- 是否尊重 `prefers-reduced-motion: reduce`，并避免无意义的大面积装饰性动效。
- 是否使用稳定控制尺寸：按钮、图标按钮、输入、选择器、日期选择器等不应在 hover/loading/focus 时造成布局跳动。
- 是否避免装饰污染：颜色、字体、gradient、阴影、圆角必须收敛到项目的设计 token 或明确视觉系统。
- 是否只存在一种页面模式：同类页面应复用项目标准组件，而不是在同一信息层里混入第二套 hero、overview、metric、toolbar、dialog、table-card 视觉模式。
- 是否保持治理摘要、筛选区、表格区、表单区的结构顺序一致，避免信息定位重复或视觉层级冲突。
- 对话框和抽屉必须走项目标准入口，不能直接混用多套尺寸、footer、padding、关闭行为。

P0/P1 视觉问题不能 approved。

## 5.1 Mechanical Contract

如果项目已经定义了视觉合同脚本，应把它纳入验证计划并在 evidence 中记录结果。

该检查推荐覆盖：

- header / tab / breadcrumb 不裁剪文本。
- 标准筛选区、列表区、批量操作区、表格区的 spacing、radius、控制高度保持一致。
- 不允许视觉 token 泄漏到未批准的原始颜色、阴影、字号或 font-weight。
- 同类系统页不得同时存在多套 hero、overview、table-card 视觉模式。

如项目存在特定 overlay，可由 overlay 覆盖本文并提供更严格的机械检查规则。

## 6. Evidence

推荐证据：

```text
.harness/evidence/<task-id>/
  summary.md
  commands.json
  screenshots/
  visual-review.md
```

如果无法截图，必须记录：

- 未截图原因
- 风险
- 后续验证方式

不能用“看起来应该没问题”作为视觉验证结论。

对于声明了 `verificationPlan.visualEvidence` 的任务，`commands.json` 中的 `browserEvidence` 负责承载机器可校验的覆盖结果：

- `viewport`
- `checkedStates`
- `url`（用于 route 覆盖）

截图和 visual gap 仍然有价值，但它们是辅助证据；没有 `browserEvidence` 覆盖时，任务不会被视为完成了 manifest 里的视觉计划。

## 6.1 Blocking Rule

If a UI task manifest declares a visual verification plan and strict mode is enabled in CI, missing `browserEvidence` coverage for the declared viewport/state/route plan is a blocking harness failure. Missing screenshots and missing explicit visual-gap records remain warnings when no other visual signal exists.
