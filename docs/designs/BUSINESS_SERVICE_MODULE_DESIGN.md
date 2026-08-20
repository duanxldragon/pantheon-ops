# Service 业务模块设计

更新时间：2026-08-20  
类型：Design  
归属层：business/service  
状态：Proposed - P0 Required  
架构评审依据：[ARCHITECTURE_REVIEW_REPORT_2026_08_20.md](../assessments/ARCHITECTURE_REVIEW_REPORT_2026_08_20.md)

本文定义 `pantheon-ops` 中 `business/service` 模块的设计，该模块是 **2026-08-20 架构评审发现的 P0 阻断问题**。ServiceInstance 抽象层缺失导致无法建立业务视图，必须在 V2 第一优先级补齐。

## 1. 问题背景

### 1.1 架构评审发现

**P0-1 问题:** ServiceInstance 抽象层缺失

目标架构设计明确提出 `BizScope → Application → Service → ServiceInstance → TargetRef` 链路，但当前代码中：
- ❌ 无 `biz_service` 表
- ❌ 无 `biz_service_instance` 表
- ❌ Deploy 任务直接绑定 `package_id` 和 `business_scope_id`，跳过了 Service 抽象
- ❌ K8s Release 也未关联 ServiceInstance

**影响:**
1. 无法回答"HIS 系统有哪些服务"、"订单服务部署在哪些主机"等基本问题
2. 部署任务与业务系统脱钩，无法建立"业务视图"
3. 可观测性无法按 Service 维度聚合（因为不知道哪些主机/Pod 属于同一服务）

### 1.2 模块定位

`business/service` 是连接业务域(BizScope)与资源(CMDB/K8s)的关键抽象层：

```text
BizScope (业务域: HIS系统)
  └─ Application (应用: 挂号服务)
      └─ Service (服务: API服务)
          └─ ServiceInstance (实例: prod环境实例1)
              └─ TargetRef (目标: 主机192.168.1.10 或 K8s Deployment)
```

**本模块解决:**
- 维护服务目录（Service Catalog）
- 管理服务实例生命周期
- 建立服务与资源的映射关系
- 为可观测性提供服务维度聚合基础

**本模块不负责:**
- 业务域管理，属于 `business/bizscope`
- 资源台账，属于 `business/cmdb`
- 部署执行，属于 `business/deploy`
- K8s 运行态，属于 `business/k8s`

## 2. 数据模型

### 2.1 biz_service (服务)

服务是业务系统中的逻辑组件，如"订单API"、"用户Web"、"支付Worker"。

| 字段 | 类型 | 说明 | 约束 |
|---|---|---|---|
| `id` | BIGINT | 主键 | PK, AUTO_INCREMENT |
| `business_scope_id` | BIGINT | 所属业务域 | NOT NULL, INDEX |
| `code` | VARCHAR(64) | 服务编码 | UNIQUE, NOT NULL |
| `name` | VARCHAR(128) | 服务名称 | NOT NULL |
| `description` | VARCHAR(512) | 服务说明 | |
| `service_type` | VARCHAR(32) | 服务类型 | `api` / `web` / `worker` / `database` / `cache` / `mq` |
| `owner` | VARCHAR(64) | 负责人 | |
| `status` | VARCHAR(32) | 服务状态 | `active` / `inactive` / `archived` |
| `metadata` | JSON | 扩展元数据 | 技术栈、依赖、文档链接等 |
| `created_at` | DATETIME | 创建时间 | NOT NULL |
| `updated_at` | DATETIME | 更新时间 | NOT NULL |
| `deleted_at` | DATETIME | 软删时间 | INDEX |
| `created_by` | VARCHAR(64) | 创建人 | |
| `updated_by` | VARCHAR(64) | 更新人 | |

**唯一约束:** `uk_service_code_deleted` (`code`, `deleted_at`)

**索引:**
- `idx_business_scope_id`
- `idx_service_type`
- `idx_status`

### 2.2 biz_service_instance (服务实例)

服务实例是服务在特定环境的部署实例，关联到具体的主机或 K8s Workload。

| 字段 | 类型 | 说明 | 约束 |
|---|---|---|---|
| `id` | BIGINT | 主键 | PK, AUTO_INCREMENT |
| `service_id` | BIGINT | 所属服务 | NOT NULL, INDEX |
| `environment` | VARCHAR(32) | 部署环境 | `dev` / `test` / `staging` / `prod` |
| `instance_name` | VARCHAR(128) | 实例名称 | NOT NULL |
| `target_type` | VARCHAR(32) | 目标类型 | `vm` / `k8s` |
| `target_ref` | JSON | 目标引用 | `{kind: "Host", id: 123}` 或 `{kind: "K8sWorkload", cluster: "...", namespace: "...", name: "..."}` |
| `version` | VARCHAR(64) | 部署版本 | 镜像tag或软件包版本 |
| `status` | VARCHAR(32) | 实例状态 | `pending` / `deploying` / `running` / `failed` / `stopped` |
| `health_status` | VARCHAR(32) | 健康状态 | `healthy` / `unhealthy` / `unknown` |
| `installed_packages` | JSON | 已装软件组件 | `[{name: "nginx", version: "1.21.0"}]` |
| `deployment_status` | VARCHAR(32) | 部署状态 | `not_deployed` / `deployed` / `upgrade_pending` |
| `last_deployed_at` | DATETIME | 最近部署时间 | |
| `metadata` | JSON | 扩展元数据 | 配置参数、资源限制等 |
| `created_at` | DATETIME | 创建时间 | NOT NULL |
| `updated_at` | DATETIME | 更新时间 | NOT NULL |
| `deleted_at` | DATETIME | 软删时间 | INDEX |

**索引:**
- `idx_service_id_environment`
- `idx_target_type`
- `idx_status`
- `idx_health_status`

### 2.3 设计约束

1. **Service 必须归属 BizScope:** `business_scope_id` 不能为空
2. **ServiceInstance 必须关联 Service:** `service_id` 不能为空
3. **target_ref 结构化验证:** Deploy 和 K8s 在回写前必须校验 JSON 结构
4. **状态流转单向:** 
   - Service: `active` ↔ `inactive` → `archived`
   - ServiceInstance: `pending` → `deploying` → `running` / `failed` / `stopped`
5. **已装组件迁移:** 从 `biz_cmdb_host.installed_components` 迁移到 `ServiceInstance.installed_packages`
6. **部署状态迁移:** 从 `biz_cmdb_host.status` 迁移到 `ServiceInstance.deployment_status`

## 3. 与其他模块的关系

### 3.1 与 BizScope 的关系

```text
BizScope (1) ---< (N) Service
```

- Service 必须归属一个 BizScope
- BizScope 删除前必须检查是否有关联的 Service
- Service 可以查询所属 BizScope 的业务域信息（通过 capability）

### 3.2 与 CMDB 的关系

```text
ServiceInstance.target_ref -> Host (通过 capability 查询)
```

- ServiceInstance 通过 `target_ref` 引用 Host，不直接依赖 `biz_cmdb_host` 表
- CMDB 提供 capability：`ResolveHostByTargetRef(targetRef) -> Host 快照`
- **已装组件迁移:** `biz_cmdb_host.installed_components` 停用，改用 `ServiceInstance.installed_packages`
- **主机状态简化:** `biz_cmdb_host.status` 只保留 `pending` / `assigned`，运行态状态由 `ServiceInstance.deployment_status` 表达

### 3.3 与 Deploy 的关系

```text
DeployTask (N) ---< (1) ServiceInstance
```

- Deploy 任务创建时必须选择 ServiceInstance（不再直接选择 Host）
- 部署成功后回写 `ServiceInstance.version / status / last_deployed_at / installed_packages`
- 部署失败时更新 `ServiceInstance.status = failed`

### 3.4 与 K8s 的关系

```text
K8sRelease (N) ---< (1) ServiceInstance
```

- K8s Release 创建时必须关联 ServiceInstance
- Release 成功后回写 `ServiceInstance.target_ref / version / status`
- ServiceInstance 可以有多个 TargetRef（多副本场景）

### 3.5 与 Observability 的关系（V2）

```text
ServiceInstance -> Metrics/Logs/Traces 聚合维度
```

- 按 `service_id` 聚合所有实例的指标
- 按 `service_id + environment` 区分不同环境的监控
- ServiceInstance 提供标签映射：`{service: "order-api", env: "prod", instance: "order-api-prod-1"}`

## 4. API 设计

前缀：`/api/v1/business/service`

### 4.1 Service API

| 方法 | 路径 | 说明 | 权限点 |
|---|---|---|---|
| `GET` | `/services` | 服务列表 | `business:service:list` |
| `GET` | `/services/:id` | 服务详情 | `business:service:detail` |
| `POST` | `/services` | 创建服务 | `business:service:create` |
| `PUT` | `/services/:id` | 编辑服务 | `business:service:update` |
| `DELETE` | `/services/:id` | 删除服务 | `business:service:delete` |
| `GET` | `/services/:id/instances` | 服务的所有实例 | `business:service:detail` |

### 4.2 ServiceInstance API

| 方法 | 路径 | 说明 | 权限点 |
|---|---|---|---|
| `GET` | `/instances` | 实例列表 | `business:service:instance:list` |
| `GET` | `/instances/:id` | 实例详情 | `business:service:instance:detail` |
| `POST` | `/instances` | 创建实例 | `business:service:instance:create` |
| `PUT` | `/instances/:id` | 编辑实例 | `business:service:instance:update` |
| `DELETE` | `/instances/:id` | 删除实例 | `business:service:instance:delete` |
| `PATCH` | `/instances/:id/status` | 更新实例状态（Deploy/K8s 回写） | `business:service:instance:update-status` |
| `PATCH` | `/instances/:id/health` | 更新健康状态（监控回写） | `business:service:instance:update-health` |

### 4.3 错误 Key

| Key | 说明 |
|---|---|
| `business.service.notFound` | 服务不存在 |
| `business.service.codeExists` | 服务编码重复 |
| `business.service.inUse` | 服务仍有实例，不能删除 |
| `business.service.instance.notFound` | 服务实例不存在 |
| `business.service.instance.targetRefInvalid` | 目标引用格式无效 |
| `business.service.instance.targetNotFound` | 目标资源不存在 |

## 5. 迁移计划

### 5.1 阶段 1: 建表与基础 CRUD (第 1 周)

1. 创建 `biz_service` 和 `biz_service_instance` 表
2. 实现 Service 和 ServiceInstance 的 CRUD API
3. 实现前端服务列表、服务详情、实例列表页面
4. 补齐权限点、菜单、i18n

### 5.2 阶段 2: Deploy 集成 (第 2 周)

1. Deploy 任务创建表单增加"选择服务实例"步骤
2. 部署成功后回写 `ServiceInstance.installed_packages / deployment_status`
3. 迁移现有 `biz_cmdb_host.installed_components` 到 ServiceInstance

### 5.3 阶段 3: K8s 集成 (第 2 周)

1. K8s Release 创建时关联 ServiceInstance
2. Release 成功后回写 `ServiceInstance.target_ref / version / status`

### 5.4 阶段 4: CMDB 边界清理 (第 1 周)

1. 停用 `biz_cmdb_host.installed_components` 字段（标记为 deprecated）
2. 简化 `biz_cmdb_host.status`：只保留 `pending` / `assigned`
3. 新增 `biz_cmdb_host.connectivity_status` 表达网络可达性

## 6. 验收标准

- ✅ Service 和 ServiceInstance 表结构创建
- ✅ Service CRUD API 可用
- ✅ ServiceInstance CRUD API 可用
- ✅ 前端服务列表、详情页可用
- ✅ Deploy 任务关联 ServiceInstance
- ✅ K8s Release 关联 ServiceInstance
- ✅ `biz_cmdb_host.installed_components` 迁移完成
- ✅ 权限点、菜单、i18n 补齐
- ✅ 文档与代码同步

## 7. 后续扩展 (V3)

- Application 层：Service 归属到 Application
- 服务依赖关系：`Service A 依赖 Service B`
- 服务拓扑图：可视化服务调用关系
- 服务健康评分：根据实例健康状态和 SLO 计算服务健康度
- 服务成本分配：按 ServiceInstance 消耗的资源统计成本

---

**交付时间:** 预计 2 周（V2 第一冲刺）  
**优先级:** P0（架构评审阻断问题）  
**责任人:** Service/Deploy/K8s owner
