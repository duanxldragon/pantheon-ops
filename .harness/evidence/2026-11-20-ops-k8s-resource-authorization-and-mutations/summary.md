# T06 K8s 资源读写授权与变更准入 — 实施摘要

- Task: `2026-11-20-ops-k8s-resource-authorization-and-mutations`
- 日期: 2026-11-24
- 状态: **代码级全部验收项已实现并验证；浏览器侧草稿恢复与 k8s smoke 仍为显式 gap**
- 交付层级: L2（自审见 `review.md`）

## 已交付

| 验收项                          | 状态             | 证据                                                                                                                                                                                                                                                                                              |
| ------------------------------- | ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| DaemonSet 不修改同名 Deployment | 已修复           | `scaleWorkload`/`restartWorkload` 改为按规范化 kind 分派；`TestRestartWorkloadPatchesNamedKindOnly`（fake clientset 同时存在同名 Deployment 与 DaemonSet）                                                                                                                                        |
| 未知 kind 不放行                | 已修复           | `ScaleWithResourceVersion`/`RestartWithResourceVersion` 拒绝 `normalizeKind==""`；`TestScaleWorkloadRejectsDaemonSetAndUnknownKind`（daemonset/DaemonSet/replicaset）                                                                                                                             |
| 0 副本可提交                    | 已由既有实现满足 | `ScaleRequest.Replicas *int32` + `min=0`，指针区分 0 与缺省                                                                                                                                                                                                                                       |
| 冲突可恢复（服务端）            | 已实现           | `common.ErrConflict` → HTTP 409（`codeConflict`），调用方可用新 resourceVersion 重试                                                                                                                                                                                                              |
| 未授权 namespace 查询/变更拒绝  | 已实现（服务端） | `NamespaceService.ReadableNamespaces` 按 Cluster 的 `business_scope_id`+`environment` 过滤绑定；NamespaceAll 聚合读在 workload/pod/configmap/secret/service/ingress/storage PVC 全部按可见集过滤，显式 namespace 读先过 `RequireReadForClusterWithContext`；写路径保持 `WithWrite`/`RequireWrite` |
| 读路由权限绑定                  | 已实现           | `business:k8s` 注册 cluster/namespace/workload/configmap/secret/service/release/ingress/storage/backup 的读与动作路由；无权限键归属的路由（集群凭据、Helm、install）保持未绑定 → 默认拒绝                                                                                                         |
| 界面保留草稿/可恢复             | 未验证           | 浏览器侧未做真实 resourceVersion 冲突演练                                                                                                                                                                                                                                                         |

## 验证

- workload 包 3/3 定向 PASS；整包 ok
- K8s 全模块 12/12 包 ok（真实 MySQL DSN），含新增 `TestReadableNamespacesFollowsClusterOwnership` 与 `TestK8sPermissionRoutesMatchActualActions`
- permission / role / tenant 套件（真实 MySQL）复跑 PASS，消费扩展后的 K8s 权限路由注册表

## 显式 gap

1. 真实 kubectl 干扰 + UI 草稿恢复未在浏览器验证。
2. k8s 浏览器 smoke 未执行（前端 dev server 被内存回收）。
3. Base 静态/参数路由守卫（`pantheon-base` PR #525，`63940437`）尚未通过 foundation release 进入 ops；当前绑定刻意避开静态/参数同层冲突路由，待发布后可进一步绑定。
