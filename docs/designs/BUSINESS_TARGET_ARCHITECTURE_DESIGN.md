# pantheon-ops 目标业务架构设计

更新时间：2026-08-18  
类型：Design  
归属层：business  
状态：Proposed  
关联评审：[BUSINESS_ARCHITECTURE_REVIEW.md](./BUSINESS_ARCHITECTURE_REVIEW.md)

本文定义 pantheon-ops 从当前模块化单体演进到企业级 SRE 平台的业务目标架构。它只规定 ops 业务域的边界、对象和交互；平台壳层、IAM、组织、审计、工作流、配置和通用 UI 仍以 `pantheon-base` 为唯一来源。

## 1. 设计原则

1. **逻辑解耦优先，物理拆分后置。** V1 保持模块化单体，只有在独立伸缩、故障隔离或外部协议需要时才拆服务。
2. **一个事实源，一个写入者。** 下游只读取稳定 capability，不直接读对方表；运行态事实不复制回 CMDB。
3. **声明意图，异步执行，持续调和。** API 创建不可变 intent，Worker 执行，Reconcile 观察外部状态并回写。
4. **所有高风险动作均有边界、审计和可回滚策略。** DataScope、资源所有权、权限点和变更记录必须在 mutation 前生效。
5. **引用优于嵌入。** 业务对象使用稳定 `ResourceRef`/`TargetRef`，不嵌入 CMDB 或 K8s 的内部模型。
6. **可观测性是外部事实。** 指标、日志、Trace 由对应系统保存，ops 只保存查询索引、责任绑定和摘要。

## 2. 逻辑模块地图

```text
                     +---------------------------+
                     | pantheon-base              |
                     | IAM / Org / Audit / Config |
                     +-------------+-------------+
                                   |
             +---------------------+----------------------+
             |                                            |
  +----------v----------+                       +---------v---------+
  | BizScope / Service  |                       | CMDB Resource      |
  | 业务目录与责任       |<---- ResourceRef ---->| 资源身份/关系/生命周期|
  +----------+----------+                       +---------+---------+
             |                                            |
             | TargetRef                                  | Capability
             v                                            v
  +----------+----------+                       +---------+---------+
  | Change / Deploy     |---- Executor -------->| SSH/Agent/K8s/... |
  | 意图/计划/执行/调和  |                       +-------------------+
  +----------+----------+
             |
             v
  +----------+----------+                       +-------------------+
  | K8s provider        |---- Kubernetes API -->| Cluster runtime    |
  | 多集群/对象/Release  |                       +-------------------+
  +----------+----------+
             |
             v
  +----------+----------+                       +-------------------+
  | Observability refs  |<---- query/link ------| Prom/Vm/Loki/ES/OTel|
  | SLO/Alert/Incident   |                       +-------------------+
  +---------------------+
```

上述模块是**逻辑边界**，V1 可以全部部署在同一进程和数据库中；数据库表、Service、capability 和权限仍按模块隔离。

## 3. 领域与事实源

### 3.1 CMDB Resource Domain

CMDB 负责“资源是什么、归谁、生命周期如何、与什么有关系”，不负责部署结果和时序观测事实。

目标对象：

```text
ResourceType
  - code, version, schema, lifecycle_policy
ResourceInstance
  - id, type, external_id, name, status, environment, region
ResourceAttribute
  - resource_id, key, value, value_type, source, observed_at
ResourceRelation
  - from_id, relation_type, to_id, source, confidence
ResourceOwnership
  - resource_id, business_scope_id, service_id, department_id, valid_from/to
ResourceObservation
  - resource_id, source, summary, observed_at, expires_at
LifecycleHistory
  - resource_id, event, actor, reason, occurred_at
```

现有 Host/Group/Label 模型作为兼容适配层逐步迁移，不要求一次性重写；新资源类型优先使用通用模型。

### 3.2 Business Catalog Domain

```text
BizScope -> Application -> Service -> ServiceInstance -> TargetRef
```

- `BizScope`：业务边界、部门范围、责任人和生命周期。
- `Application`：一个可交付业务系统的目录对象。
- `Service`：服务的稳定身份、运行环境和 owner。
- `ServiceInstance`：服务在某环境/集群/主机上的一个部署实例。
- `TargetRef`：`kind + id + external_id + locator`，指向 Host、K8s Workload 或外部资源。

Service 不保存 Host/K8s 的全部属性，只保存稳定引用和必要的投影字段。引用删除、失效和转移必须有显式状态，不能静默级联。

### 3.3 Change/Execution Domain

```text
ChangeIntent
  -> ExecutionPlan
  -> QueueItem
  -> Attempt
  -> ReconcileObservation
  -> AuditEvent
```

核心字段：

| 对象 | 必须字段 | 约束 |
|---|---|---|
| ChangeIntent | id、kind、request_snapshot、fingerprint、requester、scope、created_at | 创建后不可变；同一幂等键只能指向一个 intent |
| ExecutionPlan | intent_id、target_snapshot、executor、steps、timeout、retry_policy | 固化目标和执行参数，不回读实时表替换历史 |
| QueueItem | plan_id、priority、available_at、lease_owner、lease_until | 租约超时可被安全接管 |
| Attempt | attempt_no、worker、started/finished、stdout/stderr_ref、exit_code | 每次尝试独立记录；重试不覆盖历史 |
| ReconcileObservation | external_id、observed_state、generation、condition、observed_at | 处理重复回调、乱序回调和外部删除 |
| AuditEvent | actor、action、target、before/after、reason、trace_id | 所有高风险动作可追溯 |

执行器接口：

```go
type Executor interface {
    Plan(ctx context.Context, intent ChangeIntent) (ExecutionPlan, error)
    Execute(ctx context.Context, attempt Attempt) (ExecutionResult, error)
    Cancel(ctx context.Context, attempt Attempt) error
    Observe(ctx context.Context, attempt Attempt) (Observation, error)
}
```

首批 provider：SSH、Agent、Kubernetes；后续接入 Ansible、Terraform、Helm、GitOps。Provider 不得直接修改其他模块表，只通过 capability 和事件回写结果。

## 4. Kubernetes 多集群模型

逻辑上 `business/k8s` 保持独立模块，短期不拆物理服务。

```text
Cluster
  -> ClusterCredentialRef
  -> NamespaceBinding
  -> K8sObjectRef
  -> Release / ReleaseRevision
```

- `Cluster` 只保存连接元数据、状态、业务域归属和能力探测结果。
- `ClusterCredentialRef` 指向外部凭据，数据库不保存明文 kubeconfig。
- `NamespaceBinding` 显式绑定 `business_scope_id`、environment、allowed_actions 和 owner。
- `K8sObjectRef` 保存 cluster UID、namespace、kind、name、uid、resourceVersion。
- `Release` 是不可变发布意图；`ReleaseRevision` 保存每次变更的 manifest/values 摘要和来源。

写操作要求：

1. 先验证当前主体的 DataScope、Cluster ownership 和 NamespaceBinding。
2. 使用 `resourceVersion` 或 server-side apply 避免覆盖外部变更。
3. 记录 API intent、外部请求、rollout observation 和最终 condition。
4. Secret 只允许引用、轮换和脱敏展示；禁止在日志、审计和 API 响应回显明文。

## 5. 跨模块通信契约

### 5.1 Capability

CMDB 对 Deploy 暴露最小只读能力：

- `ListSelectableResources`
- `PreviewResourceTargets`
- `ResolveTargetSnapshot`
- `GetResourceOwnership`
- `RecordOperationalSummary`

BizScope 对其他模块暴露：

- `GetActiveScope`
- `CheckScopeAccess`
- `ListVisibleScopes`

Service 对 Deploy/K8s 暴露：

- `GetServiceInstance`
- `ValidateTargetBinding`
- `RecordReleaseRef`

Capability DTO 只含业务所需字段，不包含 GORM Model、`*gorm.DB`、内部 repository 或可变 JSON 原文。

### 5.2 事件

可选的异步事件主题：

```text
resource.ownership.changed
service.instance.changed
change.intent.created
change.attempt.finished
k8s.release.observed
alert.opened
incident.updated
```

V1 可先使用进程内 outbox 表，保证事务提交后再投递；不要求一开始引入 Kafka。

## 6. 数据和安全策略

- 每张业务表拥有明确 owner、软删除策略、审计字段和 migration 版本。
- 所有跨业务域查询带 `DataScopeReq`；默认拒绝，不能用 `nil` 表示“无限范围”。
- 凭据使用引用：Vault/KMS/Secret Manager 负责明文和轮换，ops 只存 provider、version、fingerprint。
- 租约、幂等键、generation、resourceVersion 和唯一约束共同防止重复执行和 stale write。
- 删除采用 tombstone/retire + 引用检查，禁止无审计的级联物理删除。
- 观测数据仅保存标签规范、查询链接和摘要；高基数时序和日志不落业务主库。

## 7. 物理部署演进

### V1：模块化单体

- 一个 API/Worker 部署单元，业务模块独立 package、route、permission、migration。
- MySQL 保存业务状态，Redis 可用于队列/租约，外部系统保存 K8s/SSH/观测事实。
- Worker 与 API 可先同仓库、同镜像，按进程角色启动。

### V2/V3：按伸缩和故障域拆分

- `execution-worker`：长任务、SSH/Agent/Ansible。
- `k8s-provider`：多集群连接池、watch、rollout reconcile。
- `observability-query`：统一查询和标签映射，不复制原始观测数据。
- `notification/incident`：告警、事件和通知路由。

拆分必须以 capability/事件合同为前提，不能通过共享表替代服务边界。

## 8. 迁移策略

1. 先补齐当前模型的 versioned migration 和 checksum gate。
2. 为 Host、ServiceInstance、Release 增加兼容列和双写审计，不立即删除旧列。
3. 建立 ResourceType/Instance/Relation/Ownership 新表，回填现有 Host 和 K8s Cluster。
4. Deploy 从实时目标解析切换为 immutable target snapshot，再启用 Worker。
5. 旧 API 保留兼容期；所有新接口使用稳定 Ref、scope 和 revision。
6. 完成回填校验、双读比对、灰度切换后再清理旧模型。

## 9. 明确不做

- V1 不把所有模块拆成微服务。
- V1 不自建时序数据库、日志平台或 Trace 存储。
- 未解决权限和 migration 前，不引入大量 Operator/CRD 以增加控制面复杂度。
- 不用万能脚本、Cluster Admin 或共享数据库访问替代 capability 和审计。

## 10. 设计验收

- 每个模块可列出自己的表、写入者和公开 capability。
- 跨模块 request path 没有对方 GORM Model、repository 或裸表名引用。
- 同一 ChangeIntent 可在重复请求、Worker 崩溃、回调乱序和租约接管后得到确定结果。
- 任意 K8s mutation 都能证明 Cluster、Namespace、BizScope、主体和审计链路一致。
- 新增观测后只增加引用/标签映射，不要求重写 CMDB 资源身份模型。
