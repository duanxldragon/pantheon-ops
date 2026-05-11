# SSO / OAuth2 / OIDC 设计

更新时间：2026-05-05

类型：Design
归属层：system/auth
状态：Draft

本文定义 Pantheon 后续接入 SSO / OAuth2 / OIDC 的边界。当前不实现真实 SSO，本文件用于避免未来临时拼接身份源导致认证域混乱。

---

## 1. 目标

SSO 的目标是允许外部身份源完成登录认证，并映射到本地用户与本地权限模型。

本地系统仍保留：

- 本地用户。
- 本地角色。
- 本地菜单和权限。
- 本地会话和审计。

## 2. 非目标

当前阶段不做：

- 多 provider 自动发现。
- 企业微信、钉钉、LDAP、SAML 的具体适配。
- 登录页展示未配置的 SSO 入口。
- 外部身份直接绕过本地用户和角色授权。

## 3. 核心模型

建议后续模型：

- `system_auth_provider`
- `system_external_identity`

关键字段：

- provider code。
- provider type。
- client id。
- encrypted client secret。
- issuer / authorize url / token url / userinfo url。
- local user id。
- external subject。
- optional tenant code 扩展位。

## 4. 登录流程

1. 用户选择已启用 provider。
2. 后端生成 state 和 PKCE verifier。
3. 跳转 provider authorize endpoint。
4. 回调校验 state。
5. 后端换 token 并获取 userinfo。
6. 通过 external subject 绑定本地用户。
7. 创建 Pantheon 本地 session。
8. 写登录日志和安全审计。

## 5. 本地登录兜底

除非设计明确替代本地登录，否则 SSO 启用后必须保留本地登录兜底。

原因：

- provider 故障时管理员仍需进入系统。
- 初期用户绑定可能不完整。
- 权限仍归属本地 `system/iam`。

## 6. 安全约束

- client secret 必须加密存储。
- state 必须一次性使用。
- 回调失败必须写审计。
- 外部身份绑定和解绑必须审计。
- 禁止把 provider 返回的角色直接当作本地权限。

## 7. 验收

进入实现前必须明确：

- provider 类型。
- 回调地址。
- 外部身份唯一键。
- 本地用户绑定策略。
- 本地登录是否保留。
- 注销语义。
- 审计事件。
