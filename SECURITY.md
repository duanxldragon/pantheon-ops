# Security Policy

English version: [SECURITY.en.md](./SECURITY.en.md)

## Supported Scope

Pantheon Platform 当前优先维护以下范围：

- `platform`：应用壳层、路由装配、工作台聚合
- `system/auth`：认证、会话、Token、安全中心
- `system/iam`：用户、角色、菜单、权限点、角色授权
- `system/org`：部门、岗位、组织架构
- `system/config`：字典、系统设置、配置缓存

如果问题涉及 `business/*`，请在报告中说明具体业务模块。

## Reporting a Vulnerability

请不要在公开 Issue 中直接披露以下信息：

- 管理员账号、密码、验证码
- Access Token、Refresh Token
- 数据库 DSN、Redis 密码、第三方密钥
- 可直接复现攻击的敏感 payload

建议按以下方式提交安全问题：

1. 用邮件或私有渠道先联系维护者；
2. 说明影响层级：`platform` / `system/*` / `business/*`；
3. 提供最小复现步骤、影响范围、日志或截图；
4. 如涉及权限绕过，请明确区分导航、页面、操作、接口四类授权中的哪一层失效。

## Response Expectations

- 收到报告后，会先确认问题归属层级与影响边界；
- 对高风险认证、授权、敏感配置泄露问题优先处理；
- 修复后会同步更新相关测试与文档，避免回归。
