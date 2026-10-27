---
title: 响应式断点与移动端适配 (Mobile Responsive Breakpoints)
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-05-11
---

# 响应式断点与移动端适配 (Mobile Responsive Breakpoints)

English version: [MOBILE_RESPONSIVE_BREAKPOINTS.en.md](./MOBILE_RESPONSIVE_BREAKPOINTS.en.md)

本文定义 Pantheon Base 的断点、容器宽度策略、组件在各断点的退化规则。配合 `FRONTEND_UI_SPEC.md` §12 响应式规范使用。

UI Spec §12 给响应式**原则**；本文给具体**断点数值 + 每类组件的行为表**。

优先级约束：

- 涉及 **breakpoint 数值、列表页退化行为、移动端组件降级规则** 时，**一律以本文为准**
- `FRONTEND_UI_SPEC.md` 中的响应式章节只保留设计原则、桌面优先立场与页面体验目标，不再作为断点数值真相源

---

## 1. 优先级与边界

- Pantheon 是**桌面优先**的企业后台；移动端是降级体验，**不是**移动端优先重设计
- 主要使用场景：桌面 1280-1920；笔记本 1024-1366；偶发平板 768-1024；极少手机 360-480
- **不**优化触屏游戏化交互、长滚动卡片堆叠这类移动端原生模式

---

## 2. 断点定义

```
xs:  < 480px       手机竖屏
sm:  480 - 767px   手机横屏 / 小平板
md:  768 - 1023px  平板
lg:  1024 - 1279px 笔记本
xl:  1280 - 1599px 桌面（默认目标）
2xl: ≥ 1600px      宽桌面 / 4K
```

CSS 变量：
```css
:root {
  --pantheon-breakpoint-sm: 480px;
  --pantheon-breakpoint-md: 768px;
  --pantheon-breakpoint-lg: 1024px;
  --pantheon-breakpoint-xl: 1280px;
  --pantheon-breakpoint-2xl: 1600px;
}
```

媒体查询模板：
```css
@media (min-width: 768px) { /* md+ */ }
@media (min-width: 1024px) { /* lg+ */ }
@media (min-width: 1280px) { /* xl+ */ }
```

---

## 3. 壳层结构在各断点的行为

| 断点 | Sidebar | TopBar | Tab Bar | Drawer/Modal |
|---|---|---|---|---|
| `xl` / `2xl` | 展开 240px | 完整 | 显示 | 80% 容器宽，max 1200px |
| `lg` | 收起 64px (rail 模式) | 完整 | 显示 | 90% 容器宽 |
| `md` | 抽屉式（点击 toggle 弹出） | 紧凑 | 显示，可滚动 | 全屏 - 32px |
| `sm` / `xs` | 抽屉式 | 仅 logo + menu icon + user | 隐藏，移到顶部 actions | 全屏 |

约束：
- `lg` 起开始进入 rail 模式（仅显示 icon），不强制展开
- `md` 起 Sidebar 完全隐藏到抽屉，不挤占主内容
- `xs` 模式下 TopBar 高度减半（40px），让出垂直空间

---

## 4. ListPage 在各断点的行为

| 断点 | FilterPanel | Table | 操作条 |
|---|---|---|---|
| `xl` / `2xl` | 顶部一行 6-8 字段 | 完整列宽，所有列可见 | 顶部右对齐 |
| `lg` | 顶部一行 4-5 字段 + "更多" 折叠 | 隐藏 4 个低优先级列 | 顶部右对齐 |
| `md` | 折叠为单按钮 "筛选"，点击展开 Drawer | 隐藏 8+ 列，保留核心 3-5 列 | 主操作 right 显示 |
| `sm` / `xs` | 折叠 Drawer | 卡片式渲染（每行一卡，不再用 table） | 仅一个主操作 + overflow menu |

列优先级声明示例：
```ts
columns: [
  { key: 'name',   priority: 1 },  // xs+ 显示
  { key: 'status', priority: 2 },  // sm+ 显示
  { key: 'owner',  priority: 3 },  // md+ 显示
  { key: 'created', priority: 4 }, // lg+ 显示
  { key: 'updated', priority: 5 }, // xl+ 显示
]
```

---

## 5. DetailPage 在各断点的行为

| 断点 | 布局 | Sidebar/Outline | Section 排列 |
|---|---|---|---|
| `xl` / `2xl` | 左 outline 200px + 主区 + 右辅助 280px | 显示 | 多列网格 |
| `lg` | 主区 + 右辅助 280px | 折叠为 sticky 内嵌目录 | 多列网格 |
| `md` | 单列主区 | 折叠为顶部 anchor 链接 | 双列 |
| `sm` / `xs` | 单列 | 隐藏 | 单列堆叠 |

---

## 6. FormPage 在各断点的行为

| 断点 | 表单列数 | label 位置 | 操作按钮 |
|---|---|---|---|
| `xl` / `2xl` | 2-3 列 | label 在字段左侧 (180px 宽) | sticky 底部右对齐 |
| `lg` | 2 列 | label 在字段左侧 (140px 宽) | sticky 底部右对齐 |
| `md` | 1-2 列 | label 在字段上方 | sticky 底部居中 |
| `sm` / `xs` | 1 列 | label 在字段上方 | 全宽固定底部 |

---

## 7. Dashboard 在各断点的行为

| 断点 | 网格列数 | 卡片高度 |
|---|---|---|
| `2xl` | 4 列 | 自适应内容 |
| `xl` | 3 列 | 自适应 |
| `lg` | 2-3 列 | 自适应 |
| `md` | 2 列 | 自适应 |
| `sm` | 1 列 | 自适应 |
| `xs` | 1 列 | 自适应；图表卡片高度减半，转用条形精简 |

---

## 8. 组件级退化

### 8.1 Table

- `md` 以下隐藏部分列（按 `priority`）
- `sm` 以下完全切换为卡片渲染
- 行内操作 (`...`) 始终保留

### 8.2 Modal / Drawer

- `md` 以上：Modal 居中，Drawer 从右侧滑入 400-600px 宽
- `md` 以下：Modal 全屏，Drawer 从底部滑入（bottom sheet 模式）

### 8.3 Dropdown / Select

- 桌面：常规下拉
- `sm` 以下：转为全屏 ActionSheet 风格

### 8.4 Datepicker

- 桌面：双月日历
- `md` 以下：单月日历
- `sm` 以下：原生 `<input type="date">` 降级

### 8.5 Tooltip

- `md` 以下：不显示 hover tooltip（无 hover 事件）
- 关键 tooltip 内容必须同时有可见替代（icon 旁文字、aria-label）

### 8.6 Form Field

- 输入框最小高度桌面 32px，移动端 40px
- 按钮最小高度桌面 32px，移动端 44px (a11y 触控目标)

---

## 9. 容器宽度策略

| 容器 | 最大宽度 |
|---|---|
| 业务页面主区 | min(100% - 32px, 1440px) |
| 表单容器 | min(100%, 720px)；超过 lg 时居中 |
| 文章/详情 | min(100%, 880px) |
| Dashboard 卡片网格 | 100% (无 max) |

---

## 10. 字号在各断点的调整

字号 token 不随断点变化——`THEME_TOKENS_REFERENCE.md` §6 的尺度全局生效。

但**密度** (`densityMode`) 可以由用户选择 `compact / default / comfortable`，影响行高和 padding：

| 密度 | 行高倍数 | padding 倍数 |
|---|---|---|
| `compact` | 0.85 | 0.75 |
| `default` | 1.0 | 1.0 |
| `comfortable` | 1.15 | 1.25 |

移动端建议默认 `comfortable` 以增加触控目标；桌面默认 `default`。

---

## 11. 屏幕方向

- 强制竖屏的场景：无（不强制）
- 横屏时（手机/平板横放）：保持响应式断点判断；不要为横屏单独设计
- iframe 内嵌的第三方页面横竖屏由对方负责

---

## 12. 验收

每个新页面合入前必须截图：

- `xl` (1440 宽)
- `lg` (1280 宽)
- `md` (820 宽)
- `sm` (414 宽)

最低标准：
- 所有交互元素在 4 个断点下均可访问
- 不出现水平滚动条
- 不出现重叠 / 截断
- 触控目标尺寸合规（见 `ACCESSIBILITY.md` §7）

---

## 13. 关联

- `FRONTEND_UI_SPEC.md` §12 响应式规范（原则）
- `FRONTEND_PAGE_TEMPLATES.md` 各页面类型骨架
- `BACKOFFICE_STYLE_CONSTRAINTS.md` 后台风格硬约束
- `ACCESSIBILITY.md` §7 移动端触控可达性
