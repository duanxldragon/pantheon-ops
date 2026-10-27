---
title: 暗色模式设计 (Dark Mode Design)
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-07-10
---

# 暗色模式设计 (Dark Mode Design)

English version: [DARK_MODE_DESIGN.en.md](./DARK_MODE_DESIGN.en.md)

暗色模式已落地（2026-07-10）。运行时通过 `<html data-color-mode="light|dark">` + `body[arco-theme]` 双属性切换，token 真相源见 `frontend/src/index.css` 的 `:root[data-color-mode='dark']` 块与 `THEME_TOKENS_REFERENCE.md`。实现要点：`frontend/src/core/theme/colorMode.ts`（读取顺序：本地偏好 → 系统 `prefers-color-scheme`；`index.html` 内联脚本在首绘前写入属性以防闪烁）；切换入口在应用壳层的“外观模式”偏好区。对比度由 `frontend/scripts/check-contrast.mjs` 校验（正文/链接文本双模式 WCAG AA）。下文保留设计原则作为后续演进依据。

---

## 1. 设计目标

这些目标是未来做暗色模式时才需要满足的要求：

- 不是简单反色
- 同一品牌主题在 dark mode 下仍可识别
- 背景不用纯黑
- 文本不用纯白
- 焦点环仍要满足可读性和对比度要求

---

## 2. Token 反转策略

当前未实现。若未来恢复这个方案，应该按语义角色重定义，而不是机械取反。

参考方向：

| 语义         | 未来 dark 方向     |
| ------------ | ------------------ |
| 页面背景     | 深蓝灰，而不是纯黑 |
| 默认表面     | 比页面背景更亮一档 |
| 浮层表面     | 再亮一档           |
| 主文本       | 接近白，但不是纯白 |
| 品牌色       | 更亮、略降饱和     |
| 浅色品牌背景 | 反向为深色品牌背景 |
| 阴影         | 更深、更高透明度   |

---

## 3. 品牌色在 Dark 下的调整规则

如果未来要做 dark mode，品牌色建议：

1. 提亮
2. 略降饱和
3. 保持色相
4. 把 subtle 背景反向成深色品牌面板

---

## 4. 不能简单反转的元素

这些元素将来也不能靠一键反色处理：

- 业务截图
- 公司 logo
- 业务图表
- 代码块
- zebra 表格
- Toast / Notification 边框

---

## 5. 阴影策略

未来的 dark mode 需要更深、更高透明度的阴影，必要时再加轻微高光边缘，让浮层与背景分层清楚。

---

## 6. 焦点环

如果未来做 dark mode，焦点环应使用提亮后的品牌色，并保持足够的可见度。

---

## 7. 切换行为

当前运行时**不提供切换**。若未来恢复：

- 切换只改根属性
- CSS 变量重新计算
- 不重载页面
- 不重建组件树

---

## 8. 暗色模式专属注意

未来若落地，还要额外处理：

- 透明 PNG 的边缘
- iframe 内部页面的模式同步
- print 输出强制回 light
- light / dark 双模式截图证据

---

## 9. 实现 checklist

当前全部视为未实现：

- `<html data-color-mode="light|dark">`
- 启动前写入 color mode 属性
- 暗色 token 变量
- 4 个主题 × 2 个模式的对比度自检
- 无闪烁切换
- overlay 层级清晰
- 图表与代码块的 dark 适配

---

## 10. 验收

未来若要把 dark mode 重新立项，验收应包括：

- 4 主题 × 2 模式的主要页面截图
- 刷新后保留用户偏好
- 没有显式偏好时，系统偏好可生效
- JS 关闭时仍有 CSS fallback

---

## 11. 关联

- `THEME_TOKENS_REFERENCE.md`
- `ACCESSIBILITY.md`
- `FRONTEND_UI_SPEC.md`
- `FRONTEND.md`
