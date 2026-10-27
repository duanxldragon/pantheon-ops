---
title: 无障碍设计规范 (Accessibility)
doc_type: Design
layer: platform / system/*
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-05-11
---

# 无障碍设计规范 (Accessibility)

English version: [ACCESSIBILITY.en.md](./ACCESSIBILITY.en.md)

本文是 Pantheon Base 的 a11y 设计与验收规范，作为 `FRONTEND_UI_SPEC.md` 的补充。目标是让所有键盘、屏幕阅读器、低视力用户都能完成核心管理任务。

不重复 `FRONTEND_UI_SPEC.md` 中已有的「视觉对比」「字号尺度」「色彩 token」规则，本文聚焦交互可达性。

---

## 1. 适用范围与最低标准

| 维度 | 标准 | 备注 |
|---|---|---|
| 合规基线 | WCAG 2.1 AA | 不强求 AAA，但关键流程（登录、表单提交、批量操作）需要满足 AA |
| 键盘 | 所有交互必须可纯键盘完成 | 含 Tab、Shift+Tab、Enter、Esc、方向键、空格 |
| 屏幕阅读器 | 主流设备验证：NVDA + Chrome (Win)、VoiceOver + Safari (Mac) | 不强求 JAWS |
| 色彩对比 | 正文 ≥ 4.5:1，大字号 ≥ 3:1，UI 控件边界 ≥ 3:1 | 暗色模式同样标准 |
| 触控目标 | 移动端 ≥ 44×44 CSS px | 桌面端无强制，但建议 ≥ 32×32 |

---

## 2. 键盘导航

### 2.1 焦点顺序

- Tab 顺序必须沿可视化阅读顺序（从上到下、左到右）
- 跳过装饰元素（纯视觉 icon、分隔线）
- 列表/表格内部使用方向键，不靠 Tab 逐格穿越
- Modal/Drawer 打开时，焦点必须**自动**进入容器，关闭时**返回**到触发它的元素

### 2.2 焦点可见

- 所有可聚焦元素必须有**明显的**焦点环
- 焦点环不依赖颜色单独传达（必须有外形变化：outline、shadow、border）
- 焦点环对比度 ≥ 3:1 (相对相邻背景)
- **禁止** `outline: none` 不补救

### 2.3 Skip Link

- 登录页之后的所有受保护页面，顶部必须有一个「跳到主内容」隐式 link（Tab 第一次按下时显示）
- 跳到主内容的 anchor 为页面主 `<main>` 或 `id="page-content"`

### 2.4 快捷键

- 全局快捷键：`/` 聚焦搜索；`Esc` 关闭最上层浮层；`?` 显示快捷键面板
- 局部快捷键由模块自行决定，但必须不与全局冲突
- 所有快捷键必须有可视化的提示（kbd tooltip 或快捷键面板）

---

## 3. ARIA 与语义化

### 3.1 必须遵守的 ARIA 规则

| 场景 | 要求 |
|---|---|
| 纯 icon 按钮 | 必须有 `aria-label` 或 `title` |
| 装饰性 icon | 必须 `aria-hidden="true"` |
| Dialog/Modal | 必须 `role="dialog"` + `aria-labelledby` 指向标题 |
| Drawer | 同 Dialog |
| Toast/Notification | 必须 `role="status"` 或 `role="alert"`（按严重程度） |
| 表格 | 必须 `<th scope="col">` 列头；行选择有 `aria-selected` |
| 表单错误 | 错误消息 `id` 必须被字段 `aria-describedby` 引用；字段 `aria-invalid="true"` |
| Loading state | 容器 `aria-busy="true"` |
| 路由切换 | 切换完成后必须有一次 `role="status"` 公告新页面标题 |

### 3.2 不要做的事

- 不要用 `<div>` 替代 `<button>` / `<a>` 后再加 `role="button"`——直接用语义化标签
- 不要在父元素上 `tabindex="-1"`，会让其子元素也被屏蔽
- 不要 `aria-hidden="true"` 一个仍能被键盘聚焦的元素

---

## 4. 表单可达性

### 4.1 必须

- 每个输入控件必须有可见 `<label>`，`for` 属性指向 `id`
- placeholder **不能**替代 label
- 必填字段：视觉标记 + `aria-required="true"`
- 错误消息：在 DOM 中紧跟字段，`aria-describedby` 关联
- 验证时机：blur 时验证；提交时聚焦到**第一个**错误字段并播报错误

### 4.2 复合控件

- 单选/多选组：必须包在 `<fieldset>` + `<legend>` 中
- 自定义下拉：实现 ARIA Listbox 模式（`role="combobox"` + 键盘上下选择 + Enter 确认 + Esc 关闭）
- 日期选择器：必须支持纯键盘输入（不强制只能选择）

---

## 5. 动态内容与异步状态

| 场景 | 公告策略 |
|---|---|
| 列表筛选后内容变更 | `role="status"` 公告「X 条结果」 |
| 表单提交成功 | `role="status"` 公告「保存成功」 |
| 表单提交失败 | `role="alert"` 公告「提交失败：<错误描述>」 |
| 路由切换 | `role="status"` 公告新页面 title |
| 异步加载完成 | 内容容器 `aria-busy` 切换 `true → false` |
| 实时数据更新（dashboard） | 不每次公告；只在用户主动刷新时公告 |

---

## 6. 色彩与对比度

- 正文文字与背景对比度 ≥ 4.5:1
- 18px+ 或 14px+ 粗体 → 大字号阈值 3:1
- 表单 placeholder 也必须 ≥ 4.5:1（曾经流行的浅灰 placeholder 不合规）
- 错误/警告/成功状态**不能**只靠颜色传达，必须配 icon 或文案
- 详细 token 见 `THEME_TOKENS_REFERENCE.md` 与 `DARK_MODE_DESIGN.md`

---

## 7. 移动端触控可达性

- 触控目标 ≥ 44×44 CSS px（含 padding）
- 触控目标之间间距 ≥ 8 px
- 长按、双击、滑动等手势必须有**键盘替代**或显式 fallback 按钮
- 移动端 Drawer/Modal 必须支持 swipe-down 关闭，同时**也**有 close 按钮

---

## 8. 验收清单（合入前）

每个新页面或新组件必须勾选：

- [ ] 纯键盘走通主流程（Tab → Enter）
- [ ] 关闭 CSS（仅看 DOM 结构）时仍能理解页面意图
- [ ] 屏幕阅读器读出所有交互元素的名称、角色、状态
- [ ] 表单错误能被屏幕阅读器播报
- [ ] 焦点环可见，且对比度足够
- [ ] 主要状态（loading/empty/error/forbidden）都有 ARIA 标识
- [ ] 浅色和深色主题下对比度都达标
- [ ] Tab 顺序符合阅读顺序
- [ ] Modal 打开时焦点进入、关闭时焦点返回
- [ ] 移动端断点下触控目标尺寸合规

---

## 9. 自动化检查

- 构建前：`npm run check:a11y` 跑 `axe-core` 规则集（计划项）
- E2E 流程：在 gstack browse 测试中加 a11y 断言（路由切换公告、焦点返回、错误聚焦）

---

## 10. 关联文档

- `FRONTEND_UI_SPEC.md` §9 状态设计、§10 权限交互
- `THEME_TOKENS_REFERENCE.md` §色彩 token
- `DARK_MODE_DESIGN.md` §对比度反转策略
- `EMPTY_LOADING_ERROR_STATES.md` §状态变体的 ARIA 标注
