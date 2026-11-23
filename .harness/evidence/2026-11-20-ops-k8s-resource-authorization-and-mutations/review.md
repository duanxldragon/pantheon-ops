# T07 Review — K8s 资源读写授权与变更准入

## 结论

`changes requested`（仅就完整验收而言）。变更分派缺陷（DaemonSet 误改同名 Deployment、未知 kind 落到 Deployment）已修复并有 fake-clientset 回归；但跨命名空间读授权、读路由绑定与界面草稿恢复尚未完成/验证，本包不能整体关闭。

## Findings（严重度从高到低）

1. **P1 — NamespaceAll 聚合读授权缺失。**
   跨 namespace 列表/聚合未按用户集群 RBAC 逐 namespace 枚举拒绝，越界元数据仍可能返回。与验收 #1 直接冲突。

2. **P1 — 读路由权限绑定不完整。**
   `List`/`GetPods` 等读路径未统一绑定 view/list 权限；写路径已有 `CheckHandlerPermission`。

3. **P2 — 真实 resourceVersion 冲突的 UI 草稿恢复未验证。**
   服务端 409 映射已实现，但浏览器端在 kubectl 并发干扰下的草稿保留/恢复未演练。

4. **fixed — DaemonSet 误改同名 Deployment。**
   `restartWorkload` 现按 kind 分派到 DaemonSets/StatefulSets/Deployments；`scaleWorkload` 拒绝 DaemonSet 与未知 kind。测试红→绿（先写测试复现 patch 落到 Deployment，后改实现）。

## Machine Readable

```json
{
  "taskId": "2026-11-20-ops-k8s-resource-authorization-and-mutations",
  "reviewer": "primary agent self-review (no independent review on this candidate)",
  "verdict": "changes requested",
  "decision": "changes requested",
  "scope": "workload mutation dispatch, scale/restart kind safety, resourceVersion conflict mapping",
  "taskManifest": ".harness/tasks/2026-11-20-ops-k8s-resource-authorization-and-mutations/manifest.json",
  "evidenceDir": ".harness/evidence/2026-11-20-ops-k8s-resource-authorization-and-mutations",
  "summaryFile": ".harness/evidence/2026-11-20-ops-k8s-resource-authorization-and-mutations/summary.md",
  "reviewFile": ".harness/evidence/2026-11-20-ops-k8s-resource-authorization-and-mutations/review.md",
  "changeRef": "none (uncommitted worktree)",
  "findings": [
    {"id": "T07-R1", "severity": "P1", "status": "open", "summary": "NamespaceAll cross-namespace read aggregation lacks RBAC enumeration rejection"},
    {"id": "T07-R2", "severity": "P1", "status": "open", "summary": "read-route permission bindings incomplete"},
    {"id": "T07-R3", "severity": "P2", "status": "open", "summary": "browser draft recovery under real resourceVersion conflict not exercised"},
    {"id": "T07-R4", "severity": "fixed", "status": "fixed", "summary": "DaemonSet restart patched same-name Deployment; scale accepted unknown kinds", "verification": "TestRestartWorkloadPatchesNamedKindOnly + TestScaleWorkloadRejectsDaemonSetAndUnknownKind PASS"}
  ],
  "verification": {
    "targetedWorkloadTests": "3/3 PASS",
    "workloadPackage": "ok",
    "k8sModuleTests": "11/11 packages ok"
  },
  "linkage": {
    "taskPacket": "docs/harness/tasks/2026-11-20-ops-k8s-resource-authorization-and-mutations.task.md",
    "evidence": ".harness/evidence/2026-11-20-ops-k8s-resource-authorization-and-mutations/commands.json",
    "reviewFile": ".harness/evidence/2026-11-20-ops-k8s-resource-authorization-and-mutations/review.md",
    "changeRef": "none",
    "planRefs": [".harness/tasks/2026-11-20-ops-review-remediation/plan.md"]
  }
}
```
