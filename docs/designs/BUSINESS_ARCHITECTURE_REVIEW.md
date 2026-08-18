# pantheon-ops 业务架构评审报告

更新时间：2026-08-18  
类型：Review  
归属层：business  
状态：Active  
评审对象：`pantheon-ops` 当前工作树（HEAD `ee21350`，存在未提交和未跟踪业务实现）

本文将代码审查结论、架构判断和后续治理要求固化为业务仓库的评审基线。本文不是生产就绪声明；“已实现”“已验证”“建议”三种状态必须严格区分。

## 1. 执行摘要

`pantheon-ops` 当前属于**模块化单体形态的运维管理平台，正在向 SRE 平台演进的雏形**。它已经超出简单资产台账：有业务域、服务模型、部署意图/执行记录、Kubernetes 资源管理和数据权限边界。但它还不是成熟企业级 SRE 平台，核心缺口集中在四个方面：

1. 数据库迁移、模型和发布基线没有同时固化，当前工作树不能直接作为可回滚生产版本。
2. 业务域、集群、部署目标的授权边界还存在可绕过路径，尤其是 K8s Cluster Create/Update 的 DataScope 传递。
3. 部署执行仍在 HTTP 请求中同步完成，缺少统一的队列、Worker、重试、取消、租约和调和中心。
4. CMDB 仍是 Host/Group/Label 的轻量台账，尚未形成可支撑数据库、中间件、网络、K8s 对象和关系拓扑的通用资源模型。

结论：项目值得继续投入，也具备企业级 SRE 平台潜力；下一阶段必须先完成 schema、身份边界和异步执行治理，再扩展观测能力。

## 2. 评审范围与证据

### 2.1 代码和文档范围

- `business/cmdb`：Host、Group、LabelSchema 和导入导出能力。
- `business/bizscope`：业务域、DataScope、业务归属。
- `business/service`：Application、Service、ServiceInstance。
- `business/deploy`：Package、Template、Task、Snapshot、幂等键、Lease 和执行结果。
- `business/k8s`：Cluster、Namespace、Workload、ConfigMap、Secret、Release 和 rollout observation。
- `backend/pkg/database/migrations`、路由装配、权限 seed、前端 overlay、业务 smoke。
- `pantheon-base` 的平台、IAM、组织、审计、工作流和继承约束。

### 2.2 已执行验证

| 验证 | 结果 | 解释 |
|---|---|---|
| `go test ./...` | 通过 | 说明当前 Go 单元/包测试可编译并通过，不等于跨模块不变量已覆盖 |
| `go vet ./...` | 通过 | 未发现 vet 级问题 |
| `node --test tests/scripts/check-business-module-boundaries.test.mjs` | 通过 | 业务模块边界 checker 的正反例通过 |
| `go test -race ./...` | 未完成 | Windows 临时磁盘空间不足，链接阶段失败，不能当作 race 通过或失败 |
| MySQL migration 集成测试 | 未执行 | 未设置 `PANTHEON_TEST_DSN` |
| 前端 type-check/build | 未执行 | 需在前端依赖和浏览器环境准备后补跑 |
| 真实 K8s smoke | 未执行 | 没有可用的测试集群和凭据 |

工作树存在大量未提交业务代码、migration、前端模块和 evidence。评审结论以当前工作树为依据，不把它视为已发布基线。

## 3. 项目定位评价

| 定位候选 | 判断 | 原因 |
|---|---|---|
| 工具型平台 | 已超出 | 已有业务域、服务、部署历史、K8s 资源和权限审计，不是单个脚本工具 |
| 运维自动化平台 | 当前最准确 | 现阶段核心价值是资产/业务关联、目标选择、部署执行和 K8s 管理 |
| SRE 平台 | 演进目标，不是当前状态 | 还缺 SLO/SLI、告警、Incident、On-call、Error Budget 和可靠的变更治理 |
| 企业内部 PaaS | 暂不成立 | 尚无完整应用交付、环境编排、供应链、成本和多租户产品化能力 |

**正式定位：**面向企业 SRE 团队的模块化运维管理平台，目标演进为企业级 SRE 平台；V1 不宣称 PaaS 或完整可观测性平台。

## 4. 当前架构评分

| 维度 | 评分 | 评审判断 |
|---|---:|---|
| 架构设计 | 7/10 | Base/ops overlay、业务域平级、K8s 运行态独立等方向正确；schema 和执行中心仍未固化 |
| 模块划分 | 7/10 | CMDB、BizScope、Service、Deploy、K8s 已有合理逻辑边界；跨模块契约仍需统一 registry/facade |
| 可扩展性 | 6/10 | ServiceInstance、Release intent、executor type 为未来扩展留出入口；CMDB 非通用资源模型、凭据和插件协议不足 |
| SRE 理念符合度 | 6/10 | 有幂等、快照、租约、reconcile 等工程意识；同步 SSH、缺 SLO/Incident/变更窗口是明显缺口 |
| 企业应用价值 | 6.5/10 | 能覆盖中小企业资产到部署闭环；生产级多租户、灾备、审计证据和可观测治理尚未完成 |

评分不是产品质量分，而是当前架构距离企业级 SRE 平台的成熟度估计。

## 5. 事实源与模块边界结论

CMDB **不应成为所有域的总数据库**，应成为资源目录和关系索引；每个域保留自己的 Source of Truth：

| 领域 | Source of Truth | CMDB/其他域可读取内容 |
|---|---|---|
| 资源目录 | CMDB：资源身份、生命周期、标签、所有权引用、关系摘要 | 稳定 ResourceRef、摘要、发现时间 |
| 业务目录 | BizScope/Application/Service：业务、服务、责任人、环境 | 资源引用和责任归属 |
| 部署变更 | Deploy/Release：ChangeIntent、版本、执行计划、Attempt、结果 | 目标快照和最后一次部署摘要 |
| K8s 运行态 | Kubernetes API：Namespace、Workload、Service、Config、Release 状态 | ClusterRef、对象引用、观测摘要 |
| 可观测事实 | Prometheus/VictoriaMetrics、Loki/ES、OTel | 查询链接、指标摘要、健康状态 |
| 告警治理 | Alert/Incident/On-call 域 | 当前事件、责任和处置状态 |

跨模块只能通过显式 capability/facade、稳定 DTO 和事件/引用通信；禁止 request path 直读对方表或持有对方 GORM Model。

## 6. P0：必须解决

### P0-1 Migration、模型和发布基线不一致

证据：`backend/pkg/contracts/module.go` 的 versioned migration 选择逻辑；`backend/pkg/database/migrations/000012-000014`；`business/deploy` 的 `Release` 模型；`business/service` 的 `ServiceInstance` 模型。

- Base 默认 versioned migration 模式可能跳过模块 AutoMigrate。
- `Release` 模型包含 `idempotency_key`、request snapshot/fingerprint、attempt、generation/revision、replica counters、condition、`last_reconciled_at` 等字段，但当前 `000012_business_schema.up.sql` 未完整创建。
- `ServiceInstance` 的 rollback/state/health/transition 字段未在 `000014_service_foundation.up.sql` 完整落库。
- migration 文件仍未纳入稳定提交基线，存在“代码已合并、数据库未升级”的发布遗漏风险。

**验收门槛：**空库、当前升级库、重复执行、失败恢复和回滚均在 MySQL 8 通过；应用启动不依赖不可追踪的 AutoMigrate；migration、模型、seed、fixture 和文档一致。

### P0-2 K8s Cluster Create/Update 的 DataScope 断裂

证据：`backend/modules/business/k8s/cluster_service.go:117-192`、`:402-410`，`cluster_handler.go:62-75`。

Handler 只传部门 ID，Service 创建/更新时通过 `GetActive(..., nil)` 获取业务域，可能造成跨业务域注册或重新归属集群。该问题同时影响租户边界、审计可信度和后续 K8s 对象访问。

**验收门槛：**所有 Create/Update/Get/List/Sync 路径都接受并强制应用 `DataScopeReq`；未授权、非 active、跨部门业务域在 mutation 前失败；并发变更有审计证据。

### P0-3 当前工作树不能作为生产发布基线

upstream 分支已删除，工作树包含约 147 条修改/新增，K8s、Service、migration、evidence 等大量内容未跟踪。没有可复现 commit、migration 版本、构建产物和回滚说明，任何生产试用都无法证明“运行的版本”和“可回滚版本”一致。

**验收门槛：**形成干净提交、固定 base foundation 版本、生成 migration checksum/升级报告、标注已验证命令和剩余 gap；发布包可从 clean checkout 重建。

## 7. P1：建议在 V1 关闭

1. **部署目标信任不足**：`deploy_service.go:375-424` 只做通用目标解析，未强制校验 `ServiceID`、`ServiceInstanceID`、BizScope 与 target 一致；K8s Release `normalizeCreateRequest`（约 `release_service.go:681-707`）同样需要 ServiceInstance 绑定校验。
2. **HTTP 同步执行**：`deploy_service.go:878-997`、`:1781-1801` 在请求内按 Host 顺序执行 SSH；容易超时、重复提交、连接泄漏，无法稳定取消。
3. **没有统一任务中心**：缺 queue/worker/executor/retry/timeout/cancel/reconcile 的统一语义；`CancelTask` 只改数据库状态，不能保证中断正在运行的 SSH。
4. **K8s Namespace 无业务归属绑定**：设计文档明确暂不实现 ownership，导致业务域隔离和资源授权无法闭环。
5. **K8s 权限 seed 不完整**：已有 cluster/workload/release 权限，但 namespace、configmap、secret、scale、restart、logs、sync 等动作未按风险细分。
6. **CMDB 仍是轻量 Host 台账**：缺 `ResourceType`、`ResourceInstance`、`ResourceAttribute`、`ResourceRelation`、`ResourceOwnership`、`DiscoverySource`、`LifecycleHistory`、`CredentialReference` 等通用对象。
7. **JSON 查询会成为规模瓶颈**：Host labels、Group conditions 使用 JSON，规模增长后反向关系、索引和条件查询成本高。
8. **分页和查询效率不足**：`ListTasks` 全量加载后内存过滤分页；K8s 多个 List API 无分页。
9. **凭据治理不足**：kubeconfig 虽 AES-GCM 加密，但密钥来自单一环境变量，缺 KMS/Vault、轮换和版本化；SSH 凭据仍通过请求 DTO 传入，应演进为凭据引用、脱敏和专门审计。

## 8. P2：长期规划

- SLO/SLI、Error Budget、告警路由、Incident、On-call、Postmortem。
- GitOps/ArgoCD/Kustomize/Helm provider 和渐进式发布（canary/blue-green）。
- Agent/Ansible/Terraform/Helm/GitOps 统一 Provider/Plugin SDK。
- 资源发现、依赖拓扑、容量与成本、供应链安全、灾备和多区域。
- 从模块化单体到可选物理拆分；只有队列、观测或 K8s provider 需要独立伸缩时再拆服务。

## 9. 模块评审结论

### 9.1 CMDB

**当前问题：**实际能力以 Host/Group/LabelSchema 为中心，能够做主机台账和部署目标选择，但不能作为数据库、中间件、网络、集群和应用关系的统一资源底座；JSON 标签和条件在规模上存在查询限制。

**推荐设计：**保留 Host 兼容模型，增加通用资源内核：

```text
ResourceType
  -> ResourceInstance
  -> ResourceAttribute
  -> ResourceRelation
  -> ResourceOwnership
  -> ResourceObservation
```

资源类型和实例负责身份与生命周期；属性负责扩展字段；关系负责 depends_on/runs_on/exposes/managed_by；所有权负责 BizScope/Service/Department；观察值只保存摘要和来源，不复制时序事实。

### 9.2 业务域与 Service

**当前问题：**BizScope 独立于 CMDB 的方向正确，但 Namespace ownership、ServiceInstance target 一致性和责任人/环境模型尚不完整。

**推荐设计：**`BizScope -> Application -> Service -> ServiceInstance -> TargetRef`。`TargetRef` 使用稳定引用（Host、K8s Workload、外部资源），Service 不直接嵌入资产字段；业务域只持有引用和策略，不拥有 CMDB/K8s 内部表。

### 9.3 Deploy

**当前问题：**已有 snapshot、idempotency、lease、状态机和结果回写，是正确的可靠性起点；但同步 SSH 和缺统一任务中心会限制生产规模。

**推荐设计：**

```text
ChangeIntent -> ExecutionPlan -> Queue -> Worker -> Attempt -> Reconcile -> Audit/Event
```

Executor 统一抽象为 `SSH / Agent / Ansible / Terraform / Helm / GitOps`。HTTP API 只创建 intent、查询状态、请求取消；Worker 负责执行，Attempt 负责每次尝试，Reconcile 负责外部状态收敛。

### 9.4 Kubernetes

**当前问题：**K8s API 作为运行态真相源、Release 使用不可变 intent、rollout observation/reconcile 的方向正确；但多集群租户边界、Namespace ownership、细粒度高风险权限、分页、resourceVersion/server-side apply 尚未完整。

**推荐设计：**逻辑上保持独立业务模块，短期不拆微服务。引入 `Cluster`、`ClusterCredentialRef`、`NamespaceBinding`、`K8sObjectRef`、`Release`、`ReleaseRevision`；所有写操作带 DataScope、resourceVersion 和审计；Secret 只写引用和脱敏摘要。

## 10. 业界借鉴与避免的坑

| 系统/实践 | 借鉴 | 不照搬 |
|---|---|---|
| Backstage | Service Catalog、Owner、系统/组件目录、插件模型 | 不把 Catalog 做成所有运行态事实的数据库 |
| Rancher/KubeSphere | Cluster/Project/Namespace 多集群和租户边界 | 不用宽泛 Cluster Admin 代替细粒度授权 |
| Spug | 轻量任务体验、脚本与目标选择 | 不在 HTTP 请求内执行长时间 shell |
| OpsAny/腾讯蓝鲸 | 作业编排、审批、审计、插件执行器 | 不引入巨型低代码单体和不可审计的万能脚本 |
| 夜莺 | 标签、责任绑定、告警路由 | 不把监控时序复制进 CMDB |
| Netflix SRE | SLO、Error Budget、不可变发布、渐进交付 | 不先堆平台组件而没有服务责任和可靠性指标 |

## 11. SRE 工程能力成熟度

当前已具备：资源和业务目录雏形、部署快照/幂等/租约、K8s Release 状态观察、IAM/审计基础、业务边界 checker。

当前缺少：完整生命周期（发现-变更-运行-退役）、统一任务调度、审批/变更窗口、凭据托管、环境模型、发布策略、SLO/告警/Incident/On-call、供应链和灾备。成熟度判断约为“运维自动化平台早期”，不是“企业级 SRE 平台”。

## 12. 演进路线

### V1：可发布的资源到交付闭环

- 修复 migration/schema、Cluster DataScope、ServiceInstance target 校验。
- CMDB + BizScope + Service + Deploy + K8s 形成最小闭环。
- Deploy Worker 化，补 retry/timeout/cancel/reconcile。
- Namespace ownership、K8s 细粒度权限、分页和 secret 引用。
- 固化 clean release、MySQL migration gate、跨模块 smoke 和回滚证据。

### V2：运行态可见性

- Metrics、Logs、基础 Alert。
- 变更窗口、审批、通知和任务审计。
- ServiceInstance 与 Prometheus/Loki 标签关联。

### V3：完整可观测性与可靠性治理

- OpenTelemetry Metrics/Logs/Traces、APM。
- SLO/SLI、Error Budget、Incident、On-call、Postmortem。
- GitOps/ArgoCD/Helm provider、渐进式发布和回滚。

### V4：企业级 SRE 平台

- 企业 Service Catalog、多租户、多环境、多集群。
- 资源发现、依赖拓扑、容量/成本、供应链安全和灾备。
- Provider/Plugin SDK，支持组织内标准化运维能力复用。

## 13. 最终判断

1. **是否值得继续投入：是。**业务边界和闭环方向正确，已经有可复用的可靠性基础。
2. **是否具备企业级潜力：有，但尚未达到。**潜力来自模块化边界、Service 模型和 K8s/Deploy 设计，不是来自当前功能数量。
3. **最大风险：**schema、身份边界、事实源和发布基线没有同时固化，可能造成跨域越权、数据漂移和不可回滚发布。
4. **下一阶段最应投入：**P0 migration/schema gate、DataScope/ownership、统一 Change/Execution 模型和异步 Worker；完成后再扩展观测平台。

详细执行顺序见：

- [BUSINESS_TARGET_ARCHITECTURE_DESIGN.md](./BUSINESS_TARGET_ARCHITECTURE_DESIGN.md)
- [BUSINESS_DEVELOPMENT_PLAN.md](./BUSINESS_DEVELOPMENT_PLAN.md)
- [BUSINESS_QA_ACCEPTANCE_PLAN.md](./BUSINESS_QA_ACCEPTANCE_PLAN.md)
- [BUSINESS_TEST_PLAN.md](./BUSINESS_TEST_PLAN.md)
