# Auth 模块拆分设计

更新时间：2026-04-29

类型：Design
归属层：system/auth
状态：Active

本文用于把”认证与会话”从当前 `system/user` 混合实现中拆分为独立能力域，目标是在项目早期就锁定边界，避免后期继续把登录、用户管理、角色权限、安全策略揉成一个模块。

## 1. 设计目标

- **职责清晰**：`auth` 只负责“你是谁、是否可登录、会话是否有效”。
- **边界稳定**：`iam` 负责用户、角色、菜单、权限；`auth` 不承担后台管理 CRUD。
- **易于扩展**：当前已接入 TOTP MFA，并为后续验证码、SSO、登录设备、安全策略、租户登录做好演进空间。
- **先逻辑拆分，后代码实现**：当前已完成 `auth` 物理模块与 TOTP MFA 主链路，后续按边界继续演进 SSO、风控和租户登录。

## 2. 拆分结论

**结论：必须拆。**

但拆分方式是：

- **先做模块边界拆分**
- **再做代码目录拆分**
- **暂不做微服务拆分**
- **短期不强制拆表**

也就是说，当前阶段要把“认证域”从“系统管理域”中独立建模，而不是立刻拆成独立服务。

## 3. 当前问题

当前 `backend/modules/system/user/` 同时承担：

- 登录
- refresh token 轮换
- logout
- session 管理
- 当前用户 profile
- 密码修改
- 登录日志
- 用户列表 CRUD
- 角色绑定

这会导致几个长期问题：

1. **安全能力和后台管理耦合**
2. **文件职责持续膨胀**
3. **后续引入 SSO / 设备管理 / 风控时无处安放**
4. **AI 容易继续把 auth 逻辑写进 user 模块**

## 4. 目标模块边界

建议把底座系统域拆成下面几块：

| 能力域 | 说明 | 典型职责 |
| :--- | :--- | :--- |
| `auth` | 认证与会话安全 | login、refresh、logout、session、password、security |
| `iam` | 身份与授权管理 | user、role、menu、permission |
| `org` | 组织架构 | dept、post |
| `i18n` | 多语言 | 语言包、翻译资源 |
| `audit` | 审计 | 操作日志、登录日志 |
| `config` | 平台配置 | dict、setting |

### 4.1 `auth` 的职责

`auth` 负责：

- 登录认证
- refresh token 轮换
- logout
- 当前会话读取
- 会话吊销
- 修改当前登录用户密码
- 登录日志
- 安全策略（密码策略、登录失败限制、验证码、MFA 等）

当前已接入的安全策略包括：

- `security.password_min_length`：作用于当前用户修改密码
- `security.password_require_digit` / `security.password_require_uppercase`：作用于当前用户修改密码复杂度校验
- `security.password_history_limit`：限制最近 N 次历史密码不可复用；`0` 表示关闭
- `security.password_expire_days`：控制安全中心密码过期提醒；`0` 表示关闭
- `login.max_failed_attempts`：作用于登录失败累计阈值
- `login.lock_minutes`：作用于账号临时锁定时长
- `login.source_max_failed_attempts`：作用于同一来源/IP 在窗口内的失败阈值
- `login.source_window_minutes`：作用于来源/IP 失败次数的统计窗口
- `login.source_lock_minutes`：作用于来源/IP 临时锁定时长
- `login.security_event_enabled`：控制是否记录来源锁定、账号锁定等真实安全事件
- `login.session_idle_minutes`：作用于会话空闲超时判定，平台壳层会同步执行本地倒计时退出
- `login.max_active_sessions_per_user`：作用于同账号最大活跃会话上限；新登录超过阈值时自动下线更早的活跃会话，后台账号建议默认为 `1`
- `audit.session_retention_days`：作用于历史会话保留期；已下线或已过期会话超过天数后自动清理，避免 `system_user_session` 无界增长
- `login.mfa_enabled`：控制真实 TOTP 二次验证登录链路；默认关闭，关闭时保持原账号密码登录链路
- `login.captcha_enabled` / `login.sso_enabled`：作为后续验证码、SSO 的配置开关预留；当前默认关闭，不展示伪能力

同时，`system/iam` 中的用户创建与管理员重置密码也已消费 `security.password_min_length`，确保系统管理内部策略一致。`system_auth_security_event` 和 `system_user_password_history` 属于 `system/auth`，不应下沉到 `iam/user`。

### 4.2 `auth` 不负责

`auth` 不负责：

- 用户列表 CRUD
- 角色 CRUD
- 菜单 CRUD
- 权限策略 CRUD
- 部门岗位维护

这些属于 `iam` / `org`。

## 5. 推荐目录结构

### 5.1 后端

```text
backend/modules/
  auth/
    module.go
    auth_handler.go
    auth_service.go
    auth_dto.go
    session_model.go
    login_log_model.go
  system/
  user/
    user_handler.go
    user_service.go
    user_dto.go
    user_model.go
  role/
  menu/
  permission/
  dept/
  post/
```

### 5.2 前端

```text
frontend/src/modules/
  auth/
    Login.tsx
    SecurityCenter.tsx
    SessionList.tsx
    api.ts
    index.ts
  system/
  user/
  role/
  menu/
  permission/
  dept/
  post/
  profile/
```

## 6. API 边界建议

### 6.1 `auth` 域接口

建议收敛为：

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `POST` | `/api/v1/auth/login` | 登录 |
| `POST` | `/api/v1/auth/refresh` | 刷新 token |
| `POST` | `/api/v1/auth/logout` | 注销当前会话 |
| `GET` | `/api/v1/auth/me` | 获取当前登录主体信息 |
| `PUT` | `/api/v1/auth/me/preferences` | 更新当前登录主体的平台壳层偏好，归属 `platform` 壳层协同能力 |
| `GET` | `/api/v1/auth/sessions` | 获取当前账号在线会话，仅返回未吊销且 refresh token 未过期的活跃会话，当前会话优先展示 |
| `DELETE` | `/api/v1/auth/sessions/:id` | 下线某个会话 |
| `PUT` | `/api/v1/auth/password` | 修改当前账号密码 |
| `GET` | `/api/v1/auth/security` | 获取安全配置/状态 |

### 6.2 `system/iam` 域接口

建议保留：

- `/api/v1/system/user/*`
- `/api/v1/system/role/*`
- `/api/v1/system/menu/*`
- `/api/v1/system/permission/*`
- `/api/v1/system/dept/*`
- `/api/v1/system/post/*`

## 7. 数据模型建议

### 7.1 可保留现有表

当前阶段可以先保留：

- `system_user`
- `system_user_session`
- `system_log_login`

但逻辑归属改为：

- `system_user` 归 `iam`
- `system_user_session` 归 `auth`
- `system_log_login` 归 `audit/auth`

### 7.2 后续可扩展表

后续建议预留：

- `system_auth_policy`
- `system_auth_factor`（已用于 TOTP 二次验证因子）
- `system_auth_mfa_challenge`（已用于登录 MFA 短期挑战）
- `system_user_device`
- `system_login_risk_event`
- `system_user_external_identity`
- `system_auth_provider`

### 7.2.1 TOTP 二次验证当前实现

`login.mfa_enabled` 已进入真实实现，不再只是预留开关。

当前实现策略：

- 开关关闭：`POST /api/v1/auth/login` 按原账号密码链路直接签发 Access / Refresh Token。
- 开关开启：密码校验成功后不立即签发会话，而是创建 `system_auth_mfa_challenge` 并返回 `mfaRequired=true`。
- 已绑定 TOTP 因子的用户：前端调用 `POST /api/v1/auth/mfa/verify` 提交 challenge 与 6 位动态码，通过后签发会话。
- 未绑定 TOTP 因子的用户：首次登录时返回 TOTP 手动密钥与 `otpauth://` URI，用户用认证器应用绑定后提交 6 位动态码，验证通过后写入 `system_auth_factor` 并签发会话。
- MFA 密钥按 `PANTHEON_MFA_SECRET` 派生的 AES-GCM 加密保存；生产环境必须显式配置该 secret。

这个方案避免管理员在开启开关后因“未预先绑定因子”被锁死，同时保证验证码不是由后端直接回显的伪二次验证。

#### 7.2.1.1 登录主链路时序

当前 TOTP 二次验证的真实链路如下：

1. 用户访问 `/login`，输入用户名和密码。
2. 前端调用 `POST /api/v1/auth/login`。
3. 后端先完成账号存在性、状态、密码、失败锁定、来源/IP 节流等一次认证检查。
4. 若 `login.mfa_enabled=false`，则按原链路直接签发 Access / Refresh Token。
5. 若 `login.mfa_enabled=true`，则后端不立即签发会话，而是创建一条短期 `system_auth_mfa_challenge`，有效期当前为 5 分钟。
6. 若用户已绑定 TOTP 因子：
   用户看到二次验证码输入框，输入认证器应用中的 6 位动态码，前端调用 `POST /api/v1/auth/mfa/verify`。
7. 若用户尚未绑定 TOTP 因子：
   后端返回 `totpSecret` 与 `totpProvisionUri`，前端优先把 `otpauth://` URI 渲染成二维码，同时保留手动密钥与 URI 文本复制作为兜底。
8. 用户用支持 TOTP 的认证器应用扫码或手动录入密钥，获得 6 位动态码后提交 `POST /api/v1/auth/mfa/verify`。
9. 后端校验 challenge 是否存在、未消费、未过期，再用绑定中的 secret 验证 6 位动态码。
10. 首次绑定成功时，后端把加密后的 TOTP secret 落到 `system_auth_factor`，然后消费当前 challenge 并签发登录会话。

#### 7.2.1.2 TOTP 参数与实现约束

当前实现采用标准 TOTP 方案，参数固定为：

- 算法：`HMAC-SHA1`
- 动态码位数：`6`
- 时间步长：`30` 秒
- 时间偏移容忍：前后各 `1` 个步长
- 密钥编码：Base32
- 发行方（issuer）：`Pantheon Base`

这意味着动态码由“本地密钥 + 当前时间”推导得到，不依赖短信网关，也不依赖外部身份服务。

#### 7.2.1.3 最终用户如何使用

当前推荐使用方式：

1. 管理员在 `system/config -> login` 中开启 `login.mfa_enabled`。
2. 用户首次登录时，页面会进入二次验证设置态。
3. 用户使用支持 TOTP 的认证器应用扫码页面二维码。
4. 如果设备无法扫码，则手动录入页面展示的密钥，或复制 `otpauth://` URI 到支持导入的认证器。
5. 认证器应用会每 30 秒离线生成一个 6 位动态码。
6. 用户把当前 6 位动态码填回登录页，完成绑定并登录。
7. 后续再次登录时，只需在密码验证通过后输入认证器中的 6 位动态码。

适配说明：

- MFA 本身不要求连外网，认证器应用离线即可生成动态码。
- 真正需要准确的是客户端时间。如果手机或电脑时间偏差过大，动态码会校验失败。
- 普通扫码工具只能读出二维码文本，不能替代 TOTP 认证器。必须使用支持 `otpauth://totp/...` 或 TOTP 手动密钥导入的认证器应用。

#### 7.2.1.4 前端交互约束

登录页在 MFA 设置态必须满足：

- 二维码优先展示，降低手动录入错误率。
- 保留手动密钥展示与复制。
- 保留 `otpauth://` URI 文本复制，兼容支持 URI 导入的认证器。
- 在 challenge 过期、动态码错误、未绑定因子、来源受限等失败场景下，优先展示后端返回的错误 key 对应翻译，而不是统一模糊提示。

#### 7.2.1.5 运维与安全注意事项

- 生产环境必须显式配置 `PANTHEON_MFA_SECRET`，不得依赖开发默认值。
- `PANTHEON_MFA_SECRET` 变更后，历史已加密 TOTP secret 将无法解密，属于破坏性操作，必须按迁移方案执行。
- `system_auth_mfa_challenge` 只承担短期挑战，不可长期保存为用户永久因子。
- `system_auth_factor` 保存的是加密后的因子密钥，不允许通过接口回显真实 secret。
- MFA 是 `system/auth` 登录主链路的一部分，不应被塞回 `system/iam` 用户管理页承担长期职责。

### 7.3 SSO / 外部身份接入预留

当前阶段只保留设计，不实现真实 SSO 登录流。

原因不是“以后不会做”，而是：

- 当前项目还是以内网本地账号密码登录为主
- 未来身份源尚未确定，可能是 `OIDC`、`OAuth2`、企业微信、飞书、CAS、LDAP、ADFS 或其他企业身份系统
- 如果在没有明确 IdP 的前提下提前把 SSO 流程硬塞进现有登录主链路，极易把本地登录、会话刷新、登出和审计语义一起搅乱

因此当前设计原则是：

- **只预留扩展点，不预实现协议分支**
- **本地登录链路保持稳定，不因“预支持 SSO”而改成多分支脆弱实现**
- **等未来确定身份源后，再按单协议一次做完整闭环**

建议未来新增的模型边界如下：

| 模型/表 | 归属 | 作用 |
| :--- | :--- | :--- |
| `system_auth_provider` | `auth/config` | 保存身份源配置，如协议类型、client 配置、回调地址、启用状态 |
| `system_user_external_identity` | `auth/iam` 协同 | 保存“外部身份主体”与“本地用户”绑定关系 |

`system_user_external_identity` 建议至少具备以下语义字段：

- `provider_key`：身份源唯一标识
- `provider_type`：如 `oidc` / `oauth2` / `wechat-work` / `cas` / `ldap`
- `external_subject`：外部主体唯一值，如 `sub`、企业微信 `userid`
- `external_username`：外部身份回传用户名或账号
- `external_display_name`：外部身份展示名
- `user_id`：绑定到本地 `system_user.id`
- `tenant_code`：未来若支持多租户时可扩展
- `last_login_at`：最近一次外部登录时间
- `metadata_json`：保存协议差异字段，不把协议特例直接塞进主表

`system_auth_provider` 建议至少具备以下语义字段：

- `provider_key`
- `provider_type`
- `display_name`
- `issuer_or_base_url`
- `client_id`
- `client_secret_ref`
- `authorize_url`
- `token_url`
- `userinfo_url`
- `jwks_url`
- `scopes`
- `callback_path`
- `logout_url`
- `status`
- `sort`
- `extra_json`

注意：

- `client_secret` 不建议明文入库，推荐保存到安全配置中心或环境变量，再用 `client_secret_ref` 引用
- 不同协议的个性字段收敛到 `extra_json`，避免主表被某一种 IdP 污染

### 7.4 当前代码现状与未来接入点

当前 `system/auth` 已经具备“未来可接 SSO，但尚未开始接”的基础条件：

- 登录入口已独立收口到 `backend/modules/auth/` 与 `frontend/src/modules/auth/`
- `/api/v1/auth/login` 与 `/api/v1/auth/refresh` 已从 `system/user` 职责中抽离
- 会话、刷新令牌、登录日志、来源/IP 节流、请求链路审计都已在 `auth` 域形成独立闭环
- 安全策略中已预留 `login.sso_enabled` 配置开关，但当前只做状态展示，不驱动真实登录分支

当前仍然明确属于“本地登录主链路”的部分：

- 登录页只有账号/密码表单
- 后端登录服务直接查询本地 `system_user`
- 密码校验依赖本地密码哈希
- 登录成功后由 Pantheon 自己签发 access token / refresh token
- `system_user_session` 是本地会话事实来源

这意味着未来接入 SSO 时，正确方式不是改坏现有 `Login()`，而是新增“并行登录入口”。

建议未来按以下扩展点接入：

| 扩展点 | 当前状态 | 未来做法 |
| :--- | :--- | :--- |
| 登录入口 | 本地用户名/密码 | 增加 `/auth/sso/start/:provider`、`/auth/sso/callback/:provider` |
| 用户识别 | 本地 `system_user` | 新增外部主体绑定查询，先找绑定，再决定映射或拒绝 |
| 会话签发 | 本地 JWT + `system_user_session` | 保持不变，外部认证成功后仍落到本地会话签发 |
| 审计日志 | 已支持 requestId、登录日志 | 额外记录 provider、外部 subject、回调结果 |
| 安全设置 | 已有开关预留 | 增加 provider 级配置，而不是只靠一个布尔开关 |

核心原则：

- **外部身份认证成功，不等于直接信任外部会话进入业务系统**
- **Pantheon 仍然要把外部身份换成本地受控主体，并签发自己的会话**

这样做的好处是：

- 本地权限模型、菜单模型、Casbin、审计链路可以保持稳定
- 不需要把整个前后端鉴权体系改成“完全依赖外部 token”
- 本地登录和 SSO 登录可以长期并存

### 7.5 未来 SSO 接入流程建议

建议未来统一走以下抽象流程：

1. 前端点击某个身份源入口，如“企业微信登录”或“统一身份认证登录”
2. 前端跳转到 `/api/v1/auth/sso/start/:provider`
3. 后端按 `provider_key` 读取身份源配置，构造跳转地址并写入 `state / nonce`
4. 用户在外部 IdP 完成认证
5. IdP 回调 `/api/v1/auth/sso/callback/:provider`
6. 后端校验 `state / nonce / code / id_token / access_token`
7. 后端解析外部主体标识，如 `sub` 或企业微信 `userid`
8. 后端查询 `system_user_external_identity`
9. 如果已绑定，则定位本地 `system_user`
10. 如果未绑定，则按系统策略执行“拒绝 / 人工绑定 / 首次自动建号”
11. 完成本地主体决策后，继续复用现有 `CreateSession()` 和 JWT 签发链路
12. 返回前端，由前端进入正常业务壳层

这里必须刻意避免两种错误：

- 直接把 IdP 回传 token 当作平台业务 token 使用
- 在没有本地绑定策略前就允许任意外部身份直接进入后台

### 7.6 首次绑定与账号映射策略

未来正式接入前，必须先明确账号映射策略，不能边做边猜。

建议只允许以下三种显式模式之一：

1. **严格绑定模式**
   只有已建立 `external_identity -> local_user` 绑定关系的主体可以登录
2. **管理员预绑定模式**
   管理员先在后台把外部主体绑定到本地用户，再允许使用 SSO
3. **受控自动绑定模式**
   仅在明确可控的字段匹配规则下，允许首次登录自动绑定，如邮箱、工号、企业微信 `userid`

不建议默认启用：

- 任意首次登录自动建管理员账号
- 仅凭昵称或展示名匹配本地账号
- 没有审计记录的静默绑定

### 7.7 前端设计约束

在未来真正接入某种身份源之前，登录页应继续保持当前形态：

- 不展示假按钮
- 不展示“敬请期待”的 SSO 入口
- 不因为 `login.sso_enabled` 为 `true` 就默认渲染某种固定协议文案

未来真正接入后，前端建议只新增“身份源入口区”，不要把本地登录表单拆坏。

推荐结构：

```text
LoginPage
  ├── LocalLoginForm
  ├── Divider（当至少启用一个 provider 时显示）
  └── ExternalIdentityEntrances
       ├── ProviderButton(oidc)
       ├── ProviderButton(wechat-work)
       └── ProviderButton(custom)
```

即：

- 本地登录继续可用
- 外部身份入口按启用的 provider 动态展示
- provider 元数据来自后端，而不是写死在前端

### 7.8 审计、注销与风控约束

未来一旦接入 SSO，以下行为必须同步补齐：

- 登录日志新增 `provider_key`、`provider_type`、`external_subject`
- 失败日志需要区分“本地认证失败”和“外部身份认证失败”
- 注销语义需区分“退出本地会话”与“同时发起上游单点退出”
- 若身份源支持强制下线或 token introspection，必须明确是否接入，而不是默认假定可用
- 来源/IP 节流是否作用于 SSO 回调链路，需要按协议单独定义，不能简单复用密码输错模型

### 7.9 结论

当前 `system/auth` 的正确状态是：

- **架构已具备未来接 SSO 的边界基础**
- **实现上故意不提前做协议流**
- **短期继续以本地账号密码 + 本地会话为稳定主链路**

后续只要先确定身份源类型，再补 provider 配置、主体绑定、回调接口和审计语义，就能在不破坏现有登录的前提下接入。

## 8. 前端页面规划

`auth` 拆分后，前端建议形成三个层级：

### 8.1 认证入口页

- 登录页
- 忘记密码页（后续）
- TOTP MFA 二次验证流程（已集成在登录页）

### 8.2 安全中心页

- 当前账号安全概览
- 登录设备列表
- 在线会话管理
- 密码修改
- 安全策略提示

### 8.3 账号资料页

资料维护继续保留在 `profile`，但安全相关入口逐步迁移到 `auth/security`。

## 9. 分阶段重构计划

### Phase 1：文档锁边界

- 完成 `auth` 能力域设计
- 完成前后端职责划分
- 完成 API 与页面规划

### Phase 2：后端逻辑拆分

- 从 `system/user` 中抽离 login / refresh / logout / session / password / login log
- 新建 `backend/modules/auth/`
- 由独立 `auth.InitAuthModule()` 负责装配

### Phase 3：前端模块拆分

- `Login.tsx` 移到 `modules/auth/`
- 新增 `SecurityCenter.tsx`
- 新增 `SessionList.tsx`
- `ProfileCenter` 只保留资料维护

### Phase 4：接口收口

- 将认证相关路由逐步迁到 `/api/v1/auth/*`
- 兼容保留旧路径一段时间
- 更新前端请求封装和文档

## 9.1 当前实现状态

当前代码已完成第一轮物理拆分：

- 后端已新增 `backend/modules/auth/`
- 前端登录页已迁到 `frontend/src/modules/auth/Login.tsx`
- 前端认证请求已迁到 `frontend/src/modules/auth/api.ts`
- `auth` 已由独立 `auth.InitAuthModule()` 装配，不再挂在 `system` 物理目录下
- 后端已新增 `/api/v1/auth/login`、`/api/v1/auth/refresh`、`/api/v1/auth/logout`、`/api/v1/auth/me`、`/api/v1/auth/password`
- 旧 `/api/v1/system/*` 认证路径仍保留兼容

当前仍保留的过渡点：

- `profile` 页面已收口为资料维护，安全能力已独立进入 `auth/security`
- JWT 中间件仍直接校验 `system_user_session` 表
- 当前用户会话管理、当前用户登录日志、管理员登录日志页、管理员全局会话页已落地

当前下一批待推进：
- `system/auth` 已补会话活动上报接口 `/api/v1/auth/activity`，并在 `system_user_session.last_activity_at` 基础上执行空闲超时判定；
- `platform` 壳层已补锁屏按钮与解锁遮罩，锁屏不退出会话，但仍受空闲超时规则约束；

- `GET /api/v1/auth/security` 安全概览独立接口
- 管理员会话页的更完整筛选与设备信息解析
- 登录日志与安全事件进一步归入 `audit/auth`

其中第一项已完成当前阶段落地：

- `/api/v1/auth/security` 已返回当前用户信息、当前会话、活跃会话数、最近成功登录时间
- `/api/v1/auth/security` 已返回运行中安全策略快照，包括密码最小长度、账号失败锁定阈值、来源/IP 失败锁定阈值、锁定时长、空闲退出时长与安全特性开关状态
- 当前会话与会话列表已补充 `browser / os / device / userAgent`
- 登录日志写入时会基于 User-Agent 记录浏览器与操作系统基础识别结果
- 登录接口已补来源/IP 维度的抗喷洒节流，失败状态会写入 `system_login_throttle`

## 10. 本次设计决策

### 必须坚持

- `auth` 是独立能力域，不再视为 `user` 的附属功能
- 不为“快”继续把认证逻辑塞回用户管理模块
- 先拆边界，再拆代码，再优化接口路径

### 暂不做

- 暂不微服务化
- 暂不拆数据库
- 暂不一次性改完所有路径

## 11. 后续关联文档

- `DESIGN.md`
- `docs/designs/BACKEND.md`
- `docs/designs/FRONTEND.md`
- `docs/designs/FRONTEND_UI_SPEC.md`
- `AGENTS.md`
