# K8s 资源读写授权与变更准入

- Status: completed（Base 路由守卫发布后的二次绑定为显式 gap；维护者最终视觉/功能验收 Human Gate 仍开放）
- Updated: 2026-11-24
- Baseline: a2d69d4a759b5df2b26aaa9d26bdc5175babe9dc
- Owner: K8s 授权负责人
- Priority: P1
- Activation: core
- Dependencies: T03, T04

## Implemented

- 变更分派修复：`scaleWorkload` 拒绝 DaemonSet/未知 kind；`restartWorkload` 按 kind 分派 DaemonSet/StatefulSet/Deployment，不再误改同名 Deployment。
- `ScaleRequest.Replicas *int32` + `min=0` 支持 0 副本。
- 陈旧 resourceVersion → 409（`common.ErrConflict`），调用方可刷新重试。
- 读授权统一：`NamespaceService.ReadableNamespaces` 按 Cluster 的 BusinessScopeID + Environment 过滤 namespace binding；NamespaceAll 聚合读、workload/pod/configmap/secret/service/ingress/storage PVC 列表按绑定过滤，显式 namespace 读先过 `RequireReadForClusterWithContext`。
- 读路由绑定：`business:k8s` 现在声明 cluster/namespace/workload/configmap/secret/service/release/ingress/storage/backup 的读与动作路由；无权限键归属的路由（集群凭据、Helm、install）保持未绑定，仍为默认拒绝。

## Verification

- `TestScaleWorkloadRejectsDaemonSetAndUnknownKind`、`TestRestartWorkloadPatchesNamedKindOnly` PASS；workload 包 ok；K8s 全模块 11/11 包 ok。
- 新增 `TestReadableNamespacesFollowsClusterOwnership`（真实 MySQL）：跨 scope/跨 environment 绑定不可见，scope=0 集群零可见；`TestK8sPermissionRoutesMatchActualActions` 校验每条已声明 K8s 路由真实存在且方法符合 action 语义。
- permission / role / tenant 套件（真实 MySQL）复跑 PASS。
- **live K3s（k14）PASS**：main `697c0771` 候选后端，`k8s-live.py --run-id omx-accept-20261121-k8s38` 17/17；未绑定 namespace 写入被拒。
- k8s 页面 smoke 由 PR #207 CI Smoke Sanity（17m30s）覆盖并绿。
- 证据：`../evidence/2026-11-20-ops-k8s-resource-authorization-and-mutations/`。

## Known Gaps

1. 只读详情路由在 Base 静态/参数路由守卫发布并消费后可进一步绑定；当前已绑定路由不含静态/参数同层冲突。
2. 浏览器探针覆盖 Service 编辑器冲突路径；workload/Pod 读授权矩阵由 live k14 与 CI smoke 覆盖。

## Browser Verification (2026-11-24)

- `omx-t09-browser-20261124-153621`（候选 `28cdcd00`，真 K3s `115.190.251.143:6443`）：kubectl annotate 使 Service rv `2543724→2543736`；UI 提交返回真实 409，编辑器保持打开且草稿选择器原样保留，冲突提示渲染；关闭并显式刷新列表后以新 rv 重试 200；独立 kubectl 读到 `uiRevision=1`。证据 `../../evidence/2026-11-20-ops-k8s-resource-authorization-and-mutations/browser-20261124-153621/`。
