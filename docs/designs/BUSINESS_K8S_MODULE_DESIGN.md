# Kubernetes 业务模块设计

本文件描述 `business/k8s` 的业务域边界和当前交付契约。平台认证、Casbin、
DataScope、审计、菜单、i18n 与密钥保护仍由 `pantheon-base` 提供，本模块不复制
这些系统域能力。

## 1. 所有权与真相源

- `Cluster` 是已登记 Kubernetes 集群的稳定业务身份，保存集群编码、环境、业务域
  归属、加密 kubeconfig 和必要的同步摘要。
- Kubernetes API 是 Namespace、Node、Deployment、StatefulSet、DaemonSet、
  ConfigMap 与 Secret 运行态的唯一真相源。它们只以实时 DTO 返回，不复制到
  CMDB 或本地缓存表。
- `Release` 是镜像变更的不可变意图和观测记录，不是 Kubernetes Workload 的副本。
- CMDB 不拥有 K8s 运行态；VM Deploy 与 K8s Release 只共享将来的 Service 身份和
  审计语义，不共享执行器。

## 2. 跨模块边界

- K8s Cluster 读取 BizScope 时只依赖 `business/capability.BizScopeReader`，
  通过 `GetActive` 验证归属业务域；不得直接查询 BizScope 表或 GORM Model。
- 同进程调用使用 typed Go interface。HTTP/gRPC 只留给真实的部署边界。
- Cluster 删除使用 K8s owner 的 `ReferenceChecker` 检查 Release 引用，并与
  Release intent 创建使用同一 Cluster 行锁，避免删除与创建竞态产生孤儿历史。

## 3. Release 状态机

Release 的 canonical 状态为：

```text
pending -> applying -> succeeded
                    -> failed
                    -> timed_out -> succeeded | failed
```

- `pending`：不可变请求快照、操作者、目标与幂等键已经在数据库提交，尚未调用
  Kubernetes。
- `applying`：执行器已获得条件转换，正在读取、更新或观测目标 Workload。
- `succeeded`：目标 image、接受的 generation，以及 Deployment / StatefulSet /
  DaemonSet 的 rollout 条件均满足。
- `failed`：apply、读取或 rollout 已失败，并保存错误与最近观测。
- `timed_out`：有界观测窗口结束；保留为可恢复状态，只能由 Reconcile 继续观察，
  不会盲目再次 apply。

同一 `idempotencyKey` 且同一请求快照返回既有 Release；相同键配不同快照返回冲突。
未提供键时服务端生成单次键，兼容旧调用但不提供跨请求重试去重。

## 4. 执行与恢复

1. 在 Cluster 锁内提交 `pending` Release intent。
2. 以条件更新转为 `applying`，读取并持久化变更前的 Workload 观测。
3. 调用 Kubernetes Update，持久化接受后的 target generation / revision。
4. 以有界轮询观察 rollout。单次 Update 成功不代表 Release 成功。
5. 结果写入使用条件状态转换；同一并发终态收口是幂等的，冲突终态显式失败。
6. apply 超时或结果写入不确定时只做 `Reconcile` 观察，绝不盲目重复 mutation。

环境变量 `PANTHEON_K8S_RELEASE_TIMEOUT` 与
`PANTHEON_K8S_RELEASE_POLL_INTERVAL` 仅接受正值且分别限制在十分钟和三十秒内。

## 5. 授权、审计与 UI

- 所有 K8s API 路由位于 Base 的 Token、Casbin 和 DataScope 中间件之后；Cluster 与
  Release 查询均按 DataScope 过滤。
- Release create、rollback、reconcile 有独立业务权限和审计 action key。前端按钮
  仅改善体验，后端 Casbin 仍是实际执行门禁。
- Release 页展示 immutable target、状态、rollout 副本数、最后条件和错误摘要；
  create 与 rollback 请求发送稳定的幂等键；`timed_out` 和 `applying` 可执行
  Reconcile。
- kubeconfig 和 Secret 值绝不出现在 Release DTO、审计快照或页面中。

## 6. 明确非目标与已知升级边界

- 不建立 Kubernetes 对象到 CMDB 的全量同步。
- 不实现 Namespace 到 BizScope 映射；共享集群的 Namespace 归属需单独合同。
- 不实现 GitOps、Helm controller、跨集群调度或将 K8s Release 合并到 VM Deploy。
- 现有生产表的 legacy AutoMigrate 升级转换由维护者明确延后到统一
  model-generated migration 工作；新字段依赖该后续迁移才能在既有生产库中启用。
