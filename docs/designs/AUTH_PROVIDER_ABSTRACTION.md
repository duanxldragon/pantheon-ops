---
title: 认证 Provider 抽象设计
doc_type: Design
layer: system/auth
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-05-18
---

# 认证 Provider 抽象设计

English version: [AUTH_PROVIDER_ABSTRACTION.en.md](./AUTH_PROVIDER_ABSTRACTION.en.md)

关联合同：
- `SYSTEM_AUTH_CONTRACT.md`
- `SYSTEM_IAM_CONTRACT.md`

关联路线：
- `P2_SCALE_ROADMAP.md`
- `PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md`

---

本文定义 `system/auth` 在进入真实 SSO / OIDC / LDAP / 企业身份源接入前，必须先建立的 provider 抽象边界。

目标不是马上支持某个真实身份源，而是先把 `pantheon-base` 做到：

- `sso-ready`
- `captcha-ready`
- `risk-ready`

并且避免在身份源未知时把登录能力做成伪能力。

## 1. 设计目标

本设计要解决四件事：

1. 明确“本地登录”与“外部身份登录”的职责边界
2. 为未来 provider 接入预留统一抽象，而不是每接一个身份源就新增一套分支逻辑
3. 定义外部身份绑定模型、审计语义和回退策略
4. 保证在 provider 未真正接入前，前后端不会暴露误导性控制项

## 2. 非目标

当前阶段不做：

- 真实 OIDC / OAuth2 / LDAP / 企业微信 / 钉钉 provider 接入
- 联邦登出实现
- 自动账号主数据同步
- 复杂的身份合并策略
- 绕过本地登录兜底

## 3. 抽象边界

### 3.1 `system/auth` 负责

- 登录入口编排
- 本地登录兜底
- provider 状态暴露
- 外部身份绑定关系读取
- 登录审计与风险事件留痕

### 3.2 `system/auth` 不负责

- 定义外部身份源的组织模型
- 承担 `system/org` 的组织治理
- 重写 `system/iam` 的用户、角色、权限语义
- 在 provider 未接入前伪造“可点击但不可用”的单点登录流程

## 4. Provider 抽象模型

### 4.1 Provider 类型

建议最小枚举：

- `local`
- `oidc`
- `oauth2`
- `ldap`
- `saml`
- `custom`

当前真实可用类型只有：

- `local`

其他类型在未完成接入前，只能以 `provider-ready` 元数据存在。

### 4.2 Provider 状态

建议最小状态：

- `disabled`
- `configured`
- `ready`
- `error`

解释：

- `disabled`：未启用或未配置
- `configured`：已录入配置，但未完成运行时联通或未开放登录入口
- `ready`：可以对外提供真实登录
- `error`：配置或运行状态异常

没有真实 provider 代码时，不允许把状态声明成 `ready`。

## 5. 外部身份绑定模型

建议新增或预留统一绑定关系：

- `provider_type`
- `provider_key`
- `external_subject`
- `external_account`
- `local_user_id`
- `binding_status`
- `last_login_at`
- `last_sync_at`
- `created_at`
- `updated_at`

### 5.1 绑定语义

- 一个本地用户可以绑定多个外部身份
- 一个外部身份在同一 provider 内只能绑定一个本地用户
- 绑定关系是认证映射，不等于权限来源

### 5.2 权限语义

- 权限、角色、菜单、数据范围仍由本地 `system/iam` 决定
- 外部身份源可以参与“你是谁”的识别，但不直接改写 Pantheon 内部授权模型

## 6. 登录编排

### 6.1 当前默认路径

当前正式登录路径必须保持：

- 本地账号 + 本地密码

### 6.2 未来外部登录路径

当 provider 进入真实接入阶段后，登录编排建议为：

1. 用户选择 provider
2. 跳转外部身份源
3. 回调进入 `system/auth`
4. 根据 `external_subject` 查绑定关系
5. 找到本地用户后签发 Pantheon 会话
6. 记录审计与风险事件

### 6.3 本地兜底

无论未来接入什么 provider，都必须保留：

- 管理员本地登录兜底
- provider 异常时的本地救援入口

禁止把所有登录入口都切成“只剩外部登录”而没有可控回退。

## 7. 配置模型

安全中心或系统设置只允许暴露真实状态，不允许暴露伪能力。

建议暴露：

- provider 类型
- 是否已配置
- 是否允许显示入口
- 当前状态
- 错误摘要
- 是否启用本地兜底

不允许在未接入时暴露：

- 可点击的 SSO 按钮但点击后无真实链路
- 可保存但没有任何后端语义的 CAPTCHA 开关

## 8. 审计与风险事件

所有 provider 相关动作至少要记录：

- provider 类型
- provider key
- 外部主体标识摘要
- 本地用户 ID
- 登录结果
- 来源 IP / UA / 时间

建议风险事件类型最少预留：

- `provider-login-success`
- `provider-login-failed`
- `provider-binding-missing`
- `provider-callback-invalid`
- `provider-disabled-attempt`

## 9. 前端表现约束

在 provider 未真实接入前：

- 登录页不显示伪 SSO 主入口
- 安全中心只显示 readiness 信息和说明文字
- 如展示 future capability，必须明确写成“未启用 / 未接入 / 仅预留”

## 10. 完成定义

`auth-scale` 的第一阶段完成，不以真实 provider 登录为标准，而以以下条件为标准：

- 有 provider 抽象文档
- 有绑定模型定义
- 有本地登录兜底语义
- 有审计与风险事件预留
- 前后端不会暴露伪能力

## 11. 后续实现前置条件

在进入真实 provider 实现前，必须先补清：

- 目标身份源类型
- 回调域名策略
- 账号自动创建或手工绑定策略
- 管理员兜底策略
- 登出与会话失效语义
- 故障切换与回滚策略
