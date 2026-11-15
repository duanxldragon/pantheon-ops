# 租户初始化指南

最后更新：2026-11-09。已发行行为适用于租户 V1；V2.0-O4 业务增量正在实施，未完成正式发行和消费验收。唯一权威是 [租户合同](./contracts/TENANT_CONTRACT_V1.md)，资源分类见 [scope 矩阵](./designs/TENANT_RESOURCE_SCOPE_MATRIX.md)，接口字段见 [租户 API](./api/TENANT_API.md)。

## 先确认模式和主体

默认 `compat`，多租户运行使用 `multi`。读取 `GET /api/v1/system/setting/public` 的 `data.settings["platform.tenant_mode"]`，值是无额外引号的字符串。配置存于 `system_setting`；编辑旧 `config/application.yml` 不能替代运行态配置变更。

- 平台主体可以保留 `tenant_id = 0`、无 membership；调用租户管理还必须持有平台能力。
- 租户主体的拥有关系、活动 membership、租户内角色绑定共同决定登录和权限。membership 的 `owner/admin/member` 是成员关系角色，不自动授予 IAM/Casbin 权限。
- `tenant_id = 0` 是平台/全局作用域，不应把全部全局菜单、权限、设置和平台用户搬入默认租户。
- 系统域按当前 V1 合同隔离；业务域 V1 不参与多租户隔离，套餐配额只度量与展示。

V2.0-O4 首批只采用资源矩阵登记的五张 observability 表。历史行保留全局归属0；历史和保存视图同时按租户、用户隔离。初始化时分别建立租户成员和租户内 IAM 权限，成员角色不能代替查询或保存视图权限。入口、provider 调用前及结果释放前验证当前会话、主体、模式、角色绑定、成员和租户状态；缺失模式配置或授权依赖拒绝请求。该目标及真实双租户验收由 [O4 实施计划](../.harness/tasks/2026-11-07-ops-formal-queue-foundation-gaps/o4-tenant-implementation-plan.md) 跟踪；局部 API 测试不代表消费产品已经隔离。

从 `backend` 使用配置好的 `PANTHEON_DSN` 运行 `go run ./cmd/tenantmigration status`，确认当前发布迁移链版本和 `dirty=false`。迁移应用、生产数据转换与备份恢复遵循发布和 [升级指南](./migrations/COMPAT_TO_MULTI_UPGRADE.md)。

## 通过管理控制台或 API 创建租户

使用具有平台能力的认证会话。`POST /api/v1/system/tenant` 的示例请求体：

```json
{
  "code": "acme",
  "name": "Acme 公司",
  "plan": "free",
  "admin_username": "acme-admin",
  "admin_password": "<符合实际密码策略的初始密码>",
  "admin_nickname": "Acme 管理员"
}
```

请求走公共响应 envelope，并携带会话 cookies 与 CSRF 请求头；触发高敏操作要求时，通过现有 `/api/v1/auth/operation-verify` 流程提供 operation token。

管理员字段使用 **snake_case**，不是 `adminUsername`。省略 `admin_username` 时，只初始化租户基础资源，不创建可登录管理员；这种配置不能作为“已可使用”验收。套餐代码由当前启用的套餐定义决定，既有四档为 `free/basic/professional/enterprise`，平台可维护自定义套餐。

创建租户行后，provision 事务初始化租户角色、初始管理员、角色绑定、owner membership、授权策略和默认资源；初始化失败时 service 尝试补偿删除租户行，补偿失败会记录错误。成功创建并提供管理员时，应立即能登录并操作本租户资源。初始化 helper 的幂等性是内部执行契约；重复创建同编码租户的 API 会报冲突，不存在另一个通用公开“重跑初始化”接口。

不要用只插入 `tenants` 的 SQL 行作为可用租户。绕过 service 写入用户/membership 会遗漏角色、策略、关联和审计。修复存量初始化缺口应使用已有启动 repair/helper 或单独评审的数据修复，而不能虚构一个 API。

## 登录、查看和切换

当前登录界面直接使用用户名和密码登录，已经移除历史登录前租户 picker；API 可处理合同允许的租户请求。凭据是 Redis opaque token，会话通过 cookies 保持，不能按旧 JWT/响应体 `token` 示例实现客户端。

- `GET /api/v1/auth/tenants/current`：当前租户。
- `GET /api/v1/auth/tenants/switchable`：当前主体允许切换的租户。
- `POST /api/v1/auth/tenants/switch`：按真实活动租户、membership 和角色绑定校验，拒绝无权或同租户切换。

存量跨租户 membership 不等于可登录/可切换目标；V1 已收窄到有效角色绑定。普通成员创建/导入和权限分配必须走所属租户的系统域操作。

## 成员和生命周期管理

优先使用控制台或 `/api/v1/system/tenant/:id/members` 及对应删除路由。添加成员只接受拥有关系在目标租户的用户。不能删除最后一个活动 owner。

成员移除的当前 V1 写路径删除 membership；不要照旧指南写入不存在的 `inactive` 状态，或把 soft-disable 当成已经提供的管理 API。授权角色与成员关系是两条链，必须分别验证。

租户状态为 `active/suspended/archived`；归档是终态，删除要求先归档。删除在事务中禁用剩余活动 membership、释放原编码并写内部 soft-delete 标记。暂停/归档和成员移除会尝试撤销受影响会话；撤销是 best-effort，失败记录日志，登录和刷新状态校验仍是兜底。删除依赖此前归档与状态校验，不额外承诺一次独立的会话撤销。不要直接更新虚构的 `system_user_session.status/token` 列。验收应检查登录、刷新与后续访问，以及撤销失败日志。

## 租户运营和套餐

平台租户运营提供 overview、成员视图与套餐编辑。已被租户使用的套餐不能停用；先把租户迁移到有效套餐，再停用原套餐。配额阈值、实际用量与展示应一致，当前 V1 不因超配额拒绝写入。

## 验收与排障

在隔离测试实例分别验证 compat 和 multi；multi 套件前确认模式已经启用。从 `frontend` 运行 `npm run test:smoke:tenant`，从 `backend` 使用真实隔离 MySQL/Redis 和 required flags 运行完整 `go test -race ./...`（Windows 显式 cgo/MinGW）。

验收至少包括：创建初始管理员、租户登录和系统 CRUD、owner 移除拒绝、暂停/归档/删除、套餐分配及停用拒绝、跨租户 ID/头/导出/上传拒绝、刷新重放和撤销。测试数据只在专用实例使用既有 fixture cleanup 清理。

出现 `403 permission.denied` 先确认运行模式、主体拥有关系、有效角色和 scope，再检查请求；不要禁用 tenant middleware/Casbin 来使烟测变绿。初始化、运维与运行证据以 [当前任务矩阵](https://github.com/duanxldragon/pantheon-base/blob/135730fd6f63ad73cf40f6e4b321fdd9f69ea932/.harness/evidence/2026-11-05-base-freeze-release/functionality-matrix.md) 为准，历史 `.tmp` 临时脚本不能作为当前交付凭证。

从 compat 升级见 [升级指南](./migrations/COMPAT_TO_MULTI_UPGRADE.md)。multi 产生写入后，回滚必须恢复完整已验证备份及匹配的应用/会话状态，不能仅把模式改回 compat。
