---
title: system/auth 合同文档
doc_type: Contract
layer: system/auth
status: Active
related_designs:
  - docs/designs/AUTH_MODULE_DESIGN.md
  - docs/designs/SECURITY_CENTER_DESIGN.md
  - docs/designs/ERROR_CODE_AND_I18N.md
  - docs/designs/SECURITY_POLICY_ROADMAP.md
  - docs/designs/SSO_OIDC_DESIGN.md
related_acceptances:
  - docs/acceptances/ACCEPTANCE_CHECKLIST.md
updated_at: 2026-11-11
---

# system/auth 合同文档

English version: [SYSTEM_AUTH_CONTRACT.en.md](./SYSTEM_AUTH_CONTRACT.en.md)

关联设计：

- `AUTH_MODULE_DESIGN.md`
- `SECURITY_CENTER_DESIGN.md`
- `ERROR_CODE_AND_I18N.md`
- `SECURITY_POLICY_ROADMAP.md`
- `SSO_OIDC_DESIGN.md`

关联验收：

- `ACCEPTANCE_CHECKLIST.md`

---

本文用于定义 Pantheon `system/auth` 能力域的执行契约。

### 业务私有元数据的 owner 生命周期读取端口（2026-11-11）

`businessauth.Reader` 保持原签名。已注册的 auth provider 可实现可选的
`OwnerLifecycleReader.ReadOwnerLifecycle(ctx, tenantID, actorID) (OwnerLifecycle, error)`；
`contracts.RegisteredBusinessOwnerLifecycleReader()` 在模块 Init 期间、全局 seal 前即可取得同一个 provider
的能力，缺失或 typed nil 返回错误；seal 不增加新要求，且仍禁止替换 provider。依赖孤儿清理的业务模块
须在启动时要求此能力，不能把缺少能力当作 owner 删除。该端口不挂载 HTTP API。

返回值只有私有的 `TenantExists` / `ActorExists` 两个布尔字段（均不参与 JSON 序列化）。读取只访问
权威 `tenants.id/status` 与 `system_user.id/deleted_at` 元数据，不读取凭证、名称、密码、会话、角色或
成员关系。用户未软删除即存在，包括 disabled 和全局用户；不要求属于目标租户。正数租户只要行存在且
`status != deleted` 即存在，包括 suspended / archived；租户 0 是隐含的平台 owner，但仍必须核对用户。
actor 0、nil/cancelled context、nil DB、schema/query 错误均返回固定安全错误及空投影，不能被解释成删除。
只有 nil error 时的 false 才能用于有界孤儿清理；该投影不提供请求鉴权或访问许可。

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

- [DESIGN.md](../../DESIGN.md)
- [AGENTS.md](../../AGENTS.md)
- [BACKEND.md](../designs/BACKEND.md)
- [FRONTEND.md](../designs/FRONTEND.md)
- [AUTH_MODULE_DESIGN.md](../designs/AUTH_MODULE_DESIGN.md)
- [SECURITY_CENTER_DESIGN.md](../designs/SECURITY_CENTER_DESIGN.md)
- [ERROR_CODE_AND_I18N.md](../designs/ERROR_CODE_AND_I18N.md)
- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)

## 7. 强约束

### 7.1 领域边界约束

- 认证、会话、token、安全策略属于 `system/auth`
- 用户档案、角色授权、菜单管理、权限点治理属于 `system/iam`
- `auth` 不承担后台用户管理 CRUD

### 7.2 主链路约束

- 登录、refresh、logout、session、password 形成同一条认证主链路
- 安全策略只允许影响认证主链路，不允许无边界散落到其他系统域页面
- 当前账号会话和管理员全局登录日志都应从认证域语义理解，不回塞 `iam`
- `/login` 可以暴露语言选择，但该选择的运行时优先级归 `platform` 壳层解释
- `system/auth` 不得把登录页语言选择自动写成当前用户长期偏好
- `auth/me` 或登录返回体中的 `preferences.language` 只提供用户默认偏好来源，不拥有当前会话最终语言解释权

### 7.3 MFA 与预留能力约束

- `login.mfa_enabled` 已进入真实 TOTP 二次验证实现，关闭时不得影响原账号密码登录链路
- 真实 MFA 当前仅指本地账号密码登录后的 TOTP 二次验证，不等于外部身份源 SSO
- 首次启用 MFA 时，未绑定因子的用户必须允许走“返回二维码/手动密钥并现场绑定”的链路，避免管理员被直接锁死
- 登录页必须提供二维码、手动密钥、`otpauth://` URI 三种接入信息中的至少前两种，二维码优先、手动密钥兜底
- 后端不得回显“当前动态码”本身，只允许回传绑定所需 secret/URI，由认证器本地生成 6 位动态码
- `login.captcha_enabled / login.sso_enabled` 仍可以作为配置开关存在
- 未进入真实协议实现前，不展示验证码或 SSO 的伪能力入口
- 未明确身份源前，不提前实现真实 `SSO / OIDC / OAuth2 / 企业微信` 接入
- 2026-10-31 处置（AUD-005，方案 a 第一步）：`backend/modules/auth/oidc/`（4 文件 / 587 行）是在设计与合同均未批准前提下写出的不可达实现——包外 0 引用、0 路由注册、环境变量未登记、无测试，与本条直接冲突，已整体删除。未来重新实现必须以 `docs/designs/SSO_OIDC_DESIGN.md`（保持 Draft）为设计输入并重新走批准
- 保留项：迁移 `000029_oidc_fields` 与 `system_user` 的 `auth_type` / `oidc_*` 四个可空列不删——历史迁移不可改写，列无消费者、保留成本为零，身份源确定后可直接复用 schema
- 延后事项（本条登记，非口头承诺）：先删代码观察一个 release，再按 §8 的 contract-change 纪律单独提列删除迁移

### 7.4 安全约束

- 认证失败节流、来源/IP 限制、会话空闲超时、最大活跃会话数必须由 `auth` 统一承载
- 登录日志、请求标识、会话吊销链路必须可追踪
- 认证域错误返回优先使用 key，而不是自然语言
- 浏览器运行时的 `/auth/login`、`/auth/mfa/verify`、`/auth/refresh` 以 HttpOnly Cookie 为主链路，成功响应体不得把 access / refresh token 作为前端合同暴露给 JavaScript
- 浏览器 refresh 主链路优先从 refresh cookie 续期；测试脚本若需要 Bearer token，必须显式作为测试侧派生能力处理，而不是复用产品运行时合同

### 7.5 会话生命周期边界

- 会话创建、刷新、吊销、删除、空闲超时、最大活跃会话数和历史清理属于 `system/auth/session`。
- `system/iam/user` 可以因为重置密码、禁用用户、删除用户触发会话生命周期变化，但不能直接拥有 `system_user_session` 的表结构、SQL 细节或删除/吊销实现。
- 跨域调用必须通过窄契约表达，例如“吊销某用户会话”或“清理某用户会话”，不能通过公共包暴露 auth 内部表操作。
- `pkg` 可承载稳定平台契约或过渡性的查询 scope，但不应承载 auth/session 的命令式领域动作。
- 如果某个公共 helper 开始表达“应该吊销谁、何时删除、保留几条会话”等业务决策，应迁回 `system/auth/session`。

### 7.6 文档约束

- `system/auth` 的设计、评估、整改、验收文档都必须回指本合同
- 任何“auth 相关改造”如果实质上影响的是 `iam` 用户管理，必须先说明边界

### 7.7 Legacy 双入口（/system/* 认证端点）弃用与下线约束

`/system/login`、`/system/refresh`、`/system/logout`、`/system/user/info` 是与 `/auth/*` 同 handler 的 legacy 双入口（AUD-014，2026-10-31 决策：保留并标记弃用，而非立即下线；base 与 ops 前端均只调用 `/auth/*`）。

| 端点         | Legacy 入口                    | 当前入口                    | 处置                            |
| ------------ | ------------------------------ | --------------------------- | ------------------------------- |
| 登录         | `POST /api/v1/system/login`    | `POST /api/v1/auth/login`   | 保留至 2026-12-31，响应带弃用头 |
| 刷新         | `POST /api/v1/system/refresh`  | `POST /api/v1/auth/refresh` | 保留至 2026-12-31，响应带弃用头 |
| 登出         | `POST /api/v1/system/logout`   | `POST /api/v1/auth/logout`  | 保留至 2026-12-31，响应带弃用头 |
| 当前用户信息 | `GET /api/v1/system/user/info` | `GET /api/v1/auth/me`       | 保留至 2026-12-31，响应带弃用头 |

- 命中上述四个路径的响应必须带 `Deprecation: true` 与 `Sunset: Sat, 31 Dec 2026 23:59:59 GMT`（由 `middleware.LegacyDeprecationMiddleware` 按 `legacySelfServicePaths` 匹配，见 `backend/internal/middleware/csrf_middleware.go` 与 `backend/modules/auth/module.go` 两处挂载）
- 下线日期登记为 **2026-12-31**。下线属对外承诺变更：执行前必须由维护者确认无外部集成方仍在调用这四条路径
- 下线时必须一并移除：`sys` 公开组的两条路由（组随之清空即删）、`systemProtected` 的 `/logout` 与 `/user/info` 两行、两处 `LegacyDeprecationMiddleware` 挂载，以及 `csrf_middleware.go` 中 `/system/login`、`/system/refresh` 两条精确豁免——不得留下无对应路由的豁免。执行单：`docs/runbooks/DECOMMISSION_LEGACY_AUTH_ROUTES_2026-12-31.md`

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

- 符合 [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)
- 符合 [DOCUMENT_GOVERNANCE_CONTRACT.md](../contracts/DOCUMENT_GOVERNANCE_CONTRACT.md)
- 符合 [DOCUMENT_METADATA_AND_STATUS.md](../contracts/DOCUMENT_METADATA_AND_STATUS.md)

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

- [AUTH_MODULE_DESIGN.md](../designs/AUTH_MODULE_DESIGN.md)
- [SECURITY_CENTER_DESIGN.md](../designs/SECURITY_CENTER_DESIGN.md)
- [ERROR_CODE_AND_I18N.md](../designs/ERROR_CODE_AND_I18N.md)

### 10.2 Acceptance

- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)

## 变更日志

| 日期       | 版本 | 变更内容 | 原因         |
| ---------- | ---- | -------- | ------------ |
| 2026-06-25 | v1.0 | 初始版本 | 评估审计添加 |

## Refresh rotation: atomic single use（2026-11-04）

HTTP refresh 在变更数据库会话或签发新 pair 前，通过 `authtoken.ConsumeRefresh` 原子消费旧 Redis 映射（GETDEL，部署基线 Redis 7）。跨实例使用同一旧 token 最多只有一个请求可以继续；重放或消费/存储错误不能返回成功 pair。公共 `ValidateRefresh` 仍保持只读，已有导出签名与租户 V1 范围不变。

消费后如数据库/签发失败或响应不确定，不恢复旧 token，客户端可能需要重新登录。多标签页并发与重试可能收到未授权响应，这是单次消费的安全取舍。反向索引由正常签发替换或吊销清理；消费不能删除较新的索引。吊销、改密、membership gate 与 Cookie 响应策略仍须回归。

刷新与租户切换对会话行采用条件更新：会话仍未撤销、未到期，且读取时的 refresh JTI/tenant 未改变才允许写入。过期读不能覆盖期间完成的撤销、轮换或租户切换；冲突按未授权失败处理。DB 与 Redis 跨存储仍非事务，后续失败不恢复旧刷新令牌。
