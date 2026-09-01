---
title: 主题 Token 参考表 (Theme Tokens Reference)
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-07-09
---

# 主题 Token 参考表 (Theme Tokens Reference)

English version: [THEME_TOKENS_REFERENCE.en.md](./THEME_TOKENS_REFERENCE.en.md)

本文记录当前 `frontend/src/index.css` 的真实实现。它是 light-mode 的单一真相源；文档里不再保留尚未落地的 `--pantheon-*`、暗色模式、通用 spacing/type/motion/z-index 承诺。后续改样式时，优先对齐这里和 `frontend/src/index.css`。

---

## 1. Token 命名约定

当前实现主要使用以下几组变量：

- `--brand-*`：品牌色及其透明度衍生
- `--panel-*`：表面、边框、阴影
- `--text-*`：文本颜色
- `--radius-*`：圆角
- `--shell-*`：壳层间距与布局节奏
- `--color-*`：状态语义色
- `--primary-1..10`：主题 RGB ramp，供 `color-mix()` / `rgba()` 组合使用

兼容别名仍存在，但只用于老代码过渡，不是新语义入口：

- `--danger-soft`
- `--danger-border`
- `--status-success-soft`
- `--status-warning-soft`

---

## 2. 中性色 Token

| Token                   | 值                                                                    | 用途          |
| ----------------------- | --------------------------------------------------------------------- | ------------- |
| `--app-bg`              | `#f7f8fa`                                                             | 页面背景      |
| `--app-grid-line`       | `rgba(29, 33, 41, 0.035)`                                             | 背景网格      |
| `--app-shell-overlay`   | `#ffffff`                                                             | 壳层浮层底色  |
| `--panel-bg`            | `#ffffff`                                                             | 默认表面      |
| `--panel-bg-solid`      | `#ffffff`                                                             | 卡片/控件实底 |
| `--panel-muted`         | `#f7f8fa`                                                             | 弱化区域      |
| `--panel-border`        | `rgba(229, 230, 235, 0.82)`                                           | 默认边框      |
| `--panel-border-strong` | `rgba(209, 213, 219, 0.78)`                                           | 强边框        |
| `--panel-shadow`        | `0 12px 28px rgba(15, 23, 42, 0.055)`                                 | 主阴影        |
| `--panel-shadow-soft`   | `0 1px 2px rgba(15, 23, 42, 0.04), 0 8px 18px rgba(15, 23, 42, 0.03)` | 轻阴影        |
| `--panel-shadow-strong` | `0 18px 42px rgba(15, 23, 42, 0.14)`                                  | 强阴影        |
| `--text-primary`        | `#1f2329`                                                             | 主文本        |
| `--text-secondary`      | `#4e5969`                                                             | 次文本        |
| `--text-tertiary`       | `#86909c`                                                             | 辅助文本      |

---

## 3. 品牌主题

四个内置主题都来自 `:root[data-pantheon-theme=...]`。

### 3.1 indigo

- `--brand-primary`: `#165dff`
- `--brand-primary-soft`: `rgba(22, 93, 255, 0.12)`
- `--brand-primary-muted`: `rgba(22, 93, 255, 0.08)`
- `--brand-gradient`: `#165dff`
- `--shell-sider-bg`: `#162033`
- `--shell-sider-elevated`: `#1d2940`
- `--shell-brand-shadow`: `0 12px 28px rgba(22, 93, 255, 0.24)`
- `--login-glow`: `rgba(22, 93, 255, 0.12)`

### 3.2 emerald

- `--brand-primary`: `#00a870`
- `--brand-primary-soft`: `rgba(0, 168, 112, 0.13)`
- `--brand-primary-muted`: `rgba(0, 168, 112, 0.08)`
- `--brand-gradient`: `#00a870`
- `--shell-sider-bg`: `#143126`
- `--shell-sider-elevated`: `#1a3d32`
- `--shell-brand-shadow`: `0 12px 28px rgba(0, 168, 112, 0.26)`
- `--login-glow`: `rgba(0, 168, 112, 0.14)`

### 3.3 violet

- `--brand-primary`: `#722ed1`
- `--brand-primary-soft`: `rgba(114, 46, 209, 0.13)`
- `--brand-primary-muted`: `rgba(114, 46, 209, 0.08)`
- `--brand-gradient`: `#722ed1`
- `--shell-sider-bg`: `#22193b`
- `--shell-sider-elevated`: `#2c2350`
- `--shell-brand-shadow`: `0 12px 28px rgba(114, 46, 209, 0.26)`
- `--login-glow`: `rgba(114, 46, 209, 0.14)`

### 3.4 slate

- `--brand-primary`: `#334155`
- `--brand-primary-soft`: `rgba(51, 65, 85, 0.12)`
- `--brand-primary-muted`: `rgba(51, 65, 85, 0.08)`
- `--brand-gradient`: `#334155`
- `--shell-sider-bg`: `#182235`
- `--shell-sider-elevated`: `#243148`
- `--shell-brand-shadow`: `0 12px 28px rgba(51, 65, 85, 0.24)`
- `--login-glow`: `rgba(51, 65, 85, 0.12)`

> 每个主题还会定义 `--primary-1..10` 的 RGB ramp，用于 `color-mix()` 和透明度叠加。那组数值保持在 `frontend/src/index.css`，这里不重复抄录。

---

## 4. 状态色 Token

| Token                | 值                         | 用途     |
| -------------------- | -------------------------- | -------- |
| `--color-success`    | `#00b42a`                  | 成功     |
| `--color-success-bg` | `rgba(0, 180, 42, 0.12)`   | 成功背景 |
| `--color-warning`    | `#ff7d00`                  | 警告     |
| `--color-warning-bg` | `rgba(255, 125, 0, 0.12)`  | 警告背景 |
| `--color-error`      | `#f53f3f`                  | 错误     |
| `--color-error-bg`   | `rgba(245, 63, 63, 0.12)`  | 错误背景 |
| `--color-info`       | `#3491fa`                  | 信息     |
| `--color-info-bg`    | `rgba(52, 145, 250, 0.12)` | 信息背景 |

别名映射：

- `--danger-soft` -> `--color-error-bg`
- `--danger-border` -> `color-mix(in srgb, var(--color-error) 32%, transparent)`
- `--status-success-soft` -> `--color-success-bg`
- `--status-warning-soft` -> `--color-warning-bg`

---

## 5. 间距 Scale

当前没有通用 `--space-*` 阶梯，壳层直接用 `--shell-*` 变量控制节奏：

| Token                                    | 值              | 用途                 |
| ---------------------------------------- | --------------- | -------------------- |
| `--shell-page-gap`                       | `16px`          | 页面主间距           |
| `--shell-page-header-gap`                | `12px`          | 页面头部间距         |
| `--shell-panel-body-padding`             | `18px`          | 面板内边距           |
| `--shell-panel-head-min-height`          | `56px`          | 面板头最小高度       |
| `--shell-page-split-gap`                 | `16px`          | 分栏间距             |
| `--shell-page-main-gap`                  | `16px`          | 主栏间距             |
| `--shell-page-side-gap`                  | `12px`          | 侧栏间距             |
| `--shell-table-cell-padding-y`           | `10px`          | 表格单元纵向内边距   |
| `--shell-table-cell-padding-x`           | `14px`          | 表格单元横向内边距   |
| `--shell-table-card-padding`             | `12px 14px 6px` | 表格卡片 body 内边距 |
| `--shell-table-pagination-padding`       | `12px 14px 2px` | 表格分页区内边距     |
| `--shell-table-action-min-height`        | `32px`          | 表格动作按钮最小高度 |
| `--shell-control-min-height`             | `32px`          | 通用控件最小高度     |
| `--shell-filter-body-padding`            | `14px 16px 6px` | 筛选面板内边距       |
| `--shell-filter-control-min-height`      | `34px`          | 筛选控件最小高度     |
| `--shell-filter-form-item-margin-bottom` | `12px`          | 筛选表单项间距       |
| `--shell-filter-label-padding-bottom`    | `4px`           | 筛选标签下边距       |
| `--shell-list-actions-gap`               | `8px 12px`      | 列表动作间距         |
| `--shell-action-bar-gap`                 | `8px 12px`      | 批量动作条间距       |
| `--shell-action-bar-min-height`          | `32px`          | 批量动作条最小高度   |
| `--shell-table-head-gap`                 | `8px 12px`      | 表头动作区间距       |
| `--shell-governance-select-width`        | `200px`         | 治理面板选择器宽度   |

---

## 6. 字号

当前实现没有共享的字号阶梯 token。`body` 仍使用系统字体栈，页面和组件里的字号由各自 CSS 直接写入。

真相是：

- 没有 `--font-size-*` / `--line-height-*`
- 没有 `Source Sans 3` 全局加载
- 代码字体与数据字体的特殊处理只存在于局部组件样式里

如果后续要做字号统一，需要先补实现，再补文档。

---

## 7. 圆角

| Token              | 值                 | 用途                 |
| ------------------ | ------------------ | -------------------- |
| `--radius-xs`      | `4px`              | 最小圆角             |
| `--radius-sm`      | `4px`              | 小按钮 / tag         |
| `--radius-md`      | `6px`              | 输入框 / 普通控件    |
| `--radius-lg`      | `8px`              | 卡片 / 浮层          |
| `--radius-overlay` | `8px`              | Overlay / Modal 基线 |
| `--radius-control` | `var(--radius-md)` | 控件语义别名         |
| `--radius-action`  | `var(--radius-sm)` | 动作按钮语义别名     |
| `--radius-pill`    | `999px`            | 胶囊 / 徽标          |

当前没有 `--radius-xl`，也不鼓励超出这条阶梯的自由圆角。

---

## 8. 阴影

| Token                   | 值                                                                    | 用途       |
| ----------------------- | --------------------------------------------------------------------- | ---------- |
| `--panel-shadow`        | `0 12px 28px rgba(15, 23, 42, 0.055)`                                 | 主卡片阴影 |
| `--panel-shadow-soft`   | `0 1px 2px rgba(15, 23, 42, 0.04), 0 8px 18px rgba(15, 23, 42, 0.03)` | 轻提升     |
| `--panel-shadow-strong` | `0 18px 42px rgba(15, 23, 42, 0.14)`                                  | 强浮层     |

没有单独的 dark-mode 阴影 token，因为当前运行时不提供 dark mode。

---

## 9. 动效

当前实现没有 `--motion-*` token。组件里直接写了 `0.18s`、`0.2s` 这类过渡时间，并通过 `prefers-reduced-motion` 做降级。

这意味着：

- 动效是实现细节，不是文档承诺
- 如果未来要抽象 motion token，需要先改代码，再改文档

---

## 10. Z-index

当前没有共享 `--z-*` 阶梯。层级主要依赖 Arco 默认层级和局部布局实现。

这意味着：

- 下拉、Tooltip、Modal、Drawer 的层级裁决仍在组件库和局部代码里
- 如果后续需要统一浮层层级，先补实现，再补 token 文档

---

## 11. 主题切换实现约定

- 主题通过 `:root[data-pantheon-theme="indigo|emerald|violet|slate"]` 挂载
- 切换主题只改 CSS 变量值，不重建 DOM
- 当前只支持 light mode
- 运行时没有 `data-color-mode`
- `DARK_MODE_DESIGN.md` 目前是 Deferred，不是已落地规范

---

## 12. 验收

- 4 个主题都能在同一套壳层里正常渲染
- Dashboard、列表页、row actions、状态提示都消费语义 token
- 业务代码不再新增 `--pantheon-*` 运行时引用
- `dashboard.css` 已纳入 shell visual contract 检查
- 暗色模式不再被当成已交付能力描述
