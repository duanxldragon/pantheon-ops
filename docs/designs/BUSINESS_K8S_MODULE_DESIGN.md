# Kubernetes 业务模块设计

English version: [BUSINESS_K8S_MODULE_DESIGN.en.md](./BUSINESS_K8S_MODULE_DESIGN.en.md)

- 更新时间：2026-08-24
- 类型：Design
- 归属层：`business/k8s`
- 状态：Active（Service CRUD 和外部运行矩阵为 V1 冻结目标）

## 1. 目标与事实源

`business/k8s` 在明确的 Cluster ownership、NamespaceBinding、权限和并发控制下管理多集群运行态与发布。

- `Cluster` 是已登记集群的稳定业务身份，保存编码、环境、BizScope/department 归属、受保护 credential metadata 和同步摘要。
- Kubernetes API 是 Namespace、Node、Deployment、StatefulSet、DaemonSet、Service、ConfigMap、Secret 和 Pod 运行态的唯一事实源。
- `NamespaceBinding` 是 namespace mutation 的业务授权边界，保存 BizScope、environment 和 allowed actions。
- `Release` 是不可变发布意图与观测记录，不是 Workload 副本。
- Service Catalog 只引用 cluster/namespace/workload，不复制运行态对象。

## 2. 跨模块边界

- Cluster 通过 `BizScopeReader.GetActive` 验证业务域，不直读 BizScope 表。
- Service 通过 `K8sTargetReader` 验证 target，不直接依赖 Cluster/Namespace service。
- Cluster 删除通过 `ReferenceChecker` 检查 Release 和 ServiceInstance 引用。
- 所有 Kubernetes 调用先验证 Token、Casbin、DataScope、Cluster ownership 和 NamespaceBinding。
- VM Deploy 与 K8s Release 共享 Service 身份和审计语义，但不共享执行器或任务表。

## 3. 功能范围

| 能力      | 当前基线                                                               | V1 要求                                                                          |
| --------- | ---------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| Cluster   | list/get/create/update/delete、sync、nodes、credential metadata/rotate | 并发安全 ClientPool、凭据不回显、外部连接证据                                    |
| Namespace | list/create/delete + NamespaceBinding                                  | ownership/action 校验，Delete 使用 resourceVersion                               |
| Workload  | list、pods、logs、scale、restart                                       | scale/restart stale write 返回 409                                               |
| ConfigMap | 结构化读取/写入                                                        | scope、audit、resourceVersion 一致                                               |
| Secret    | 结构化管理                                                             | API/日志/审计永不返回 value                                                      |
| Release   | create、rollback、reconcile、rollout 观测                              | 幂等、不可变 intent、超时后只 reconcile                                          |
| Service   | 尚未闭环                                                               | list/get/create/update/delete，结构化 type/selectors/ports，强制 resourceVersion |

## 4. Cluster 与凭据

- kubeconfig 通过受保护写通道存储；读接口只返回 provider、version、fingerprint、updatedAt 等 metadata。
- credential rotate 创建新版本并使新连接使用新版本；旧 client entry 安全淘汰。
- 删除 Cluster 前检查 Release、ServiceInstance 和其他 owner 引用，并记录审计。
- Sync 有明确超时，不把 API timeout 写成集群不存在。

### 4.1 ClientPool 并发合同

1. 不得在持有 `RLock` 时修改 `lastUsed`。
2. entry replace、credential rotation 和 cleanup 使用一致锁策略。
3. kubeconfig parse、client build 和网络调用不持有全局写锁。
4. 同 cluster 并发初始化使用 double-check insertion 或 singleflight 等价协调。
5. 测试覆盖同集群、不同集群、rotation、cleanup、expired entry 和 race。

## 5. NamespaceBinding

NamespaceBinding 至少包含：`cluster_id`、`namespace`、`business_scope_id`、`environment`、`allowed_actions` 和审计字段。

- Create Namespace 先创建/验证 binding，再调用 Kubernetes；外部失败时清理未生效 binding。
- 每个 namespace mutation 调用 `RequireWrite(cluster, namespace, action)`。
- 不存在、inactive、cross-scope 或 action 未授权的 binding 默认拒绝。
- 删除 Namespace 要求 `resourceVersion`，并检查 ServiceInstance/Release 引用和资源清理影响。

## 6. Workload 与 Pod

- Workload list 支持 cluster、namespace、kind、keyword 等筛选。
- 详情上下文显示 replicas、ready、image、generation、resourceVersion、conditions 和 Pod 摘要。
- Scale 只允许 Deployment/StatefulSet，并要求 expected resourceVersion。
- Restart 在 apply 前验证 expected resourceVersion，记录目标和审计。
- Pod logs 只在授权 namespace 内读取，设置大小/时间限制并执行统一脱敏。
- API timeout、forbidden、not found、conflict 和 rollout failure 使用不同错误 key。

## 7. ConfigMap、Secret 与 Service

### ConfigMap

- 使用结构化 key/value API，不接受任意 shell 或未验证 YAML。
- Create/Update/Delete 验证 NamespaceBinding 和 resourceVersion。

### Secret

- Create/Update 接受受保护输入；list/get 只返回 name、type、keys、resourceVersion 和 metadata。
- Secret value 不进入响应、错误、日志、审计、导出或 evidence。

### Service（V1 冻结目标）

- 路径位于 `/api/v1/business/k8s`，按 cluster + namespace 定位。
- list/get/create/update/delete `corev1.Service`。
- Create/Update 只接受结构化 `type`、`selectors` 和 `ports`；不接收任意 YAML 字符串。
- Update/Delete 必须携带 `resourceVersion`；stale write 返回 HTTP 409 与稳定错误 key。
- Cluster、Namespace ownership 和 DataScope 检查是 mutation 前置条件。
- 每个 mutation 有独立 action permission、审计 metadata、loading/error/conflict UI 状态。

## 8. Release 状态机

```text
pending -> applying -> succeeded
                    -> failed
                    -> timed_out -> succeeded | failed
```

1. 在 Cluster 锁内提交不可变 `pending` intent、请求快照、操作者、目标和幂等键。
2. 条件更新为 `applying`，读取并持久化变更前观测。
3. 调用 Kubernetes Update，保存 target generation/revision。
4. 有界观察 rollout；单次 Update 成功不代表 Release 成功。
5. 终态使用条件转换；重复收口幂等，冲突终态显式失败。
6. `timed_out` 或结果不确定时只做 Reconcile，不盲目重复 mutation。

同一 `idempotencyKey` + 同一请求快照返回既有 Release；相同键配不同快照返回 409。

## 9. UI 与交互

- Cluster 列表/详情展示业务域、环境、连接状态、credential metadata、节点和 namespace 上下文。
- Workload 页支持筛选、Pod 查看、日志、scale 和 restart；动作只在 permission + binding + state 满足时出现。
- Release 页显示 immutable target、请求 revision、状态、rollout 副本、最后 condition、错误摘要和 reconcile。
- Service 管理放在 cluster/namespace 上下文内，避免新增孤立一级导航。
- 409 冲突显示“资源已变化”，保留用户输入并提供刷新后重试，不静默覆盖。
- Secret 页面不提供 reveal/copy value 行为。
- loading、empty、error、forbidden、conflict、timeout、partial cleanup 都是独立状态。

## 10. 权限与审计

权限按 Cluster、Namespace、Workload、ConfigMap、Secret、Service、Release 的 view/create/update/delete/scale/restart/logs/rollback/reconcile 分离。

所有 mutation 审计至少记录 actor、action、cluster、namespace、kind/name、business scope、request ID、resourceVersion、before/after 摘要和 result。秘密数据只保存 redacted summary。

## 11. 外部运行验收

在隔离且明确授权的测试集群执行：

1. 登记 Cluster 并验证 kubeconfig 不回显。
2. 创建带 task ID 后缀的 Namespace。
3. 创建/更新/删除 Nginx Deployment + Service。
4. 创建/更新/删除 Redis Deployment + Service，并从临时 Pod 验证 DNS/TCP。
5. 验证 forbidden RBAC、DataScope denial、stale resourceVersion 和 API timeout。
6. 删除所有测试资源并记录 cleanup。

证据必须记录命令/API、对象、actor、时间、响应、耗时和清理结果；禁止使用生产 namespace。

## 12. 非目标

- V1 不建立 Kubernetes 对象到 CMDB 的全量同步。
- V1 不实现 GitOps、Helm controller、跨集群调度或 Cluster Admin 默认授权。
- V1 不把 K8s Release 合并进 VM Deploy 执行器。
- P1 前不建立通用 Resource/K8s fact copy。

## 13. 验收

- ClientPool 并发/race 测试通过，无 `RLock` 写入或慢操作持全局写锁。
- Cluster、Namespace、Workload、ConfigMap、Secret、Release 的权限/DataScope/ownership 负向用例通过。
- Service 结构化 CRUD、409 和 secret non-disclosure 通过。
- Release 幂等、timeout/reconcile、rollback 和引用删除竞态结果确定。
- 外部 Nginx/Redis matrix、错误矩阵和 cleanup evidence 完整。
- 菜单、权限、i18n、migration、frontend build 和 V1 冻结门禁通过。

发布状态以 [V1 冻结检查表](../V1_RELEASE_CHECKLIST.md) 为准。
