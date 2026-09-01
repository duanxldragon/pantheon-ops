---
title: 导航信息架构深化设计
doc_type: Design
layer: platform / system/iam
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-05-11
---

# 导航信息架构深化设计

English version: [NAVIGATION_IA_STRATEGY.en.md](./NAVIGATION_IA_STRATEGY.en.md)

本文定义菜单 IA、外链、iframe、页签缓存、面包屑和高亮策略的深化规范。配合 `FRONTEND_UI_SPEC.md` §5、`PERMISSION_MODEL.md`、`MOBILE_RESPONSIVE_BREAKPOINTS.md` 使用。

---

## 1. 边界

- 导航视觉、壳层模式、页签和面包屑属于 `platform`。
- 菜单元数据、页面权限、按钮权限和组件键属于 `system/iam`。
- 业务模块只声明自己的导航入口，不直接控制平台壳层行为。
- 顶层菜单默认优先呈现业务域入口；平台工作台和系统治理类菜单排在业务入口之后。

## 2. 菜单类型

| 类型 | 说明 | 约束 |
| :--- | :--- | :--- |
| `M` | 目录 | 不直接承载页面业务逻辑 |
| `C` | 页面 | 必须有 path、routeName、titleKey、component key、pagePerm |
| `F` | 按钮/动作权限 | 不参与侧边栏导航 |
| external | 外链 | 必须明确是否新窗口打开 |
| iframe | iframe 页面 | 必须进入白名单和安全说明 |

## 3. 高亮与面包屑

- 详情页、创建页、编辑页通常不作为侧边栏菜单。
- 这类页面必须通过 `activeMenu` 指向所属列表页。
- 面包屑应来自路由元数据和菜单树，不应由页面硬编码自然语言。

## 4. 页签与缓存

后续若启用页签缓存，应按页面类型分级：

- 列表页可缓存筛选和分页。
- 表单页默认不缓存提交中状态。
- 高敏页面不缓存敏感输入。
- iframe 页面必须单独评估缓存和销毁策略。

## 5. 外链与 iframe 安全

- 外链必须声明 `isExternal`。
- iframe 必须声明白名单域名、sandbox 策略和权限边界。
- 动态菜单不能允许管理员写入任意 iframe URL 后绕过安全策略。

---

## 6. 菜单状态机

每个菜单项在运行时有 5 种可能状态：

```
[隐藏] ──无权限/被禁用──┐
                         ↓
[可见] ──hover──> [hover] ──click──> [激活]
                         ↑
[加载中] ──异步元数据加载完成────┘
```

| 状态 | 触发 | 视觉 | ARIA |
|---|---|---|---|
| `hidden` | 无 `pagePerm` 或菜单被管理员禁用 | 不渲染（也不占位） | — |
| `visible` | 有权限，未选中 | 默认色 | `role="menuitem"` |
| `hover` | 鼠标悬停或键盘焦点 | `bg-hover` + 微缩进 | `aria-current="false"` |
| `active` | 当前路由匹配 `activeMenu` | accent 色块 + 粗体 | `aria-current="page"` |
| `loading` | 异步菜单元数据加载中 | skeleton 条 | `aria-busy="true"` |

**禁止**：「可见但 disabled」——无权限直接 hide，不要灰显占位。

---

## 7. 页签缓存详细算法

每个 Tab 维护一份**轻量元数据**：

```ts
interface TabState {
  routeName: string
  titleKey: string
  activeMenuId: number
  pageType: 'list' | 'detail' | 'form' | 'config' | 'dashboard' | 'iframe'
  cachePolicy: 'memo' | 'fresh' | 'sensitive'  // 缓存策略
  scrollY: number                                // 离开时记忆滚动位置
  filters?: Record<string, unknown>              // 列表筛选
  pagination?: { page: number; size: number }
}
```

### 7.1 缓存策略

| `cachePolicy` | 切回 Tab 时 | 适用页面 |
|---|---|---|
| `memo` | 完全保留组件状态，**不**重新拉数据 | 列表页、Dashboard |
| `fresh` | 销毁后重建组件，重新拉数据 | 表单页（除草稿场景）、详情页 |
| `sensitive` | 销毁，**清空** sessionStorage 关联键，不留状态 | 安全中心、API key 页 |

默认：list/dashboard → `memo`；其他 → `fresh`；含密码/密钥/审计敏感的 → `sensitive`。

### 7.2 Tab 数量上限

- 默认 12 个，超出按 LRU 淘汰
- 用户可在偏好里调到 4-30
- 被关闭的 Tab 在 5 分钟内可通过 "重新打开关闭的页签" 恢复

### 7.3 强制刷新

- 用户在 Tab 上右键 "刷新"：强制 `fresh` 一次，不改变 `cachePolicy`
- 路由参数变化（如 `/cmdb/host/123` → `/cmdb/host/456`）：判定为同一 Tab 模板下的不同实例，按 detail 页规则 `fresh`

---

## 8. 面包屑生成算法

面包屑**不允许**由页面手写。统一由 `<Breadcrumbs>` 组件根据路由元数据自动生成：

```
matched route chain (router.matched)
  + each route's meta.titleKey
  + each route's meta.activeMenu fallback
  → 输出 Breadcrumb[]
```

### 8.1 三种节点类型

| 节点 | 来源 | 可点击 |
|---|---|---|
| 目录节点（M） | 菜单树中的目录 | 否（无 path） |
| 列表节点（C） | 当前路由的 `activeMenu` 列表页 | 是 |
| 当前页节点 | 当前路由本身 | 否（末节点） |

### 8.2 动态参数

详情页面包屑可以包含动态名：

```
配置 / CMDB / 主机 / web-01.prod.lan
```

但**只允许**末节点是动态名；中间节点必须是静态菜单标题。

实现：`<Breadcrumbs lastNode={resourceName}>`。

### 8.3 多层目录的折叠

超过 4 层时，中间合并为 `...`：

```
配置 / CMDB / ... / 主机 / web-01
```

`...` 鼠标悬停展开完整路径。

---

## 9. iframe 白名单与安全约束

### 9.1 白名单格式

存储为 `system_iframe_whitelist` 表：

```sql
CREATE TABLE system_iframe_whitelist (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  domain VARCHAR(255) NOT NULL UNIQUE,
  sandbox VARCHAR(255) DEFAULT 'allow-same-origin allow-scripts',
  allow_postmessage BOOLEAN DEFAULT FALSE,
  description VARCHAR(255),
  created_at TIMESTAMP,
  ...
);
```

### 9.2 创建菜单时的校验

管理员保存 iframe 类型菜单时，后端必须：

1. 解析 `path` 的协议和域名
2. 协议**必须** `https`
3. 域名必须**精确匹配**白名单（不允许 `*.example.com` 通配，避免子域名劫持）
4. `sandbox` 属性继承白名单条目

### 9.3 运行时强制

前端渲染 iframe 时：

```html
<iframe
  src="..."
  sandbox="allow-same-origin allow-scripts"
  referrerpolicy="strict-origin"
  loading="lazy"
></iframe>
```

- `sandbox` 不可由动态菜单覆盖
- `referrerpolicy="strict-origin"` 避免泄露完整 URL
- 不允许 `allow-top-navigation`，防止 iframe 重定向父窗

### 9.4 postMessage 协议

如果某 iframe 域名 `allow_postmessage=true`，约定一套消息协议：

```ts
type IframeMessage =
  | { type: 'navigate'; route: string }     // iframe 内部请求父窗导航
  | { type: 'resize'; height: number }      // iframe 自报高度
  | { type: 'auth-failed' }                 // iframe 检测到会话失效
```

父窗接收到 `navigate` 消息时，必须**验证**目标路由属于已知菜单，再决定导航。

---

## 10. 外链 UX

| 触发 | 行为 |
|---|---|
| 单击 | 默认新窗口打开（含 `rel="noopener noreferrer"`） |
| 中键/Cmd+Click | 浏览器默认行为 |
| 标题前缀 | 在 i18n key 之外，菜单视觉上**追加** `↗` 图标 |

外链不进入 Tab 历史；不参与面包屑；不参与缓存策略。

---

## 11. 全局搜索导航 (Cmd+K)

- 全局快捷键 `/` 或 `Cmd+K` 唤起命令面板
- 搜索源：当前用户**有权限**的所有菜单 + 最近 20 个访问路径
- 模糊匹配标题 + i18n 别名 + 路径关键词
- 选中后切换到目标路由（按 `cachePolicy` 处理 Tab）

权限规则：搜索结果**绝不**显示无权限菜单——不要泄露存在性。

---

## 12. 多租户与角色的导航差异

- 不同租户/角色看到的菜单**结构相同**，但**节点可见性不同**
- 隐藏节点导致的"空目录"必须**也隐藏**——不能出现可点击但展开为空的目录
- 切换租户时，整套菜单元数据重新拉取（`GET /api/v1/system/menu/me`）

---

## 13. 路由恢复

刷新页面后必须恢复：

- 当前路由及参数
- 打开的 Tab 列表（仅元数据，不恢复内部状态）
- Sidebar 折叠状态、活动主题
- 全局密度模式

不恢复：

- 表单未提交内容（除非显式实现"草稿"）
- 弹窗、Drawer 打开态
- 滚动位置（除非 Tab `cachePolicy=memo`）

---

## 14. 移动端断点下的导航

详见 `MOBILE_RESPONSIVE_BREAKPOINTS.md` §3。本节关键约束：

- `md` 及以下断点 Sidebar 默认收起为抽屉式
- Tab 在 `sm/xs` 隐藏，由顶部 `<` 按钮和命令面板替代
- 抽屉打开时必须有 mask + Esc/外击关闭
- 抽屉打开时主区 inert (`aria-hidden="true"`)

---

## 15. 仪表化与可观测

- 每次导航发送一次客户端事件：`{ from, to, durationMs, method }` (method = click / shortcut / cmd-k / breadcrumb / back)
- 用于分析路径分布、菜单效率、命令面板使用率
- 敏感页面（auth、security-center、user-detail）的事件**只**记录到达，不记录搜索关键词

---

## 16. 验收

- 横版顶栏和竖版侧栏的选中态一致。
- 目录、页面、外链、iframe、动作权限语义不混写。
- 详情页能正确高亮所属列表页。
- 面包屑和页签标题使用 i18n key。
- 未注册组件键不能被保存进菜单。
- 菜单状态机 5 种状态在 Chromatic（或截图归档）中各覆盖一次。
- iframe 类型菜单：保存非白名单域名时后端拒绝。
- Tab `cachePolicy` 三种各有一个真实页面覆盖。
- 面包屑超过 4 层时正确折叠。
- 刷新后路由和 Tab 列表恢复。
- 移动端断点抽屉打开/关闭无障碍焦点处理正确。
- Cmd+K 命令面板不暴露无权限菜单。

---

## 17. 关联

- `FRONTEND_UI_SPEC.md` §5 导航与信息架构规范
- `PERMISSION_MODEL.md` 权限与菜单可见性
- `MOBILE_RESPONSIVE_BREAKPOINTS.md` §3 壳层结构断点
- `ACCESSIBILITY.md` §2 键盘导航
- `EMPTY_LOADING_ERROR_STATES.md` §4 详情页 not-found 处理
