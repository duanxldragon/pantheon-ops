# pantheon-ops 业务开发计划

更新时间：2026-08-18  
类型：Plan  
归属层：business  
状态：Proposed  
基线：[BUSINESS_ARCHITECTURE_REVIEW.md](./BUSINESS_ARCHITECTURE_REVIEW.md)  
目标设计：[BUSINESS_TARGET_ARCHITECTURE_DESIGN.md](./BUSINESS_TARGET_ARCHITECTURE_DESIGN.md)

本文把架构评审结论转换为可排期的业务开发计划。时间按连续 8 个开发周估算，实际开始日期以 clean release 和维护者质量门禁决定为准；角色是责任边界，不预设具体人员。

## 1. 目标和范围

### 1.1 V1 目标

在一个可复现、可回滚的 release 中交付：

```text
CMDB Resource/Host -> BizScope -> ServiceInstance -> Deploy Change -> K8s/SSH execution
```

V1 必须具备：schema 一致性、DataScope/ownership、一致的目标快照、异步 Worker、幂等/租约/取消、细粒度权限、审计和跨模块 smoke。

### 1.2 V1 不包含

- 完整 Metrics/Logs/Tracing/APM 平台。
- 自建 Kafka、时序库、日志库或复杂微服务拆分。
- 大规模通用低代码编排。
- 未经权限和回滚验证的 Operator/CRD 扩张。

## 2. 工作流和优先级

| 阶段 | 周期 | 优先级 | 主要责任角色 | 依赖 |
|---|---:|---|---|---|
| A. 发布基线和 schema | 第 1 周 | P0 | 后端/数据库/发布维护 | 无 |
| B. 身份边界和 capability | 第 2 周 | P0 | 后端/安全/业务域 owner | A |
| C. 目标快照和 Service 绑定 | 第 3 周 | P0/P1 | Service/Deploy owner | B |
| D. Execution Center Worker | 第 4-5 周 | P1 | Deploy/运行时 owner | C |
| E. K8s ownership 与权限 | 第 5-6 周 | P1 | K8s/安全 owner | B、C |
| F. V1 全链路验收和 release | 第 7 周 | P0 | QA/发布/所有模块 owner | A-E |
| G. V2 观测准备 | 第 8 周 | P2 | 观测/平台 owner | F |

优先级规则：P0 未关闭时停止扩展功能；P1 未关闭时不得宣称生产级；P2 只在 V1 质量门禁通过后排期。

## 3. 详细任务清单

### A. 发布基线和 schema（P0）

| ID | 工作项 | 交付物 | 完成标准 |
|---|---|---|---|
| A-01 | 对齐 `Release`、`ServiceInstance`、Task、K8s 模型与 up/down migration | migration 000012-000014 修订、模型差异表 | 空库/升级库/重复执行一致，无 unknown column |
| A-02 | 固定 migration runner 和 checksum gate | migration CI 命令、失败恢复报告 | 不依赖未追踪 AutoMigrate |
| A-03 | 清理未提交业务资产并固定 base foundation 版本 | clean commit、发布 manifest、回滚说明 | clean checkout 可重建 backend/frontend |
| A-04 | 对 schema、seed、permission、menu、i18n 建立版本清单 | V1 release checklist | 每项都有 owner 和证据路径 |

### B. 身份边界和 capability（P0）

| ID | 工作项 | 交付物 | 完成标准 |
|---|---|---|---|
| B-01 | 修复 K8s Cluster Create/Update DataScope 传递 | handler/service 改造及测试 | nil scope 不再代表无限可见 |
| B-02 | 收敛 BizScope/CMDB/Service/K8s capability DTO | contract 定义和边界 checker allowlist | request path 无跨 owner 表/Model |
| B-03 | 强制 ServiceInstance target 与 BizScope/Cluster/Namespace 一致 | capability 校验和拒绝用例 | mutation 前拒绝跨域、inactive、失效引用 |
| B-04 | 设计 NamespaceBinding 和高风险权限矩阵 | 权限 seed、菜单、审计矩阵 | secret/restart/scale/sync 等动作可分别授权 |

### C. 目标快照和 Service 绑定（P0/P1）

| ID | 工作项 | 交付物 | 完成标准 |
|---|---|---|---|
| C-01 | 统一 `TargetRef` 与 target snapshot | DTO、数据库字段、快照 fingerprint | 执行期间 CMDB 变化不改历史目标 |
| C-02 | Deploy 创建时强校验 ServiceID/ServiceInstanceID | service capability、错误 key、测试 | 服务、实例、资源、业务域四者一致 |
| C-03 | K8s Release 固化 manifest/values/revision | release schema 和 reconcile 状态 | 重复请求、乱序回调结果确定 |
| C-04 | 迁移 SSH/kubeconfig 到 CredentialRef | 外部凭据接口、脱敏审计 | API/日志/数据库不保存明文 |

### D. Execution Center Worker（P1）

| ID | 工作项 | 交付物 | 完成标准 |
|---|---|---|---|
| D-01 | 抽象 Executor/provider | SSH、Agent、K8s provider interface | provider 可替换，业务服务不直接执行 shell |
| D-02 | 实现 queue/worker/lease/attempt | Worker 进程角色、队列表或 Redis 适配器 | API 创建任务不等待远程执行 |
| D-03 | 实现 timeout/retry/cancel | 状态机、context 取消、Attempt 历史 | 取消能通知执行器，超时有确定终态 |
| D-04 | 实现 Reconcile 和 outbox/audit | reconcile loop、事件、审计链 | Worker 崩溃和重复回调可恢复 |
| D-05 | 增加查询分页和索引 | SQL 分页、慢查询基线 | 任务/K8s List 不全量加载 |

### E. K8s ownership 与权限（P1）

| ID | 工作项 | 交付物 | 完成标准 |
|---|---|---|---|
| E-01 | Namespace 与 BizScope/Environment 显式绑定 | NamespaceBinding 表/API/UI | 无绑定的写操作全部拒绝 |
| E-02 | ClusterCredentialRef 和密钥轮换 | provider 接口、版本字段 | 明文 kubeconfig 不进入业务库 |
| E-03 | resourceVersion/server-side apply | mutation helper、冲突错误 | stale write 不覆盖外部变更 |
| E-04 | 补齐 Workload/ConfigMap/Secret/scale/restart/logs/sync 权限 | seed、契约、smoke | 每个动作有 view/list/create/update/delete/operate 语义 |

### F. 验收和发布（P0）

| ID | 工作项 | 交付物 | 完成标准 |
|---|---|---|---|
| F-01 | 执行 QA 验收问题集 | QA 记录、失败项、回归结果 | P0 全部关闭，P1 有明确延期原因 |
| F-02 | 执行测试计划和业务 smoke | `.harness/evidence/<task-id>/summary.md` | backend/frontend/migration/smoke 证据完整 |
| F-03 | 生成 clean release 和 rollback rehearsal | 发布包、回滚演练记录 | 从 clean checkout 可复现构建和回滚 |

### G. V2 观测准备（P2）

| ID | 工作项 | 交付物 | 完成标准 |
|---|---|---|---|
| G-01 | 统一 ServiceInstance 观测标签 | label mapping 设计 | Prometheus/Loki/OTel 可按 service/env/instance 查询 |
| G-02 | 定义首批 SLI/SLO 数据合同 | SLO 草案、指标命名 | 不把时序数据写入 CMDB |
| G-03 | Alert/Incident/On-call 边界设计 | 业务观测模块 design | 与 base workflow/通知能力无重复实现 |

## 4. Definition of Ready / Done

### Ready

- 明确所属 business 模块和 owner。
- 有 capability/API、数据模型、状态机、权限和审计说明。
- 已检查是否应回 `pantheon-base`，没有在 ops 复制平台能力。
- 有正向、越权、重复、并发、失败恢复和回滚测试场景。

### Done

- 代码、migration、seed、菜单、权限、i18n、fixture、smoke 和文档同步。
- `go test ./...`、`go vet ./...`、边界 checker 通过；适用时完成 race、MySQL、前端和 K8s 验证。
- 高风险 mutation 有审计和 DataScope 证据。
- `.harness/evidence/<task-id>/` 有 commands、summary、review；剩余 gap 明确列出。
- 干净工作树可重建，发布版本和回滚步骤可复现。

## 5. 依赖和停点

- 发现平台层、IAM、组织、通用工作流或 UI 规范问题：停止本地复制，创建 base-first 任务。
- migration 与模型无法对齐：停止新增功能，先修 P0-1。
- DataScope/ownership 不能在 mutation 前校验：禁止进行生产试用。
- Worker 无法可靠取消或恢复：保留模拟执行，不宣称真实执行生产就绪。
- 缺少 MySQL、凭据或 K8s 外部环境：记录验证 gap，不用单元测试替代外部系统验收。

## 6. 后续迭代路线

### V1（本计划）

关闭 P0，交付 CMDB + BizScope + Service + Deploy + K8s 的可回滚最小闭环。

### V2

Metrics、Logs、基础 Alert；变更窗口、审批、通知；ServiceInstance 观测标签。

### V3

OTel 全链路、SLO/Error Budget、Incident/On-call/Postmortem、GitOps/渐进式发布。

### V4

企业 Service Catalog、多租户、多环境、多集群、容量/成本、供应链安全、灾备和 Provider SDK。
