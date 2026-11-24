# T06 K8s 资源读写授权与变更准入 — 实施摘要

- Task: `2026-11-20-ops-k8s-resource-authorization-and-mutations`
- 日期: 2026-11-24
- 状态: **代码级全部验收项已实现并验证；live K3s 的 namespace ownership/写保护（k14）PASS；真实 kubectl 干扰下的浏览器草稿保留/恢复已补验（2026-11-24）**
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
| 界面保留草稿/可恢复             | 已验证（2026-11-24） | 浏览器探针 `omx-t09-browser-20261124-153621`：kubectl annotate 使 rv 2543724→2543736；UI 提交收到真实 409，编辑器保持打开且草稿选择器原样保留，冲突提示渲染；关闭并显式刷新列表后用新 rv 重试 200，独立 kubectl 读到 `uiRevision=1`。证据 `browser-20261124-153621/`  |                                                                                                                                                                                                                                                       |

## 验证

- workload 包 3/3 定向 PASS；整包 ok
- K8s 全模块 12/12 包 ok（真实 MySQL DSN），含新增 `TestReadableNamespacesFollowsClusterOwnership` 与 `TestK8sPermissionRoutesMatchActualActions`
- permission / role / tenant 套件（真实 MySQL）复跑 PASS，消费扩展后的 K8s 权限路由注册表
- **live K3s（k14）PASS**：main `697c0771` 候选后端，`k8s-live.py --run-id omx-accept-20261121-k8s38` 17/17；未绑定 namespace 的写入在创建对象前被拒。
- k8s 页面 smoke 由 PR #207 CI Smoke Sanity（17m30s）执行并绿。

## 显式 gap

1. Base 静态/参数路由守卫（`pantheon-base` PR #525，`63940437`）尚未通过 foundation release 进入 ops；当前绑定刻意避开静态/参数同层冲突路由，待发布后可进一步绑定。
2. 浏览器探针覆盖 Service 编辑器冲突路径；workload/Pod 的读授权矩阵由 live k14 与 CI smoke 覆盖，本浏览器运行未重复该矩阵。
