---
title: 空 / 加载 / 错误状态规范 (Empty / Loading / Error States)
doc_type: Design
layer: platform / system/*
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-05-11
---

# 空 / 加载 / 错误状态规范 (Empty / Loading / Error States)

English version: [EMPTY_LOADING_ERROR_STATES.en.md](./EMPTY_LOADING_ERROR_STATES.en.md)

本文规定每类页面在「**非正常路径**」下的视觉、文案、交互、ARIA 标识。配合 `FRONTEND_UI_SPEC.md` §9 状态设计、`FRONTEND_PAGE_TEMPLATES.md`、`ACCESSIBILITY.md` §5 使用。

UI Spec §9 给视觉原则；本文给**具体每页**的状态变体规格。

---

## 1. 状态分类

| 状态 | 触发条件 | 优先级 |
|---|---|---|
| `loading` | 数据首次加载或刷新中 | 第 1 帧 |
| `empty-initial` | 加载完成，资源天然为空（用户从未创建） | 引导创建 |
| `empty-filtered` | 加载完成，因筛选条件无结果 | 引导调整筛选 |
| `error-network` | 请求失败 / 网络问题 | 提供重试 |
| `error-server` | 后端 5xx 或业务异常 | 提供重试 + 上报 |
| `forbidden` | 当前用户无权访问 | 引导联系管理员 |
| `not-found` | 资源不存在或已删除 | 引导返回 |
| `submitting` | 表单/批量操作进行中 | 禁用提交按钮 + spinner |

---

## 2. 通用视觉规格

所有状态用同一个 `<PageState>` 组件包装：

```
+---------------------------+
|                           |
|        [icon 64px]        |
|                           |
|        Title (heading)    |
|                           |
|     Description (body)    |
|                           |
|     [Primary CTA]         |
|     [Secondary]           |
|                           |
+---------------------------+
```

约束：
- 容器最小高度 = 父容器可视高度的 60%
- icon 单色，使用 `fg-tertiary` 颜色
- title 使用 `type-heading`，body 使用 `type-body` `fg-secondary`
- 整体居中（horizontal + vertical），不靠左
- CTA 按钮符合权限规则——无权限则隐藏

---

## 3. 列表页 (ListPage) 状态变体

| 状态 | icon | 标题 (i18n key) | 描述 | Primary CTA | ARIA |
|---|---|---|---|---|---|
| `loading` | spinner | — | — | — | `aria-busy="true"` 包裹 table |
| `empty-initial` | inbox | `state.list.empty.title` | `state.list.empty.description` | "新建" (跳转到创建表单/打开 Modal) | `role="status"` 公告 |
| `empty-filtered` | filter-off | `state.list.filteredEmpty.title` | `state.list.filteredEmpty.description` | "清除筛选" | `role="status"` 公告 |
| `error-network` | wifi-off | `state.error.network.title` | `state.error.network.description` | "重试" | `role="alert"` |
| `error-server` | alert | `state.error.server.title` | `state.error.server.description` + 「请联系管理员」 | "重试" | `role="alert"` |
| `forbidden` | lock | `state.forbidden.title` | `state.forbidden.description` | "返回工作台" | `role="alert"` |

文案要求：
- 描述不超过 2 句，给具体的下一步动作
- 不要写「未知错误」「请稍后重试」这种空洞文案；要么说明原因，要么给操作

---

## 4. 详情页 (DetailPage) 状态变体

| 状态 | 处理 |
|---|---|
| `loading` | hero 区显示骨架屏（避免布局抖动）；section 区显示 spinner |
| `not-found` | 整页 `PageState`：icon `compass`，文案「资源不存在或已被删除」，CTA「返回列表」 |
| `error-server` | 整页 `PageState`：CTA「重试」+「返回列表」 |
| `forbidden` | 整页 `PageState`：CTA「返回工作台」 |
| 部分 section 加载失败 | 该 section 独立显示 inline error，**不**让整页失败 |

详情页关键约束：**任何一个 section 加载失败不能让整页 5xx**——降级为 section 级错误。

---

## 5. 表单页 (FormPage) 状态变体

| 状态 | 处理 |
|---|---|
| `loading` (编辑模式回填中) | 整个 form 区显示骨架；提交按钮 disabled |
| `submitting` | 提交按钮 disabled + spinner；其他字段保持可见但 readonly |
| `validation-error` | 第一个错误字段聚焦 + `aria-invalid`；错误消息在字段下方 |
| `error-server` (提交后) | 顶部出现 `Alert` banner，描述错误并提示如何处理；表单内容**保留**，不清空 |
| `forbidden` | 整页拦截，不显示空表单 |

约束：
- 提交失败后**绝不**清空用户已填字段
- 防抖：500ms 内的重复 submit 忽略
- 校验时机：失焦校验单字段；提交校验所有字段

---

## 6. Dashboard 状态变体

| 状态 | 处理 |
|---|---|
| 单卡 loading | 卡片内 skeleton；不阻塞其他卡片 |
| 单卡 error | 卡片内 inline error + 重试按钮；不影响其他卡片 |
| 单卡 empty | 卡片内 `PageState` mini 变体（icon 40px，无 CTA） |
| 全部 forbidden | 整页拦截，引导联系管理员 |

Dashboard 关键约束：**每个卡片状态独立**，不允许一个 widget 失败导致整页 fallback。

---

## 7. 树形 / 抽屉 / Tab 切换内状态

| 容器 | 加载策略 |
|---|---|
| Tree 节点展开 | 节点旁 spinner（不要全屏 mask） |
| Tab 切换 | 切换瞬间显示 skeleton 占位；不闪屏 |
| Drawer 打开 | Drawer 内 skeleton；不卡住 Drawer 出现动画 |
| Modal 内嵌表单 | Modal body 内 loading；Modal 不闪 |

---

## 8. 文案库（标准 i18n key）

所有状态文案统一走 i18n key，禁止硬编码自然语言。

```
state.list.empty.title              // 还没有任何 {{resource}}
state.list.empty.description        // 你可以新建第一个 {{resource}} 来开始
state.list.filteredEmpty.title      // 没有匹配的 {{resource}}
state.list.filteredEmpty.description // 试着调整或清除筛选条件
state.error.network.title           // 无法连接到服务器
state.error.network.description     // 检查网络后重试
state.error.server.title            // 服务暂时不可用
state.error.server.description      // 系统在处理这次请求时出错，请稍后重试
state.forbidden.title               // 暂无权限访问此页面
state.forbidden.description         // 联系管理员申请相应权限后再试
state.notFound.title                // 资源不存在
state.notFound.description          // 该资源可能已被删除，或链接错误
state.submitting                    // 处理中…
state.loading                       // 加载中…
state.retry                         // 重试
state.clearFilters                  // 清除筛选
state.goBack                        // 返回
state.goWorkbench                   // 返回工作台
state.contactAdmin                  // 联系管理员
```

模块自己的 key 可以**继承覆盖**这些通用 key，但不能省略对应状态。

---

## 9. ARIA 与无障碍

详见 `ACCESSIBILITY.md` §5。本节是状态特定补充：

| 状态 | ARIA |
|---|---|
| `loading` | 容器 `aria-busy="true"` |
| `empty-*` | `role="status"` + 文案被屏幕阅读器读出 |
| `error-*` | `role="alert"` |
| `forbidden` / `not-found` | `role="alert"` |
| `submitting` | 按钮 `aria-disabled="true"` + 旁边 `<span role="status">` 公告进度 |

---

## 10. 验收

- 每个新页面必须实现 4-7 种状态（按页面类型）
- 状态切换不闪屏、不重新布局
- 文案走 i18n key
- 错误状态提供具体下一步（重试 / 联系管理员 / 返回）
- 单元素失败不影响整页

---

## 11. 关联

- `FRONTEND_UI_SPEC.md` §9 状态设计原则
- `FRONTEND_PAGE_TEMPLATES.md` §2.3 PageState
- `FRONTEND_COMPONENT_PLAN.md` §4.5 反馈与状态类
- `ACCESSIBILITY.md` §5 动态内容与异步状态
- `THEME_TOKENS_REFERENCE.md` §状态色 token
