# Platform + Auth 整改结案清单

更新时间：2026-04-29

类型：Acceptance
归属层：platform
状态：Archived

本文用于对本轮 `platform` 与 `system/auth` 整改做结案确认，避免把“已经完成的闭环”“明确延期的演进项”“本轮不做的能力”混在一起。

## 1. 本轮范围

本轮整改边界如下：

- `platform`
  - 请求链路可追踪性
  - 健康检查
  - 平台级验收与报告纠偏
- `system/auth`
  - 本地登录主链路安全加固
  - 会话与登录日志可追踪性
  - 安全中心策略可见性
  - `CAPTCHA / MFA / SSO` 设计预留

不在本轮范围：

- `system/iam` 数据权限实现
- 真实 `MFA / CAPTCHA / SSO` 协议接入
- 新身份源接入联调
- 业务域功能扩展

## 2. 已完成

### 2.1 `platform`

- 已新增健康检查接口 `GET /api/v1/health`
- 已补 `X-Request-ID` 与 `X-Trace-ID` 透传/生成能力
- 已将请求标识接入统一操作日志链路
- 已补健康检查与请求链路相关测试

### 2.2 `system/auth`

- 已完成认证域物理拆分后的主链路回归验证
- 已新增来源/IP 维度登录失败节流
- 已新增 `system_login_throttle` 作为来源/IP 节流状态存储
- 已将 `requestId` 写入登录日志
- 已保持本地账号密码登录、会话创建、JWT 签发链路稳定
- 已在安全中心返回并展示以下策略快照：
  - `login.source_max_failed_attempts`
  - `login.source_window_minutes`
  - `login.source_lock_minutes`
  - `login.captcha_enabled`
  - `login.mfa_enabled`
  - `login.sso_enabled`

### 2.3 文档与验收

- 已纠偏旧评估报告中的过时结论
- 已形成整改优先级清单
- 已在 `AUTH_MODULE_DESIGN.md` 补齐未来 SSO / 外部身份接入设计
- 已在 `ACCEPTANCE_CHECKLIST.md` 增补 `system/auth` 外部身份接入专项验收约束

## 3. 已验证

- 后端测试已通过：
  - `go test ./backend/modules/auth ./backend/modules/system/audit ./backend/modules/platform ./backend/internal/middleware`
- 前端 locale 审计已通过：
  - `npm run audit:i18n-locales`
- 运行中接口已完成 smoke：
  - `GET /api/v1/health`
  - `POST /api/v1/auth/login`
  - `GET /api/v1/auth/security`

## 4. 明确延期

以下事项不是遗漏，而是明确延期到后续阶段：

- `MFA` 真实认证流
- `CAPTCHA` 真实校验流
- `SSO / OIDC / OAuth2 / 企业微信 / 其他身份源` 真实接入
- 登录风险事件、地理位置、更多风控维度
- 忘记密码页
- 二次验证页

延期原则：

- 未确定身份源前，不提前实现真实 SSO 协议流
- 不在登录页展示伪能力
- 不为了“预留”而破坏当前本地登录主链路

## 5. 当前结论

本轮 `platform + system/auth` 整改已达到结案条件：

- 主链路缺陷已收口
- 请求追踪链路已补齐
- 安全策略已具备可见性
- 未来 SSO 接入路径已形成稳定设计锚点

当前没有必须继续推进的阻塞项。

## 6. 后续进入条件

只有在满足以下任一条件时，才建议重新开启下一轮 `system/auth` 身份能力整改：

- 已明确身份源类型，如 `OIDC`、企业微信、CAS、LDAP
- 已明确“外部主体 -> 本地账号”的绑定策略
- 已明确登录并存策略：仅本地、仅外部、双入口并存
- 已明确上游注销与本地会话语义

在上述条件未满足前，当前设计与实现保持不变即可。
