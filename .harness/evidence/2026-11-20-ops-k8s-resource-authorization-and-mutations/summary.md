# T07 K8s 资源读写授权与变更准入 — 实施摘要

- Task: `2026-11-20-ops-k8s-resource-authorization-and-mutations`
- 日期: 2026-11-24
- 状态: **变更准入分支已修复并锁定；读写授权与 UI 草稿恢复仍为显式 gap**
- 交付层级: L2（自审见 `review.md`）

## 已交付

| 验收项 | 状态 | 证据 |
|---|---|---|
| DaemonSet 不修改同名 Deployment | 已修复 | `scaleWorkload`/`restartWorkload` 改为按规范化 kind 分派；`TestRestartWorkloadPatchesNamedKindOnly`（fake clientset 同时存在同名 Deployment 与 DaemonSet） |
| 未知 kind 不放行 | 已修复 | `ScaleWithResourceVersion`/`RestartWithResourceVersion` 拒绝 `normalizeKind==""`；`TestScaleWorkloadRejectsDaemonSetAndUnknownKind`（daemonset/DaemonSet/replicaset） |
| 0 副本可提交 | 已由既有实现满足 | `ScaleRequest.Replicas *int32` + `min=0`，指针区分 0 与缺省 |
| 冲突可恢复（服务端） | 已实现 | `common.ErrConflict` → HTTP 409（`codeConflict`），调用方可用新 resourceVersion 重试 |
| 未授权 namespace 查询/变更拒绝 | 部分 | 变更路由已有 `CheckHandlerPermission`；NamespaceAll 跨命名空间聚合与读路由绑定仍是 gap |
| 界面保留草稿/可恢复 | 未验证 | 浏览器侧未做真实 resourceVersion 冲突演练 |

## 验证

- workload 包 3/3 定向 PASS；整包 ok
- K8s 全模块 11/11 包 ok（真实 MySQL DSN）

## 显式 gap

1. NamespaceAll 读授权：跨 namespace 聚合仍未做 RBAC 枚举拒绝。
2. 读路由权限绑定不完整（view/list 对集群 RBAC）。
3. 真实 kubectl 干扰 + UI 草稿恢复未在浏览器验证。
4. k8s 浏览器 smoke 未执行（前端 dev server 被内存回收）。
