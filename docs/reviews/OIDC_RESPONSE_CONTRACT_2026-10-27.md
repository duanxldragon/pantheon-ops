# OIDC 响应结构决策（RC-026 / S3 波次）

- **任务包**: `.harness/tasks/2026-10-27-s3-oidc-envelope/manifest.json`
- **决策日期**: 2026-10-28（maintainer-delegated，默认保留原则）
- **结论**: **六处裸 `c.JSON` 全部保留原样**——没有一处属于 IdP 协议面；同时该模块当前未挂载到任何路由，也没有前端消费方。

## 背景

审计发现 `backend/modules/auth/oidc/handler.go` 写了六处裸 `c.JSON`，而全库其余响应走
`common.Success` / `common.Fail` 信封。疑似与 tenant handler 同类的 envelope 漂移。
本波次逐条对照 OIDC 协议合同分类：属于协议合同的保留裸响应并注明合同；确实是业务端点的迁到
common 信封。

## 分类表（逐条对照 RFC 6749 / OpenID Connect Core 1.0）

| # | 端点 | 响应 | 协议合同？ | 判定 |
|---|------|------|-----------|------|
| 1 | `GET /auth/oidc/login` 失败 | 500 `{"error"}` | 否——RP 自己的登录入口；协议元素只有下方的 302 重定向（RFC 6749 §4.1.1） | 保留裸响应（浏览器导航消费；无 IdP/SPA 解析此 JSON） |
| 2 | `GET /auth/oidc/callback` 缺参 | 400 `{"error"}` | 否——redirect_uri 由用户浏览器到达；IdP 侧协议错误以重定向 query 参数传递（RFC 6749 §4.1.2.1），没有任何一方读取此 JSON 体 | 保留裸响应 |
| 3 | `GET /auth/oidc/callback` 认证失败 | 401 `{"error","details"}` | 否——同上，IdP 不会解析此响应 | 保留裸响应 |
| 4 | `GET /auth/oidc/callback` 成功 | 200 `{success,user,is_new,message}` | 否——handler 自带 TODO（:62/:80）：会话服务尚未接入，这是**临时业务负载**；迁移信封必须与 session 接入及其前端消费方一起做 | 保留裸响应；信封迁移归 session 接入 PR |
| 5 | `GET /auth/oidc/config`（禁用） | 200 `{enabled:false}` | 否——前端发现配置；当前**无路由挂载、无前端消费方**（见下） | 保留裸响应；未来 SSO 落地时与 SPA 消费方一起改信封 |
| 6 | `GET /auth/oidc/config` | 200 `{enabled,loginURL,provider,scopes}` | 否——同上 | 保留裸响应 |

另：`c.Redirect(302)`（handler.go:35）是唯一真正的协议元素（RFC 6749 §4.1.1 授权请求重定向），
非 JSON，不在六处之列，未改动。

## 关键事实：模块当前处于休眠状态

- 六个 handler 均未在任何 router 注册（`grep -rn "InitiateOIDCLogin|HandleOIDCCallback|GetOIDCConfig" backend --include=*.go` 仅命中 oidc 包自身）；
- 前端无任何 OIDC 消费方（仅 i18n 文案提及「为后续 SSO / OIDC 接入预留的控制开关」）。

因此「IdP 不能改解析器所以必须保留裸响应」的论证甚至不需要动用——当前根本不存在网络消费方。
**分类为将来 SSO 落地提供了边界**：届时 #4/#5/#6（业务形状）应与 session 接入、SPA 消费方一起
迁到 common 信封；#1/#2/#3 保持面向浏览器的原始形状即可。

## 决策依据

1. **默认保留**（任务包 implementationNotes）：交付物是分类而非 diff；「一个零改动但说明了为什么的波次是成功的波次」。
2. **响应体/状态/字段变更在 scope.out**（任务包 scope.out 第 1 条）。
3. **信封变更需要人工确认**（任务包 humanGates）：auto 模式下无人可确认，取默认。

## 运行态证据 gap

无法执行「一次 OIDC 登录往返」：模块未挂路由（无入口可打）且无 IdP 配置。此 gap 与分类结论
一致（休眠模块），由未来 SSO 接入任务包补齐。
