# K8s 资源读写授权与变更准入

- Status: in-progress
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

## Verification

- `TestScaleWorkloadRejectsDaemonSetAndUnknownKind`、`TestRestartWorkloadPatchesNamedKindOnly` PASS；workload 包 ok；K8s 全模块 11/11 包 ok。
- 证据：`../evidence/2026-11-20-ops-k8s-resource-authorization-and-mutations/`。

## Known Gaps

1. NamespaceAll 跨 namespace 聚合读授权（RBAC 枚举拒绝）缺失（P1）。
2. 读路由 view/list 权限绑定不完整（P1）。
3. 真实 kubectl 干扰下 UI 草稿保留/恢复未在浏览器验证。
4. k8s 浏览器 smoke 未执行（前端 dev server 被内存回收）。

## Next Atomic Action

补读授权枚举与读路由绑定，并在 k8s smoke 可用时演练真实冲突的草稿恢复。
