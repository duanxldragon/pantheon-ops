# 安全中心设计

更新时间：2026-04-17

类型：Design
归属层：system/auth
状态：Active

本文定义 Pantheon Base 的安全中心设计，用于承接 `auth` 模块拆分后的安全能力。

安全中心不是“个人中心里放一个改密码表单”，而是用户和管理员理解账号安全、会话状态、登录风险的入口。

## 1. 设计目标

- 把认证与安全能力从 `user/profile` 中独立出来
- 支持当前用户管理自己的会话和密码
- 支持管理员查看安全事件和登录日志
- 当前已接入 TOTP MFA 登录主链路、来源级节流、安全事件、历史密码复用限制和密码过期提醒，并为后续 SSO、更细风控预留空间
- 保持企业级后台的克制、清晰、可信

## 2. 能力边界

安全中心属于 `auth` 能力域。

它负责：

- 当前会话
- 在线设备
- 密码修改
- 登录日志
- 会话吊销
- 安全提示
- 登录风险事件（后续）

它不负责：

- 用户列表 CRUD
- 角色授权
- 菜单配置
- 业务数据权限

## 3. 信息架构

建议安全中心包含：

```text
安全中心
  ├── 安全概览
  ├── 在线会话
  ├── 登录日志
  ├── 密码安全
  └── 安全策略（后续）
```

## 4. 页面规划

## 4.1 当前用户安全中心

路径建议：

```text
/auth/security
```

页面模板：

- `ProfilePage`
- `ConfigPage`

功能：

- 查看账号安全状态
- 修改密码
- 查看当前在线会话
- 下线其他设备
- 查看最近登录记录

## 4.2 在线会话页

路径建议：

```text
/auth/sessions
```

页面模板：

- `ListPage`

功能：

- 当前会话标识
- 登录 IP
- User-Agent
- 最近刷新时间
- refresh token 过期时间
- 会话创建时间
- 是否已吊销
- 下线某个会话

## 4.3 登录日志页

路径建议：

```text
/system/login-log
```

说明：

- 当前用户可看自己的登录日志
- 管理员可看全局登录日志

页面模板：

- `ListPage`

功能：

- 用户名
- IP
- 浏览器
- 操作系统
- 状态
- 登录时间
- 失败原因

## 5. 前端页面结构

## 5.1 SecurityCenter

推荐结构：

```text
SecurityCenter
  ├── SecurityOverviewCard
  ├── Tabs
  │   ├── PasswordPanel
  │   ├── SessionsPanel
  │   └── RecentLoginPanel
  └── SecurityTips
```

## 5.2 SecurityOverviewCard

展示：

- 当前账号
- 最近登录时间
- 当前会话状态
- 密码最近更新时间（后续）
- MFA 策略状态

## 5.3 SessionsPanel

展示：

- 当前设备
- 其他设备
- 最后活动时间
- 下线按钮

当前设备禁止直接删除，只能通过 logout 退出当前会话。

## 5.4 PasswordPanel

展示：

- 当前密码
- 新密码
- 确认新密码
- 密码规则提示

规则：

- 修改密码后吊销其他会话
- 当前会话继续有效或要求重新登录，必须在产品上明确

建议：

- 当前阶段保持当前会话有效
- 自动吊销其他会话

## 6. API 设计

## 6.1 当前用户安全接口

建议：

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `GET` | `/api/v1/auth/security` | 获取当前账号安全概览 |
| `GET` | `/api/v1/auth/sessions` | 获取当前账号活跃会话列表，仅返回未吊销且 refresh token 未过期的会话 |
| `DELETE` | `/api/v1/auth/sessions/:id` | 下线指定会话 |
| `PUT` | `/api/v1/auth/password` | 修改当前账号密码 |
| `GET` | `/api/v1/auth/login-logs` | 获取当前账号登录日志 |
| `POST` | `/api/v1/auth/mfa/verify` | 校验 TOTP 二次验证 challenge，成功后签发登录会话 |

## 6.2 管理员安全审计接口

建议：

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `GET` | `/api/v1/system/login-log/list` | 登录日志列表 |
| `POST` | `/api/v1/system/login-log/export` | 登录日志导出 |
| `POST` | `/api/v1/system/login-log/cleanup` | 按保留期清理登录日志，当前只允许保留最近 `1 / 7 / 30` 天，并要求二次验证 |
| `POST` | `/api/v1/system/login-log/batch-delete` | 按选择集批量删除登录日志，并要求二次验证 |
| `GET` | `/api/v1/system/session/list` | 全局会话列表 |
| `DELETE` | `/api/v1/system/session/:id` | 管理员下线会话 |

补充约束：

- 登录日志不再默认开放“清空全部”。
- 管理员侧危险动作统一收敛为“保留最近 `1 / 7 / 30` 天清理”或“按选择集删除”。
- 选择集删除与保留期清理都必须复用敏感操作二次验证链路。

## 7. 数据模型

## 7.1 当前已有表

当前已有：

- `system_user_session`
- `system_log_login`
- `system_auth_factor`
- `system_auth_mfa_challenge`

## 7.2 `system_user_session`

当前字段可支持：

- sessionId
- userId
- refreshJTI
- refreshExpiresAt
- lastRefreshAt
- lastIP
- userAgent
- revokedAt
- createdAt
- updatedAt

建议补充或派生展示：

- 是否当前会话
- 设备名称（由 User-Agent 解析）
- 浏览器
- 操作系统
- 地理位置（后续）

## 7.3 `system_log_login`

当前字段可支持：

- username
- ipaddr
- browser
- os
- status
- msg
- loginTime

建议补充：

- userId
- requestId
- tenantId（后续）
- riskLevel（后续）

## 8. 权限设计

## 8.1 当前用户自助权限

这些接口只要求登录态，不要求角色菜单授权：

- 查看自己安全概览
- 查看自己会话
- 下线自己其他会话
- 修改自己密码
- 查看自己登录日志

## 8.2 管理员权限

建议权限点：

```text
auth:session:view
auth:session:delete
auth:login-log:view
auth:security:view
```

如果短期统一在 `system` scope 下：

```text
system:session:view
system:session:delete
system:login-log:view
```

## 9. 菜单设计

建议新增菜单：

```text
安全中心
  ├── 在线会话
  └── 登录日志
```

说明：

- 当前用户安全中心可以不放在侧边栏，通过顶部用户菜单进入。
- 管理员视角的登录日志和全局会话应放到“安全审计”一级菜单下。

## 10. i18n key 规划

建议 key：

```text
auth.security.title
auth.security.overview
auth.security.sessions
auth.security.password
auth.security.loginLogs
auth.session.current
auth.session.revoke
auth.session.revokeSuccess
auth.loginLog.status.success
auth.loginLog.status.failed
```

## 11. UI 设计要求

安全中心必须给人“可信”和“可控”的感觉。

要求：

- 不做花哨视觉
- 不使用大面积危险色
- 当前设备明确标识
- 危险操作二次确认
- 修改密码提示影响范围
- 会话下线后即时刷新列表

## 12. 状态设计

必须覆盖：

- 会话加载中
- 无其他会话
- 会话下线中
- 登录日志为空
- 修改密码中
- 修改密码失败

## 13. 审计规则

以下操作必须记录审计：

- 登录成功
- 登录失败
- 修改密码
- 下线会话
- 管理员下线他人会话

## 14. 分阶段实现

## 14.1 Phase 1：复用现有表

- 用 `system_user_session` 做在线会话
- 用 `system_log_login` 做登录日志
- 安全中心只做当前用户自助

## 14.2 Phase 2：管理员审计

- 登录日志管理页
- 全局会话管理页
- 管理员强制下线

## 14.3 Phase 3：安全策略

- 密码策略
- 登录失败限制
- MFA（当前已进入真实 TOTP 登录链路）
- 风险事件

## 15. 与个人中心的关系

个人中心继续负责：

- 昵称
- 头像
- 邮箱
- 手机号

安全中心负责：

- 密码
- 会话
- 登录日志
- 安全策略

不要再把安全能力塞进 `ProfileCenter` 里继续膨胀。

## 16. 当前落地差距

当前已有：

- 修改密码
- 修改密码后吊销其他会话
- 登录日志写入
- session 表
- 安全中心页面
- 安全概览接口 `/api/v1/auth/security`
- 当前用户在线会话列表 API
- 当前用户下线其他会话 API
- 当前用户登录日志 API
- 管理员登录日志页
- 管理员会话页
- 顶部用户菜单和个人中心安全入口

当前缺少：

- 更完整的 User-Agent/设备品牌解析与地理位置识别
- 安全中心菜单与 i18n key

## 17. 验收清单

安全中心完成时必须满足：

- 用户能看到当前会话
- 用户能下线其他会话
- 用户能修改密码
- 用户能看到最近登录记录
- 修改密码后其他会话失效
- 管理员能查看登录日志
- 关键操作有审计
- 文案全部走 i18n

## 18. 下一份建议补的文档

下一份建议补：

- `docs/acceptances/ACCEPTANCE_CHECKLIST.md`

因为安全中心设计完成后，下一步应当把系统能力设计统一纳入阶段验收，而不是停留在单篇设计文档。
