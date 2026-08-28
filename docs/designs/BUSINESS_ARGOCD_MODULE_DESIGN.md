# ArgoCD GitOps 集成业务模块设计

更新时间：2026-08-28
类型：Design
归属层：business/cicd/argocd
状态：Draft（v1.2 目标；当前代码为壳级实现，菜单与路由在 V1 冻结版下架，见 `V1_FREEZE_REMEDIATION_DEV_PLAN.md` Phase 0-2）

本文定义 pantheon-ops 与 ArgoCD 的集成设计。核心原则沿用 `BUSINESS_DEPLOY_MODULE_DESIGN.md` §1：**扩展点不提供伪成功实现**——2026-08-28 评审确认当前代码违反了该原则（`SyncApplication` 未调用 ArgoCD API 即写入 Succeeded 历史并被判定为 P0 缺陷），本文是整改后的目标设计。

## 1. 模块定位与边界

ArgoCD 模块负责：GitOps 应用声明的纳管、同步触发、状态呈现、回滚。

本模块不负责：

- 镜像构建与推送（属于代码→镜像链路，V1 不做）。
- Git 仓库托管与 Webhook 接收（V1 只做轮询，不做推送式同步）。
- 部署包与 SSH 部署（属于 `business/deploy`）。

架构关系（与部署双轨）：

```
VM 场景:   Ops Deploy → SSH → 主机
K8s 场景:  Ops Deploy → ArgoCD Application → 集群
```

## 2. 数据模型

四表结构保留（`27209f5` 已建）：

| 表 | 关键字段 | 说明 |
|---|---|---|
| `biz_argocd_server` | name, url, auth_token(加密), insecure, status | ArgoCD 实例注册，支持多实例 |
| `biz_argocd_repository` | server_id, repo_url, auth_type, credential_ref | Git 仓库凭据走 CredentialRef，不入明文 |
| `biz_argocd_application` | name, server_id, repo_url, path, target_revision, destination_cluster, destination_namespace, sync_policy(JSON), sync_status, health_status, revision, business_scope_id | Application 声明 + 缓存状态 |
| `biz_argocd_sync_history` | application_id, action(sync/rollback/refresh), revision_before, revision_after, result, message | 操作审计与回滚依据 |

与业务域贯通：application 必须挂 business_scope_id；列表查询走 DataScope（注册入 `contracts.DataScopedGroup`，修复当前裸路由问题）。

## 3. 安全要求（P1 修复项）

1. `auth_token`/`password`/`ssh_key` 必须走统一 `v1:` AES-256-GCM 加密（复用 `deploy_credential.go` 实现），禁止明文入库——当前代码仅 `json:"-"` 不回显，属 P1 缺陷。
2. 路由必须经 `DataScopedGroup`（TokenAuth + Casbin + DataScope + RequireWrite），当前裸 RouterGroup 注册属 P0 缺陷。
3. 权限键：`business:cicd:argocd:{view|create|update|delete|sync|rollback}`；sync/rollback 为危险操作，挂 SecureAction 二次确认。

## 4. 生命周期（真实实现要求）

```
注册 Server（连通性真实探测，失败拒绝保存）
→ 注册 Repository（在 ArgoCD 侧创建 repo 连接并探测）
→ 创建 Application（调用 ArgoCD API 创建，成功才落库）
→ 同步 Sync（调用 API，返回 API 错误即任务失败，禁止本地伪造 Succeeded）
→ 状态轮询（后台 reconciler 定时刷新 sync_status/health_status/revision）
→ 回滚 Rollback（基于 sync_history 的 revision_before，调用 ArgoCD API 回退）
→ 删除（调用 ArgoCD API 删除 Application，本地级联清理）
```

实现依赖：`github.com/argoproj/argo-cd/v2` pkg/apiclient（server 端 token 认证）。

## 5. 状态同步

- reconciler 每 60s（可配）轮询已注册 server 的 Application 列表，diff 后更新本地缓存字段。
- 轮询失败：server 置 unreachable，页面显示离线状态，不删除缓存。
- 集群侧删除的 Application：本地标记 orphaned，提示用户处理。

## 6. 验收标准（v1.2）

1. TestServerConnection 对不可达地址返回失败（当前恒 true 属 P0）。
2. Sync 全链路在真实 ArgoCD 实例（可用 k3s + argocd helm chart 搭建验收环境）验证：sync → healthy → 修改 git revision → drift 检测 → rollback 到上一 revision。
3. 断网/错误 token 场景：操作失败且历史记录 result=failed，无伪造成功。
4. 权限：无 `business:cicd:argocd:sync` 权限的用户调用 sync 返回 403。
