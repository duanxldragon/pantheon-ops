# system/auth 合同文档

更新时间：2026-04-30

类型：Contract
归属层：system/auth
状态：Active

关联设计：
- `AUTH_MODULE_DESIGN.md`
- `SECURITY_CENTER_DESIGN.md`
- `ERROR_CODE_AND_I18N.md`
- `SECURITY_POLICY_ROADMAP.md`
- `SSO_OIDC_DESIGN.md`

关联评估：
- `SYSTEM_MODULE_AUDIT.md`

关联整改：
- `PLATFORM_AUTH_REMEDIATION_CLOSEOUT_20260429.md`

关联验收：
- `ACCEPTANCE_CHECKLIST.md`
- `QA_SMOKE_REPORT_20260420.md`

---

本文用于定义 Pantheon `system/auth` 能力域的执行契约。

它锁定的是认证、会话、安全中心、登录日志和认证策略的职责边界，避免后续再次把 `auth`、`iam`、`org`、`config` 混成一个“大 system 安全杂物间”。

---

## 1. 背景

Pantheon 早期最明显的结构性问题之一，是认证与后台用户管理长期混写。

如果没有 `system/auth` 合同，后续最容易继续发生：

- 登录、refresh、logout、session、password、login log 重新塞回 `user` 模块
- 安全中心和用户管理页混成一个能力域
- `MFA / CAPTCHA / SSO` 一类预留能力没有稳定落点
- 认证主链路和后台管理 CRUD 互相拖累

所以这份合同的作用，是明确 `auth` 的唯一职责：

> 它负责“你是谁、你能否登录、会话是否还有效、安全策略如何生效”，不负责系统管理 CRUD。

## 2. 归属层

本合同归属 `system/auth`。

它覆盖：

- 登录
- refresh token 轮换
- logout
- 当前主体身份
- 当前账号会话管理
- 登录日志
- 安全中心
- 认证域安全策略

它不等于：

- `system/iam` 用户、角色、菜单、权限管理
- `system/org` 组织归属治理
- `platform` 应用壳层和工作台聚合

## 3. 目标

`system/auth` 合同的目标是锁定以下 5 件事：

1. 明确 `auth` 与 `iam` 的职责边界
2. 明确认证主链路只围绕“身份验证与会话有效性”展开
3. 明确安全中心、会话管理和登录日志属于 `auth`
4. 明确预留型安全能力与真实协议实现的区分
5. 明确认证域的完成定义和验收口径

## 4. 非目标

本合同明确不负责：

- 用户列表 CRUD
- 角色 CRUD
- 菜单与权限策略 CRUD
- 部门与岗位治理
- 业务域账号模型
- 在未确定身份源前实现真实 `CAPTCHA / SSO` 协议流

换句话说：

- `auth` 可以预留 `CAPTCHA / SSO` 这类能力；
- 但不能为了“预留未来”破坏当前本地登录主链路。

## 5. 边界

### 5.1 覆盖对象

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/mfa/verify`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/me`
- `GET /api/v1/auth/sessions`
- `DELETE /api/v1/auth/sessions/:id`
- `PUT /api/v1/auth/password`
- `GET /api/v1/auth/security`
- `/login`
- `/auth/security`
- `/system/session`
- `/system/login-log`

### 5.2 不覆盖对象

- `/system/user`
- `/system/role`
- `/system/menu`
- `/system/permission`
- `/system/dept`
- `/system/post`
- `platform` 壳层导航与工作台

## 6. 依赖

`system/auth` 合同依赖以下文档与约束：

- [DESIGN.md](D:/workspace/go/pantheon-ops/DESIGN.md)
- [AGENTS.md](D:/workspace/go/pantheon-ops/AGENTS.md)
- [BACKEND.md](D:/workspace/go/pantheon-ops/docs/designs/BACKEND.md)
- [FRONTEND.md](D:/workspace/go/pantheon-ops/docs/designs/FRONTEND.md)
- [AUTH_MODULE_DESIGN.md](D:/workspace/go/pantheon-ops/docs/designs/AUTH_MODULE_DESIGN.md)
- [SECURITY_CENTER_DESIGN.md](D:/workspace/go/pantheon-ops/docs/designs/SECURITY_CENTER_DESIGN.md)
- [ERROR_CODE_AND_I18N.md](D:/workspace/go/pantheon-ops/docs/designs/ERROR_CODE_AND_I18N.md)
- [ACCEPTANCE_CHECKLIST.md](D:/workspace/go/pantheon-ops/docs/acceptances/ACCEPTANCE_CHECKLIST.md)

## 7. 强约束

### 7.1 领域边界约束

- 认证、会话、token、安全策略属于 `system/auth`
- 用户档案、角色授权、菜单管理、权限点治理属于 `system/iam`
- `auth` 不承担后台用户管理 CRUD

### 7.2 主链路约束

- 登录、refresh、logout、session、password 形成同一条认证主链路
- 安全策略只允许影响认证主链路，不允许无边界散落到其他系统域页面
- 当前账号会话和管理员全局登录日志都应从认证域语义理解，不回塞 `iam`

### 7.3 MFA 与预留能力约束

- `login.mfa_enabled` 已进入真实 TOTP 二次验证实现，关闭时不得影响原账号密码登录链路
- 真实 MFA 当前仅指本地账号密码登录后的 TOTP 二次验证，不等于外部身份源 SSO
- 首次启用 MFA 时，未绑定因子的用户必须允许走“返回二维码/手动密钥并现场绑定”的链路，避免管理员被直接锁死
- 登录页必须提供二维码、手动密钥、`otpauth://` URI 三种接入信息中的至少前两种，二维码优先、手动密钥兜底
- 后端不得回显“当前动态码”本身，只允许回传绑定所需 secret/URI，由认证器本地生成 6 位动态码
- `login.captcha_enabled / login.sso_enabled` 仍可以作为配置开关存在
- 未进入真实协议实现前，不展示验证码或 SSO 的伪能力入口
- 未明确身份源前，不提前实现真实 `SSO / OIDC / OAuth2 / 企业微信` 接入

### 7.4 安全约束

- 认证失败节流、来源/IP 限制、会话空闲超时、最大活跃会话数必须由 `auth` 统一承载
- 登录日志、请求标识、会话吊销链路必须可追踪
- 认证域错误返回优先使用 key，而不是自然语言

### 7.5 文档约束

- `system/auth` 的设计、评估、整改、验收文档都必须回指本合同
- 任何“auth 相关改造”如果实质上影响的是 `iam` 用户管理，必须先说明边界

## 8. 完成定义

`system/auth` 达到“当前已完成”至少应满足：

### 8.1 职责完成

- `auth` 已从 `user` 中独立建模
- 登录、会话、安全中心、登录日志归属清晰

### 8.2 主链路完成

- 登录、refresh、logout、会话读取、会话吊销链路稳定
- 当前用户修改密码链路稳定
- 登录日志和安全中心链路稳定

### 8.3 策略完成

- 密码最小长度、密码复杂度、登录失败锁定、来源/IP 节流、空闲超时、最大会话数有统一策略入口
- 预留型能力明确为“仅预留”还是“真实实现”

### 8.4 文档与验收完成

- `auth` 的主设计、验收和整改文档都能回链本合同
- 验收清单能够区分“设计预留”和“真实协议实现”

## 9. 验收标准

`system/auth` 相关改动至少应通过以下验收：

### 9.1 文档验收

- 符合 [ACCEPTANCE_CHECKLIST.md](D:/workspace/go/pantheon-ops/docs/acceptances/ACCEPTANCE_CHECKLIST.md)
- 符合 [DOCUMENT_GOVERNANCE_CONTRACT.md](D:/workspace/go/pantheon-ops/docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md)
- 符合 [DOCUMENT_METADATA_AND_STATUS.md](D:/workspace/go/pantheon-ops/docs/contracts/DOCUMENT_METADATA_AND_STATUS.md)

### 9.2 后端与接口验收

- `go test ./backend/modules/auth`
- 若影响追踪链路或中间件，补 `go test ./backend/internal/middleware`

### 9.3 前端与构建验收

- `cd frontend && npm run build`
- 如果影响认证主链路或安全页主链路，需补冒烟或验收记录

### 9.4 页面与主链路验收

- `/login`
- `/auth/security`
- `/system/session`
- `/system/login-log`

补充：

- 当 `login.mfa_enabled=false` 时，账号密码登录链路必须保持可用
- 当 `login.mfa_enabled=true` 且用户未绑定因子时，`/login` 必须进入现场绑定流程，而不是直接失败
- 用户名或密码错误时，登录页必须展示认证域错误 key 对应翻译，例如“用户名或密码错误”，不能统一退化成模糊失败提示
- 二次验证码错误、challenge 过期、来源/IP 受限时，登录页必须优先展示后端错误 key 对应翻译

### 9.5 预留能力验收

- 如果文档声明 `MFA / CAPTCHA / SSO` 仅为预留，必须明确“不进入真实协议实现”
- 如果进入真实实现，必须新增独立设计和验收链路，不得沿用“预留”口径

## 10. 关联文档

### 10.1 Design

- [AUTH_MODULE_DESIGN.md](D:/workspace/go/pantheon-ops/docs/designs/AUTH_MODULE_DESIGN.md)
- [SECURITY_CENTER_DESIGN.md](D:/workspace/go/pantheon-ops/docs/designs/SECURITY_CENTER_DESIGN.md)
- [ERROR_CODE_AND_I18N.md](D:/workspace/go/pantheon-ops/docs/designs/ERROR_CODE_AND_I18N.md)

### 10.2 Assessment

- [SYSTEM_MODULE_AUDIT.md](D:/workspace/go/pantheon-ops/docs/assessments/SYSTEM_MODULE_AUDIT.md)

### 10.3 Remediation

- [PLATFORM_AUTH_REMEDIATION_CLOSEOUT_20260429.md](D:/workspace/go/pantheon-ops/docs/archive/PLATFORM_AUTH_REMEDIATION_CLOSEOUT_20260429.md)

### 10.4 Acceptance

- [ACCEPTANCE_CHECKLIST.md](D:/workspace/go/pantheon-ops/docs/acceptances/ACCEPTANCE_CHECKLIST.md)
- [QA_SMOKE_REPORT_20260420.md](D:/workspace/go/pantheon-ops/docs/archive/QA_SMOKE_REPORT_20260420.md)
